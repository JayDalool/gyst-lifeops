package services

import (
	"reflect"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func fp(v float64) *float64 { return &v }

func set(kind string, kg, reps float64) models.TrainingSet {
	return models.TrainingSet{Kind: kind, LoadKG: fp(kg), Reps: fp(reps)}
}

func workout(at time.Time, minutes int, exercises ...models.TrainingExercise) models.TrainingEntry {
	e := models.TrainingEntry{ID: at.Format(time.RFC3339), Kind: "workout", StartedAt: at, Exercises: exercises}
	if minutes > 0 {
		e.EndedAt = at.Add(time.Duration(minutes) * time.Minute)
	}
	return e
}

func exercise(name, muscle string, sets ...models.TrainingSet) models.TrainingExercise {
	return models.TrainingExercise{Name: name, PrimaryMuscle: muscle, Sets: sets}
}

func TestSummarizeTrainingWeek(t *testing.T) {
	loc, _ := time.LoadLocation("America/Winnipeg")
	now := time.Date(2026, 10, 4, 23, 45, 0, 0, loc) // Sunday
	entries := []models.TrainingEntry{
		// Monday: 3×10 at 100 kg = 3000 kg, warmup and bodyweight sets excluded.
		workout(time.Date(2026, 9, 28, 18, 0, 0, 0, loc), 45, exercise("Leg Press", "quadriceps",
			set("warmup", 60, 10), set("normal", 100, 10), set("normal", 100, 10), set("failure", 100, 10),
			models.TrainingSet{Kind: "normal", Reps: fp(15)})),
		// Sunday 23:30 local is 04:30 UTC Monday: still this week, last day.
		workout(time.Date(2026, 10, 4, 23, 30, 0, 0, loc), 0, exercise("Plank", "abdominals", models.TrainingSet{Kind: "normal", Seconds: fp(45)})),
		// Previous Sunday and a future workout are outside the week.
		workout(time.Date(2026, 9, 27, 12, 0, 0, 0, loc), 60),
		workout(time.Date(2026, 10, 5, 0, 30, 0, 0, loc), 60),
		{Kind: "routine", Title: "Push"},
	}
	week := SummarizeTrainingWeek(entries, now, loc)
	want := TrainingWeek{Workouts: 2, Days: [7]bool{true, false, false, false, false, false, true}, Today: 6, VolumeKG: 3000, Minutes: 45, UnknownTime: 1}
	if week != want {
		t.Fatalf("got %+v\nwant %+v", week, want)
	}
	if got := SummarizeTrainingWeek(nil, now, loc); got.Workouts != 0 || got.VolumeKG != 0 || got.Today != 6 {
		t.Fatalf("empty week %+v", got)
	}
}

func TestWeekSpanningDSTEnd(t *testing.T) {
	loc, _ := time.LoadLocation("America/Winnipeg")
	// DST ends Sunday 2026-11-01; the week of Oct 26 has a 25-hour day.
	now := time.Date(2026, 11, 1, 22, 0, 0, 0, loc)
	week := SummarizeTrainingWeek([]models.TrainingEntry{workout(time.Date(2026, 11, 1, 21, 0, 0, 0, loc), 30)}, now, loc)
	if !week.Days[6] || week.Today != 6 {
		t.Fatalf("DST week %+v", week)
	}
}

func TestWorkoutsPerWeek(t *testing.T) {
	loc, _ := time.LoadLocation("America/Winnipeg")
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, loc) // Thursday
	entries := []models.TrainingEntry{
		workout(time.Date(2026, 9, 15, 9, 0, 0, 0, loc), 30), // two weeks ago
		workout(time.Date(2026, 9, 29, 9, 0, 0, 0, loc), 30), // this week
		workout(time.Date(2026, 9, 30, 9, 0, 0, 0, loc), 30), // this week
		workout(time.Date(2026, 10, 2, 9, 0, 0, 0, loc), 30), // future
	}
	if got := WorkoutsPerWeek(entries, now, loc, 3); !reflect.DeepEqual(got, []int{1, 0, 2}) {
		t.Fatalf("got %v", got)
	}
}

