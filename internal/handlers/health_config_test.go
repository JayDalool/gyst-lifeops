package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func cfgPost(h *Handler, path string, form url.Values, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func cfgGet(t *testing.T, h *Handler, path string) string {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s: %d", path, w.Code)
	}
	return w.Body.String()
}

// Health Today tells app.js the instant its day ends, from GYST's pinned
// zone data, so the day reload never depends on the device's zone rules
// (#193). 00:30 on 2026-11-02 in Winnipeg: the day ends at 05:00Z, UTC-05.
func TestHealthTodaySendsDayEnd(t *testing.T) {
	h := testHandler(t)
	h.Service.Clock.Now = func() time.Time { return time.Date(2026, 11, 2, 5, 30, 0, 0, time.UTC) }
	body := cfgGet(t, h, "/health")
	if want := `data-day-page="2026-11-02" data-day-tz="America/Winnipeg" data-day-ends="2026-11-03T05:00:00Z"`; !strings.Contains(body, want) {
		t.Fatalf("Health Today lacks %s", want)
	}
}

func TestChecklistLifecycle(t *testing.T) {
	h := testHandler(t)
	loc := h.Service.Location()
	day := time.Date(2026, 10, 7, 9, 0, 0, 0, loc)
	h.Service.Clock.Now = func() time.Time { return day }

	if body := cfgGet(t, h, "/health"); !strings.Contains(body, "Add your own daily habits") {
		t.Fatal("Today should show the empty checklist state")
	}
	if body := cfgGet(t, h, "/health/checklist"); !strings.Contains(body, "No items yet") || !strings.Contains(body, `value="protein"`) {
		t.Fatal("empty settings page or missing protein option")
	}
	for _, f := range []url.Values{{"name": {"Creatine"}, "kind": {"habit"}}, {"name": {"<b>Mag</b>nesium"}, "kind": {"habit"}}, {"name": {"Protein"}, "kind": {"protein"}}} {
		if w := cfgPost(h, "/health/checklist", f, "http://example.com"); w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/health/checklist" {
			t.Fatalf("add %v: %d", f, w.Code)
		}
	}
	items, _ := h.Service.Repo.ListHealthChecklistItems("2026-10-07")
	if len(items) != 3 || items[0].Name != "Creatine" || items[2].Kind != "protein" {
		t.Fatalf("items %+v", items)
	}
	settings := cfgGet(t, h, "/health/checklist")
	if !strings.Contains(settings, "&lt;b&gt;Mag&lt;/b&gt;nesium") || strings.Contains(settings, "<b>Mag</b>") {
		t.Fatal("item name not escaped")
	}
	if strings.Contains(settings, `value="protein"`) {
		t.Fatal("a second protein target should not be offered")
	}
	if w := cfgPost(h, "/health/checklist", url.Values{"name": {"Protein again"}, "kind": {"protein"}}, ""); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "already have a protein target") {
		t.Fatalf("second protein item: %d", w.Code)
	}

	// Toggle a habit for today, then undo; the measured row can't be toggled.
	creatine, protein := idString(items[0].ID), idString(items[2].ID)
	if w := cfgPost(h, "/health/checklist/"+creatine+"/toggle", nil, ""); w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/health#checklist" {
		t.Fatalf("toggle: %d %q", w.Code, w.Header().Get("Location"))
	}
	if body := cfgGet(t, h, "/health"); !strings.Contains(body, `aria-pressed="true" aria-label="Creatine: done, tap to undo"`) {
		t.Fatal("habit not shown as done")
	}
	// The next local day starts unticked.
	h.Service.Clock.Now = func() time.Time { return day.AddDate(0, 0, 1) }
	if body := cfgGet(t, h, "/health"); !strings.Contains(body, `aria-label="Creatine: not done, tap to mark done"`) {
		t.Fatal("tick leaked into the next day")
	}
	h.Service.Clock.Now = func() time.Time { return day }
	cfgPost(h, "/health/checklist/"+creatine+"/toggle", nil, "")
	if body := cfgGet(t, h, "/health"); !strings.Contains(body, `aria-label="Creatine: not done, tap to mark done"`) {
		t.Fatal("second toggle should undo")
	}
	for _, path := range []string{"/health/checklist/" + protein + "/toggle", "/health/checklist/999/toggle", "/health/checklist/x/toggle", "/health/checklist/" + creatine + "/rename"} {
		if w := cfgPost(h, path, nil, ""); w.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}

	// Remove hides the item everywhere; removing again is 404.
	if w := cfgPost(h, "/health/checklist/"+creatine+"/remove", nil, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("remove: %d", w.Code)
	}
	if body := cfgGet(t, h, "/health"); strings.Contains(body, "Creatine") {
		t.Fatal("removed item still on Today")
	}
	if w := cfgPost(h, "/health/checklist/"+creatine+"/remove", nil, ""); w.Code != http.StatusNotFound {
		t.Fatalf("second remove: %d", w.Code)
	}
}

