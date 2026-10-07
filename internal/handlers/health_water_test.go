package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

// Water (#60): bottle taps, the 3D box and the measured checklist row.

func TestLitres(t *testing.T) {
	for ml, want := range map[int64]string{1500: "1.5", 3000: "3.0", 355: "0.355", 0: "0.0", 10000: "10.0"} {
		if got := litres(ml); got != want {
			t.Errorf("litres(%d) = %q, want %q", ml, got, want)
		}
	}
}

func TestBuildHealthWater(t *testing.T) {
	plan := &models.HealthCycle{WaterTargetML: 3000, WaterBottleML: 1500}
	for _, tc := range []struct {
		name          string
		cycle         *models.HealthCycle
		total         int64
		taps          int
		pct           int
		done, logged  bool
		left, total_s string
		ticks         int
	}{
		{"no plan", nil, 0, 0, 0, false, false, "", "0.0", 0},
		{"no target", &models.HealthCycle{WaterBottleML: 1500}, 0, 0, 0, false, false, "", "0.0", 0},
		{"nothing yet", plan, 0, 0, 0, false, false, "3.0 L to go", "0.0", 1},
		{"one bottle is half", plan, 1500, 1, 50, false, true, "1.5 L to go", "1.5", 1},
		{"two bottles is full", plan, 3000, 2, 100, true, true, "", "3.0", 1},
		{"over target (target lowered later) stays full", plan, 4500, 3, 100, true, true, "", "4.5", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := buildHealthWater(tc.cycle, tc.total, tc.taps, -1)
			if v.Percent != tc.pct || v.Done != tc.done || v.Logged != tc.logged || v.Left != tc.left || v.Total != tc.total_s || len(v.Ticks) != tc.ticks || v.FromPercent != -1 {
				t.Fatalf("%+v", v)
			}
		})
	}
	v := buildHealthWater(plan, 3000, 2, 1500)
	if v.FromPercent != 50 || v.Tappable || v.Level() != "1.00" || v.FromLevel() != "0.50" || v.BottlesNeeded != 2 || v.Ticks[0] != 50 {
		t.Fatalf("animation/ticks %+v", v)
	}
	// Bottle marks in SVG units: 50% is halfway between y 342 and y 80.
	if v.TickY(50) != "211.0" || v.TickY(100) != "80.0" || v.TickY(0) != "342.0" {
		t.Fatalf("tick y %s", v.TickY(50))
	}
	// A 0.75 L bottle against 3.0 L needs 4 bottles, marks at 25/50/75%.
	v = buildHealthWater(&models.HealthCycle{WaterTargetML: 3000, WaterBottleML: 750}, 0, 0, -1)
	if v.BottlesNeeded != 4 || fmt.Sprint(v.Ticks) != "[25 50 75]" {
		t.Fatalf("small bottle %+v", v)
	}
}

func TestWaterQueryIsBounded(t *testing.T) {
	for raw, want := range map[string]int64{"water=added&wfrom=1500": 1500, "water=added&wfrom=99999": -1, "water=added&wfrom=-5": -1, "wfrom=1500": -1, "water=nobottle&wfrom=1500": -1} {
		q, _ := url.ParseQuery(raw)
		if got, _, _ := waterQuery(q); got != want {
			t.Errorf("%s: from %d, want %d", raw, got, want)
		}
	}
}

func TestParseLitres(t *testing.T) {
	for raw, want := range map[string]int64{"": 0, " 3 ": 3000, "1.5": 1500, "1,5": 1500, "0.355": 355} {
		if got, err := parseLitres(raw, "x"); err != nil || got != want {
			t.Errorf("%q: %d %v", raw, got, err)
		}
	}
	for _, raw := range []string{"abc", "-1", "NaN", "Inf"} {
		if _, err := parseLitres(raw, "x"); err == nil {
			t.Errorf("%q accepted", raw)
		}
	}
}

func waterHandler(t *testing.T, target, bottle string) (*Handler, *time.Time) {
	t.Helper()
	h := testHandler(t)
	now := time.Date(2026, 10, 7, 15, 0, 0, 0, h.Service.Location())
	h.Service.Clock.Now = func() time.Time { return now }
	form := healthPlanSubmission()
	form.Set("start_date", "2026-10-01")
	form.Set("end_date", "2027-03-31")
	form.Set("water_target_l", target)
	form.Set("water_bottle_l", bottle)
	if w := cfgPost(h, "/health/plan", form, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("plan: %d %s", w.Code, w.Body.String())
	}
	return h, &now
}

