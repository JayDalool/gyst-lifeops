package handlers

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// Health Today redesign (#70).

func TestHealthNudgeRules(t *testing.T) {
	steps := func(v int64) *int64 { return &v }
	mins := func(v float64) *float64 { return &v }
	base := nudgeInput{Hour: 15, HasSnapshot: true, Steps: steps(4000), StepTarget: 10000, Exercise: mins(10)}
	with := func(f func(*nudgeInput)) nudgeInput { in := base; f(&in); return in }
	for _, tc := range []struct {
		name  string
		in    nudgeInput
		kind  string // "" = no nudge
		words string
	}{
		{"steps behind", base, "move", "6,000 steps to your 10,000 goal. A 30–45 min walk"},
		{"nearly there", with(func(i *nudgeInput) { i.Steps = steps(8500) }), "move", "1,500 steps to your 10,000 goal. A short walk"},
		{"no snapshot", with(func(i *nudgeInput) { i.HasSnapshot = false }), "", ""},
		{"stale data", with(func(i *nudgeInput) { i.Stale = true }), "", ""},
		{"morning", with(func(i *nudgeInput) { i.Hour = 10 }), "", ""},
		{"late night", with(func(i *nudgeInput) { i.Hour = 21 }), "", ""},
		{"trained 30 min", with(func(i *nudgeInput) { i.Exercise = mins(30) }), "", ""},
		{"step goal reached", with(func(i *nudgeInput) { i.Steps = steps(10000) }), "", ""},
		{"strength today", with(func(i *nudgeInput) { i.Workout = "Leg Day"; i.Exercise = mins(55) }), "recovery", "Nice work on Leg Day"},
		{"no step target", with(func(i *nudgeInput) { i.StepTarget = 0 }), "move", "A 30–45 min walk"},
		{"no steps, no exercise", with(func(i *nudgeInput) { i.Steps, i.Exercise = nil, nil }), "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := buildHealthNudge(tc.in)
			if tc.kind == "" {
				if n != nil {
					t.Fatalf("want no nudge, got %+v", *n)
				}
				return
			}
			if n == nil || n.Kind != tc.kind || !strings.Contains(n.Title+" "+n.Body, tc.words) {
				t.Fatalf("got %+v, want %s with %q", n, tc.kind, tc.words)
			}
			// Never a calorie target, never "burn off" food.
			text := strings.ToLower(n.Title + " " + n.Body)
			for _, banned := range []string{"kcal", "calorie", "burn", "earn", "deficit"} {
				if strings.Contains(text, banned) {
					t.Errorf("nudge mentions %q: %s", banned, text)
				}
			}
		})
	}
}

func TestCalorieStateUsesDisplayedNumbers(t *testing.T) {
	for _, tc := range []struct {
		name      string
		snapshot  *models.DailyHealthSnapshot
		target    float64
		key, word string
	}{
		{"waiting", nil, 2000, "idle", "Waiting for sync"},
		{"no food", &models.DailyHealthSnapshot{}, 2000, "idle", "No food logged"},
		{"no target", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(900)}}, 0, "idle", "No target set"},
		{"within", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(1500)}}, 2000, "ok", "Within budget"},
		{"last tenth", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(1800)}}, 2000, "watch", "Close to budget"},
		{"exactly on", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(2000.4)}}, 2000, "watch", "Close to budget"},
		{"over", &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(2250)}}, 2000, "over", "Over budget"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cycle := viewCycle()
			cycle.DailyCalorieTarget = tc.target
			view := buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10", Snapshot: tc.snapshot, HealthCycle: &cycle}, viewNow())
			if view.CalState.Key != tc.key || view.CalState.Label != tc.word {
				t.Fatalf("state %+v, want %s %q", view.CalState, tc.key, tc.word)
			}
			body := renderBody(t, "health.html", HealthData{Page: "health", Today: view})
			if !strings.Contains(body, `class="hv3-cstate `+tc.key+`"`) || !strings.Contains(body, tc.word) {
				t.Errorf("state pill not rendered with words for %s", tc.key)
			}
		})
	}
}

