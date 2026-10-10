package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

func reminderDate(t *testing.T, repo *Repository, id int64) string {
	t.Helper()
	var d string
	if err := repo.db.QueryRow(`SELECT reminder_date FROM reminders WHERE source_type = 'task' AND source_id = ?`, id).Scan(&d); err != nil {
		t.Fatalf("reminder row: %v", err)
	}
	return d
}

func TestMoveOverdueTaskToDate(t *testing.T) {
	repo := newTestRepo(t)
	mk := func(task models.Task) int64 {
		t.Helper()
		if task.RepeatType == "" {
			task.RepeatType = "none"
		}
		if err := repo.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
		return task.ID
	}

	id := mk(models.Task{Title: "Fictional overdue", DueDate: "2026-10-01", ReminderTime: "08:15"})
	if got, err := repo.MoveOverdueTaskToDate(id, "2026-10-09"); err != nil || !got {
		t.Fatalf("move = %v, %v; want true", got, err)
	}
	task, _ := repo.TaskByID(id)
	if task.DueDate != "2026-10-09" || reminderDate(t, repo, id) != "2026-10-09" {
		t.Fatalf("due %s reminder %s; want both 2026-10-09", task.DueDate, reminderDate(t, repo, id))
	}
	if got, err := repo.MoveOverdueTaskToDate(id, "2026-10-09"); err != nil || got {
		t.Fatalf("second call = %v, %v; want a no-op", got, err)
	}

	noOps := map[string]int64{
		"due today":   mk(models.Task{Title: "Due today", DueDate: "2026-10-09"}),
		"future":      mk(models.Task{Title: "Future", DueDate: "2026-10-20"}),
		"repeating":   mk(models.Task{Title: "Repeats", DueDate: "2026-10-01", RepeatType: "daily"}),
		"no due date": mk(models.Task{Title: "Undated"}),
		"completed":   mk(models.Task{Title: "Finished", DueDate: "2026-10-01"}),
		"skipped":     mk(models.Task{Title: "Skipped", DueDate: "2026-10-01"}),
		"archived":    mk(models.Task{Title: "Archived", DueDate: "2026-10-01"}),
	}
	repo.db.Exec(`UPDATE tasks SET status='completed' WHERE id = ?`, noOps["completed"])
	repo.db.Exec(`UPDATE tasks SET status='skipped' WHERE id = ?`, noOps["skipped"])
	if err := repo.ArchiveTask(noOps["archived"]); err != nil {
		t.Fatal(err)
	}
	for name, tid := range noOps {
		before, _ := repo.TaskByID(tid)
		if got, err := repo.MoveOverdueTaskToDate(tid, "2026-10-09"); err != nil || got {
			t.Errorf("%s: move = %v, %v; want a no-op", name, got, err)
		}
		after, _ := repo.TaskByID(tid)
		if after != before {
			t.Errorf("%s: row changed %#v -> %#v", name, before, after)
		}
	}
}
