package models

import "math"

// Food diary (Issue #207). All nutrient values are nullable: nil means
// "not recorded", never zero.

// Intake sources for one day. Exactly one is counted; they are never summed.
const (
	IntakeSourceGYST        = "gyst"
	IntakeSourceAppleHealth = "apple_health"
	IntakeSourceNone        = "none"
)

// NutritionProvenanceAIEstimate marks item nutrition the photo AI estimated.
const NutritionProvenanceAIEstimate = "ai_estimate"

// Nutrients is one set of the five tracked nutrients.
type Nutrients struct {
	Kcal     *float64 `json:"kcal"`
	ProteinG *float64 `json:"protein_g"`
	CarbsG   *float64 `json:"carbs_g"`
	FatG     *float64 `json:"fat_g"`
	FiberG   *float64 `json:"fiber_g"`
}

// Any reports whether at least one nutrient is recorded.
func (n Nutrients) Any() bool {
	return n.Kcal != nil || n.ProteinG != nil || n.CarbsG != nil || n.FatG != nil || n.FiberG != nil
}

// NutrientGaps counts, per nutrient, the live items whose value is NULL. A
// non-zero count means the matching sum is partial.
type NutrientGaps struct {
	Kcal     int `json:"kcal"`
	ProteinG int `json:"protein_g"`
	CarbsG   int `json:"carbs_g"`
	FatG     int `json:"fat_g"`
	FiberG   int `json:"fiber_g"`
}

// NutritionDayTotals is the sum of the live items of one local day. A sum is
// nil when no live item has that nutrient.
type NutritionDayTotals struct {
	Date      string          `json:"date"`
	Items     int             `json:"items"`
	Sum       Nutrients       `json:"sum"`
	Gaps      NutrientGaps    `json:"gaps"`
	Estimated EstimatedIntake `json:"estimated"`
}

// EstimatedIntake is the part of a GYST day's total that comes from AI macro
// estimates: the live items whose nutrition_provenance is "ai_estimate"
// (GYST-180). It is a subset of Counted, never added to it. Kcal and ProteinG
// are nil when no such item has the value. It is the single source every
// screen, the Coach and the Secretary label from; nothing recounts provenance.
type EstimatedIntake struct {
	Items    int      `json:"items"`
	Kcal     *float64 `json:"kcal"`
	ProteinG *float64 `json:"protein_g"`
}

// DayIntake is the one counted intake of a local day. Source says where
// Counted came from. When Source is gyst, Reference holds the synced Apple
// Health values (shown, not counted). Snapshot is the day's synced row, which
// also carries steps and energy.
type DayIntake struct {
	Date      string               `json:"date"`
	Source    string               `json:"source"`
	Resolved  bool                 `json:"resolved"` // a source supplied values
	Counted   Nutrients            `json:"counted"`
	Missing   NutrientGaps         `json:"missing"`
	Items     int                  `json:"items"`
	Reference *Nutrients           `json:"reference"`
	Snapshot  *DailyHealthSnapshot `json:"snapshot"`
	// Estimated is the AI-estimated share of Counted (GYST days only; an Apple
	// Health or empty day has none).
	Estimated EstimatedIntake `json:"estimated"`
}

// NutritionFood is one immutable catalogue row. Values are per 100 g when
// Basis is per_100g and per serving when it is per_serving.
type NutritionFood struct {
	ID             int64    `json:"id"`
	Provider       string   `json:"provider"`
	ProviderID     string   `json:"provider_id"`
	Version        int64    `json:"version"`
	Name           string   `json:"name"`
	Brand          string   `json:"brand"`
	NormalizedName string   `json:"normalized_name"`
	Basis          string   `json:"basis"`
	ServingLabel   string   `json:"serving_label"`
	ServingGrams   *float64 `json:"serving_grams"`
	Nutrients
	DataProvenance string `json:"data_provenance"`
	DatasetVersion string `json:"dataset_version"`
	CreatedAt      string `json:"created_at"`
	SupersededAt   string `json:"superseded_at"`
	ArchivedAt     string `json:"archived_at"`
}