// The shown parts always add up: burned is resting + active as displayed,
// balance is eaten − burned as displayed, budget is target − eaten as
// displayed. Same formulas; rounding happens before, not after (#70).
func TestTodayNumbersAddUpOnScreen(t *testing.T) {
	cycle := viewCycle()
	cycle.DailyCalorieTarget = 2000
	snapshot := models.DailyHealthSnapshot{Date: "2026-09-10", HealthMetrics: models.HealthMetrics{
		DietaryCalories: viewFloat(1500.5), RestingEnergyKcal: viewFloat(1850.6), ActiveEnergyKcal: viewFloat(420.6),
	}}
	today := models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snapshot, HealthCycle: &cycle}
	today.Observed = services.CalculateObservedHealthEnergy(&snapshot, &cycle)
	view := buildHealthTodayView(today, viewNow())
	if view.RestingEnergy.Value != "1,851" || view.ActiveEnergy.Value != "421" || view.ObservedBurn.Value != "2,272" {
		t.Fatalf("burn %s + %s = %s", view.RestingEnergy.Value, view.ActiveEnergy.Value, view.ObservedBurn.Value)
	}
	if view.Calories.Consumed.Value != "1,501" || view.ObservedEnergy.Value != "−771" {
		t.Fatalf("balance %s − %s = %s", view.Calories.Consumed.Value, view.ObservedBurn.Value, view.ObservedEnergy.Value)
	}
	if view.BudgetAbs != "499" {
		t.Fatalf("budget left %s, want 2,000 − 1,501 = 499", view.BudgetAbs)
	}
}

func TestTodayWorkoutTitleUsesLocalDay(t *testing.T) {
	loc, _ := time.LoadLocation("America/Winnipeg")
	entries := []models.TrainingEntry{
		{Kind: "routine", Title: "Push Day"},
		{Kind: "workout", Title: "Yesterday", StartedAt: time.Date(2026, 10, 4, 22, 0, 0, 0, time.UTC)},
		// 03:10 UTC on Oct 6 is 22:10 on Oct 5 in Winnipeg.
		{Kind: "workout", Title: "Leg Day", StartedAt: time.Date(2026, 10, 6, 3, 10, 0, 0, time.UTC)},
	}
	if got := todayWorkoutTitle(entries, "2026-10-05", loc); got != "Leg Day" {
		t.Fatalf("got %q", got)
	}
	if got := todayWorkoutTitle(entries, "2026-10-06", loc); got != "" {
		t.Fatalf("Oct 6 has no local workout, got %q", got)
	}
}