func TestChecklistValidationAndGuards(t *testing.T) {
	h := testHandler(t)
	for _, f := range []url.Values{{"name": {"  "}, "kind": {"habit"}}, {"name": {strings.Repeat("x", 81)}, "kind": {"habit"}}, {"name": {"Water"}, "kind": {"steps"}}} {
		w := cfgPost(h, "/health/checklist", f, "")
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `role="alert"`) {
			t.Fatalf("%v: %d", f, w.Code)
		}
	}
	if w := cfgPost(h, "/health/checklist", url.Values{"name": {"Water"}, "kind": {"steps"}}, ""); !strings.Contains(w.Body.String(), `value="Water"`) {
		t.Fatal("typed name not kept")
	}
	if w := cfgPost(h, "/health/checklist", url.Values{"name": {"Creatine"}, "kind": {"habit"}}, "https://evil.example"); w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin: %d", w.Code)
	}
	if items, _ := h.Service.Repo.ListHealthChecklistItems("2026-10-07"); len(items) != 0 {
		t.Fatalf("rejected input stored %+v", items)
	}
	for _, tc := range []struct{ method, path string }{{"GET", "/health/checklist/1/toggle"}, {"GET", "/health/checklist/1/remove"}, {"PUT", "/health/checklist"}, {"DELETE", "/health/foods"}, {"PUT", "/health/foods/1"}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s: %d", tc.method, tc.path, w.Code)
		}
	}
	for _, path := range []string{"/health/checklist", "/health/checklist/1/toggle", "/health/foods", "/health/foods/1"} {
		if authExemptPath(path) {
			t.Fatalf("%s must require a session", path)
		}
	}
}

func TestChecklistProteinRowIsMeasured(t *testing.T) {
	cycle := viewCycle() // protein target 180
	items := []models.HealthChecklistItem{{ID: 1, Name: "Protein", Kind: "protein"}, {ID: 2, Name: "Creatine", Kind: "habit", DoneToday: true}}
	for _, tc := range []struct {
		protein *float64
		value   string
		done    bool
	}{{nil, "— / 180 g", false}, {viewFloat(132), "132 / 180 g", false}, {viewFloat(181), "181 / 180 g", true}} {
		snap := models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{ProteinG: tc.protein}}
		rows := buildChecklistRows(items, buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &snap, HealthCycle: &cycle}, viewNow()))
		if !rows[0].Measured || rows[0].Habit || rows[0].Value != tc.value || rows[0].Done != tc.done || !rows[1].Done || !rows[1].Habit {
			t.Fatalf("protein %v: %+v", tc.protein, rows)
		}
	}
	// No plan: the value has no target and is never "met".
	rows := buildChecklistRows(items[:1], buildHealthTodayView(models.HealthTodayResponse{Date: "2026-09-10", Snapshot: &models.DailyHealthSnapshot{HealthMetrics: models.HealthMetrics{ProteinG: viewFloat(200)}}}, viewNow()))
	if rows[0].Value != "200 g" || rows[0].Done {
		t.Fatalf("no plan: %+v", rows[0])
	}
}

