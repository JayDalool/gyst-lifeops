package handlers

import (
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
)

// All values are fictional.
func avgEntries(loc *time.Location, source string, readings ...[3]float64) []models.HealthWeightEntry {
	// each reading: {month-day offset from 2026-10-10 (negative = earlier), hour, kg}
	base := time.Date(2026, 10, 10, 0, 0, 0, 0, loc)
	out := []models.HealthWeightEntry{}
	for _, r := range readings {
		at := base.AddDate(0, 0, int(r[0])).Add(time.Duration(r[1]) * time.Hour)
		out = append(out, models.HealthWeightEntry{MeasuredAt: at.Format(time.RFC3339), WeightKg: r[2], Source: source})
	}
	return out
}

func TestWeightAverage(t *testing.T) {
	loc, _ := clock.Load("America/Winnipeg")
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, loc)

	// Three days with readings is not enough, however many readings there are.
	few := avgEntries(loc, "manual", [3]float64{0, 7, 90}, [3]float64{-1, 7, 91}, [3]float64{-2, 7, 92}, [3]float64{-2, 20, 93})
	if v := buildWeightAverage(few, now, loc); v.Present || v.Line != "Not enough readings for a 7-day average" {
		t.Fatalf("3 days: %+v", v)
	}
	if v := buildWeightAverage(nil, now, loc); v.Present || v.Avg != "" {
		t.Fatalf("none: %+v", v)
	}

	// Four days: the mean of the days. A second reading on the same day counts once.
	four := avgEntries(loc, "manual", [3]float64{0, 7, 90}, [3]float64{-1, 7, 91}, [3]float64{-2, 7, 92}, [3]float64{-3, 7, 90}, [3]float64{-3, 20, 92})
	v := buildWeightAverage(four, now, loc)
	// day values 90, 91, 92, 91 (the -3 day averages 90 and 92) -> 91.0
	if !v.Present || v.Avg != "91" || v.Days != 4 || v.Line != "7-day average 91 kg · 4 days" || v.HasPrev || v.MixedScales {
		t.Fatalf("4 days: %+v", v)
	}

	// Readings older than 7 days (day -7) are outside the window; future ones are ignored.
	out := append(four, avgEntries(loc, "manual", [3]float64{-7, 7, 99}, [3]float64{1, 7, 99})...)
	if v := buildWeightAverage(out, now, loc); v.Avg != "91" || v.Days != 4 {
		t.Fatalf("window edge: %+v", v)
	}
}

func TestWeightAveragePreviousWindowAndMixedSources(t *testing.T) {
	loc, _ := clock.Load("America/Winnipeg")
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, loc)
	entries := avgEntries(loc, "manual", [3]float64{0, 7, 90}, [3]float64{-1, 7, 90}, [3]float64{-2, 7, 90}, [3]float64{-3, 7, 90})
	// Previous window is days -13..-7; three days is not enough.
	entries = append(entries, avgEntries(loc, "manual", [3]float64{-7, 7, 91}, [3]float64{-8, 7, 91}, [3]float64{-9, 7, 91})...)
	if v := buildWeightAverage(entries, now, loc); v.HasPrev || v.ChangeLine != "" {
		t.Fatalf("3 previous days: %+v", v)
	}
	entries = append(entries, avgEntries(loc, "apple_shortcuts", [3]float64{-13, 7, 91.4})...)
	v := buildWeightAverage(entries, now, loc)
	// Previous: (91*3 + 91.4)/4 = 91.1; change 90 - 91.1 = -1.1
	if !v.HasPrev || v.Prev != "91.1" || v.PrevDays != 4 || v.Change != "−1.1 kg" || !v.Down || v.ChangeLine != "−1.1 kg vs the previous 7 days" {
		t.Fatalf("previous window: %+v", v)
	}
	if v.MixedScales {
		t.Fatalf("mixed sources only in the previous window must not flag the current one: %+v", v)
	}
	entries = append(entries, avgEntries(loc, "apple_shortcuts", [3]float64{-1, 20, 90})...)
	if v := buildWeightAverage(entries, now, loc); !v.MixedScales || v.Note != "Readings from different scales" {
		t.Fatalf("mixed: %+v", v)
	}
}

func TestWeightAverageLocalDayBoundary(t *testing.T) {
	loc, _ := clock.Load("America/Winnipeg")
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, loc)
	// 23:30 local on day -1 is the next day in UTC; it must stay on day -1,
	// so day -1 has two readings (one day), not a reading on today.
	late := time.Date(2026, 10, 9, 23, 30, 0, 0, loc)
	if late.UTC().Day() != 10 {
		t.Fatal("test setup: late reading must fall on the next UTC day")
	}
	entries := avgEntries(loc, "manual", [3]float64{-1, 7, 90}, [3]float64{-2, 7, 91}, [3]float64{-3, 7, 92})
	entries = append(entries, models.HealthWeightEntry{MeasuredAt: late.Format(time.RFC3339), WeightKg: 92, Source: "manual"})
	if v := buildWeightAverage(entries, now, loc); v.Present {
		t.Fatalf("late reading was counted as a separate day: %+v", v)
	}
	entries = append(entries, avgEntries(loc, "manual", [3]float64{-4, 7, 90})...)
	// days: -1 -> (90+92)/2 = 91, -2 91, -3 92, -4 90 -> 91.0
	if v := buildWeightAverage(entries, now, loc); !v.Present || v.Days != 4 || v.Avg != "91" {
		t.Fatalf("boundary: %+v", v)
	}
}

