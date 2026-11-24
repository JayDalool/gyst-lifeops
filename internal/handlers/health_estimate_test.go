package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// GYST-180 (labels): every screen that shows a day's food total says when it
// includes AI estimates, from DayIntake.Estimated alone. Fictional values.

func estCP(v float64) *float64 { return &v }

// addEstimateMeal saves one photo meal whose items keep the AI's own numbers
// untouched, so each is stored with provenance ai_estimate.
func addEstimateMeal(t *testing.T, h *Handler, key string, kcals ...float64) {
	t.Helper()
	var items []services.CameraItem
	for i, k := range kcals {
		est := &models.Nutrients{Kcal: estCP(k), ProteinG: estCP(k / 20), CarbsG: estCP(k / 10), FatG: estCP(k / 40)}
		items = append(items, services.CameraItem{Name: "Test stew " + key + string(rune('a'+i)), Grams: 100, AIEstGrams: estCP(100), AIEstNutrients: est, Nutrients: *est})
	}
	if _, err := h.Service.AddCameraMeal(context.Background(), services.DiaryActor{}, services.CameraMeal{IdempotencyKey: strings.Repeat(key, 32), MealSlot: "dinner", Items: items}); err != nil {
		t.Fatal(err)
	}
}

func estGetPage(t *testing.T, h *Handler, path string) string {
	t.Helper()
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
	if r.Code != 200 {
		t.Fatalf("GET %s: %d", path, r.Code)
	}
	return r.Body.String()
}

func estTodayView(t *testing.T, h *Handler) HealthTodayView {
	t.Helper()
	today, err := h.Service.HealthToday()
	if err != nil {
		t.Fatal(err)
	}
	return buildHealthTodayView(today, h.Service.Now())
}

func TestEstimateNoteWording(t *testing.T) {
	for _, tc := range []struct {
		e    models.EstimatedIntake
		want string
	}{
		{models.EstimatedIntake{}, ""},
		{models.EstimatedIntake{Items: 2, Kcal: estCP(410)}, "Includes AI estimates: 2 items, about 410 kcal"},
		{models.EstimatedIntake{Items: 1, Kcal: estCP(1234.5)}, "Includes AI estimates: 1 item, about 1,235 kcal"},
		{models.EstimatedIntake{Items: 3}, "Includes AI estimates: 3 items"}, // no kcal: left out, never zero
	} {
		if got := estimateNote(tc.e); got != tc.want {
			t.Errorf("%+v = %q, want %q", tc.e, got, tc.want)
		}
	}
	if estimateDaysNote(0, 7, "") != "" || estimateDaysNote(3, 7, "") != "Includes AI estimates on 3 of 7 days" ||
		estimateDaysNote(2, 5, "counted") != "Includes AI estimates on 2 of 5 days counted" {
		t.Fatal("days wording")
	}
}

