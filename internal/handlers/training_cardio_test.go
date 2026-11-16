package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func TestTrainingViewFormatsTreadmillSets(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	v := trainingView(models.TrainingEntry{Exercises: []models.TrainingExercise{{Name: "Treadmill", Sets: []models.TrainingSet{
		{Kind: "normal", Seconds: f(1200), InclinePct: f(8), SpeedKmh: f(5)},
		{Kind: "warmup", Seconds: f(90), SpeedKmh: f(4.5)},
		{Kind: "normal", Seconds: f(1200)},
		{Kind: "normal", Seconds: f(600), InclinePct: f(0)},
	}}}})
	want := []string{"20 min · 8% · 5.0 km/h", "warmup · 1.5 min · 4.5 km/h", "normal · 1200 sec", "10 min · 0%"}
	for i, w := range want {
		if got := v.Exercises[0].Sets[i]; got != w {
			t.Errorf("set %d: %q want %q", i, got, w)
		}
	}
}

func TestCoachRecentWorkoutsIncludeInclineAndSpeedOnlyWhenParsed(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	fc := &fakeCoach{call: []string{"get_recent_workouts"}, reply: "ok"}
	h := coachHandler(t, fc)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, h.Service.Location())
	h.Service.Clock.Now = func() time.Time { return now }
	entries := []models.TrainingEntry{{ID: "a", Kind: "workout", Title: "Cardio", StartedAt: now.AddDate(0, 0, -1), Exercises: []models.TrainingExercise{
		{Name: "Treadmill", PrimaryMuscle: "cardio", Sets: []models.TrainingSet{{Kind: "normal", Seconds: f(1200), InclinePct: f(8), SpeedKmh: f(5)}}},
		{Name: "Bike", PrimaryMuscle: "cardio", Sets: []models.TrainingSet{{Kind: "normal", Seconds: f(600)}}},
	}}}
	if err := h.Service.Repo.SaveTraining("hevy", entries, now); err != nil {
		t.Fatal(err)
	}
	if w := ask(h, "How was cardio?", "http://example.com"); w.Code != http.StatusSeeOther {
		t.Fatalf("ask: %d", w.Code)
	}
	out := fc.toolData["get_recent_workouts"]
	if !strings.Contains(out, `"incline_pct":8`) || !strings.Contains(out, `"speed_kmh":5`) {
		t.Fatalf("missing incline or speed: %s", out)
	}
	if strings.Count(out, "incline_pct") != 1 || strings.Count(out, "speed_kmh") != 1 {
		t.Fatalf("absent values were emitted: %s", out)
	}
}

func TestCardioTrendOnProgressView(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	h := testHandler(t)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, h.Service.Location())
	h.Service.Clock.Now = func() time.Time { return now }
	get := func() string {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health/training?view=progress", nil))
		if r.Code != 200 {
			t.Fatal(r.Code)
		}
		return r.Body.String()
	}
	if err := h.Service.Repo.SaveTraining("hevy", []models.TrainingEntry{{ID: "s", Kind: "workout", Title: "Legs", StartedAt: now.AddDate(0, 0, -1)}}, now); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(get(), "Cardio minutes per week") {
		t.Fatal("cardio card shown without cardio")
	}
	entries := []models.TrainingEntry{{ID: "c", Kind: "workout", Title: "Cardio", StartedAt: now.AddDate(0, 0, -1), Exercises: []models.TrainingExercise{
		{Name: "Treadmill", PrimaryMuscle: "cardio", Sets: []models.TrainingSet{{Kind: "normal", Seconds: f(1800), InclinePct: f(8), SpeedKmh: f(5)}}},
	}}}
	if err := h.Service.Repo.SaveTraining("hevy", entries, now); err != nil {
		t.Fatal(err)
	}
	page := get()
	for _, want := range []string{"Cardio minutes per week", "This week · 30 min", "1 session · avg 8.0% · avg 5.0 km/h"} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q", want)
		}
	}
}
