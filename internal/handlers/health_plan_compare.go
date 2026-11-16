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
// complete. 7-day weight averages are not shown: no helper exists yet.

type cycleComparison struct {
	ID                                     int64
	Title, Range                           string
	TargetKcal, TargetProtein, TargetFiber string
	AvgKcal, AvgProtein, AvgFiber          string
	DaysUsed                               int
	DaysText                               string
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

func buildCycleComparison(c models.HealthCycle, days []models.DailyHealthSnapshot, today string, loc *time.Location) cycleComparison {
	v := cycleComparison{ID: c.ID, Title: c.Title, Range: c.StartDate + " – " + c.EndDate,
		TargetKcal: formatThousands(c.DailyCalorieTarget) + " kcal", TargetProtein: niceNumber(c.ProteinTargetG) + " g", TargetFiber: niceNumber(c.FiberTargetG) + " g"}
	var kcal, protein, fiber runningMean
	for _, s := range days {
		if s.Date == today || s.DietaryCalories == nil || weekNothingLogged(s) || weekPartial(s, loc) {
			continue
		}
		v.DaysUsed++
		kcal.add(s.DietaryCalories)
		protein.add(s.ProteinG)
		fiber.add(s.FiberG)
	}
	// Protein and fibre average only the complete days that recorded them.
	v.AvgKcal, v.AvgProtein, v.AvgFiber = kcal.text(" kcal", true), protein.text(" g", false), fiber.text(" g", false)
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
