package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"lifeops-tracker/internal/healthcoach"
)

// GYST-180 (estimates): the photo AI's macros prefill an unmatched food, never
// a matched one, are held server-side, and are stored as ai_est_* with the
// provenance rules of GYST-179. Fictional values; the provider is the in-process
// fake (the whole package has no way to reach a real one).

func stewEst() *healthcoach.FoodEstimate {
	return &healthcoach.FoodEstimate{Kcal: 400, ProteinG: 20, CarbsG: 40, FatG: 12} // at 200 g
}

func estPlate() healthcoach.FoodPhotoResult {
	return healthcoach.FoodPhotoResult{Items: []healthcoach.FoodPhotoItem{
		{Name: "test stew", Role: "main", EstGrams: gramsPtr(200), Confidence: 0.9, Visible: true, Est: stewEst()},
		{Name: "test lentils", Role: "side", EstGrams: gramsPtr(150), Confidence: 0.9, Visible: true, Est: &healthcoach.FoodEstimate{Kcal: 999, ProteinG: 1, CarbsG: 1, FatG: 1}},
		{Name: "test plain", Role: "side", EstGrams: gramsPtr(100), Confidence: 0.9, Visible: true},
	}, Uncertainties: []string{}, Questions: []string{}}
}

// c1/c2: an unmatched food carries the estimate; a matched one does not, even
// though the provider sent macros for it.
func TestCameraReviewEstimateOnlyForUnmatchedFood(t *testing.T) {
	h := testHandler(t)
	cameraCatalogueFood(t, h) // "Test lentils" per 100 g
	h.Coach = &fakeVision{result: estPlate()}
	w := doPhoto(h, photoReq(t, "photo", testJPEG(t), nil))
	if w.Code != 200 {
		t.Fatalf("analyze %d: %s", w.Code, w.Body.String())
	}
	var raw struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	stew, lentils, plain := raw.Items[0], raw.Items[1], raw.Items[2]
	est, ok := stew["est_nutrition"].(map[string]any)
	if !ok || est["kcal"] != float64(400) || est["protein_g"] != float64(20) || est["carbs_g"] != float64(40) || est["fat_g"] != float64(12) || stew["est_grams"] != float64(200) {
		t.Fatalf("unmatched item: %v", stew)
	}
	if _, has := lentils["est_nutrition"]; has || lentils["match"] == nil {
		t.Fatalf("matched item kept AI macros: %v", lentils)
	}
	if _, has := plain["est_nutrition"]; has {
		t.Fatalf("item without an estimate shows one: %v", plain)
	}
	// Server side: held for the unmatched item only.
	var rv photoReview
	_ = json.Unmarshal(w.Body.Bytes(), &rv)
	n := h.camera.m[rv.Nonce]
	if n.estNut[0] == nil || *n.estNut[0].Kcal != 400 || n.estNut[1] != nil || n.estNut[2] != nil {
		t.Fatalf("estNut %+v", n.estNut)
	}
	// And a matched food saved from this review stores no ai_est macros.
	b, _ := json.Marshal(map[string]any{"nonce": rv.Nonce, "slot": "dinner", "items": []map[string]any{
		{"idx": 1, "name": "Test lentils", "grams": 150, "kcal": 150, "protein_g": 12, "food_ref": rv.Items[1].Match.Ref}}})
	if w := cameraConfirmDo(h, string(b), nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
	if len(items) != 1 || items[0].NutritionProvenance != "database" || items[0].AIEstKcal != nil || items[0].AIEstProteinG != nil || items[0].AIEstCarbsG != nil || items[0].AIEstFatG != nil {
		t.Fatalf("matched item: %+v", items)
	}
}

func analyzeStew(t *testing.T, h *Handler) photoReview {
	t.Helper()
	return cameraAnalyze(t, h, healthcoach.FoodPhotoResult{Items: estPlate().Items[:1], Uncertainties: []string{}, Questions: []string{}})
}

func confirmStew(t *testing.T, h *Handler, nonce string, extra map[string]any) {
	t.Helper()
	item := map[string]any{"idx": 0, "name": "Test stew", "grams": 300, "kcal": 600, "protein_g": 30, "carbs_g": 60, "fat_g": 18} // the estimate at 300 g
	for k, v := range extra {
		item[k] = v
	}
	b, _ := json.Marshal(map[string]any{"nonce": nonce, "slot": "dinner", "items": []map[string]any{item}})
	if w := cameraConfirmDo(h, string(b), nil); w.Code != http.StatusOK {
		t.Fatalf("confirm %d: %s", w.Code, w.Body.String())
	}
}

// c5: as estimated (scaled) -> ai_estimate with the AI's own numbers kept; the
// browser cannot supply ai_est values; the day total counts it as an estimate.
func TestCameraConfirmKeepsEstimateAndProvenance(t *testing.T) {
	h := testHandler(t)
	rv := analyzeStew(t, h)
	confirmStew(t, h, rv.Nonce, map[string]any{"ai_est_kcal": 1, "ai_est_protein_g": 1, "ai_est_kcal_x": 5})
	items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
	it := items[0]
	if it.NutritionProvenance != "ai_estimate" || *it.Kcal != 600 || *it.ProteinG != 30 || *it.CarbsG != 60 || *it.FatG != 18 {
		t.Fatalf("saved: %+v", it)
	}
	if *it.AIEstKcal != 400 || *it.AIEstProteinG != 20 || *it.AIEstCarbsG != 40 || *it.AIEstFatG != 12 || *it.AIEstGrams != 200 {
		t.Fatalf("kept estimate must be the server's, not the browser's: %+v", it)
	}
	in, err := h.Service.DayIntake(h.Service.TodayString())
	if err != nil || in.Estimated.Items != 1 || *in.Estimated.Kcal != 600 || *in.Estimated.ProteinG != 30 {
		t.Fatalf("estimated %+v %v", in.Estimated, err)
	}
}

// Edited: the item is Jay's (user) and ai_est_* stay what the AI gave, but each
// value still equal to the scaled estimate keeps counting and stays disclosed
// (Codex P1). A value he re-entered differently does not.
func TestCameraPartialEditKeepsUntouchedValuesCountedAsEstimates(t *testing.T) {
	ptr := func(v float64) *float64 { return &v }
	cases := []struct {
		name      string
		edit      map[string]any
		prov      string
		items     int
		kcal, pro *float64
	}{
		{"no edit", nil, "ai_estimate", 1, ptr(600), ptr(30)},
		{"protein edited", map[string]any{"protein_g": 31}, "user", 1, ptr(600), nil},
		{"kcal edited", map[string]any{"kcal": 550}, "user", 1, nil, ptr(30)},
		{"both edited", map[string]any{"kcal": 550, "protein_g": 31}, "user", 0, nil, nil},
		{"fibre added", map[string]any{"fiber_g": 4}, "user", 1, ptr(600), ptr(30)},
	}
	for _, c := range cases {
		h := testHandler(t)
		confirmStew(t, h, analyzeStew(t, h).Nonce, c.edit)
		items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
		it := items[0]
		if it.NutritionProvenance != c.prov || *it.AIEstKcal != 400 || *it.AIEstProteinG != 20 || *it.AIEstFatG != 12 {
			t.Fatalf("%s: %+v", c.name, it)
		}
		in, _ := h.Service.DayIntake(h.Service.TodayString())
		e := in.Estimated
		same := func(a, b *float64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
		if e.Items != c.items || !same(e.Kcal, c.kcal) || !same(e.ProteinG, c.pro) {
			t.Errorf("%s: estimated %+v kcal %v protein %v", c.name, e, deref(e.Kcal), deref(e.ProteinG))
		}
		// The item stays in the day's counted total whatever is disclosed.
		if in.Counted.Kcal == nil {
			t.Errorf("%s: counted lost", c.name)
		}
	}
}

// c6: the FD-3 accuracy query sees the gap for items created through confirm.
func TestAccuracyQueryOnItemsFromConfirm(t *testing.T) {
	h := testHandler(t)
	plate := healthcoach.FoodPhotoResult{Items: []healthcoach.FoodPhotoItem{estPlate().Items[0], estPlate().Items[0], estPlate().Items[2]}, Uncertainties: []string{}, Questions: []string{}}
	rv := cameraAnalyze(t, h, plate)
	b, _ := json.Marshal(map[string]any{"nonce": rv.Nonce, "slot": "dinner", "items": []map[string]any{
		{"idx": 0, "name": "Test stew", "grams": 300, "kcal": 600, "protein_g": 30, "carbs_g": 60, "fat_g": 18},     // as estimated
		{"idx": 1, "name": "Test stew two", "grams": 300, "kcal": 650, "protein_g": 30, "carbs_g": 60, "fat_g": 18}, // kcal 50 over
		{"idx": 2, "name": "Test plain", "grams": 100, "kcal": 80},                                                  // no estimate: not paired for macros
	}})
	if w := cameraConfirmDo(h, string(b), nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	today := h.Service.TodayString()
	weeks, err := h.Service.Repo.AIAccuracyByWeek(today, today)
	if err != nil || len(weeks) != 1 {
		t.Fatalf("%v %+v", err, weeks)
	}
	w := weeks[0]
	if w.Items != 3 || w.Kcal.Paired != 2 || w.ProteinG.Paired != 2 || w.CarbsG.Paired != 2 || w.FatG.Paired != 2 || w.Grams.Paired != 3 {
		t.Fatalf("paired counts: %+v", w)
	}
	// saved - estimate, the estimate being the AI's own at its 200 g:
	// kcal (600-400 + 650-400)/2 = 225; protein 30-20 = 10.
	if w.Kcal.Bias == nil || *w.Kcal.Bias != 225 || w.ProteinG.Bias == nil || *w.ProteinG.Bias != 10 {
		t.Fatalf("bias kcal %v protein %v", w.Kcal.Bias, w.ProteinG.Bias)
	}
}

// Nothing invalid reaches the nonce or the database: the parser drops it, so
// the handler never sees it (and an estimate without grams is never kept).
func TestCameraEstimateWithoutGramsIsNotKept(t *testing.T) {
	h := testHandler(t)
	res := healthcoach.FoodPhotoResult{Items: []healthcoach.FoodPhotoItem{{Name: "test stew", Role: "main", Confidence: 0.9, Visible: true, Est: stewEst()}}, Uncertainties: []string{}, Questions: []string{}}
	rv := cameraAnalyze(t, h, res)
	if rv.Items[0].Est != nil || h.camera.m[rv.Nonce].estNut[0] != nil {
		t.Fatalf("estimate kept without grams: %+v", rv.Items[0])
	}
}
