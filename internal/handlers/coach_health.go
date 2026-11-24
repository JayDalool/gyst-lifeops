package handlers

import (
	"sort"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/secretary"
	"lifeops-tracker/internal/services"
)

// coachHealth gives internal/secretary the Health data it needs (Issue #236).
// The tool bodies are the ones that lived in coachTools; they moved here
// unchanged because the view types they read (HealthData, healthWeightView,
// coachReferenceTargets) belong to this package and secretary cannot import it.
// now and loc are captured once per request, as coachTools always did.
type coachHealth struct {
	h    *Handler
	data HealthData
	now  time.Time
	loc  *time.Location
}

var _ secretary.HealthReader = coachHealth{}

func (c coachHealth) training() []models.TrainingEntry {
	cache, err := c.h.Service.Repo.TrainingCache("hevy")
	if err != nil {
		return nil
	}
	return cache.Entries
}

func (c coachHealth) Today() (any, error) { return coachToday(c.data), nil }

func (c coachHealth) Plan() (any, error) {
	h, data := c.h, c.data
	cy, err := h.Service.GetHealthCycleForDate(data.Today.Date)
	var cycleID *int64
	if err == nil {
		cycleID = &cy.ID
	}
	refs, rerr := h.coachReferenceTargets(cycleID)
	if rerr != nil {
		return nil, rerr
	}
	if err != nil {
		return map[string]any{"plan": nil, "reference_targets": refs}, nil
	}
	return map[string]any{"reference_targets": refs, "title": cy.Title, "goal": cy.PrimaryGoal, "start": cy.StartDate, "end": cy.EndDate, "calories": cy.DailyCalorieTarget, "protein_g": cy.ProteinTargetG,
		"carbs_g": cy.CarbsTargetG, "fat_g": cy.FatTargetG, "fiber_g": cy.FiberTargetG, "target_daily_deficit_kcal": cy.TargetDeficitKcal,
		"strength_sessions_per_week": cy.StrengthSessionsPerWeek, "cardio_sessions_per_week": cy.CardioSessionsPerWeek, "step_target": cy.StepTarget}, nil
}

func (c coachHealth) WeightTrend() (any, error) {
	data, now, loc := c.data, c.now, c.loc
	entries, err := c.h.Service.ListHealthWeightEntries()
	if err != nil {
		return nil, err
	}
	readings := []map[string]any{}
	cut := now.AddDate(0, 0, -90)
	for _, e := range entries {
		at, err := time.Parse(time.RFC3339Nano, e.MeasuredAt)
		if err == nil && at.After(cut) && !at.After(now) {
			readings = append(readings, map[string]any{"date": at.In(loc).Format("2006-01-02"), "kg": e.WeightKg})
		}
	}
	if len(readings) > 30 {
		readings = readings[:30]
	}
	avg := data.Weight.Average
	out := map[string]any{"latest_kg": metricOrNilString(data.Weight.Present, data.Weight.Latest), "change_since": data.Weight.Since, "change": data.Weight.Change,
		"change_is_between_7_day_averages": data.Weight.ChangeIsAverage, "readings_newest_first": readings,
		"seven_day_average_kg": metricOrNilString(avg.Present, avg.Avg), "seven_day_average_days_used": nil, "seven_day_average_note": avg.Line,
		"previous_seven_day_average_kg": metricOrNilString(avg.HasPrev, avg.Prev), "change_vs_previous_seven_days": metricOrNilString(avg.HasPrev, avg.Change)}
	if avg.Present {
		out["seven_day_average_days_used"] = avg.Days
		if avg.MixedScales {
			out["readings_note"] = avg.Note
		}
	}
	return out, nil
}

func (c coachHealth) TrainingSummary() (any, error) {
	now, loc := c.now, c.loc
	e := c.training()
	week := services.SummarizeTrainingWeek(e, now, loc)
	target := 0
	if cy, err := c.h.Service.GetHealthCycleForDate(c.data.Today.Date); err == nil {
		target = int(cy.StrengthSessionsPerWeek)
	}
	groups := []map[string]any{}
	for _, g := range services.MuscleGroupShares(e, now, 30) {
		groups = append(groups, map[string]any{"group": g.Name, "percent": g.Percent, "sets": g.Sets})
	}
	return map[string]any{"this_week_workouts": week.Workouts, "weekly_target": target, "volume_kg": week.VolumeKG, "training_minutes": week.Minutes,
		"workouts_without_end_time": week.UnknownTime, "workouts_per_week_oldest_first": services.WorkoutsPerWeek(e, now, loc, 8), "muscle_groups_30_days": groups}, nil
}