func TestTodayPageRedesign(t *testing.T) {
	h := testHandler(t)
	loc := h.Service.Location()
	day := time.Date(2026, 10, 7, 15, 0, 0, 0, loc)
	h.Service.Clock.Now = func() time.Time { return day }
	cycle := models.HealthCycle{Title: "Cut", PrimaryGoal: "Weight loss", StartDate: "2026-10-01", EndDate: "2027-03-31", DailyCalorieTarget: 2250,
		ProteinTargetG: 180, CarbsTargetG: 225, FatTargetG: 70, FiberTargetG: 35, TargetDeficitKcal: 750, StepTarget: 10000}
	if err := h.Service.CreateHealthCycle(&cycle); err != nil {
		t.Fatal(err)
	}
	snap := models.DailyHealthSnapshot{Date: "2026-10-07", LastSyncedAt: day.Add(-20 * time.Minute).UTC().Format(time.RFC3339Nano), HealthMetrics: models.HealthMetrics{
		DietaryCalories: viewFloat(2400), ProteinG: viewFloat(185), CarbsG: viewFloat(240), FatG: viewFloat(60), FiberG: viewFloat(12),
		StepsWatch: viewInt(4200), StepsPhone: viewInt(3900), ActiveEnergyKcal: viewFloat(300), RestingEnergyKcal: viewFloat(1700), ExerciseMinutes: viewFloat(12),
	}}
	if err := h.Service.Repo.UpsertDailyHealthSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Creatine", "Water"} {
		if _, err := h.Service.AddHealthChecklistItem(name, "habit"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.Service.AddHealthChecklistItem("Protein target", "protein"); err != nil {
		t.Fatal(err)
	}

	body := cfgGet(t, h, "/health")
	for _, want := range []string{
		"<h1>Today</h1>", "Calories today", `class="hv3-cstate over"`, "Over budget", "150</strong><span>kcal over budget",
		"Eaten", "of 2,250 target", "Burned so far", "2,000", "1,700 resting + 300 active", "Balance so far", "&#43;400", "Observed, not final",
		"A little movement would help", "5,800 steps to your 10,000 goal",
		`hv3-mac protein`, `hv3-mac carbs`, `hv3-mac fat`, `hv3-mac fiber`, "15 g over", "Target met",
		`<p class="hv3-tile-v">4,200</p>`, "of 10,000 · Apple Watch", "iPhone 3,900 · counted separately", "12 min exercise",
		"Daily checklist", "1 of 3 complete", `aria-label="Creatine: not done, tap to mark done"`, "Measured from food logged, target met",
		"Apple Health · updated 20 min ago",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Today is missing %q", want)
		}
	}
	for _, banned := range []string{"8,100", "net left", "to reach goal", "calories to go", "burn it off", "~400"} {
		if strings.Contains(body, banned) {
			t.Errorf("Today shows %q", banned)
		}
	}

	// A Hevy workout today swaps the walk nudge for a recovery note.
	if err := h.Service.Repo.SaveTraining("hevy", []models.TrainingEntry{{ID: "w1", Kind: "workout", Title: "Leg Day", StartedAt: day.Add(-3 * time.Hour)}}, day); err != nil {
		t.Fatal(err)
	}
	body = cfgGet(t, h, "/health")
	if !strings.Contains(body, "Nice work on Leg Day") || strings.Contains(body, "A little movement would help") {
		t.Error("recovery note should replace the movement nudge after a workout")
	}

	// Late evening: no nudge at all.
	h.Service.Clock.Now = func() time.Time { return day.Add(7 * time.Hour) }
	if body = cfgGet(t, h, "/health"); strings.Contains(body, "hv3-nudge ") {
		t.Error("no nudge after 21:00")
	}
}

// An unreadable Hevy cache must not take down Today, More or Coach: the
// page renders without a nudge (#71 review).
func TestTodaySurvivesBadTrainingCache(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	h := New(services.New(repository.New(database)), "../../web/templates")
	loc := h.Service.Location()
	day := time.Date(2026, 10, 7, 15, 0, 0, 0, loc)
	h.Service.Clock.Now = func() time.Time { return day }
	if err := h.Service.Repo.UpsertDailyHealthSnapshot(models.DailyHealthSnapshot{Date: "2026-10-07", LastSyncedAt: day.UTC().Format(time.RFC3339Nano),
		HealthMetrics: models.HealthMetrics{StepsWatch: viewInt(1200), ExerciseMinutes: viewFloat(0)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO training_snapshots(provider,entries_json,synced_at) VALUES('hevy','{not json','')`); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Service.Repo.TrainingCache("hevy"); err == nil {
		t.Fatal("test setup: cache should fail to read")
	}
	for _, path := range []string{"/health", "/health/more", "/health/coach"} {
		body := cfgGet(t, h, path)
		if path == "/health" && (!strings.Contains(body, "Calories today") || strings.Contains(body, "hv3-nudge ")) {
			t.Errorf("Today should render without a nudge")
		}
	}
}