func TestPlanStartAverageChange(t *testing.T) {
	loc, _ := clock.Load("America/Winnipeg")
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, loc)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
	mk := func(day int, kg float64) models.HealthWeightEntry {
		return models.HealthWeightEntry{MeasuredAt: time.Date(2026, 10, day, 7, 0, 0, 0, loc).Format(time.RFC3339), WeightKg: kg, Source: "manual"}
	}
	entries := []models.HealthWeightEntry{mk(1, 92), mk(2, 92), mk(3, 91), mk(5, 91), mk(20, 90), mk(19, 90), mk(18, 89), mk(16, 89)}
	diff, ok := planStartAverageChange(entries, start, now, loc)
	if !ok || diff != -2 { // 91.5 -> 89.5 would be -2.0
		t.Fatalf("diff %v ok %v", diff, ok)
	}
	// Plan started inside the current window: windows overlap, so no average change.
	if _, ok := planStartAverageChange(entries, time.Date(2026, 10, 15, 0, 0, 0, 0, loc), now, loc); ok {
		t.Fatal("overlapping windows must fall back")
	}
	// Too few days at the start.
	if _, ok := planStartAverageChange(entries[3:], start, now, loc); ok {
		t.Fatal("start window with fewer than 4 days must fall back")
	}
}

// Today, Progress and the coach tool read the same stored data and must show
// the same number.
func TestWeightAverageAgreesAcrossTodayProgressAndCoach(t *testing.T) {
	fc := &fakeCoach{call: []string{"get_weight_trend"}, reply: "OK"}
	h := coachHandler(t, fc)
	loc := h.Service.Location()
	h.Service.Clock.Now = func() time.Time { return time.Date(2026, 10, 10, 12, 0, 0, 0, loc) }
	for i, kg := range []float64{90.0, 90.6, 91.2, 90.4, 90.8} {
		at := time.Date(2026, 10, 10-i, 7, 0, 0, 0, loc)
		w := &models.HealthWeightEntry{MeasuredAt: at.Format(time.RFC3339), WeightKg: kg, Source: "manual"}
		if err := h.Service.CreateHealthWeightEntry(w); err != nil {
			t.Fatal(err)
		}
	}
	// (90.0+90.6+91.2+90.4+90.8)/5 = 90.6
	const want = "7-day average 90.6 kg · 5 days"
	if body := cfgGet(t, h, "/health"); !strings.Contains(body, want) {
		t.Fatalf("Today lacks %q", want)
	}
	if body := cfgGet(t, h, "/health/progress"); !strings.Contains(body, want) {
		t.Fatalf("Progress lacks %q", want)
	}
	if w := ask(h, "How is my weight trending?", "http://example.com"); w.Code != 303 {
		t.Fatalf("ask: %d", w.Code)
	}
	out := fc.toolData["get_weight_trend"]
	if !strings.Contains(out, `"seven_day_average_kg":"90.6"`) || !strings.Contains(out, `"seven_day_average_days_used":5`) {
		t.Fatalf("coach tool: %s", out)
	}
}

func TestWeightAverageNotEnoughNeverZero(t *testing.T) {
	fc := &fakeCoach{call: []string{"get_weight_trend"}, reply: "OK"}
	h := coachHandler(t, fc)
	loc := h.Service.Location()
	h.Service.Clock.Now = func() time.Time { return time.Date(2026, 10, 10, 12, 0, 0, 0, loc) }
	for i := 0; i < 3; i++ {
		at := time.Date(2026, 10, 10-i, 7, 0, 0, 0, loc)
		if err := h.Service.CreateHealthWeightEntry(&models.HealthWeightEntry{MeasuredAt: at.Format(time.RFC3339), WeightKg: 90 + float64(i), Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	const msg = "Not enough readings for a 7-day average"
	for _, p := range []string{"/health", "/health/progress"} {
		body := cfgGet(t, h, p)
		if !strings.Contains(body, msg) {
			t.Fatalf("%s lacks %q", p, msg)
		}
		if strings.Contains(body, "7-day average 0") {
			t.Fatalf("%s shows a zero average", p)
		}
	}
	ask(h, "weight?", "http://example.com")
	if out := fc.toolData["get_weight_trend"]; !strings.Contains(out, `"seven_day_average_kg":null`) || !strings.Contains(out, `"seven_day_average_days_used":null`) {
		t.Fatalf("coach tool: %s", out)
	}
}
