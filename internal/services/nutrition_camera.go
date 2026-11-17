package services

import (
	"context"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

// AI food camera save (Issue #211). Jay has reviewed and edited every item;
// this writes one camera meal. Items are free-form (no catalogue row): the
// nutrition values are the ones Jay confirmed, so their provenance is "user".
// The model never supplies nutrient values.

const maxCameraItems = 20

// CameraItem is one reviewed item. AIEstGrams is the server-held AI estimate
// for this item (nil when the item was added by hand or had no estimate).
type CameraItem struct {
	Name       string
	Grams      float64
	AIEstGrams *float64
	Nutrients  models.Nutrients
}

// CameraMeal is the input of AddCameraMeal.
type CameraMeal struct {
	IdempotencyKey string
	MealSlot       string
	Items          []CameraItem
}

// AddCameraMeal saves a reviewed photo meal on today's date with
// created_via=camera. The same idempotency key never creates a second meal.
// Grams equal to the AI estimate record quantity_provenance ai_estimate (Jay
// accepted it); any other value records user_confirmed.
func (s *Service) AddCameraMeal(ctx context.Context, actor DiaryActor, in CameraMeal) (DiaryResult, error) {
	if !diaryKeyPattern.MatchString(in.IdempotencyKey) {
		return DiaryResult{}, healthInvalid("the review is out of date; take the photo again")
	}
	if !diarySlots[in.MealSlot] {
		return DiaryResult{}, healthInvalid("choose breakfast, lunch, dinner or snack")
	}
	if len(in.Items) == 0 || len(in.Items) > maxCameraItems {
		return DiaryResult{}, healthInvalid("add at least one food, and no more than 20")
	}
	rows := make([]repository.DiaryItemInsert, 0, len(in.Items))
	for _, it := range in.Items {
		name := strings.Join(strings.Fields(it.Name), " ")
		if name == "" || utf8.RuneCountInString(name) > 120 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
			return DiaryResult{}, healthInvalid("each food needs a name of 1 to 120 characters")
		}
		if !validDiaryAmount(it.Grams) {
			return DiaryResult{}, healthInvalid("grams must be more than 0 and at most 100,000")
		}
		grams := it.Grams
		if err := checkScaled(it.Nutrients, &grams); err != nil {
			return DiaryResult{}, err
		}
		row := repository.DiaryItemInsert{
			Name: name, Unit: "g", Amount: grams, Grams: &grams,
			QuantityProvenance: "user_confirmed", NutritionProv: "user", Nutrients: it.Nutrients,
		}
		if e := it.AIEstGrams; e != nil && !math.IsNaN(*e) && *e >= 0 && *e <= diaryMaxAmount {
			est := *e
			row.AIEstGrams = &est
			if est == grams {
				row.QuantityProvenance = "ai_estimate"
			}
		}
		rows = append(rows, row)
	}
	now := s.Now()
	meal := repository.DiaryMealInsert{
		LocalDate: s.DateOf(now), MealSlot: in.MealSlot, CreatedVia: "camera",
		IdempotencyKey: in.IdempotencyKey, CreatedAt: clock.FormatInstant(now),
	}
	mealID, ids, created, err := s.Repo.CreateMealWithItems(ctx, meal, rows)
	if err != nil {
		return DiaryResult{}, err
	}
	res := DiaryResult{MealID: mealID, Created: created}
	if len(ids) > 0 {
		res.ItemID = ids[0]
	}
	if created {
		s.auditDiary(ctx, actor, AuditActionNutritionMealCreate, "meal:"+strconv.FormatInt(mealID, 10), "camera meal added")
	}
	return res, nil
}
