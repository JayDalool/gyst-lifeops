package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func TestNutritionRouteAndMethods(t *testing.T) {
	h := testHandler(t)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(method, "/health/nutrition", nil))
		if method == http.MethodGet {
			if r.Code != 200 || !strings.Contains(r.Body.String(), "No food logged yet today") {
				t.Fatalf("GET: %d %s", r.Code, r.Body.String())
			}
		} else if r.Code != 405 {
			t.Fatalf("POST: %d", r.Code)
		}
	}
	if authExemptPath("/health/nutrition") {
		t.Fatal("nutrition must require a session")
	}
}

// Nutrition v3 (#55): the same protections as before in the new layout.
// Missing stays "—"/"Not received", a recorded zero is 0, over target is
// shown with its real value, and no plan means no bars.
func TestNutritionObservationStates(t *testing.T) {
	cycle := viewCycle()
	for _, tc := range []struct {
		name         string
		metrics      models.HealthMetrics
		hasNutrition bool
		want         []string
	}{
		{"missing", models.HealthMetrics{}, false, []string{"No food logged yet today", "Not received"}},
		{"activity only", models.HealthMetrics{StepsWatch: viewInt(100)}, false, []string{"No food logged yet today"}},
		{"zero", models.HealthMetrics{DietaryCalories: viewFloat(0)}, true, []string{`0<small> / 2,300 kcal</small>`, "2,300 kcal left"}},
		{"partial over target", models.HealthMetrics{DietaryCalories: viewFloat(3000), ProteinG: viewFloat(12)}, true, []string{"3,000", "700 kcal over", "12<small> / 180 g</small>", "Not received", `hv3-bar lg warn`, `width:100%`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := models.DailyHealthSnapshot{HealthMetrics: tc.metrics, LastSyncedAt: "2026-09-09T00:00:00Z"}
			data := buildNutritionData(buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snapshot, HealthCycle: &cycle}, viewNow()))
			if data.HasNutrition != tc.hasNutrition {
				t.Fatal("nutrition presence inferred incorrectly")
			}
			body := renderBody(t, "nutrition.html", data)
			for _, want := range append(tc.want, "Stale", "MyFitnessPal → Apple Health", "Daily totals only", `href="https://www.myfitnesspal.com/"`) {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			if !tc.hasNutrition && (!strings.Contains(body, `hv3-bar lg empty"`) || strings.Contains(body, `hv3-bar lg warn"`)) {
				t.Fatal("missing intake must render only the empty calorie track")
			}
		})
	}
	data := buildNutritionData(buildHealthTodayView(models.HealthTodayResponse{Snapshot: &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(123)}}}, viewNow()))
	body := renderBody(t, "nutrition.html", data)
	if !strings.Contains(body, "123<small> kcal</small>") || !strings.Contains(body, "No health plan yet") || strings.Contains(body, "hv3-bar lg") {
		t.Fatal("targetless intake misrepresented")
	}
}

// The week shows logged days, marks unlogged days as such (not zero), flags
// days over target and averages only logged days before today. A day of
// recorded zeros (0 kcal, 0 or no protein) counts as nothing logged (#69).
func TestNutritionWeek(t *testing.T) {
	data := nutritionData{}
	history := []models.DailyHealthSnapshot{
		{Date: "2026-09-04", HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(2000)}},
		{Date: "2026-09-06", HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(2600)}},
		{Date: "2026-09-08", HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(0)}},
		{Date: "2026-09-09", HealthMetrics: models.HealthMetrics{StepsWatch: viewInt(500)}},
		{Date: "2026-09-10", HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(900)}},
	}
	buildNutritionWeek(&data, history, "2026-09-10", 2250, time.UTC)
	if len(data.Week) != 7 || !data.Week[6].IsToday || data.Week[6].Value != "900" {
		t.Fatalf("week %+v", data.Week)
	}
	if data.Week[1].Present || data.Week[5].Present || data.Week[4].Present {
		t.Fatal("unlogged days and a day of recorded zeros must be absent")
	}
	if !data.Week[2].Over || data.Week[0].Over {
		t.Fatal("over-target day not flagged")
	}
	if data.WeekLogged != 2 || data.WeekWithin != 1 || data.WeekAvg != "2,300" || data.WeekPartial != 0 {
		t.Fatalf("logged %d within %d avg %q", data.WeekLogged, data.WeekWithin, data.WeekAvg)
	}
	if data.TargetLine <= 0 || !strings.Contains(data.WeekLabel, "not logged") || !strings.Contains(data.WeekLabel, "target 2,250") {
		t.Fatalf("label %q line %d", data.WeekLabel, data.TargetLine)
	}
	// 2250.4 of 2250 displays as 2,250 and is neither over nor outside target.
	edge := nutritionData{}
	buildNutritionWeek(&edge, []models.DailyHealthSnapshot{{Date: "2026-09-09", HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(2250.4)}}}, "2026-09-10", 2250, time.UTC)
	if edge.Week[5].Over || edge.WeekWithin != 1 {
		t.Fatal("rounding differs from the Today card")
	}
	empty := nutritionData{}
	buildNutritionWeek(&empty, nil, "2026-09-10", 0, time.UTC)
	if empty.WeekAvg != "" || empty.TargetLine != 0 || empty.WeekLogged != 0 {
		t.Fatal("empty week invented data")
	}
}

