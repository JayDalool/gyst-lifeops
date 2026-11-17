package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

// Food diary writes (Issue #207). Create, edit, archive and restore act on
// today's items only; editing past days waits for the confirmation policy of
// Issue #212. Every write is one transaction; the audit row is written after
// it commits and holds fixed words only, never a food name, amount or
// nutrient (docs/audit/vnext/09_SECURITY_FOUNDATION.md B.8).

// DiaryUndoWindow is how long after removing an item it can be restored.
const DiaryUndoWindow = 10 * time.Minute

// ErrDiaryItemNotFound means the item does not exist or is not in the state
// the change needs (for example, restoring an item that is not removed).
var ErrDiaryItemNotFound = errors.New("diary item not found")

const (
	diaryMaxAmount      = 100000
	diarySearchMaxChars = 80
	diarySearchLimit    = 20
)

var (
	diaryKeyPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
	diarySlots      = map[string]bool{"breakfast": true, "lunch": true, "dinner": true, "snack": true}
	diaryVia        = map[string]bool{"manual": true, "camera": true, "chat": true, "barcode": true}
	diaryUnits      = map[string]bool{"g": true, "serving": true}
	diaryQuantity   = map[string]bool{"user_confirmed": true, "ai_estimate": true, "usual_portion": true, "recipe_serving": true}
)

// DiaryActor identifies who made a diary change, for the audit row only.
// Ref must match the audit ref pattern (for example "session:ab12cd34").
// The zero value is an internal human actor.
type DiaryActor struct {
	Ref       string
	Ingress   string
	RequestID string
}

// NormalizeFoodName is the search form of a name: lower case, single spaces.
func NormalizeFoodName(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// DiaryResult is what a diary write reports.
type DiaryResult struct {
	MealID  int64
	ItemID  int64
	Created bool // false when an idempotent repeat found the meal already there
}

// healthFoodAsCatalogue is a My foods row (health_foods, per serving) in the
// catalogue's form: provider "user", id "health_food:<id>". Nothing is written;
// AddDiaryItem mirrors it inside its own transaction.
func healthFoodAsCatalogue(f models.HealthFood) models.NutritionFood {
	return models.NutritionFood{
		Provider: "user", ProviderID: "health_food:" + strconv.FormatInt(f.ID, 10),
		Name: f.Name, NormalizedName: NormalizeFoodName(f.Name),
		Basis: "per_serving", ServingLabel: f.Serving,
		Nutrients:      models.Nutrients{Kcal: f.Kcal, ProteinG: f.ProteinG, CarbsG: f.CarbsG, FatG: f.FatG, FiberG: f.FiberG},
		DataProvenance: "user",
	}
}

// FoodChoice is one search result. Ref is what a form posts back: "nf:<id>" for
// a catalogue row, "hf:<id>" for a My foods row.
type FoodChoice struct {
	Ref string
	models.NutritionFood
}

// SearchFoods finds foods by name. It only reads: active My foods are read
// from health_foods at their current values, and catalogue rows come from
// nutrition_foods (current versions, not archived). A catalogue row that
// mirrors a My food is skipped, since the My foods row is the live source.
func (s *Service) SearchFoods(ctx context.Context, query string) ([]FoodChoice, error) {
	q := NormalizeFoodName(query)
	if q == "" || utf8.RuneCountInString(q) > diarySearchMaxChars {
		return nil, healthInvalid("search text must be 1 to 80 characters")
	}
	mine, err := s.Repo.ListHealthFoods(false)
	if err != nil {
		return nil, err
	}
	out := []FoodChoice{}
	for _, f := range mine {
		if f.Active && strings.Contains(NormalizeFoodName(f.Name), q) {
			out = append(out, FoodChoice{Ref: "hf:" + strconv.FormatInt(f.ID, 10), NutritionFood: healthFoodAsCatalogue(f)})
		}
	}
	found, err := s.Repo.SearchNutritionFoods(q, diarySearchLimit)
	if err != nil {
		return nil, err
	}
	for _, f := range found {
		if f.Provider == "user" && strings.HasPrefix(f.ProviderID, "health_food:") {
			continue
		}
		out = append(out, FoodChoice{Ref: "nf:" + strconv.FormatInt(f.ID, 10), NutritionFood: f})
	}
	if len(out) > diarySearchLimit {
		out = out[:diarySearchLimit]
	}
	return out, nil
}

// checkScaled refuses a scaled entry that the database limits would reject,
// with a message instead of a constraint error.
func checkScaled(n models.Nutrients, grams *float64) error {
	tooBig := func(v *float64, max float64) bool {
		return v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || *v < 0 || *v > max)
	}
	if tooBig(grams, 100000) || tooBig(n.Kcal, 100000) || tooBig(n.ProteinG, 10000) || tooBig(n.CarbsG, 10000) ||
		tooBig(n.FatG, 10000) || tooBig(n.FiberG, 10000) {
		return healthInvalid("that amount is too large for this food; check the amount")
	}
	return nil
}

