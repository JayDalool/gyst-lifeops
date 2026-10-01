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

func TestSameHostReferer(t *testing.T) {
	const host = "gyst.example.test"
	cases := []struct {
		name, referer, want string
	}{
		{"empty", "", "/today"},
		{"same host keeps path query fragment", "https://gyst.example.test/tasks?status=all#task-3", "/tasks?status=all#task-3"},
		{"same host http", "http://gyst.example.test/projects/4", "/projects/4"},
		{"same host no path", "https://gyst.example.test", "/"},
		{"host differs only in case", "https://GYST.Example.Test/health?period=week", "/health?period=week"},
		{"cross host", "https://evil.example/tasks", "/today"},
		{"cross host with same-host prefix", "https://gyst.example.test.evil.example/tasks", "/today"},
		{"userinfo trick", "https://gyst.example.test@evil.example/tasks", "/today"},
		{"scheme relative", "//evil.example", "/today"},
		{"scheme relative same host path", "//evil.example/gyst.example.test", "/today"},
		{"backslash path", "https://gyst.example.test/\\evil.example", "/today"},
		{"encoded backslash path", "https://gyst.example.test/%5Cevil.example", "/today"},
		{"double slash path", "https://gyst.example.test//evil.example", "/today"},
		{"javascript scheme", "javascript:alert(1)", "/today"},
		{"non-http scheme same host", "ftp://gyst.example.test/tasks", "/today"},
		{"unparsable", "http://[::1", "/today"},
		{"path only kept", "/courses/2#block-5", "/courses/2#block-5"},
		{"path only hostile", "/\\evil.example", "/today"},
		{"bare host", "evil.example/x", "/today"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.Host = host
		if tc.referer != "" {
			req.Header.Set("Referer", tc.referer)
		}
		if got := sameHostReferer(req, "/today"); got != tc.want {
			t.Errorf("%s: sameHostReferer(%q) = %q, want %q", tc.name, tc.referer, got, tc.want)
		}
		rec := httptest.NewRecorder()
		redirectBack(rec, req, "/today")
		if loc := rec.Header().Get("Location"); loc != tc.want || rec.Code != http.StatusSeeOther {
			t.Errorf("%s: redirectBack Location = %q (%d), want %q", tc.name, loc, rec.Code, tc.want)
		}
	}
}
