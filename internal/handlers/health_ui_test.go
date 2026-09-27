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
			if strings.Contains(body, `class="health-intake-progress"`) != tc.bar {
				t.Fatal("intake graphic misrepresents availability")
			}
			if strings.Contains(body, `class="health-ring"`) {
				t.Fatal("decorative ring still implies measured progress")
			}
			if tc.name == "over target" && !strings.Contains(body, "3,000") {
				t.Fatal("actual over-target intake was hidden")
			}
			for _, want := range []string{`health-observed`, `food calories remaining`, `Observed energy balance`, `Watch steps`, `Phone steps`, `/health/plan`, `/goals/new`} {
				if !strings.Contains(body, want) {
					t.Errorf("missing %s", want)
				}
			}
		})
	}
}
