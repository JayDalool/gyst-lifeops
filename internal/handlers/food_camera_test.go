package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"lifeops-tracker/internal/healthcoach"
)

// All values here are fictional. The fake provider never touches the network.

const confirmPath = "/health/nutrition/photo/confirm"

func cameraAnalyze(t *testing.T, h *Handler, res healthcoach.FoodPhotoResult) photoReview {
	t.Helper()
	h.Coach = &fakeVision{result: res}
	w := doPhoto(h, photoReq(t, "photo", testJPEG(t), nil))
	if w.Code != 200 {
		t.Fatalf("analyze %d: %s", w.Code, w.Body.String())
	}
	var rv photoReview
	if err := json.Unmarshal(w.Body.Bytes(), &rv); err != nil {
		t.Fatal(err)
	}
	return rv
}

func cameraConfirmDo(h *Handler, body string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, confirmPath, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func gramsPtr(v float64) *float64 { return &v }

var testPlate = healthcoach.FoodPhotoResult{
	Items: []healthcoach.FoodPhotoItem{
		{Name: "test rice", Role: "side", EstGrams: gramsPtr(150), Confidence: 0.9, Visible: true},
		{Name: "unknown mystery dish", Role: "main", EstGrams: nil, Confidence: 0.2, Visible: true},
	},
	Uncertainties: []string{}, Questions: []string{"How much was on the plate?"},
}

func TestCameraAnalyzeIssuesNonceAndSavesNothing(t *testing.T) {
	h := testHandler(t)
	rv := cameraAnalyze(t, h, testPlate)
	if len(rv.Nonce) != 32 || len(rv.Items) != 2 || rv.Items[1].EstGrams != nil {
		t.Fatalf("review %+v", rv)
	}
	if rv.Items[0].Match != nil || rv.Items[1].Match != nil {
		t.Fatal("an unknown food must not get invented nutrition")
	}
	if diaryCount(t, h) != 0 {
		t.Fatal("analysis wrote to the diary")
	}
}

func TestCameraConfirmSavesWithProvenance(t *testing.T) {
	h := testHandler(t)
	rv := cameraAnalyze(t, h, testPlate)
	body := `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[
	 {"idx":0,"name":"Test rice","grams":150,"kcal":195,"protein_g":4},
	 {"idx":1,"name":"Test stew","grams":240},
	 {"idx":null,"name":"Added by hand","grams":30,"kcal":100}]}`
	w := cameraConfirmDo(h, body, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"already":false`) {
		t.Fatalf("confirm %d: %s", w.Code, w.Body.String())
	}
	items, err := h.Service.ListDiaryItems(h.Service.TodayString())
	if err != nil || len(items) != 3 {
		t.Fatalf("items %d %v", len(items), err)
	}
	byName := map[string]int{}
	for i, it := range items {
		byName[it.Name] = i
	}
	rice, stew, hand := items[byName["Test rice"]], items[byName["Test stew"]], items[byName["Added by hand"]]
	if rice.QuantityProvenance != "ai_estimate" || rice.NutritionProvenance != "user" || rice.Amount != 150 {
		t.Fatalf("rice %+v", rice)
	}
	if stew.QuantityProvenance != "user_confirmed" || stew.Kcal != nil {
		t.Fatalf("stew %+v (unknown nutrition must stay empty)", stew)
	}
	if hand.QuantityProvenance != "user_confirmed" {
		t.Fatalf("hand %+v", hand)
	}
	// The AI estimate snapshot sits beside the confirmed grams.
	if rice.AIEstGrams == nil || *rice.AIEstGrams != 150 || rice.CreatedVia != "camera" {
		t.Fatalf("rice est %v via %s", rice.AIEstGrams, rice.CreatedVia)
	}
	if stew.AIEstGrams != nil {
		t.Fatal("an item with no AI estimate must store NULL")
	}
}

func TestCameraConfirmAdjustedGramsKeepsEstimate(t *testing.T) {
	h := testHandler(t)
	rv := cameraAnalyze(t, h, testPlate)
	w := cameraConfirmDo(h, `{"nonce":"`+rv.Nonce+`","slot":"lunch","items":[{"idx":0,"name":"Test rice","grams":200}]}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
	if len(items) != 1 || items[0].QuantityProvenance != "user_confirmed" || items[0].Amount != 200 {
		t.Fatalf("%+v", items)
	}
	if items[0].AIEstGrams == nil || *items[0].AIEstGrams != 150 {
		t.Fatalf("ai_est_grams = %v", items[0].AIEstGrams)
	}
}

func TestCameraNonceIsSingleUseAndDoubleSubmitMakesOneMeal(t *testing.T) {
	h := testHandler(t)
	rv := cameraAnalyze(t, h, testPlate)
	body := `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"idx":0,"name":"Test rice","grams":150}]}`
	if w := cameraConfirmDo(h, body, nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w := cameraConfirmDo(h, body, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"already":true`) {
		t.Fatalf("second submit %d: %s", w.Code, w.Body.String())
	}
	if diaryCount(t, h) != 1 {
		t.Fatalf("double submit created %d items", diaryCount(t, h))
	}
	// Concurrent submits of one nonce also make one meal.
	nonce := h.camera.issue("", []*float64{gramsPtr(150)}, h.Service.Now())
	body = `{"nonce":"` + nonce + `","slot":"dinner","items":[{"idx":0,"name":"Test rice","grams":150}]}`
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); cameraConfirmDo(h, body, nil) }()
	}
	wg.Wait()
	if diaryCount(t, h) != 2 {
		t.Fatalf("concurrent submits: %d items, want 2", diaryCount(t, h))
	}
}

func TestCameraConfirmRefusals(t *testing.T) {
	h := testHandler(t)
	rv := cameraAnalyze(t, h, testPlate)
	good := `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"idx":0,"name":"Test rice","grams":150}]}`
	if w := cameraConfirmDo(h, `{"nonce":"`+strings.Repeat("a", 32)+`","items":[]}`, nil); w.Code != http.StatusGone {
		t.Errorf("unknown nonce: %d", w.Code)
	}
	if w := cameraConfirmDo(h, `not json`, nil); w.Code != 400 {
		t.Errorf("bad json: %d", w.Code)
	}
	for name, body := range map[string]string{
		"zero grams": `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"name":"x","grams":0}]}`,
		"no items":   `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[]}`,
		"no name":    `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"name":" ","grams":5}]}`,
		"neg kcal":   `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"name":"x","grams":5,"kcal":-1}]}`,
		"bad slot":   `{"nonce":"` + rv.Nonce + `","slot":"brunch","items":[{"name":"x","grams":5}]}`,
	} {
		if w := cameraConfirmDo(h, body, nil); w.Code != 400 {
			t.Errorf("%s: %d %s", name, w.Code, w.Body.String())
		}
	}
	if diaryCount(t, h) != 0 {
		t.Fatal("a refused review wrote rows")
	}
	// A refused attempt does not burn the nonce: the corrected review saves.
	if w := cameraConfirmDo(h, good, nil); w.Code != 200 || diaryCount(t, h) != 1 {
		t.Fatalf("retry after refusal: %d", w.Code)
	}
	if w := cameraConfirmDo(h, good, map[string]string{"Origin": "https://evil.example"}); w.Code != http.StatusForbidden {
		t.Errorf("foreign origin: %d", w.Code)
	}
	if w := httptest.NewRecorder(); true {
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, confirmPath, nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET: %d", w.Code)
		}
	}
}

func TestCameraNonceBoundToSession(t *testing.T) {
	h := authHandler(t)
	h.Coach = &fakeVision{result: testPlate}
	cookie := signIn(t, h)
	r := photoReq(t, "photo", testJPEG(t), nil)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(w, r)
	var rv photoReview
	if err := json.Unmarshal(w.Body.Bytes(), &rv); err != nil || rv.Nonce == "" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	body := `{"nonce":"` + rv.Nonce + `","slot":"dinner","items":[{"idx":0,"name":"Test rice","grams":150}]}`
	// No session: refused by auth. Another session: the nonce is not theirs.
	noAuth := httptest.NewRequest(http.MethodPost, confirmPath, strings.NewReader(body))
	w = httptest.NewRecorder()
	h.Authenticated().ServeHTTP(w, noAuth)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
	if _, res := h.camera.claim(rv.Nonce, "another-session", h.Service.Now()); res != claimInvalid {
		t.Fatal("a nonce was claimable by another session")
	}
	ok := httptest.NewRequest(http.MethodPost, confirmPath, strings.NewReader(body))
	ok.AddCookie(cookie)
	w = httptest.NewRecorder()
	h.Authenticated().ServeHTTP(w, ok)
	if w.Code != 200 {
		t.Fatalf("own session: %d %s", w.Code, w.Body.String())
	}
}

func TestCameraProviderFailureSavesNothing(t *testing.T) {
	h := testHandler(t)
	h.Coach = &fakeVision{err: healthcoach.ErrUnavailable}
	w := doPhoto(h, photoReq(t, "photo", testJPEG(t), nil))
	if w.Code != http.StatusBadGateway || errorOf(t, w) != healthcoach.ErrUnavailable.Error() {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if diaryCount(t, h) != 0 {
		t.Fatal("a provider failure wrote rows")
	}
}

// A food GYST already knows is offered as a per-100 g prefill from its own data.
func TestCameraMatchUsesOwnFoodsOnly(t *testing.T) {
	h := testHandler(t)
	itemsAddFood(t, h, "Test oat porridge", viewFloat(300)) // My foods: no gram weight, cannot be scaled
	rv := cameraAnalyze(t, h, healthcoach.FoodPhotoResult{
		Items:         []healthcoach.FoodPhotoItem{{Name: "oat porridge", Role: "main", Confidence: 0.9, Visible: true}},
		Uncertainties: []string{}, Questions: []string{},
	})
	if rv.Items[0].Match != nil {
		t.Fatalf("a per-serving food with no weight cannot be offered per 100 g: %+v", rv.Items[0].Match)
	}
}

func TestNutritionPageHasCameraReview(t *testing.T) {
	h := testHandler(t)
	rec := itemsDo(h, "GET", "/health/nutrition", nil, nil)
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	for _, want := range []string{`id="camera-input"`, `accept="image/*"`, `id="camera-review"`, `id="cam-portion"`, `min="0.5"`, `step="0.05"`, `id="cam-portion-reset"`, `id="cam-total"`, `id="cam-live"`, `id="cam-find"`, `/static/js/food-camera.js?v=cam-6`, `/static/js/food-camera-math.js`, `/static/js/food-camera.js`, `Save meal`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("page lacks %s", want)
		}
	}
}