// One fixture feeds every surface: the day note, the pages, the Coach and the
// brief. Adding an estimated item changes all of them together, and a day
// without estimates (or an Apple Health day) shows none.
func TestEstimateLabelComesFromOneSourceOnEverySurface(t *testing.T) {
	h, _ := diaryHandlerWithSyncedDay(t)

	// Apple Health day: never labelled.
	view := estTodayView(t, h)
	if view.Estimated.Present {
		t.Fatalf("Apple Health day labelled: %+v", view.Estimated)
	}
	for _, p := range []string{"/health/nutrition", "/health", "/today"} {
		if strings.Contains(strings.ToLower(estGetPage(t, h, p)), "includes ai estimate") {
			t.Fatalf("%s mentions AI estimates on an Apple Health day", p)
		}
	}

	// A GYST day with only database food: still none.
	addTestOats(t, h, "a", viewFloat(13), 40)
	if estTodayView(t, h).Estimated.Present {
		t.Fatal("database food labelled as estimate")
	}

	check := func(note string, items int) {
		t.Helper()
		view := estTodayView(t, h)
		if view.Estimated.Note != note || view.Estimated.Items != items {
			t.Fatalf("view note %q items %d, want %q %d", view.Estimated.Note, view.Estimated.Items, note, items)
		}
		for _, p := range []string{"/health/nutrition", "/health"} {
			body := estGetPage(t, h, p)
			if !strings.Contains(body, note) {
				t.Errorf("%s lacks %q", p, note)
			}
		}
		nut := estGetPage(t, h, "/health/nutrition")
		for _, id := range []string{`id="kcal-estimate"`, `id="macro-estimate"`, `id="diary-estimate"`} {
			if !strings.Contains(nut, id) {
				t.Errorf("Nutrition lacks %s", id)
			}
		}
		if !strings.Contains(estGetPage(t, h, "/today"), "includes AI estimates") {
			t.Error("Today tile lacks the label")
		}
		c := coachToday(HealthData{Today: view})["calories_kcal"].(map[string]any)["ai_estimated"].(map[string]any)
		if c["items"] != items || c["kcal"] != metricOrNil(view.Estimated.Kcal) {
			t.Errorf("coach %v", c)
		}
		b, err := briefOf(t, h).BriefHealth()
		if err != nil {
			t.Fatal(err)
		}
		if b.FoodAIEstimatedItems == nil || *b.FoodAIEstimatedItems != items || *b.FoodAIEstimatedKcal != view.Estimated.Kcal.Value {
			t.Errorf("brief %+v", b)
		}
	}
	addEstimateMeal(t, h, "b", 200, 240)
	check("Includes AI estimates: 2 items, about 440 kcal", 2)
	addEstimateMeal(t, h, "c", 80)
	check("Includes AI estimates: 3 items, about 520 kcal", 3)

	// The diary row says so too.
	if body := estGetPage(t, h, "/health/nutrition"); !strings.Contains(body, "AI estimate</span>") || !strings.Contains(body, "about 200 kcal") {
		t.Error("diary row not labelled")
	}
}

// The Coach and the brief get aggregates only: exactly these keys, absent when
// there is no estimate, and no food name, id, date or photo byte.
func TestEstimateCoachAndBriefGetAggregatesOnly(t *testing.T) {
	h, _ := diaryHandlerWithSyncedDay(t)
	addTestOats(t, h, "a", viewFloat(13), 40)

	none := coachToday(HealthData{Today: estTodayView(t, h)})["calories_kcal"].(map[string]any)
	if _, ok := none["ai_estimated"]; ok {
		t.Fatal("ai_estimated present with no estimates")
	}
	b, _ := briefOf(t, h).BriefHealth()
	if b.FoodAIEstimatedItems != nil || b.FoodAIEstimatedKcal != nil || b.FoodAIEstimatedProteinG != nil {
		t.Fatalf("brief has estimate fields with none: %+v", b)
	}
	baseline := keysOf(none)

	addEstimateMeal(t, h, "b", 200, 240)
	view := estTodayView(t, h)
	all := coachToday(HealthData{Today: view})
	cal := all["calories_kcal"].(map[string]any)
	if !reflect.DeepEqual(keysOf(cal), append(append([]string{}, baseline...), "ai_estimated")) && !reflect.DeepEqual(keysOf(cal), sortedWith(baseline, "ai_estimated")) {
		t.Fatalf("calories_kcal keys %v", keysOf(cal))
	}
	est := cal["ai_estimated"].(map[string]any)
	if got := keysOf(est); !reflect.DeepEqual(got, []string{"items", "kcal", "note", "protein_g"}) {
		t.Fatalf("ai_estimated keys %v", got)
	}
	if est["note"] != aiEstimatedCoachNote || est["kcal"] != "440" || est["protein_g"] != "22" {
		t.Fatalf("ai_estimated %v", est)
	}
	// Nothing else in the Coach's day changed shape.
	delete(cal, "ai_estimated")
	if !reflect.DeepEqual(keysOf(cal), baseline) {
		t.Fatalf("other calories keys moved: %v", keysOf(cal))
	}
	raw, _ := json.Marshal(all)
	bb, _ := briefOf(t, h).BriefHealth()
	rawBrief, _ := json.Marshal(bb)
	for _, s := range []string{string(raw), string(rawBrief)} {
		for _, bad := range []string{"Test stew", "photo", "base64", "nonce", "ai_est_"} {
			if strings.Contains(s, bad) {
				t.Errorf("%q leaked into context", bad)
			}
		}
		if regexp.MustCompile(`20\d\d-\d\d-\d\d`).MatchString(strings.ReplaceAll(s, view.Date, "")) {
			t.Error("a date leaked into context besides the day itself")
		}
	}
}

