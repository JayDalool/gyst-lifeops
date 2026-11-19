package handlers

import (
	"net/http"
	"net/url"
	"testing"
)

// GYST-138: POST /goals/new validates start_date and end_date on the server.

func goalCreateValues(start, end string) url.Values {
	return url.Values{
		"title": {"Read"}, "type": {"yearly"}, "target_value": {"10"},
		"current_value": {"3"}, "start_date": {start}, "end_date": {end},
	}
}

func TestNewGoalRejectsBadDatesWithoutWriting(t *testing.T) {
	for _, b := range malformedDates {
		for name, v := range map[string]url.Values{
			"start": goalCreateValues(b, ""),
			"end":   goalCreateValues("", b),
		} {
			t.Run(name+" "+b, func(t *testing.T) {
				h := testHandler(t)
				pinClock(t, h, "2026-07-16T04:30:00Z")
				if rec := postForm(t, h, "/goals/new", v); rec.Code != http.StatusBadRequest {
					t.Fatalf("status %d; want 400", rec.Code)
				}
				if goals, _ := h.Service.Repo.ListGoals(true); len(goals) != 0 {
					t.Fatalf("%d goals written", len(goals))
				}
			})
		}
	}
}

func TestNewGoalRejectsEndBeforeStart(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T04:30:00Z")
	if rec := postForm(t, h, "/goals/new", goalCreateValues("2026-08-01", "2026-07-01")); rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d; want 400", rec.Code)
	}
	if goals, _ := h.Service.Repo.ListGoals(true); len(goals) != 0 {
		t.Fatalf("%d goals written", len(goals))
	}
}

func TestNewGoalFutureStartAcceptedInitialEntryToday(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T04:30:00Z") // 2026-07-15 local
	if rec := postForm(t, h, "/goals/new", goalCreateValues("2026-09-01", "2026-12-31")); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	goals, _ := h.Service.Repo.ListGoals(true)
	if len(goals) != 1 || goals[0].Goal.StartDate != "2026-09-01" || goals[0].Goal.EndDate != "2026-12-31" {
		t.Fatalf("goals = %#v", goals)
	}
	entries, _ := h.Service.Repo.GoalProgressEntries(goals[0].Goal.ID)
	if len(entries) != 1 || entries[0].EntryDate != "2026-07-15" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestNewGoalEmptyDatesStillAccepted(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T04:30:00Z")
	if rec := postForm(t, h, "/goals/new", goalCreateValues("", "")); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	goals, _ := h.Service.Repo.ListGoals(true)
	if len(goals) != 1 || goals[0].Goal.StartDate != "" || goals[0].Goal.EndDate != "" {
		t.Fatalf("goals = %#v", goals)
	}
	entries, _ := h.Service.Repo.GoalProgressEntries(goals[0].Goal.ID)
	if len(entries) != 1 || entries[0].EntryDate != "2026-07-15" {
		t.Fatalf("entries = %#v", entries)
	}
}
