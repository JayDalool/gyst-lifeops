package services

import (
	"database/sql"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func healthFloat(v float64) *float64 { return &v }
func healthInt(v int64) *int64       { return &v }
func cycleFixture() models.HealthCycle {
	return models.HealthCycle{Title: "Autumn", PrimaryGoal: "Consistency", StartDate: "2026-09-10", EndDate: "2026-12-03", DailyCalorieTarget: 2300, ProteinTargetG: 180}
}
func TestHealthCycleService(t *testing.T) {
	s := testService(t)
	c := cycleFixture()
	if err := s.CreateHealthCycle(&c); err != nil {
		t.Fatal(err)
	}
	if got, err := s.GetHealthCycle(c.ID); err != nil || got.Title != c.Title {
		t.Fatalf("get: %+v %v", got, err)
	}
	c.Notes = "Updated"
	if err := s.UpdateHealthCycle(&c); err != nil {
		t.Fatal(err)
	}
	overlap := cycleFixture()
	overlap.StartDate = c.EndDate
	overlap.EndDate = "2027-01-01"
	if err := s.CreateHealthCycle(&overlap); err == nil {
		t.Fatal("inclusive overlap accepted")
	}
	next := cycleFixture()
	next.StartDate = "2026-12-04"
	next.EndDate = "2027-01-01"
	if err := s.CreateHealthCycle(&next); err != nil {
		t.Fatal(err)
	}
	next.StartDate = "2026-12-03"
	if err := s.UpdateHealthCycle(&next); err == nil {
		t.Fatal("overlap update accepted")
	}
	if got, err := s.GetHealthCycle(next.ID); err != nil || got.StartDate != "2026-12-04" {
		t.Fatalf("failed update mutated cycle: %+v %v", got, err)
	}
	for _, date := range []string{c.StartDate, c.EndDate} {
		if got, err := s.GetHealthCycleForDate(date); err != nil || got.ID != c.ID {
			t.Fatalf("date lookup: %+v %v", got, err)
		}
	}
	if _, err := s.GetHealthCycleForDate("2026-09-09"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if cycles, err := s.ListHealthCycles(); err != nil || len(cycles) != 2 {
		t.Fatalf("list: %v %v", cycles, err)
	}
}
func TestHealthCycleValidation(t *testing.T) {
	for name, change := range map[string]func(*models.HealthCycle){
		"equal dates":   func(c *models.HealthCycle) { c.EndDate = c.StartDate },
		"reverse dates": func(c *models.HealthCycle) { c.EndDate = "2026-01-01" },
		"invalid date":  func(c *models.HealthCycle) { c.StartDate = "2026-02-30" },
		"short date":    func(c *models.HealthCycle) { c.StartDate = "2026-9-10" },
		"blank title":   func(c *models.HealthCycle) { c.Title = " " },
		"blank goal":    func(c *models.HealthCycle) { c.PrimaryGoal = "" },
		"calories":      func(c *models.HealthCycle) { c.DailyCalorieTarget = -1 },
		"protein":       func(c *models.HealthCycle) { c.ProteinTargetG = -1 },
		"carbs":         func(c *models.HealthCycle) { c.CarbsTargetG = -1 },
		"fat":           func(c *models.HealthCycle) { c.FatTargetG = -1 },
		"fiber":         func(c *models.HealthCycle) { c.FiberTargetG = -1 },
		"deficit":       func(c *models.HealthCycle) { c.TargetDeficitKcal = -1 },
		"strength":      func(c *models.HealthCycle) { c.StrengthSessionsPerWeek = -1 },
		"cardio":        func(c *models.HealthCycle) { c.CardioSessionsPerWeek = -1 },
		"steps":         func(c *models.HealthCycle) { c.StepTarget = -1 },
		"nan":           func(c *models.HealthCycle) { c.FatTargetG = math.NaN() },
		"infinity":      func(c *models.HealthCycle) { c.FiberTargetG = math.Inf(1) },
	} {
		t.Run(name, func(t *testing.T) {
			c := cycleFixture()
			change(&c)
			if err := ValidateHealthCycle(c); err == nil {
				t.Fatal("invalid cycle accepted")
			}
		})
	}
	// Arbitrary durations are supported, including leap day.
	c := cycleFixture()
	c.StartDate = "2028-02-28"
	c.EndDate = "2028-03-01"
	if err := ValidateHealthCycle(c); err != nil {
		t.Fatal(err)
	}
}
func TestConcurrentHealthCyclesDoNotOverlap(t *testing.T) {
	s := testService(t)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c := cycleFixture(); errs <- s.CreateHealthCycle(&c) }()
	}
	wg.Wait()
	close(errs)
	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful overlapping writes: %d", successes)
	}
}
func TestObservedHealthEnergy(t *testing.T) {
	snapshot := models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: healthFloat(130), ActiveEnergyKcal: healthFloat(211), RestingEnergyKcal: healthFloat(1318.596), StepsWatch: healthInt(1195), StepsPhone: healthInt(1302), ExerciseMinutes: healthFloat(42)}}
	cycle := cycleFixture()
	got := CalculateObservedHealthEnergy(&snapshot, &cycle)
	if math.Abs(*got.ObservedTotalBurn-1529.596) > 1e-9 || math.Abs(*got.ObservedEnergyBalance-(-1399.596)) > 1e-9 || *got.FoodRemaining != 2170 {
		t.Fatalf("observed: %+v", got)
	}
	snapshot.StepsWatch = healthInt(90000)
	snapshot.StepsPhone = healthInt(80000)
	snapshot.ExerciseMinutes = healthFloat(180)
	other := CalculateObservedHealthEnergy(&snapshot, &cycle)
	if *other.ObservedTotalBurn != *got.ObservedTotalBurn {
		t.Fatal("steps or workout minutes changed burn")
	}
	snapshot.DietaryCalories = healthFloat(2500)
	other = CalculateObservedHealthEnergy(&snapshot, &cycle)
	if *other.FoodRemaining != -200 {
		t.Fatal("over-target food was clamped")
	}
	for _, test := range []struct {
		name                string
		snapshot            *models.DailyHealthSnapshot
		cycle               *models.HealthCycle
		burn, balance, food bool
	}{
		{"none", nil, nil, false, false, false},
		{"no metrics", &models.DailyHealthSnapshot{}, &cycle, false, false, false},
		{"only active", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{ActiveEnergyKcal: healthFloat(0)}}, &cycle, false, false, false},
		{"only food", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: healthFloat(0)}}, &cycle, false, false, true},
		{"no cycle", &snapshot, nil, true, true, false},
		{"no food", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{ActiveEnergyKcal: healthFloat(0), RestingEnergyKcal: healthFloat(0)}}, &cycle, true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := CalculateObservedHealthEnergy(test.snapshot, test.cycle)
			if (got.ObservedTotalBurn != nil) != test.burn || (got.ObservedEnergyBalance != nil) != test.balance || (got.FoodRemaining != nil) != test.food {
				t.Fatalf("availability: %+v", got)
			}
		})
	}
}
func TestHealthSyncAndLocalToday(t *testing.T) {
	s := testService(t)
	s.Now = func() time.Time { return time.Date(2026, 9, 11, 1, 0, 0, 0, time.UTC) } // Sep 10 in Winnipeg.
	input := models.HealthSyncRequest{Date: "2026-09-10", Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{DietaryCalories: healthFloat(130)}}
	if _, err := s.SyncHealth(input); err != nil {
		t.Fatal(err)
	}
	input.HealthMetrics = models.HealthMetrics{ProteinG: healthFloat(0)}
	if _, err := s.SyncHealth(input); err != nil {
		t.Fatal(err)
	}
	today, err := s.HealthToday()
	if err != nil || today.Date != input.Date || *today.Snapshot.DietaryCalories != 130 || *today.Snapshot.ProteinG != 0 || today.HealthCycle != nil || today.Observed.ObservedTotalBurn != nil {
		t.Fatalf("today: %+v %v", today, err)
	}
	state, err := s.GetHealthSyncState("apple_shortcuts")
	if err != nil || state.LastSyncedAt != "2026-09-11T01:00:00Z" {
		t.Fatalf("state: %+v %v", state, err)
	}
	for _, input := range []models.HealthSyncRequest{
		{Date: "bad", Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{ProteinG: healthFloat(1)}},
		{Date: "2026-09-10", Source: "unknown", HealthMetrics: models.HealthMetrics{ProteinG: healthFloat(1)}},
		{Date: "2026-09-10", Source: "apple_shortcuts"},
		{Date: "2026-09-10", Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{ProteinG: healthFloat(-1)}},
		{Date: "2026-09-10", Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{StepsPhone: healthInt(-1)}},
	} {
		if _, err := s.SyncHealth(input); err == nil {
			t.Fatalf("accepted: %+v", input)
		}
	}
}
func TestHealthWeightService(t *testing.T) {
	s := testService(t)
	w := models.HealthWeightEntry{MeasuredAt: "2026-09-10T09:00:00-05:00", WeightKg: 80.2, Source: "manual"}
	if err := s.CreateHealthWeightEntry(&w); err != nil {
		t.Fatal(err)
	}
	if w.MeasuredAt != "2026-09-10T14:00:00.000000000Z" {
		t.Fatal(w.MeasuredAt)
	}
	rows, err := s.ListHealthWeightEntries()
	if err != nil || len(rows) != 1 || rows[0].WeightKg != 80.2 {
		t.Fatalf("weights: %+v %v", rows, err)
	}
	for _, value := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		w.WeightKg = value
		if err := s.CreateHealthWeightEntry(&w); err == nil {
			t.Fatal("invalid weight accepted")
		}
	}
	w.WeightKg = 80
	w.MeasuredAt = "2026-09-10"
	if err := s.CreateHealthWeightEntry(&w); err == nil {
		t.Fatal("date without timestamp accepted")
	}
}
