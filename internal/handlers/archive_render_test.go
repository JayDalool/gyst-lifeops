package handlers

import (
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func TestArchiveShellEmpty(t *testing.T) {
	body := renderBody(t, "archive.html", ArchiveData{Page: "archive"})
	for _, line := range []string{"No archived tasks.", "No archived projects.", "No archived goals."} {
		if !strings.Contains(body, `<div class="gyst-empty"><p>`+line+`</p></div>`) {
			t.Errorf("missing empty line %q", line)
		}
	}
	if n := strings.Count(body, `class="gyst-row"`); n != 0 {
		t.Errorf("gyst-row count = %d, want 0", n)
	}
	if n := strings.Count(body, "<h1"); n != 1 {
		t.Errorf("<h1 count = %d, want 1", n)
	}
	if !strings.Contains(body, "gyst-scope") {
		t.Error("missing gyst-scope")
	}
	for _, bad := range []string{"page-head", "card tight", "--xp"} {
		if strings.Contains(body, bad) {
			t.Errorf("body contains legacy %q", bad)
		}
	}
}

func TestArchiveShellOneEach(t *testing.T) {
	body := renderBody(t, "archive.html", ArchiveData{
		Page:     "archive",
		Tasks:    []models.Task{{ID: 1, Title: "Fictional task", ArchivedAt: "2026-01-02 03:04:05"}},
		Projects: []models.ProjectWithProgress{{Project: models.Project{ID: 2, Name: "Fictional project"}, Progress: 40}},
		Goals:    []models.GoalWithProgress{{Goal: models.Goal{ID: 3, Title: "Fictional goal"}, Progress: 40}},
	})
	for _, want := range []string{`href="/tasks/1"`, `href="/projects/2"`, `href="/goals/3"`, "40% complete", "Fictional task", "Fictional project", "Fictional goal"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if n := strings.Count(body, `class="gyst-row"`); n != 3 {
		t.Errorf("gyst-row count = %d, want 3", n)
	}
	if strings.Contains(body, "gyst-empty") {
		t.Error("unexpected empty state")
	}
	if strings.Contains(body, "--xp") {
		t.Error("body contains --xp")
	}
}

func TestArchiveShellManyTasks(t *testing.T) {
	body := renderBody(t, "archive.html", ArchiveData{
		Page:  "archive",
		Tasks: []models.Task{{ID: 1, Title: "A"}, {ID: 2, Title: "B"}, {ID: 3, Title: "C"}},
	})
	if n := strings.Count(body, `href="/tasks/`); n != 3 {
		t.Errorf("task rows = %d, want 3", n)
	}
}
