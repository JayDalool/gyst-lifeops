package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// AI food camera review and save (Issue #211). Analyze (food_photo.go) returns
// what the model identified plus a one-time nonce. Nothing is saved until
// POST /health/nutrition/photo/confirm carries that nonce with the items Jay
// reviewed. The AI's gram estimates stay on the server, keyed by the nonce, so
// the stored "AI estimate" cannot be edited from the browser. The nonce doubles
// as the meal's idempotency key.

const (
	// cameraNonceTTL is longer than the 2 minutes of 09 §C: that figure is for
	// a confirmation page, while this nonce must outlive editing a review.
	cameraNonceTTL     = 15 * time.Minute
	cameraMaxNonces    = 50
	cameraConfirmBytes = 32 << 10
)

const (
	nonceFresh = iota
	nonceInflight
	nonceUsed
)

type cameraNonce struct {
	session string
	expires time.Time
	est     []*float64
	state   int
}

// cameraStore holds issued nonces in memory. The zero value is ready. A
// restart drops them, which only means the photo is analysed again.
type cameraStore struct {
	mu sync.Mutex
	m  map[string]*cameraNonce
}

// sessionKey is what the store keeps in place of the raw session token.
func sessionKey(session string) string {
	sum := sha256.Sum256([]byte("camera-nonce:" + session))
	return hex.EncodeToString(sum[:])
}

func (c *cameraStore) issue(session string, est []*float64, now time.Time) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	nonce := hex.EncodeToString(b)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string]*cameraNonce{}
	}
	for k, v := range c.m {
		if now.After(v.expires) {
			delete(c.m, k)
		}
	}
	if len(c.m) >= cameraMaxNonces { // drop the soonest to expire
		var oldest string
		for k, v := range c.m {
			if oldest == "" || v.expires.Before(c.m[oldest].expires) {
				oldest = k
			}
		}
		delete(c.m, oldest)
	}
	c.m[nonce] = &cameraNonce{session: sessionKey(session), expires: now.Add(cameraNonceTTL), est: est}
	return nonce
}

type claimResult int

const (
	claimOK claimResult = iota
	claimInvalid
	claimInflight // the first confirm is still saving
	claimAlready  // already saved
)

// claim marks a fresh nonce in flight. A nonce that is unknown, expired or
// issued to another session is invalid; one already in flight or used is
// "already".
func (c *cameraStore) claim(nonce, session string, now time.Time) (*cameraNonce, claimResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := c.m[nonce]
	if n == nil || now.After(n.expires) || n.session != sessionKey(session) {
		return nil, claimInvalid
	}
	switch n.state {
	case nonceInflight:
		return n, claimInflight
	case nonceUsed:
		return n, claimAlready
	}
	n.state = nonceInflight
	return n, claimOK
}

func (c *cameraStore) finish(nonce string, saved bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n := c.m[nonce]; n != nil {
		if saved {
			n.state = nonceUsed
		} else {
			n.state = nonceFresh
		}
	}
}

// photoMatch is a candidate from GYST's own foods, per 100 g, to prefill the
// review. Values come from the database or label, never from the model.
type photoMatch struct {
	Ref    string           `json:"food_ref"` // "nf:<id>"
	Name   string           `json:"name"`
	Source string           `json:"source"` // "My foods" or "database"
	Per100 models.Nutrients `json:"per_100g"`
}

type photoReviewItem struct {
	healthcoach.FoodPhotoItem
	Idx   int         `json:"idx"`
	Match *photoMatch `json:"match"`
	// AskGrams: low confidence or likely hidden. The review leaves grams
	// blank so Jay must type a value; the estimate is still kept server-side.
	AskGrams bool `json:"ask_grams"`
}

type photoReview struct {
	Items         []photoReviewItem `json:"items"`
	Uncertainties []string          `json:"uncertainties"`
	Questions     []string          `json:"questions"`
	Nonce         string            `json:"nonce"`
}

// buildPhotoReview matches each identified name against GYST's foods and
// issues the nonce. A name with no per-100 g match gets none: the review asks
// Jay for the values rather than inventing them.
func (h *Handler) buildPhotoReview(r *http.Request, res healthcoach.FoodPhotoResult) photoReview {
	out := photoReview{Items: make([]photoReviewItem, 0, len(res.Items)), Uncertainties: res.Uncertainties, Questions: res.Questions}
	est := make([]*float64, len(res.Items))
	for i, it := range res.Items {
		est[i] = it.EstGrams
		out.Items = append(out.Items, photoReviewItem{FoodPhotoItem: it, Idx: i, Match: h.matchFood(r, it.Name), AskGrams: it.Confidence < 0.5 || !it.Visible})
	}
	out.Nonce = h.camera.issue(sessionToken(r), est, h.Service.Now())
	return out
}

// per100Of is a catalogue food's nutrients per 100 g, when that can be known.
func per100Of(f models.NutritionFood) (models.Nutrients, bool) {
	per := f.Nutrients
	switch {
	case f.Basis == "per_100g":
	case f.Basis == "per_serving" && f.ServingGrams != nil && *f.ServingGrams > 0:
		per = scalePer(f.Nutrients, 100 / *f.ServingGrams)
	default:
		return per, false
	}
	return per, per.Any()
}

func (h *Handler) matchFood(r *http.Request, name string) *photoMatch {
	foods, err := h.Service.SearchFoods(r.Context(), name)
	if err != nil {
		return nil
	}
	for _, f := range foods {
		if !strings.HasPrefix(f.Ref, "nf:") { // only rows a diary item can point at
			continue
		}
		per, ok := per100Of(f.NutritionFood)
		if !ok {
			continue
		}
		return &photoMatch{Ref: f.Ref, Name: f.Name, Source: "database", Per100: per}
	}
	return nil
}

