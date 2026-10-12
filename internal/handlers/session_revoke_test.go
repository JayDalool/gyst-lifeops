package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lifeops-tracker/internal/services"
)

// GYST-VN-078: POST /settings/sessions/revoke-all. Fictional values only.

const revokePath = "/settings/sessions/revoke-all"

func revokeReq(method string, cookie *http.Cookie, headers map[string]string) *http.Request {
	r := httptest.NewRequest(method, revokePath, strings.NewReader(url.Values{}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func sessionCount(t *testing.T, rig *logRig) int {
	t.Helper()
	n, err := rig.h.Service.Repo.CountSessions()
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRevokeAllSignsOutEveryDevice(t *testing.T) {
	rig := newLogRig(t)
	phone, laptop := rig.signIn(t), rig.signIn(t)
	w := rig.do(revokeReq(http.MethodPost, phone, map[string]string{"Origin": "http://example.com"}))
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login" {
		t.Fatalf("status %d Location %q, want 303 /login", w.Code, w.Header().Get("Location"))
	}
	cleared := false
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("session cookie not cleared")
	}
	if n := sessionCount(t, rig); n != 0 {
		t.Fatalf("sessions left = %d", n)
	}
	for _, c := range []*http.Cookie{phone, laptop} {
		r := httptest.NewRequest(http.MethodGet, "/today", nil)
		r.AddCookie(c)
		if got := rig.do(r); got.Code != http.StatusSeeOther || !strings.HasPrefix(got.Header().Get("Location"), "/login") {
			t.Fatalf("old cookie got %d %q, want login redirect", got.Code, got.Header().Get("Location"))
		}
	}

	rows, err := rig.h.Service.Repo.RecentAuditEvents(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, e := range rows {
		if e.Action != services.AuditActionSessionRevokeAll {
			continue
		}
		found++
		if e.Outcome != services.AuditOutcomeSuccess || e.ActorType != services.AuditActorHuman ||
			!strings.HasPrefix(e.ActorRef, "session:") {
			t.Fatalf("audit row = %+v", e)
		}
		if strings.Contains(rowText(e), phone.Value) || strings.Contains(rowText(e), laptop.Value) {
			t.Fatal("audit row carries a token")
		}
	}
	if found != 1 {
		t.Fatalf("revoke-all audit rows = %d, want 1", found)
	}
	logs := rig.buf.String()
	if !strings.Contains(logs, `"event":"session.revoked_all"`) || !strings.Contains(logs, `"count":2`) {
		t.Fatalf("missing log line: %s", logs)
	}
	for _, secret := range []string{phone.Value, laptop.Value, rig.h.Auth.PasswordHash} {
		if strings.Contains(logs, secret) {
			t.Fatal("log carries a secret")
		}
	}
}

func TestRevokeAllNegatives(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		cookie  bool
		headers map[string]string
		status  int
	}{
		{"anonymous POST", http.MethodPost, false, nil, http.StatusUnauthorized},
		{"cross-site", http.MethodPost, true, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"foreign origin", http.MethodPost, true, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"GET", http.MethodGet, true, nil, http.StatusMethodNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rig := newLogRig(t)
			cookie := rig.signIn(t)
			if !tc.cookie {
				cookie = nil
			}
			w := rig.do(revokeReq(tc.method, cookie, tc.headers))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			if tc.method == http.MethodGet && w.Header().Get("Allow") != "POST" {
				t.Fatalf("Allow = %q", w.Header().Get("Allow"))
			}
			if n := sessionCount(t, rig); n != 1 {
				t.Fatalf("sessions = %d, want 1 (nothing revoked)", n)
			}
		})
	}
}

func TestAnonymousJSONRevokeGets401(t *testing.T) {
	rig := newLogRig(t)
	rig.signIn(t)
	r := revokeReq(http.MethodPost, nil, map[string]string{"Accept": "application/json"})
	r.URL.Path = "/api" + revokePath
	if w := rig.do(r); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if n := sessionCount(t, rig); n != 1 {
		t.Fatal("anonymous request revoked sessions")
	}
}

func TestSettingsRendersSecuritySection(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	r := httptest.NewRequest(http.MethodGet, "/settings", nil)
	r.AddCookie(cookie)
	w := rig.do(r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`id="security"`, `<h2 class="section-title">Security</h2>`,
		`action="/settings/sessions/revoke-all"`, `method="post"`,
		`gyst-button gyst-button--secondary" type="submit">Sign out everywhere</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("settings page missing %q", want)
		}
	}
	if strings.Contains(body, "--xp") {
		t.Error("settings page uses an --xp class")
	}
	if strings.Index(body, `id="security"`) < strings.Index(body, ">Motion<") {
		t.Error("Security must come after Motion")
	}
}
