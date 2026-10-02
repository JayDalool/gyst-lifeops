package app

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func healthzApp(t *testing.T) *App {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(appTestPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(Config{DBPath: filepath.Join(t.TempDir(), "healthz.db"), MigrationsDir: "../../migrations", TemplateDir: "../../web/templates", StaticDir: "../../web/static", PasswordHash: string(hash)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return a
}

func probe(a *App, method, path string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	return rec
}

func TestHealthzOKWithoutSessionOrSameOrigin(t *testing.T) {
	a := healthzApp(t)
	// A cross-site browser-style request must not be rejected either.
	rec := probe(a, http.MethodGet, "/healthz", func(r *http.Request) {
		r.Header.Set("Origin", "https://elsewhere.example")
		r.Header.Set("Sec-Fetch-Site", "cross-site")
	})
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("GET /healthz = %d %q, want 200 ok", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Set-Cookie") != "" {
		t.Fatal("healthz set a cookie")
	}
}

func TestHealthzHeadAllowedAndBodyless(t *testing.T) {
	rec := probe(healthzApp(t), http.MethodHead, "/healthz", nil)
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD /healthz = %d %q", rec.Code, rec.Body.String())
	}
}

func TestHealthzRejectsOtherMethods(t *testing.T) {
	a := healthzApp(t)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rec := probe(a, method, "/healthz", nil)
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" || rec.Body.Len() != 0 {
			t.Fatalf("%s /healthz = %d allow=%q body=%q", method, rec.Code, rec.Header().Get("Allow"), rec.Body.String())
		}
	}
}

func TestHealthz503WithClosedDatabaseLeaksNothing(t *testing.T) {
	a := healthzApp(t)
	if err := a.DB.Close(); err != nil {
		t.Fatal(err)
	}
	rec := probe(a, http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed database: %d, want 503", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("503 body must be empty, got %q", rec.Body.String())
	}
	assertCleanHeaders(t, rec)
}

func TestHealthzCarriesNoIdentifyingDetail(t *testing.T) {
	rec := probe(healthzApp(t), http.MethodGet, "/healthz", nil)
	assertCleanHeaders(t, rec)
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want exactly ok", rec.Body.String())
	}
}

func assertCleanHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	allowed := map[string]bool{"Content-Type": true, "Content-Length": true, "Cache-Control": true}
	for name, values := range rec.Header() {
		if !allowed[name] {
			t.Fatalf("unexpected header %s: %v", name, values)
		}
		for _, v := range values {
			lower := strings.ToLower(v)
			for _, bad := range []string{"sqlite", "go1", "version", "closed", "/srv", "/app", "error"} {
				if strings.Contains(lower, bad) {
					t.Fatalf("header %s=%q contains %q", name, v, bad)
				}
			}
		}
	}
}

func TestHealthzDoesNotLogProbes(t *testing.T) {
	a := healthzApp(t)
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	probe(a, http.MethodGet, "/healthz", nil)
	a.DB.Close()
	probe(a, http.MethodGet, "/healthz", nil)
	if buf.Len() != 0 {
		t.Fatalf("probes logged: %q", buf.String())
	}
}

// Variant spellings must not become an unauthenticated hole: the exact route
// is the only one the probe handler owns; everything else still needs a session.
func TestHealthzVariantsDoNotBypassAuth(t *testing.T) {
	a := healthzApp(t)

	// ServeMux cleans "//healthz" and redirects; the redirect itself must not
	// carry the handler's answer.
	rec := probe(a, http.MethodGet, "//healthz", nil)
	if rec.Body.String() == "ok" || rec.Code == http.StatusOK {
		t.Fatalf("//healthz reached the handler directly: %d %q", rec.Code, rec.Body.String())
	}

	for _, path := range []string{"/healthz/", "/HEALTHZ", "/Healthz", "/healthz/x"} {
		rec := probe(a, http.MethodGet, path, nil)
		if rec.Body.String() == "ok" {
			t.Fatalf("%s reached the health handler", path)
		}
		if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/login") {
			t.Fatalf("%s = %d (Location %q); want redirect to login", path, rec.Code, rec.Header().Get("Location"))
		}
	}
}
