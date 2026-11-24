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
	// AIEstNutrients is the AI's own nutrition at AIEstGrams, held server-side
	// per nonce; nil until the photo AI supplies any (GYST-180). Never from the
	// browser.
	AIEstNutrients *models.Nutrients
	Nutrients      models.Nutrients
	// FoodID/FoodVersion/NutritionProv are set only when the values are the
	// untouched data of that catalogue row; otherwise the item is Jay's own.
	FoodID, FoodVersion *int64
	NutritionProv       string
}

// CameraMeal is the input of AddCameraMeal.
type CameraMeal struct {
	IdempotencyKey string
	MealSlot       string
	Items          []CameraItem
	// Portion is the whole-meal slider (x0.5 to x2 in 5% steps); nil means x1.
	// The browser has already scaled each item's grams, so it only decides
	// provenance here: moving the slider is Jay's change (GYST-179).
	Portion *float64
}

// CameraMacroTolerance is how far a posted nutrient may sit from the server's
// own figure: one 0.1 rounding step of the review plus float noise. It only
// decides accept or refuse; the stored value is always the server's.
const CameraMacroTolerance = 0.11

// Round1 rounds to 0.1, the precision the review shows and the diary stores.
func Round1(v float64) float64 { return round1(v) }

// CameraScale is per * grams / base, in the order the review script uses
// ((per*g)/base) so both sides see the same float. nil stays nil.
func CameraScale(per models.Nutrients, grams, base float64) models.Nutrients {
	f := func(v *float64) *float64 {
		if v == nil {
			return nil
		}
		x := *v * grams / base
		return &x
	}
	return models.Nutrients{Kcal: f(per.Kcal), ProteinG: f(per.ProteinG), CarbsG: f(per.CarbsG), FatG: f(per.FatG), FiberG: f(per.FiberG)}
}

// RoundNutrients rounds each present nutrient to 0.1.
func RoundNutrients(n models.Nutrients) models.Nutrients {
	f := func(v *float64) *float64 {
		if v == nil {
			return nil
		}
		x := round1(*v)
		return &x
	}
	return models.Nutrients{Kcal: f(n.Kcal), ProteinG: f(n.ProteinG), CarbsG: f(n.CarbsG), FatG: f(n.FatG), FiberG: f(n.FiberG)}
}

// nutrientPairs lines the five nutrients up for a field-by-field comparison.
func nutrientPairs(a, b models.Nutrients) [5][2]*float64 {
	return [5][2]*float64{{a.Kcal, b.Kcal}, {a.ProteinG, b.ProteinG}, {a.CarbsG, b.CarbsG}, {a.FatG, b.FatG}, {a.FiberG, b.FiberG}}
}

// NutrientsContradict reports whether a posted value is present and more than
// the tolerance away from the expected one, or present where none is expected.
// A posted nil never contradicts.
func NutrientsContradict(posted, expected models.Nutrients) bool {
	for _, p := range nutrientPairs(posted, expected) {
		if p[0] == nil {
			continue
		}
		if p[1] == nil || math.Abs(*p[0]-*p[1]) > CameraMacroTolerance {
			return true
		}
	}
	return false
}

// nutrientsEqual is true when every nutrient is present on both sides with the
// same value (to float noise), or absent on both.
func nutrientsEqual(posted, shown models.Nutrients) bool {
	for _, p := range nutrientPairs(posted, shown) {
		if (p[0] == nil) != (p[1] == nil) || (p[0] != nil && math.Abs(*p[0]-*p[1]) > 1e-6) {
			return false
		}
	}
	return true
}

