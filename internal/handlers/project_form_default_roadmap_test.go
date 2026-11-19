package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-076 review: an unassigned project is saved into the active dashboard
// roadmap, so the form names that roadmap and posts its ID.

func archiveAllRoadmapRows(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec(`UPDATE roadmaps SET status = 'archived', archived_at = CURRENT_TIMESTAMP`); err != nil {
		t.Fatal(err)
	}
}

func newTestRoadmap(t *testing.T, h *Handler, name, privacy string, priority int) models.Roadmap {
	t.Helper()
	rm := models.Roadmap{Name: name, Priority: priority, Status: "active", Privacy: privacy}
	if err := h.Service.Repo.CreateRoadmap(&rm); err != nil {
		t.Fatal(err)
	}
	return rm
}

func TestProjectFormWithoutActiveRoadmapStaysUnassigned(t *testing.T) {
	h, database := cycleHandler(t)
	archiveAllRoadmapRows(t, database)
	existing := cycleProject(t, h, "Fictional Loose Project", 0)
	for _, path := range []string{"/projects/new", fmt.Sprintf("/projects/%d/edit", existing.ID)} {
		rec := serve(h, http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status %d", path, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(selectBlock(t, body, "roadmap_id"), `<option value="" selected>Unassigned</option>`) {
			t.Errorf("%s: want Unassigned option without an active roadmap", path)
		}
	}
}

func TestProjectFormPostsDisplayedDefaultRoadmap(t *testing.T) {
	h, database := cycleHandler(t)
	archiveAllRoadmapRows(t, database)
	a := newTestRoadmap(t, h, "Fictional Harbor Plan", "private", 5)

	body := serve(h, http.MethodGet, "/projects/new", nil).Body.String()
	sel := selectBlock(t, body, "roadmap_id")
	want := fmt.Sprintf(`<option value="%d" selected>Fictional Harbor Plan (default)</option>`, a.ID)
	if !strings.Contains(sel, want) || strings.Contains(sel, `value=""`) || strings.Contains(sel, ">Unassigned<") {
		t.Fatalf("default option wrong: %s", sel)
	}
	form := formValues(t, body, "Fictional Form Project")
	form.Set("project_kind", "main")

	// Another tab makes roadmap B the default before this form is submitted.
	b := newTestRoadmap(t, h, "Fictional Lantern Plan", "public", 9)
	if got, _ := h.Service.Repo.GetDashboardRoadmap(); got.ID != b.ID {
		t.Fatalf("setup: default is %d, want %d", got.ID, b.ID)
	}
	rec := serve(h, http.MethodPost, "/projects/new", form)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST status %d: %s", rec.Code, rec.Body.String())
	}
	var roadmapID int64
	if err := database.QueryRow(`SELECT roadmap_id FROM projects WHERE name = 'Fictional Form Project'`).Scan(&roadmapID); err != nil {
		t.Fatal(err)
	}
	if roadmapID != a.ID {
		t.Fatalf("project saved to roadmap %d, want the displayed default %d", roadmapID, a.ID)
	}
}

// A prefilled ?roadmap= that points at an archived roadmap is not offered, so
// the form must fall back to the default (or Unassigned) and always have
// exactly one selected option; the posted value is then that option's.
func TestProjectFormArchivedPrefillFallsBack(t *testing.T) {
	h, database := cycleHandler(t)
	archiveAllRoadmapRows(t, database)
	old := newTestRoadmap(t, h, "Fictional Retired Plan", "public", 1)
	if err := h.Service.Repo.ArchiveRoadmap(old.ID); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/projects/new?roadmap=%d", old.ID)

	// No active roadmap: Unassigned is the one selected option.
	body := serve(h, http.MethodGet, path, nil).Body.String()
	sel := selectBlock(t, body, "roadmap_id")
	if strings.Count(sel, "selected") != 1 || !strings.Contains(sel, `<option value="" selected>Unassigned</option>`) {
		t.Fatalf("want only Unassigned selected: %s", sel)
	}

	// With an active default, that option is the one selected and posted.
	def := newTestRoadmap(t, h, "Fictional Harbor Plan", "private", 5)
	body = serve(h, http.MethodGet, path, nil).Body.String()
	sel = selectBlock(t, body, "roadmap_id")
	if strings.Count(sel, "selected") != 1 || !strings.Contains(sel, fmt.Sprintf(`<option value="%d" selected>Fictional Harbor Plan (default)</option>`, def.ID)) {
		t.Fatalf("want only the default selected: %s", sel)
	}
	if got := formValues(t, body, "x").Get("roadmap_id"); got != fmt.Sprint(def.ID) {
		t.Fatalf("submitted roadmap_id = %q, want %d", got, def.ID)
	}

	// A listed prefill still wins over the default.
	other := newTestRoadmap(t, h, "Fictional Lantern Plan", "public", 1)
	body = serve(h, http.MethodGet, fmt.Sprintf("/projects/new?roadmap=%d", other.ID), nil).Body.String()
	sel = selectBlock(t, body, "roadmap_id")
	if strings.Count(sel, "selected") != 1 || !strings.Contains(sel, fmt.Sprintf(`<option value="%d" selected>Fictional Lantern Plan</option>`, other.ID)) {
		t.Fatalf("want the prefilled roadmap selected: %s", sel)
	}
}

// The default comes from the same list the options are rendered from: a
// roadmap created after that list was read must not become a selected value
// that is not an option; the form falls back to Unassigned.
func TestDefaultRoadmapIsAlwaysAnOption(t *testing.T) {
	h, database := cycleHandler(t)
	archiveAllRoadmapRows(t, database)
	old := newTestRoadmap(t, h, "Fictional Harbor Plan", "public", 1)
	stale, err := h.Service.Repo.ListRoadmaps(false) // options read here
	if err != nil {
		t.Fatal(err)
	}

	def, err := h.defaultRoadmap(stale)
	if err != nil || def == nil || def.ID != old.ID {
		t.Fatalf("default = %+v, %v; want roadmap %d", def, err, old.ID)
	}
	if got := selectedRoadmapID(0, stale, nil, def); got != old.ID || !roadmapListed(stale, got) {
		t.Fatalf("selected %d is not a listed option", got)
	}

	// A newer default appears after the options were listed.
	newTestRoadmap(t, h, "Fictional Lantern Plan", "public", 9)
	def, err = h.defaultRoadmap(stale)
	if err != nil {
		t.Fatal(err)
	}
	if def != nil {
		t.Fatalf("default %d is not among the rendered options", def.ID)
	}
	if got := selectedRoadmapID(0, stale, nil, def); got != 0 {
		t.Fatalf("selected = %d, want 0 (Unassigned)", got)
	}
}