func TestExerciseProgress(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	day := func(n int) time.Time { return now.AddDate(0, 0, -n) }
	entries := []models.TrainingEntry{
		workout(day(80), 0, exercise("Leg Press", "quadriceps", set("normal", 100, 10))), // e1RM 133.3
		workout(day(40), 0, exercise("leg press ", "quadriceps", set("normal", 120, 10), set("warmup", 200, 10))),
		workout(day(2), 0, exercise("Leg Press", "quadriceps", set("normal", 150, 10), set("normal", 160, 15))), // 15 reps ignored; e1RM 200
		workout(day(30), 0, exercise("Chest Press", "chest", set("normal", 50, 8))),
		workout(day(3), 0, exercise("Chest Press", "chest", set("normal", 55, 8))),
		workout(day(5), 0, exercise("Curl", "biceps", set("normal", 20, 10))), // single session: no trend
		workout(day(4), 0, exercise("Push-up", "chest", models.TrainingSet{Kind: "normal", Reps: fp(20)}), exercise("Push-up", "chest", models.TrainingSet{Kind: "normal", Reps: fp(20)})),
		workout(day(200), 0, exercise("Chest Press", "chest", set("normal", 10, 8))), // outside window
	}
	got := ExerciseProgress(entries, now, 84)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	lp, cp := got[0], got[1]
	if lp.Name != "Leg Press" || lp.Sessions != 3 || lp.ChangePct != 50 || lp.BestSet != "150 kg × 10" || !reflect.DeepEqual(lp.Series, []float64{133.3, 160, 200}) {
		t.Fatalf("leg press %+v", lp)
	}
	if cp.Name != "Chest Press" || cp.ChangePct != 10 || cp.Sessions != 2 {
		t.Fatalf("chest press %+v", cp)
	}
	if len(ExerciseProgress(entries, now, 35)) != 1 {
		t.Fatal("a 5-week window should keep only chest press")
	}
	if ExerciseProgress(nil, now, 84) == nil {
		t.Fatal("want empty slice, not nil")
	}
}

func TestMuscleGroupShares(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	entries := []models.TrainingEntry{
		workout(now.AddDate(0, 0, -1), 0,
			exercise("Squat", "quadriceps", set("normal", 100, 5), set("normal", 100, 5), set("warmup", 60, 5)),
			exercise("Row", "lats", set("normal", 60, 10)),
			exercise("Bike", "cardio", models.TrainingSet{Kind: "normal", Seconds: fp(600)})),
		workout(now.AddDate(0, 0, -40), 0, exercise("Bench", "chest", set("normal", 80, 5))),
	}
	got := MuscleGroupShares(entries, now, 30)
	names, pcts := []string{}, 0
	for _, g := range got {
		names = append(names, g.Name)
		pcts += g.Percent
	}
	if !reflect.DeepEqual(names, []string{"Legs", "Back", "Other"}) || pcts != 100 || got[0].Sets != 2 || got[0].Percent != 50 {
		t.Fatalf("got %+v", got)
	}
	if MuscleGroupShares(nil, now, 30) != nil {
		t.Fatal("no sets should give no shares")
	}
}

// Hevy pound entries arrive as long kilogram floats; display rounds to 0.1 kg.
func TestFormatLoadKG(t *testing.T) {
	for in, want := range map[float64]string{43.0913263963858: "43.1", 24.9476100189602: "24.9", 104.5: "104.5", 18: "18", 0.04: "0"} {
		if got := FormatLoadKG(in); got != want {
			t.Errorf("FormatLoadKG(%v) = %q, want %q", in, got, want)
		}
	}
	if got := formatSet(set("normal", 43.0913263963858, 12)); got != "43.1 kg × 12" {
		t.Fatalf("best set %q", got)
	}
}
