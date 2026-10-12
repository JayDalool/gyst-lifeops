package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// GYST-073: /learn is an alias of the course list handler and /courses keeps working.
func TestLearnAlias(t *testing.T) {
	h := testHandler(t)
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}
	learn, courses := get("/learn"), get("/courses")
	if learn.Code != 200 || courses.Code != 200 {
		t.Fatalf("learn=%d courses=%d", learn.Code, courses.Code)
	}
	if !strings.Contains(primaryMarkup(t, learn.Body.String()), `href="/learn" aria-current="page"`) {
		t.Fatal("Learn is not the current tab at /learn")
	}
	main := func(b string) string {
		i := strings.Index(b, "<main ")
		j := strings.Index(b, "</main>")
		return b[i:j]
	}
	if main(learn.Body.String()) != main(courses.Body.String()) {
		t.Fatal("/learn content differs from /courses")
	}
	if c := get("/learn/x").Code; c != 404 {
		t.Fatalf("/learn/x = %d", c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/learn", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /learn = %d", rec.Code)
	}
}

func TestLearnAuthNegatives(t *testing.T) {
	h := authHandler(t)
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/learn", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login?next=%2Flearn" {
		t.Fatalf("unauthenticated /learn = %d %s", rec.Code, rec.Header().Get("Location"))
	}
	cookie := signIn(t, h)
	r := httptest.NewRequest(http.MethodPost, "/learn", nil)
	r.Header.Set("Origin", "https://evil.example")
	r.Host = "gyst.zaylolabs.com"
	r.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site POST /learn = %d", rec.Code)
	}
}

func TestOldTopLevelURLsKeepStatus(t *testing.T) {
	h := testHandler(t)
	for _, p := range []string{"/", "/today", "/projects", "/galaxy", "/roadmap", "/health", "/health/training", "/health/nutrition", "/health/more", "/courses", "/courses/new", "/more", "/goals", "/tasks", "/reminders", "/monthly", "/analytics", "/archive", "/settings", "/plan-import", "/checklist"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != 200 {
			t.Errorf("%s = %d", p, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	if rec.Code != http.StatusPermanentRedirect || rec.Header().Get("Location") != "/today" {
		t.Errorf("/dashboard = %d %s", rec.Code, rec.Header().Get("Location"))
	}
}
