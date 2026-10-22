package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// GYST-152: a blank or garbled progress value must never be saved as 0.

func newProgressGoal(t *testing.T, h *Handler, target, start float64) models.Goal {
	t.Helper()
	pinClock(t, h, "2026-10-12T18:00:00Z")
	goal := models.Goal{Title: "Read books", Type: "yearly", TargetValue: target, Unit: "books", StartDate: "2026-01-01", EndDate: "2026-12-31"}
	if err := h.Service.Repo.CreateGoal(&goal, h.Service.TodayString()); err != nil {
		t.Fatal(err)
	}
	if start > 0 {
		if err := h.Service.Repo.AddGoalProgress(goal.ID, "2026-10-01", start, ""); err != nil {
			t.Fatal(err)
		}
	}
	return goal
}

func goalState(t *testing.T, h *Handler, id int64) (models.Goal, int) {
	t.Helper()
	goal, err := h.Service.Repo.GoalByID(id)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := h.Service.Repo.GoalProgressEntries(id)
	if err != nil {
		t.Fatal(err)
	}
	return goal, len(entries)
}

func TestBadGoalProgressValuesAreRejectedWithoutWriting(t *testing.T) {
	cases := map[string]string{
		"blank": "", "spaces": "   ", "text": "abc", "NaN": "NaN", "nan": "nan",
		"Inf": "Inf", "-Inf": "-Inf", "+Infinity": "+Infinity", "out of range": "1e400", "negative": "-1",
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			h := testHandler(t)
			goal := newProgressGoal(t, h, 120, 18)
			rec := postForm(t, h, fmt.Sprintf("/goals/%d/progress", goal.ID), url.Values{"value": {bad}, "entry_date": {"2026-10-12"}})
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d; want 400", rec.Code)
			}
			body := rec.Body.String()
			if strings.TrimSpace(bad) != "" && strings.Contains(body, bad) {
				t.Errorf("body echoes the posted value: %q", body)
			}
			if strings.Contains(strings.ToLower(body), "sql") {
				t.Errorf("body leaks internals: %q", body)
			}
			got, entries := goalState(t, h, goal.ID)
			if got.CurrentValue != 18 || got.Status != "active" || entries != 1 {
				t.Fatalf("goal changed: current=%v status=%q entries=%d", got.CurrentValue, got.Status, entries)
			}
		})
	}
}

func TestBlankProgressDoesNotReopenCompletedGoal(t *testing.T) {
	h := testHandler(t)
	goal := newProgressGoal(t, h, 20, 20)
	before, n := goalState(t, h, goal.ID)
	if before.Status != "completed" {
		t.Fatalf("setup: status %q; want completed", before.Status)
	}
	rec := postForm(t, h, fmt.Sprintf("/goals/%d/progress", goal.ID), url.Values{"value": {""}, "entry_date": {"2026-10-12"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d; want 400", rec.Code)
	}
	after, m := goalState(t, h, goal.ID)
	if after.Status != "completed" || after.CurrentValue != 20 || m != n {
		t.Fatalf("completed goal changed: %q %v entries %d->%d", after.Status, after.CurrentValue, n, m)
	}
}

func TestMissingProgressValueKeyIsRejected(t *testing.T) {
	h := testHandler(t)
	goal := newProgressGoal(t, h, 120, 18)
	rec := postForm(t, h, fmt.Sprintf("/goals/%d/progress", goal.ID), url.Values{"entry_date": {"2026-10-12"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d; want 400", rec.Code)
	}
	if got, n := goalState(t, h, goal.ID); got.CurrentValue != 18 || n != 1 {
		t.Fatalf("goal changed: %v entries %d", got.CurrentValue, n)
	}
}

func TestValidGoalProgressValuesAreStored(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want float64
	}{{"0", 0}, {"12.5", 12.5}, {" 7 ", 7}} {
		t.Run(tc.raw, func(t *testing.T) {
			h := testHandler(t)
			goal := newProgressGoal(t, h, 120, 18)
			rec := postForm(t, h, fmt.Sprintf("/goals/%d/progress", goal.ID), url.Values{"value": {tc.raw}, "entry_date": {"2026-10-12"}})
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status %d; want 303", rec.Code)
			}
			got, n := goalState(t, h, goal.ID)
			if got.CurrentValue != tc.want || n != 2 {
				t.Fatalf("current=%v entries=%d; want %v and 2", got.CurrentValue, n, tc.want)
			}
		})
	}
}

func TestBadValueIsCheckedBeforeDate(t *testing.T) {
	h := testHandler(t)
	goal := newProgressGoal(t, h, 120, 0)
	rec := postForm(t, h, fmt.Sprintf("/goals/%d/progress", goal.ID), url.Values{"value": {""}, "entry_date": {"x"}})
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Enter a progress value.") {
		t.Fatalf("status %d body %q; want the value message", rec.Code, rec.Body.String())
	}
	if _, n := goalState(t, h, goal.ID); n != 0 {
		t.Fatalf("entries = %d; want 0", n)
	}
}

func TestParseGoalProgressValue(t *testing.T) {
	const blank, notNumber, negative = "Enter a progress value.", "Progress value must be a number, for example 12.5.", "Progress value must be 0 or more."
	for raw, msg := range map[string]string{
		"": blank, "  ": blank, "abc": notNumber, "NaN": notNumber, "Inf": notNumber,
		"-Infinity": notNumber, "1e400": notNumber, "-0.5": negative,
	} {
		_, err := parseGoalProgressValue(raw)
		got, ok := services.UserMessage(err)
		if !ok || got != msg {
			t.Errorf("parse %q: message %q ok=%v; want %q", raw, got, ok, msg)
		}
	}
	if v, err := parseGoalProgressValue(" 0 "); err != nil || v != 0 {
		t.Errorf("parse 0: %v, %v", v, err)
	}
}

func TestGoalDetailProgressInputIsRequired(t *testing.T) {
	h := testHandler(t)
	goal := newProgressGoal(t, h, 120, 18)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/goals/%d", goal.ID), nil))
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `inputmode="decimal"`) || !strings.Contains(body, `name="value" required`) {
		t.Fatalf("status %d; the Value input needs inputmode=decimal and required", rec.Code)
	}
}
