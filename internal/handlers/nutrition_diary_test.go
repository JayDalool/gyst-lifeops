package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// Food diary views (Issue #207). Fictional values only.

func diaryHandlerWithSyncedDay(t *testing.T) (*Handler, string) {
	t.Helper()
	h := testHandler(t)
	date := h.Service.TodayString()
	cycle := models.HealthCycle{Title: "Autumn", PrimaryGoal: "Consistency", StartDate: date, EndDate: h.Service.Today().AddDate(0, 0, 60).Format("2006-01-02"), DailyCalorieTarget: 2000, ProteinTargetG: 150}
	if err := h.Service.CreateHealthCycle(&cycle); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Service.SyncHealth(models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{
		DietaryCalories: viewFloat(1800), ProteinG: viewFloat(90), StepsWatch: viewInt(900), ActiveEnergyKcal: viewFloat(300), RestingEnergyKcal: viewFloat(1700)}}); err != nil {
		t.Fatal(err)
	}
	return h, date
}

func addTestOats(t *testing.T, h *Handler, key string, protein *float64, grams float64) {
	t.Helper()
	food, err := h.Service.Repo.UpsertNutritionFoodVersion(context.Background(), models.NutritionFood{
		Provider: "usda_fdc", ProviderID: "oats-" + key, Name: "Test oats " + key, NormalizedName: "test oats " + key, Basis: "per_100g",
		Nutrients: models.Nutrients{Kcal: viewFloat(380), ProteinG: protein}, DataProvenance: "database",
	}, "2026-11-16T15:00:00.000000000Z")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Service.AddDiaryItem(context.Background(), services.DiaryActor{}, models.NewDiaryItem{
		IdempotencyKey: strings.Repeat(key, 32), MealSlot: "breakfast", FoodID: food.ID, Unit: "g", Amount: grams}); err != nil {
		t.Fatal(err)
	}
}

