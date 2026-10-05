package handlers

import (
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

// Controls and notices for Health features that do not exist (05A D8,
// GYST-VN-052). None of them may render on a Health surface.
var unbuiltHealthMarkup = []string{
	`data-period=`, `class="health-period"`, "Summary period",
	"Week, month and year summaries are not built yet",
	"Planned tools", "Next meal", "Plan meal", "Start workout",
	"Strength trend", "No strength records connected",
	"Nutrition &amp; activity trends", "Historical summaries and plan adherence are not available yet",
	"Weight entry and weight import setup are not available",
}

func assertNoUnbuiltHealthMarkup(t *testing.T, body string) {
	t.Helper()
	for _, banned := range unbuiltHealthMarkup {
		if strings.Contains(body, banned) {
			t.Errorf("unbuilt Health placeholder rendered: %q", banned)
		}
	}
}

func TestHealthSummaryHidesUnbuiltControls(t *testing.T) {
	cycle := viewCycle()
	for _, tc := range []struct {
		name     string
		response models.HealthTodayResponse
		want     []string
		absent   []string
	}{
		{
			name:     "no data",
			response: models.HealthTodayResponse{Date: "2026-09-10"},
			want:     []string{"Waiting for today&#39;s Apple Health sync", "nothing synced today. Run GYST Sync All on your iPhone.", "No health plan yet", `<span class="hv3-absent">—</span>`, "No steps received yet"},
			absent:   []string{"which is not built yet"},
		},
		{
			// A recorded zero stays a zero; a missing phone count stays missing.
			name: "observed zero watch steps, phone missing",
			response: models.HealthTodayResponse{Date: "2026-09-10", HealthCycle: &cycle, Snapshot: &models.DailyHealthSnapshot{
				HealthMetrics: models.HealthMetrics{StepsWatch: viewInt(0), DietaryCalories: viewFloat(0)},
			}},
			want:   []string{`<p class="hv3-v">0 <small>Apple Watch</small></p>`, "iPhone · not received"},
			absent: []string{"nothing synced today"},
		},
		{
			// No Watch count: the iPhone count is shown large and labelled,
			// never a large dash, and never added to anything.
			name: "phone steps only",
			response: models.HealthTodayResponse{Date: "2026-09-10", HealthCycle: func() *models.HealthCycle { c := viewCycle(); c.StepTarget = 10000; return &c }(), Snapshot: &models.DailyHealthSnapshot{
				HealthMetrics: models.HealthMetrics{StepsPhone: viewInt(5834)},
			}},
			want:   []string{`<p class="hv3-v">5,834 <small>iPhone</small></p>`, "Apple Watch · not received", `of 10,000 steps, iPhone`},
			absent: []string{"No steps received yet"},
		},
		{
			name: "full day",
			response: models.HealthTodayResponse{Date: "2026-09-10", HealthCycle: &cycle, Snapshot: &models.DailyHealthSnapshot{
				HealthMetrics: models.HealthMetrics{
					DietaryCalories: viewFloat(1800), ProteinG: viewFloat(120), CarbsG: viewFloat(200), FatG: viewFloat(60), FiberG: viewFloat(25),
					StepsWatch: viewInt(9000), StepsPhone: viewInt(4000), ActiveEnergyKcal: viewFloat(500), RestingEnergyKcal: viewFloat(1700), ExerciseMinutes: viewFloat(30),
				},
			}, Observed: models.ObservedHealthEnergy{ObservedTotalBurn: viewFloat(2200)}},
			// Steps are shown per source and never summed (9,000 + 4,000).
			want:   []string{"<dd>2,200</dd>", "1,700 resting + 500 active", "9,000 <small>Apple Watch</small>", "iPhone 4,000 · counted separately"},
			absent: []string{"13,000", "nothing synced today"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := buildHealthTodayView(tc.response, viewNow())
			body := renderBody(t, "health.html", HealthData{Page: "health", Today: view})
			assertNoUnbuiltHealthMarkup(t, body)
			for _, want := range append([]string{"Sep 10, 2026", "Burned so far", "Balance so far", "Macros", "Activity", "Steps", "Weight", `href="/health/progress#log"`, `href="/health/nutrition"`}, tc.want...) {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			for _, banned := range tc.absent {
				if strings.Contains(body, banned) {
					t.Errorf("unexpected %q", banned)
				}
			}
		})
	}
}

func TestHealthProgressHidesUnbuiltTrends(t *testing.T) {
	location, _ := time.LoadLocation("America/Winnipeg")
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, location)
	one := []models.HealthWeightEntry{{ID: 1, MeasuredAt: "2026-09-12T08:00:00-05:00", WeightKg: 81.5, Source: "manual"}}
	many := append(one, models.HealthWeightEntry{ID: 2, MeasuredAt: "2026-09-10T08:00:00-05:00", WeightKg: 82, Source: "apple_shortcuts"})
	for _, tc := range []struct {
		name    string
		entries []models.HealthWeightEntry
		want    []string
	}{
		{"empty", nil, []string{"No weight readings in this range", "Missing readings are not shown as zero", `href="?range=all"`}},
		{"one", one, []string{"81.5 kg", "Recorded measurements"}},
		{"many", many, []string{"81.5", "82", "Apple Shortcuts", `role="img"`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := renderBody(t, "health_progress.html", buildHealthProgress(tc.entries, "30", now, location))
			assertNoUnbuiltHealthMarkup(t, body)
			for _, want := range append([]string{"Weight history", "30 days", "90 days", "All time"}, tc.want...) {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
		})
	}
}

// The other Health pages never carried these placeholders; keep it that way.
func TestHealthSubpagesHaveNoUnbuiltControls(t *testing.T) {
	view := buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10"}, viewNow())
	assertNoUnbuiltHealthMarkup(t, renderBody(t, "nutrition.html", nutritionData{Page: "health", Today: view}))
	assertNoUnbuiltHealthMarkup(t, renderBody(t, "training.html", trainingData{HealthData: HealthData{Page: "health"}}))
}
