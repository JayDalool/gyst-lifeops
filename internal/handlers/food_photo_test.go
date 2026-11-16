package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lifeops-tracker/internal/healthcoach"
)

const photoPath = "/health/nutrition/photo/analyze"

// fakeVision is a Provider that also analyses photos.
type fakeVision struct {
	fakeCoach
	gotJPEG []byte
	calls   int
	result  healthcoach.FoodPhotoResult
	err     error
}

func (f *fakeVision) AnalyzeFoodPhoto(_ context.Context, jpeg []byte) (healthcoach.FoodPhotoResult, error) {
	f.calls++
	f.gotJPEG = jpeg
	return f.result, f.err
}

func testJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 60, 40))
	for x := 0; x < 60; x++ {
		for y := 0; y < 40; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 4), 120, uint8(y * 6), 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func photoReq(t *testing.T, field string, data []byte, hdr map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile(field, "meal.jpg")
	fw.Write(data)
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, photoPath, &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	return r
}

func visionHandler(t *testing.T, f *fakeVision) *Handler {
	t.Helper()
	h := testHandler(t)
	if f != nil {
		h.Coach = f
	}
	return h
}

func doPhoto(h *Handler, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func errorOf(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct{ Error string }
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil {
		t.Fatalf("not JSON: %q", w.Body.String())
	}
	return e.Error
}

func TestPhotoAnalyzeSuccess(t *testing.T) {
	grams := 120.0
	f := &fakeVision{result: healthcoach.FoodPhotoResult{
		Items:         []healthcoach.FoodPhotoItem{{Name: "rice", Role: "side", EstGrams: &grams, Confidence: 0.8, Visible: true}},
		Uncertainties: []string{}, Questions: []string{},
	}}
	h := visionHandler(t, f)
	w := doPhoto(h, photoReq(t, "photo", testJPEG(t), map[string]string{"Origin": "http://example.com"}))
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("headers %v", w.Header())
	}
	var got healthcoach.FoodPhotoResult
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || len(got.Items) != 1 || got.Items[0].Name != "rice" {
		t.Fatalf("body %s", w.Body.String())
	}
	// The provider got a normalised JPEG, not the upload.
	if f.calls != 1 || len(f.gotJPEG) < 3 || f.gotJPEG[0] != 0xFF || f.gotJPEG[1] != 0xD8 {
		t.Fatal("provider did not receive a JPEG")
	}
}

