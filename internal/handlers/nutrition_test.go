package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func TestNutritionRouteAndMethods(t *testing.T) {
	h := testHandler(t)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(method, "/health/nutrition", nil))
		if method == http.MethodGet {
			if r.Code != 200 || !strings.Contains(r.Body.String(), "No nutrition data received") {
				t.Fatalf("GET: %d %s", r.Code, r.Body.String())
			}
		} else if r.Code != 405 {
			t.Fatalf("POST: %d", r.Code)
		}
	}
	if authExemptPath("/health/nutrition") {
		t.Fatal("nutrition must require a session")
	}
}

func TestNutritionObservationStates(t *testing.T) {
	cycle := viewCycle()
	for _, tc := range []struct {
		name         string
		metrics      models.HealthMetrics
		hasNutrition bool
		want         []string
	}{
		{"missing", models.HealthMetrics{}, false, []string{"No nutrition data received", "Not received yet"}},
		{"activity only", models.HealthMetrics{StepsWatch: viewInt(100)}, false, []string{"No nutrition data received"}},
		{"zero", models.HealthMetrics{DietaryCalories: viewFloat(0)}, true, []string{"0</strong>", "2,300 kcal remaining"}},
		{"partial over target", models.HealthMetrics{DietaryCalories: viewFloat(3000), ProteinG: viewFloat(12)}, true, []string{"3,000", "700 kcal over target", "12", "Not received yet", `stroke-dasharray="100 100"`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := models.DailyHealthSnapshot{HealthMetrics: tc.metrics, LastSyncedAt: "2026-09-09T00:00:00Z"}
			data := buildNutritionData(buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snapshot, HealthCycle: &cycle}, viewNow()))
			if data.HasNutrition != tc.hasNutrition {
				t.Fatal("nutrition presence inferred incorrectly")
			}
			body := renderBody(t, "nutrition.html", data)
			for _, want := range append(tc.want, "Data may be out of date", "/health/plan", "not individual meals") {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			if !tc.hasNutrition && strings.Contains(body, `class="gyst-ring-fill"`) {
				t.Fatal("missing intake rendered a filled ring")
			}
		})
	}
	data := buildNutritionData(buildHealthTodayView(models.HealthTodayResponse{Snapshot: &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(123)}}}, viewNow()))
	body := renderBody(t, "nutrition.html", data)
	if !strings.Contains(body, "123") || !strings.Contains(body, "Target not set") || strings.Contains(body, `class="gyst-ring-fill"`) {
		t.Fatal("targetless intake misrepresented")
	}
}
