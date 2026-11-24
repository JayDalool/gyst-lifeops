package handlers

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/services"
)

// GYST-188 (spec a3): the rebuilt sign-in page. Every value is fictional.

var (
	bannerRE   = regexp.MustCompile(`(?s)\s*<div class="gyst-banner".*?</div>`)
	invalidRE  = regexp.MustCompile(` aria-invalid="true"`)
	mainTagRE  = regexp.MustCompile(`<main[ >]`)
	onAttrRE   = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
	scriptTags = regexp.MustCompile(`(?i)<script[^>]*>`)
)

func loginGet(h *Handler, query string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/login"+query, nil)
	for _, c := range cookies {
		r.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, r)
	return rec
}

func TestLoginPageFollowsTheThemeAndHasNoInlineScript(t *testing.T) {
	h, _ := throttledHandler(t)
	rec := loginGet(h, "")
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	for _, want := range []string{
		`<html lang="en" data-theme="light" data-gyst-theme="light">`,
		`<meta name="theme-color" content="#F1EFF9" data-light="#F1EFF9" data-dark="#09080F">`,
		`<link rel="stylesheet" href="/static/css/ui-signin.css?v=signin-1">`,
		`<script defer src="/static/js/signin.js?v=signin-1"></script>`,
		`<h1 class="gyst-signin-title">Sign in</h1>`,
		`<p class="gyst-signin-brand">GYST</p>`,
		`<input class="gyst-signin-input" id="password" name="password" type="password" autocomplete="current-password" enterkeyhint="go" required autofocus>`,
		`<button class="gyst-signin-submit" type="submit">Sign in</button>`,
		`data-signin-show aria-pressed="false" aria-label="Show password" hidden>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sign-in page lacks %s", want)
		}
	}
	// theme.js sits in the head ahead of every stylesheet, so the scheme is set before first paint.
	head := body[:strings.Index(body, "</head>")]
	if i, j := strings.Index(head, `/static/js/theme.js?v=appearance-2`), strings.Index(head, `rel="stylesheet"`); i < 0 || j < 0 || i > j {
		t.Error("theme.js must be in the head, before the stylesheets")
	}
	if strings.Contains(body, "styles.css") {
		t.Error("the sign-in page must not load styles.css")
	}
	if n := len(mainTagRE.FindAllString(body, -1)); n != 1 {
		t.Errorf("%d <main> elements, want 1", n)
	}
	if n := strings.Count(body, "<h1"); n != 1 {
		t.Errorf("%d <h1> elements, want 1", n)
	}
	// c5: no inline script and no event-handler attribute, so the CSP (script-src 'self') is enough.
	for _, tag := range scriptTags.FindAllString(body, -1) {
		if !strings.Contains(tag, ` src="/static/js/`) {
			t.Errorf("inline or off-site script: %s", tag)
		}
	}
	if onAttrRE.MatchString(body) {
		t.Error("an on*= attribute is in the page")
	}
	if strings.Contains(body, `role="alert"`) || strings.Contains(body, `role="status"`) || strings.Contains(body, "gyst-banner") {
		t.Error("the empty state shows no banner")
	}
	if strings.Contains(body, "aria-describedby") || strings.Contains(body, "aria-invalid") {
		t.Error("the empty state has no description or invalid mark")
	}
}

// The CSP is the policy master sent; the sign-in rebuild adds a same-origin script and style only.
func TestContentSecurityPolicyIsUnchanged(t *testing.T) {
	const master = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; connect-src 'self'; font-src 'self'; worker-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
	if contentSecurityPolicy != master {
		t.Errorf("contentSecurityPolicy changed:\n%s", contentSecurityPolicy)
	}
}

