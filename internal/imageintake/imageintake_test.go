package imageintake

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Every fixture is generated here; no real photo is used.

func gradient(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 90, 255})
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// gpsMarker is a synthetic stand-in for GPS data; it must never survive.
const gpsMarker = "FAKE-GPS-49.8951N-97.1384W"

// withEXIF inserts an APP1 Exif segment carrying orientation and a fake GPS
// blob right after SOI.
func withEXIF(t *testing.T, jpg []byte, orientation uint16) []byte {
	t.Helper()
	var tiff bytes.Buffer
	tiff.WriteString("MM")
	binary.Write(&tiff, binary.BigEndian, uint16(42))
	binary.Write(&tiff, binary.BigEndian, uint32(8))
	binary.Write(&tiff, binary.BigEndian, uint16(1)) // one entry
	binary.Write(&tiff, binary.BigEndian, uint16(0x0112))
	binary.Write(&tiff, binary.BigEndian, uint16(3))
	binary.Write(&tiff, binary.BigEndian, uint32(1))
	binary.Write(&tiff, binary.BigEndian, orientation)
	binary.Write(&tiff, binary.BigEndian, uint16(0))
	binary.Write(&tiff, binary.BigEndian, uint32(0)) // next IFD
	tiff.WriteString(gpsMarker)
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func decodeOut(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
	return img
}

func TestNormalizeStripsEXIFAndGPS(t *testing.T) {
	src := withEXIF(t, encodeJPEG(t, gradient(64, 48)), 1)
	if !bytes.Contains(src, []byte(gpsMarker)) || !bytes.Contains(src, []byte("Exif")) {
		t.Fatal("fixture should carry EXIF with GPS")
	}
	out, err := Normalize(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{gpsMarker, "Exif", "GPS"} {
		if bytes.Contains(out, []byte(bad)) {
			t.Fatalf("output still contains %q", bad)
		}
	}
	// No APP1..APP15 or COM segment may remain before the scan.
	for i := 2; i+4 < len(out) && out[i] == 0xFF && out[i+1] != 0xDA; {
		if m := out[i+1]; (m >= 0xE1 && m <= 0xEF) || m == 0xFE {
			t.Fatalf("metadata segment %#x survived", m)
		}
		i += 2 + int(binary.BigEndian.Uint16(out[i+2:]))
	}
	decodeOut(t, out)
}

func TestNormalizeResizesToLongEdge(t *testing.T) {
	out, err := Normalize(encodeJPEG(t, gradient(3000, 2000)))
	if err != nil {
		t.Fatal(err)
	}
	b := decodeOut(t, out).Bounds()
	if b.Dx() != MaxEdge || b.Dy() != 682 {
		t.Fatalf("got %dx%d, want 1024x682", b.Dx(), b.Dy())
	}
	// Small images are not enlarged.
	out, _ = Normalize(encodeJPEG(t, gradient(100, 50)))
	if b := decodeOut(t, out).Bounds(); b.Dx() != 100 || b.Dy() != 50 {
		t.Fatalf("small image resized to %dx%d", b.Dx(), b.Dy())
	}
}

func TestNormalizeAppliesOrientationThenDropsIt(t *testing.T) {
	out, err := Normalize(withEXIF(t, encodeJPEG(t, gradient(80, 40)), 6))
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeOut(t, out).Bounds(); b.Dx() != 40 || b.Dy() != 80 {
		t.Fatalf("orientation 6 should give 40x80, got %dx%d", b.Dx(), b.Dy())
	}
	if jpegOrientation(out) != 1 {
		t.Fatal("output still carries an orientation tag")
	}
}

func TestNormalizePNGWithAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 20, 20)) // fully transparent
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	out, err := Normalize(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	r, g, bl, _ := decodeOut(t, out).At(10, 10).RGBA()
	if r>>8 < 240 || g>>8 < 240 || bl>>8 < 240 {
		t.Fatal("transparent PNG should flatten to white")
	}
}

