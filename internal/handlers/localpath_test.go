package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

var hostileLocalPaths = []string{
	"/\\evil.example", "/\\/evil.example", "/%5cevil.example", "/%5Cevil.example",
	"/\t/evil.example", "/\n/evil.example", "/\r/evil.example", "/\x00/evil",
	"//evil.example", "https://evil.example/", "http:/evil.example", "javascript:alert(1)",
	"evil.example", "", "/ok\r\nX-Injected: 1",
}

var ownLocalPaths = []string{
	"/today", "/today#task-1", "/tasks?status=all#task-3", "/projects/4#mission-8",
	"/health?period=week", "/courses/2#block-5",
}

func TestIsLocalPath(t *testing.T) {
	for _, candidate := range hostileLocalPaths {
		if isLocalPath(candidate) {
			t.Errorf("isLocalPath(%q) = true, want false", candidate)
		}
	}
	for _, candidate := range ownLocalPaths {
		if !isLocalPath(candidate) {
			t.Errorf("isLocalPath(%q) = false, want true", candidate)
		}
	}
}

// TestReturnToNeverRedirectsOffSite drives the real redirect path: a hostile
// return_to must fall back to the handler's own page, never reach Location.
func TestReturnToNeverRedirectsOffSite(t *testing.T) {
	for _, hostile := range hostileLocalPaths {
		form := url.Values{"return_to": {hostile}}
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		redirectAfter(rec, req, "/today")
		if loc := rec.Header().Get("Location"); strings.Contains(strings.ToLower(loc), "evil") || strings.Contains(loc, "\\") {
			t.Errorf("return_to %q redirected to %q", hostile, loc)
		}
	}
	for _, own := range ownLocalPaths {
		form := url.Values{"return_to": {own}}
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		redirectAfter(rec, req, "/today")
		if loc := rec.Header().Get("Location"); loc != own {
			t.Errorf("return_to %q redirected to %q, want it kept", own, loc)
		}
	}
}

func TestNextPathRejectsBackslashRedirects(t *testing.T) {
	for _, candidate := range []string{"/\\evil.example", "/%5cevil.example", "/\t/evil.example"} {
		if got := safeNextPath(candidate); got != "" {
			t.Errorf("safeNextPath(%q) = %q, want \"\"", candidate, got)
		}
	}
}
