package handlers

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func refPostForm() url.Values {
	return url.Values{"source_label": {"Example dietitian"}, "source_kind": {"dietitian"}, "issued_on": {"2026-11-01"}, "kcal": {"1900"}, "protein_g": {"120"}}
}

func refHandlerWithCycle(t *testing.T) (*Handler, models.HealthCycle) {
	t.Helper()
	h := testHandler(t)
	h.Service.Now = func() time.Time { return time.Date(2026, 11, 16, 18, 0, 0, 0, time.UTC) }
	c := models.HealthCycle{Title: "Autumn", PrimaryGoal: "Consistency", StartDate: "2026-11-01", EndDate: "2026-12-31", DailyCalorieTarget: 2300, ProteinTargetG: 180, FiberTargetG: 30}
	if err := h.Service.CreateHealthCycle(&c); err != nil {
		t.Fatal(err)
	}
	return h, c
}

func TestHealthReferenceTargetsHandlers(t *testing.T) {
	h, c := refHandlerWithCycle(t)
	before, _ := h.Service.ListHealthCycles()

	// Cross-origin and wrong-method requests are rejected.
	if w := cfgPost(h, "/health/plan/reference", refPostForm(), "https://untrusted.example"); w.Code != 403 {
		t.Fatalf("cross-origin add: %d", w.Code)
	}
	if w := cfgPost(h, "/health/plan/reference/1/archive", url.Values{}, "https://untrusted.example"); w.Code != 403 {
		t.Fatalf("cross-origin archive: %d", w.Code)
	}
	if w := healthRequest(h, "GET", "/health/plan/reference", "", "", ""); w.Code != 405 {
		t.Fatalf("GET add: %d", w.Code)
	}
	if w := healthRequest(h, "GET", "/health/plan/reference/1/archive", "", "", ""); w.Code != 405 {
		t.Fatalf("GET archive: %d", w.Code)
	}
	if list, _ := h.Service.ListHealthReferenceTargets(); len(list) != 0 {
		t.Fatal("a rejected request stored a target")
	}

	// Invalid input is shown again with the typed values.
	bad := refPostForm()
	bad.Set("protein_g", "-5")
	w := cfgPost(h, "/health/plan/reference", bad, "")
	if w.Code != 400 || !strings.Contains(w.Body.String(), `role="alert"`) || !strings.Contains(w.Body.String(), `value="Example dietitian"`) {
		t.Fatalf("invalid add: %d", w.Code)
	}

	// A valid add stores the target; blank numbers render as an em dash.
	if w := cfgPost(h, "/health/plan/reference", refPostForm(), ""); w.Code != 303 {
		t.Fatalf("add: %d %s", w.Code, w.Body.String())
	}
	page := cfgGet(t, h, "/health/plan")
	for _, want := range []string{"Reference targets (not active)", "Example dietitian", "1900 kcal", "120 g", "<dd>—</dd>", "Cycle comparison"} {
		if !strings.Contains(page, want) {
			t.Fatalf("plan page lacks %q", want)
		}
	}
	if strings.Contains(page, "<dd>0 g</dd>") || strings.Contains(page, "<dd>0 kcal</dd>") {
		t.Fatal("missing value rendered as zero")
	}
	after, _ := h.Service.ListHealthCycles()
	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatal("adding a reference target changed health_cycles")
	}
	if got, _ := h.Service.GetHealthCycle(c.ID); got.DailyCalorieTarget != 2300 {
		t.Fatal("active target changed")
	}

	// Archive hides it without deleting; a second archive is 404.
	list, _ := h.Service.ListHealthReferenceTargets()
	path := fmt.Sprintf("/health/plan/reference/%d/archive", list[0].ID)
	if w := cfgPost(h, path, url.Values{}, ""); w.Code != 303 {
		t.Fatalf("archive: %d", w.Code)
	}
	if page := cfgGet(t, h, "/health/plan"); strings.Contains(page, "<strong>Example dietitian</strong>") || !strings.Contains(page, "No reference targets yet.") {
		t.Fatal("archived target still shown")
	}
	if w := cfgPost(h, path, url.Values{}, ""); w.Code != 404 {
		t.Fatalf("second archive: %d", w.Code)
	}

	// A too-long note of four-byte characters still fits the body cap, so the
	// user sees the message rather than a bare 400.
	long := refPostForm()
	long.Set("source_label", strings.Repeat("😀", 200))
	long.Set("notes", strings.Repeat("😀", 401))
	if w := cfgPost(h, "/health/plan/reference", long, ""); w.Code != 400 || !strings.Contains(w.Body.String(), "at most 400 characters") {
		t.Fatalf("long note: %d", w.Code)
	}
	long.Set("notes", strings.Repeat("😀", 400))
	if w := cfgPost(h, "/health/plan/reference", long, ""); w.Code != 303 {
		t.Fatalf("max-size note rejected: %d", w.Code)
	}
}

