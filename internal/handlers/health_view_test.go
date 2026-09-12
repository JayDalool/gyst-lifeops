package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

func viewFloat(value float64) *float64 { return &value }
func viewInt(value int64) *int64       { return &value }

func viewNow() time.Time { return time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC) }

func viewCycle() models.HealthCycle {
	return models.HealthCycle{
		Title: "Autumn", DailyCalorieTarget: 2300, ProteinTargetG: 180,
		CarbsTargetG: 220, FatTargetG: 70, FiberTargetG: 30, TargetDeficitKcal: 400,
	}
}

// TestAbsentMetricsNeverRenderAsZero is the rule the whole view model exists
// for. NULL means "not observed" and 0 means "observed zero"; a dashboard that
// shows 0 for an unsynced morning is stating something false about the day.
func TestAbsentMetricsNeverRenderAsZero(t *testing.T) {
	view := buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10"}, viewNow())

	if view.HasSnapshot {
		t.Error("HasSnapshot true with no snapshot")
	}
	for _, metric := range []struct {
		name string
		view HealthMetricView
	}{
		{"FoodRemaining", view.FoodRemaining},
		{"ObservedBurn", view.ObservedBurn},
		{"ObservedEnergy", view.ObservedEnergy},
		{"StepsWatch", view.StepsWatch},
		{"StepsPhone", view.StepsPhone},
		{"ActiveEnergy", view.ActiveEnergy},
		{"RestingEnergy", view.RestingEnergy},
		{"ExerciseMinutes", view.ExerciseMinutes},
	} {
		if metric.view.Present {
			t.Errorf("%s reports Present with no observation", metric.name)
		}
		if metric.view.Value != "" {
			t.Errorf("%s = %q with no observation, want empty", metric.name, metric.view.Value)
		}
	}
	for _, macro := range view.Macros {
		if macro.Consumed.Present || macro.HasBar {
			t.Errorf("%s shows a value or bar with nothing logged", macro.Label)
		}
	}
}

// TestObservedZeroIsDistinctFromAbsent is the other half: a real zero must be
// shown as a zero.
func TestObservedZeroIsDistinctFromAbsent(t *testing.T) {
	cycle := viewCycle()
	view := buildHealthTodayView(models.HealthTodayResponse{
		Date:        "2026-09-10",
		HealthCycle: &cycle,
		Snapshot: &models.DailyHealthSnapshot{
			Date:          "2026-09-10",
			HealthMetrics: models.HealthMetrics{ProteinG: viewFloat(0), StepsWatch: viewInt(0)},
		},
	}, viewNow())

	protein := view.Macros[1]
	if protein.Label != "Protein" {
		t.Fatalf("macro order changed: %s", protein.Label)
	}
	if !protein.Consumed.Present || protein.Consumed.Value != "0" {
		t.Errorf("observed zero protein = %+v, want a present 0", protein.Consumed)
	}
	if !view.StepsWatch.Present || view.StepsWatch.Value != "0" {
		t.Errorf("observed zero steps = %+v, want a present 0", view.StepsWatch)
	}
}

// TestFoodRemainingIsNotSynthesised pins the decision taken in review: with no
// nutrition observation, the remaining figure is unknown. Showing the full
// target would state that nothing has been eaten, which is not what an
// unsynced MyFitnessPal means.
func TestFoodRemainingIsNotSynthesised(t *testing.T) {
	cycle := viewCycle()
	view := buildHealthTodayView(models.HealthTodayResponse{
		Date:        "2026-09-10",
		HealthCycle: &cycle,
		Snapshot: &models.DailyHealthSnapshot{
			Date:          "2026-09-10",
			HealthMetrics: models.HealthMetrics{StepsWatch: viewInt(2000)},
		},
	}, viewNow())

	if view.FoodRemaining.Present {
		t.Errorf("FoodRemaining = %+v with no dietary observation, want absent", view.FoodRemaining)
	}
	// The target is still shown, so the page can say what the goal is without
	// claiming how much of it is left.
	if !view.CalorieTarget.Present || view.CalorieTarget.Value != "2,300" {
		t.Errorf("CalorieTarget = %+v, want a present 2,300", view.CalorieTarget)
	}
}