func (a DiaryActor) event(action, target, detail, outcome string) repository.AuditEvent {
	ingress := a.Ingress
	if ingress == "" {
		ingress = AuditIngressInternal
	}
	return repository.AuditEvent{ActorType: AuditActorHuman, ActorRef: a.Ref, Ingress: ingress, Action: action,
		TargetRef: target, Outcome: outcome, Detail: detail, RequestID: a.RequestID}
}

// auditDiary writes the audit row after the change committed. A failure is
// logged without content and never undoes the change.
func (s *Service) auditDiary(ctx context.Context, a DiaryActor, action, target, detail string) {
	if err := s.RecordAudit(ctx, a.event(action, target, detail, AuditOutcomeSuccess)); err != nil {
		slog.Error("audit write failed", "action", action)
	}
}

// AddDiaryItem logs one food into a new meal on today's date. The same
// idempotency key never creates a second meal and writes no second audit row.
func (s *Service) AddDiaryItem(ctx context.Context, actor DiaryActor, in models.NewDiaryItem) (DiaryResult, error) {
	if !diaryKeyPattern.MatchString(in.IdempotencyKey) {
		return DiaryResult{}, healthInvalid("the form is out of date; reload and try again")
	}
	if !diarySlots[in.MealSlot] {
		return DiaryResult{}, healthInvalid("choose breakfast, lunch, dinner or snack")
	}
	if in.CreatedVia == "" {
		in.CreatedVia = "manual"
	}
	if in.QuantityProvenance == "" {
		in.QuantityProvenance = "user_confirmed"
	}
	if !diaryVia[in.CreatedVia] || !diaryQuantity[in.QuantityProvenance] {
		return DiaryResult{}, healthInvalid("unsupported entry source")
	}
	if !diaryUnits[in.Unit] {
		return DiaryResult{}, healthInvalid("choose grams or servings")
	}
	if !validDiaryAmount(in.Amount) {
		return DiaryResult{}, healthInvalid("amount must be more than 0 and at most 100,000")
	}
	now := s.Now()
	meal := repository.DiaryMealInsert{
		LocalDate: s.DateOf(now), MealSlot: in.MealSlot, CreatedVia: in.CreatedVia,
		IdempotencyKey: in.IdempotencyKey, CreatedAt: clock.FormatInstant(now),
	}
	unitWord := map[string]string{"g": "grams", "serving": "servings"}[in.Unit]
	itemFor := func(food models.NutritionFood) (repository.DiaryItemInsert, error) {
		nutrients, grams, err := ScaleNutrients(food, in.Unit, in.Amount)
		if err != nil {
			return repository.DiaryItemInsert{}, healthInvalid("this food cannot be logged in " + unitWord + "; it has no serving weight")
		}
		if err := checkScaled(nutrients, grams); err != nil {
			return repository.DiaryItemInsert{}, err
		}
		foodID, version := food.ID, food.Version
		return repository.DiaryItemInsert{
			FoodID: &foodID, FoodVersion: &version, Name: food.Name, Unit: in.Unit, Amount: in.Amount, Grams: grams,
			QuantityProvenance: in.QuantityProvenance, NutritionProv: food.DataProvenance, Nutrients: nutrients,
		}, nil
	}
	var mealID, itemID int64
	var created bool
	if in.HealthFoodID > 0 {
		hf, err := s.Repo.GetHealthFood(in.HealthFoodID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && !hf.Active) {
			return DiaryResult{}, healthInvalid("food not found; search again")
		}
		if err != nil {
			return DiaryResult{}, err
		}
		mirror := healthFoodAsCatalogue(hf)
		if _, err := itemFor(mirror); err != nil { // same checks, before any write
			return DiaryResult{}, err
		}
		// The mirror row is written in the same transaction as the meal, and
		// only when the meal is new (a repeated key writes nothing).
		mealID, itemID, created, err = s.Repo.CreateMealWithMirroredFood(ctx, meal, mirror, clock.FormatInstant(now), itemFor)
		if err != nil {
			return DiaryResult{}, err
		}
	} else {
		food, err := s.Repo.GetNutritionFood(in.FoodID)
		if errors.Is(err, sql.ErrNoRows) {
			return DiaryResult{}, healthInvalid("food not found; search again")
		}
		if err != nil {
			return DiaryResult{}, err
		}
		if food.Provider == "user" && strings.HasPrefix(food.ProviderID, "health_food:") {
			// My foods are logged only by their live row (HealthFoodID), never by a stale mirror.
			return DiaryResult{}, healthInvalid("food not found; search again")
		}
		if food.SupersededAt != "" || food.ArchivedAt != "" {
			return DiaryResult{}, healthInvalid("that food changed; search again")
		}
		item, err := itemFor(food)
		if err != nil {
			return DiaryResult{}, err
		}
		var itemIDs []int64
		mealID, itemIDs, created, err = s.Repo.CreateMealWithItems(ctx, meal, []repository.DiaryItemInsert{item})
		if err != nil {
			return DiaryResult{}, err
		}
		if len(itemIDs) > 0 {
			itemID = itemIDs[0]
		}
	}
	res := DiaryResult{MealID: mealID, ItemID: itemID, Created: created}
	if created {
		s.auditDiary(ctx, actor, AuditActionNutritionMealCreate, "meal:"+strconv.FormatInt(mealID, 10), "item added")
	}
	return res, nil
}

