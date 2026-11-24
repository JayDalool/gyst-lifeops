package services

import (
	"context"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-180: DayIntake.Estimated is the AI-estimated share of a GYST day, from
// the live items stored with nutrition_provenance ai_estimate. Fictional values.
func TestDayIntakeEstimatedCountsOnlyLiveAIEstimateItems(t *testing.T) {
	s, _ := diaryService(t)
	est := nut(200, 10, 20, 8)
	cameraSave(t, s, nil,
		CameraItem{Name: "Test stew", Grams: 100, AIEstGrams: cp(100), AIEstNutrients: est, Nutrients: *est},
		CameraItem{Name: "Test soup", Grams: 100, AIEstGrams: cp(100), AIEstNutrients: nut(120, 4, 16, 4), Nutrients: *nut(120, 4, 16, 4)})
	// Edited by hand: Jay's own value, not counted as an estimate (c5).
	cameraSave(t, s, nil, CameraItem{Name: "Test edited", Grams: 100, AIEstGrams: cp(100), AIEstNutrients: est, Nutrients: *nut(150, 10, 20, 8)})
	// No estimate at all: provenance user.
	cameraSave(t, s, nil, CameraItem{Name: "Test plain", Grams: 100, Nutrients: *nut(50, 1, 1, 1)})

	in, err := s.DayIntake(s.TodayString())
	if err != nil {
		t.Fatal(err)
	}
	if in.Source != models.IntakeSourceGYST || in.Items != 4 {
		t.Fatalf("intake %+v", in)
	}
	if in.Estimated.Items != 2 || *in.Estimated.Kcal != 320 || *in.Estimated.ProteinG != 14 {
		t.Fatalf("estimated %+v", in.Estimated)
	}
	if *in.Counted.Kcal != 520 { // Estimated is a subset of the counted total, never added to it
		t.Fatalf("counted %v", *in.Counted.Kcal)
	}

	// Removing an estimated item removes it from the estimate.
	live, _ := s.Repo.ListDiaryItems(s.TodayString())
	for _, it := range live {
		if it.Name == "Test soup" {
			if err := s.ArchiveDiaryItem(context.Background(), DiaryActor{}, it.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	in, err = s.DayIntake(s.TodayString())
	if err != nil {
		t.Fatal(err)
	}
	if in.Estimated.Items != 1 || *in.Estimated.Kcal != 200 {
		t.Fatalf("after removal %+v", in.Estimated)
	}
}

func TestDayIntakeEstimatedNilWhenNoneAndForAppleHealthDays(t *testing.T) {
	s, _ := diaryService(t)
	cameraSave(t, s, nil, CameraItem{Name: "Test plain", Grams: 100, Nutrients: *nut(50, 1, 1, 1)})
	in, _ := s.DayIntake(s.TodayString())
	if in.Estimated.Items != 0 || in.Estimated.Kcal != nil || in.Estimated.ProteinG != nil {
		t.Fatalf("estimated %+v", in.Estimated)
	}
	snap := models.DailyHealthSnapshot{Date: "2026-11-20"}
	snap.DietaryCalories = cp(1800)
	apple := ResolveDayIntake("2026-11-20", nil, &snap)
	if apple.Estimated.Items != 0 || apple.Source != models.IntakeSourceAppleHealth {
		t.Fatalf("apple %+v", apple)
	}
}
