package handlers

import (
	"strconv"

	"lifeops-tracker/internal/models"
)

// AI estimate labels (GYST-180). A GYST day's total can include macros the
// photo AI estimated. Every screen that shows that total, and the Coach and
// the Secretary, take their wording from DayIntake.Estimated through the
// helpers in this file, so no surface counts provenance or phrases the label
// on its own and none can forget it. Numbers use formatThousands, the same
// whole-number rounding as the totals they sit beside.

// HealthEstimateView is the label for one day's AI-estimated share.
type HealthEstimateView struct {
	Present  bool
	Items    int
	Kcal     HealthMetricView // nil sum: not Present
	ProteinG HealthMetricView
	// Note is "Includes AI estimates: 2 items, about 410 kcal".
	Note string
}

func itemsText(n int) string {
	if n == 1 {
		return "1 item"
	}
	return strconv.Itoa(n) + " items"
}

// estimateNote is the one sentence every day-total screen shows. The kcal part
// is left out, never zeroed, when no estimated item has calories.
func estimateNote(e models.EstimatedIntake) string {
	if e.Items <= 0 {
		return ""
	}
	n := "Includes AI estimates: " + itemsText(e.Items)
	if e.Kcal != nil {
		n += ", about " + formatThousands(*e.Kcal) + " kcal"
	}
	return n
}

func buildEstimateView(e models.EstimatedIntake) HealthEstimateView {
	if e.Items <= 0 {
		return HealthEstimateView{}
	}
	return HealthEstimateView{Present: true, Items: e.Items, Kcal: metricFromFloat(e.Kcal, "kcal"),
		ProteinG: metricFromFloat(e.ProteinG, "g"), Note: estimateNote(e)}
}

// estimateDaysNote labels a multi-day figure: "Includes AI estimates on 3 of
// 7 days". suffix is appended for the plan comparison ("counted").
func estimateDaysNote(withEstimates, of int, suffix string) string {
	if withEstimates <= 0 {
		return ""
	}
	n := "Includes AI estimates on " + strconv.Itoa(withEstimates) + " of " + strconv.Itoa(of) + " days"
	if suffix != "" {
		n += " " + suffix
	}
	return n
}

// aiEstimated is the Coach's and the brief's aggregate: counts and totals of
// values already inside the day's totals, never names, photos, ids or dates.
// It is nil (key absent) when the day has no estimated item.
func (e HealthEstimateView) aiEstimated() map[string]any {
	if !e.Present {
		return nil
	}
	return map[string]any{
		"items":     e.Items,
		"kcal":      metricOrNil(e.Kcal),
		"protein_g": metricOrNil(e.ProteinG),
		"note":      aiEstimatedCoachNote,
	}
}

// aiEstimatedCoachNote tells the model how to treat the numbers. The totals it
// sends (eaten, budget, energy balance) all use the same intake.
const aiEstimatedCoachNote = "These totals include AI estimates from meal photos; treat as approximate, not measured. energy_balance_so_far and budget use the same total."
