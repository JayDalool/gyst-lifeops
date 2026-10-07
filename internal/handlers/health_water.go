package handlers

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// Water (#60, #76). Jay logs water by tapping his bottle on Today: each tap
// adds one bottle of the plan's size to today, and the bottle fills to the
// share of the plan's target (one 1.5 L bottle of a 3.0 L target fills it
// halfway). Nothing is estimated: no taps means nothing logged.

// waterChecklistKey holds the name of the measured water checklist row when
// Jay has added it; unset means the row isn't shown. It lives in
// app_settings so the checklist table (and its kind constraint) is untouched.
const waterChecklistKey = "health.checklist.water"

type healthWaterView struct {
	HasTarget, HasBottle bool
	Logged               bool // at least one tap today
	Taps                 int
	Total, Target        string // litres, "1.5"
	Bottle               string
	BottlesNeeded        int // taps to reach the target; 0 when either is unset
	Percent              int // fill height, 0-100
	// FromPercent is where the fill animation starts after a tap or undo;
	// -1 means no animation (a normal page load).
	FromPercent int
	Done        bool
	Left        string // "1.5 L to go"
	Ticks       []int  // bottle-share marks on the bottle, % of the fill height
	Notice      string // after a tap or undo
	Error       string

	targetML int64
}

// litres formats millilitres as litres with at least one decimal: 1500 →
// "1.5", 3000 → "3.0", 355 → "0.355".
func litres(ml int64) string {
	s := strconv.FormatFloat(float64(ml)/1000, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func buildHealthWater(cycle *models.HealthCycle, totalML int64, taps int, fromML int64) healthWaterView {
	v := healthWaterView{Logged: taps > 0, Taps: taps, Total: litres(totalML), FromPercent: -1}
	if cycle == nil {
		return v
	}
	v.HasTarget, v.HasBottle, v.targetML = cycle.WaterTargetML > 0, cycle.WaterBottleML > 0, cycle.WaterTargetML
	v.Target, v.Bottle = litres(cycle.WaterTargetML), litres(cycle.WaterBottleML)
	if !v.HasTarget {
		return v
	}
	share := func(ml int64) int {
		return int(math.Min(100, math.Round(float64(ml)/float64(cycle.WaterTargetML)*100)))
	}
	v.Percent = share(totalML)
	v.animateFrom(fromML)
	v.Done = totalML >= cycle.WaterTargetML
	if !v.Done {
		v.Left = litres(cycle.WaterTargetML-totalML) + " L to go"
	}
	if v.HasBottle {
		v.BottlesNeeded = int(math.Ceil(float64(cycle.WaterTargetML) / float64(cycle.WaterBottleML)))
		for k := int64(1); k*cycle.WaterBottleML < cycle.WaterTargetML; k++ {
			v.Ticks = append(v.Ticks, share(k*cycle.WaterBottleML))
		}
	}
	return v
}

// animateFrom sets where the fill animation starts, from a redirect's wfrom.
func (v *healthWaterView) animateFrom(fromML int64) {
	v.FromPercent = -1
	if fromML >= 0 && v.targetML > 0 {
		v.FromPercent = int(math.Min(100, math.Round(float64(fromML)/float64(v.targetML)*100)))
	}
}

// TickY is a bottle mark's height in the bottle SVG's coordinates: the
// water surface runs from y 342 (empty) to y 80 (full).
func (v healthWaterView) TickY(percent int) string {
	return strconv.FormatFloat(80+(1-float64(percent)/100)*262, 'f', 1, 64)
}

// Level is the fill as a 0-1 number for the CSS custom property.
func (v healthWaterView) Level() string {
	return strconv.FormatFloat(float64(v.Percent)/100, 'f', 2, 64)
}

// FromLevel is the animation start as a 0-1 number.
func (v healthWaterView) FromLevel() string {
	return strconv.FormatFloat(float64(v.FromPercent)/100, 'f', 2, 64)
}

// waterQuery reads the redirect parameters after a tap or undo. wfrom is
// only an animation start, bounded and otherwise ignored.
func waterQuery(q url.Values) (fromML int64, notice, errText string) {
	fromML = -1
	if n, err := strconv.ParseInt(q.Get("wfrom"), 10, 64); err == nil && n >= 0 && n <= 20000 {
		fromML = n
	}
	switch q.Get("water") {
	case "added":
		notice = "Bottle added"
	case "undone":
		notice = "Last bottle removed"
	case "nobottle":
		errText, fromML = "Set your bottle size in Health → Plan first", -1
	case "max":
		errText, fromML = "That's more than 15 L today; check for a double tap", -1
	default:
		fromML = -1
	}
	return fromML, notice, errText
}

// waterChecklistRow is the measured water row for the daily checklist.
func waterChecklistRow(name string, v healthWaterView) healthChecklistRow {
	row := healthChecklistRow{Name: name, Measured: true, Source: "water bottles logged", Done: v.Done, Value: "Set target in Plan"}
	if v.HasTarget {
		row.Value = "— / " + v.Target + " L"
		if v.Logged {
			row.Value = v.Total + " / " + v.Target + " L"
		}
	}
	return row
}

// healthWater handles POST /health/water: one bottle for today.
func (h *Handler) healthWater(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "POST") {
		return
	}
	before, _, err := h.Service.Repo.HealthWaterDay(h.Service.TodayString())
	if err != nil {
		serverError(w, err)
		return
	}
	_, err = h.Service.LogHealthWaterBottle()
	var invalid *services.HealthValidationError
	switch {
	case err == nil:
		http.Redirect(w, r, "/health?water=added&wfrom="+strconv.FormatInt(before, 10)+"#water", http.StatusSeeOther)
	case errors.Is(err, services.ErrWaterDailyCap):
		http.Redirect(w, r, "/health?water=max#water", http.StatusSeeOther)
	case errors.As(err, &invalid):
		http.Redirect(w, r, "/health?water=nobottle#water", http.StatusSeeOther)
	default:
		serverError(w, err)
	}
}

// healthWaterUndo handles POST /health/water/undo: removes today's last tap.
func (h *Handler) healthWaterUndo(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "POST") {
		return
	}
	before, _, err := h.Service.Repo.HealthWaterDay(h.Service.TodayString())
	if err != nil {
		serverError(w, err)
		return
	}
	switch _, err := h.Service.UndoHealthWater(); {
	case errors.Is(err, sql.ErrNoRows):
		http.Redirect(w, r, "/health#water", http.StatusSeeOther)
	case err != nil:
		serverError(w, err)
	default:
		http.Redirect(w, r, "/health?water=undone&wfrom="+strconv.FormatInt(before, 10)+"#water", http.StatusSeeOther)
	}
}

// parseLitres reads an optional litres field ("3", "1.5", "1,5"); blank is 0.
func parseLitres(raw, label string) (int64, error) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0, &services.HealthValidationError{Message: label + " must be a number of litres, like 1.5"}
	}
	return int64(math.Round(v * 1000)), nil
}

// litresField shows a stored millilitre value in the Plan form; 0 is blank.
func litresField(ml int64) string {
	if ml <= 0 {
		return ""
	}
	return litres(ml)
}
