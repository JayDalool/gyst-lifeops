package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
)

// GYST-179 server side of "adjust before logging". Fictional values only; the
// fake provider never touches the network.

func adjustPost(h *Handler, nonce string, portion any, items ...map[string]any) *httptest.ResponseRecorder {
	req := map[string]any{"nonce": nonce, "slot": "dinner", "items": items}
	if portion != nil {
		req["portion"] = portion
	}
	b, _ := json.Marshal(req)
	return cameraConfirmDo(h, string(b), nil)
}

func lentilsReview(t *testing.T, h *Handler) (photoReview, models.NutritionFood) {
	food := cameraCatalogueFood(t, h)
	rv := cameraAnalyze(t, h, healthcoach.FoodPhotoResult{
		Items:         []healthcoach.FoodPhotoItem{{Name: "test lentils", Role: "main", EstGrams: gramsPtr(150), Confidence: 0.9, Visible: true}},
		Uncertainties: []string{}, Questions: []string{},
	})
	if rv.Items[0].Match == nil {
		t.Fatal("no catalogue match")
	}
	return rv, food
}

func liveItems(t *testing.T, h *Handler) []models.DiaryItem {
	t.Helper()
	items, err := h.Service.ListDiaryItems(h.Service.TodayString())
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func TestCameraAdjustedGramsOnMatchedItemRecomputeFromTheDatabase(t *testing.T) {
	h := testHandler(t)
	rv, food := lentilsReview(t, h)
	ref := rv.Items[0].Match.Ref
	// 160 g where the AI said 150: the macros are the database's, scaled by the server.
	w := adjustPost(h, rv.Nonce, nil,
		map[string]any{"idx": 0, "name": "Test lentils", "grams": 160, "kcal": 160, "protein_g": 12.8, "food_ref": ref},
		map[string]any{"name": "Hand added oil", "grams": 10, "kcal": 90}) // added by hand: no idx
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	by := map[string]models.DiaryItem{}
	for _, it := range liveItems(t, h) {
		by[it.Name] = it
	}
	m, oil := by["Test lentils"], by["Hand added oil"]
	if *m.Kcal != 160 || *m.ProteinG != 12.8 || m.NutritionProvenance != "database" || m.QuantityProvenance != "user_confirmed" || *m.FoodID != food.ID {
		t.Fatalf("matched item: %+v", m)
	}
	if oil.NutritionProvenance != "user" || oil.QuantityProvenance != "user_confirmed" || oil.AIEstGrams != nil || oil.AIEstKcal != nil || oil.FoodID != nil {
		t.Fatalf("hand-added item: %+v", oil)
	}
}

func TestCameraDatabaseMacroEditIsRefusedAndNothingIsSaved(t *testing.T) {
	h := testHandler(t)
	rv, _ := lentilsReview(t, h)
	ref := rv.Items[0].Match.Ref
	for name, item := range map[string]map[string]any{
		"kcal edited":       {"idx": 0, "name": "Test lentils", "grams": 150, "kcal": 140, "protein_g": 12, "food_ref": ref},
		"protein edited":    {"idx": 0, "name": "Test lentils", "grams": 150, "kcal": 150, "protein_g": 13, "food_ref": ref},
		"fat out of food":   {"idx": 0, "name": "Test lentils", "grams": 150, "kcal": 150, "protein_g": 12, "fat_g": 5, "food_ref": ref},
		"just over 0.11":    {"idx": 0, "name": "Test lentils", "grams": 150, "kcal": 150.12, "protein_g": 12, "food_ref": ref},
		"detach is unknown": {"idx": 0, "name": "Test lentils", "grams": 150, "kcal": 99, "food_ref": ref, "detach": true, "nutrition_provenance": "user"},
	} {
		w := adjustPost(h, rv.Nonce, nil, item)
		if w.Code != 400 || !strings.Contains(w.Body.String(), "database food") || diaryCount(t, h) != 0 {
			t.Errorf("%s: %d %s (rows %d)", name, w.Code, w.Body.String(), diaryCount(t, h))
		}
	}
	// The refusals did not burn the nonce: the corrected review saves. 150.04 g
	// of the same food shows 150.04 kcal-ish unrounded values; the server stores its own.
	w := adjustPost(h, rv.Nonce, nil, map[string]any{"idx": 0, "name": "Test lentils", "grams": 150.04, "kcal": 150.04, "protein_g": 12, "food_ref": ref})
	if w.Code != 200 {
		t.Fatalf("retry: %d %s", w.Code, w.Body.String())
	}
	it := liveItems(t, h)[0]
	if *it.Grams != 150 || *it.Kcal != 150 || *it.ProteinG != 12 || it.NutritionProvenance != "database" {
		t.Fatalf("server value not stored: %+v", it)
	}
}

func TestCameraForgedFoodRefStaysFreeForm(t *testing.T) {
	h := testHandler(t)
	nonce := h.camera.issue("", []*float64{nil}, h.Service.Now())
	w := adjustPost(h, nonce, nil, map[string]any{"name": "x", "grams": 100, "kcal": 100, "food_ref": "nf:99999"})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if it := liveItems(t, h)[0]; it.FoodID != nil || it.NutritionProvenance != "user" || *it.Kcal != 100 {
		t.Fatalf("%+v", it)
	}
}

func TestCameraPortionOverHTTP(t *testing.T) {
	h := testHandler(t)
	for _, bad := range []any{0.45, 2.05, 1.03, 0, -1, "1.2", "NaN", true, []int{1}} {
		nonce := h.camera.issue("", []*float64{gramsPtr(150)}, h.Service.Now())
		if w := adjustPost(h, nonce, bad, map[string]any{"idx": 0, "name": "Test rice", "grams": 150}); w.Code != 400 || diaryCount(t, h) != 0 {
			t.Errorf("portion %v: %d %s", bad, w.Code, w.Body.String())
		}
	}
	for _, ok := range []any{0.5, 1.05, 2.0, 2} {
		nonce := h.camera.issue("", []*float64{gramsPtr(150)}, h.Service.Now())
		if w := adjustPost(h, nonce, ok, map[string]any{"idx": 0, "name": "Test rice", "grams": 150}); w.Code != 200 {
			t.Errorf("portion %v: %d %s", ok, w.Code, w.Body.String())
		}
	}
	// x1.2 with grams +20%: user_confirmed for every item. The second item's
	// grams (90 x 1.2 = 108 vs est 108) equal its estimate after rounding.
	h2 := testHandler(t)
	nonce := h2.camera.issue("", []*float64{gramsPtr(150), gramsPtr(108)}, h2.Service.Now())
	w := adjustPost(h2, nonce, 1.2,
		map[string]any{"idx": 0, "name": "Test rice", "grams": 180},
		map[string]any{"idx": 1, "name": "Test beans", "grams": 108})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	for _, it := range liveItems(t, h2) {
		if it.QuantityProvenance != "user_confirmed" || it.AIEstGrams == nil {
			t.Errorf("%+v", it)
		}
	}
}

func TestCameraGramsThatRoundToZeroAreRefused(t *testing.T) {
	h := testHandler(t)
	nonce := h.camera.issue("", nil, h.Service.Now())
	if w := adjustPost(h, nonce, nil, map[string]any{"name": "Test rice", "grams": 0.04}); w.Code != 400 || diaryCount(t, h) != 0 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestCameraAIEstimateNutritionThroughConfirm(t *testing.T) {
	h := testHandler(t)
	est := []*models.Nutrients{{Kcal: gramsPtr(200), ProteinG: gramsPtr(10), CarbsG: gramsPtr(30), FatG: gramsPtr(5)}}
	nonce := h.camera.issueWithNutrition("", []*float64{gramsPtr(100)}, est, h.Service.Now())
	// Untouched at the estimate grams: ai_estimate, estimate kept.
	if w := adjustPost(h, nonce, nil, map[string]any{"idx": 0, "name": "Test stew", "grams": 100, "kcal": 200, "protein_g": 10, "carbs_g": 30, "fat_g": 5}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	it := liveItems(t, h)[0]
	if it.NutritionProvenance != "ai_estimate" || it.QuantityProvenance != "ai_estimate" || *it.AIEstKcal != 200 {
		t.Fatalf("equal: %+v", it)
	}
	// Edited: Jay's own, estimate still there.
	nonce = h.camera.issueWithNutrition("", []*float64{gramsPtr(100)}, est, h.Service.Now())
	if w := adjustPost(h, nonce, nil, map[string]any{"idx": 0, "name": "Test stew edited", "grams": 100, "kcal": 180, "protein_g": 10, "carbs_g": 30, "fat_g": 5}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	for _, it := range liveItems(t, h) {
		if it.Name == "Test stew edited" && (it.NutritionProvenance != "user" || *it.Kcal != 180 || *it.AIEstKcal != 200 || *it.AIEstFatG != 5) {
			t.Fatalf("edited: %+v", it)
		}
	}
}

func foodsPost(h http.Handler, body string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/health/nutrition/photo/foods", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestCameraFoodSearch(t *testing.T) {
	var logs bytes.Buffer
	h := testHandler(t)
	h.Logger = slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	vision := &fakeVision{}
	h.Coach = vision
	// Ten catalogue foods that match, one without per-100 g data, and a My food.
	for i := 0; i < 10; i++ {
		_, err := h.Service.Repo.UpsertNutritionFoodVersion(context.Background(), models.NutritionFood{
			Provider: "cnf", ProviderID: "zq-" + string(rune('a'+i)), Name: "Zqx bean " + string(rune('a'+i)), NormalizedName: "zqx bean " + string(rune('a'+i)),
			Basis: "per_100g", Nutrients: models.Nutrients{Kcal: gramsPtr(100)}, DataProvenance: "database", DatasetVersion: "test",
		}, clock.FormatInstant(h.Service.Now()))
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := h.Service.Repo.UpsertNutritionFoodVersion(context.Background(), models.NutritionFood{
		Provider: "cnf", ProviderID: "zq-empty", Name: "Zqx empty", NormalizedName: "zqx empty", Basis: "per_100g", DataProvenance: "database", DatasetVersion: "test",
	}, clock.FormatInstant(h.Service.Now()))
	if err != nil {
		t.Fatal(err)
	}
	itemsAddFood(t, h, "Zqx my porridge", viewFloat(300))

	w := foodsPost(h, `{"q":"zqx"}`, nil)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
	var res []photoMatch
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 8 {
		t.Fatalf("%d results, want the 8 cap", len(res))
	}
	for _, m := range res {
		if !strings.HasPrefix(m.Ref, "nf:") || m.Per100.Kcal == nil || strings.Contains(m.Name, "porridge") || strings.Contains(m.Name, "empty") || m.Source != "database" {
			t.Errorf("unexpected result %+v", m)
		}
	}
	if w := foodsPost(h, `{"q":"nothing matches this"}`, nil); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Errorf("empty: %d %q", w.Code, w.Body.String())
	}
	// Refusals.
	cases := map[string]struct {
		body string
		hdr  map[string]string
		want int
	}{
		"empty q":      {`{"q":""}`, nil, 400},
		"blank q":      {`{"q":"   "}`, nil, 400},
		"81 chars":     {`{"q":"` + strings.Repeat("a", 81) + `"}`, nil, 400},
		"not json":     {`zqx`, nil, 400},
		"over 1 KiB":   {`{"q":"zqx","pad":"` + strings.Repeat("a", 1100) + `"}`, nil, http.StatusRequestEntityTooLarge},
		"cross-site":   {`{"q":"zqx"}`, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		"other origin": {`{"q":"zqx"}`, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
	}
	for name, c := range cases {
		if w := foodsPost(h, c.body, c.hdr); w.Code != c.want {
			t.Errorf("%s: %d, want %d", name, w.Code, c.want)
		}
	}
	if w := foodsPost(h, `{"q":"`+strings.Repeat("a", 80)+`"}`, nil); w.Code != 200 {
		t.Errorf("80 chars: %d", w.Code)
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health/nutrition/photo/foods?q=zqx", nil))
	if r.Code != http.StatusMethodNotAllowed || r.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET: %d", r.Code)
	}
	// No AI call, no write, and the search text is in no log line.
	if vision.calls != 0 || diaryCount(t, h) != 0 {
		t.Errorf("provider calls %d, rows %d", vision.calls, diaryCount(t, h))
	}
	if strings.Contains(strings.ToLower(logs.String()), "zqx") || strings.Contains(logs.String(), strings.Repeat("a", 81)) {
		t.Errorf("a log line holds the search text: %s", logs.String())
	}
}

func TestCameraFoodSearchNeedsASession(t *testing.T) {
	h := authHandler(t)
	w := foodsPost(h.Authenticated(), `{"q":"zqx"}`, nil)
	if w.Code == 200 || w.Body.Len() > 0 && strings.HasPrefix(strings.TrimSpace(w.Body.String()), "[") {
		t.Fatalf("search answered without a session: %d %s", w.Code, w.Body.String())
	}
	cookie := signIn(t, h)
	r := httptest.NewRequest(http.MethodPost, "/health/nutrition/photo/foods", strings.NewReader(`{"q":"zqx"}`))
	r.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, r)
	if rec.Code != 200 {
		t.Fatalf("with a session: %d %s", rec.Code, rec.Body.String())
	}
}

// Codex P2 on #278: more than 20 matches that cannot be scaled by grams must not
// hide an eligible one behind the SQL limit.
func TestCameraFoodSearchFindsEligibleFoodBehindIneligibleOnes(t *testing.T) {
	h := testHandler(t)
	at := clock.FormatInstant(h.Service.Now())
	up := func(id, name string, f models.NutritionFood) {
		f.Provider, f.ProviderID, f.Name, f.NormalizedName, f.DataProvenance, f.DatasetVersion = "cnf", id, name, strings.ToLower(name), "database", "test"
		if _, err := h.Service.Repo.UpsertNutritionFoodVersion(context.Background(), f, at); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 25; i++ { // short names sort first; per serving with no gram weight, or no nutrients
		n := "Yqz " + string(rune('a'+i))
		if i%2 == 0 {
			up("in-"+n, n, models.NutritionFood{Basis: "per_serving", Nutrients: models.Nutrients{Kcal: gramsPtr(90)}})
		} else {
			up("in-"+n, n, models.NutritionFood{Basis: "per_100g"})
		}
	}
	up("ok-1", "Yqz good lentils with a long name", models.NutritionFood{Basis: "per_100g", Nutrients: models.Nutrients{Kcal: gramsPtr(100)}})
	w := foodsPost(h, `{"q":"yqz"}`, nil)
	var res []photoMatch
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || len(res) != 1 || !strings.Contains(res[0].Name, "good lentils") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
