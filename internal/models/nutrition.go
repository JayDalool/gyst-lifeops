package models

// Food diary (Issue #207). All nutrient values are nullable: nil means
// "not recorded", never zero.

// Intake sources for one day. Exactly one is counted; they are never summed.
const (
	IntakeSourceGYST        = "gyst"
	IntakeSourceAppleHealth = "apple_health"
	IntakeSourceNone        = "none"
)

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
	Date  string       `json:"date"`
	Items int          `json:"items"`
	Sum   Nutrients    `json:"sum"`
	Gaps  NutrientGaps `json:"gaps"`
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
	CreatedVia string   `json:"created_via"`
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
