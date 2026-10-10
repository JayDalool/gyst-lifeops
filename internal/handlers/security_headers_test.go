package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSecurityHeadersOnEveryResponseKind(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/redir", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/ok", http.StatusFound) })
	mux.HandleFunc("/err", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusForbidden) })
	h := SecurityHeaders(mux)
	want := map[string]string{
		"Content-Security-Policy-Report-Only": contentSecurityPolicy,
		"X-Frame-Options":                     "DENY",
		"X-Content-Type-Options":              "nosniff",
		"Referrer-Policy":                     "same-origin",
		"Permissions-Policy":                  "camera=(), microphone=(), geolocation=(), payment=(), usb=()",
	}
	for _, path := range []string{"/ok", "/redir", "/err", "/missing"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		for k, v := range want {
			if got := rec.Header().Get(k); got != v {
				t.Errorf("%s %s = %q, want %q", path, k, got, v)
			}
		}
		if rec.Header().Get("Content-Security-Policy") != "" {
			t.Errorf("%s: enforcing CSP header sent in report-only phase", path)
		}
	}
}

func TestContentSecurityPolicyShape(t *testing.T) {
	want := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; connect-src 'self'; font-src 'self'; worker-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
	if contentSecurityPolicy != want {
		t.Fatalf("CSP changed:\n%s", contentSecurityPolicy)
	}
	for _, must := range []string{"frame-ancestors 'none'", "object-src 'none'"} {
		if !strings.Contains(contentSecurityPolicy, must) {
			t.Errorf("CSP missing %s", must)
		}
	}
	for _, d := range strings.Split(contentSecurityPolicy, ";") {
		d = strings.TrimSpace(d)
		if strings.HasPrefix(d, "script-src") {
			for _, bad := range []string{"'unsafe-inline'", "'unsafe-eval'", "*", "nonce-"} {
				if strings.Contains(d, bad) {
					t.Errorf("script-src contains %s", bad)
				}
			}
		}
	}
}

// Served templates must not contain inline script or on*= handlers, or the
// script-src 'self' policy would block them. ui_gallery.html is only rendered
// by a test and is never served.
func TestTemplatesHaveNoInlineScriptOrHandlers(t *testing.T) {
	files, err := filepath.Glob("../../web/templates/*.html")
	if err != nil || len(files) == 0 {
		t.Fatalf("no templates found: %v", err)
	}
	scriptTag := regexp.MustCompile(`(?is)<script\b[^>]*>`)
	hasSrc := regexp.MustCompile(`(?i)\ssrc\s*=`)
	handler := regexp.MustCompile(`(?i)<[a-z][^>]*\son[a-z]+\s*=`)
	for _, f := range files {
		if filepath.Base(f) == "ui_gallery.html" {
			continue
		}
		b, _ := os.ReadFile(f)
		s := string(b)
		for _, m := range scriptTag.FindAllString(s, -1) {
			if !hasSrc.MatchString(m) {
				t.Errorf("%s: inline script tag %q", f, m)
			}
		}
		if handler.MatchString(s) {
			t.Errorf("%s: inline on*= attribute", f)
		}
	}
}
