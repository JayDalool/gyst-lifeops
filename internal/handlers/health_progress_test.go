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
