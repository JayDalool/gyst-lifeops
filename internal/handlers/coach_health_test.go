package handlers

import (
	"net/http/httptest"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// The real adapter behind the Secretary's brief (Issue #236): null stays null,
// steps are one count with its source, no step-derived calories, and the food
// budget and the observed balance are separate (RULES 3.3-3.5). Fictional values.

func briefOf(t *testing.T, h *Handler) coachHealth {
	t.Helper()
	data, ok := h.healthBase(httptest.NewRecorder())
	if !ok {
		t.Fatal("healthBase")
	}
	return coachHealth{h: h, data: data, now: h.Service.Now(), loc: h.Service.Location()}
}

func TestCoachHealthBriefSeeded(t *testing.T) {
	h, closeDB := goldenHandler(t)
	defer closeDB()
	goldenSeed(t, h)
	b, err := briefOf(t, h).BriefHealth()
	if err != nil {
		t.Fatal(err)
	}
	str := func(p *string) string {
		if p == nil {
			return "<nil>"
		}
		return *p
	}
	if str(b.FoodEatenKcal) != "1,450" || str(b.FoodTargetKcal) != "2,300" || str(b.FoodBudgetLeftKcal) != "850" || b.FoodOverBudgetByKcal != nil {
		t.Fatalf("food: %s %s left %s over %s", str(b.FoodEatenKcal), str(b.FoodTargetKcal), str(b.FoodBudgetLeftKcal), str(b.FoodOverBudgetByKcal))
	}
	// Observed burn is resting plus active only (1,100 + 310); the balance is
	// its own number, not the food budget.
	if str(b.BurnedSoFarKcal) != "1,410" || str(b.EnergyBalanceSoFarKcal) != "+40" {
		t.Fatalf("burn %s balance %s", str(b.BurnedSoFarKcal), str(b.EnergyBalanceSoFarKcal))
	}
	if str(b.EnergyBalanceSoFarKcal) == str(b.FoodBudgetLeftKcal) {
		t.Fatal("balance and food budget are one value")
	}
	if str(b.Steps) != "5,200" || b.StepsSource != "watch" {
		t.Fatalf("steps %s from %q", str(b.Steps), b.StepsSource)
	}
	if str(b.ProteinEatenG) != "96" || str(b.ProteinTargetG) != "180" || str(b.WaterLoggedL) != "1.5" || str(b.WaterTargetL) != "3.0" || b.WeightKg == nil {
		t.Fatalf("protein %s/%s water %s/%s", str(b.ProteinEatenG), str(b.ProteinTargetG), str(b.WaterLoggedL), str(b.WaterTargetL))
	}
	if !b.AsOf.Equal(h.Service.Now()) || b.Stale {
		t.Fatalf("as_of %v stale %v", b.AsOf, b.Stale)
	}
}

func TestCoachHealthBriefEmptyStaysNull(t *testing.T) {
	h, closeDB := goldenHandler(t)
	defer closeDB()
	b, err := briefOf(t, h).BriefHealth()
	if err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]*string{"eaten": b.FoodEatenKcal, "target": b.FoodTargetKcal, "left": b.FoodBudgetLeftKcal, "over": b.FoodOverBudgetByKcal,
		"protein": b.ProteinEatenG, "protein target": b.ProteinTargetG, "steps": b.Steps, "burn": b.BurnedSoFarKcal, "balance": b.EnergyBalanceSoFarKcal,
		"water": b.WaterLoggedL, "water target": b.WaterTargetL, "weight": b.WeightKg} {
		if p != nil {
			t.Errorf("%s = %q on an empty day, want nil (never 0)", name, *p)
		}
	}
	if b.StepsSource != "" || !b.AsOf.IsZero() {
		t.Fatalf("source %q as_of %v", b.StepsSource, b.AsOf)
	}
}

// Watch and phone counts overlap, so they are never added; the phone is only a
// fallback when there is no Watch count.
func TestCoachHealthBriefNeverSumsSteps(t *testing.T) {
	h, closeDB := goldenHandler(t)
	defer closeDB()
	date := h.Service.TodayString()
	sync := func(m models.HealthMetrics) {
		t.Helper()
		if _, err := h.Service.SyncHealth(models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: m}); err != nil {
			t.Fatal(err)
		}
	}
	sync(models.HealthMetrics{StepsPhone: viewInt(3000)})
	b, _ := briefOf(t, h).BriefHealth()
	if b.Steps == nil || *b.Steps != "3,000" || b.StepsSource != "phone" {
		t.Fatalf("phone fallback: %v %q", b.Steps, b.StepsSource)
	}
	sync(models.HealthMetrics{StepsWatch: viewInt(5200)})
	b, _ = briefOf(t, h).BriefHealth()
	if b.Steps == nil || *b.Steps != "5,200" || b.StepsSource != "watch" {
		t.Fatalf("both sources: %v %q (8,200 would be a sum)", b.Steps, b.StepsSource)
	}
	// Steps alone never produce an energy figure.
	if b.BurnedSoFarKcal != nil || b.EnergyBalanceSoFarKcal != nil {
		t.Fatalf("calories derived from steps: %v %v", b.BurnedSoFarKcal, b.EnergyBalanceSoFarKcal)
	}
}

