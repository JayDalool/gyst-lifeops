package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
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

func TestPrimaryNavigationHasFiveDestinations(t *testing.T) {
	body := renderBody(t, "today.html", TodayData{Page: "today", Today: "2026-08-28"})
	start := strings.Index(body, `<nav aria-label="Primary"`)
	if start < 0 {
		t.Fatal("primary navigation not found")
	}
	end := strings.Index(body[start:], `</nav>`)
	if end < 0 {
		t.Fatal("primary navigation end not found")
	}
	nav := body[start : start+end]
	for _, label := range []string{"Today", "Projects", "Health", "Courses", "Menu"} {
		if !strings.Contains(nav, label) {
			t.Fatalf("primary navigation missing %q", label)
		}
	}
	if strings.Contains(nav, "<summary>Add</summary>") {
		t.Fatal("Add occupies a primary destination")
	}
	for _, oldLabel := range []string{"Dashboard", "Add"} {
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
		"/today",
		"/more",
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

func TestHealthFoundation(t *testing.T) {
	h := testHandler(t) // Uses t.TempDir; never the application database.
	for _, agent := range []string{"Mozilla/5.0 (iPhone)", "Mozilla/5.0 (X11; Linux x86_64)"} {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("User-Agent", agent)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("health status %d", rec.Code)
		}
		body := rec.Body.String()
		more := httptest.NewRecorder()
		h.ServeHTTP(more, httptest.NewRequest(http.MethodGet, "/more", nil))
		for _, path := range []string{"/roadmap", "/monthly", "/analytics", "/archive", "/goals", "/reminders"} {
			if !strings.Contains(more.Body.String(), `href="`+path+`"`) {
				t.Errorf("secondary route unreachable via More: %s", path)
			}
		}
		// No projection is implemented, so the page must not imply one. The
		// demo markup previously showed a "projected total burn" and an
		// "estimated" balance derived from it.
		// Ban the labels that presented a projection as a figure, not the word
		// itself: the energy card explains in prose that nothing is projected,
		// and that disclaimer is worth keeping.
		for _, banned := range []string{"Projected total burn", "Estimated energy balance", "Projected balance"} {
			if strings.Contains(body, banned) {
				t.Errorf("health page shows a projection it does not compute: %q", banned)
			}
		}
		for _, text := range []string{"Health sections", "Summary", "Nutrition", "Training", "Progress", "Plan", "Today's energy", "food calories remaining", "Observed energy balance", "Next meal", "Plan meal", "Start workout", "Activity", "Weight trend", "Strength trend", "Health goals", "Health tasks", "Gym week", "Live app data", "Week, month and year summaries are not built yet", "viewport-fit=cover", "data-period=\"Year\""} {
			if !strings.Contains(body, text) {
				t.Errorf("health missing %q", text)
			}
		}
		for _, path := range []string{"/today", "/galaxy", "/projects", "/health", "/courses", "/more", "/tasks", "/settings", "/plan-import"} {
			if !strings.Contains(body, `href="`+path+`"`) {
				t.Errorf("missing navigation %s", path)
			}
		}
	}
}

func TestHealthPreservesLiveContent(t *testing.T) {
	body := renderBody(t, "health.html", HealthData{
		Page:        "health",
		HealthGoals: []models.GoalWithProgress{{Goal: models.Goal{ID: 72, Title: "Weekly walking goal", TargetValue: 5, CurrentValue: 2}, Progress: 40}},
		HealthTasks: []models.Task{{ID: 83, Title: "Evening stretch", RepeatType: "daily", Priority: "medium"}},
	})
	for _, text := range []string{"Weekly walking goal", `href="/goals/72"`, "Evening stretch", `href="/tasks/83"`, "40%"} {
		if !strings.Contains(body, text) {
			t.Errorf("live health content missing %q", text)
		}
	}
}
