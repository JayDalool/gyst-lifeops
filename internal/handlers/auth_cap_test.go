package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// GYST-VN-077: the renewed cookie never outlives the 90-day cap, and a capped
// session is treated exactly like any other lapsed one.

func capDay(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }

func visitAt(h *Handler, cookie *http.Cookie, method, path string, at time.Time) *httptest.ResponseRecorder {
	h.Service.Now = func() time.Time { return at }
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(recorder, request)
	return recorder
}

func sessionCookieOf(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			return c
		}
	}
	return nil
}

func TestCookieRenewalNeverPassesTheCapAndCappedSessionIsRefused(t *testing.T) {
	h := authHandler(t)
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	h.Service.Now = func() time.Time { return base }
	cookie := signIn(t, h)
	cap := base.Add(capDay(90))

	for _, d := range []int{25, 50, 75} {
		rec := visitAt(h, cookie, http.MethodGet, "/today", base.Add(capDay(d)))
		if rec.Code != http.StatusOK {
			t.Fatalf("+%dd status = %d, want 200", d, rec.Code)
		}
		if c := sessionCookieOf(t, rec); c != nil && c.Expires.After(cap) {
			t.Errorf("+%dd cookie Expires %v is past the cap %v", d, c.Expires, cap)
		}
	}

	rec := visitAt(h, cookie, http.MethodGet, "/today", base.Add(capDay(89)))
	if rec.Code != http.StatusOK {
		t.Fatalf("+89d status = %d, want 200", rec.Code)
	}
	renewed := sessionCookieOf(t, rec)
	if renewed == nil {
		t.Fatal("cookie not renewed at +89d")
	}
	if !renewed.Expires.Equal(cap.Truncate(time.Second)) {
		t.Errorf("renewed Expires = %v, want exactly %v", renewed.Expires, cap)
	}
	if renewed.MaxAge != 0 {
		t.Errorf("Max-Age = %d; cookie lifetime must come only from the capped Expires", renewed.MaxAge)
	}

	// +91d is inside the 30-day idle window but past the cap.
	rec = visitAt(h, cookie, http.MethodGet, "/today", base.Add(capDay(91)))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login?next=%2Ftoday" {
		t.Errorf("GET at +91d = %d -> %q, want 303 /login?next=%%2Ftoday", rec.Code, rec.Header().Get("Location"))
	}
	post := visitAt(h, cookie, http.MethodPost, "/tasks", base.Add(capDay(91)))
	if post.Code != http.StatusUnauthorized {
		t.Errorf("POST at +91d = %d, want 401", post.Code)
	}
}

func TestCorruptCreatedAtIsTreatedAsUnauthenticated(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	h := New(services.New(repository.New(database)), "../../web/templates")
	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	h.Auth = services.AuthConfig{PasswordHash: string(hash)}

	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	h.Service.Now = func() time.Time { return base }
	cookie := signIn(t, h)
	if _, err := database.Exec(`UPDATE sessions SET created_at = 'garbage'`); err != nil {
		t.Fatalf("corrupt: %v", err)
	}

	rec := visitAt(h, cookie, http.MethodGet, "/today", base.Add(time.Hour))
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/login") {
		t.Errorf("corrupt created_at served %d -> %q; must redirect to /login", rec.Code, rec.Header().Get("Location"))
	}
}
