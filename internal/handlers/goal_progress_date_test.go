package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-150: goal progress entries cannot be dated after the application's
// local today (narrows Issue #3 decision D4 for goal progress only).

const goalFutureMessage = "Progress can't be dated after today."

func newDateGoal(t *testing.T, h *Handler, target float64) models.Goal {
	t.Helper()
	goal := models.Goal{Title: "Read", Type: "yearly", TargetValue: target, StartDate: "2026-01-01", EndDate: "2027-12-31"}
	if err := h.Service.Repo.CreateGoal(&goal, h.Service.TodayString()); err != nil {
		t.Fatal(err)
	}
	return goal
}

func postGoalProgress(t *testing.T, h *Handler, id int64, date, value string) (int, string) {
	t.Helper()
	rec := postForm(t, h, fmt.Sprintf("/goals/%d/progress", id), url.Values{"entry_date": {date}, "value": {value}})
	return rec.Code, rec.Body.String()
}

func TestGoalProgressFutureDateRefusedWithoutWriting(t *testing.T) {
	for _, date := range []string{"2026-07-16", "2027-07-15"} {
		t.Run(date, func(t *testing.T) {
			h := testHandler(t)
			pinClock(t, h, "2026-07-16T04:30:00Z") // 2026-07-15 local
			goal := newDateGoal(t, h, 20)
			before, _ := h.Service.Repo.GoalByID(goal.ID)
			entriesBefore, _ := h.Service.Repo.GoalProgressEntries(goal.ID)

			code, body := postGoalProgress(t, h, goal.ID, date, "5")
			if code != http.StatusBadRequest {
				t.Fatalf("status %d; want 400", code)
			}
			if !strings.Contains(body, goalFutureMessage) {
				t.Errorf("body %q lacks the fixed message", body)
			}
			if strings.Contains(body, date) || strings.Contains(body, "5") {
				t.Errorf("body %q echoes the posted date or value", body)
			}
			after, _ := h.Service.Repo.GoalByID(goal.ID)
			entriesAfter, _ := h.Service.Repo.GoalProgressEntries(goal.ID)
			// sync_version is not on the model; the whole goal row (including
			// updated_at) must be unchanged.
			if len(entriesAfter) != len(entriesBefore) || after != before {
				t.Fatalf("a refused entry changed data: entries %d->%d goal %+v -> %+v",
					len(entriesBefore), len(entriesAfter), before, after)
			}
		})
	}
}

func TestGoalProgressTodayAndEarlierAccepted(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T04:30:00Z") // 2026-07-15 local
	goal := newDateGoal(t, h, 20)
	for _, date := range []string{"2026-07-15", "2026-07-10"} {
		if code, _ := postGoalProgress(t, h, goal.ID, date, "3"); code != http.StatusSeeOther {
			t.Errorf("entry_date %s: status %d; want 303", date, code)
		}
	}
	entries, _ := h.Service.Repo.GoalProgressEntries(goal.ID)
	seen := map[string]bool{}
	for _, e := range entries {
		seen[e.EntryDate] = true
	}
	if !seen["2026-07-15"] || !seen["2026-07-10"] {
		t.Fatalf("accepted entries not stored: %v", seen)
	}
}

func TestGoalProgressLocalMidnightBoundary(t *testing.T) {
	cases := []struct {
		name, instant, date string
		want                int
	}{
		{"CDT before local midnight", "2026-07-16T04:59:00Z", "2026-07-16", http.StatusBadRequest},
		{"CDT at local midnight", "2026-07-16T05:00:00Z", "2026-07-16", http.StatusSeeOther},
		{"CST before local midnight", "2026-01-16T05:59:00Z", "2026-01-16", http.StatusBadRequest},
		{"CST at local midnight", "2026-01-16T06:00:00Z", "2026-01-16", http.StatusSeeOther},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := testHandler(t)
			pinClock(t, h, c.instant)
			goal := newDateGoal(t, h, 20)
			if code, _ := postGoalProgress(t, h, goal.ID, c.date, "4"); code != c.want {
				t.Fatalf("status %d; want %d", code, c.want)
			}
		})
	}
}

func TestGoalProgressEmptyDateDefaultsToLocalToday(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T04:30:00Z") // 2026-07-15 local
	goal := newDateGoal(t, h, 20)
	if code, _ := postGoalProgress(t, h, goal.ID, "", "4"); code != http.StatusSeeOther {
		t.Fatalf("status %d; want 303", code)
	}
	entries, _ := h.Service.Repo.GoalProgressEntries(goal.ID)
	found := false
	for _, e := range entries {
		if e.EntryDate == "2026-07-15" && e.Value == 4 {
			found = true
		}
	}
	if !found {
		t.Fatalf("empty date was not stored as local today: %+v", entries)
	}
}

// Night-shift repro: late in the evening a mistyped year used to freeze the
// goal; now the future entry is refused and the real one completes it.
func TestGoalProgressNightShiftRepro(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-10-13T03:30:00Z") // 2026-10-12 local, evening
	goal := newDateGoal(t, h, 20)
	if code, _ := postGoalProgress(t, h, goal.ID, "2027-10-12", "7"); code != http.StatusBadRequest {
		t.Fatalf("future entry status %d; want 400", code)
	}
	if code, _ := postGoalProgress(t, h, goal.ID, "2026-10-12", "20"); code != http.StatusSeeOther {
		t.Fatalf("today entry status %d; want 303", code)
	}
	got, _ := h.Service.Repo.GoalByID(goal.ID)
	if got.CurrentValue != 20 || got.Status != "completed" {
		t.Fatalf("goal current=%v status=%q; want 20 completed", got.CurrentValue, got.Status)
	}
}
