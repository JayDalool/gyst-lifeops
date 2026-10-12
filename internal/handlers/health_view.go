package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// The Day view renders observations, never inventions. Three rules shape every
// type here:
//
//  1. A metric that was not observed is absent, not zero. NULL and 0 mean
//     different things all the way down the stack - "MyFitnessPal has not
//     synced" versus "you ate nothing" - and collapsing them at the last step
//     would throw away the distinction the schema exists to keep.
//  2. Nothing is projected. No projection is implemented, so the view has no
//     place to put one; the demo markup's "projected total burn" is removed
//     rather than reproduced with real-looking arithmetic.
//  3. Food budget and observed energy balance stay separate. They answer
//     different questions and happen to have similar magnitudes, which is
//     exactly why conflating them is easy and harmful.
type HealthMetricView struct {
	Value   string // preformatted; empty when absent
	Unit    string
	Present bool
}

// absentMetric is the honest rendering of a metric with no observation.
func absentMetric(unit string) HealthMetricView {
	return HealthMetricView{Unit: unit}
}

// HealthMacroView is one nutrition row. Consumed and Target are independent:
// a target with no intake logged is a normal morning, and must not read as
// "0 g consumed".
type HealthMacroView struct {
	Label        string
	Consumed     HealthMetricView
	Target       HealthMetricView
	Remaining    HealthMetricView
	Percent      int
	HasBar       bool // only when both sides are known
	ProgressAria string
	// State is "ok" when a minimum (protein, fiber) is met and "warn" when a
	// limit (calories, carbs, fat) is exceeded; empty otherwise.
	State string
	// Note is the short line under a macro: "58 g to go", "Target met", "12 g over".
	Note string
	// Over is decided from the numbers at displayed precision, so 2000.3 of
	// 2000 is not "over" (#54 review).
	Over bool
}

// HealthTodayView is everything the Day view renders.
type HealthTodayView struct {
	Date string

	HasSnapshot bool
	HasCycle    bool
	CycleTitle  string

	// LastSynced describes the whole row, not one metric: the snapshot carries
	// a single last_synced_at. Per-metric freshness needs the observation store
	// that is still to come, so the wording here stays deliberately coarse.
	LastSynced      string
	HasLastSynced   bool
	LastSyncedStale bool

	// Calories is also the first entry of Macros. It is named explicitly
	// because the energy card needs it on its own, and reaching in by index
	// would break the page outright on a zero-valued view - a template "index
	// out of range" aborts execution and silently drops everything after it.
	Calories HealthMacroView
	Macros   []HealthMacroView

	FoodRemaining  HealthMetricView
	CalorieTarget  HealthMetricView
	ObservedBurn   HealthMetricView
	ObservedEnergy HealthMetricView
	TargetEnergy   HealthMetricView

	StepTarget HealthMetricView
	// Steps is the one count shown large: Apple Watch when observed,
	// otherwise iPhone. StepsSource names it; StepsOther is the other source,
	// shown separately when both exist. The two are never added together.
	Steps           HealthMetricView
	StepsSource     string
	StepsOther      HealthMetricView
	OtherSource     string
	StepPercent     int // Steps against the target; 0 when either is absent
	CaloriePercent  int // for the ring: eaten against target, capped at 100
	CaloriesOver    bool
	BudgetAbs       string // FoodRemaining without its sign, for "120 kcal over"
	Nutrients       []HealthMacroView
	StepsWatch      HealthMetricView
	StepsPhone      HealthMetricView
	ActiveEnergy    HealthMetricView
	RestingEnergy   HealthMetricView
	ExerciseMinutes HealthMetricView

	// CalState is the calorie card's status (#70).
	CalState healthCalorieState
	// caloriesLeft is target − eaten at displayed precision; only meaningful
	// when Calories.HasBar.
	caloriesLeft float64
}

func formatThousands(value float64) string {
	rounded := strconv.FormatFloat(math.Round(value), 'f', 0, 64)
	if rounded == "-0" { // −0.3 rounds to zero; never show a signed zero
		rounded = "0"
	}
	negative := strings.HasPrefix(rounded, "-")
	rounded = strings.TrimPrefix(rounded, "-")

	var grouped strings.Builder
	for index, digit := range rounded {
		if index > 0 && (len(rounded)-index)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(digit)
	}
	if negative {
		// Minus sign, not a hyphen: these sit next to numerals.
		return "−" + grouped.String()
	}
	return grouped.String()
}

func metricFromFloat(value *float64, unit string) HealthMetricView {
	if value == nil {
		return absentMetric(unit)
	}
	return HealthMetricView{Value: formatThousands(*value), Unit: unit, Present: true}
}

