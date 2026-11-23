package handlers

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
)

// Health navigation (GYST-174, 13B DD-22): Health has four sections, "summary",
// "nutrition", "progress" and "training". A section page's data reports which
// one it is, so the switcher and the sidebar can mark it. Every other Health
// page (Plan, More, Coach, checklist, foods, profile) is pushed from a row and
// reports "": nothing defaults to a section, or More and Coach would show
// Summary as the open one.
type healthTabber interface{ HealthTabName() string }

func (d HealthData) HealthTabName() string       { return d.HealthTab }
func (nutritionData) HealthTabName() string      { return "nutrition" }
func (healthProgressData) HealthTabName() string { return "progress" }

func healthTab(data any) string {
	if t, ok := data.(healthTabber); ok {
		return t.HealthTabName()
	}
	return ""
}

// healthWeightView is the compact weight card: latest reading, change since
// the plan started (or the earliest reading in range), and a sparkline.
type healthWeightView struct {
	Present          bool
	Latest, LatestAt string
	HasChange, Down  bool
	// NoneSincePlan: readings exist, but none since the plan started.
	NoneSincePlan    bool
	Change, Since    string
	Points           string
	Readings         int
	SparkDescription string
	// Average is the shared 7-day rolling average (GYST-154). ChangeIsAverage:
	// Change compares two 7-day averages, not two single readings.
	Average         weightAvgView
	ChangeIsAverage bool
}

func buildHealthWeight(entries []models.HealthWeightEntry, cycleStart string, now time.Time, loc *time.Location) healthWeightView {
	type reading struct {
		at time.Time
		kg float64
	}
	readings := []reading{}
	for _, e := range entries {
		at, err := time.Parse(time.RFC3339Nano, e.MeasuredAt)
		if err == nil && !at.After(now) {
			readings = append(readings, reading{at, e.WeightKg})
		}
	}
	if len(readings) == 0 {
		return healthWeightView{}
	}
	sort.SliceStable(readings, func(i, j int) bool { return readings[i].at.Before(readings[j].at) })
	latest := readings[len(readings)-1]
	v := healthWeightView{Average: buildWeightAverage(entries, now, loc), Present: true, Latest: trimKg(latest.kg), LatestAt: dayLabel(latest.at, now, loc), Readings: len(readings)}

	// Baseline: first reading on or after the plan start, else the first
	// reading of the last 90 days.
	from := now.AddDate(0, 0, -90)
	label := ""
	if start, err := time.ParseInLocation("2006-01-02", cycleStart, loc); err == nil {
		from, label = start, start.Format("Jan 2")
	}
	var baseline *reading
	spark := []float64{}
	for i := range readings {
		if readings[i].at.Before(from) {
			continue
		}
		if baseline == nil {
			baseline = &readings[i]
		}
		spark = append(spark, readings[i].kg)
	}
	if baseline == nil && label != "" {
		v.NoneSincePlan, v.Since = true, label
	}
	if baseline != nil && baseline.at.Before(latest.at) {
		diff := math.Round((latest.kg-baseline.kg)*10) / 10
		v.HasChange, v.Down = true, diff <= 0
		v.Change = signedKg(diff)
		if label == "" {
			label = baseline.at.In(loc).Format("Jan 2")
		}
		v.Since = label
	}
	// Prefer 7-day averages over single readings when both windows qualify.
	if start, err := time.ParseInLocation("2006-01-02", cycleStart, loc); err == nil {
		if diff, ok := planStartAverageChange(entries, start, now, loc); ok {
			v.HasChange, v.NoneSincePlan, v.ChangeIsAverage = true, false, true
			v.Down, v.Change, v.Since = diff <= 0, signedKg(diff), start.Format("Jan 2")
		}
	}
	if len(spark) >= 2 {
		v.Points = sparkPoints(spark)
		v.SparkDescription = fmt.Sprintf("Weight trend, %d readings from %s to %s kg", len(spark), trimKg(spark[0]), trimKg(spark[len(spark)-1]))
	}
	return v
}

func trimKg(kg float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", kg), ".0")
}

func signedKg(diff float64) string {
	switch {
	case diff < 0:
		return "−" + trimKg(-diff) + " kg"
	case diff > 0:
		return "+" + trimKg(diff) + " kg"
	}
	return "No change"
}

// dayLabel says Today, Yesterday or a short date, in the app timezone.
func dayLabel(at, now time.Time, loc *time.Location) string {
	a, n := at.In(loc), now.In(loc)
	day := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc) }
	switch int(day(n).Sub(day(a)).Round(24*time.Hour) / (24 * time.Hour)) {
	case 0:
		return "Today"
	case 1:
		return "Yesterday"
	}
	if a.Year() != n.Year() {
		return a.Format("Jan 2, 2006")
	}
	return a.Format("Jan 2")
}

// healthMoreData backs /health/more: plan, weight, goals and data sources in
// grouped lists. Everything here already existed on the old Health page.
type healthMoreData struct {
	HealthData
	HevyConnected bool
	HevySynced    string
}

func (h *Handler) healthMore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	data, ok := h.healthBase(w)
	if !ok {
		return
	}
	more := healthMoreData{HealthData: data, HevyConnected: h.Service.Training != nil}
	if cache, err := h.Service.Repo.TrainingCache("hevy"); err == nil && !cache.SyncedAt.IsZero() {
		more.HevySynced = humaniseSince(h.Service.Now(), cache.SyncedAt)
	}
	h.render(w, "health_more.html", more)
}
