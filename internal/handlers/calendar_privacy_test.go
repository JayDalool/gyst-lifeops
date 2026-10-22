package handlers

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// Every .ics route built on projectTaskEvents leaves out private and
// local-only projects (target, item, milestone) and keeps public ones.
func TestCalendarRoutesOmitPrivateAndLocalOnlyProjects(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	// Migration 006 makes the default roadmap private; this test covers
	// project-level flags, so make that roadmap public first.
	if _, err := database.Exec(`UPDATE roadmaps SET privacy = 'public'`); err != nil {
		t.Fatalf("make roadmap public: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	repo := repository.New(database)
	h := New(services.New(repo), "../../web/templates")
	h.AllowPublicFeeds = true // so the plain routes are reachable too

	mk := func(name string) int64 {
		p := models.Project{Name: name, Status: "active", TargetDate: "2026-08-01"}
		if err := repo.CreateProject(&p); err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateProjectItem(&models.ProjectItem{ProjectID: p.ID, Title: name + " item", DueDate: "2026-07-12"}); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`INSERT INTO roadmap_milestones (project_id, title, due_date) VALUES (?, ?, '2026-07-20')`, p.ID, name+" milestone"); err != nil {
			t.Fatal(err)
		}
		return p.ID
	}
	mk("Fictional Bakery Site")
	priv := mk("Fictional Garden Planner")
	local := mk("Fictional Boat Log")
	for id, col := range map[int64]string{priv: "is_private", local: "local_only"} {
		if _, err := database.Exec(`INSERT OR IGNORE INTO roadmap_project_meta (project_id) VALUES (?)`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`UPDATE roadmap_project_meta SET `+col+` = 1 WHERE project_id = ?`, id); err != nil {
			t.Fatal(err)
		}
	}

	token, err := h.Service.GenerateCalendarToken()
	if err != nil {
		t.Fatal(err)
	}
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		return rec
	}
	for _, path := range []string{
		"/calendar/feed/" + token + ".ics",
		"/calendar/feed/" + token + "/projects.ics",
		"/calendar.ics",
		"/projects/calendar.ics",
	} {
		rec := get(path)
		if rec.Code != 200 {
			t.Fatalf("%s = %d, want 200", path, rec.Code)
		}
		body := rec.Body.String()
		for _, want := range []string{"Target: Fictional Bakery Site", "Fictional Bakery Site item", "Fictional Bakery Site milestone"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s missing %q", path, want)
			}
		}
		for _, leak := range []string{"Fictional Garden Planner", "Fictional Boat Log"} {
			if strings.Contains(body, leak) {
				t.Errorf("%s leaks %q", path, leak)
			}
		}
	}
	if rec := get("/calendar/feed/wrongtoken.ics"); rec.Code != 404 {
		t.Fatalf("wrong token = %d, want 404", rec.Code)
	}
}