func validDiaryAmount(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v > 0 && v <= diaryMaxAmount
}

// todayItem loads an item and refuses anything that is not on today's date.
func (s *Service) todayItem(id int64) (models.DiaryItem, error) {
	it, err := s.Repo.GetDiaryItem(id)
	if errors.Is(err, sql.ErrNoRows) {
		return it, ErrDiaryItemNotFound
	}
	if err != nil {
		return it, err
	}
	if it.LocalDate != s.TodayString() {
		return it, healthInvalid("only today's entries can be changed")
	}
	return it, nil
}

func diaryTarget(it models.DiaryItem) string { return "meal:" + strconv.FormatInt(it.MealID, 10) }

// UpdateDiaryItemAmount changes the amount of a live item of today, in its
// own unit. Nutrients are recomputed from the food version that was logged,
// so a later catalogue change never alters the entry.
func (s *Service) UpdateDiaryItemAmount(ctx context.Context, actor DiaryActor, id int64, amount float64) error {
	if !validDiaryAmount(amount) {
		return healthInvalid("amount must be more than 0 and at most 100,000")
	}
	it, err := s.todayItem(id)
	if err != nil {
		return err
	}
	if it.ArchivedAt != "" {
		return ErrDiaryItemNotFound
	}
	if it.FoodID == nil {
		return healthInvalid("this entry has no food to rescale")
	}
	food, err := s.Repo.GetNutritionFood(*it.FoodID)
	if err != nil {
		return err
	}
	nutrients, grams, err := ScaleNutrients(food, it.Unit, amount)
	if err != nil {
		return healthInvalid("this food cannot be rescaled in " + it.Unit)
	}
	if err := checkScaled(nutrients, grams); err != nil {
		return err
	}
	if err := s.Repo.UpdateDiaryItemAmount(id, amount, grams, nutrients); err != nil {
		if errors.Is(err, repository.ErrDiaryItemState) {
			return ErrDiaryItemNotFound
		}
		return err
	}
	s.auditDiary(ctx, actor, AuditActionNutritionMealUpdate, diaryTarget(it), "amount changed")
	return nil
}

// ArchiveDiaryItem removes a live item of today from the day's totals. The
// row is kept so it can be restored.
func (s *Service) ArchiveDiaryItem(ctx context.Context, actor DiaryActor, id int64) error {
	it, err := s.todayItem(id)
	if err != nil {
		return err
	}
	if err := s.Repo.ArchiveDiaryItem(id, clock.FormatInstant(s.Now())); err != nil {
		if errors.Is(err, repository.ErrDiaryItemState) {
			return ErrDiaryItemNotFound
		}
		return err
	}
	s.auditDiary(ctx, actor, AuditActionNutritionMealArchive, diaryTarget(it), "item removed")
	return nil
}

// RestoreDiaryItem undoes a removal made within DiaryUndoWindow, on today's
// date. Anything else is ErrDiaryItemNotFound.
func (s *Service) RestoreDiaryItem(ctx context.Context, actor DiaryActor, id int64) error {
	it, err := s.todayItem(id)
	if err != nil {
		return err
	}
	archived, perr := clock.ParseInstant(it.ArchivedAt)
	if it.ArchivedAt == "" || perr != nil || s.Now().Sub(archived) > DiaryUndoWindow || archived.Sub(s.Now()) > time.Minute {
		return ErrDiaryItemNotFound
	}
	if err := s.Repo.RestoreDiaryItem(id); err != nil {
		if errors.Is(err, repository.ErrDiaryItemState) {
			return ErrDiaryItemNotFound
		}
		return err
	}
	s.auditDiary(ctx, actor, AuditActionNutritionMealUpdate, diaryTarget(it), "item restored")
	return nil
}

// ListDiaryItems returns the live items of a local day for display.
func (s *Service) ListDiaryItems(date string) ([]models.DiaryItem, error) {
	if err := ValidateHealthDate(date); err != nil {
		return nil, err
	}
	return s.Repo.ListDiaryItems(date)
}
