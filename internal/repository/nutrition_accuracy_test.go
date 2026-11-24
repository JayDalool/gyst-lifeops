package repository

import (
	"context"
	"fmt"
	"math"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-179: the weekly gap between the AI estimate and what was saved. Fictional
// fixtures only. Weeks start on Monday.

const accInstant = "2026-11-16T15:00:00.000000000Z"

var accKey int

func accMeal(t *testing.T, r *Repository, date, via string, items ...DiaryItemInsert) (mealID int64, ids []int64) {
	t.Helper()
	accKey++
	id, ids, _, err := r.CreateMealWithItems(context.Background(), DiaryMealInsert{LocalDate: date, MealSlot: "dinner", CreatedVia: via,
		IdempotencyKey: fmt.Sprintf("%032x", accKey), CreatedAt: accInstant}, items)
	if err != nil {
		t.Fatal(err)
	}
	return id, ids
}

func f64(v float64) *float64 { return &v }

func accItem(est, saved float64, estKcal, kcal *float64) DiaryItemInsert {
	it := DiaryItemInsert{Name: "Test item", Unit: "g", Amount: saved, Grams: f64(saved), QuantityProvenance: "user_confirmed", NutritionProv: "user",
		AIEstGrams: f64(est), Nutrients: models.Nutrients{Kcal: kcal}}
	if estKcal != nil {
		it.AIEst = &models.Nutrients{Kcal: estKcal}
	}
	return it
}

func near(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 0.005 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func TestAIAccuracyByWeek(t *testing.T) {
	r := testRepository(t)
	// Week of Monday 2026-11-09, including its Sunday 2026-11-15: +30, -40, 0.
	accMeal(t, r, "2026-11-09", "camera", accItem(150, 180, nil, nil))
	accMeal(t, r, "2026-11-15", "camera", accItem(200, 160, nil, nil), accItem(100, 100, nil, nil))
	// Excluded: no estimate grams, a manual meal, an archived item, an archived meal.
	noEst := accItem(0, 90, nil, nil)
	noEst.AIEstGrams = nil
	accMeal(t, r, "2026-11-10", "camera", noEst)
	accMeal(t, r, "2026-11-10", "manual", accItem(50, 500, nil, nil))
	_, ids := accMeal(t, r, "2026-11-11", "camera", accItem(10, 999, nil, nil))
	if err := r.ArchiveDiaryItem(ids[0], accInstant); err != nil {
		t.Fatal(err)
	}
	mealID, _ := accMeal(t, r, "2026-11-12", "camera", accItem(10, 888, nil, nil))
	if _, err := r.db.Exec(`UPDATE nutrition_meals SET archived_at = ? WHERE id = ?`, accInstant, mealID); err != nil {
		t.Fatal(err)
	}
	// Monday 2026-11-16 starts the next week: kcal est 200 saved 230 (+30), plus an item with no kcal estimate.
	accMeal(t, r, "2026-11-16", "camera", accItem(100, 120, f64(200), f64(230)), accItem(100, 100, nil, f64(500)))
	// Outside the asked range.
	accMeal(t, r, "2026-12-01", "camera", accItem(100, 500, nil, nil))

	weeks, err := r.AIAccuracyByWeek("2026-11-09", "2026-11-22")
	if err != nil {
		t.Fatal(err)
	}
	if len(weeks) != 2 || weeks[0].WeekStart != "2026-11-09" || weeks[1].WeekStart != "2026-11-16" {
		t.Fatalf("weeks %+v", weeks)
	}
	w := weeks[0]
	if w.Items != 3 || w.Grams.Paired != 3 {
		t.Fatalf("week 1 counts: %+v", w)
	}
	near(t, "bias", w.Grams.Bias, -3.33) // (30 - 40 + 0) / 3
	near(t, "mae", w.Grams.MAE, 23.33)   // (30 + 40 + 0) / 3
	// No macro estimate that week: nil, not 0, and nothing paired.
	for name, s := range map[string]models.AIAccuracyStat{"kcal": w.Kcal, "protein": w.ProteinG, "carbs": w.CarbsG, "fat": w.FatG} {
		if s.Bias != nil || s.MAE != nil || s.Paired != 0 {
			t.Errorf("%s must be nil with no estimates: %+v", name, s)
		}
	}
	w2 := weeks[1]
	if w2.Items != 2 || w2.Grams.Paired != 2 || w2.Kcal.Paired != 1 {
		t.Fatalf("week 2 counts: %+v", w2)
	}
	near(t, "kcal bias", w2.Kcal.Bias, 30)
	near(t, "kcal mae", w2.Kcal.MAE, 30)
	near(t, "grams bias", w2.Grams.Bias, 10) // (+20 + 0) / 2

	// Inclusive range ends and an empty range.
	one, err := r.AIAccuracyByWeek("2026-11-15", "2026-11-15")
	if err != nil || len(one) != 1 || one[0].WeekStart != "2026-11-09" || one[0].Items != 2 {
		t.Fatalf("single Sunday: %+v %v", one, err)
	}
	none, err := r.AIAccuracyByWeek("2027-01-01", "2027-01-07")
	if err != nil || len(none) != 0 || none == nil {
		t.Fatalf("empty: %+v %v", none, err)
	}
}
