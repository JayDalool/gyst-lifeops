package handlers

import (
	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWeightHistoryRangeAndPlot(t *testing.T) {
	location, _ := clock.Load("America/Winnipeg")
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, location)
	entries := []models.HealthWeightEntry{
		{ID: 4, MeasuredAt: now.Add(time.Hour).Format(time.RFC3339), WeightKg: 999, Source: "manual"},
		{ID: 3, MeasuredAt: "2026-09-13T10:00:00-05:00", WeightKg: 80.25, Source: "manual"},
		{ID: 2, MeasuredAt: "2026-08-15T00:00:00-05:00", WeightKg: 82, Source: "apple_shortcuts"},
		{ID: 1, MeasuredAt: "2026-08-14T23:59:59-05:00", WeightKg: 83, Source: "manual"},
	}
	data := buildHealthProgress(entries, "30", now, location)
	if len(data.Points) != 2 || data.Latest != "80.25" || data.Minimum != "80.25" || data.Maximum != "82" {
		t.Fatalf("unexpected data: %+v", data)
	}
	if data.Points[0].X != "20.00" || data.Points[1].X != "580.00" || data.Points[0].Y != "20.00" || data.Points[1].Y != "180.00" {
		t.Fatalf("unexpected coordinates: %+v", data.Points)
	}
	if len(buildHealthProgress(entries, "90", now, location).Points) != 3 || len(buildHealthProgress(entries, "all", now, location).Points) != 3 {
		t.Fatal("range filtering lost readings or included a future reading")
	}
	body := renderBody(t, "health_progress.html", data)
	for _, want := range []string{"80.25", "Apple Shortcuts", "Recorded measurements", `role="img"`, "America/Winnipeg"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, "<td>999</td>") {
		t.Fatal("future reading displayed")
	}
	single := buildHealthProgress(entries[1:2], "30", now, location)
	if single.Points[0].X != "300.00" || single.Points[0].Y != "100.00" {
		t.Fatal("single point not centered")
	}
	entries[2].WeightKg = 80.25
	flat := buildHealthProgress(entries[1:3], "30", now, location)
	if flat.Points[0].Y != "100.00" || flat.Points[1].Y != "100.00" {
		t.Fatal("flat range incorrectly scaled")
	}
}

// Reading times carry Winnipeg's real UTC offset for each date (CST, CDT,
// then permanent UTC-05 from 2026-11-01) and never a zone abbreviation, so
// tzdb's "EST" label for Manitoba (Issue #193) does not reach the page.
func TestWeightTimesShowUTCOffsetNotAbbreviation(t *testing.T) {
	location, err := clock.Load("America/Winnipeg")
	if err != nil {
		t.Fatal(err)
	}
	entries := []models.HealthWeightEntry{
		{ID: 1, MeasuredAt: "2026-01-15T18:00:00Z", WeightKg: 81, Source: "manual"},
		{ID: 2, MeasuredAt: "2026-07-15T17:00:00Z", WeightKg: 80, Source: "manual"},
		{ID: 3, MeasuredAt: "2026-11-15T17:00:00Z", WeightKg: 79, Source: "manual"},
		{ID: 4, MeasuredAt: "2027-01-15T17:00:00Z", WeightKg: 78, Source: "manual"},
	}
	data := buildHealthProgress(entries, "all", time.Date(2027, 2, 1, 12, 0, 0, 0, time.UTC), location)
	want := []string{
		"2026-01-15 12:00:00 UTC-06:00",
		"2026-07-15 12:00:00 UTC-05:00",
		"2026-11-15 12:00:00 UTC-05:00",
		"2027-01-15 12:00:00 UTC-05:00",
	}
	if len(data.Points) != len(want) {
		t.Fatalf("got %d points, want %d", len(data.Points), len(want))
	}
	for i, w := range want {
		if data.Points[i].Date != w {
			t.Errorf("point %d = %q, want %q", i, data.Points[i].Date, w)
		}
	}
	body := renderBody(t, "health_progress.html", data)
	for _, abbreviation := range []string{" EST", " CST", " CDT", " MBT"} {
		if strings.Contains(body, abbreviation) {
			t.Errorf("page shows the zone abbreviation %q", strings.TrimSpace(abbreviation))
		}
	}
}

func TestHealthProgressRoute(t *testing.T) {
	h := testHandler(t)
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/health/progress", 200}, {"GET", "/health/progress?range=90", 200}, {"GET", "/health/progress?range=all", 200}, {"GET", "/health/progress?range=bad", 400}, {"POST", "/health/progress", 405},
	} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(tc.method, tc.path, nil))
		if r.Code != tc.status {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, r.Code, r.Body.String())
		}
		if r.Code == http.StatusOK && (!strings.Contains(r.Body.String(), "No weight readings") || strings.Contains(r.Body.String(), `class="weight-dot"`)) {
			t.Fatal("empty history should not draw data")
		}
	}
	if authExemptPath("/health/progress") {
		t.Fatal("progress must require authentication")
	}
}