// #69: a zero-kcal day with protein logged is still a real day; a past day
// whose last sync came before 20:00 local is partial, shown hatched and left
// out of the average and the within-target count; today is never partial.
func TestNutritionWeekPartialAndZeroDays(t *testing.T) {
	loc, _ := time.LoadLocation("America/Winnipeg")
	at := func(day string, hour int) string {
		d, _ := time.ParseInLocation("2006-01-02", day, loc)
		return d.Add(time.Duration(hour) * time.Hour).UTC().Format(time.RFC3339Nano)
	}
	history := []models.DailyHealthSnapshot{
		// Complete: last sync 22:00 that day.
		{Date: "2026-09-05", LastSyncedAt: at("2026-09-05", 22), HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(2100), ProteinG: viewFloat(150)}},
		// Complete: last sync after midnight, the next day.
		{Date: "2026-09-06", LastSyncedAt: at("2026-09-07", 0), HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(1900), ProteinG: viewFloat(140)}},
		// Partial: last sync 14:00.
		{Date: "2026-09-07", LastSyncedAt: at("2026-09-07", 14), HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(800), ProteinG: viewFloat(60)}},
		// Recorded zeros: nothing logged, even though it synced.
		{Date: "2026-09-08", LastSyncedAt: at("2026-09-08", 22), HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(0), ProteinG: viewFloat(0)}},
		// 0 kcal but protein logged: a real (odd) day, kept.
		{Date: "2026-09-09", LastSyncedAt: at("2026-09-09", 23), HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(0), ProteinG: viewFloat(25)}},
		// Today, synced at 09:00: in progress, never partial.
		{Date: "2026-09-10", LastSyncedAt: at("2026-09-10", 9), HealthMetrics: models.HealthMetrics{DietaryCalories: viewFloat(400), ProteinG: viewFloat(30)}},
	}
	data := nutritionData{}
	buildNutritionWeek(&data, history, "2026-09-10", 2000, loc)
	w := data.Week
	if w[1].Partial || w[2].Partial || !w[3].Partial || w[6].Partial {
		t.Fatalf("partial flags %v %v %v %v", w[1].Partial, w[2].Partial, w[3].Partial, w[6].Partial)
	}
	if w[4].Present || !w[5].Present || w[5].Value != "0" {
		t.Fatal("recorded zeros must read nothing logged; 0 kcal with protein stays")
	}
	// Counted: 2100, 1900, 0 → avg 1,333; within: 1900 and 0 (2100 is over).
	if data.WeekLogged != 3 || data.WeekWithin != 2 || data.WeekAvg != "1,333" || data.WeekPartial != 1 {
		t.Fatalf("logged %d within %d avg %q partial %d", data.WeekLogged, data.WeekWithin, data.WeekAvg, data.WeekPartial)
	}
	if !strings.Contains(data.WeekLabel, "Mon Sep 7 800 (partial)") || !strings.Contains(data.WeekLabel, "Tue Sep 8 not logged") {
		t.Fatalf("label %q", data.WeekLabel)
	}
	data.Page = "health"
	body := renderBody(t, "nutrition.html", data)
	if !strings.Contains(body, "2 of 3 logged days within target · 1 partial day, not counted · today so far") || !strings.Contains(body, "partial") {
		t.Fatal("summary line must name the partial day")
	}
	// Only partial days: no "No food logged" claim.
	only := nutritionData{Page: "health"}
	buildNutritionWeek(&only, history[2:3], "2026-09-10", 2000, loc)
	if body := renderBody(t, "nutrition.html", only); strings.Contains(body, "No food logged in the previous six days") || !strings.Contains(body, "1 partial day, not counted · today so far") {
		t.Fatal("a partial-only week must not say nothing was logged")
	}
}
