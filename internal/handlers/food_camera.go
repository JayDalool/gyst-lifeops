package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
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
	c.m[nonce] = &cameraNonce{session: session, expires: now.Add(cameraNonceTTL), est: est}
	return nonce
}

type claimResult int

const (
	claimOK claimResult = iota
	claimInvalid
	claimAlready
)

// claim marks a fresh nonce in flight. A nonce that is unknown, expired or
// issued to another session is invalid; one already in flight or used is
// "already".
func (c *cameraStore) claim(nonce, session string, now time.Time) (*cameraNonce, claimResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := c.m[nonce]
	if n == nil || now.After(n.expires) || n.session != session {
		return nil, claimInvalid
	}
	if n.state != nonceFresh {
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
	Name   string           `json:"name"`
	Source string           `json:"source"` // "My foods" or "database"
	Per100 models.Nutrients `json:"per_100g"`
}

type photoReviewItem struct {
	healthcoach.FoodPhotoItem
	Idx   int         `json:"idx"`
	Match *photoMatch `json:"match"`
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
		out.Items = append(out.Items, photoReviewItem{FoodPhotoItem: it, Idx: i, Match: h.matchFood(r, it.Name)})
	}
	out.Nonce = h.camera.issue(sessionToken(r), est, h.Service.Now())
	return out
}

func (h *Handler) matchFood(r *http.Request, name string) *photoMatch {
	foods, err := h.Service.SearchFoods(r.Context(), name)
	if err != nil {
		return nil
	}
	for _, f := range foods {
		per := f.Nutrients
		switch {
		case f.Basis == "per_100g":
		case f.Basis == "per_serving" && f.ServingGrams != nil && *f.ServingGrams > 0:
			per = scalePer(f.Nutrients, 100 / *f.ServingGrams)
		default:
			continue
		}
		if !per.Any() {
			continue
		}
		src := "database"
		if f.Provider == "user" {
			src = "My foods"
		}
		return &photoMatch{Name: f.Name, Source: src, Per100: per}
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
	Idx   *int     `json:"idx"`
	Name  string   `json:"name"`
	Grams float64  `json:"grams"`
	Kcal  *float64 `json:"kcal"`
	P     *float64 `json:"protein_g"`
	C     *float64 `json:"carbs_g"`
	F     *float64 `json:"fat_g"`
	Fib   *float64 `json:"fiber_g"`
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
	for _, it := range in.Items {
		ci := services.CameraItem{Name: it.Name, Grams: it.Grams,
			Nutrients: models.Nutrients{Kcal: it.Kcal, ProteinG: it.P, CarbsG: it.C, FatG: it.F, FiberG: it.Fib}}
		if it.Idx != nil && *it.Idx >= 0 && *it.Idx < len(n.est) {
			ci.AIEstGrams = n.est[*it.Idx]
		}
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
