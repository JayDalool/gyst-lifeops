package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

// Project mission items and roadmap milestones with due dates must surface for
// the calendar feed (carrying their project name and uuid).
func TestDeadlineQueries(t *testing.T) {
	repo := testRepository(t)

	if _, err := repo.db.Exec(`INSERT INTO projects (id, name) VALUES (1, 'Real Estate AI')`); err != nil {
		t.Fatalf("project: %v", err)
	}
	// A dated pending mission, a dated completed mission (excluded), an undated one (excluded).
	if err := repo.CreateProjectItem(&models.ProjectItem{ProjectID: 1, Title: "Ship API", DueDate: "2026-07-10"}); err != nil {
		t.Fatalf("item: %v", err)
	}
	if err := repo.CreateProjectItem(&models.ProjectItem{ProjectID: 1, Title: "No date"}); err != nil {
		t.Fatalf("item2: %v", err)
	}
	if _, err := repo.db.Exec(`UPDATE project_items SET status='completed' WHERE title='No date'`); err != nil {
		t.Fatal(err)
	}

	items, err := repo.ProjectItemDeadlines()
	if err != nil {
		t.Fatalf("ProjectItemDeadlines: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Ship API" || items[0].ProjectName != "Real Estate AI" || items[0].UUID == "" {
		t.Fatalf("unexpected item deadlines: %+v", items)
	}

	// A milestone (no public create method — milestones come from roadmap import).
	if _, err := repo.db.Exec(`INSERT INTO roadmap_milestones (project_id, title, status, due_date) VALUES (1, 'MVP', 'pending', '2026-08-01')`); err != nil {
		t.Fatalf("milestone: %v", err)
	}
	ms, err := repo.MilestoneDeadlines()
	if err != nil {
		t.Fatalf("MilestoneDeadlines: %v", err)
	}
	if len(ms) != 1 || ms[0].Title != "MVP" || ms[0].ProjectName != "Real Estate AI" || ms[0].UUID == "" {
		t.Fatalf("unexpected milestone deadlines: %+v", ms)
	}

	// Reminders: a standalone reminder must surface; a task-derived one must NOT
	// (it duplicates the task's own due-date event).
	if _, err := repo.db.Exec(`INSERT INTO reminders (source_type, source_id, title, reminder_date, reminder_time)
		VALUES ('manual', NULL, 'Call accountant', '2026-07-15', '09:00')`); err != nil {
		t.Fatalf("standalone reminder: %v", err)
	}
	if _, err := repo.db.Exec(`INSERT INTO reminders (source_type, source_id, title, reminder_date, reminder_time)
		VALUES ('task', 1, 'Task mirror', '2026-07-10', '08:00')`); err != nil {
		t.Fatalf("task reminder: %v", err)
	}
	rem, err := repo.ReminderDeadlines()
	if err != nil {
		t.Fatalf("ReminderDeadlines: %v", err)
	}
	if len(rem) != 1 || rem[0].Title != "Call accountant" || rem[0].Time != "09:00" || rem[0].UUID == "" {
		t.Fatalf("expected only the standalone reminder, got %+v", rem)
	}
}

func TestAppSettingsRoundTrip(t *testing.T) {
	repo := testRepository(t)

	got, err := repo.GetSetting("missing")
	if err != nil || got != "" {
		t.Fatalf("missing key should be empty: %q err %v", got, err)
	}
	if err := repo.SetSetting("k", "v1"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got, _ := repo.GetSetting("k"); got != "v1" {
		t.Fatalf("get after set = %q", got)
	}
	if err := repo.SetSetting("k", "v2"); err != nil { // upsert
		t.Fatalf("upsert: %v", err)
	}
	if got, _ := repo.GetSetting("k"); got != "v2" {
		t.Fatalf("get after upsert = %q", got)
	}
	if err := repo.DeleteSetting("k"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got, _ := repo.GetSetting("k"); got != "" {
		t.Fatalf("get after delete = %q", got)
	}
}
