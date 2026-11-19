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
// roadmap, so the form names that roadmap, posts its ID, and warns when the
// move would change the project's privacy.

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
		if !strings.Contains(selectBlock(t, body, "roadmap_id"), `<option value="">Unassigned</option>`) {
			t.Errorf("%s: want Unassigned option without an active roadmap", path)
		}
		if strings.Contains(body, "Choosing the default") {
			t.Errorf("%s: unexpected privacy note without an active roadmap", path)
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

func TestProjectFormPrivacyNote(t *testing.T) {
	h, database := cycleHandler(t)
	archiveAllRoadmapRows(t, database)
	existing := cycleProject(t, h, "Fictional Loose Project", 0) // no roadmap yet
	newTestRoadmap(t, h, "Fictional Harbor Plan", "private", 5)
	// The roadmap write adopted the loose project, so it is already private.

	const note = "Choosing the default changes this project from public to private."
	if body := serve(h, http.MethodGet, "/projects/new", nil).Body.String(); !strings.Contains(body, note) {
		t.Error("new project form: missing privacy note")
	}
	if body := serve(h, http.MethodGet, fmt.Sprintf("/projects/%d/edit", existing.ID), nil).Body.String(); strings.Contains(body, "Choosing the default") {
		t.Error("edit form: note shown although privacy does not change")
	}

	// A project left outside the roadmap and public shows the note on edit.
	outside := cycleProject(t, h, "Fictional Outside Project", 0)
	otherPublic := newTestRoadmap(t, h, "Fictional Open Plan", "public", 1)
	if _, err := database.Exec(`UPDATE projects SET roadmap_id = ? WHERE id = ?`, otherPublic.ID, outside.ID); err != nil {
		t.Fatal(err)
	}
	if body := serve(h, http.MethodGet, fmt.Sprintf("/projects/%d/edit", outside.ID), nil).Body.String(); !strings.Contains(body, note) {
		t.Error("edit form: missing privacy note for a public project")
	}

	// The project's own Private flag keeps it private either way: no note.
	if _, err := database.Exec(`INSERT INTO roadmap_project_meta (project_id, is_private) VALUES (?, 1)`, outside.ID); err != nil {
		t.Fatal(err)
	}
	if body := serve(h, http.MethodGet, fmt.Sprintf("/projects/%d/edit", outside.ID), nil).Body.String(); strings.Contains(body, "Choosing the default") {
		t.Error("edit form: note shown although the own Private flag keeps it private")
	}
}
