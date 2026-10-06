package handlers

import (
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func gapView(protein, kcal *float64) nutritionData {
	cycle := viewCycle() // protein 180 g, 2,300 kcal
	snap := models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{ProteinG: protein, DietaryCalories: kcal}}
	return buildNutritionData(buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snap, HealthCycle: &cycle}, viewNow()))
}

func TestProteinGap(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	foods := []models.HealthFood{
		{ID: 1, Name: "Wrap", Kcal: f(160), ProteinG: f(12), Active: true},
		{ID: 2, Name: "Turkey", Kcal: f(110), ProteinG: f(24), Active: true},
		{ID: 3, Name: "Lentil soup", ProteinG: f(18), Active: true}, // kcal unknown: last
		{ID: 4, Name: "Rice", Kcal: f(200), ProteinG: f(4), Active: true},
		{ID: 5, Name: "Old shake", Kcal: f(150), ProteinG: f(30), Active: false},
		{ID: 6, Name: "Big meal", Kcal: f(2500), ProteinG: f(60), Active: true},
	}
	d := gapView(f(132), f(1500))
	buildProteinGap(&d, foods)
	if !d.ShowGap || d.GapLeft != "48" || len(d.GapFoods) != 3 {
		t.Fatalf("gap %+v", d)
	}
	// Protein per kcal: turkey 0.22, wrap 0.075, big meal 0.024; soup (unknown kcal) last.
	if d.GapFoods[0].Name != "Turkey" || d.GapFoods[1].Name != "Wrap" || d.GapFoods[2].Name != "Big meal" {
		t.Fatalf("order %+v", d.GapFoods)
	}
	if !d.GapFoods[0].Fits || !d.GapFoods[1].Fits || d.GapFoods[2].Fits {
		t.Fatal("fits-the-budget flag wrong (800 kcal left)")
	}
	body := renderBody(t, "nutrition.html", d)
	for _, want := range []string{"Close the protein gap", "48 g to go", "+24 g", "fits your budget", "over your budget left"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, banned := range []string{"Rice", "Old shake"} {
		if strings.Contains(body, banned) {
			t.Errorf("%s should not be suggested", banned)
		}
	}

	// Nothing to suggest without both values, when met, or with no foods.
	for name, d := range map[string]nutritionData{"no protein logged": gapView(nil, f(1500)), "target met": gapView(f(181), f(1500))} {
		buildProteinGap(&d, foods)
		if d.ShowGap {
			t.Errorf("%s: gap shown", name)
		}
	}
	empty := gapView(f(100), f(1500))
	buildProteinGap(&empty, nil)
	if !empty.ShowGap || empty.HasFoods || len(empty.GapFoods) != 0 || !strings.Contains(renderBody(t, "nutrition.html", empty), "Add foods you usually have") {
		t.Fatal("empty foods state wrong")
	}
	// Unknown budget (no calories logged): no fits claim either way.
	nb := gapView(f(100), nil)
	buildProteinGap(&nb, foods)
	if body := renderBody(t, "nutrition.html", nb); strings.Contains(body, "fits your budget") || strings.Contains(body, "over your budget left") {
		t.Fatal("budget claimed without calorie data")
	}
}
