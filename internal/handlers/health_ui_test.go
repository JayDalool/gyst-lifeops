package handlers

import (
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func TestHealthIntakeGraphicRequiresObservedIntakeAndTarget(t *testing.T) {
	cycle := viewCycle()
	for _, tc := range []struct {
		name   string
		intake *float64
		cycle  *models.HealthCycle
		bar    bool
	}{
		{"missing intake", nil, &cycle, false},
		{"missing target", viewFloat(1200), nil, false},
		{"observed zero", viewFloat(0), &cycle, true},
		{"over target", viewFloat(3000), &cycle, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-12", HealthCycle: tc.cycle, Snapshot: &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: tc.intake}}}, viewNow())
			body := renderBody(t, "health.html", HealthData{Page: "health", Today: view})
			// Health v3 (#53): the ring's value arc appears only when both
			// intake and target were observed; otherwise the track is dashed.
			if strings.Contains(body, `class="val" cx="60"`) != tc.bar {
				t.Fatal("calorie ring misrepresents availability")
			}
			if tc.name == "over target" && (!strings.Contains(body, "3,000") || !strings.Contains(body, "kcal over budget")) {
				t.Fatal("actual over-target intake was hidden")
			}
			for _, want := range []string{`Burned so far`, `Balance so far`, `Steps`, `Phone`, `href="/health/progress#log"`, `href="/health/nutrition"`} {
				if !strings.Contains(body, want) {
					t.Errorf("missing %s", want)
				}
			}
		})
	}
}