func scalePer(n models.Nutrients, k float64) models.Nutrients {
	m := func(v *float64) *float64 {
		if v == nil {
			return nil
		}
		x := *v * k
		return &x
	}
	return models.Nutrients{Kcal: m(n.Kcal), ProteinG: m(n.ProteinG), CarbsG: m(n.CarbsG), FatG: m(n.FatG), FiberG: m(n.FiberG)}
}

type cameraConfirmItem struct {
	Idx     *int     `json:"idx"`
	FoodRef string   `json:"food_ref"` // set only while the values are the untouched prefill
	Name    string   `json:"name"`
	Grams   float64  `json:"grams"`
	Kcal    *float64 `json:"kcal"`
	P       *float64 `json:"protein_g"`
	C       *float64 `json:"carbs_g"`
	F       *float64 `json:"fat_g"`
	Fib     *float64 `json:"fiber_g"`
}

type cameraConfirm struct {
	Nonce string              `json:"nonce"`
	Slot  string              `json:"slot"`
	Items []cameraConfirmItem `json:"items"`
}

// healthFoodPhotoConfirm saves a reviewed photo meal: POST
// /health/nutrition/photo/confirm with JSON.
func (h *Handler) healthFoodPhotoConfirm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		methodNotAllowed(w)
		return
	}
	if requireSameOrigin(w, r) {
		return
	}
	var in cameraConfirm
	r.Body = http.MaxBytesReader(w, r.Body, cameraConfirmBytes)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONError(w, http.StatusBadRequest, "That review could not be read. Take the photo again.")
		return
	}
	n, res := h.camera.claim(in.Nonce, sessionToken(r), h.Service.Now())
	switch res {
	case claimInvalid:
		writeJSONError(w, http.StatusGone, "This review expired. Take the photo again.")
		return
	case claimInflight:
		writeJSONError(w, http.StatusConflict, "Still saving\u2026")
		return
	case claimAlready:
		writeCameraDone(w, true)
		return
	}
	saved := false
	defer func() { h.camera.finish(in.Nonce, saved) }()

	if in.Slot == "" {
		in.Slot = slotForHour(h.Service.Now().In(h.Service.Location()).Hour())
	}
	meal := services.CameraMeal{IdempotencyKey: in.Nonce, MealSlot: in.Slot}
	seen := map[int]bool{}
	for _, it := range in.Items {
		if it.Idx != nil {
			if seen[*it.Idx] {
				writeJSONError(w, http.StatusBadRequest, "Check the review: the same photo item appears twice.")
				return
			}
			seen[*it.Idx] = true
		}
		ci := services.CameraItem{Name: it.Name, Grams: it.Grams,
			Nutrients: models.Nutrients{Kcal: it.Kcal, ProteinG: it.P, CarbsG: it.C, FatG: it.F, FiberG: it.Fib}}
		if it.Idx != nil && *it.Idx >= 0 && *it.Idx < len(n.est) {
			ci.AIEstGrams = n.est[*it.Idx]
		}
		h.linkPrefill(&ci, it.FoodRef)
		meal.Items = append(meal.Items, ci)
	}
	out, err := h.Service.AddCameraMeal(r.Context(), diaryActor(r), meal)
	var invalid *services.HealthValidationError
	switch {
	case errors.As(err, &invalid):
		writeJSONError(w, http.StatusBadRequest, "Check the review: "+invalid.Error()+".")
		return
	case err != nil:
		h.log().Error("camera meal save failed", "request_id", requestIDFrom(r.Context()))
		writeJSONError(w, http.StatusInternalServerError, "Could not save the meal. Nothing was saved. Try again.")
		return
	}
	saved = true
	writeCameraDone(w, !out.Created)
}

func writeCameraDone(w http.ResponseWriter, already bool) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	redirect := diaryPage + "?added=1#diary"
	if already {
		redirect = diaryPage + "?dup=1#diary"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "already": already, "redirect": redirect})
}

// linkPrefill points the item at the catalogue food it was prefilled from,
// but only when the values still equal that food's data for these grams (to
// the 0.1 shown in the review). Anything else stays Jay's own ("user", no
// food), so an edited value is never attributed to the database.
func (h *Handler) linkPrefill(ci *services.CameraItem, ref string) {
	kind, raw, _ := strings.Cut(ref, ":")
	id, err := strconv.ParseInt(raw, 10, 64)
	if kind != "nf" || err != nil || id <= 0 {
		return
	}
	f, err := h.Service.Repo.GetNutritionFood(id)
	if err != nil || f.SupersededAt != "" || f.ArchivedAt != "" ||
		(f.Provider == "user" && strings.HasPrefix(f.ProviderID, "health_food:")) {
		return
	}
	per, ok := per100Of(f)
	if !ok {
		return
	}
	scaled := scalePer(per, ci.Grams/100)
	same := func(got, want *float64) bool {
		if got == nil || want == nil {
			return got == nil && want == nil
		}
		return math.Abs(*got-*want) <= 0.051
	}
	n := ci.Nutrients
	if !same(n.Kcal, scaled.Kcal) || !same(n.ProteinG, scaled.ProteinG) || !same(n.CarbsG, scaled.CarbsG) ||
		!same(n.FatG, scaled.FatG) || !same(n.FiberG, scaled.FiberG) {
		return
	}
	foodID, ver := f.ID, f.Version
	ci.FoodID, ci.FoodVersion, ci.NutritionProv = &foodID, &ver, f.DataProvenance
}
