package handlers

import (
	"errors"
	"lifeops-tracker/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTodayHealthSnapshotStates(t *testing.T) {
	now := time.Date(2026, 9, 11, 14, 0, 0, 0, time.UTC)
	zero := float64(0)
	watch, phone := int64(0), int64(999)
	for _, tt := range []struct {
		name     string
		snapshot *models.DailyHealthSnapshot
		err      error
		state    string
	}{
		{"absent", nil, nil, "empty"},
		{"observed zero", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: &zero, StepsWatch: &watch, StepsPhone: &phone}, LastSyncedAt: now.Format(time.RFC3339)}, nil, "normal"},
		{"partial", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{StepsPhone: &phone}, LastSyncedAt: now.Format(time.RFC3339)}, nil, "partial"},
		{"stale", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: &zero, StepsWatch: &watch}, LastSyncedAt: now.Add(-9 * time.Hour).Format(time.RFC3339)}, nil, "stale"},
		{"unknown freshness", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{StepsWatch: &watch}}, nil, "partial"},
		{"error", nil, errors.New("fixture read error"), "error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			view := buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-11", Snapshot: tt.snapshot}, now)
			if tt.snapshot != nil && tt.snapshot.DietaryCalories != nil {
				view.FoodRemaining = HealthMetricView{Value: "2,000", Unit: "kcal", Present: true}
			}
			got := buildTodayHealthSnapshot(view, tt.err)
			if got.Status.Kind() != tt.state {
				t.Fatalf("state %s, want %s", got.Status.Kind(), tt.state)
			}
			body := renderBody(t, "today.html", TodayData{Page: "today", HealthSnapshot: got})
			if tt.err != nil {
				if len(got.Metrics) != 0 || !strings.Contains(body, "Health snapshot unavailable") {
					t.Fatal("error exposed values or disappeared")
				}
				return
			}
			if tt.snapshot == nil {
				for _, m := range got.Metrics {
					if m.ShowValue() {
						t.Fatal("absent observation became a value")
					}
				}
			}
			if tt.name == "observed zero" {
				if got.Metrics[1].Value != "0" || !got.Metrics[1].ShowValue() || got.Metrics[1].Label != "Watch steps" {
					t.Fatal("zero/watch precedence lost")
				}
				if strings.Contains(body, `<p class="gyst-number">999`) {
					t.Fatal("phone value leaked into Watch steps")
				}
			}
			if tt.name == "partial" && (got.Metrics[1].Label != "Phone steps (fallback)" || got.Metrics[1].Value != "999") {
				t.Fatal("phone fallback not explicit")
			}
		})
	}
}

func TestNextTodayReminder(t *testing.T) {
	now := time.Date(2026, 9, 11, 14, 0, 0, 0, time.FixedZone("local", -5*3600))
	rows := []models.ReminderView{
		{Title: "past", ReminderDate: "2026-09-11", ReminderTime: "09:00"},
		{Title: "later", ReminderDate: "2026-09-11", ReminderTime: "18:00"},
		{Title: "next", ReminderDate: "2026-09-11", ReminderTime: "15:00"},
		{Title: "tomorrow", ReminderDate: "2026-09-12", ReminderTime: "14:01"},
		{Title: "untimed", ReminderDate: "2026-09-11"},
	}
	if got := nextTodayReminder(rows, now); got == nil || got.Title != "next" {
		t.Fatalf("next = %#v", got)
	}
	if nextTodayReminder(rows, now.Add(5*time.Hour)) != nil {
		t.Fatal("past reminder presented as upcoming")
	}
}

func TestTodayPreservesActionsAndRealFocus(t *testing.T) {
	body := renderBody(t, "today.html", TodayData{Page: "today", Today: "2026-09-11", Pending: []models.TaskOccurrence{{Task: models.Task{ID: 81, Title: "Actual pending task"}, Date: "2026-09-11"}}, Skipped: []models.TaskOccurrence{{Task: models.Task{ID: 82, Title: "Skipped fixture"}, Date: "2026-09-11"}}, Learning: models.TodayLearning{Blocks: []models.LearningBlock{{ID: 31, CourseID: 11, Title: "Real lesson"}}}})
	for _, text := range []string{`action="/tasks/81/complete"`, `action="/tasks/81/skip"`, `action="/tasks/82/complete"`, `name="occurrence_date" value="2026-09-11"`, `value="/today#task-81"`, `action="/courses/11/blocks/31/complete"`, `action="/courses/11/blocks/31/postpone"`, `data-prayer-enable`, `data-prayer-test`, `data-prayer-audio`, `aria-label="Focus item"`, `href="/tasks/81"`, `Health snapshot`, `Today’s checklist`, `No upcoming timed reminders today.`} {
		if !strings.Contains(body, text) {
			t.Errorf("lost %s", text)
		}
	}
	if strings.Count(body, "Actual pending task") < 2 {
		t.Fatal("focus does not use actual pending task")
	}
}

func TestDailyChecklistRoute(t *testing.T) {
	h := testHandler(t)
	for _, path := range []string{"/checklist", "/checklist/"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Daily checklist") {
			t.Fatalf("checklist %s: %d", path, rec.Code)
		}
		if !strings.Contains(primaryMarkup(t, rec.Body.String()), `href="/today" aria-current="page"`) {
			t.Fatal("checklist lost Today ownership")
		}
	}
	for _, tt := range []struct {
		method, path string
		status       int
	}{{"POST", "/checklist", 405}, {"GET", "/checklist/nope", 404}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
		if rec.Code != tt.status {
			t.Fatalf("%s %s: %d", tt.method, tt.path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, httptest.NewRequest("GET", "/checklist", nil))
	if rec.Code != 303 {
		t.Fatal("checklist bypassed auth")
	}
	body := renderBody(t, "checklist.html", TodayData{Page: "today", ChecklistOnly: true, Pending: []models.TaskOccurrence{{Task: models.Task{ID: 81, Title: "Actual pending task"}, Date: "2026-09-11"}}})
	if !strings.Contains(body, `value="/checklist#task-81"`) {
		t.Fatal("checklist completion lost return destination")
	}
}

func TestTodayChecklistPreviewKeepsFullDestination(t *testing.T) {
	items := make([]models.TaskOccurrence, 6)
	data := TodayData{Page: "today", Pending: items}
	body := renderBody(t, "today.html", data)
	if strings.Count(body, `class="task"`) != 4 || !strings.Contains(body, `href="/checklist"`) {
		t.Fatal("Today preview is not bounded or lacks full destination")
	}
	data.ChecklistOnly = true
	body = renderBody(t, "checklist.html", data)
	if strings.Count(body, `class="task"`) != 6 {
		t.Fatal("dedicated checklist dropped tasks")
	}
}

func TestTodayChecklistHeaderCountsSkippedOnce(t *testing.T) {
	body := renderBody(t, "today.html", TodayData{Page: "today", Today: "2026-09-11",
		Pending: []models.TaskOccurrence{{Task: models.Task{ID: 81, Title: "Pending fixture"}, Date: "2026-09-11"}},
		Skipped: []models.TaskOccurrence{{Task: models.Task{ID: 82, Title: "Skipped fixture"}, Date: "2026-09-11"}},
	})
	if !strings.Contains(body, "0 done · 1 left · 1 skipped") || !strings.Contains(body, `class="gyst-progress"`) {
		t.Error("header caption or progress bar missing")
	}
	if strings.Contains(body, "% complete") {
		t.Error("duplicate percent caption is back")
	}
}