func TestWaterPlanFields(t *testing.T) {
	h, _ := waterHandler(t, "3,0", "1.5")
	cycles, _ := h.Service.ListHealthCycles()
	if cycles[0].WaterTargetML != 3000 || cycles[0].WaterBottleML != 1500 {
		t.Fatalf("saved %+v", cycles[0])
	}
	if body := cfgGet(t, h, "/health/plan"); !strings.Contains(body, `name="water_target_l"`) || !strings.Contains(body, `value="3.0"`) || !strings.Contains(body, `value="1.5"`) {
		t.Fatal("plan form doesn't show the water fields")
	}
	for _, bad := range []struct{ field, value string }{{"water_target_l", "0.2"}, {"water_target_l", "12"}, {"water_bottle_l", "5"}, {"water_bottle_l", "lots"}} {
		form := healthPlanSubmission()
		form.Set("cycle_id", fmt.Sprint(cycles[0].ID))
		form.Set("start_date", "2026-10-01")
		form.Set("end_date", "2027-03-31")
		form.Set(bad.field, bad.value)
		if w := cfgPost(h, "/health/plan", form, ""); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `role="alert"`) {
			t.Errorf("%s=%s: %d", bad.field, bad.value, w.Code)
		}
	}
}

func TestWaterBottleFlow(t *testing.T) {
	h, now := waterHandler(t, "3", "1.5")
	body := cfgGet(t, h, "/health")
	if !strings.Contains(body, `<form class="hv3-wbtl-form" method="post" action="/health/water">`) {
		t.Fatal("the bottle must be the submit button of the /health/water form")
	}
	if !strings.Contains(body, "No water logged yet") || !strings.Contains(body, "0 of 2 bottles") || !strings.Contains(body, "Tap the bottle to add 1.5 L") || !strings.Contains(body, `aria-label="Add a 1.5 L bottle. Today: nothing logged yet, target 3.0 L"`) || strings.Contains(body, "Undo last bottle") || strings.Contains(body, "0.0<small>") {
		t.Fatal("empty water state wrong")
	}
	w := cfgPost(h, "/health/water", nil, "")
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/health?water=added&wfrom=0#water" {
		t.Fatalf("first bottle: %d %s", w.Code, w.Header().Get("Location"))
	}
	body = cfgGet(t, h, "/health?water=added&wfrom=0")
	if !strings.Contains(body, "1.5<small> / 3.0 L</small>") || !strings.Contains(body, "1.5 L to go · 1 of 2 bottles") || !strings.Contains(body, "hv3-wbtl rise") || !strings.Contains(body, "--lvl:0.50;--from:0.00;") || !strings.Contains(body, "Bottle added") {
		t.Fatal("one bottle should fill half and animate from empty")
	}
	if w := cfgPost(h, "/health/water", nil, ""); w.Header().Get("Location") != "/health?water=added&wfrom=1500#water" {
		t.Fatalf("second bottle: %s", w.Header().Get("Location"))
	}
	body = cfgGet(t, h, "/health")
	if !strings.Contains(body, "Target reached") || !strings.Contains(body, "2 of 2 bottles") || !strings.Contains(body, "hv3-wbtl static full") || strings.Contains(body, " rise") {
		t.Fatal("two bottles should fill the bottle, without animating on a plain load")
	}
	// #78: at the target the bottle is no longer a button, and the server
	// refuses a stale tap.
	if strings.Contains(body, "hv3-wbtl-form") || strings.Contains(body, "Tap the bottle") || !strings.Contains(body, "Done for today · resets at midnight") || !strings.Contains(body, `aria-label="Water bottle full: 3.0 of 3.0 L, target reached"`) || !strings.Contains(body, "Undo last bottle") {
		t.Fatal("a full bottle must not be tappable; Undo stays")
	}
	if w := cfgPost(h, "/health/water", nil, ""); w.Header().Get("Location") != "/health?water=full#water" {
		t.Fatalf("tap past the target: %s", w.Header().Get("Location"))
	}
	if total, taps, _ := h.Service.Repo.HealthWaterDay(h.Service.TodayString()); total != 3000 || taps != 2 {
		t.Fatalf("stored %d in %d taps past the target", total, taps)
	}
	if body := cfgGet(t, h, "/health?water=full"); !strings.Contains(body, "Today&#39;s water target is reached") {
		t.Fatal("refused tap should say why")
	}
	// Undo removes the last bottle, animates down from full, and the bottle
	// is tappable again.
	if w := cfgPost(h, "/health/water/undo", nil, ""); w.Header().Get("Location") != "/health?water=undone&wfrom=3000#water" {
		t.Fatalf("undo: %s", w.Header().Get("Location"))
	}
	if body := cfgGet(t, h, "/health"); !strings.Contains(body, "hv3-wbtl-form") {
		t.Fatal("below the target the bottle is a button again")
	}
	cfgPost(h, "/health/water/undo", nil, "")
	if w := cfgPost(h, "/health/water/undo", nil, ""); w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/health#water" {
		t.Fatalf("undo with nothing logged: %d %s", w.Code, w.Header().Get("Location"))
	}
	// A new local day starts empty. 20:00 local is already the next day in
	// UTC, so this fails if days were counted in UTC.
	*now = time.Date(2026, 10, 7, 20, 0, 0, 0, h.Service.Location())
	cfgPost(h, "/health/water", nil, "")
	*now = now.Add(5 * time.Hour)
	if body := cfgGet(t, h, "/health"); !strings.Contains(body, "No water logged yet") {
		t.Fatal("yesterday's bottle leaked into today")
	}
}

