package services

import (
	"errors"
	"testing"

	"lifeops-tracker/internal/models"
)

// Fictional values only.

func syncedSnapshot(date string, kcal, protein *float64) *models.DailyHealthSnapshot {
	return &models.DailyHealthSnapshot{Date: date, HealthMetrics: models.HealthMetrics{DietaryCalories: kcal, ProteinG: protein}}
}

func TestResolveDayIntakeSourceRule(t *testing.T) {
	const day = "2026-11-16"
	gyst := func(items int, kcal, protein *float64, gaps models.NutrientGaps) *models.NutritionDayTotals {
		return &models.NutritionDayTotals{Date: day, Items: items, Sum: models.Nutrients{Kcal: kcal, ProteinG: protein}, Gaps: gaps}
	}
	stepsOnly := &models.DailyHealthSnapshot{Date: day, HealthMetrics: models.HealthMetrics{StepsWatch: func() *int64 { v := int64(900); return &v }()}}
	cases := []struct {
		name         string
		gyst         *models.NutritionDayTotals
		snap         *models.DailyHealthSnapshot
		source       string
		kcal         *float64 // expected Counted.Kcal
		wantRefKcal  *float64
		wantMissKcal int
	}{
		{"gyst day ignores the synced total for counting", gyst(2, fp(520), fp(30), models.NutrientGaps{}), syncedSnapshot(day, fp(1800), fp(90)), models.IntakeSourceGYST, fp(520), fp(1800), 0},
		{"gyst day without a snapshot", gyst(1, fp(300), nil, models.NutrientGaps{ProteinG: 1}), nil, models.IntakeSourceGYST, fp(300), nil, 0},
		{"day without diary items uses the synced total", nil, syncedSnapshot(day, fp(1800), fp(90)), models.IntakeSourceAppleHealth, fp(1800), nil, 0},
		{"zero diary items (all removed) falls back to Apple Health", gyst(0, nil, nil, models.NutrientGaps{}), syncedSnapshot(day, fp(1800), nil), models.IntakeSourceAppleHealth, fp(1800), nil, 0},
		{"a synced recorded zero is a value", nil, syncedSnapshot(day, fp(0), nil), models.IntakeSourceAppleHealth, fp(0), nil, 0},
		{"steps only is no intake, not zero", nil, stepsOnly, models.IntakeSourceNone, nil, nil, 0},
		{"nothing at all", nil, nil, models.IntakeSourceNone, nil, nil, 0},
		{"partial sums record the gap", gyst(3, fp(410), nil, models.NutrientGaps{Kcal: 1}), syncedSnapshot(day, fp(2000), nil), models.IntakeSourceGYST, fp(410), fp(2000), 1},
		{"gyst day whose items have no kcal counts nil, never the synced kcal", gyst(1, nil, fp(12), models.NutrientGaps{Kcal: 1}), syncedSnapshot(day, fp(2000), nil), models.IntakeSourceGYST, nil, fp(2000), 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveDayIntake(day, tc.gyst, tc.snap)
			if got.Source != tc.source {
				t.Fatalf("source = %q, want %q", got.Source, tc.source)
			}
			if got.Resolved != (tc.source != models.IntakeSourceNone) {
				t.Fatalf("resolved = %v for source %q", got.Resolved, got.Source)
			}
			if !sameF(got.Counted.Kcal, tc.kcal) {
				t.Fatalf("counted kcal = %v, want %v (never a sum of the two sources)", deref(got.Counted.Kcal), deref(tc.kcal))
			}
			if tc.wantRefKcal == nil {
				if got.Reference != nil {
					t.Fatalf("unexpected reference %+v", got.Reference)
				}
			} else if got.Reference == nil || !sameF(got.Reference.Kcal, tc.wantRefKcal) {
				t.Fatalf("reference = %+v, want kcal %v", got.Reference, *tc.wantRefKcal)
			}
			if got.Missing.Kcal != tc.wantMissKcal {
				t.Fatalf("missing kcal = %d, want %d", got.Missing.Kcal, tc.wantMissKcal)
			}
			if got.Snapshot != tc.snap {
				t.Fatal("the synced row must stay available for steps and energy")
			}
		})
	}
}

func sameF(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func deref(a *float64) any {
	if a == nil {
		return nil
	}
	return *a
}

func TestScaleNutrients(t *testing.T) {
	grams := func(v float64) *float64 { return &v }
	per100 := models.NutritionFood{Basis: "per_100g", Nutrients: models.Nutrients{Kcal: fp(380), ProteinG: fp(13.5), CarbsG: fp(60), FatG: fp(7)}} // no fiber: NULL
	per100Serv := per100
	per100Serv.ServingGrams = grams(40)
	perServ := models.NutritionFood{Basis: "per_serving", Nutrients: models.Nutrients{Kcal: fp(90), ProteinG: fp(4), FiberG: fp(2)}}
	perServGrams := perServ
	perServGrams.ServingGrams = grams(30)

	cases := []struct {
		name      string
		food      models.NutritionFood
		unit      string
		amount    float64
		kcal      float64
		wantGrams *float64
		wantErr   bool
	}{
		{"per_100g in g", per100, "g", 50, 190, grams(50), false},
		{"per_100g in servings needs serving grams", per100, "serving", 1, 0, nil, true},
		{"per_100g in servings", per100Serv, "serving", 2, 304, grams(80), false},
		{"per_serving in servings", perServ, "serving", 1.5, 135, nil, false},
		{"per_serving in servings with serving grams", perServGrams, "serving", 2, 180, grams(60), false},
		{"per_serving in g needs serving grams", perServ, "g", 30, 0, nil, true},
		{"per_serving in g", perServGrams, "g", 45, 135, grams(45), false},
		{"zero amount refused", per100, "g", 0, 0, nil, true},
		{"unknown unit refused", per100, "oz", 1, 0, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, g, err := ScaleNutrients(tc.food, tc.unit, tc.amount)
			if tc.wantErr {
				if !errors.Is(err, ErrNutritionScale) {
					t.Fatalf("err = %v, want ErrNutritionScale", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if n.Kcal == nil || *n.Kcal != tc.kcal {
				t.Fatalf("kcal = %v, want %v", deref(n.Kcal), tc.kcal)
			}
			if !sameF(g, tc.wantGrams) {
				t.Fatalf("grams = %v, want %v", deref(g), deref(tc.wantGrams))
			}
		})
	}
	// NULL stays NULL: a missing nutrient is never turned into zero.
	n, _, err := ScaleNutrients(per100, "g", 100)
	if err != nil || n.FiberG != nil || n.FatG == nil || *n.FatG != 7 {
		t.Fatalf("NULL fiber must stay NULL: %+v %v", n, err)
	}
}
