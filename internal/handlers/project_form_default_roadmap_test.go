package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-076 review: the empty Roadmap choice is saved into the active dashboard
// roadmap, so the form must name that destination, not say "Unassigned", and
// say so when the move changes the project's privacy.
func TestProjectFormNamesDefaultRoadmap(t *testing.T) {
	h, database := cycleHandler(t)
	if _, err := database.Exec(`UPDATE roadmaps SET status = 'archived', archived_at = CURRENT_TIMESTAMP`); err != nil {
		t.Fatal(err)
	}
	existing := cycleProject(t, h, "Fictional Loose Project", 0)

	// No active roadmap: the option stays "Unassigned" and there is no note.
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

	// An active private roadmap becomes the labelled default.
	rm := models.Roadmap{Name: "Fictional Harbor Plan", Priority: 5, Status: "active", Privacy: "private"}
	if err := h.Service.Repo.CreateRoadmap(&rm); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/projects/new", fmt.Sprintf("/projects/%d/edit", existing.ID)} {
		body := serve(h, http.MethodGet, path, nil).Body.String()
		sel := selectBlock(t, body, "roadmap_id")
		if !strings.Contains(sel, `<option value="">Fictional Harbor Plan (default)</option>`) {
			t.Errorf("%s: default option not labelled with the roadmap: %s", path, sel)
		}
		if strings.Contains(sel, ">Unassigned<") {
			t.Errorf("%s: still says Unassigned", path)
		}
	}

	// A new project is public today, so the private default needs a note.
	body := serve(h, http.MethodGet, "/projects/new", nil).Body.String()
	if !strings.Contains(body, "Choosing the default moves this project to a private roadmap; it is public now.") {
		t.Error("new project form: missing privacy note")
	}
	// The existing project was adopted into that roadmap already: no change, no note.
	body = serve(h, http.MethodGet, fmt.Sprintf("/projects/%d/edit", existing.ID), nil).Body.String()
	if strings.Contains(body, "Choosing the default") {
		t.Error("edit form: note shown although privacy does not change")
	}
}
