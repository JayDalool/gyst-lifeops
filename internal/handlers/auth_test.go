package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"lifeops-tracker/internal/services"
)

const testPassword = "correct horse battery staple"

// authHandler returns a handler with a real database and a known password.
func authHandler(t *testing.T) *Handler {
	t.Helper()
	h := testHandler(t)
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	h.Auth = services.AuthConfig{PasswordHash: string(hash)}
	return h
}

// signIn performs a real login and returns the session cookie, so the tests
// exercise the same path a browser does rather than fabricating a session.
func signIn(t *testing.T, h *Handler) *http.Cookie {
	t.Helper()
	form := strings.NewReader("password=" + strings.ReplaceAll(testPassword, " ", "+"))
	request := httptest.NewRequest(http.MethodPost, "/login", form)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	h.Authenticated().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("login returned %d, want 303: %s", recorder.Code, recorder.Body.String())
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatal("login did not set a session cookie")
	return nil
}

func TestUnauthenticatedPageRedirectsToLogin(t *testing.T) {
	h := authHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/today", nil)
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	location := recorder.Header().Get("Location")
	if !strings.HasPrefix(location, "/login") {
		t.Errorf("Location = %q, want a /login redirect", location)
	}
	if !strings.Contains(location, "next=") {
		t.Errorf("Location = %q, want the original path preserved in next=", location)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Error("an authentication decision must not be cacheable")
	}
}

func TestUnauthenticatedAPIGets401(t *testing.T) {
	h := authHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/push/vapid-key", nil)
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "<html") {
		t.Error("an API caller should not receive an HTML page")
	}
}

// TestUnauthenticatedMutationIsNotRedirected covers a case a redirect would
// silently swallow: a POST cannot be replayed after login, so answering it with
// a redirect to the login page would drop the submission without saying so.
func TestUnauthenticatedMutationIsNotRedirected(t *testing.T) {
	h := authHandler(t)

	request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(""))
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestAuthenticatedRequestIsServed(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)

	request := httptest.NewRequest(http.MethodGet, "/today", nil)
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
}

// TestExemptPathsAreReachableWithoutASession asserts each exemption
// individually so the list cannot grow silently. Every entry is here because it
// carries its own credential or is needed to obtain one - never for convenience.
func TestExemptPathsAreReachableWithoutASession(t *testing.T) {
	h := authHandler(t)

	for _, testCase := range []struct {
		name string
		path string
		why  string
	}{
		{"login", "/login", "the page that issues a session"},
		{"health sync", "/api/health/sync", "authenticates with its own bearer token"},
		{"health api root", "/api/health", "same bearer-token surface"},
		{"calendar feed", "/calendar/feed/sometoken.ics", "carries a path token; external calendar clients send no cookie"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			recorder := httptest.NewRecorder()
			h.Authenticated().ServeHTTP(recorder, request)

			if recorder.Code == http.StatusSeeOther &&
				strings.HasPrefix(recorder.Header().Get("Location"), "/login") {
				t.Fatalf("%s was redirected to login but is exempt because it %s", testCase.path, testCase.why)
			}
			if recorder.Code == http.StatusUnauthorized && strings.Contains(recorder.Body.String(), "unauthorized") {
				t.Fatalf("%s got the session 401 but is exempt because it %s", testCase.path, testCase.why)
			}
		})
	}
}

// TestNonExemptPathsRequireASession is the other half: a route that looks
// adjacent to an exempt one must not inherit the exemption.
func TestNonExemptPathsRequireASession(t *testing.T) {
	h := authHandler(t)

	for _, path := range []string{
		"/", "/today", "/projects", "/health", "/health/plan", "/health/more",
		"/settings", "/analytics", "/api/push/vapid-key",
		"/calendar.ics", "/calendar/feedback",
		// /healthz is served by the app mux ahead of this middleware; it is
		// deliberately not on the exempt list.
		"/healthz", "/healthz/", "/HEALTHZ",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			h.Authenticated().ServeHTTP(recorder, request)

			redirected := recorder.Code == http.StatusSeeOther &&
				strings.HasPrefix(recorder.Header().Get("Location"), "/login")
			if !redirected && recorder.Code != http.StatusUnauthorized {
				t.Fatalf("%s served status %d without a session; it must require one", path, recorder.Code)
			}
		})
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	h := authHandler(t)

	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password=wrong"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName && cookie.Value != "" {
			t.Fatal("a failed login set a session cookie")
		}
	}
}

func TestLoginFailsClosedWithoutAConfiguredPassword(t *testing.T) {
	h := testHandler(t) // no Auth configured

	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password=anything"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookieName && cookie.Value != "" {
			t.Fatal("login succeeded with no password configured")
		}
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)

	if !cookie.HttpOnly {
		t.Error("session cookie is not HttpOnly; script could read it")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Errorf("Path = %q, want /", cookie.Path)
	}
}