func TestLoginBannerPerState(t *testing.T) {
	const secret = "fictional-password-do-not-echo"
	type state struct {
		name       string
		status     int
		dataState  string
		icon       string
		sentence   string
		second     string
		invalid    bool
		setup      func(t *testing.T) *httptest.ResponseRecorder
		wantStatus int
	}
	postForm := func(h *Handler, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		r.Host = "gyst.zaylolabs.com"
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("CF-Connecting-IP", "203.0.113.70")
		rec := httptest.NewRecorder()
		h.Authenticated().ServeHTTP(rec, r)
		return rec
	}
	states := []state{
		{name: "wrong", status: 401, dataState: "crit", icon: "alert", sentence: "Incorrect password.", second: "Check it and try again.", invalid: true,
			setup: func(t *testing.T) *httptest.ResponseRecorder {
				h, _ := throttledHandler(t)
				return postForm(h, "password="+secret)
			}},
		{name: "throttled", status: 429, dataState: "warn", icon: "clock", sentence: throttleText, second: "Your data is untouched.",
			setup: func(t *testing.T) *httptest.ResponseRecorder {
				h, clk := throttledHandler(t)
				failFive(t, h, clk, "203.0.113.70")
				return postForm(h, "password="+secret)
			}},
		{name: "unreadable", status: 400, dataState: "crit", icon: "alert", sentence: "That form could not be read. Try again.",
			setup: func(t *testing.T) *httptest.ResponseRecorder {
				h, _ := throttledHandler(t)
				return postForm(h, "password="+secret+"&pad="+strings.Repeat("a", 9000))
			}},
		{name: "unconfigured", status: 503, dataState: "warn", icon: "alert", sentence: "No password is configured. Set GYST_PASSWORD_HASH on the server.",
			setup: func(t *testing.T) *httptest.ResponseRecorder {
				h, _ := throttledHandler(t)
				h.Auth = services.AuthConfig{}
				return postForm(h, "password="+secret)
			}},
	}
	for _, s := range states {
		t.Run(s.name, func(t *testing.T) {
			rec := s.setup(t)
			body := rec.Body.String()
			if rec.Code != s.status {
				t.Fatalf("status %d, want %d", rec.Code, s.status)
			}
			if n := strings.Count(body, `class="gyst-banner"`); n != 1 {
				t.Fatalf("%d banners, want exactly 1", n)
			}
			want := `<div class="gyst-banner" id="login-error" role="alert" data-state="` + s.dataState + `">`
			if !strings.Contains(body, want) {
				t.Errorf("banner opening is not %s", want)
			}
			if !strings.Contains(body, `<use href="/static/icons.svg?v=r7-6#i-`+s.icon+`"/>`) {
				t.Errorf("banner icon is not i-%s", s.icon)
			}
			sentence := "<b>" + strings.ReplaceAll(s.sentence, "'", "&#39;") + "</b>"
			if s.second != "" {
				sentence += " " + s.second
			}
			if !strings.Contains(body, sentence+"</span>") {
				t.Errorf("banner text is not %q in %s", sentence, body)
			}
			// Tied to the field for a screen reader; invalid is for a wrong password only.
			if !strings.Contains(body, ` aria-describedby="login-error"`) || !strings.Contains(body, `id="login-error"`) {
				t.Error("the field is not described by the banner")
			}
			if got := invalidRE.MatchString(body); got != s.invalid {
				t.Errorf("aria-invalid present = %v, want %v", got, s.invalid)
			}
			if strings.Contains(body, secret) {
				t.Error("the submitted password came back in the page")
			}
			if strings.Contains(body, "You were signed out") {
				t.Error("a signed-out banner shown beside an error banner")
			}
			if strings.Contains(body, "Retry-After") || rec.Header().Get("Retry-After") != "" {
				t.Error("Retry-After must not exist")
			}
		})
	}
}