func TestMacrosAndObservedEnergy(t *testing.T) {
	cycle := viewCycle()
	snapshot := models.DailyHealthSnapshot{
		Date: "2026-09-10",
		HealthMetrics: models.HealthMetrics{
			DietaryCalories:   viewFloat(1840),
			ProteinG:          viewFloat(142),
			ActiveEnergyKcal:  viewFloat(610),
			RestingEnergyKcal: viewFloat(1800),
		},
	}
	today := models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snapshot, HealthCycle: &cycle}
	today.Observed = services.CalculateObservedHealthEnergy(&snapshot, &cycle)

	view := buildHealthTodayView(today, viewNow())

	calories := view.Macros[0]
	if calories.Consumed.Value != "1,840" || calories.Target.Value != "2,300" {
		t.Errorf("calories = %+v", calories)
	}
	if !calories.HasBar || calories.Percent != 80 {
		t.Errorf("calorie bar = %d%%, want 80", calories.Percent)
	}
	if calories.Remaining.Value != "460" {
		t.Errorf("calories remaining = %q, want 460", calories.Remaining.Value)
	}

	// 1840 - (1800 + 610) = -570
	if view.ObservedBurn.Value != "2,410" {
		t.Errorf("observed burn = %q, want 2,410", view.ObservedBurn.Value)
	}
	if view.ObservedEnergy.Value != "−570" {
		t.Errorf("observed balance = %q, want −570", view.ObservedEnergy.Value)
	}
	if view.FoodRemaining.Value != "460" {
		t.Errorf("food remaining = %q, want 460", view.FoodRemaining.Value)
	}
}

// TestTargetEnergyIsShownWithTheObservedSign covers the schema asymmetry: the
// cycle stores a deficit as a positive magnitude while the observed balance is
// negative when under. The two must be comparable on screen.
func TestTargetEnergyIsShownWithTheObservedSign(t *testing.T) {
	cycle := viewCycle() // TargetDeficitKcal: 400
	view := buildHealthTodayView(models.HealthTodayResponse{
		Date: "2026-09-10", HealthCycle: &cycle,
	}, viewNow())

	if !view.TargetEnergy.Present || view.TargetEnergy.Value != "−400" {
		t.Errorf("target energy = %+v, want −400 to match the observed sign", view.TargetEnergy)
	}
}

// TestNoProjectionIsRendered guards rule 2: no projection is implemented, so
// the view must not carry one.
func TestNoProjectionIsRendered(t *testing.T) {
	cycle := viewCycle()
	snapshot := models.DailyHealthSnapshot{
		Date: "2026-09-10",
		HealthMetrics: models.HealthMetrics{
			DietaryCalories: viewFloat(1840), RestingEnergyKcal: viewFloat(900),
			ActiveEnergyKcal: viewFloat(300),
		},
	}
	today := models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snapshot, HealthCycle: &cycle}
	today.Observed = services.CalculateObservedHealthEnergy(&snapshot, &cycle)

	view := buildHealthTodayView(today, viewNow())
	// Observed burn is exactly resting + active, with nothing extrapolated to
	// the end of the day.
	if view.ObservedBurn.Value != "1,200" {
		t.Errorf("observed burn = %q, want exactly resting+active (1,200)", view.ObservedBurn.Value)
	}
}

