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
	for _, text := range []string{`action="/tasks/81/complete"`, `action="/tasks/81/skip"`, `action="/tasks/82/complete"`, `name="occurrence_date" value="2026-09-11"`, `value="/today#task-81"`, `action="/courses/11/blocks/31/complete"`, `action="/courses/11/blocks/31/postpone"`, `data-prayer-enable`, `data-prayer-test`, `data-prayer-audio`, `Health snapshot`, `Today’s checklist`} {
		if !strings.Contains(body, text) {
			t.Errorf("lost %s", text)
		}
	}
	// GYST-077 (RULES 2.2, deliberate): the Focus and Next up cards are gone, so
	// the pending task is listed exactly once.
	if strings.Count(body, `href="/tasks/81"`) != 1 || strings.Count(body, `action="/tasks/81/complete"`) != 1 {
		t.Fatal("pending task is not listed exactly once")
	}
}

func TestTodayListsEachItemOnce(t *testing.T) {
	d := "2026-09-11"
	pending := []models.TaskOccurrence{
		{Task: models.Task{ID: 91, Title: "Fictional alpha", ReminderTime: "09:00"}, Date: d},
		{Task: models.Task{ID: 92, Title: "Fictional bravo"}, Date: d},
		{Task: models.Task{ID: 93, Title: "Fictional charlie"}, Date: d},
	}
	data := TodayData{Page: "today", Today: d, Pending: pending,
		Completed: []models.TaskOccurrence{{Task: models.Task{ID: 94, Title: "Fictional done"}, Date: d, CompletedAt: d + " 08:00:00"}},
		Skipped:   []models.TaskOccurrence{{Task: models.Task{ID: 95, Title: "Fictional skipped"}, Date: d}},
		Learning: models.TodayLearning{Blocks: []models.LearningBlock{
			{ID: 31, CourseID: 11, Title: "Fictional lesson one", CourseName: "Course A", BlockType: "lesson"},
			{ID: 32, CourseID: 11, Title: "Fictional lesson two", CourseName: "Course A", BlockType: "lesson"},
		}},
		RemindersToday: standaloneReminders([]models.ReminderView{
			{SourceType: "task", SourceID: 91, Title: "Fictional alpha", ReminderDate: d, ReminderTime: "09:00"},
			{SourceType: "reminder", SourceID: 7, Title: "Fictional standalone reminder", ReminderDate: d, ReminderTime: "10:30"},
		}),
	}
	body := renderBody(t, "today.html", data)
	open, rest, ok := strings.Cut(body, `<details class="today-done"`)
	if !ok {
		t.Fatal("no Done disclosure")
	}
	if n := strings.Count(open, `class="task"`); n != 5 {
		t.Fatalf("open rows before Done = %d, want 5", n)
	}
	for _, link := range []string{`href="/tasks/91"`, `href="/tasks/92"`, `href="/tasks/93"`, `href="/courses/11#block-31"`, `href="/courses/11#block-32"`, `href="/tasks/94"`, `href="/tasks/95"`} {
		if c := strings.Count(body, link); c != 1 {
			t.Errorf("%s appears %d times, want 1", link, c)
		}
	}
	details, _, _ := strings.Cut(rest, `</details>`)
	if !strings.Contains(details, "Done · 1 · 1 skipped") || !strings.Contains(details, "Fictional done") || !strings.Contains(details, "Fictional skipped") {
		t.Errorf("Done disclosure wrong: %s", details)
	}
	if strings.Contains(body, `<details class="today-done" open`) {
		t.Error("Done is open by default on Today")
	}
	if strings.Contains(body, `<span class="grow">Fictional alpha`) || strings.Count(body, `<span class="grow">Fictional standalone reminder`) != 1 {
		t.Error("task reminder duplicated or standalone reminder missing")
	}
	if !strings.Contains(body, "09:00") {
		t.Error("task reminder time missing from row meta")
	}
	if strings.Contains(body, `aria-label="Study blocks"`) || strings.Contains(body, "today-context") {
		t.Error("removed cards are back")
	}
	// Zero-value data still renders.
	renderOK(t, "today.html", TodayData{})
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
	// GYST-077 (RULES 2.2, deliberate): the 4-row preview cap is gone, so
	// Today lists all 6 rows and still links to the dedicated checklist.
	items := make([]models.TaskOccurrence, 6)
	data := TodayData{Page: "today", Pending: items}
	body := renderBody(t, "today.html", data)
	if strings.Count(body, `class="task"`) != 6 || !strings.Contains(body, `href="/checklist"`) {
		t.Fatal("Today does not list every row or lacks full destination")
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
