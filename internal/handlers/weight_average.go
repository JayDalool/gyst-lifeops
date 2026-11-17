package handlers

import (
	"fmt"
	"math"
	"time"

	"lifeops-tracker/internal/models"
)

// GYST-154: one shared 7-day rolling average of body weight, used by the Today
// weight card, the Progress page and the coach tool so they always agree.
//
// Readings are bucketed by local day; a day's value is the mean of that day's
// readings, so two readings on one day count once. A window of 7 local days
// needs at least minWeightAverageDays days with readings, otherwise there is
// no average. Days without readings are never filled or counted as zero.
const (
	weightWindowDays     = 7
	minWeightAverageDays = 4
)

// weightAvgView is the display form of the average, with the wording fixed in
// one place.
type weightAvgView struct {
	Present bool
	Avg     string // kg, one decimal
	Days    int    // days with readings in the window
	// Line is the headline: "7-day average 90.4 kg · 5 days", or the
	// not-enough-readings message.
	Line string
	// HasPrev: the previous window (days -13..-7) also has enough readings.
	HasPrev     bool
	Prev        string
	PrevDays    int
	Change      string // signed change vs the previous window
	Down        bool
	ChangeLine  string // "−0.8 kg vs the previous 7 days"
	MixedScales bool
	Note        string // "Readings from different scales" when the window mixes sources
}

type weightWindow struct {
	ok      bool
	avg     float64
	days    int
	sources map[string]bool
}

type localWeightDay struct {
	sum     float64
	n       int
	sources map[string]bool
}

// groupWeightDays groups readings by local day. Readings after now are ignored.
func groupWeightDays(entries []models.HealthWeightEntry, now time.Time, loc *time.Location) map[string]*localWeightDay {
	days := map[string]*localWeightDay{}
	for _, e := range entries {
		at, err := time.Parse(time.RFC3339Nano, e.MeasuredAt)
		if err != nil || at.After(now) {
			continue
		}
		key := at.In(loc).Format("2006-01-02")
		d := days[key]
		if d == nil {
			d = &localWeightDay{sources: map[string]bool{}}
			days[key] = d
		}
		d.sum += e.WeightKg
		d.n++
		d.sources[e.Source] = true
	}
	return days
}

// windowEndingOn averages the 7 local days ending on (and including) end.
func windowEndingOn(days map[string]*localWeightDay, end time.Time) weightWindow {
	w := weightWindow{sources: map[string]bool{}}
	var sum float64
	for i := 0; i < weightWindowDays; i++ {
		d := days[end.AddDate(0, 0, -i).Format("2006-01-02")]
		if d == nil {
			continue
		}
		sum += d.sum / float64(d.n)
		w.days++
		for s := range d.sources {
			w.sources[s] = true
		}
	}
	if w.days >= minWeightAverageDays {
		w.ok, w.avg = true, sum/float64(w.days)
	}
	return w
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// buildWeightAverage computes the 7-day average ending today, and the previous
// window when it also qualifies.
func buildWeightAverage(entries []models.HealthWeightEntry, now time.Time, loc *time.Location) weightAvgView {
	days := groupWeightDays(entries, now, loc)
	today := now.In(loc)
	cur := windowEndingOn(days, today)
	if !cur.ok {
		return weightAvgView{Line: "Not enough readings for a 7-day average"}
	}
	v := weightAvgView{Present: true, Avg: trimKg(cur.avg), Days: cur.days}
	v.Line = fmt.Sprintf("7-day average %s kg · %d %s", v.Avg, cur.days, pluralDays(cur.days))
	if len(cur.sources) > 1 {
		v.MixedScales, v.Note = true, "Readings from different scales"
	}
	if prev := windowEndingOn(days, today.AddDate(0, 0, -weightWindowDays)); prev.ok {
		diff := round1(cur.avg - prev.avg)
		v.HasPrev, v.Prev, v.PrevDays = true, trimKg(prev.avg), prev.days
		v.Down, v.Change = diff <= 0, signedKg(diff)
		v.ChangeLine = v.Change + " vs the previous 7 days"
		if diff == 0 {
			v.ChangeLine = "No change vs the previous 7 days"
		}
	}
	return v
}

func pluralDays(n int) string {
	if n == 1 {
		return "day"
	}
	return "days"
}

// planStartAverageChange is the weight change since the plan start measured
// between 7-day averages: the 7 days beginning on the plan start against the
// current window. It reports ok only when both qualify and the windows do not
// overlap; otherwise the caller keeps the single-reading logic.
func planStartAverageChange(entries []models.HealthWeightEntry, start, now time.Time, loc *time.Location) (diff float64, ok bool) {
	days := groupWeightDays(entries, now, loc)
	today := now.In(loc)
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	endBase := startDay.AddDate(0, 0, weightWindowDays-1)
	if !endBase.Before(today.AddDate(0, 0, -(weightWindowDays - 1))) {
		return 0, false
	}
	base, cur := windowEndingOn(days, endBase), windowEndingOn(days, today)
	if !base.ok || !cur.ok {
		return 0, false
	}
	return round1(cur.avg - base.avg), true
}
