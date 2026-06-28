package handlers

import (
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

func testHandler(t *testing.T) *Handler {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	h := New(services.New(repository.New(database)), "../../web/templates")
	return h
}

func getProjects(t *testing.T, h *Handler, path string) string {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("GET %s status %d: %s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

// All-roadmaps view renders one section per roadmap, keeps subprojects nested
// under their parent, and orders featured-first.
func TestProjectsAllRoadmapsGrouped(t *testing.T) {
	h := testHandler(t)

	// The migration leaves a featured default roadmap (priority 1). Add a
	// higher-priority, non-featured roadmap; featured must still sort first.
	rmB := models.Roadmap{Name: "Sim Rig Build", Priority: 9, Status: "active", Privacy: "private"}
	if err := h.Service.Repo.CreateRoadmap(&rmB); err != nil {
		t.Fatalf("create roadmap: %v", err)
	}
	defaultRM, err := h.Service.Repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatalf("dashboard roadmap: %v", err)
	}

	// rmB: a top-level trip with a *locked* subproject (the flatten trap).
	top := models.Project{Name: "Rig Chassis", ProjectKind: "main", Status: "active", RoadmapID: rmB.ID, AutoUnlock: true}
	if err := h.Service.Repo.CreateProject(&top); err != nil {
		t.Fatalf("create top: %v", err)
	}
	sub := models.Project{Name: "Pedal Mod", ProjectKind: "subproject", Status: "locked", RoadmapID: rmB.ID, ParentProjectID: top.ID, AutoUnlock: true}
	if err := h.Service.Repo.CreateProject(&sub); err != nil {
		t.Fatalf("create sub: %v", err)
	}
	// default roadmap: a top-level trip of its own.
	other := models.Project{Name: "LifeOps Core", ProjectKind: "main", Status: "active", RoadmapID: defaultRM.ID, AutoUnlock: true}
	if err := h.Service.Repo.CreateProject(&other); err != nil {
		t.Fatalf("create other: %v", err)
	}

	body := getProjects(t, h, "/projects")

	if n := strings.Count(body, "roadmap-section-head"); n < 2 {
		t.Fatalf("expected a section per roadmap (>=2), got %d", n)
	}
	// Featured default roadmap section comes before the higher-priority one.
	if strings.Index(body, defaultRM.Name) > strings.Index(body, "Sim Rig Build") {
		t.Fatalf("featured roadmap must sort first")
	}
	// The locked subproject nests as a stop, never as a top-level locked card.
	if !strings.Contains(body, "Pedal Mod") {
		t.Fatalf("subproject should still be visible (nested)")
	}
	if strings.Contains(body, `locked-card`) && strings.Contains(body, `<h2>Pedal Mod</h2>`) {
		t.Fatalf("subproject was flattened into a top-level planned card")
	}
}

// Selecting a specific roadmap keeps the single filtered deck (radio tabs), not
// the grouped layout.
func TestProjectsSpecificRoadmapSingleView(t *testing.T) {
	h := testHandler(t)
	rm, err := h.Service.Repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatalf("dashboard roadmap: %v", err)
	}
	p := models.Project{Name: "Solo Trip", ProjectKind: "main", Status: "active", RoadmapID: rm.ID, AutoUnlock: true}
	if err := h.Service.Repo.CreateProject(&p); err != nil {
		t.Fatalf("create: %v", err)
	}

	body := getProjects(t, h, "/projects?roadmap="+strconv.FormatInt(rm.ID, 10))
	if strings.Contains(body, "roadmap-section-head") {
		t.Fatalf("specific-roadmap view must not render group sections")
	}
	if !strings.Contains(body, "deck-panel") {
		t.Fatalf("specific-roadmap view should render the single filtered deck")
	}
	if !strings.Contains(body, "Solo Trip") {
		t.Fatalf("selected roadmap's trip should render")
	}
}

func TestSortRoadmapGroupsOrdering(t *testing.T) {
	groups := []RoadmapGroup{
		{Roadmap: models.Roadmap{Name: "B Mid", Priority: 5, UpdatedAt: "2026-01-01 00:00:01"}},
		{Roadmap: models.Roadmap{Name: "A Mid", Priority: 5, UpdatedAt: "2026-01-01 00:00:01"}},
		{Roadmap: models.Roadmap{Name: "High", Priority: 9}},
		{Roadmap: models.Roadmap{Name: "Featured Low", Priority: 1, Featured: true}},
		{Roadmap: models.Roadmap{Name: "Recent Mid", Priority: 5, UpdatedAt: "2026-02-01 00:00:00"}},
	}
	sortRoadmapGroups(groups)
	got := make([]string, len(groups))
	for i, g := range groups {
		got[i] = g.Roadmap.Name
	}
	want := []string{"Featured Low", "High", "Recent Mid", "A Mid", "B Mid"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordering wrong:\n got  %v\n want %v", got, want)
		}
	}
}