func refSnap(date string, kcal, protein, fiber *float64, synced string) models.DailyHealthSnapshot {
	return models.DailyHealthSnapshot{Date: date, HealthMetrics: models.HealthMetrics{DietaryCalories: kcal, ProteinG: protein, FiberG: fiber}, LastSyncedAt: synced}
}

func TestCycleComparisonUsesCompleteDaysOnly(t *testing.T) {
	c := models.HealthCycle{ID: 1, Title: "Autumn", StartDate: "2026-11-01", EndDate: "2026-11-30", DailyCalorieTarget: 2000, ProteinTargetG: 150, FiberTargetG: 30}
	f := viewFloat
	days := []models.DailyHealthSnapshot{
		refSnap("2026-11-01", f(2000), f(100), f(20), "2026-11-01T21:00:00Z"), // complete
		refSnap("2026-11-02", f(2200), f(120), nil, "2026-11-02T22:00:00Z"),   // complete, no fibre
		refSnap("2026-11-03", f(900), f(40), f(5), "2026-11-03T10:00:00Z"),    // partial: synced before 20:00
		refSnap("2026-11-04", f(0), nil, nil, "2026-11-04T21:00:00Z"),         // recorded zeros: nothing logged
		refSnap("2026-11-05", nil, nil, nil, "2026-11-05T21:00:00Z"),          // no dietary observation
		refSnap("2026-11-10", f(3000), f(200), f(40), "2026-11-10T09:00:00Z"), // today: never complete
	}
	got := buildCycleComparison(c, days, nil, "2026-11-10", time.UTC)
	if got.DaysUsed != 2 || got.DaysText != "2 complete days counted" {
		t.Fatalf("days: %+v", got)
	}
	if got.AvgKcal != "2,100 kcal" || got.AvgProtein != "110 g" || got.AvgFiber != "20 g" {
		t.Fatalf("averages: %+v", got)
	}
	if got.TargetKcal != "2,000 kcal" || got.TargetProtein != "150 g" || got.TargetFiber != "30 g" {
		t.Fatalf("targets: %+v", got)
	}
	empty := buildCycleComparison(c, nil, nil, "2026-11-10", time.UTC)
	if empty.DaysUsed != 0 || empty.AvgKcal != "—" || empty.AvgProtein != "—" || empty.AvgFiber != "—" || empty.DaysText != "No complete days counted" {
		t.Fatalf("empty cycle must show dashes, not zeros: %+v", empty)
	}
	if empty.WeightStart != "—" || empty.WeightEnd != "—" {
		t.Fatalf("no readings must show dashes: %+v", empty)
	}
}

func TestCycleComparisonUnsetTargetsShowDash(t *testing.T) {
	c := models.HealthCycle{ID: 1, Title: "New", StartDate: "2026-11-01", EndDate: "2026-11-30"} // all targets 0 = not set
	got := buildCycleComparison(c, nil, nil, "2026-11-10", time.UTC)
	if got.TargetKcal != "—" || got.TargetProtein != "—" || got.TargetFiber != "—" {
		t.Fatalf("a target of 0 must render as a dash: %+v", got)
	}
}