func TestPhotoAnalyzeNotConfigured(t *testing.T) {
	for name, h := range map[string]*Handler{
		"no provider":     visionHandler(t, nil),
		"text-only coach": coachHandler(t, &fakeCoach{}),
	} {
		w := doPhoto(h, photoReq(t, "photo", testJPEG(t), nil))
		if w.Code != http.StatusServiceUnavailable || !strings.Contains(errorOf(t, w), "isn't set up") {
			t.Errorf("%s: %d %s", name, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: cacheable response", name)
		}
	}
}

func TestPhotoAnalyzeMethodAndOrigin(t *testing.T) {
	f := &fakeVision{}
	h := visionHandler(t, f)
	w := doPhoto(h, httptest.NewRequest(http.MethodGet, photoPath, nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "POST" {
		t.Fatalf("GET: %d", w.Code)
	}
	for name, hdr := range map[string]map[string]string{
		"cross-site fetch":  {"Sec-Fetch-Site": "cross-site"},
		"foreign origin":    {"Origin": "https://evil.example"},
		"sibling subdomain": {"Origin": "https://other.example.com", "Sec-Fetch-Site": "same-site"},
	} {
		w := doPhoto(h, photoReq(t, "photo", testJPEG(t), hdr))
		if w.Code != http.StatusForbidden {
			t.Errorf("%s: %d, want 403", name, w.Code)
		}
	}
	if f.calls != 0 {
		t.Fatal("a rejected request reached the provider")
	}
}

// Through the real auth wrapper: no session means no analysis.
func TestPhotoAnalyzeRequiresSession(t *testing.T) {
	h := authHandler(t)
	f := &fakeVision{}
	h.Coach = f
	post := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(post, photoReq(t, "photo", testJPEG(t), nil))
	if post.Code != http.StatusUnauthorized {
		t.Fatalf("POST without session: %d", post.Code)
	}
	get := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(get, httptest.NewRequest(http.MethodGet, photoPath, nil))
	if get.Code != http.StatusSeeOther || !strings.HasPrefix(get.Header().Get("Location"), "/login") {
		t.Fatalf("GET without session: %d %s", get.Code, get.Header().Get("Location"))
	}
	// Cross-origin with a valid session is still refused.
	cookie := signIn(t, h)
	r := photoReq(t, "photo", testJPEG(t), map[string]string{"Sec-Fetch-Site": "cross-site"})
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin with session: %d", w.Code)
	}
	ok := photoReq(t, "photo", testJPEG(t), nil)
	ok.AddCookie(cookie)
	w = httptest.NewRecorder()
	h.Authenticated().ServeHTTP(w, ok)
	if w.Code != 200 || f.calls != 1 {
		t.Fatalf("signed-in same-origin: %d calls=%d", w.Code, f.calls)
	}
}

func TestPhotoAnalyzeRejectsBadUploads(t *testing.T) {
	f := &fakeVision{}
	h := visionHandler(t, f)
	heic := append([]byte{0, 0, 0, 24}, []byte("ftypheic\x00\x00\x00\x00heicmif1")...)
	cases := []struct {
		name   string
		field  string
		data   []byte
		status int
		text   string
	}{
		{"wrong type", "photo", []byte("<html>not an image</html>"), 415, "JPEG, PNG or WebP"},
		{"heic", "photo", heic, 415, "Most Compatible"},
		{"oversized", "photo", append([]byte{0xFF, 0xD8, 0xFF}, make([]byte, 8<<20)...), 413, "8 MB"},
		{"wrong field", "file", testJPEG(t), 400, "Choose a photo"},
		{"corrupt", "photo", testJPEG(t)[:200], 400, "couldn't be read"},
	}
	for _, c := range cases {
		h.photoLimit = coachLimiter{} // not testing the limiter here
		w := doPhoto(h, photoReq(t, c.field, c.data, nil))
		if w.Code != c.status || !strings.Contains(errorOf(t, w), c.text) {
			t.Errorf("%s: %d %s", c.name, w.Code, w.Body.String())
		}
	}
	if f.calls != 0 {
		t.Fatal("a rejected upload reached the provider")
	}
	// A non-multipart body is refused too.
	r := httptest.NewRequest(http.MethodPost, photoPath, bytes.NewReader(testJPEG(t)))
	r.Header.Set("Content-Type", "image/jpeg")
	h.photoLimit = coachLimiter{}
	if w := doPhoto(h, r); w.Code != 400 {
		t.Errorf("raw body: %d", w.Code)
	}
}

func TestPhotoAnalyzeProviderErrorsAreFixedMessages(t *testing.T) {
	for _, c := range []struct {
		err    error
		status int
	}{
		{healthcoach.ErrUnavailable, 502},
		{healthcoach.ErrBusy, 429},
		{healthcoach.ErrNotConfigured, 503},
		{healthcoach.ErrDeclined, 502},
		{healthcoach.ErrPhotoUnreadable, 502},
		{errors.New("vendor said: sk-LEAK prompt text"), 502},
	} {
		h := visionHandler(t, &fakeVision{err: c.err})
		w := doPhoto(h, photoReq(t, "photo", testJPEG(t), nil))
		if w.Code != c.status {
			t.Errorf("%v: %d", c.err, w.Code)
		}
		if strings.Contains(w.Body.String(), "LEAK") || strings.Contains(w.Body.String(), "vendor") {
			t.Errorf("vendor text leaked: %s", w.Body.String())
		}
	}
}

func TestPhotoAnalyzeRateLimited(t *testing.T) {
	f := &fakeVision{}
	h := visionHandler(t, f)
	if w := doPhoto(h, photoReq(t, "photo", testJPEG(t), nil)); w.Code != 200 {
		t.Fatalf("first: %d", w.Code)
	}
	// Immediately again: inside the minimum gap.
	w := doPhoto(h, photoReq(t, "photo", testJPEG(t), nil))
	if w.Code != http.StatusTooManyRequests || f.calls != 1 {
		t.Fatalf("second: %d calls=%d", w.Code, f.calls)
	}
}

// Logs carry a fixed category only: no image, no vendor text.
func TestPhotoAnalyzeFailureLogHasNoContent(t *testing.T) {
	var buf bytes.Buffer
	h := visionHandler(t, &fakeVision{err: errors.New("echo of prompt: SECRET-MEAL-DATA")})
	h.Logger = slog.New(slog.NewJSONHandler(&buf, nil))
	doPhoto(h, photoReq(t, "photo", testJPEG(t), nil))
	out := buf.String()
	if !strings.Contains(out, "photo.analyze.failed") || strings.Contains(out, "SECRET-MEAL-DATA") {
		t.Fatalf("log: %s", out)
	}
}

// No file may appear in the temp dir or working dir during analysis.
func TestPhotoAnalyzeWritesNothingToDisk(t *testing.T) {
	h := visionHandler(t, &fakeVision{})
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	wd := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if w := doPhoto(h, photoReq(t, "photo", testJPEG(t), nil)); w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	for _, dir := range []string{tmp, wd} {
		if m, _ := filepath.Glob(filepath.Join(dir, "*")); len(m) != 0 {
			t.Fatalf("files written in %s: %v", dir, m)
		}
	}
}