// TestSessionCookieIsRenewedWithTheSession pins the cookie to the rolling
// server-side expiry. Before this, the session rolled forward on use but the
// cookie kept its login-time expiry, so the browser dropped a live session.
func TestSessionCookieIsRenewedWithTheSession(t *testing.T) {
	h := authHandler(t)
	base := time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC)
	h.Service.Now = func() time.Time { return base }
	cookie := signIn(t, h)

	visit := func(at time.Time) *http.Cookie {
		t.Helper()
		h.Service.Now = func() time.Time { return at }
		request := httptest.NewRequest(http.MethodGet, "/today", nil)
		request.Header.Set("X-Forwarded-Proto", "https")
		request.AddCookie(cookie)
		recorder := httptest.NewRecorder()
		h.Authenticated().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", recorder.Code)
		}
		for _, set := range recorder.Result().Cookies() {
			if set.Name == sessionCookieName {
				return set
			}
		}
		return nil
	}

	// Inside the refresh interval nothing is rewritten, cookie included.
	if renewed := visit(base.Add(time.Hour)); renewed != nil {
		t.Errorf("cookie re-sent inside the refresh interval, expires %v", renewed.Expires)
	}

	// Past it, the cookie is re-sent with the session's new expiry and the
	// same attributes and token as at login.
	later := base.Add(25 * time.Hour)
	renewed := visit(later)
	if renewed == nil {
		t.Fatal("cookie not renewed when the session rolled forward")
	}
	if want := later.Add(30 * 24 * time.Hour); !renewed.Expires.Equal(want.Truncate(time.Second)) {
		t.Errorf("renewed Expires = %v, want %v", renewed.Expires, want)
	}
	if renewed.Value != cookie.Value {
		t.Error("renewal changed the token; it must only extend the expiry")
	}
	if !renewed.HttpOnly || !renewed.Secure || renewed.SameSite != http.SameSiteLaxMode || renewed.Path != "/" {
		t.Errorf("renewed cookie lost attributes: %+v", renewed)
	}
}

// TestSecureCookieFollowsTheConnection pins the reason Secure is conditional:
// setting it unconditionally makes login fail silently over plain HTTP, because
// the browser drops the cookie and the next request bounces back to /login.
func TestSecureCookieFollowsTheConnection(t *testing.T) {
	plain := httptest.NewRequest(http.MethodGet, "/login", nil)
	if secureCookie(plain) {
		t.Error("Secure set on a plain HTTP request; the cookie would be dropped")
	}

	forwarded := httptest.NewRequest(http.MethodGet, "/login", nil)
	forwarded.Header.Set("X-Forwarded-Proto", "https")
	if !secureCookie(forwarded) {
		t.Error("Secure not set behind an HTTPS reverse proxy")
	}
}

func TestLogoutRevokesTheSession(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)

	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want 303", recorder.Code)
	}

	// The cookie must stop working server-side, not merely be cleared in the
	// browser: a copied token has to die too.
	after := httptest.NewRequest(http.MethodGet, "/today", nil)
	after.AddCookie(cookie)
	afterRecorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(afterRecorder, after)
	if afterRecorder.Code != http.StatusSeeOther {
		t.Fatalf("a revoked session still served /today with status %d", afterRecorder.Code)
	}
}

// TestCrossSiteMutationsAreRejected covers the correction that the guard must
// apply to every unsafe method, not POST alone.
func TestCrossSiteMutationsAreRejected(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)

	for _, method := range []string{
		http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
	} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/tasks", strings.NewReader(""))
			request.AddCookie(cookie)
			request.Header.Set("Sec-Fetch-Site", "cross-site")
			recorder := httptest.NewRecorder()
			h.Authenticated().ServeHTTP(recorder, request)

			if recorder.Code != http.StatusForbidden {
				t.Errorf("cross-site %s served status %d, want 403", method, recorder.Code)
			}
		})
	}
}

func TestCrossOriginMutationIsRejectedByOriginHost(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)

	request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(""))
	request.AddCookie(cookie)
	request.Header.Set("Origin", "https://untrusted.example")
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", recorder.Code)
	}
}

// TestSafeGetIsNotBlockedByTheGuard keeps the guard from becoming a general
// referrer check: a normal navigation from elsewhere must still load.
func TestSafeGetIsNotBlockedByTheGuard(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)

	request := httptest.NewRequest(http.MethodGet, "/today", nil)
	request.AddCookie(cookie)
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("cross-site GET status = %d, want 200", recorder.Code)
	}
}

// TestNextPathRejectsOpenRedirects stops the login page becoming a redirector
// to somebody else's site.
func TestNextPathRejectsOpenRedirects(t *testing.T) {
	for _, candidate := range []string{
		"//evil.example/", "https://evil.example/", "http:/evil.example",
		"javascript:alert(1)", "/ok\r\nX-Injected: 1", "",
	} {
		if got := safeNextPath(candidate); got != "" {
			t.Errorf("safeNextPath(%q) = %q, want \"\"", candidate, got)
		}
	}
	for _, candidate := range []string{"/today", "/projects/4", "/health?period=week"} {
		if got := safeNextPath(candidate); got != candidate {
			t.Errorf("safeNextPath(%q) = %q, want it kept", candidate, got)
		}
	}
}

// TestAuthDisabledEscapeHatch covers the lockout recovery path. It must work,
// and it must be obvious that it is on.
func TestAuthDisabledEscapeHatch(t *testing.T) {
	h := authHandler(t)
	h.AuthDisabled = true

	request := httptest.NewRequest(http.MethodGet, "/today", nil)
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with auth disabled", recorder.Code)
	}
}
