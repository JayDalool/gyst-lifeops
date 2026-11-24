package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// Food diary on Nutrition (Issue #216, over the #207 service). Every write
// is a POST behind the session and postGuard, answers 303 to the diary, and
// re-renders the page with 400 when the input is refused. No JavaScript:
// Undo is a form that posts to the restore route.

const diaryPage = "/health/nutrition"

type diarySlotOption struct{ Slot, Label string }

var diarySlotOrder = []diarySlotOption{
	{"breakfast", "Breakfast"}, {"lunch", "Lunch"}, {"dinner", "Dinner"}, {"snack", "Snack"},
}

type diaryItemView struct {
	ID       int64
	Name     string
	Amount   string // "120 g" or "1.5 servings"
	AmountIn string // the number alone, for the edit field
	UnitName string // "grams" or "servings"
	Kcal     string // "" when unknown
	// AIEstimate marks a row whose nutrition is an AI estimate (GYST-180); its
	// calories read "about N kcal" and carry a chip.
	AIEstimate bool
}

type diaryGroupView struct {
	Label string
	Items []diaryItemView
}

type diaryResultView struct {
	Ref      string // "nf:<id>" or "hf:<id>"
	Name     string
	Detail   string // basis and calories
	Key      string // idempotency key of this rendered form
	UnitName string
	Default  string
}

type nutritionDiary struct {
	Groups   []diaryGroupView
	Count    int
	Query    string
	Searched bool
	Results  []diaryResultView
	Slots    []diarySlotOption
	Preslot  string
	Error    string
	Added    bool  // just added
	Already  bool  // repeated idempotency key
	Updated  bool  // amount changed
	Restored bool  // Undo done
	Removed  int64 // id to offer Undo for
}

// IntakeNote says which source counts for the day, in plain words.
func (v HealthTodayView) IntakeNote() string {
	switch v.IntakeSource {
	case models.IntakeSourceGYST:
		n := "from your GYST diary"
		if v.ReferenceKcal.Present {
			n += " · Apple Health " + v.ReferenceKcal.Value + " kcal, not counted"
		}
		return n
	case models.IntakeSourceAppleHealth:
		return "from Apple Health (MyFitnessPal)"
	}
	return ""
}

// PartialNote is "partial · n items without a value" for a macro with gaps.
func (m HealthMacroView) PartialNote() string {
	switch {
	case m.Gaps == 0:
		return ""
	case m.Gaps == 1:
		return "partial · 1 item without a value"
	}
	return "partial · " + strconv.Itoa(m.Gaps) + " items without a value"
}

func newDiaryKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// slotForHour picks the meal slot a new entry starts on.
func slotForHour(hour int) string {
	switch {
	case hour < 11:
		return "breakfast"
	case hour < 15:
		return "lunch"
	case hour < 21:
		return "dinner"
	}
	return "snack"
}

// foodUnit: a per-100 g food is logged in grams, a per-serving food in servings.
func foodUnit(f models.NutritionFood) (unit, name, def string) {
	if f.Basis == "per_100g" {
		return "g", "grams", "100"
	}
	return "serving", "servings", "1"
}

func diaryAmountText(it models.DiaryItem) (shown, plain string) {
	plain = niceNumber(it.Amount)
	switch {
	case it.Unit == "g":
		return plain + " g", plain
	case it.Amount == 1:
		return "1 serving", plain
	}
	return plain + " servings", plain
}

// fillDiary loads today's diary and, for ?q=, the search results. It returns
// a message when the page cannot show them. It only reads: My foods come
// from health_foods and reach the catalogue only when one is logged.
func (h *Handler) fillDiary(r *http.Request, data *nutritionData) string {
	d := &data.Diary
	d.Slots = diarySlotOrder
	d.Preslot = slotForHour(h.Service.Now().In(h.Service.Location()).Hour())
	q := r.URL.Query()
	d.Added, d.Already, d.Updated, d.Restored = q.Get("added") == "1", q.Get("dup") == "1", q.Get("updated") == "1", q.Get("restored") == "1"
	if id, err := strconv.ParseInt(q.Get("removed"), 10, 64); err == nil && id > 0 {
		d.Removed = id
	}
	items, err := h.Service.ListDiaryItems(h.Service.TodayString())
	if err != nil {
		return "The diary could not be loaded."
	}
	d.Count = len(items)
	for _, slot := range diarySlotOrder {
		g := diaryGroupView{Label: slot.Label}
		for _, it := range items {
			if it.MealSlot != slot.Slot {
				continue
			}
			shown, plain := diaryAmountText(it)
			v := diaryItemView{ID: it.ID, Name: it.Name, Amount: shown, AmountIn: plain, UnitName: map[string]string{"g": "grams", "serving": "servings"}[it.Unit]}
			if it.Kcal != nil {
				v.Kcal = formatThousands(*it.Kcal)
			}
			v.AIEstimate = it.NutritionProvenance == models.NutritionProvenanceAIEstimate
			g.Items = append(g.Items, v)
		}
		if len(g.Items) > 0 {
			d.Groups = append(d.Groups, g)
		}
	}
	d.Query = strings.TrimSpace(q.Get("q"))
	if d.Query == "" {
		return ""
	}
	if utf8.RuneCountInString(d.Query) > 80 {
		d.Query = string([]rune(d.Query)[:80])
		return "Search text must be 80 characters or fewer."
	}
	foods, err := h.Service.SearchFoods(r.Context(), d.Query)
	var invalid *services.HealthValidationError
	if errors.As(err, &invalid) {
		return invalid.Error()
	}
	if err != nil {
		return "Search failed. Try again."
	}
	d.Searched = true
	for _, f := range foods {
		_, name, def := foodUnit(f.NutritionFood)
		detail := "per 100 g"
		if f.Basis == "per_serving" {
			detail = "per serving"
			if f.ServingLabel != "" {
				detail = "per " + f.ServingLabel
			}
		}
		if f.Kcal != nil {
			detail += " · " + formatThousands(*f.Kcal) + " kcal"
		} else {
			detail += " · kcal unknown"
		}
		d.Results = append(d.Results, diaryResultView{Ref: f.Ref, Name: f.Name, Detail: detail, Key: newDiaryKey(), UnitName: name, Default: def})
	}
	return ""
}

