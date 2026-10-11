package repository

import (
	"database/sql"
	"errors"
	"testing"

	"lifeops-tracker/internal/models"
)

func newGoal(t *testing.T, repo *Repository, target, current float64) int64 {
	t.Helper()
	g := &models.Goal{Title: "Read pages", TargetValue: target, CurrentValue: current, Unit: "pages"}
	if err := repo.CreateGoal(g, "2026-03-10"); err != nil {
		t.Fatalf("create goal: %v", err)
	}
	return g.ID
}

func goalState(t *testing.T, repo *Repository, id int64) (float64, string) {
	t.Helper()
	g, err := repo.GoalByID(id)
	if err != nil {
		t.Fatalf("goal by id: %v", err)
	}
	return g.CurrentValue, g.Status
}

func addEntry(t *testing.T, repo *Repository, id int64, date string, v float64) {
	t.Helper()
	if err := repo.AddGoalProgress(id, date, v, ""); err != nil {
		t.Fatalf("add progress: %v", err)
	}
}

func TestBackdatedEntryKeepsNewerValueAndCompleted(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 10, 0)
	addEntry(t, repo, id, "2026-03-10", 12)
	addEntry(t, repo, id, "2026-03-01", 5)
	if v, s := goalState(t, repo, id); v != 12 || s != "completed" {
		t.Fatalf("got %v %q, want 12 completed", v, s)
	}
}

func TestNewerEntryUpdatesCurrent(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 100, 0)
	addEntry(t, repo, id, "2026-03-01", 5)
	addEntry(t, repo, id, "2026-03-05", 8)
	if v, s := goalState(t, repo, id); v != 8 || s != "active" {
		t.Fatalf("got %v %q, want 8 active", v, s)
	}
}

func TestSameDateLatestIDWins(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 100, 0)
	addEntry(t, repo, id, "2026-03-05", 8)
	addEntry(t, repo, id, "2026-03-05", 6)
	if v, _ := goalState(t, repo, id); v != 6 {
		t.Fatalf("got %v, want 6", v)
	}
}

func TestReachingTargetCompletes(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 10, 0)
	addEntry(t, repo, id, "2026-03-05", 10)
	if _, s := goalState(t, repo, id); s != "completed" {
		t.Fatalf("status %q, want completed", s)
	}
}

func TestDroppingBelowTargetReactivates(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 10, 0)
	addEntry(t, repo, id, "2026-03-05", 12)
	addEntry(t, repo, id, "2026-03-06", 4)
	if v, s := goalState(t, repo, id); v != 4 || s != "active" {
		t.Fatalf("got %v %q, want 4 active", v, s)
	}
}

func TestArchivedStaysArchived(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 10, 0)
	if _, err := repo.db.Exec(`UPDATE goals SET status = 'archived' WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	addEntry(t, repo, id, "2026-03-05", 12)
	if v, s := goalState(t, repo, id); v != 12 || s != "archived" {
		t.Fatalf("got %v %q, want 12 archived", v, s)
	}
}

func TestMissingGoalWritesNothing(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.AddGoalProgress(9999, "2026-03-05", 3, "")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err = %v, want ErrNoRows", err)
	}
	var n int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM goal_progress_entries`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("%d entries written, want 0", n)
	}
}

func TestEmptyDateRejected(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 10, 0)
	if err := repo.AddGoalProgress(id, "", 3, ""); err == nil {
		t.Fatal("expected error for empty date")
	}
}

func TestGoalProgressEntriesOrder(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 100, 0)
	addEntry(t, repo, id, "2026-03-05", 8)
	addEntry(t, repo, id, "2026-03-01", 5)
	addEntry(t, repo, id, "2026-03-05", 6)
	entries, err := repo.GoalProgressEntries(id)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{5, 8, 6}
	if len(entries) != len(want) {
		t.Fatalf("%d entries, want %d", len(entries), len(want))
	}
	for i, w := range want {
		if entries[i].Value != w {
			t.Fatalf("entry %d = %v, want %v", i, entries[i].Value, w)
		}
	}
}

func TestCreateGoalSetsInitialValue(t *testing.T) {
	repo := newTestRepo(t)
	id := newGoal(t, repo, 100, 7)
	if v, s := goalState(t, repo, id); v != 7 || s != "active" {
		t.Fatalf("got %v %q, want 7 active", v, s)
	}
	entries, _ := repo.GoalProgressEntries(id)
	if len(entries) != 1 {
		t.Fatalf("%d entries, want 1", len(entries))
	}
}