func TestFoodsLifecycle(t *testing.T) {
	h := testHandler(t)
	if body := cfgGet(t, h, "/health/foods"); !strings.Contains(body, "No foods yet") {
		t.Fatal("empty state missing")
	}
	add := url.Values{"name": {"Greek yogurt"}, "serving": {"175 g pot"}, "kcal": {"105"}, "protein": {"18,5"}, "carbs": {""}, "notes": {"<script>alert(1)</script>"}}
	if w := cfgPost(h, "/health/foods", add, "http://example.com"); w.Code != http.StatusSeeOther {
		t.Fatalf("add: %d %s", w.Code, w.Body.String())
	}
	foods, _ := h.Service.Repo.ListHealthFoods(true)
	if len(foods) != 1 || *foods[0].ProteinG != 18.5 || foods[0].CarbsG != nil || !foods[0].Active {
		t.Fatalf("stored %+v", foods)
	}
	list := cfgGet(t, h, "/health/foods")
	if !strings.Contains(list, "175 g pot · 105 kcal · 18.5 g protein") {
		t.Fatal("food meta missing")
	}
	id := idString(foods[0].ID)
	edit := cfgGet(t, h, "/health/foods/"+id)
	if !strings.Contains(edit, "&lt;script&gt;alert(1)&lt;/script&gt;") || strings.Contains(edit, "<script>alert(1)") {
		t.Fatal("notes not escaped")
	}
	if !strings.Contains(edit, `name="carbs" inputmode="decimal" autocomplete="off" value=""`) {
		t.Fatal("unknown value should stay blank, not 0")
	}
	upd := url.Values{"name": {"Greek yogurt 0%"}, "serving": {"175 g pot"}, "kcal": {"100"}, "active": {"0"}}
	if w := cfgPost(h, "/health/foods/"+id, upd, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("edit: %d", w.Code)
	}
	if list := cfgGet(t, h, "/health/foods"); !strings.Contains(list, "Greek yogurt 0%") || !strings.Contains(list, "· inactive") {
		t.Fatal("inactive food not marked")
	}
	if active, _ := h.Service.Repo.ListHealthFoods(false); len(active) != 0 {
		t.Fatal("inactive food listed as active")
	}
	for _, bad := range []url.Values{{"name": {""}}, {"name": {"X"}, "kcal": {"-1"}}, {"name": {"X"}, "protein": {"abc"}}, {"name": {"X"}, "fat": {"1001"}}, {"name": {strings.Repeat("y", 121)}}, {"name": {"X"}, "notes": {strings.Repeat("n", 501)}}} {
		w := cfgPost(h, "/health/foods", bad, "")
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `role="alert"`) {
			t.Fatalf("%v: %d", bad, w.Code)
		}
	}
	if w := cfgPost(h, "/health/foods", url.Values{"name": {"X"}, "kcal": {"-1"}}, ""); !strings.Contains(w.Body.String(), `value="-1"`) {
		t.Fatal("typed values not kept")
	}
	if w := cfgPost(h, "/health/foods", add, "https://evil.example"); w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin: %d", w.Code)
	}
	for _, path := range []string{"/health/foods/999", "/health/foods/x"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	if more := cfgGet(t, h, "/health/more"); !strings.Contains(more, `href="/health/checklist"`) || !strings.Contains(more, `href="/health/foods"`) {
		t.Fatal("More must link to checklist and foods")
	}
}

// The database enforces the same limits as the forms.
func TestHealthConfigConstraints(t *testing.T) {
	h := testHandler(t)
	neg := -5.0
	if err := h.Service.Repo.CreateHealthFood(&models.HealthFood{Name: "Bad", Kcal: &neg, Active: true}); err == nil {
		t.Fatal("negative kcal accepted by the database")
	}
	if err := h.Service.Repo.CreateHealthChecklistItem(&models.HealthChecklistItem{Name: "Water", Kind: "water"}); err == nil {
		t.Fatal("unknown kind accepted by the database")
	}
	if err := h.Service.Repo.CreateHealthChecklistItem(&models.HealthChecklistItem{Name: "   ", Kind: "habit"}); err == nil {
		t.Fatal("blank name accepted by the database")
	}
}

// Concurrent toggles never fail and never leave a duplicate tick (#61 review).
func TestChecklistToggleConcurrent(t *testing.T) {
	h := testHandler(t)
	item, err := h.Service.AddHealthChecklistItem("Creatine", "habit")
	if err != nil {
		t.Fatal(err)
	}
	errs := make(chan error, 40)
	for i := 0; i < 40; i++ {
		go func() {
			_, err := h.Service.Repo.ToggleHealthHabit(item.ID, "2026-10-07")
			errs <- err
		}()
	}
	for i := 0; i < 40; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("toggle failed: %v", err)
		}
	}
	items, _ := h.Service.Repo.ListHealthChecklistItems("2026-10-07")
	if len(items) != 1 {
		t.Fatalf("items %+v", items)
	}
}