func TestCoachHealthBriefOverBudgetAndStale(t *testing.T) {
	h, closeDB := goldenHandler(t)
	defer closeDB()
	date := h.Service.TodayString()
	cycle := models.HealthCycle{Title: "Plan", PrimaryGoal: "x", StartDate: "2026-10-01", EndDate: "2027-03-31", DailyCalorieTarget: 2000}
	if err := h.Service.CreateHealthCycle(&cycle); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Service.SyncHealth(models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(2200)}}); err != nil {
		t.Fatal(err)
	}
	now := h.Service.Now().Add(9 * time.Hour)
	h.Service.Clock.Now = func() time.Time { return now }
	b, err := briefOf(t, h).BriefHealth()
	if err != nil {
		t.Fatal(err)
	}
	if b.FoodOverBudgetByKcal == nil || *b.FoodOverBudgetByKcal != "200" || b.FoodBudgetLeftKcal != nil {
		t.Fatalf("over budget: %v left %v", b.FoodOverBudgetByKcal, b.FoodBudgetLeftKcal)
	}
	if !b.Stale {
		t.Fatal("a snapshot nine hours old is stale (HealthSyncStaleAfter is 8 h)")
	}
}

// GYST-164 c2: values, stale and as_of come from the one read healthBase made.
// Before, BriefHealth read the snapshot a second time for as_of, so a sync that
// landed between the two reads gave old values with a new as_of (RULES 3.2).
func TestCoachHealthBriefOneRead(t *testing.T) {
	h, closeDB := goldenHandler(t)
	defer closeDB()
	date := h.Service.TodayString()
	sync := func(steps int) {
		t.Helper()
		if _, err := h.Service.SyncHealth(models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{StepsWatch: viewInt(int64(steps))}}); err != nil {
			t.Fatal(err)
		}
	}
	setNow := func(at time.Time) { h.Service.Clock.Now = func() time.Time { return at } }
	t0 := h.Service.Now()
	sync(1000)

	// Fresh view, then a later sync lands before BriefHealth runs.
	old := briefOf(t, h)
	setNow(t0.Add(9 * time.Hour))
	sync(2000)
	b, err := old.BriefHealth()
	if err != nil {
		t.Fatal(err)
	}
	if b.Steps == nil || *b.Steps != "1,000" || !b.AsOf.Equal(t0) || b.Stale {
		t.Fatalf("old view: steps %v as_of %v (want %v) stale %v", b.Steps, b.AsOf, t0, b.Stale)
	}
	if want := old.now.Sub(b.AsOf) >= services.HealthSyncStaleAfter; b.Stale != want {
		t.Fatalf("stale %v disagrees with as_of age", b.Stale)
	}
	fresh, err := briefOf(t, h).BriefHealth()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Steps == nil || *fresh.Steps != "2,000" || !fresh.AsOf.Equal(t0.Add(9*time.Hour)) || fresh.Stale {
		t.Fatalf("new view: steps %v as_of %v stale %v", fresh.Steps, fresh.AsOf, fresh.Stale)
	}
}

func TestCoachHealthBriefOneReadStaleView(t *testing.T) {
	h, closeDB := goldenHandler(t)
	defer closeDB()
	date := h.Service.TodayString()
	t0 := h.Service.Now()
	if _, err := h.Service.SyncHealth(models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{StepsWatch: viewInt(1000)}}); err != nil {
		t.Fatal(err)
	}
	later := t0.Add(9 * time.Hour)
	h.Service.Clock.Now = func() time.Time { return later }
	stale := briefOf(t, h) // built while the copy is 9 h old
	if _, err := h.Service.SyncHealth(models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{StepsWatch: viewInt(2000)}}); err != nil {
		t.Fatal(err)
	}
	b, err := stale.BriefHealth()
	if err != nil {
		t.Fatal(err)
	}
	if !b.Stale || !b.AsOf.Equal(t0) {
		t.Fatalf("stale view: stale %v as_of %v (want true, %v)", b.Stale, b.AsOf, t0)
	}
	if want := stale.now.Sub(b.AsOf) >= services.HealthSyncStaleAfter; b.Stale != want {
		t.Fatalf("stale %v disagrees with as_of age", b.Stale)
	}
}