// #78: when the bottle doesn't divide the target, the tap that reaches or
// passes it is the last one.
func TestWaterTargetUnevenBottle(t *testing.T) {
	h, _ := waterHandler(t, "2.5", "1")
	for i, want := range []string{"added", "added", "added", "full"} {
		w := cfgPost(h, "/health/water", nil, "")
		if !strings.Contains(w.Header().Get("Location"), "water="+want) {
			t.Fatalf("tap %d: %s, want %s", i+1, w.Header().Get("Location"), want)
		}
	}
	if total, taps, _ := h.Service.Repo.HealthWaterDay(h.Service.TodayString()); total != 3000 || taps != 3 {
		t.Fatalf("stored %d in %d taps", total, taps)
	}
}

// #78: quick simultaneous taps can't pass the target.
func TestWaterTargetConcurrentTaps(t *testing.T) {
	h, _ := waterHandler(t, "3", "1.5")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); cfgPost(h, "/health/water", nil, "") }()
	}
	wg.Wait()
	if total, taps, _ := h.Service.Repo.HealthWaterDay(h.Service.TodayString()); total != 3000 || taps != 2 {
		t.Fatalf("8 simultaneous taps stored %d in %d taps", total, taps)
	}
}

func TestWaterNeedsPlanAndGuards(t *testing.T) {
	h := testHandler(t)
	if body := cfgGet(t, h, "/health"); !strings.Contains(body, "Set a daily water target and your bottle size") || strings.Contains(body, "hv3-wbtl-form") || !strings.Contains(body, "Empty water bottle, nothing logged yet") {
		t.Fatal("no plan: should point to Plan, bottle not tappable")
	}
	if w := cfgPost(h, "/health/water", nil, ""); w.Header().Get("Location") != "/health?water=nobottle#water" {
		t.Fatalf("no plan tap: %s", w.Header().Get("Location"))
	}
	if body := cfgGet(t, h, "/health?water=nobottle"); !strings.Contains(body, "Set your water target and bottle size in Health → Plan first") {
		t.Fatal("no-bottle message missing")
	}
	if total, taps, _ := h.Service.Repo.HealthWaterDay(h.Service.TodayString()); total != 0 || taps != 0 {
		t.Fatal("tap without a plan was stored")
	}
	// Target without a bottle: amount shown, no button.
	h2, _ := waterHandler(t, "3", "")
	if body := cfgGet(t, h2, "/health"); !strings.Contains(body, "Set your bottle size in") || strings.Contains(body, "hv3-wbtl-form") {
		t.Fatal("target without bottle")
	}
	for _, path := range []string{"/health/water", "/health/water/undo"} {
		w := httptestGet(h2, path)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET %s: %d", path, w.Code)
		}
		if w := cfgPost(h2, path, nil, "https://evil.example"); w.Code != http.StatusForbidden {
			t.Errorf("cross-origin %s: %d", path, w.Code)
		}
	}
}

func TestWaterChecklistRow(t *testing.T) {
	h, _ := waterHandler(t, "3", "1.5")
	if w := cfgPost(h, "/health/checklist", url.Values{"name": {"Water"}, "kind": {"water"}}, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("add water row: %d", w.Code)
	}
	settings := cfgGet(t, h, "/health/checklist")
	if !strings.Contains(settings, "Measured · water bottles logged vs your plan") || strings.Contains(settings, `value="water"`) {
		t.Fatal("settings should list water once and stop offering it")
	}
	body := cfgGet(t, h, "/health")
	if !strings.Contains(body, "0 of 1 complete") || !strings.Contains(body, "— / 3.0 L") {
		t.Fatal("water row should be measured and not done")
	}
	cfgPost(h, "/health/water", nil, "")
	cfgPost(h, "/health/water", nil, "")
	if body := cfgGet(t, h, "/health"); !strings.Contains(body, "1 of 1 complete") || !strings.Contains(body, "3.0 / 3.0 L") {
		t.Fatal("water row should be done at the target")
	}
	if w := cfgPost(h, "/health/checklist/water/remove", nil, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("remove: %d", w.Code)
	}
	if body := cfgGet(t, h, "/health"); strings.Contains(body, "3.0 / 3.0 L") {
		t.Fatal("removed water row still on Today")
	}
	if w := cfgPost(h, "/health/checklist/water/toggle", nil, ""); w.Code != http.StatusNotFound {
		t.Fatalf("water can't be ticked: %d", w.Code)
	}
	if w := cfgPost(h, "/health/checklist", url.Values{"name": {" "}, "kind": {"water"}}, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("blank water name: %d", w.Code)
	}
}

func httptestGet(h *Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}