func TestNoDiaryRowsLeavesTodayAndNutritionOnAppleHealth(t *testing.T) {
	h, _ := diaryHandlerWithSyncedDay(t)
	today, err := h.Service.HealthToday()
	if err != nil {
		t.Fatal(err)
	}
	if today.Intake.Source != models.IntakeSourceAppleHealth {
		t.Fatalf("source = %q", today.Intake.Source)
	}
	view := buildHealthTodayView(today, h.Service.Now())
	if view.Calories.Consumed.Value != "1,800" || view.ReferenceKcal.Present || view.Calories.Partial {
		t.Fatalf("view changed without diary rows: %+v", view.Calories)
	}
	if *today.Observed.ObservedEnergyBalance != -200 || *today.Observed.FoodRemaining != 200 {
		t.Fatalf("observed: %+v", today.Observed)
	}
	for _, path := range []string{"/health/nutrition", "/health", "/today"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != 200 {
			t.Fatalf("GET %s: %d", path, r.Code)
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health/nutrition", nil))
	if !strings.Contains(r.Body.String(), "1,800") {
		t.Fatal("Nutrition must show the synced total when there is no diary")
	}
	c := coachToday(HealthData{Today: view})
	if c["intake_source"] != models.IntakeSourceAppleHealth || c["apple_health_kcal_not_counted"] != nil {
		t.Fatalf("coach: %v %v", c["intake_source"], c["apple_health_kcal_not_counted"])
	}
}

func TestGYSTDayCountsDiaryAndShowsAppleHealthAsReference(t *testing.T) {
	h, _ := diaryHandlerWithSyncedDay(t)
	addTestOats(t, h, "a", viewFloat(13), 40) // 152 kcal, 5.2 g protein
	today, err := h.Service.HealthToday()
	if err != nil {
		t.Fatal(err)
	}
	view := buildHealthTodayView(today, h.Service.Now())
	if view.IntakeSource != models.IntakeSourceGYST || view.Calories.Consumed.Value != "152" {
		t.Fatalf("Today must count the diary only: source %q, eaten %q", view.IntakeSource, view.Calories.Consumed.Value)
	}
	if !view.ReferenceKcal.Present || view.ReferenceKcal.Value != "1,800" {
		t.Fatalf("Apple Health reference = %+v", view.ReferenceKcal)
	}
	// Energy balance and food remaining use the diary; burn is still from the sync.
	if view.ObservedBurn.Value != "2,000" || view.ObservedEnergy.Value != "−1,848" || view.FoodRemaining.Value != "1,848" {
		t.Fatalf("energy: burn %q balance %q remaining %q", view.ObservedBurn.Value, view.ObservedEnergy.Value, view.FoodRemaining.Value)
	}
	if view.Calories.Partial || view.Macros[1].Partial {
		t.Fatal("complete data must not be marked partial")
	}

	// The Nutrition page shows the diary total, not the synced one and not their sum.
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health/nutrition", nil))
	body := r.Body.String()
	if r.Code != 200 || !strings.Contains(body, "152") || strings.Contains(body, "1,952") || strings.Contains(body, "1,800<small>") {
		t.Fatalf("nutrition page: %d", r.Code)
	}

	// Coach: one source, the reference labelled not counted, and the rule in the prompt.
	c := coachToday(HealthData{Today: view})
	if c["intake_source"] != models.IntakeSourceGYST || c["apple_health_kcal_not_counted"] != "1,800" {
		t.Fatalf("coach intake: %v %v", c["intake_source"], c["apple_health_kcal_not_counted"])
	}
	if !strings.Contains(coachSystemPrompt(nil, c), "never add it to the eaten values") {
		t.Fatal("system prompt lost the never-add rule")
	}
}

func TestGYSTDayWithMissingValuesIsPartialNotZero(t *testing.T) {
	h, _ := diaryHandlerWithSyncedDay(t)
	addTestOats(t, h, "a", nil, 40) // kcal known, protein unknown
	today, _ := h.Service.HealthToday()
	view := buildHealthTodayView(today, h.Service.Now())
	protein := view.Macros[1]
	if protein.Consumed.Present || !protein.Partial {
		t.Fatalf("all-unknown protein must be absent and flagged, not 0: %+v", protein)
	}
	if view.Calories.Partial {
		t.Fatal("kcal is complete")
	}
	c := coachToday(HealthData{Today: view})
	macros := c["macros"].([]map[string]any)
	if macros[0]["name"] != "Protein" || macros[0]["partial"] != true || macros[0]["eaten_g"] != nil {
		t.Fatalf("coach macro: %v", macros[0])
	}
	// A second item without kcal makes the calorie total a lower bound.
	food, err := h.Service.Repo.UpsertNutritionFoodVersion(context.Background(), models.NutritionFood{
		Provider: "usda_fdc", ProviderID: "water-1", Name: "Test mystery", NormalizedName: "test mystery", Basis: "per_100g", DataProvenance: "ai_estimate"}, "2026-11-16T15:00:00.000000000Z")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Service.AddDiaryItem(context.Background(), services.DiaryActor{}, models.NewDiaryItem{IdempotencyKey: strings.Repeat("b", 32), MealSlot: "lunch", FoodID: food.ID, Unit: "g", Amount: 10}); err != nil {
		t.Fatal(err)
	}
	today, _ = h.Service.HealthToday()
	view = buildHealthTodayView(today, h.Service.Now())
	if !view.Calories.Partial || view.Calories.Consumed.Value != "152" {
		t.Fatalf("partial calories: %+v", view.Calories)
	}
}

func TestTodayCardHasFoodOnAGYSTDayWithoutSnapshot(t *testing.T) {
	h := testHandler(t)
	date := h.Service.TodayString()
	cycle := models.HealthCycle{Title: "Autumn", PrimaryGoal: "Consistency", StartDate: date, EndDate: h.Service.Today().AddDate(0, 0, 60).Format("2006-01-02"), DailyCalorieTarget: 2000}
	if err := h.Service.CreateHealthCycle(&cycle); err != nil {
		t.Fatal(err)
	}
	addTestOats(t, h, "a", nil, 40)
	today, _ := h.Service.HealthToday()
	view := buildHealthTodayView(today, h.Service.Now())
	if view.HasSnapshot || view.CalState.Label == "Waiting for sync" || view.FoodRemaining.Value != "1,848" {
		t.Fatalf("a GYST day must not wait for a sync: %+v %q", view.CalState, view.FoodRemaining.Value)
	}
	snap := buildTodayHealthSnapshot(view, nil)
	if snap.Status.Kind() == "empty" || snap.Metrics[0].Value != "1,848" {
		t.Fatalf("Today card: %+v %+v", snap.Status, snap.Metrics[0])
	}
}

func TestNutritionWeekAppliesAppleHealthRulesOnlyToAppleHealthDays(t *testing.T) {
	day := func(d int) string { return time.Date(2026, 11, 10+d, 0, 0, 0, 0, time.UTC).Format("2006-01-02") } // 10..16
	// Early sync on Apple Health days is partial; a GYST day is judged by its own gaps.
	early := "2026-11-11T14:00:00Z" // 08:00 Winnipeg
	apple := func(date string, kcal float64, synced string) models.DayIntake {
		s := models.DailyHealthSnapshot{Date: date, HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(kcal)}, LastSyncedAt: synced}
		return services.ResolveDayIntake(date, nil, &s)
	}
	gyst := func(date string, kcal float64, missing int) models.DayIntake {
		s := models.DailyHealthSnapshot{Date: date, HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(9999)}, LastSyncedAt: early}
		return services.ResolveDayIntake(date, &models.NutritionDayTotals{Date: date, Items: 2, Sum: models.Nutrients{Kcal: viewFloat(kcal)}, Gaps: models.NutrientGaps{Kcal: missing}}, &s)
	}
	days := []models.DayIntake{
		apple(day(0), 2000, "2026-11-10T14:00:00Z"), // early sync: partial
		apple(day(1), 0, "2026-11-11T23:00:00Z"),    // recorded zero: not logged
		gyst(day(2), 1500, 0),                       // early sync but a GYST day: complete
		gyst(day(3), 1000, 1),                       // missing kcal: partial
		apple(day(4), 2200, "2026-11-14T23:00:00Z"), // complete
	}
	var data nutritionData
	buildNutritionWeekDays(&data, days, day(6), 2300, time.UTC)
	if data.WeekLogged != 2 || data.WeekPartial != 2 || data.WeekAvg != "1,850" {
		t.Fatalf("logged %d partial %d avg %q", data.WeekLogged, data.WeekPartial, data.WeekAvg)
	}
	byName := map[string]nutritionDay{}
	for _, d := range data.Week {
		byName[d.Name] = d
	}
	if g := byName["Fri Nov 13"]; !g.Partial || g.Value != "1,000" {
		t.Fatalf("gyst day with a missing kcal: %+v", g)
	}
	if g := byName["Thu Nov 12"]; g.Partial || g.Value != "1,500" {
		t.Fatalf("a GYST day must not use the synced 9,999 or its sync time: %+v", g)
	}
	if g := byName["Wed Nov 11"]; g.Present {
		t.Fatalf("recorded zero must read not logged: %+v", g)
	}
}
