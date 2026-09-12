package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func primaryMarkup(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<nav aria-label="Primary"`)
	if start < 0 {
		t.Fatal("primary nav absent")
	}
	end := strings.Index(body[start:], "</nav>")
	if end < 0 {
		t.Fatal("primary nav not closed")
	}
	return body[start : start+end]
}
func TestUIShellFiveLinksAndNestedSelection(t *testing.T) {
	h := testHandler(t)
	for _, tt := range []struct{ path, current string }{{"/", "today"}, {"/today", "today"}, {"/projects", "projects"}, {"/projects/new", "projects"}, {"/galaxy", "projects"}, {"/roadmap", "projects"}, {"/health", "health"}, {"/health/plan", "health"}, {"/courses", "courses"}, {"/courses/new", "courses"}, {"/more", "more"}, {"/settings", "more"}, {"/tasks/new", "more"}, {"/plan-import", "more"}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if rec.Code != 200 {
			t.Fatalf("%s: %d", tt.path, rec.Code)
		}
		nav := primaryMarkup(t, rec.Body.String())
		if strings.Count(nav, "<a ") != 5 || strings.Count(nav, `aria-current="page"`) != 1 {
			t.Fatalf("invalid nav: %s", nav)
		}
		for _, path := range []string{"today", "projects", "health", "courses", "more"} {
			if !strings.Contains(nav, `href="/`+path+`"`) {
				t.Fatalf("missing destination %s", path)
			}
		}
		if !strings.Contains(nav, `href="/`+tt.current+`" aria-current="page"`) {
			t.Fatalf("wrong active state at %s: %s", tt.path, nav)
		}
		if strings.Count(nav, "<svg") != 5 {
			t.Fatal("navigation must use five SVG icons")
		}
		for _, hook := range []string{`data-command-open`, `data-quick-task`, `href="/tasks/new"`, `href="/projects/new"`, `href="/courses/new"`, `href="/plan-import"`, `href="#main-content"`} {
			if !strings.Contains(rec.Body.String(), hook) {
				t.Fatalf("%s lost action %s", tt.path, hook)
			}
		}
	}
}
func TestMoreMethodAndReachability(t *testing.T) {
	h := testHandler(t)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, "/more", nil))
		if rec.Code != 405 || rec.Header().Get("Allow") != "GET" {
			t.Fatalf("More accepted %s: %d", method, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/more/not-real", nil))
	if rec.Code != 404 {
		t.Fatal("More accepted unknown subroute")
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/more", nil))
	for _, path := range []string{"/today", "/tasks", "/reminders", "/monthly", "/galaxy", "/roadmap", "/goals", "/analytics", "/plan-import", "/archive", "/settings"} {
		if !strings.Contains(rec.Body.String(), `href="`+path+`"`) {
			t.Fatalf("More missing %s", path)
		}
		dest := httptest.NewRecorder()
		h.ServeHTTP(dest, httptest.NewRequest(http.MethodGet, path, nil))
		if dest.Code != 200 {
			t.Fatalf("More links to broken destination %s", path)
		}
	}
	denied := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/more", nil))
	if denied.Code != 303 || !strings.HasPrefix(denied.Header().Get("Location"), "/login") {
		t.Fatalf("More bypassed authentication: %d", denied.Code)
	}
}
