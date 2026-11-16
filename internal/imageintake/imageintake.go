// Package imageintake takes in a user-supplied photo safely (Issue #217).
//
// The upload is read into memory only: nothing is written to disk, not even
// the multipart spill file that ParseMultipartForm would create. The real type
// is sniffed from the bytes (the client's Content-Type and file name are
// never trusted), the image is checked for decompression bombs before it is
// decoded, then decoded and re-encoded as a JPEG of at most MaxEdge pixels on
// the long edge. Re-encoding drops EXIF (including GPS), ICC data, comments
// and anything appended after the image, which also defuses polyglot files.
//
// Reusable by the barcode scanner (#210).
package imageintake

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"io"
	"mime"
	"mime/multipart"
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder for image.DecodeConfig/Decode
)

const (
	// MaxUpload is the cap on the uploaded file itself.
	MaxUpload = 8 << 20
	// maxBody leaves room for the multipart framing around the file.
	maxBody = MaxUpload + 64<<10
	// MaxEdge is the long edge, in pixels, of the re-encoded JPEG.
	MaxEdge = 1024
	// MaxPixels caps the decoded size (width*height) before any full decode,
	// so a tiny file that expands to gigabytes (decompression bomb) is refused.
	MaxPixels = 40_000_000
	// jpegQuality is the re-encode quality.
	jpegQuality = 85
	// maxParts bounds how many multipart parts are inspected.
	maxParts = 8
)

// Errors are safe to show to the user; they never carry file content.
var (
	ErrNotMultipart = errors.New("Send the photo as a file upload")
	ErrNoFile       = errors.New("Choose a photo first")
	ErrTooLarge     = errors.New("That photo is too large. The limit is 8 MB")
	ErrUnsupported  = errors.New("That file isn't a photo GYST can read. Use a JPEG, PNG or WebP image")
	ErrHEIC         = errors.New("GYST can't read HEIC photos yet. On iPhone, set Settings > Camera > Formats to Most Compatible, then take the photo again")
	ErrCorrupt      = errors.New("That photo couldn't be read. Try another one")
	ErrDimensions   = errors.New("That photo's dimensions are too large. Try a smaller photo")
)

// Kind is a sniffed image type.
type Kind string

const (
	KindJPEG    Kind = "jpeg"
	KindPNG     Kind = "png"
	KindWebP    Kind = "webp"
	KindHEIC    Kind = "heic"
	KindUnknown Kind = ""
)

// Sniff identifies the image type from the leading bytes only.
func Sniff(b []byte) Kind {
	switch {
	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return KindJPEG
	case len(b) >= 8 && bytes.Equal(b[:8], []byte("\x89PNG\r\n\x1a\n")):
		return KindPNG
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return KindWebP
	case len(b) >= 12 && string(b[4:8]) == "ftyp":
		switch string(b[8:12]) {
		case "heic", "heix", "heim", "heis", "hevc", "hevx", "mif1", "msf1":
			return KindHEIC
		}
	}
	return KindUnknown
}

// ReadUpload reads the single file part named field from a multipart POST,
// entirely in memory, and returns the normalised JPEG. It streams the body
// with MultipartReader so nothing is spilled to a temp file.
func ReadUpload(w http.ResponseWriter, r *http.Request, field string) ([]byte, error) {
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/form-data" || params["boundary"] == "" {
		return nil, ErrNotMultipart
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	mr := multipart.NewReader(r.Body, params["boundary"])
	for i := 0; i < maxParts; i++ {
		part, err := mr.NextPart()
		if err == io.EOF {
			return nil, ErrNoFile
		}
		if err != nil {
			return nil, readErr(err)
		}
		if part.FormName() != field || part.FileName() == "" {
			_, _ = io.Copy(io.Discard, io.LimitReader(part, 1<<10))
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(part, MaxUpload+1))
		if err != nil {
			return nil, readErr(err)
		}
		if len(raw) > MaxUpload {
			return nil, ErrTooLarge
		}
		return Normalize(raw)
	}
	return nil, ErrNoFile
}

func readErr(err error) error {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return ErrTooLarge
	}
	return ErrCorrupt
}

// Normalize validates raw image bytes and returns a metadata-free JPEG no
// larger than MaxEdge on its long edge.
func Normalize(raw []byte) (out []byte, err error) {
	// The WebP decoder has known panics on hostile input (govulncheck:
	// GO-2026-5061, fixed in x/image v0.43, which needs Go 1.26). A panic
	// becomes an ordinary "unreadable" error instead of a crashed request.
	defer func() {
		if recover() != nil {
			out, err = nil, ErrCorrupt
		}
	}()
	if len(raw) == 0 {
		return nil, ErrNoFile
	}
	if len(raw) > MaxUpload {
		return nil, ErrTooLarge
	}
	kind := Sniff(raw)
	switch kind {
	case KindHEIC:
		return nil, ErrHEIC
	case KindUnknown:
		return nil, ErrUnsupported
	}
	// Dimensions come from the header, before any pixel buffer exists.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || string(kind) != format {
		return nil, ErrCorrupt
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 1<<16 || cfg.Height > 1<<16 ||
		int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return nil, ErrDimensions
	}
	src, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil || string(kind) != format {
		return nil, ErrCorrupt
	}
	orient := 1
	if kind == KindJPEG {
		orient = jpegOrientation(raw)
	}
	return encode(src, orient)
}

// encode flattens onto white (JPEG has no alpha), scales to MaxEdge and
// writes a fresh JPEG with no metadata.
func encode(src image.Image, orient int) ([]byte, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > MaxEdge || h > MaxEdge {
		if w >= h {
			h, w = max(1, h*MaxEdge/w), MaxEdge
		} else {
			w, h = max(1, w*MaxEdge/h), MaxEdge
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, applyOrientation(dst, orient), &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, ErrCorrupt
	}
	return out.Bytes(), nil
}