func diaryActor(r *http.Request) services.DiaryActor {
	ingress, _, _ := loginClient(r)
	return services.DiaryActor{Ref: services.SessionActorRef(sessionToken(r)), Ingress: ingress, RequestID: requestIDFrom(r.Context())}
}

const amountHelp = "Amount must be a number. Use a dot or one comma for decimals, and no thousands separators."

// parseAmount reads a typed amount with the shared rule, refusing thousands
// separators so 1,000 never becomes 1.
func parseAmount(raw string) (float64, bool) { return services.ParseTypedNumber(raw, false) }

// nutritionItems routes /health/nutrition/items[/{id}[/{archive|restore}]].
func (h *Handler) nutritionItems(w http.ResponseWriter, r *http.Request, rest []string) {
	if !postGuard(w, r, "POST") {
		return
	}
	var id int64
	if len(rest) > 0 {
		var err error
		if id, err = strconv.ParseInt(rest[0], 10, 64); err != nil || id <= 0 {
			http.NotFound(w, r)
			return
		}
	}
	if len(rest) > 2 || (len(rest) == 2 && rest[1] != "archive" && rest[1] != "restore") {
		http.NotFound(w, r)
		return
	}
	var err error
	redirect := diaryPage + "?updated=1#diary"
	switch {
	case len(rest) == 0:
		redirect, err = h.diaryAdd(r)
	case len(rest) == 1:
		amount, ok := parseAmount(r.PostFormValue("amount"))
		if !ok {
			err = &services.HealthValidationError{Message: amountHelp}
			break
		}
		err = h.Service.UpdateDiaryItemAmount(r.Context(), diaryActor(r), id, amount)
	case rest[1] == "archive":
		err = h.Service.ArchiveDiaryItem(r.Context(), diaryActor(r), id)
		redirect = diaryPage + "?removed=" + strconv.FormatInt(id, 10) + "#diary"
	default:
		err = h.Service.RestoreDiaryItem(r.Context(), diaryActor(r), id)
		redirect = diaryPage + "?restored=1#diary"
	}
	var invalid *services.HealthValidationError
	switch {
	case err == nil:
		http.Redirect(w, r, redirect, http.StatusSeeOther)
	case errors.As(err, &invalid):
		h.diaryRefuse(w, r, invalid.Error())
	case errors.Is(err, services.ErrDiaryItemNotFound):
		h.diaryRefuse(w, r, "That entry is already removed, or Undo is no longer possible.")
	default:
		serverError(w, err)
	}
}

func (h *Handler) diaryAdd(r *http.Request) (string, error) {
	amount, ok := parseAmount(r.PostFormValue("amount"))
	if !ok {
		return "", &services.HealthValidationError{Message: amountHelp}
	}
	kind, rawID, _ := strings.Cut(r.PostFormValue("food_ref"), ":")
	id, _ := strconv.ParseInt(rawID, 10, 64)
	in := models.NewDiaryItem{IdempotencyKey: r.PostFormValue("key"), MealSlot: r.PostFormValue("slot"), Amount: amount}
	switch kind {
	case "hf": // a My foods row is always per serving
		in.HealthFoodID, in.Unit = id, "serving"
	case "nf":
		food, err := h.Service.Repo.GetNutritionFood(id)
		if err != nil {
			return "", &services.HealthValidationError{Message: "Food not found. Search again."}
		}
		in.FoodID = id
		in.Unit, _, _ = foodUnit(food)
	default:
		return "", &services.HealthValidationError{Message: "Food not found. Search again."}
	}
	res, err := h.Service.AddDiaryItem(r.Context(), diaryActor(r), in)
	if err != nil {
		return "", err
	}
	if !res.Created {
		return diaryPage + "?dup=1#diary", nil
	}
	return diaryPage + "?added=1#diary", nil
}

// diaryRefuse re-renders Nutrition with the reason and status 400. The
// search text stays so the results are still there.
func (h *Handler) diaryRefuse(w http.ResponseWriter, r *http.Request, msg string) {
	if q := strings.TrimSpace(r.PostFormValue("q")); q != "" {
		u := *r.URL
		vals := u.Query()
		vals.Set("q", q)
		u.RawQuery = vals.Encode()
		r.URL = &u
	}
	h.renderNutrition(w, r, http.StatusBadRequest, msg)
}
