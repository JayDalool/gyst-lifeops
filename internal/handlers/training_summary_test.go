package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

func TestThousandsAndWeekTime(t *testing.T) {
	for in, want := range map[float64]string{0: "0", 999: "999", 1000: "1,000", 14820: "14,820", 1234567: "1,234,567", -2500: "-2,500"} {
		if got := thousands(in); got != want {
			t.Errorf("thousands(%v) = %q, want %q", in, got, want)
		}
	}
	for _, tc := range []struct {
		week services.TrainingWeek
		want string
	}{
		{services.TrainingWeek{}, "0 min"},
		{services.TrainingWeek{Workouts: 1, UnknownTime: 1}, ""},
		{services.TrainingWeek{Workouts: 2, Minutes: 92}, "1 h 32 min"},
		{services.TrainingWeek{Workouts: 1, Minutes: 45}, "45 min"},
	} {
		if got := trainingWeekViewOf(tc.week, 3).Time; got != tc.want {
			t.Errorf("%+v: time %q, want %q", tc.week, got, tc.want)
		}
	}
	if pts := sparkPoints([]float64{5, 5}); pts != "0.0,16.0 100.0,16.0" {
		t.Fatalf("flat series %q", pts)
	}
}

func TestTrainingPageShowsSummary(t *testing.T) {
	h := testHandler(t)
	loc := h.Service.Location()
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, loc) // a Wednesday, so nothing crosses a week boundary
	h.Service.Clock.Now = func() time.Time { return now }
	f := func(v float64) *float64 { return &v }
	lift := func(kg float64) models.TrainingExercise {
		return models.TrainingExercise{Name: "Leg Press", PrimaryMuscle: "quadriceps", Sets: []models.TrainingSet{{Kind: "normal", LoadKG: f(kg), Reps: f(10)}, {Kind: "warmup", LoadKG: f(500), Reps: f(10)}}}
	}
	recent := now.Add(-time.Minute)
	entries := []models.TrainingEntry{
		{ID: "a", Kind: "workout", Title: "Full Body A", StartedAt: now.AddDate(0, 0, -30), Exercises: []models.TrainingExercise{lift(100)}},
		{ID: "b", Kind: "workout", Title: "Full Body A", StartedAt: recent.Add(-45 * time.Minute), EndedAt: recent, Exercises: []models.TrainingExercise{lift(118)}},
	}
	if err := h.Service.Repo.SaveTraining("hevy", entries, now); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/training", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"This week", "<strong>1</strong> workouts", "1,180 kg", "45 min", "Top progress · 12 weeks", "Leg Press", `hv3-prog-pct">&#43;18%`, "Best 118 kg × 10", "Muscle focus", "Legs", "100%", "workout logged", "Latest workout", "45 min · 1 exercise"} {
		if !strings.Contains(body, want) {
			t.Errorf("training page missing %q", want)
		}
	}
	if strings.Contains(body, "5,000 kg") || strings.Contains(body, "6,180 kg") {
		t.Fatal("warmup counted in volume")
	}
}

// The weekly target is read from the active health plan, never fixed (#54).
func TestWorkoutTargetFollowsHealthPlan(t *testing.T) {
	for _, target := range []int64{2, 3} {
		h := testHandler(t)
		now := time.Date(2026, 10, 7, 12, 0, 0, 0, h.Service.Location())
		h.Service.Clock.Now = func() time.Time { return now }
		cycle := models.HealthCycle{Title: "Fat loss", PrimaryGoal: "Lose weight", StartDate: "2026-09-22", EndDate: "2027-04-04", StrengthSessionsPerWeek: target}
		if err := h.Service.CreateHealthCycle(&cycle); err != nil {
			t.Fatal(err)
		}
		if err := h.Service.Repo.SaveTraining("hevy", []models.TrainingEntry{{ID: "r", Kind: "routine", Title: "A"}}, now); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health/training", nil))
		body := w.Body.String()
		want := fmt.Sprintf("<strong>0</strong> / %d workouts", target)
		if !strings.Contains(body, want) || !strings.Contains(body, fmt.Sprintf("Target %d a week", target)) {
			t.Fatalf("target %d not shown from the plan", target)
		}
	}
}