func keysOf(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedWith(base []string, k string) []string {
	out := append(append([]string{}, base...), k)
	sort.Strings(out)
	return out
}

func TestEstimateWeekAndPlanComparisonCountDays(t *testing.T) {
	gyst := func(date string, items int) models.DayIntake {
		return services.ResolveDayIntake(date, &models.NutritionDayTotals{Date: date, Items: 2, Sum: models.Nutrients{Kcal: estCP(1800)},
			Estimated: models.EstimatedIntake{Items: items, Kcal: estCP(400)}}, nil)
	}
	days := []models.DayIntake{gyst("2026-11-18", 1), gyst("2026-11-19", 0), gyst("2026-11-20", 2), gyst("2026-11-21", 0), gyst("2026-11-22", 0), gyst("2026-11-23", 0), gyst("2026-11-24", 1)}
	var data nutritionData
	buildNutritionWeekDays(&data, days, "2026-11-24", 2000, time.UTC)
	if data.WeekEstimateNote != "Includes AI estimates on 3 of 7 days" {
		t.Fatalf("week note %q", data.WeekEstimateNote)
	}
	var none nutritionData
	buildNutritionWeekDays(&none, days[1:2], "2026-11-24", 2000, time.UTC)
	if none.WeekEstimateNote != "" {
		t.Fatalf("week note without estimates %q", none.WeekEstimateNote)
	}
	cmp := buildCycleComparison(models.HealthCycle{ID: 1, Title: "Test", StartDate: "2026-11-18", EndDate: "2026-12-30", DailyCalorieTarget: 2000}, days, nil, "2026-11-24", time.UTC)
	// Today (11-24) and days that are not complete are not counted: 6 complete
	// days, 2 of which include estimates.
	if cmp.EstimateNote != "Includes AI estimates on 2 of 6 days counted" {
		t.Fatalf("plan note %q (days used %d)", cmp.EstimateNote, cmp.DaysUsed)
	}
}

// A new reader of DayIntake that shows or sends a total must carry the label.
// Every non-test file that reads a day's counted food has to use .Estimated, or
// be listed here with the reason it shows no total.
func TestEveryDayIntakeReaderCarriesTheEstimateLabel(t *testing.T) {
	exempt := map[string]string{
		"internal/services/nutrition_intake.go":      "the source: ResolveDayIntake fills Estimated",
		"internal/services/health.go":                "observed energy: ResolveDayIntake on a synced snapshot (no GYST items, Estimated empty) and Intake passed through to HealthTodayView",
		"internal/handlers/health_plan_reference.go": "dayIntakeSpan only fetches days; buildCycleComparison labels them",
		"internal/handlers/health_view.go":           "builds HealthTodayView.Estimated",
	}
	reader := regexp.MustCompile(`\.Counted\b|DayIntakeRange\(|ResolveDayIntake\(|\.Intake\.`)
	root := filepath.Join("..", "..")
	var bad []string
	for _, dir := range []string{"internal/handlers", "internal/services", "internal/secretary"} {
		files, _ := filepath.Glob(filepath.Join(root, dir, "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			raw, _ := os.ReadFile(f)
			rel := filepath.ToSlash(filepath.Join(dir, filepath.Base(f)))
			if !reader.Match(raw) {
				continue
			}
			if _, ok := exempt[rel]; ok || strings.Contains(string(raw), "Estimated") {
				continue
			}
			bad = append(bad, rel)
		}
	}
	if len(bad) > 0 {
		t.Fatalf("reads the day's counted food without the AI-estimate label (use DayIntake.Estimated, or list it with a reason): %v", bad)
	}
}

// Only the repository counts provenance; no handler recounts it.
func TestNoHandlerCountsEstimateProvenance(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		raw, _ := os.ReadFile(f)
		if strings.Contains(string(raw), `"ai_estimate"`) && f != "food_camera.go" {
			t.Errorf("%s names the ai_estimate provenance; counts belong to NutritionDayTotals", f)
		}
	}
}

// Review round: the checklist's measured protein row shows AI protein, so it
// says so from the same field, and the Coach's checklist tool carries the
// aggregate, counts and sums only. Only protein still equal to the estimate
// counts (the shared per-value rule, one snapshot with the day total).
func TestChecklistProteinRowAndCoachToolDiscloseAIProtein(t *testing.T) {
	h, _ := diaryHandlerWithSyncedDay(t)
	if _, err := h.Service.AddHealthChecklistItem("Protein target", "protein"); err != nil {
		t.Fatal(err)
	}
	coachList := func() map[string]any {
		out, err := briefOf(t, h).DailyChecklist()
		if err != nil {
			t.Fatal(err)
		}
		return out.(map[string]any)
	}
	if body := estGetPage(t, h, "/health"); strings.Contains(body, `id="checklist-estimate"`) {
		t.Fatal("label without estimates")
	}
	if _, ok := coachList()["ai_estimated"]; ok {
		t.Fatal("coach checklist discloses without estimates")
	}

	addEstimateMeal(t, h, "b", 200, 240) // protein 10 + 12 = 22 g, still the estimate
	body := estGetPage(t, h, "/health")
	if !strings.Contains(body, `id="checklist-estimate"`) || !strings.Contains(body, "Includes AI estimates: about 22 g protein") {
		t.Fatal("protein row not labelled")
	}
	view := estTodayView(t, h)
	if view.Estimated.ProteinNote != "Includes AI estimates: about 22 g protein" {
		t.Fatalf("note %q", view.Estimated.ProteinNote)
	}
	ai, ok := coachList()["ai_estimated"].(map[string]any)
	if !ok || !reflect.DeepEqual(keysOf(ai), []string{"items", "kcal", "note", "protein_g"}) || ai["protein_g"] != "22" || ai["note"] != aiChecklistCoachNote {
		t.Fatalf("coach checklist ai_estimated: %v", coachList())
	}
	raw, _ := json.Marshal(coachList())
	for _, bad := range []string{"Test stew", "photo\"", "nonce", "ai_est_"} {
		if strings.Contains(string(raw), bad) {
			t.Errorf("%q leaked", bad)
		}
	}
	// Same numbers as the other surfaces: one source.
	if ai["items"] != view.Estimated.Items {
		t.Fatalf("items %v vs %d", ai["items"], view.Estimated.Items)
	}
}

func TestChecklistProteinNotLabelledOnceProteinIsReEntered(t *testing.T) {
	h, _ := diaryHandlerWithSyncedDay(t)
	_, _ = h.Service.AddHealthChecklistItem("Protein target", "protein")
	est := &models.Nutrients{Kcal: estCP(200), ProteinG: estCP(10), CarbsG: estCP(20), FatG: estCP(5)}
	edited := models.Nutrients{Kcal: estCP(200), ProteinG: estCP(12), CarbsG: estCP(20), FatG: estCP(5)} // protein typed over
	if _, err := h.Service.AddCameraMeal(context.Background(), services.DiaryActor{}, services.CameraMeal{IdempotencyKey: strings.Repeat("e", 32), MealSlot: "dinner",
		Items: []services.CameraItem{{Name: "Test stew", Grams: 100, AIEstGrams: estCP(100), AIEstNutrients: est, Nutrients: edited}}}); err != nil {
		t.Fatal(err)
	}
	if body := estGetPage(t, h, "/health"); strings.Contains(body, `id="checklist-estimate"`) {
		t.Fatal("re-entered protein still labelled as estimate")
	}
	if _, ok := mustChecklist(t, h)["ai_estimated"]; ok {
		t.Fatal("coach discloses re-entered protein")
	}
}

func mustChecklist(t *testing.T, h *Handler) map[string]any {
	t.Helper()
	out, err := briefOf(t, h).DailyChecklist()
	if err != nil {
		t.Fatal(err)
	}
	return out.(map[string]any)
}