func TestStalenessIsFlagged(t *testing.T) {
	now := viewNow()
	for _, testCase := range []struct {
		name      string
		synced    time.Time
		wantStale bool
	}{
		{"fresh", now.Add(-10 * time.Minute), false},
		{"a few hours", now.Add(-3 * time.Hour), false},
		{"beyond the window", now.Add(-9 * time.Hour), true},
		{"yesterday", now.Add(-30 * time.Hour), true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			snapshot := models.DailyHealthSnapshot{
				Date:          "2026-09-10",
				LastSyncedAt:  testCase.synced.Format(time.RFC3339Nano),
				HealthMetrics: models.HealthMetrics{StepsWatch: viewInt(100)},
			}
			view := buildHealthTodayView(models.HealthTodayResponse{
				Date: "2026-09-10", Snapshot: &snapshot,
			}, now)

			if !view.HasLastSynced {
				t.Fatal("last synced not parsed")
			}
			if view.LastSyncedStale != testCase.wantStale {
				t.Errorf("stale = %v, want %v (synced %s)", view.LastSyncedStale, testCase.wantStale, view.LastSynced)
			}
		})
	}
}

func TestFormatThousands(t *testing.T) {
	for _, testCase := range []struct {
		value float64
		want  string
	}{
		{0, "0"}, {7, "7"}, {999, "999"}, {1000, "1,000"},
		{7421, "7,421"}, {1318.6, "1,319"}, {-570, "−570"}, {-1010, "−1,010"},
	} {
		if got := formatThousands(testCase.value); got != testCase.want {
			t.Errorf("formatThousands(%v) = %q, want %q", testCase.value, got, testCase.want)
		}
	}
}

// TestHealthPageRendersWithoutData is the end-to-end guard: the most common
// state today is an empty one, and it must render rather than blow up on a nil
// snapshot or print a misleading zero.
func TestHealthPageRendersWithoutData(t *testing.T) {
	h := testHandler(t)
	view := buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10"}, viewNow())

	recorder := httptest.NewRecorder()
	h.render(recorder, "health.html", HealthData{Page: "health", Today: view})

	if recorder.Code != 200 {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if strings.Contains(body, "executing \"") {
		t.Fatalf("template error: %s", body)
	}
	if !strings.Contains(body, "No data for 2026-09-10") {
		t.Error("empty state is not shown")
	}
	// The old demonstration numbers must be gone for good.
	for _, ghost := range []string{"1,840", "2,410", "7,421", "Upper body", "Demo"} {
		if strings.Contains(body, ghost) {
			t.Errorf("demo value %q still rendered with no data", ghost)
		}
	}
}

// TestHealthPageRendersWithData is the populated counterpart: real numbers
// reach the page, and the two energy concepts stay distinct on screen.
func TestHealthPageRendersWithData(t *testing.T) {
	h := testHandler(t)
	cycle := viewCycle()
	snapshot := models.DailyHealthSnapshot{
		Date:         "2026-09-10",
		LastSyncedAt: viewNow().Add(-15 * time.Minute).Format(time.RFC3339Nano),
		HealthMetrics: models.HealthMetrics{
			DietaryCalories: viewFloat(1840), ProteinG: viewFloat(142),
			ActiveEnergyKcal: viewFloat(610), RestingEnergyKcal: viewFloat(1800),
			StepsWatch: viewInt(7421), StepsPhone: viewInt(6210),
		},
	}
	today := models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snapshot, HealthCycle: &cycle}
	today.Observed = services.CalculateObservedHealthEnergy(&snapshot, &cycle)

	recorder := httptest.NewRecorder()
	h.render(recorder, "health.html", HealthData{Page: "health", Today: buildHealthTodayView(today, viewNow())})

	if recorder.Code != 200 {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if strings.Contains(body, "executing \"") {
		t.Fatalf("template error: %s", body)
	}
	for _, want := range []string{
		"1,840", "2,300", "460", // consumed, target, food remaining
		"2,410", // observed burn
		"−570",  // observed balance
		"−400",  // target balance, signed to match
		"7,421", // watch steps
		"6,210", // phone steps, listed separately
		"15 min ago",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered page is missing %q", want)
		}
	}
	// The two energy concepts must both be present and labelled distinctly.
	if !strings.Contains(body, "food calories remaining") || !strings.Contains(body, "Observed energy balance") {
		t.Error("food budget and observed balance are not both labelled")
	}
	// Steps are never summed: 7421+6210=13631 must not appear.
	if strings.Contains(body, "13,631") {
		t.Error("watch and phone steps were summed")
	}
}
