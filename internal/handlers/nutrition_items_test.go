package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// Food diary routes and screen (Issue #216). Fictional values only.

func itemsAddFood(t *testing.T, h *Handler, name string, kcal *float64) {
	t.Helper()
	f := models.HealthFood{Name: name, Serving: "1 bowl", Kcal: kcal, ProteinG: viewFloat(10), Active: true}
	if err := h.Service.Repo.CreateHealthFood(&f); err != nil {
		t.Fatal(err)
	}
}

func itemsDo(h *Handler, method, path string, form url.Values, hdr map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

var (
	reFoodID = regexp.MustCompile(`name="food_ref" value="([a-z]+:\d+)"><input type="hidden" name="key" value="([0-9a-f]{32})"`)
	reItemID = regexp.MustCompile(`/health/nutrition/items/(\d+)/archive`)
)

func itemsSearch(t *testing.T, h *Handler, q string) (page, foodID, key string) {
	t.Helper()
	rec := itemsDo(h, "GET", "/health/nutrition?q="+url.QueryEscape(q), nil, nil)
	if rec.Code != 200 {
		t.Fatalf("search %q: %d", q, rec.Code)
	}
	page = rec.Body.String()
	if m := reFoodID.FindStringSubmatch(page); m != nil {
		foodID, key = m[1], m[2]
	}
	return
}

func diaryCount(t *testing.T, h *Handler) int {
	t.Helper()
	items, err := h.Service.ListDiaryItems(h.Service.TodayString())
	if err != nil {
		t.Fatal(err)
	}
	return len(items)
}

func TestNutritionDiaryAddEditRemoveUndo(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test bean chili", viewFloat(300))
	page, foodID, key := itemsSearch(t, h, "chili")
	if foodID == "" || !strings.Contains(page, "Test bean chili") || !strings.Contains(page, "Amount (servings)") {
		t.Fatalf("search did not list the food: %s", page)
	}
	form := url.Values{"food_ref": {foodID}, "key": {key}, "amount": {"1.5"}, "slot": {"dinner"}, "q": {"chili"}}
	rec := itemsDo(h, "POST", "/health/nutrition/items", form, nil)
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/health/nutrition?added=1") {
		t.Fatalf("add: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	rec = itemsDo(h, "POST", "/health/nutrition/items", form, nil)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "dup=1") || diaryCount(t, h) != 1 {
		t.Fatalf("repeat key: %d %q count %d", rec.Code, rec.Header().Get("Location"), diaryCount(t, h))
	}
	page = itemsDo(h, "GET", "/health/nutrition?dup=1", nil, nil).Body.String()
	if !strings.Contains(page, "Already added") || !strings.Contains(page, "Dinner") || !strings.Contains(page, "1.5 servings · 450 kcal") {
		t.Fatalf("diary list: %s", page)
	}
	if !strings.Contains(page, "from your GYST diary") {
		t.Fatal("source wording missing")
	}
	id := reItemID.FindStringSubmatch(page)[1]
	rec = itemsDo(h, "POST", "/health/nutrition/items/"+id, url.Values{"amount": {"2"}}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("edit: %d", rec.Code)
	}
	if page = itemsDo(h, "GET", "/health/nutrition", nil, nil).Body.String(); !strings.Contains(page, "2 servings · 600 kcal") {
		t.Fatal("edit not shown")
	}
	rec = itemsDo(h, "POST", "/health/nutrition/items/"+id+"/archive", url.Values{}, nil)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "removed="+id) || diaryCount(t, h) != 0 {
		t.Fatalf("remove: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	page = itemsDo(h, "GET", strings.Split(rec.Header().Get("Location"), "#")[0], nil, nil).Body.String()
	if !strings.Contains(page, "Removed.") || !strings.Contains(page, "/health/nutrition/items/"+id+"/restore") {
		t.Fatalf("no Undo: %s", page)
	}
	rec = itemsDo(h, "POST", "/health/nutrition/items/"+id+"/restore", url.Values{}, nil)
	if rec.Code != http.StatusSeeOther || diaryCount(t, h) != 1 {
		t.Fatalf("undo: %d count %d", rec.Code, diaryCount(t, h))
	}
	rec = itemsDo(h, "POST", "/health/nutrition/items/"+id+"/restore", url.Values{}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("restore of a live item = %d", rec.Code)
	}
}

func TestNutritionDiaryValidationRerenders400(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test lentil soup", nil)
	_, foodID, key := itemsSearch(t, h, "lentil")
	for name, form := range map[string]url.Values{
		"zero amount": {"food_ref": {foodID}, "key": {key}, "amount": {"0"}, "slot": {"lunch"}, "q": {"lentil"}},
		"text amount": {"food_ref": {foodID}, "key": {key}, "amount": {"lots"}, "slot": {"lunch"}},
		"bad slot":    {"food_ref": {foodID}, "key": {key}, "amount": {"1"}, "slot": {"brunch"}},
		"bad key":     {"food_ref": {foodID}, "key": {"short"}, "amount": {"1"}, "slot": {"lunch"}},
		"no food":     {"food_ref": {"nf:99999"}, "key": {key}, "amount": {"1"}, "slot": {"lunch"}},
	} {
		rec := itemsDo(h, "POST", "/health/nutrition/items", form, nil)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `role="alert"`) {
			t.Fatalf("%s: %d", name, rec.Code)
		}
	}
	if diaryCount(t, h) != 0 {
		t.Fatal("a refused add wrote an item")
	}
	// A food with no kcal logs fine and reads "kcal unknown", never 0.
	rec := itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {foodID}, "key": {key}, "amount": {"1"}, "slot": {"lunch"}}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("add: %d", rec.Code)
	}
	page := itemsDo(h, "GET", "/health/nutrition", nil, nil).Body.String()
	if !strings.Contains(page, "kcal unknown") || strings.Contains(page, "partial · ") {
		t.Fatalf("an item with no kcal must read unknown, and a total of nothing is not partial: %s", page)
	}
	itemsAddFood(t, h, "Test lentil stew", viewFloat(300))
	_, stewID, stewKey := itemsSearch(t, h, "lentil stew")
	itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {stewID}, "key": {stewKey}, "amount": {"1"}, "slot": {"dinner"}}, nil)
	if page = itemsDo(h, "GET", "/health/nutrition", nil, nil).Body.String(); !strings.Contains(page, "partial · 1 item without a value") {
		t.Fatalf("partial total not labelled: %s", page)
	}
}

