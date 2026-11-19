package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-138: goal dates are validated and the initial entry is never future-dated.
func TestCreateGoalFutureStartDatesInitialEntryToday(t *testing.T) {
	repo := newTestRepo(t)
	g := &models.Goal{Title: "Planned", TargetValue: 10, CurrentValue: 7, StartDate: "2026-09-01"}
	if err := repo.CreateGoal(g, "2026-03-10"); err != nil {
		t.Fatal(err)
	}
	stored, _ := repo.GoalByID(g.ID)
	if stored.StartDate != "2026-09-01" {
		t.Fatalf("start date = %q", stored.StartDate)
	}
	entries, _ := repo.GoalProgressEntries(g.ID)
	if len(entries) != 1 || entries[0].EntryDate != "2026-03-10" {
		t.Fatalf("entries = %#v", entries)
	}
	// A real entry dated today is no longer shadowed by a future initial entry.
	addEntry(t, repo, g.ID, "2026-03-10", 9)
	if v, _ := goalState(t, repo, g.ID); v != 9 {
		t.Fatalf("current value = %v, want 9", v)
	}
}

func TestCreateGoalPastStartKeepsStartDate(t *testing.T) {
	repo := newTestRepo(t)
	g := &models.Goal{Title: "Past", TargetValue: 10, CurrentValue: 2, StartDate: "2026-03-01"}
	if err := repo.CreateGoal(g, "2026-03-10"); err != nil {
		t.Fatal(err)
	}
	entries, _ := repo.GoalProgressEntries(g.ID)
	if len(entries) != 1 || entries[0].EntryDate != "2026-03-01" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestCreateGoalRejectsBadDatesWritingNothing(t *testing.T) {
	repo := newTestRepo(t)
	for _, c := range [][2]string{
		{"2026-3-1", ""}, {"2026-02-30", ""}, {"0000-01-01", ""}, {"", "2026-13-01"},
		{"2026-05-01", "2026-04-01"},
	} {
		g := &models.Goal{Title: "Bad", TargetValue: 10, CurrentValue: 1, StartDate: c[0], EndDate: c[1]}
		if err := repo.CreateGoal(g, "2026-03-10"); err == nil {
			t.Errorf("start %q end %q: want error", c[0], c[1])
		}
	}
	goals, _ := repo.ListGoals(true)
	if len(goals) != 0 {
		t.Fatalf("%d goals written", len(goals))
	}
}
