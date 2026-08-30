package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootAndTodayRenderSamePage(t *testing.T) {
	h := testHandler(t)
	render := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status %d: %s", path, rec.Code, rec.Body.String())
		}
		return rec
	}

	root := render("/")
	today := render("/today")
	if root.Body.String() != today.Body.String() {
		t.Fatal("/ and /today did not render identical Today pages")
	}
}

func TestPrimaryNavigationIsReduced(t *testing.T) {
	body := renderBody(t, "today.html", TodayData{Page: "today", Today: "2026-08-28"})
	start := strings.Index(body, `<nav aria-label="Primary">`)
	if start < 0 {
		t.Fatal("primary navigation not found")
	}
	end := strings.Index(body[start:], `</nav>`)
	if end < 0 {
		t.Fatal("primary navigation end not found")
	}
	nav := body[start : start+end]
	for _, label := range []string{"Today", "Galaxy", "Add", "More"} {
		if !strings.Contains(nav, label) {
			t.Fatalf("primary navigation missing %q", label)
		}
	}
	for _, oldLabel := range []string{"Dashboard", "Projects", "Roadmaps"} {
		if strings.Contains(nav, ">"+oldLabel+"</a>") {
			t.Fatalf("old primary navigation item %q still visible", oldLabel)
		}
	}
	if strings.Contains(body, `<footer class="foot">`) {
		t.Fatal("legacy secondary footer still renders")
	}
}

func TestGalaxyAndSecondaryRoutesRemainReachable(t *testing.T) {
	h := testHandler(t)
	paths := []string{
		"/galaxy",
		"/projects",
		"/roadmap",
		"/courses",
		"/monthly",
		"/analytics",
		"/health",
		"/archive",
		"/tasks",
		"/goals",
		"/reminders",
		"/settings",
		"/plan-import",
	}
	for _, path := range paths {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status %d", path, rec.Code)
		}
	}
	dashboard := httptest.NewRecorder()
	h.ServeHTTP(dashboard, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	if dashboard.Code != http.StatusPermanentRedirect || dashboard.Header().Get("Location") != "/today" {
		t.Fatalf("legacy dashboard redirect = %d %q", dashboard.Code, dashboard.Header().Get("Location"))
	}
}
