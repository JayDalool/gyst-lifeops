package services

import (
	"testing"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
)

func TestCardioPerWeek(t *testing.T) {
	loc, _ := clock.Load("America/Winnipeg")
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, loc) // Wednesday
	tread := func(sec, incline, speed float64) models.TrainingExercise {
		s := models.TrainingSet{Kind: "normal", Seconds: fp(sec)}
		if incline >= 0 {
			s.InclinePct = fp(incline)
		}
		if speed >= 0 {
			s.SpeedKmh = fp(speed)
		}
		return exercise("Treadmill", "cardio", s)
	}
	entries := []models.TrainingEntry{
		workout(time.Date(2026, 10, 5, 7, 0, 0, 0, loc), 0, tread(1200, 8, 5)),
		workout(time.Date(2026, 10, 6, 7, 0, 0, 0, loc), 0, tread(600, 4, -1)),
		// Last week: parsed values absent, minutes still counted.
		workout(time.Date(2026, 9, 29, 7, 0, 0, 0, loc), 0, tread(1800, -1, -1)),
		// Strength and a plank (not cardio) never count.
		workout(time.Date(2026, 10, 6, 18, 0, 0, 0, loc), 0, exercise("Plank", "abdominals", models.TrainingSet{Kind: "normal", Seconds: fp(300)}), exercise("Press", "chest", set("normal", 50, 5))),
		// Older than 8 weeks.
		workout(time.Date(2026, 7, 1, 7, 0, 0, 0, loc), 0, tread(3600, 5, 5)),
	}
	weeks := CardioPerWeek(entries, now, loc, 8)
	if len(weeks) != 8 {
		t.Fatal(len(weeks))
	}
	cur, prev := weeks[7], weeks[6]
	if cur.Minutes != 30 || cur.Sessions != 2 || cur.InclinePct == nil || *cur.InclinePct != 6 || cur.SpeedKmh == nil || *cur.SpeedKmh != 5 {
		t.Fatalf("this week %+v", cur)
	}
	if prev.Minutes != 30 || prev.Sessions != 1 || prev.InclinePct != nil || prev.SpeedKmh != nil {
		t.Fatalf("last week %+v", prev)
	}
	for _, w := range weeks[:6] {
		if w.Minutes != 0 || w.Sessions != 0 {
			t.Fatalf("earlier week %+v", w)
		}
	}
}
