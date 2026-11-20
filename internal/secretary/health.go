package secretary

import (
	"context"
	"encoding/json"

	"lifeops-tracker/internal/healthcoach"
)

// HealthReader is where the Health tools get their data. The data code
// (views, targets, the Hevy cache) lives in internal/handlers and cannot be
// imported from here, so handlers implement this interface. Each method
// returns the exact value the Coach has always encoded for that tool.
type HealthReader interface {
	Today() (any, error)
	Plan() (any, error)
	WeightTrend() (any, error)
	TrainingSummary() (any, error)
	// RecentWorkouts gets a count already clamped to 1-5.
	RecentWorkouts(count int) (any, error)
	// ExerciseProgress gets weeks already reduced to 4, 12 or 26.
	ExerciseProgress(weeks int) (any, error)
	HevyRoutines() (any, error)
	FoodProducts() (any, error)
	DailyChecklist() (any, error)
	// BriefHealth is today's Health for the day brief. Missing values stay nil.
	BriefHealth() (HealthBrief, error)
}

// healthSpecs are the Coach's nine Health tools: the same names, descriptions
// and schemas as before they moved here (Issue #236, golden file in
// internal/handlers/testdata/coach_tools.golden.json). Here they have no
// Sources and no freshness: this is what the Coach takes. The Secretary's
// Catalogue wraps the four Hevy-backed ones (healthBuilder in catalogue.go).
func healthSpecs(r HealthReader) []ToolSpec {
	plain := func(f func() (any, error)) func(context.Context, json.RawMessage) (any, error) {
		return func(context.Context, json.RawMessage) (any, error) {
			v, err := f()
			if err != nil {
				return nil, errDataUnavailable
			}
			return v, nil
		}
	}
	return []ToolSpec{
		{Name: "get_health_today", Description: "Today's calories (eaten, target, budget left), burned so far, energy balance so far, macros, steps, exercise minutes, water logged vs target and latest weight.",
			Run: plain(r.Today)},
		{Name: "get_health_plan", Description: "The current health plan: dates and daily targets (calories, macros, steps, strength and cardio sessions per week), plus reference_targets: outside recommendations (for example a dietitian) labelled with their source, all not_active. Never present a reference target as the active plan.",
			Run: plain(r.Plan)},
		{Name: "get_weight_trend", Description: "Weight readings from the last 90 days (date, kg), latest reading, the 7-day average (null when fewer than 4 days have readings) and change since the plan started.",
			Run: plain(r.WeightTrend)},
		{Name: "get_training_summary", Description: "This week's workouts against the plan target, volume, training time, workouts per week for 8 weeks, and muscle-group shares over 30 days.",
			Run: plain(r.TrainingSummary)},
		{Name: "get_recent_workouts", Description: "The most recent Hevy workouts with each exercise's best set. Input: count (1-5).",
			Properties: map[string]any{"count": map[string]any{"type": "integer", "minimum": 1, "maximum": 5}},
			Run: func(_ context.Context, input json.RawMessage) (any, error) {
				var in struct {
					Count int `json:"count"`
				}
				// Bad JSON keeps the zero value, which clamps to 1, as before.
				_ = json.Unmarshal(input, &in)
				return plain(func() (any, error) { return r.RecentWorkouts(min(max(in.Count, 1), 5)) })(nil, nil)
			}},
		{Name: "get_exercise_progress", Description: "Change in best estimated 1-rep max per exercise (exercises with 2+ sessions). Input: weeks (4, 12 or 26).",
			Properties: map[string]any{"weeks": map[string]any{"type": "integer", "enum": []int{4, 12, 26}}},
			Run: func(_ context.Context, input json.RawMessage) (any, error) {
				var in struct {
					Weeks int `json:"weeks"`
				}
				_ = json.Unmarshal(input, &in)
				if in.Weeks != 4 && in.Weeks != 26 {
					in.Weeks = 12
				}
				return plain(func() (any, error) { return r.ExerciseProgress(in.Weeks) })(nil, nil)
			}},
		{Name: "get_hevy_routines", Description: "Jay's Hevy routines and their exercises.",
			Run: plain(r.HevyRoutines)},
		{Name: "get_food_products", Description: "Foods and products Jay usually has, with per-serving nutrition (null = unknown).",
			Run: plain(r.FoodProducts)},
		{Name: "get_daily_checklist", Description: "Today's checklist: Jay's own habits (done or not) and measured targets.",
			Run: plain(r.DailyChecklist)},
	}
}

// HealthTools are the Coach's nine Health tools, in the order it offers them.
// The Coach adds suggest_gyst_page itself and gets no other tool and no
// freshness sync.
func HealthTools(r HealthReader) []healthcoach.Tool {
	specs := healthSpecs(r)
	out := make([]healthcoach.Tool, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.tool(nil))
	}
	return out
}