// DiaryItem is one logged food with its snapshot nutrients.
type DiaryItem struct {
	ID                  int64    `json:"id"`
	MealID              int64    `json:"meal_id"`
	MealSlot            string   `json:"meal_slot"`
	LocalDate           string   `json:"local_date"`
	FoodID              *int64   `json:"food_id"`
	FoodVersion         *int64   `json:"food_version"`
	Name                string   `json:"name"`
	Unit                string   `json:"unit"`
	Amount              float64  `json:"amount"`
	Grams               *float64 `json:"grams"`
	QuantityProvenance  string   `json:"quantity_provenance"`
	NutritionProvenance string   `json:"nutrition_provenance"`
	Nutrients
	CreatedAt  string `json:"created_at"`
	ArchivedAt string `json:"archived_at"`
	// AIEstGrams is the camera's estimate beside the confirmed Grams (#211).
	AIEstGrams *float64 `json:"ai_est_grams"`
	// AIEst* are the AI's own nutrient figures at its estimated grams (GYST-179);
	// nil when the AI gave none, never 0.
	AIEstKcal     *float64 `json:"ai_est_kcal"`
	AIEstProteinG *float64 `json:"ai_est_protein_g"`
	AIEstCarbsG   *float64 `json:"ai_est_carbs_g"`
	AIEstFatG     *float64 `json:"ai_est_fat_g"`
	CreatedVia    string   `json:"created_via"`
}

// AIAccuracyStat is the gap between the AI estimate and the saved value of one
// measure over the items of a week that have both. Bias and MAE are nil, not 0,
// when no item has both (NULL is not zero).
type AIAccuracyStat struct {
	Paired int      `json:"paired"`
	Bias   *float64 `json:"bias"` // mean of saved - estimate
	MAE    *float64 `json:"mae"`  // mean of |saved - estimate|
}

// AIAccuracyWeek is one Monday-start week of camera items (GYST-179).
type AIAccuracyWeek struct {
	WeekStart string         `json:"week_start"`
	Items     int            `json:"items"`
	Grams     AIAccuracyStat `json:"grams"`
	Kcal      AIAccuracyStat `json:"kcal"`
	ProteinG  AIAccuracyStat `json:"protein_g"`
	CarbsG    AIAccuracyStat `json:"carbs_g"`
	FatG      AIAccuracyStat `json:"fat_g"`
}

// NewDiaryItem is the input for logging one food into a new meal.
type NewDiaryItem struct {
	IdempotencyKey     string
	MealSlot           string
	CreatedVia         string
	FoodID             int64 // the catalogue row to log (a specific version)
	HealthFoodID       int64 // a My foods row to log instead; mirrored into the catalogue by the write itself
	Unit               string
	Amount             float64
	QuantityProvenance string
}

// EstimatedValues says which of an item's four AI-estimable values are still
// the AI's (GYST-180).
type EstimatedValues struct{ Kcal, ProteinG, CarbsG, FatG bool }

// Any reports whether at least one value is still an estimate.
func (e EstimatedValues) Any() bool { return e.Kcal || e.ProteinG || e.CarbsG || e.FatG }

// EstimateHolds is the ONE rule for "this saved value is still the AI's
// estimate": it equals the estimate scaled to the saved grams and rounded to
// 0.1, in the order of services.CameraScale and food-camera-math.js (v*g/base,
// then round). The day totals, the diary rows and the Coach all classify
// through it, so a value never counts in one place and not in another.
func EstimateHolds(saved, est *float64, grams, estGrams float64) bool {
	if saved == nil || est == nil || !(grams > 0) || !(estGrams > 0) {
		return false
	}
	return math.Abs(*saved-math.Round(*est*grams/estGrams*10)/10) <= 1e-9
}

// ItemEstimated classifies one live item. Only items saved from a photo review
// with the AI's numbers kept (provenance ai_estimate, or user after an edit)
// can hold estimates; a database or label item never does.
func ItemEstimated(provenance string, grams, estGrams *float64, saved, est Nutrients) EstimatedValues {
	if (provenance != NutritionProvenanceAIEstimate && provenance != "user") || grams == nil || estGrams == nil {
		return EstimatedValues{}
	}
	return EstimatedValues{
		Kcal:     EstimateHolds(saved.Kcal, est.Kcal, *grams, *estGrams),
		ProteinG: EstimateHolds(saved.ProteinG, est.ProteinG, *grams, *estGrams),
		CarbsG:   EstimateHolds(saved.CarbsG, est.CarbsG, *grams, *estGrams),
		FatG:     EstimateHolds(saved.FatG, est.FatG, *grams, *estGrams),
	}
}

// Estimated classifies a diary item the same way the day total does.
func (it DiaryItem) Estimated() EstimatedValues {
	return ItemEstimated(it.NutritionProvenance, it.Grams, it.AIEstGrams,
		Nutrients{Kcal: it.Kcal, ProteinG: it.ProteinG, CarbsG: it.CarbsG, FatG: it.FatG},
		Nutrients{Kcal: it.AIEstKcal, ProteinG: it.AIEstProteinG, CarbsG: it.AIEstCarbsG, FatG: it.AIEstFatG})
}