func (c coachHealth) RecentWorkouts(count int) (any, error) {
	e := c.training()
	sort.SliceStable(e, func(i, j int) bool { return e[i].StartedAt.After(e[j].StartedAt) })
	out := []map[string]any{}
	for _, w := range e {
		if w.Kind != "workout" || len(out) == count {
			continue
		}
		exercises := []map[string]any{}
		for _, ex := range w.Exercises {
			item := map[string]any{"name": ex.Name, "muscle": services.MuscleName(ex.PrimaryMuscle), "best_set": bestSetText(ex.Sets), "work_sets": workSetCount(ex.Sets)}
			addCardioFields(item, ex.Sets)
			exercises = append(exercises, item)
		}
		item := map[string]any{"title": w.Title, "date": w.StartedAt.In(c.loc).Format("2006-01-02"), "exercises": exercises}
		if !w.EndedAt.IsZero() {
			item["minutes"] = int(w.EndedAt.Sub(w.StartedAt).Minutes())
		}
		out = append(out, item)
	}
	return map[string]any{"workouts": out}, nil
}

func (c coachHealth) ExerciseProgress(weeks int) (any, error) {
	out := []map[string]any{}
	for _, t := range services.ExerciseProgress(c.training(), c.now, weeks*7) {
		out = append(out, map[string]any{"exercise": t.Name, "change_percent": t.ChangePct, "sessions": t.Sessions, "best_set": t.BestSet})
	}
	return map[string]any{"weeks": weeks, "exercises": out, "note": "Only exercises logged in two or more sessions in the window appear."}, nil
}

func (c coachHealth) HevyRoutines() (any, error) {
	out := []map[string]any{}
	for _, r := range c.training() {
		if r.Kind != "routine" {
			continue
		}
		names := []string{}
		for _, ex := range r.Exercises {
			names = append(names, ex.Name)
		}
		out = append(out, map[string]any{"title": r.Title, "exercises": names})
	}
	return map[string]any{"routines": out}, nil
}

func (c coachHealth) FoodProducts() (any, error) {
	foods, err := c.h.Service.Repo.ListHealthFoods(false)
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, f := range foods {
		out = append(out, map[string]any{"name": f.Name, "serving": f.Serving, "kcal": f.Kcal, "protein_g": f.ProteinG, "carbs_g": f.CarbsG, "fat_g": f.FatG, "fiber_g": f.FiberG, "notes": f.Notes})
	}
	return map[string]any{"foods": out}, nil
}

func (c coachHealth) DailyChecklist() (any, error) {
	out := []map[string]any{}
	estimated := false
	for _, r := range c.data.Checklist {
		item := map[string]any{"name": r.Name, "done": r.Done}
		if r.Measured {
			item["measured"] = r.Value
			estimated = estimated || r.Estimate != ""
		}
		out = append(out, item)
	}
	res := map[string]any{"items": out}
	// The measured protein row includes AI estimates (GYST-180): the same
	// aggregate as get_health_today, counts and sums only.
	if estimated {
		res["ai_estimated"] = c.data.Today.Estimated.aiEstimated(aiChecklistCoachNote)
	}
	return res, nil
}

// BriefHealth is today's Health for the Secretary's day brief. It reads the
// same view fields as coachToday, so the brief and the Coach cannot disagree,
// and it follows RULES 3.3-3.5: not-observed stays nil (never 0), steps are the
// one count the Today view shows with its source (Watch and phone are never
// added), nothing is derived from steps, and the food budget and the observed
// energy balance are separate fields.
func (c coachHealth) BriefHealth() (secretary.HealthBrief, error) {
	t := c.data.Today
	str := func(m HealthMetricView) *string {
		if !m.Present {
			return nil
		}
		v := m.Value
		return &v
	}
	out := secretary.HealthBrief{
		AsOf:                   t.LastSyncedAt, // same parse as Stale: one read (GYST-164)
		Stale:                  t.LastSyncedStale,
		FoodEatenKcal:          str(t.Calories.Consumed),
		FoodTargetKcal:         str(t.CalorieTarget),
		BurnedSoFarKcal:        str(t.ObservedBurn),
		EnergyBalanceSoFarKcal: str(t.ObservedEnergy),
	}
	if t.Calories.HasBar {
		budget := t.BudgetAbs
		if t.CaloriesOver {
			out.FoodOverBudgetByKcal = &budget
		} else {
			out.FoodBudgetLeftKcal = &budget
		}
	}
	if t.Estimated.Present {
		items := t.Estimated.Items
		out.FoodAIEstimatedItems = &items
		out.FoodAIEstimatedKcal, out.FoodAIEstimatedProteinG = str(t.Estimated.Kcal), str(t.Estimated.ProteinG)
	}
	for _, m := range t.Nutrients {
		if m.Short() == "Protein" {
			out.ProteinEatenG, out.ProteinTargetG = str(m.Consumed), str(m.Target)
		}
	}
	if out.Steps = str(t.Steps); out.Steps != nil {
		out.StepsSource = map[string]string{"Apple Watch": "watch", "iPhone": "phone"}[t.StepsSource]
	}
	if w := c.data.Water; w.Logged {
		total := w.Total
		out.WaterLoggedL = &total
		if w.HasTarget {
			target := w.Target
			out.WaterTargetL = &target
		}
	} else if w.HasTarget {
		target := w.Target
		out.WaterTargetL = &target
	}
	if c.data.Weight.Present {
		latest := c.data.Weight.Latest
		out.WeightKg = &latest
	}
	return out, nil
}