func TestNormalizeRejects(t *testing.T) {
	jpg := encodeJPEG(t, gradient(30, 30))
	heic := append([]byte{0, 0, 0, 24}, []byte("ftypheic\x00\x00\x00\x00heicmif1")...)
	bomb := func() []byte { // 50000x50000 grayscale PNG header: ~2.5 GP, never decoded
		var b bytes.Buffer
		b.WriteString("\x89PNG\r\n\x1a\n")
		chunk := func(typ string, data []byte) {
			binary.Write(&b, binary.BigEndian, uint32(len(data)))
			b.WriteString(typ)
			b.Write(data)
			crc := crc32.NewIEEE()
			crc.Write([]byte(typ))
			crc.Write(data)
			binary.Write(&b, binary.BigEndian, crc.Sum32())
		}
		ihdr := make([]byte, 13)
		binary.BigEndian.PutUint32(ihdr[0:], 50000)
		binary.BigEndian.PutUint32(ihdr[4:], 50000)
		ihdr[8], ihdr[9] = 8, 0
		chunk("IHDR", ihdr)
		chunk("IDAT", []byte{0}) // DecodeConfig stops at the first IDAT
		return b.Bytes()
	}()
	cases := []struct {
		name string
		in   []byte
		want error
	}{
		{"empty", nil, ErrNoFile},
		{"text", []byte("hello, not an image"), ErrUnsupported},
		{"html", []byte("<html><script>alert(1)</script></html>"), ErrUnsupported},
		{"pdf", []byte("%PDF-1.7 ..."), ErrUnsupported},
		{"gif", []byte("GIF89a......"), ErrUnsupported},
		{"heic", heic, ErrUnsupported},
		{"webp", append([]byte("RIFF\x20\x00\x00\x00WEBPVP8 \x14\x00\x00\x00"), bytes.Repeat([]byte{0xFF}, 40)...), ErrUnsupported},
		{"truncated jpeg", jpg[:len(jpg)/3], ErrCorrupt},
		{"jpeg magic only", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0}, ErrCorrupt},
		{"decompression bomb", bomb, ErrDimensions},
		{"too large", make([]byte, MaxUpload+1), ErrTooLarge},
	}
	for _, c := range cases {
		if _, err := Normalize(c.in); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

// A JPEG with a script, a ZIP and an HTML page appended is a classic polyglot.
// It is accepted as the image it is, but none of the extra bytes survive.
func TestNormalizeDropsPolyglotPayload(t *testing.T) {
	payload := []byte("PK\x03\x04<script>alert('x')</script>POLYGLOT-PAYLOAD")
	src := append(encodeJPEG(t, gradient(40, 40)), payload...)
	// And a COM segment hiding text inside the header area.
	src = append(src[:2:2], append([]byte{0xFF, 0xFE, 0, 12}, append([]byte("POLYGLOT!"), src[2:]...)...)...)
	out, err := Normalize(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"POLYGLOT", "<script", "PK\x03\x04"} {
		if bytes.Contains(out, []byte(bad)) {
			t.Fatalf("output still contains %q", bad)
		}
	}
	// PNG magic over JPEG bytes (type confusion) is refused.
	if _, err := Normalize(append([]byte("\x89PNG\r\n\x1a\n"), encodeJPEG(t, gradient(8, 8))...)); err == nil {
		t.Fatal("PNG magic with JPEG body accepted")
	}
}

func multipartReq(t *testing.T, field, filename string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("note", "hello")
	if field != "" {
		fw, _ := mw.CreateFormFile(field, filename)
		fw.Write(data)
	}
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/x", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

func TestReadUpload(t *testing.T) {
	jpg := encodeJPEG(t, gradient(50, 50))
	out, err := ReadUpload(httptest.NewRecorder(), multipartReq(t, "photo", "meal.jpg", jpg), "photo")
	if err != nil || len(out) == 0 {
		t.Fatalf("valid upload: %v", err)
	}
	if _, err := ReadUpload(httptest.NewRecorder(), multipartReq(t, "", "", nil), "photo"); !errors.Is(err, ErrNoFile) {
		t.Errorf("missing file: %v", err)
	}
	if _, err := ReadUpload(httptest.NewRecorder(), multipartReq(t, "other", "a.jpg", jpg), "photo"); !errors.Is(err, ErrNoFile) {
		t.Errorf("wrong field: %v", err)
	}
	// The client's file name and part type are ignored: text named .jpg is refused.
	if _, err := ReadUpload(httptest.NewRecorder(), multipartReq(t, "photo", "evil.jpg", []byte("not an image")), "photo"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("fake extension: %v", err)
	}
	big := multipartReq(t, "photo", "big.jpg", append([]byte{0xFF, 0xD8, 0xFF}, make([]byte, MaxUpload)...))
	if _, err := ReadUpload(httptest.NewRecorder(), big, "photo"); !errors.Is(err, ErrTooLarge) {
		t.Errorf("oversized: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(jpg))
	r.Header.Set("Content-Type", "image/jpeg")
	if _, err := ReadUpload(httptest.NewRecorder(), r, "photo"); !errors.Is(err, ErrNotMultipart) {
		t.Errorf("non-multipart: %v", err)
	}
}

// Intake must not touch the filesystem: not the temp dir, not the cwd. The
// process temp dir is pointed at an empty directory and checked afterwards.
func TestReadUploadWritesNothingToDisk(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	wd := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	// Larger than the 32 MiB-free default would not matter: ParseMultipartForm
	// is what spills, and it is never called. Use a big-ish valid upload.
	jpg := encodeJPEG(t, gradient(1500, 1000))
	if _, err := ReadUpload(httptest.NewRecorder(), multipartReq(t, "photo", "m.jpg", jpg), "photo"); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{tmp, wd} {
		entries, _ := filepath.Glob(filepath.Join(dir, "*"))
		if len(entries) != 0 {
			t.Fatalf("files written under %s: %v", dir, entries)
		}
	}
}

func TestSniffHEICBrands(t *testing.T) {
	for _, brand := range []string{"heic", "heix", "mif1"} {
		if Sniff(append([]byte{0, 0, 0, 0x18}, []byte("ftyp"+brand)...)) != KindHEIC {
			t.Errorf("%s not detected", brand)
		}
	}
	if Sniff([]byte("\x00\x00\x00\x18ftypisom")) != KindUnknown {
		t.Error("mp4 misdetected as an image")
	}
}
