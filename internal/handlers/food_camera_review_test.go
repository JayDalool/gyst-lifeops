package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
)

// Fictional values only; fake provider, no network.

func TestCameraLowConfidenceAsksForGrams(t *testing.T) {
	h := testHandler(t)
	rv := cameraAnalyze(t, h, healthcoach.FoodPhotoResult{
		Items: []healthcoach.FoodPhotoItem{
			{Name: "test soup", Role: "main", EstGrams: gramsPtr(300), Confidence: 0.3, Visible: true},
			{Name: "test oil", Role: "oil", EstGrams: gramsPtr(10), Confidence: 0.9, Visible: false},
			{Name: "test rice", Role: "side", EstGrams: gramsPtr(150), Confidence: 0.5, Visible: true},
		},
		Uncertainties: []string{}, Questions: []string{},
	})
	if !rv.Items[0].AskGrams || !rv.Items[1].AskGrams || rv.Items[2].AskGrams {
		t.Fatalf("ask_grams = %v %v %v", rv.Items[0].AskGrams, rv.Items[1].AskGrams, rv.Items[2].AskGrams)
	}
	// The estimate is still kept server-side as the snapshot.
	body := `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"idx":0,"name":"Test soup","grams":250}]}`
	if w := cameraConfirmDo(h, body, nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
	if len(items) != 1 || items[0].AIEstGrams == nil || *items[0].AIEstGrams != 300 || items[0].QuantityProvenance != "user_confirmed" {
		t.Fatalf("%+v", items)
	}
}

func TestCameraDuplicateIdxRejected(t *testing.T) {
	h := testHandler(t)
	rv := cameraAnalyze(t, h, testPlate)
	body := `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"idx":0,"name":"A","grams":10},{"idx":0,"name":"B","grams":20}]}`
	if w := cameraConfirmDo(h, body, nil); w.Code != 400 {
		t.Fatalf("duplicate idx: %d", w.Code)
	}
	if diaryCount(t, h) != 0 {
		t.Fatal("a refused review wrote rows")
	}
	// The nonce is not burned: the corrected review saves.
	ok := `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"idx":0,"name":"A","grams":10},{"idx":1,"name":"B","grams":20}]}`
	if w := cameraConfirmDo(h, ok, nil); w.Code != 200 || diaryCount(t, h) != 2 {
		t.Fatalf("retry: %d", w.Code)
	}
}

func TestCameraAcceptedRoundedEstimateIsAIEstimate(t *testing.T) {
	h := testHandler(t)
	rv := cameraAnalyze(t, h, healthcoach.FoodPhotoResult{
		Items:         []healthcoach.FoodPhotoItem{{Name: "test rice", Role: "side", EstGrams: gramsPtr(123.456), Confidence: 0.9, Visible: true}},
		Uncertainties: []string{}, Questions: []string{},
	})
	// The review shows 123.5; accepting it must count as the estimate.
	body := `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"idx":0,"name":"Test rice","grams":123.5}]}`
	if w := cameraConfirmDo(h, body, nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
	if len(items) != 1 || items[0].QuantityProvenance != "ai_estimate" {
		t.Fatalf("%+v", items)
	}
}

func TestCameraInflightConfirmIs409(t *testing.T) {
	var c cameraStore
	now := time.Now()
	n := c.issue("sess", nil, now)
	if _, res := c.claim(n, "sess", now); res != claimOK {
		t.Fatal("first claim")
	}
	if _, res := c.claim(n, "sess", now); res != claimInflight {
		t.Fatal("second claim while saving must be inflight")
	}
	c.finish(n, false) // failed save: usable again
	if _, res := c.claim(n, "sess", now); res != claimOK {
		t.Fatal("a failed save must release the nonce")
	}
	c.finish(n, true)
	if _, res := c.claim(n, "sess", now); res != claimAlready {
		t.Fatal("a used nonce is already")
	}

	h := testHandler(t)
	nonce := h.camera.issue("", []*float64{gramsPtr(150)}, h.Service.Now())
	h.camera.claim(nonce, "", h.Service.Now()) // first confirm still saving
	w := cameraConfirmDo(h, `{"nonce":"`+nonce+`","slot":"dinner","items":[{"idx":0,"name":"A","grams":10}]}`, nil)
	if w.Code != http.StatusConflict || strings.Contains(w.Body.String(), "already") || diaryCount(t, h) != 0 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestCameraNonceTTLAndEviction(t *testing.T) {
	var c cameraStore
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	n := c.issue("s", nil, now)
	if _, res := c.claim(n, "s", now.Add(cameraNonceTTL-time.Second)); res != claimOK {
		t.Fatal("inside the TTL")
	}
	c.finish(n, false)
	if _, res := c.claim(n, "s", now.Add(cameraNonceTTL+time.Second)); res != claimInvalid {
		t.Fatal("expired nonce must be invalid")
	}
	// Eviction: at the cap, the soonest to expire goes.
	var d cameraStore
	first := d.issue("s", nil, now)
	for i := 1; i < cameraMaxNonces; i++ {
		d.issue("s", nil, now.Add(time.Duration(i)*time.Second))
	}
	if len(d.m) != cameraMaxNonces {
		t.Fatalf("size %d", len(d.m))
	}
	d.issue("s", nil, now.Add(time.Hour/100))
	if len(d.m) != cameraMaxNonces {
		t.Fatalf("size after cap %d", len(d.m))
	}
	if _, ok := d.m[first]; ok {
		t.Fatal("the soonest-to-expire nonce should have been evicted")
	}
	// Expired entries are swept on the next issue.
	d.issue("s", nil, now.Add(2*cameraNonceTTL))
	if len(d.m) != 1 {
		t.Fatalf("expired entries not swept: %d", len(d.m))
	}
}

func TestCameraStoreKeepsNoRawSessionToken(t *testing.T) {
	var c cameraStore
	c.issue("raw-session-token-value", nil, time.Now())
	for _, v := range c.m {
		if strings.Contains(v.session, "raw-session-token-value") || len(v.session) != 64 {
			t.Fatalf("store holds %q", v.session)
		}
	}
}

func cameraCatalogueFood(t *testing.T, h *Handler) models.NutritionFood {
	t.Helper()
	k := func(v float64) *float64 { return &v }
	f, err := h.Service.Repo.UpsertNutritionFoodVersion(context.Background(), models.NutritionFood{
		Provider: "cnf", ProviderID: "test-1", Name: "Test lentils", NormalizedName: "test lentils", Basis: "per_100g",
		Nutrients:      models.Nutrients{Kcal: k(100), ProteinG: k(8)},
		DataProvenance: "database", DatasetVersion: "test",
	}, clock.FormatInstant(h.Service.Now()))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCameraUntouchedPrefillKeepsFoodProvenance(t *testing.T) {
	h := testHandler(t)
	food := cameraCatalogueFood(t, h)
	rv := cameraAnalyze(t, h, healthcoach.FoodPhotoResult{
		Items:         []healthcoach.FoodPhotoItem{{Name: "test lentils", Role: "main", EstGrams: gramsPtr(150), Confidence: 0.9, Visible: true}, {Name: "test lentils", Role: "side", Confidence: 0.9, Visible: true}},
		Uncertainties: []string{}, Questions: []string{},
	})
	m := rv.Items[0].Match
	if m == nil || !strings.HasPrefix(m.Ref, "nf:") || m.Per100.Kcal == nil || *m.Per100.Kcal != 100 {
		t.Fatalf("match %+v", m)
	}
	req := map[string]any{"nonce": rv.Nonce, "slot": "dinner", "items": []map[string]any{
		{"idx": 0, "name": "Test lentils", "grams": 150, "kcal": 150, "protein_g": 12, "food_ref": m.Ref},      // untouched prefill
		{"idx": 1, "name": "Test lentils edited", "grams": 100, "kcal": 90, "protein_g": 8, "food_ref": m.Ref}, // edited kcal
	}}
	b, _ := json.Marshal(req)
	if w := cameraConfirmDo(h, string(b), nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
	byName := map[string]models.DiaryItem{}
	for _, it := range items {
		byName[it.Name] = it
	}
	a, e := byName["Test lentils"], byName["Test lentils edited"]
	if a.FoodID == nil || *a.FoodID != food.ID || a.NutritionProvenance != "database" {
		t.Fatalf("untouched prefill: %+v", a)
	}
	if e.FoodID != nil || e.NutritionProvenance != "user" {
		t.Fatalf("edited values must stay user: %+v", e)
	}
	// A forged ref with matching-looking values for another food is not linked.
	rv2 := h.camera.issue("", []*float64{nil}, h.Service.Now())
	b, _ = json.Marshal(map[string]any{"nonce": rv2, "slot": "dinner", "items": []map[string]any{
		{"name": "x", "grams": 100, "kcal": 100, "food_ref": "nf:99999"}}})
	if w := cameraConfirmDo(h, string(b), nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
}