func TestNutritionItemRoutesGuards(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test rice bowl", viewFloat(200))
	_, foodID, key := itemsSearch(t, h, "rice")
	good := url.Values{"food_ref": {foodID}, "key": {key}, "amount": {"1"}, "slot": {"lunch"}}
	if rec := itemsDo(h, "POST", "/health/nutrition/items", good, nil); rec.Code != http.StatusSeeOther {
		t.Fatal(rec.Code)
	}
	id := strconv.Itoa(1)
	items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
	id = strconv.FormatInt(items[0].ID, 10)
	routes := []string{"/health/nutrition/items", "/health/nutrition/items/" + id, "/health/nutrition/items/" + id + "/archive", "/health/nutrition/items/" + id + "/restore"}
	for _, route := range routes {
		rec := itemsDo(h, "GET", route, nil, nil)
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "POST" {
			t.Fatalf("GET %s: %d allow %q", route, rec.Code, rec.Header().Get("Allow"))
		}
		rec = itemsDo(h, "POST", route, url.Values{"amount": {"3"}}, map[string]string{"Origin": "https://evil.example"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("cross-origin %s: %d", route, rec.Code)
		}
		big := url.Values{"amount": {"1"}, "pad": {strings.Repeat("x", 9<<10)}}
		if rec = itemsDo(h, "POST", route, big, nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("oversized %s: %d", route, rec.Code)
		}
		// No session: the protected handler refuses before the route runs.
		req := httptest.NewRequest("POST", route, strings.NewReader("amount=3"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec = httptest.NewRecorder()
		h.Authenticated().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusSeeOther {
			t.Fatalf("no session %s: %d", route, rec.Code)
		}
	}
	items, _ = h.Service.ListDiaryItems(h.Service.TodayString())
	if len(items) != 1 || items[0].Amount != 1 {
		t.Fatalf("a refused request changed the diary: %+v", items)
	}
	if rec := itemsDo(h, "POST", "/health/nutrition/items/abc", url.Values{}, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("bad id: %d", rec.Code)
	}
}

func TestNutritionSearchLiteralWildcardsAndHidden(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test 100% juice", viewFloat(90))
	itemsAddFood(t, h, "Test snack_bar", viewFloat(150))
	itemsAddFood(t, h, "Test plain toast", viewFloat(80))
	for q, want := range map[string]string{"%": "Test 100% juice", "_": "Test snack_bar"} {
		page, _, _ := itemsSearch(t, h, q)
		if !strings.Contains(page, want) || strings.Contains(page, "Test plain toast") {
			t.Fatalf("q=%q must match literally: %s", q, page)
		}
	}
	// Renaming a food supersedes the old version: only the new one is listed.
	foods, _ := h.Service.Repo.ListHealthFoods(true)
	for _, f := range foods {
		if f.Name == "Test plain toast" {
			itemsSearch(t, h, "toast")
			f.Name = "Test seeded toast"
			if err := h.Service.Repo.UpdateHealthFood(f); err != nil {
				t.Fatal(err)
			}
		}
	}
	if page, _, _ := itemsSearch(t, h, "toast"); strings.Contains(page, "Test plain toast") || !strings.Contains(page, "Test seeded toast") {
		t.Fatalf("superseded version shown: %s", page)
	}
	// Empty result points to My foods; a too-long query is refused with 400.
	if page, _, _ := itemsSearch(t, h, "zzzz"); !strings.Contains(page, `No food matches`) || !strings.Contains(page, `href="/health/foods"`) {
		t.Fatal("empty state missing the My foods link")
	}
	if rec := itemsDo(h, "GET", "/health/nutrition?q="+strings.Repeat("a", 81), nil, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("long query: %d", rec.Code)
	}
}

func TestNutritionGystDayShowsReferenceNotCounted(t *testing.T) {
	h, _ := diaryHandlerWithSyncedDay(t)
	itemsAddFood(t, h, "Test pasta plate", viewFloat(500))
	_, foodID, key := itemsSearch(t, h, "pasta")
	if rec := itemsDo(h, "POST", "/health/nutrition/items", url.Values{"food_ref": {foodID}, "key": {key}, "amount": {"1"}, "slot": {"dinner"}}, nil); rec.Code != http.StatusSeeOther {
		t.Fatal(rec.Code)
	}
	page := itemsDo(h, "GET", "/health/nutrition", nil, nil).Body.String()
	if !strings.Contains(page, "from your GYST diary · Apple Health 1,800 kcal, not counted") {
		t.Fatalf("source line missing: %s", page)
	}
	if today := itemsDo(h, "GET", "/health", nil, nil).Body.String(); strings.Contains(today, "Waiting for sync") {
		t.Fatal("Health says Waiting for sync on a GYST day")
	}
	if page := itemsDo(h, "GET", "/today", nil, nil).Body.String(); strings.Contains(page, "Waiting for sync") {
		t.Fatal("Today says Waiting for sync on a GYST day")
	}
}
