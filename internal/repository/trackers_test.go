package repository

import (
	"path/filepath"
	"testing"

	"lifeops-tracker/internal/db"
)

// newTestRepo spins up a throwaway SQLite database with the real migrations
// applied (which also seed the prayer/athkar/gym tracker_items).
func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "tracker_test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(database)
}

func TestToggleCreatesLog(t *testing.T) {
	repo := newTestRepo(t)
	const date = "2026-06-17"

	status, err := repo.ToggleTrackerLogByKey("prayer_fajr", date)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if status != "done" {
		t.Fatalf("first toggle status = %q, want %q", status, "done")
	}

	logs, err := repo.TrackerLogsBetween(date, date)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("got %d logs, want exactly 1", len(logs))
	}
	if logs[0].Status != "done" {
		t.Fatalf("log status = %q, want %q", logs[0].Status, "done")
	}
}

func TestToggleUpdatesExistingLogNoDuplicate(t *testing.T) {
	repo := newTestRepo(t)
	const date = "2026-06-17"

	if _, err := repo.ToggleTrackerLogByKey("prayer_fajr", date); err != nil {
		t.Fatalf("first toggle: %v", err)
	}
	status, err := repo.ToggleTrackerLogByKey("prayer_fajr", date)
	if err != nil {
		t.Fatalf("second toggle: %v", err)
	}
	if status != "not_done" {
		t.Fatalf("second toggle status = %q, want %q", status, "not_done")
	}

	logs, err := repo.TrackerLogsBetween(date, date)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("got %d logs after two toggles, want exactly 1 (no duplicate)", len(logs))
	}
	if logs[0].Status != "not_done" {
		t.Fatalf("log status = %q, want %q", logs[0].Status, "not_done")
	}
}

func TestGymCycleLogic(t *testing.T) {
	repo := newTestRepo(t)
	const date = "2026-06-17"

	// Gym default is "rest"; cycle is rest -> done -> not_done -> rest -> ...
	want := []string{"done", "not_done", "rest", "done"}
	for i, expected := range want {
		got, err := repo.ToggleTrackerLogByKey("gym", date)
		if err != nil {
			t.Fatalf("toggle %d: %v", i, err)
		}
		if got != expected {
			t.Fatalf("toggle %d status = %q, want %q", i, got, expected)
		}
	}
}
