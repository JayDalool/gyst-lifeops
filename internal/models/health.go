package models

// HealthCycle dates are inclusive calendar dates; cycles may have any duration.
type HealthCycle struct {
	ID                      int64   `json:"id"`
	Title                   string  `json:"title"`
	StartDate               string  `json:"start_date"`
	EndDate                 string  `json:"end_date"`
	PrimaryGoal             string  `json:"primary_goal"`
	DailyCalorieTarget      float64 `json:"daily_calorie_target"`
	ProteinTargetG          float64 `json:"protein_target_g"`
	CarbsTargetG            float64 `json:"carbs_target_g"`
	FatTargetG              float64 `json:"fat_target_g"`
	FiberTargetG            float64 `json:"fiber_target_g"`
	TargetDeficitKcal       float64 `json:"target_deficit_kcal"`
	StrengthSessionsPerWeek int64   `json:"strength_sessions_per_week"`
	CardioSessionsPerWeek   int64   `json:"cardio_sessions_per_week"`
	StepTarget              int64   `json:"step_target"`
	// Water (#60), in millilitres; 0 = not set.
	WaterTargetML int64  `json:"water_target_ml"`
	WaterBottleML int64  `json:"water_bottle_ml"`
	Notes         string `json:"notes"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// HealthMetrics preserves absent values as nil. Watch and phone steps are independent.
type HealthMetrics struct {
	DietaryCalories   *float64 `json:"dietary_calories"`
	ProteinG          *float64 `json:"protein_g"`
	CarbsG            *float64 `json:"carbs_g"`
	FatG              *float64 `json:"fat_g"`
	FiberG            *float64 `json:"fiber_g"`
	StepsWatch        *int64   `json:"steps_watch"`
	StepsPhone        *int64   `json:"steps_phone"`
	ActiveEnergyKcal  *float64 `json:"active_energy_kcal"`
	RestingEnergyKcal *float64 `json:"resting_energy_kcal"`
	ExerciseMinutes   *float64 `json:"exercise_minutes"`
}

type DailyHealthSnapshot struct {
	Date string `json:"date"`
	HealthMetrics
	LastSyncedAt string `json:"last_synced_at"`
}

// HealthSyncRequest treats absent and JSON null fields as unreceived (no change).
// Explicit numeric zero is a received value. There is no metric-clearing operation.
type HealthSyncRequest struct {
	Date   string `json:"date"`
	Source string `json:"source"`
	HealthMetrics
}

type HealthWeightEntry struct {
	ID         int64   `json:"id"`
	MeasuredAt string  `json:"measured_at"`
	WeightKg   float64 `json:"weight_kg"`
	Source     string  `json:"source"`
	CreatedAt  string  `json:"created_at"`
}

type HealthSyncState struct {
	Source       string `json:"source"`
	LastSyncedAt string `json:"last_synced_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ObservedHealthEnergy is arithmetic on received observations, never a projection.
type ObservedHealthEnergy struct {
	ObservedTotalBurn     *float64 `json:"observed_total_burn"`
	ObservedEnergyBalance *float64 `json:"observed_energy_balance"`
	FoodRemaining         *float64 `json:"food_remaining"`
}

type HealthTodayResponse struct {
	Date        string               `json:"date"`
	Snapshot    *DailyHealthSnapshot `json:"snapshot"`
	HealthCycle *HealthCycle         `json:"health_cycle"`
	Observed    ObservedHealthEnergy `json:"observed"`
}