func TestRenderProjectsGroupedMultipleGroups(t *testing.T) {
	groups := []RoadmapGroup{
		{
			Roadmap: models.Roadmap{ID: 1, Name: "AI Engineer", Priority: 10, Featured: true, Privacy: "private", Status: "active"},
			Count:   2,
			Tree: models.ProjectTree{
				Main:     []models.ProjectWithProgress{{Project: models.Project{ID: 1, Name: "Lead Scorer", Status: "active"}, Progress: 40}},
				Children: map[int64][]models.ProjectWithProgress{1: {{Project: models.Project{ID: 2, Name: "Phase 1", Status: "locked", ProjectKind: "subproject"}}}},
				Upcoming: []models.ProjectWithProgress{{Project: models.Project{ID: 3, Name: "Healthcare AI", Status: "locked"}}},
			},
		},
		{
			Roadmap: models.Roadmap{ID: 2, Name: "Sim Rig", Priority: 7, Privacy: "public", Status: "active"},
			Count:   0,
			Tree:    models.ProjectTree{Children: map[int64][]models.ProjectWithProgress{}},
		},
	}
	body := renderBody(t, "projects_list.html", ProjectListData{
		Page:          "projects",
		RoadmapNames:  map[int64]string{},
		Grouped:       true,
		Groups:        groups,
		AnyProjects:   true,
		OpenAllGroups: true,
	})
	if strings.Count(body, "roadmap-section-head") != 2 {
		t.Fatalf("expected two group sections")
	}
	// Section header carries name, priority, dashboard/featured + count.
	for _, want := range []string{"AI Engineer", "P10", "Dashboard", "2 trips", "Sim Rig"} {
		if !strings.Contains(body, want) {
			t.Fatalf("section header missing %q", want)
		}
	}
}

// When every roadmap is empty, show the single create-project empty state, not
// a stack of empty sections.
func TestRenderProjectsAllRoadmapsEmpty(t *testing.T) {
	body := renderBody(t, "projects_list.html", ProjectListData{
		Page:         "projects",
		RoadmapNames: map[int64]string{},
		Grouped:      true,
		AnyProjects:  false,
		Groups: []RoadmapGroup{
			{Roadmap: models.Roadmap{ID: 1, Name: "Empty A", Status: "active", Privacy: "private"}, Tree: models.ProjectTree{Children: map[int64][]models.ProjectWithProgress{}}},
			{Roadmap: models.Roadmap{ID: 2, Name: "Empty B", Status: "active", Privacy: "private"}, Tree: models.ProjectTree{Children: map[int64][]models.ProjectWithProgress{}}},
		},
	})
	if !strings.Contains(body, "Plot your first trip") {
		t.Fatalf("all-empty view should show the create-project empty state")
	}
	if strings.Contains(body, "roadmap-section-head") {
		t.Fatalf("all-empty view must not render group sections")
	}
}