func metricFromInt(value *int64, unit string) HealthMetricView {
	if value == nil {
		return absentMetric(unit)
	}
	return HealthMetricView{Value: formatThousands(float64(*value)), Unit: unit, Present: true}
}

// metricFromTarget treats a zero target as unset. A cycle row defaults every
// target to 0, so 0 means "not configured" rather than "aim for nothing".
func metricFromTarget(target float64, unit string) HealthMetricView {
	if target <= 0 {
		return absentMetric(unit)
	}
	return HealthMetricView{Value: formatThousands(target), Unit: unit, Present: true}
}

func buildMacro(label string, consumed *float64, target float64, unit string) HealthMacroView {
	macro := HealthMacroView{
		Label:    label,
		Consumed: metricFromFloat(consumed, unit),
		Target:   metricFromTarget(target, unit),
		Remaining: HealthMetricView{
			Unit: unit,
		},
	}
	if !macro.Consumed.Present || !macro.Target.Present {
		// Without both halves there is no meaningful bar or remainder. Showing
		// a full bar for an unlogged morning would read as "nothing left".
		macro.ProgressAria = label + ": no data yet"
		return macro
	}

	// Both sides are rounded first, so the remainder always matches the two
	// numbers on screen (2,000 − 1,501 shows 499).
	remaining := math.Round(target) - math.Round(*consumed)
	macro.Remaining = HealthMetricView{Value: formatThousands(remaining), Unit: unit, Present: true}
	macro.Over = math.Round(*consumed) > math.Round(target)
	macro.Percent = int(math.Round((*consumed / target) * 100))
	if macro.Percent < 0 {
		macro.Percent = 0
	}
	macro.HasBar = true
	macro.ProgressAria = fmt.Sprintf("%s: %s of %s %s", label, macro.Consumed.Value, macro.Target.Value, unit)
	return macro
}

// Short is the label for small tiles ("Carbs" rather than "Carbohydrates").
func (m HealthMacroView) Short() string {
	if m.Label == "Carbohydrates" {
		return "Carbs"
	}
	return m.Label
}

// Unit is the macro's unit, whichever side carries it.
func (m HealthMacroView) Unit() string {
	if m.Target.Unit != "" {
		return m.Target.Unit
	}
	return m.Consumed.Unit
}

