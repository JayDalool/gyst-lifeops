package repository

import (
	"database/sql"
	"errors"
	"testing"

	"lifeops-tracker/internal/models"
)

func TestUpdateGoalSavesEditableFieldsOnly(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 10, 0)
	addEntry(t, repo, id, "2026-03-10", 4)
	addEntry(t, repo, id, "2026-03-01", 9) // backdated, older: must not win

	edit := models.Goal{ID: id, Title: "Read more", Description: "d", Type: "monthly", Category: "coding",
		TargetValue: 30, Unit: "hours", StartDate: "2026-03-01", EndDate: "2026-03-31", Status: "completed"}
	if err := repo.UpdateGoal(&edit); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := repo.GoalByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Read more" || got.Description != "d" || got.Type != "monthly" || got.Category != "coding" ||
		got.TargetValue != 30 || got.Unit != "hours" || got.StartDate != "2026-03-01" || got.EndDate != "2026-03-31" || got.Status != "completed" {
		t.Fatalf("fields not saved: %+v", got)
	}
	// #135 invariant: current value still follows the latest-dated entry.
	if got.CurrentValue != 4 {
		t.Fatalf("current_value = %v, want 4 (latest-dated entry)", got.CurrentValue)
	}
	entries, _ := repo.GoalProgressEntries(id)
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	addEntry(t, repo, id, "2026-02-01", 99)
	if v, _ := goalState(t, repo, id); v != 4 {
		t.Fatalf("after edit and backdated entry current = %v, want 4", v)
	}
	edit.StartDate, edit.EndDate = "", ""
	if err := repo.UpdateGoal(&edit); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GoalByID(id); got.StartDate != "" || got.EndDate != "" {
		t.Fatalf("dates not cleared: %+v", got)
	}
}

func TestUpdateGoalMissingOrArchivedIsNoRows(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 10, 0)
	if err := repo.UpdateGoal(&models.Goal{ID: 9999, Title: "x", Type: "yearly", Category: "other", Unit: "u", Status: "active"}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown id err = %v, want ErrNoRows", err)
	}
	if err := repo.ArchiveGoal(id); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateGoal(&models.Goal{ID: id, Title: "changed", Type: "yearly", Category: "other", Unit: "u", Status: "active"}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("archived err = %v, want ErrNoRows", err)
	}
	if got, _ := repo.GoalByID(id); got.Title != "Read pages" {
		t.Fatalf("archived goal was changed: %+v", got)
	}
}
