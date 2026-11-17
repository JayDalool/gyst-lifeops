package handlers

import (
	"encoding/json"
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
		{Kind: "normal", Seconds: f(900), DistanceM: f(1500), SpeedKmh: f(6)},
	}}}})
	want := []string{"20 min · 8% · 5.0 km/h", "1.5 min · 4.5 km/h · warmup", "normal · 1200 sec", "10 min · 0%", "15 min · 6.0 km/h · 1500 m"}
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
		{Name: "Intervals", PrimaryMuscle: "cardio", Sets: []models.TrainingSet{
			{Kind: "normal", Seconds: f(300), InclinePct: f(6), SpeedKmh: f(4.5)},
			{Kind: "normal", Seconds: f(300), InclinePct: f(8), SpeedKmh: f(5)},
			{Kind: "normal", Seconds: f(300), InclinePct: f(10), SpeedKmh: f(5.5)},
		}},
	}}}
	if err := h.Service.Repo.SaveTraining("hevy", entries, now); err != nil {
		t.Fatal(err)
	}
	if w := ask(h, "How was cardio?", "http://example.com"); w.Code != http.StatusSeeOther {
		t.Fatalf("ask: %d", w.Code)
	}
	out := fc.toolData["get_recent_workouts"]
	var got struct {
		Workouts []struct {
			Exercises []map[string]any
		}
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || len(got.Workouts) != 1 || len(got.Workouts[0].Exercises) != 3 {
		t.Fatalf("bad output %v: %s", err, out)
	}
	tread, bike, intervals := got.Workouts[0].Exercises[0], got.Workouts[0].Exercises[1], got.Workouts[0].Exercises[2]
	if tread["incline_pct"] != 8.0 || tread["speed_kmh"] != 5.0 || len(tread["sets"].([]any)) != 1 {
		t.Fatalf("uniform exercise: %v", tread)
	}
	if _, ok := bike["incline_pct"]; ok {
		t.Fatalf("absent values were emitted: %v", bike)
	}
	if _, ok := bike["sets"]; ok {
		t.Fatalf("sets emitted without parsed values: %v", bike)
	}
	// Differing sets: per-set values, exercise-level null (never the first set's).
	if v, ok := intervals["incline_pct"]; !ok || v != nil || intervals["speed_kmh"] != nil {
		t.Fatalf("exercise-level values for mixed sets: %v", intervals)
	}
	sets := intervals["sets"].([]any)
	last := sets[2].(map[string]any)
	if len(sets) != 3 || sets[0].(map[string]any)["incline_pct"] != 6.0 || last["incline_pct"] != 10.0 || last["speed_kmh"] != 5.5 || last["duration_min"] != 5.0 {
		t.Fatalf("per-set values: %v", sets)
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