func validPortion(p float64) bool {
	return !math.IsNaN(p) && !math.IsInf(p, 0) && p >= 0.5 && p <= 2 && math.Abs(p*20-math.Round(p*20)) < 1e-6
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
	portion := 1.0
	if in.Portion != nil {
		portion = *in.Portion
	}
	if !validPortion(portion) {
		return DiaryResult{}, healthInvalid("portion must be between x0.5 and x2 in 5% steps")
	}
	rows := make([]repository.DiaryItemInsert, 0, len(in.Items))
	for _, it := range in.Items {
		name := strings.Join(strings.Fields(it.Name), " ")
		if name == "" || utf8.RuneCountInString(name) > 120 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
			return DiaryResult{}, healthInvalid("each food needs a name of 1 to 120 characters")
		}
		// Grams are stored to 0.1, the precision the review shows; a value that
		// rounds to 0 is refused with the rest of the non-positive ones.
		grams := round1(it.Grams)
		if !validDiaryAmount(grams) {
			return DiaryResult{}, healthInvalid("grams must be more than 0 and at most 100,000")
		}
		if err := checkScaled(it.Nutrients, &grams); err != nil {
			return DiaryResult{}, err
		}
		row := repository.DiaryItemInsert{
			Name: name, Unit: "g", Amount: grams, Grams: &grams,
			QuantityProvenance: "user_confirmed", NutritionProv: "user", Nutrients: it.Nutrients,
		}
		matched := it.FoodID != nil && it.FoodVersion != nil && it.NutritionProv != ""
		if matched {
			row.FoodID, row.FoodVersion, row.NutritionProv = it.FoodID, it.FoodVersion, it.NutritionProv
		}
		if e := it.AIEstGrams; e != nil && !math.IsNaN(*e) && *e >= 0 && *e <= diaryMaxAmount {
			est := *e
			row.AIEstGrams = &est
			// The review shows the estimate to 0.1 g; accepting that number
			// counts as accepting the estimate. Moving the whole-meal slider is
			// Jay's change even when rounding lands on the estimate again.
			if portion == 1 && round1(est) == round1(grams) {
				row.QuantityProvenance = "ai_estimate"
			}
			if n := it.AIEstNutrients; n != nil && n.Any() && est > 0 && checkScaled(*n, nil) == nil {
				row.AIEst = n // kept as the AI gave them, whatever Jay does next
				if !matched {
					// Unedited AI nutrition stays "ai_estimate", stored as the
					// server's own scaling. "Unedited" means exactly the value
					// the review displayed (the estimate at the posted grams,
					// to 0.1): any difference, even 0.1, is Jay's entry and is
					// kept as posted (docs/RULES.md 3.2).
					shown := RoundNutrients(CameraScale(*n, it.Grams, est))
					stored := RoundNutrients(CameraScale(*n, grams, est))
					if nutrientsEqual(it.Nutrients, shown) {
						row.Nutrients = stored
						row.NutritionProv = "ai_estimate"
					} else {
						// A partial edit makes the item Jay's (provenance user),
						// but a value he left as shown is still the estimate:
						// store the server's own scaling for it so the day total
						// can tell which values are still AI (GYST-180).
						keep := func(posted, show, st *float64) *float64 {
							if posted != nil && show != nil && math.Abs(*posted-*show) <= 1e-6 {
								return st
							}
							return posted
						}
						row.Nutrients.Kcal = keep(it.Nutrients.Kcal, shown.Kcal, stored.Kcal)
						row.Nutrients.ProteinG = keep(it.Nutrients.ProteinG, shown.ProteinG, stored.ProteinG)
						row.Nutrients.CarbsG = keep(it.Nutrients.CarbsG, shown.CarbsG, stored.CarbsG)
						row.Nutrients.FatG = keep(it.Nutrients.FatG, shown.FatG, stored.FatG)
					}
				}
			}
		}
		// The final values (rescaled at the stored grams) face the same limits as
		// the table's CHECKs, so a value that crossed them is the review's 400.
		if err := checkScaled(row.Nutrients, row.Grams); err != nil {
			return DiaryResult{}, err
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

func round1(v float64) float64 { return math.Round(v*10) / 10 }
