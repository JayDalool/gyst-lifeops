package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func TestBuildHealthWeight(t *testing.T) {
	loc, _ := time.LoadLocation("America/Winnipeg")
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, loc)
	at := func(month time.Month, day int) string {
		return time.Date(2026, month, day, 7, 0, 0, 0, loc).Format(time.RFC3339)
	}
	entry := func(month time.Month, day int, kg float64) models.HealthWeightEntry {
		return models.HealthWeightEntry{MeasuredAt: at(month, day), WeightKg: kg}
	}
	if v := buildHealthWeight(nil, "2026-09-22", now, loc); v.Present {
		t.Fatal("no readings must not be present")
	}
	// Change since plan start uses the first reading on or after the start.
	v := buildHealthWeight([]models.HealthWeightEntry{entry(9, 10, 140), entry(9, 23, 136.2), entry(10, 1, 135), entry(10, 5, 134.5)}, "2026-09-22", now, loc)
	if !v.Present || v.Latest != "134.5" || v.Change != "−1.7 kg" || v.Since != "Sep 22" || !v.Down || v.LatestAt != "Today" || v.Points == "" {
		t.Fatalf("since plan start %+v", v)
	}
	// Readings only before the plan start: say so instead of "First reading".
	v = buildHealthWeight([]models.HealthWeightEntry{entry(9, 20, 136), entry(9, 28, 135)}, "2026-10-01", now, loc)
	if !v.NoneSincePlan || v.HasChange || v.Since != "Oct 1" || v.Points != "" {
		t.Fatalf("none since plan %+v", v)
	}
	// One reading, gain shown with a plus sign; future readings ignored.
	v = buildHealthWeight([]models.HealthWeightEntry{entry(10, 4, 134.5), {MeasuredAt: now.Add(time.Hour).Format(time.RFC3339), WeightKg: 99}}, "", now, loc)
	if v.Latest != "134.5" || v.Readings != 1 || v.HasChange || v.LatestAt != "Yesterday" {
		t.Fatalf("single %+v", v)
	}
	v = buildHealthWeight([]models.HealthWeightEntry{entry(10, 1, 134), entry(10, 4, 134.4)}, "", now, loc)
	if v.Change != "+0.4 kg" || v.Down {
		t.Fatalf("gain %+v", v)
	}
}

func TestDayLabel(t *testing.T) {
	loc, _ := time.LoadLocation("America/Winnipeg")
	// The morning after DST ends: a 25-hour day must still be "Yesterday".
	now := time.Date(2026, 11, 2, 8, 0, 0, 0, loc)
	for at, want := range map[time.Time]string{
		time.Date(2026, 11, 2, 0, 5, 0, 0, loc):  "Today",
		time.Date(2026, 11, 1, 0, 5, 0, 0, loc):  "Yesterday",
		time.Date(2026, 10, 20, 9, 0, 0, 0, loc): "Oct 20",
		time.Date(2025, 5, 6, 19, 0, 0, 0, loc):  "May 6, 2025",
	} {
		if got := dayLabel(at, now, loc); got != want {
			t.Errorf("dayLabel(%v) = %q, want %q", at, got, want)
		}
	}
}

func TestTinyOvershootIsNotOver(t *testing.T) {
	cycle := viewCycle()
	cycle.DailyCalorieTarget = 2000
	snap := models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(2000.3)}}
	v := buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snap, HealthCycle: &cycle}, viewNow())
	if v.CaloriesOver || v.BudgetAbs != "0" || v.Calories.Remaining.Value != "0" || v.CalState.Key == "over" {
		t.Fatalf("2000.3 of 2000 reads as over: %+v", v.Calories)
	}
	snap.DietaryCalories = viewFloat(2001)
	v = buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snap, HealthCycle: &cycle}, viewNow())
	if !v.CaloriesOver || v.BudgetAbs != "1" {
		t.Fatalf("2001 of 2000 should be over: %+v", v.Calories)
	}
}

func TestHealthMoreRoute(t *testing.T) {
	h := testHandler(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health/more", nil))
	if r.Code != 200 || !strings.Contains(r.Body.String(), `href="/health/more" aria-current="page"`) || !strings.Contains(r.Body.String(), "Data sources") {
		t.Fatalf("GET: %d", r.Code)
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/health/more", nil))
	if r.Code != http.StatusMethodNotAllowed || r.Header().Get("Allow") != "GET" {
		t.Fatalf("POST: %d %q", r.Code, r.Header().Get("Allow"))
	}
	if authExemptPath("/health/more") {
		t.Fatal("/health/more must require a session")
	}
}

func TestWorkoutProgressViewAndEarlierWorkouts(t *testing.T) {
	h := testHandler(t)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, h.Service.Location())
	h.Service.Clock.Now = func() time.Time { return now }
	f := func(v float64) *float64 { return &v }
	lift := func(kg float64) []models.TrainingExercise {
		return []models.TrainingExercise{{Name: "Leg Press", PrimaryMuscle: "quadriceps", Sets: []models.TrainingSet{{Kind: "normal", LoadKG: f(kg), Reps: f(10)}}}}
	}
	entries := []models.TrainingEntry{
		{ID: "a", Kind: "workout", Title: "Session A", StartedAt: now.AddDate(0, 0, -20), Exercises: lift(100)},
		{ID: "b", Kind: "workout", Title: "Session B", StartedAt: now.AddDate(0, 0, -10), Exercises: lift(110)},
		{ID: "c", Kind: "workout", Title: "Session C", StartedAt: now.AddDate(0, 0, -1), Exercises: lift(120)},
	}
	if err := h.Service.Repo.SaveTraining("hevy", entries, now); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path string
		want []string
	}{
		{"/health/training", []string{"Latest workout", "Session C", "Earlier workouts", "2 more from Hevy", "Session B", "Session A"}},
		{"/health/training?view=progress&period=4", []string{`aria-current="page">4 wk`, "Workouts per week", "3 in 4 weeks", "3 wk ago", "Exercise progress · 4 weeks", "Leg Press", "&#43;20%"}},
		{"/health/training?view=progress&period=99", []string{`aria-current="page">12 wk`, "in 12 weeks", "11 wk ago"}},
	} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if r.Code != 200 {
			t.Fatalf("%s: %d", tc.path, r.Code)
		}
		for _, want := range tc.want {
			if !strings.Contains(r.Body.String(), want) {
				t.Errorf("%s missing %q", tc.path, want)
			}
		}
	}
}