// buildHealthTodayView turns the service response into presentation values.
// All formatting and every absent-versus-zero decision happens here so the
// template holds no arithmetic and cannot accidentally print a zero for
// missing data.
func buildHealthTodayView(today models.HealthTodayResponse, now time.Time) HealthTodayView {
	view := HealthTodayView{
		Date:        today.Date,
		HasSnapshot: today.Snapshot != nil,
		HasCycle:    today.HealthCycle != nil,
	}

	var snapshot models.DailyHealthSnapshot
	if today.Snapshot != nil {
		snapshot = *today.Snapshot
		if synced, err := time.Parse(time.RFC3339Nano, snapshot.LastSyncedAt); err == nil {
			view.HasLastSynced = true
			view.LastSynced = humaniseSince(now, synced)
			view.LastSyncedStale = now.Sub(synced) >= services.HealthSyncStaleAfter
		}
	}

	var cycle models.HealthCycle
	if today.HealthCycle != nil {
		cycle = *today.HealthCycle
		view.CycleTitle = cycle.Title
	}

	view.Calories = buildMacro("Calories", snapshot.DietaryCalories, cycle.DailyCalorieTarget, "kcal")
	view.Macros = []HealthMacroView{
		view.Calories,
		buildMacro("Protein", snapshot.ProteinG, cycle.ProteinTargetG, "g"),
		buildMacro("Carbohydrates", snapshot.CarbsG, cycle.CarbsTargetG, "g"),
		buildMacro("Fat", snapshot.FatG, cycle.FatTargetG, "g"),
		buildMacro("Fiber", snapshot.FiberG, cycle.FiberTargetG, "g"),
	}

	view.CalorieTarget = metricFromTarget(cycle.DailyCalorieTarget, "kcal")
	view.FoodRemaining = metricFromFloat(today.Observed.FoodRemaining, "kcal")
	view.ObservedBurn = metricFromFloat(today.Observed.ObservedTotalBurn, "kcal")
	view.ObservedEnergy = metricFromFloat(today.Observed.ObservedEnergyBalance, "kcal")

	// The cycle stores a deficit as a positive magnitude while the observed
	// balance is negative when under. Negate here, in one place, so the two
	// numbers can be compared on screen without the reader doing it. The
	// signed column that removes this asymmetry is a later commit.
	if cycle.TargetDeficitKcal > 0 {
		view.TargetEnergy = HealthMetricView{
			Value:   formatThousands(-cycle.TargetDeficitKcal),
			Unit:    "kcal",
			Present: true,
		}
	} else {
		view.TargetEnergy = absentMetric("kcal")
	}

	// Watch and phone steps stay separate: the sources overlap, so adding them
	// double-counts. This is an architectural rule, not a display preference.
	view.StepsWatch = metricFromInt(snapshot.StepsWatch, "")
	view.StepsPhone = metricFromInt(snapshot.StepsPhone, "")
	view.StepTarget = metricFromTarget(float64(cycle.StepTarget), "")
	primary := snapshot.StepsWatch
	view.Steps, view.StepsSource, view.StepsOther, view.OtherSource = view.StepsWatch, "Apple Watch", view.StepsPhone, "iPhone"
	if snapshot.StepsWatch == nil && snapshot.StepsPhone != nil {
		primary = snapshot.StepsPhone
		view.Steps, view.StepsSource, view.StepsOther, view.OtherSource = view.StepsPhone, "iPhone", view.StepsWatch, "Apple Watch"
	}
	if primary != nil && cycle.StepTarget > 0 {
		view.StepPercent = int(math.Min(100, math.Round(float64(*primary)/float64(cycle.StepTarget)*100)))
	}
	limits := map[string]bool{"Calories": true, "Carbohydrates": true, "Fat": true}
	for i := range view.Macros {
		m := &view.Macros[i]
		if !m.HasBar {
			continue
		}
		over := m.Over
		left := strings.TrimPrefix(m.Remaining.Value, "−")
		switch {
		case over && limits[m.Label]:
			m.State, m.Note = "warn", left+" "+m.Unit()+" over"
		case over || m.Remaining.Value == "0":
			m.Note = "Target met"
			if !limits[m.Label] {
				m.State = "ok"
			}
		default:
			m.Note = left + " " + m.Unit() + " to go"
		}
	}
	view.Calories = view.Macros[0]
	view.Nutrients = view.Macros[1:]
	// The ring and tiles use the calorie row's own target − eaten, so they
	// need both observations and nothing else.
	if view.Calories.HasBar {
		view.BudgetAbs = strings.TrimPrefix(view.Calories.Remaining.Value, "−")
	}
	if view.Calories.HasBar {
		view.CaloriePercent = int(math.Min(100, float64(view.Calories.Percent)))
		view.CaloriesOver = view.Calories.Over
	}
	view.ActiveEnergy = metricFromFloat(snapshot.ActiveEnergyKcal, "kcal")
	view.RestingEnergy = metricFromFloat(snapshot.RestingEnergyKcal, "kcal")
	view.ExerciseMinutes = metricFromFloat(snapshot.ExerciseMinutes, "min")

	// Burned and balance are shown from the rounded parts, so 1,851 resting +
	// 421 active reads 2,272, never 2,271 (#70). Same formula as the
	// service: burn = resting + active; balance = eaten − burn.
	if snapshot.ActiveEnergyKcal != nil && snapshot.RestingEnergyKcal != nil {
		burn := math.Round(*snapshot.RestingEnergyKcal) + math.Round(*snapshot.ActiveEnergyKcal)
		view.ObservedBurn = HealthMetricView{Value: formatThousands(burn), Unit: "kcal", Present: true}
		if snapshot.DietaryCalories != nil {
			balance := math.Round(*snapshot.DietaryCalories) - burn
			view.ObservedEnergy = HealthMetricView{Value: formatThousands(balance), Unit: "kcal", Present: true}
			if balance > 0 { // a surplus reads "+400", so the sign is never guessed
				view.ObservedEnergy.Value = "+" + view.ObservedEnergy.Value
			}
		}
	}
	if view.Calories.HasBar {
		view.caloriesLeft = math.Round(cycle.DailyCalorieTarget) - math.Round(*snapshot.DietaryCalories)
	}
	view.CalState = buildCalorieState(view, cycle.DailyCalorieTarget)

	return view
}

// humaniseSince renders a coarse age. Deliberately vague: the point is whether
// the number on screen is current, not the exact minute it arrived.
func humaniseSince(now, then time.Time) string {
	elapsed := now.Sub(then)
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		minutes := int(elapsed.Minutes())
		return fmt.Sprintf("%d min ago", minutes)
	case elapsed < 24*time.Hour:
		hours := int(elapsed.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(elapsed.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