func refWeights(t *testing.T, readings map[string]float64, now time.Time) map[string]*localWeightDay {
	t.Helper()
	var entries []models.HealthWeightEntry
	for date, kg := range readings {
		entries = append(entries, models.HealthWeightEntry{MeasuredAt: date + "T12:00:00Z", WeightKg: kg, Source: "scale"})
	}
	return groupWeightDays(entries, now, time.UTC)
}

func TestCycleComparisonWeightWindows(t *testing.T) {
	c := models.HealthCycle{ID: 1, Title: "Autumn", StartDate: "2026-11-01", EndDate: "2026-11-30"}
	now := time.Date(2026, 11, 10, 18, 0, 0, 0, time.UTC)
	// Start window (Nov 1-7) has 4 days. End window (Nov 4-10, today while the
	// cycle runs) has only Nov 4 and Nov 9: too few, so a dash.
	w := refWeights(t, map[string]float64{"2026-11-01": 80, "2026-11-02": 80.4, "2026-11-03": 80, "2026-11-04": 80.4, "2026-11-09": 79}, now)
	got := buildCycleComparison(c, nil, w, "2026-11-10", time.UTC)
	if got.WeightStart != "80.2 kg · 4 days" || got.WeightEnd != "—" {
		t.Fatalf("weights: %+v", got)
	}
	// Add readings so the end window qualifies (Nov 4, 8, 9, 10).
	w = refWeights(t, map[string]float64{"2026-11-01": 80, "2026-11-02": 80.4, "2026-11-03": 80, "2026-11-04": 80.4, "2026-11-08": 79, "2026-11-09": 79, "2026-11-10": 79.2}, now)
	got = buildCycleComparison(c, nil, w, "2026-11-10", time.UTC)
	if got.WeightEnd != "79.4 kg · 4 days" {
		t.Fatalf("end window: %+v", got)
	}
	// A finished cycle ends on its own last day, not today.
	done := models.HealthCycle{ID: 2, Title: "Done", StartDate: "2026-10-01", EndDate: "2026-10-31"}
	w = refWeights(t, map[string]float64{"2026-10-01": 85, "2026-10-02": 85, "2026-10-03": 85, "2026-10-04": 85, "2026-10-28": 83, "2026-10-29": 83, "2026-10-30": 83, "2026-10-31": 83}, now)
	got = buildCycleComparison(done, nil, w, "2026-11-10", time.UTC)
	if got.WeightStart != "85 kg · 4 days" || got.WeightEnd != "83 kg · 4 days" {
		t.Fatalf("finished cycle: %+v", got)
	}
}

func TestCoachGetHealthPlanLabelsReferenceTargets(t *testing.T) {
	h, c := refHandlerWithCycle(t)
	other := models.HealthCycle{Title: "Old", PrimaryGoal: "x", StartDate: "2026-01-01", EndDate: "2026-02-01"}
	if err := h.Service.CreateHealthCycle(&other); err != nil {
		t.Fatal(err)
	}
	for _, in := range []models.HealthReferenceTarget{
		{SourceLabel: "Example dietitian", SourceKind: "dietitian", Kcal: viewFloat(1900), ProteinG: viewFloat(120)},
		{CycleID: &c.ID, SourceLabel: "Example device", SourceKind: "device"},
		{CycleID: &other.ID, SourceLabel: "Old cycle source", SourceKind: "other"},
	} {
		in := in
		if err := h.Service.CreateHealthReferenceTarget(&in); err != nil {
			t.Fatal(err)
		}
	}
	data := HealthData{}
	data.Today.Date = "2026-11-16"
	var action string
	var out string
	for _, tool := range h.coachTools(data, &action) {
		if tool.Name == "get_health_plan" {
			var err error
			if out, err = tool.Run(context.Background(), nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, want := range []string{`"reference_targets"`, `"not_active":true`, `"source_label":"Example dietitian"`, `"source_kind":"dietitian"`, `"source_label":"Example device"`, `"calories":1900`, `"carbs_g":null`, `"calories":2300`} {
		if !strings.Contains(out, want) {
			t.Fatalf("coach plan output lacks %s: %s", want, out)
		}
	}
	if strings.Contains(out, "Old cycle source") {
		t.Fatal("a reference target for another cycle leaked into the plan")
	}
}
