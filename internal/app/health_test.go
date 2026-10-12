package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const appTestPassword = "app-test-password"

// signInThroughMux logs in against the fully wired handler and returns the
// session cookie, so the route assertions below exercise the real stack -
// including the auth middleware - rather than bypassing it.
func signInThroughMux(t *testing.T, handler http.Handler) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader("password="+appTestPassword))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login through mux: %d %s", rec.Code, rec.Body.String())
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "gyst_session" && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatal("login through mux set no session cookie")
	return nil
}

func TestHealthAPIConfigAndRootMux(t *testing.T) {
	for _, token := range []string{"", "test-only-app-health-token"} {
		t.Run("configured="+map[bool]string{true: "yes", false: "no"}[token != ""], func(t *testing.T) {
			hash, err := bcrypt.GenerateFromPassword([]byte(appTestPassword), bcrypt.MinCost)
			if err != nil {
				t.Fatal(err)
			}
			a, err := New(Config{DBPath: filepath.Join(t.TempDir(), "health.db"), MigrationsDir: "../../migrations", TemplateDir: "../../web/templates", StaticDir: "../../web/static", LoadSeed: false, HealthSyncToken: token, PasswordHash: string(hash)})
			if err != nil {
				t.Fatal(err)
			}
			defer a.Close()
			a.service.Now = func() time.Time { return time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC) }
			// The Health API is write-only, so root-mux routing is probed with
			// the sync endpoint rather than a read.
			req := httptest.NewRequest(http.MethodPost, "/api/health/sync",
				strings.NewReader(`{"date":"2026-09-10","source":"apple_shortcuts","dietary_calories":130}`))
			req.Header.Set("Authorization", "Bearer test-only-app-health-token")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			a.Handler.ServeHTTP(rec, req)
			want := 200
			if token == "" {
				want = 503
			}
			if rec.Code != want || !json.Valid(rec.Body.Bytes()) {
				t.Fatalf("root API routing: %d %s", rec.Code, rec.Body.String())
			}
			session := signInThroughMux(t, a.Handler)
			for _, path := range []string{"/health", "/health/plan", "/today", "/galaxy", "/projects", "/courses"} {
				rec := httptest.NewRecorder()
				pageReq := httptest.NewRequest("GET", path, nil)
				pageReq.AddCookie(session)
				a.Handler.ServeHTTP(rec, pageReq)
				if rec.Code != 200 {
					t.Errorf("%s: %d", path, rec.Code)
				}
				if strings.Contains(rec.Body.String(), "test-only-app-health-token") {
					t.Fatal("token leaked to HTML")
				}
			}
		})
	}
}

// TestShellAssetsLoadWithoutASession covers a failure that would make the app
// unusable rather than merely locked: the login page references the stylesheet,
// the favicon and the manifest, and the PWA needs its service worker. If any of
// those sat behind the session, a signed-out visitor would get an unstyled page
// or, in standalone mode, no way back in at all.
func TestShellAssetsLoadWithoutASession(t *testing.T) {
	a, err := New(Config{
		DBPath:        filepath.Join(t.TempDir(), "shell.db"),
		MigrationsDir: "../../migrations",
		TemplateDir:   "../../web/templates",
		StaticDir:     "../../web/static",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	for _, path := range []string{
		"/static/css/styles.css",
		"/manifest.webmanifest",
		"/service-worker.js",
		"/favicon.ico",
	} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			a.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Errorf("%s served %d without a session, want 200", path, rec.Code)
			}
		})
	}
}

// TestLoginPageRendersWhenSignedOut is the other half of the same concern: the
// way back in must work with no cookie at all.
func TestLoginPageRendersWhenSignedOut(t *testing.T) {
	a, err := New(Config{
		DBPath:        filepath.Join(t.TempDir(), "login.db"),
		MigrationsDir: "../../migrations",
		TemplateDir:   "../../web/templates",
		StaticDir:     "../../web/static",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/login served %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="password"`) {
		t.Error("login page has no password field")
	}
	if strings.Contains(body, "executing \"") {
		t.Errorf("login template failed to execute: %s", body)
	}
}