// c3: the signed-out banner depends on the cookie being present and on nothing else, and the page
// makes no database read, log line or audit row for it.
func TestLoginSignedOutBannerDependsOnTheCookieOnly(t *testing.T) {
	rig := newAuditRig(t)
	h := rig.h
	// "expired" is a real session left unused past its 30-day lifetime; "revoked" one signed out elsewhere.
	expired := signIn(t, h)
	rig.clk.Advance(31 * 24 * time.Hour)
	live := signIn(t, h)
	revoked := signIn(t, h)
	if ok, err := h.Service.RevokeSession(revoked.Value); err != nil || !ok {
		t.Fatalf("revoke: %v %v", ok, err)
	}
	if h.Service.ValidateSession(expired.Value) == nil {
		t.Fatal("the expired session is still valid")
	}
	rig.buf.Reset()
	rowsBefore := len(rig.rows(t))

	plain := loginGet(h, "")
	empty := loginGet(h, "", &http.Cookie{Name: sessionCookieName, Value: ""})
	random := loginGet(h, "", &http.Cookie{Name: sessionCookieName, Value: "not-a-real-token-fictional"})
	exp := loginGet(h, "", expired)
	rev := loginGet(h, "", revoked)

	if plain.Body.String() != empty.Body.String() {
		t.Error("an empty cookie must show the plain form")
	}
	for name, rec := range map[string]*httptest.ResponseRecorder{"random": random, "expired": exp, "revoked": rev} {
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", name, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `<div class="gyst-banner" role="status">`) || !strings.Contains(rec.Body.String(), "<b>You were signed out.</b> If you were saving something, it was not saved. Sign in and do it again.") {
			t.Errorf("%s: no signed-out banner", name)
		}
		if rec.Body.String() != random.Body.String() {
			t.Errorf("%s differs from the random-cookie page", name)
		}
		// The banner is the only difference from the plain page.
		if bannerRE.ReplaceAllString(rec.Body.String(), "") != plain.Body.String() {
			t.Errorf("%s differs from the plain page by more than the banner", name)
		}
		if rec.Header().Get("Set-Cookie") != "" {
			t.Errorf("%s set a cookie", name)
		}
		if rec.Header().Get("Cache-Control") != plain.Header().Get("Cache-Control") || rec.Header().Get("Content-Type") != plain.Header().Get("Content-Type") {
			t.Errorf("%s headers differ", name)
		}
		if strings.Contains(rec.Body.String(), "not-a-real-token-fictional") {
			t.Errorf("%s echoed the cookie", name)
		}
	}
	if got := len(rig.rows(t)); got != rowsBefore {
		t.Errorf("audit rows %d -> %d: the page must write none", rowsBefore, got)
	}
	if rig.buf.Len() != 0 {
		t.Errorf("the page logged: %s", rig.buf.String())
	}

	// With next set the hint changes; without a cookie it stays the default.
	withNext := loginGet(h, "?next=/tasks", &http.Cookie{Name: sessionCookieName, Value: "not-a-real-token-fictional"})
	if !strings.Contains(withNext.Body.String(), "After sign-in you go back to <b>where you were</b>.") {
		t.Error("signed out with next: hint not changed")
	}
	if strings.Contains(loginGet(h, "?next=/tasks").Body.String(), "where you were") {
		t.Error("no cookie: the default hint stays")
	}
	// A valid session is still sent away, never shown the form.
	if rec := loginGet(h, "", live); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Errorf("valid session: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	// Auth off never shows it, whatever the cookie.
	h.AuthDisabled = true
	if strings.Contains(loginGet(h, "", &http.Cookie{Name: sessionCookieName, Value: "x"}).Body.String(), "You were signed out") {
		t.Error("AuthDisabled showed the signed-out banner")
	}
}

// The three banners an error shows are the one banner element in different words: after the banner and
// the aria-invalid mark are removed the 401, 429, 400 and 503 pages are the same page.
func TestLoginErrorPagesDifferOnlyInTheBanner(t *testing.T) {
	h, clk := throttledHandler(t)
	wrong := pub(h, wrongPW, "203.0.113.80")
	failFive(t, h, clk, "203.0.113.81")
	throttled := pub(h, wrongPW, "203.0.113.81")
	strip := func(s string) string { return invalidRE.ReplaceAllString(bannerRE.ReplaceAllString(s, ""), "") }
	if strip(wrong.Body.String()) != strip(throttled.Body.String()) {
		t.Error("the 401 and 429 pages differ by more than the banner and aria-invalid")
	}
}
