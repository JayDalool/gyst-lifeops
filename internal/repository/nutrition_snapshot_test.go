package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

// Codex round b: the day totals and the AI share come from one read snapshot,
// so Estimated is a subset of the total even when a meal is saved between the
// two reads. The seam runs after the first read; a write there (another
// connection, WAL) must show in neither.
func TestDayTotalsAndEstimatedShareOneSnapshot(t *testing.T) {
	r := testRepository(t)
	item := func(kcal float64) DiaryItemInsert {
		return DiaryItemInsert{Name: "Test stew", Unit: "g", Amount: 100, Grams: f64(100), QuantityProvenance: "user_confirmed", NutritionProv: "ai_estimate",
			AIEstGrams: f64(100), AIEst: &models.Nutrients{Kcal: f64(kcal)}, Nutrients: models.Nutrients{Kcal: f64(kcal)}}
	}
	accMeal(t, r, "2026-11-16", "camera", item(200))
	afterDayTotalsRead = func() { accMeal(t, r, "2026-11-16", "camera", item(300)) }
	defer func() { afterDayTotalsRead = nil }()
	got, err := r.NutritionDayTotals("2026-11-16", "2026-11-16")
	if err != nil {
		t.Fatal(err)
	}
	d := got["2026-11-16"]
	if d.Items != 1 || *d.Sum.Kcal != 200 || d.Estimated.Items != 1 || *d.Estimated.Kcal != 200 {
		t.Fatalf("total and estimate disagree: %+v", d)
	}
	afterDayTotalsRead = nil
	again, _ := r.NutritionDayTotals("2026-11-16", "2026-11-16")
	if a := again["2026-11-16"]; a.Items != 2 || *a.Estimated.Kcal != 500 {
		t.Fatalf("second read: %+v", a)
	}
}
