package services

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-179: the camera save keeps the AI's own estimate beside what Jay saved.
// Fictional values only.

func cp(v float64) *float64 { return &v }

func cameraSave(t *testing.T, s *Service, portion *float64, items ...CameraItem) []models.DiaryItem {
	t.Helper()
	res, err := s.AddCameraMeal(context.Background(), DiaryActor{}, CameraMeal{IdempotencyKey: newKey(), MealSlot: "dinner", Items: items, Portion: portion})
	if err != nil {
		t.Fatal(err)
	}
	live, err := s.Repo.ListDiaryItems(s.TodayString())
	if err != nil {
		t.Fatal(err)
	}
	var out []models.DiaryItem
	for _, it := range live {
		if it.MealID == res.MealID {
			out = append(out, it)
		}
	}
	return out
}

func nut(kcal, p, c, f float64) *models.Nutrients {
	return &models.Nutrients{Kcal: cp(kcal), ProteinG: cp(p), CarbsG: cp(c), FatG: cp(f)}
}

func TestCameraPortionRules(t *testing.T) {
	s, _ := diaryService(t)
	for _, bad := range []float64{0, 0.45, 2.05, 1.03, 0.49, -1, math.NaN(), math.Inf(1)} {
		_, err := s.AddCameraMeal(context.Background(), DiaryActor{}, CameraMeal{IdempotencyKey: newKey(), MealSlot: "dinner",
			Items: []CameraItem{{Name: "Test rice", Grams: 100}}, Portion: cp(bad)})
		var inv *HealthValidationError
		if !errors.As(err, &inv) || !strings.Contains(inv.Error(), "portion") {
			t.Errorf("portion %v accepted: %v", bad, err)
		}
	}
	for _, ok := range []float64{0.5, 1, 1.05, 1.2, 1.95, 2} {
		cameraSave(t, s, cp(ok), CameraItem{Name: "Test rice", Grams: 100})
	}
	cameraSave(t, s, nil, CameraItem{Name: "Test rice", Grams: 100}) // absent = x1
}

func TestCameraPortionForcesUserConfirmedEvenWhenRoundingEqualsEstimate(t *testing.T) {
	s, _ := diaryService(t)
	// Grams equal the estimate (as after x1.2 then back by hand); x1 keeps it AI,
	// any other portion is Jay's change.
	at1 := cameraSave(t, s, cp(1), CameraItem{Name: "Test rice", Grams: 150, AIEstGrams: cp(150)})
	at12 := cameraSave(t, s, cp(1.2), CameraItem{Name: "Test rice", Grams: 150, AIEstGrams: cp(150)}, CameraItem{Name: "Test beans", Grams: 90, AIEstGrams: cp(75)})
	if at1[0].QuantityProvenance != "ai_estimate" {
		t.Fatalf("x1: %+v", at1[0])
	}
	for _, it := range at12 {
		if it.QuantityProvenance != "user_confirmed" || it.AIEstGrams == nil {
			t.Fatalf("x1.2: %+v", it)
		}
	}
}

func TestCameraGramsAreStoredToOneDecimalAndMustBePositive(t *testing.T) {
	s, _ := diaryService(t)
	it := cameraSave(t, s, nil, CameraItem{Name: "Test rice", Grams: 150.04})[0]
	if *it.Grams != 150 || it.Amount != 150 {
		t.Fatalf("%+v", it)
	}
	_, err := s.AddCameraMeal(context.Background(), DiaryActor{}, CameraMeal{IdempotencyKey: newKey(), MealSlot: "dinner", Items: []CameraItem{{Name: "Test rice", Grams: 0.04}}})
	var inv *HealthValidationError
	if !errors.As(err, &inv) {
		t.Fatalf("grams rounding to 0 accepted: %v", err)
	}
}

func TestCameraAIEstimateNutritionProvenanceAndKeptEstimate(t *testing.T) {
	s, _ := diaryService(t)
	est := nut(200, 10, 30, 5) // at 100 g
	// Untouched: posted values are the estimate scaled to 150 g (within the tolerance).
	got := cameraSave(t, s, nil, CameraItem{Name: "Test stew", Grams: 150, AIEstGrams: cp(100), AIEstNutrients: est,
		Nutrients: models.Nutrients{Kcal: cp(300), ProteinG: cp(15), CarbsG: cp(45), FatG: cp(7.5)}})[0]
	if got.NutritionProvenance != "ai_estimate" || *got.Kcal != 300 || *got.CarbsG != 45 {
		t.Fatalf("unedited: %+v", got)
	}
	if *got.AIEstKcal != 200 || *got.AIEstProteinG != 10 || *got.AIEstCarbsG != 30 || *got.AIEstFatG != 5 || *got.AIEstGrams != 100 {
		t.Fatalf("estimate not kept as the AI gave it: %+v", got)
	}
	// Edited kcal: Jay's own, the estimate is still kept untouched.
	ed := cameraSave(t, s, nil, CameraItem{Name: "Test stew", Grams: 150, AIEstGrams: cp(100), AIEstNutrients: est,
		Nutrients: models.Nutrients{Kcal: cp(250), ProteinG: cp(15), CarbsG: cp(45), FatG: cp(7.5)}})[0]
	if ed.NutritionProvenance != "user" || *ed.Kcal != 250 || *ed.AIEstKcal != 200 || *ed.AIEstFatG != 5 {
		t.Fatalf("edited: %+v", ed)
	}
	// A cleared value is an edit too, and stays NULL, never 0.
	cl := cameraSave(t, s, nil, CameraItem{Name: "Test stew", Grams: 150, AIEstGrams: cp(100), AIEstNutrients: est,
		Nutrients: models.Nutrients{Kcal: cp(300)}})[0]
	if cl.NutritionProvenance != "user" || cl.ProteinG != nil {
		t.Fatalf("cleared: %+v", cl)
	}
}

