package services

import (
	"lifeops-tracker/internal/models"
	"testing"
	"time"
)

func TestMuscleSummarySeparatesPlansHistoryAndSecondary(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	exercises := []models.TrainingExercise{{PrimaryMuscle: "chest", SecondaryMuscles: []string{"chest", "triceps", "triceps"}, Sets: []models.TrainingSet{{Kind: "warmup"}, {Kind: "normal"}, {Kind: "failure"}, {Kind: "dropset"}, {Kind: "unexpected"}}}, {Sets: []models.TrainingSet{{Kind: "normal"}}}}
	entries := []models.TrainingEntry{{ID: "r", Kind: "routine", Exercises: exercises}, {ID: "w", Kind: "workout", StartedAt: now.Add(-time.Hour), Exercises: exercises}, {ID: "old", Kind: "workout", StartedAt: now.AddDate(0, 0, -8), Exercises: exercises}, {ID: "future", Kind: "workout", StartedAt: now.Add(time.Hour), Exercises: exercises}}
	for _, mode := range []string{"planned", "7", "30"} {
		s := SummarizeTraining(entries, mode, "", now)
		n := 1
		if mode == "30" {
			n = 2
		}
		if s.Entries != n || s.Sets != 4*n || s.Unclassified != n || len(s.Muscles) != 2 || s.Muscles[0].Primary != 3*n || s.Muscles[1].Secondary != 3*n || s.Muscles[1].Primary != 0 {
			t.Fatalf("%s: %+v", mode, s)
		}
		if s.Heat["chest"] != 3 || s.Heat["triceps"] != 0 {
			t.Fatal("secondary sets coloured as primary")
		}
	}
	if s := SummarizeTraining(entries, "planned", "different", now); s.Sets != 0 {
		t.Fatal("routine filter ignored")
	}
}
