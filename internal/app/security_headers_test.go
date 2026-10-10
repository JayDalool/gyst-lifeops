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