func TestCameraRowsWithoutAIEstimateHaveNullEstMacros(t *testing.T) {
	s, _ := diaryService(t)
	food := catalogueFood(t, s, "lent-1", "Test lentils", fp(100), fp(8), nil)
	v := food.Version
	got := cameraSave(t, s, nil,
		CameraItem{Name: "Test lentils", Grams: 100, AIEstGrams: cp(100), FoodID: &food.ID, FoodVersion: &v, NutritionProv: "database", Nutrients: models.Nutrients{Kcal: cp(100), ProteinG: cp(8)}},
		CameraItem{Name: "Hand added oil", Grams: 10, Nutrients: models.Nutrients{Kcal: cp(90)}},
		CameraItem{Name: "No estimate grams", Grams: 50, AIEstNutrients: nut(100, 1, 1, 1)})
	for _, it := range got {
		if it.AIEstKcal != nil || it.AIEstProteinG != nil || it.AIEstCarbsG != nil || it.AIEstFatG != nil {
			t.Fatalf("estimate macros must stay NULL: %+v", it)
		}
	}
}

func TestDiaryEditAmountLeavesAIEstimateAlone(t *testing.T) {
	s, _ := diaryService(t)
	food := catalogueFood(t, s, "lent-2", "Test lentils", fp(100), fp(8), nil)
	v := food.Version
	it := cameraSave(t, s, nil, CameraItem{Name: "Test lentils", Grams: 100, AIEstGrams: cp(120), AIEstNutrients: nut(150, 9, 20, 2),
		FoodID: &food.ID, FoodVersion: &v, NutritionProv: "database", Nutrients: models.Nutrients{Kcal: cp(100), ProteinG: cp(8)}})[0]
	if it.AIEstKcal == nil {
		t.Fatalf("estimate not stored for a matched item: %+v", it)
	}
	if err := s.UpdateDiaryItemAmount(context.Background(), DiaryActor{}, it.ID, 60); err != nil {
		t.Fatal(err)
	}
	after, _ := s.Repo.GetDiaryItem(it.ID)
	if *after.Grams != 60 || *after.Kcal != 60 {
		t.Fatalf("amount not changed: %+v", after)
	}
	if *after.AIEstGrams != 120 || *after.AIEstKcal != 150 || *after.AIEstProteinG != 9 || *after.AIEstCarbsG != 20 || *after.AIEstFatG != 2 {
		t.Fatalf("Edit amount changed the AI estimate: %+v", after)
	}
}

// Codex P2 on #278: an owner edit of 0.1 inside the old 0.11 tolerance was
// overwritten with the estimate and labelled ai_estimate. The owner's entry wins
// (docs/RULES.md 3.2): only the exact displayed value keeps the AI label.
func TestCameraAIEstimateSmallOwnerEditIsKept(t *testing.T) {
	s, _ := diaryService(t)
	est := &models.Nutrients{Kcal: cp(15)}
	for name, posted := range map[string]float64{"+0.1": 15.1, "-0.1": 14.9, "+0.05": 15.05} {
		it := cameraSave(t, s, nil, CameraItem{Name: "Test sauce", Grams: 100, AIEstGrams: cp(100), AIEstNutrients: est,
			Nutrients: models.Nutrients{Kcal: cp(posted)}})[0]
		if it.NutritionProvenance != "user" || *it.Kcal != posted || *it.AIEstKcal != 15 {
			t.Errorf("%s: %+v", name, it)
		}
	}
	it := cameraSave(t, s, nil, CameraItem{Name: "Test sauce", Grams: 100, AIEstGrams: cp(100), AIEstNutrients: est, Nutrients: models.Nutrients{Kcal: cp(15)}})[0]
	if it.NutritionProvenance != "ai_estimate" {
		t.Errorf("displayed value must keep the AI label: %+v", it)
	}
}

// Codex P2 on #278: values recomputed at the stored (rounded) grams can leave
// the CHECK range even when the posted ones were inside it; that must be the
// review's 400, not a database constraint error.
func TestCameraFinalNutrientsAreRangeChecked(t *testing.T) {
	s, _ := diaryService(t)
	// 104 kcal at 0.1 g: at the posted 96.1538 g the review shows 100000 (within
	// range), but the stored 96.2 g rescales to 100048.
	_, err := s.AddCameraMeal(context.Background(), DiaryActor{}, CameraMeal{IdempotencyKey: newKey(), MealSlot: "dinner", Items: []CameraItem{{
		Name: "Test oil", Grams: 96.1538, AIEstGrams: cp(0.1), AIEstNutrients: &models.Nutrients{Kcal: cp(104)},
		Nutrients: models.Nutrients{Kcal: cp(100000)}}}})
	var inv *HealthValidationError
	if !errors.As(err, &inv) {
		t.Fatalf("want the review's validation error, got %v", err)
	}
}
