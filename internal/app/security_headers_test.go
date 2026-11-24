package app

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersThroughAppHandler(t *testing.T) {
	a := healthzApp(t)
	cases := []struct{ method, path string }{
		{"GET", "/login"},
		{"GET", "/static/css/styles.css"},
		{"GET", "/service-worker.js"},
		{"GET", "/manifest.webmanifest"},
		{"GET", "/favicon.ico"},
		{"GET", "/healthz"},
		{"POST", "/api/health/sync"},
		{"GET", "/today"},    // redirect to /login when signed out
		{"GET", "/nope-404"}, // falls through to the session middleware
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		a.Handler.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		h := rec.Header()
		if !strings.Contains(h.Get("Content-Security-Policy-Report-Only"), "frame-ancestors 'none'") {
			t.Errorf("%s %s: missing CSP", c.method, c.path)
		}
		if h.Get("X-Frame-Options") != "DENY" || h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Referrer-Policy") != "same-origin" || h.Get("Permissions-Policy") == "" {
			t.Errorf("%s %s: missing headers: %v", c.method, c.path, h)
		}
	}
}

// GYST-169, 13B DD-15: the icon sprite is a file of this site (img-src 'self' covers it) and is
// served as SVG, with the same headers as every other static file.
func TestIconSpriteIsServedSameOrigin(t *testing.T) {
	a := healthzApp(t)
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, httptest.NewRequest("GET", "/static/icons.svg?v=r7-4", nil))
	if rec.Code != 200 {
		t.Fatalf("GET /static/icons.svg = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/svg+xml") {
		t.Errorf("Content-Type = %q, want image/svg+xml", ct)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("sprite served without nosniff")
	}
	if !strings.Contains(rec.Body.String(), `<symbol id="i-home" viewBox="0 0 24 24">`) {
		t.Error("sprite body is not the icon sprite")
	}
}
