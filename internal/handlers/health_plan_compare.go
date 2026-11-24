package handlers

import (
	"fmt"
	"math"
	"time"

	"lifeops-tracker/internal/models"
)

// Cycle comparison (#205): what each cycle aimed for and what the complete
// days inside it actually averaged. "Complete" reuses the nutrition week's
// rules (#69): a day needs a dietary observation, must not be recorded zeros
// (weekNothingLogged), must not be partial (weekPartial), and today is never
// complete. Weight uses the shared 7-day average (weight_average.go): the
// window starting on the cycle's first day and the window ending on its last
// day (today while the cycle is still running). A window with fewer than
// minWeightAverageDays days of readings shows "—".

type cycleComparison struct {
	ID                                     int64
	Title, Range                           string
	TargetKcal, TargetProtein, TargetFiber string
	AvgKcal, AvgProtein, AvgFiber          string
	WeightStart, WeightEnd                 string
	DaysUsed                               int
	DaysText                               string
	// EstimateNote is "Includes AI estimates on 2 of 5 days counted" (GYST-180).
	EstimateNote string
}

type runningMean struct {
	sum float64
	n   int
}

func (m *runningMean) add(v *float64) {
	if v != nil {
		m.sum += *v
		m.n++
	}
}

func (m runningMean) text(unit string, whole bool) string {
	if m.n == 0 {
		return missingValue
	}
	avg := m.sum / float64(m.n)
	if whole {
		return formatThousands(avg) + unit
	}
	return niceNumber(math.Round(avg*10)/10) + unit
}

// targetText shows a cycle target; 0 means not set, so it is "—" (as on Today).
func targetText(v float64, unit string, whole bool) string {
	if v <= 0 {
		return missingValue
	}
	if whole {
		return formatThousands(v) + unit
	}
	return niceNumber(v) + unit
}

func weightWindowText(w weightWindow) string {
	if !w.ok {
		return missingValue
	}
	return fmt.Sprintf("%s kg · %d %s", trimKg(w.avg), w.days, pluralDays(w.days))
}

// dayComplete applies the complete-day rules to a counted day. The counted
// source is the day's one source (GYST diary or Apple Health, never summed).
// An Apple Health day uses the nutrition week's rules (#69): it needs dietary
// calories, must not be recorded zeros, and must not be partial. A GYST day
// is complete when its calories are known for every live item; a day with a
// missing calorie value is partial.
func dayComplete(d models.DayIntake, loc *time.Location) bool {
	switch d.Source {
	case models.IntakeSourceGYST:
		return d.Counted.Kcal != nil && d.Missing.Kcal == 0
	case models.IntakeSourceAppleHealth:
		s := d.Snapshot
		return s != nil && s.DietaryCalories != nil && !weekNothingLogged(*s) && !weekPartial(*s, loc)
	}
	return false
}

func buildCycleComparison(c models.HealthCycle, days []models.DayIntake, weights map[string]*localWeightDay, today string, loc *time.Location) cycleComparison {
	v := cycleComparison{ID: c.ID, Title: c.Title, Range: c.StartDate + " – " + c.EndDate,
		TargetKcal: targetText(c.DailyCalorieTarget, " kcal", true), TargetProtein: targetText(c.ProteinTargetG, " g", false), TargetFiber: targetText(c.FiberTargetG, " g", false),
		WeightStart: missingValue, WeightEnd: missingValue}
	if start, err := time.ParseInLocation("2006-01-02", c.StartDate, loc); err == nil {
		v.WeightStart = weightWindowText(windowEndingOn(weights, start.AddDate(0, 0, weightWindowDays-1)))
		endDate := c.EndDate
		if today < endDate {
			endDate = today
		}
		if end, err := time.ParseInLocation("2006-01-02", endDate, loc); err == nil && endDate >= c.StartDate {
			v.WeightEnd = weightWindowText(windowEndingOn(weights, end))
		}
	}
	var kcal, protein, fiber runningMean
	estimated := 0
	for _, d := range days {
		if d.Date == today || !dayComplete(d, loc) {
			continue
		}
		v.DaysUsed++
		if d.Estimated.Items > 0 {
			estimated++
		}
		kcal.add(d.Counted.Kcal)
		// A sum with a gap (an item without that nutrient) is partial: skip it.
		if d.Missing.ProteinG == 0 {
			protein.add(d.Counted.ProteinG)
		}
		if d.Missing.FiberG == 0 {
			fiber.add(d.Counted.FiberG)
		}
	}
	// Protein and fibre average only the complete days that recorded them.
	v.AvgKcal, v.AvgProtein, v.AvgFiber = kcal.text(" kcal", true), protein.text(" g", false), fiber.text(" g", false)
	v.EstimateNote = estimateDaysNote(estimated, v.DaysUsed, "counted")
	switch v.DaysUsed {
	case 0:
		v.DaysText = "No complete days counted"
	case 1:
		v.DaysText = "1 complete day counted"
	default:
		v.DaysText = fmt.Sprintf("%d complete days counted", v.DaysUsed)
	}
	return v
}
