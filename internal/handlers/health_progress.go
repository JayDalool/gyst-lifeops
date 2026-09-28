package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"lifeops-tracker/internal/models"
)

type weightPoint struct {
	Date, Timestamp, Value, Source string
	X, Y                           string
}
type healthProgressData struct {
	Page, Range, Timezone                         string
	Points                                        []weightPoint
	Latest, Minimum, Maximum, FirstDate, LastDate string
}

// Plot observations at their actual times, without filling unobserved days.
func buildHealthProgress(entries []models.HealthWeightEntry, period string, now time.Time, location *time.Location) healthProgressData {
	data := healthProgressData{Page: "health", Range: period, Timezone: location.String()}
	type reading struct {
		entry models.HealthWeightEntry
		at    time.Time
	}
	readings := []reading{}
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	if period == "30" {
		start = start.AddDate(0, 0, -29)
	} else {
		start = start.AddDate(0, 0, -89)
	}
	for _, entry := range entries {
		at, err := time.Parse(time.RFC3339Nano, entry.MeasuredAt)
		if err != nil || at.After(now) || (period != "all" && at.Before(start)) {
			continue
		}
		readings = append(readings, reading{entry, at})
	}
	sort.SliceStable(readings, func(i, j int) bool {
		if readings[i].at.Equal(readings[j].at) {
			return readings[i].entry.ID < readings[j].entry.ID
		}
		return readings[i].at.Before(readings[j].at)
	})
	if len(readings) == 0 {
		return data
	}
	first, last := readings[0].at, readings[len(readings)-1].at
	low, high := readings[0].entry.WeightKg, readings[0].entry.WeightKg
	for _, r := range readings {
		if r.entry.WeightKg < low {
			low = r.entry.WeightKg
		}
		if r.entry.WeightKg > high {
			high = r.entry.WeightKg
		}
	}
	number := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	data.Minimum, data.Maximum = number(low), number(high)
	data.Latest = number(readings[len(readings)-1].entry.WeightKg)
	data.FirstDate, data.LastDate = first.In(location).Format("2006-01-02"), last.In(location).Format("2006-01-02")
	for _, r := range readings {
		x, y := 300.0, 100.0
		if !last.Equal(first) {
			x = 20 + 560*r.at.Sub(first).Seconds()/last.Sub(first).Seconds()
		}
		if high != low {
			y = 180 - 160*(r.entry.WeightKg-low)/(high-low)
		}
		source := "Unknown source"
		switch r.entry.Source {
		case "manual":
			source = "Manual"
		case "apple_shortcuts":
			source = "Apple Shortcuts"
		}
		data.Points = append(data.Points, weightPoint{Date: r.at.In(location).Format("2006-01-02 15:04:05 MST"), Timestamp: r.entry.MeasuredAt, Value: number(r.entry.WeightKg), Source: source, X: fmt.Sprintf("%.2f", x), Y: fmt.Sprintf("%.2f", y)})
	}
	return data
}

func (h *Handler) healthProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	period := r.URL.Query().Get("range")
	if period == "" {
		period = "30"
	}
	if period != "30" && period != "90" && period != "all" {
		http.Error(w, "range must be 30, 90 or all", http.StatusBadRequest)
		return
	}
	entries, err := h.Service.ListHealthWeightEntries()
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "health_progress.html", buildHealthProgress(entries, period, h.Service.Now(), h.Service.Location()))
}
