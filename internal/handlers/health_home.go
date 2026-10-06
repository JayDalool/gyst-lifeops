package handlers

import (
	"fmt"
	"math"
	"time"

	"lifeops-tracker/internal/models"
)

// Health Today redesign (#70). Everything here is presentation: it reads the
// same observations and targets as before and invents no new numbers.

// healthCalorieState is the calorie card's status. Colour is never the only
// signal: every state has its own words and icon.
type healthCalorieState struct {
	Key   string // ok | watch | over | idle
	Label string
	Icon  string
}

// calorieWatchShare is when "within budget" becomes "close to budget": the
// last tenth of the day's target.
const calorieWatchShare = 0.10

// buildCalorieState compares eaten with the target at displayed precision,
// the same numbers the ring shows.
func buildCalorieState(v HealthTodayView, target float64) healthCalorieState {
	switch {
	case !v.HasSnapshot:
		return healthCalorieState{"idle", "Waiting for sync", "refresh-cw"}
	case !v.Calories.Consumed.Present:
		return healthCalorieState{"idle", "No food logged", "info"}
	case !v.Calories.HasBar:
		return healthCalorieState{"idle", "No target set", "info"}
	case v.CaloriesOver:
		return healthCalorieState{"over", "Over budget", "circle-alert"}
	case v.caloriesLeft <= math.Round(target)*calorieWatchShare:
		return healthCalorieState{"watch", "Close to budget", "info"}
	default:
		return healthCalorieState{"ok", "Within budget", "check"}
	}
}

// HealthNudge is the optional movement suggestion under the calorie card.
// It is decided here, from observed data only, and never mentions calories:
// GYST has no active-energy goal, so "burn 400 kcal" would be an invented
// target, and food is never something to work off.
type HealthNudge struct {
	Kind  string // "move" or "recovery"
	Title string
	Body  string
}

type nudgeInput struct {
	Hour        int // local hour, 0-23
	HasSnapshot bool
	Stale       bool
	Steps       *int64 // the step count shown: Watch, else iPhone; never a sum
	StepTarget  int64
	Exercise    *float64 // Apple Health exercise minutes
	Workout     string   // title of today's Hevy workout, if any
}

// Nudge rules (#70). In order:
//   - no trustworthy data (no snapshot today, or a stale one): nothing
//   - before 11:00 or from 21:00: nothing. Mornings are too early to judge
//     and late evenings are for rest.
//   - a Hevy workout today: a short recovery note, never a push for more
//   - 30+ exercise minutes, or the step target reached: nothing
//   - steps under the target: how many steps are left, plus easy options
//   - no step target, under 30 exercise minutes: easy options only
const (
	nudgeFromHour       = 11
	nudgeUntilHour      = 21
	nudgeActiveMinutes  = 30
	nudgeShortWalkSteps = 2000
)

func buildHealthNudge(in nudgeInput) *HealthNudge {
	if !in.HasSnapshot || in.Stale || in.Hour < nudgeFromHour || in.Hour >= nudgeUntilHour {
		return nil
	}
	if in.Workout != "" {
		return &HealthNudge{Kind: "recovery", Title: "Nice work on " + in.Workout,
			Body: "Strength work done. An easy walk later helps recovery; no extra workout needed."}
	}
	if in.Exercise != nil && *in.Exercise >= nudgeActiveMinutes {
		return nil
	}
	if in.Steps != nil && in.StepTarget > 0 {
		left := in.StepTarget - *in.Steps
		if left <= 0 {
			return nil
		}
		body := "A 30–45 min walk or an easy swim would round out your day."
		if left <= nudgeShortWalkSteps {
			body = "A short walk would get you there."
		}
		return &HealthNudge{Kind: "move", Title: "A little movement would help",
			Body: fmt.Sprintf("%s steps to your %s goal. %s", formatThousands(float64(left)), formatThousands(float64(in.StepTarget)), body)}
	}
	if in.Exercise != nil {
		return &HealthNudge{Kind: "move", Title: "A little movement would help",
			Body: "A 30–45 min walk or an easy swim would round out your day."}
	}
	return nil
}

// nudgeUnlessErr drops the nudge when its inputs couldn't all be read.
func nudgeUnlessErr(n *HealthNudge, err error) *HealthNudge {
	if err != nil {
		return nil
	}
	return n
}

// todayWorkoutTitle is the most recent Hevy workout that started on the
// given local day, or "".
func todayWorkoutTitle(entries []models.TrainingEntry, date string, loc *time.Location) string {
	title, latest := "", time.Time{}
	for _, e := range entries {
		if e.Kind != "workout" || e.StartedAt.In(loc).Format("2006-01-02") != date || !e.StartedAt.After(latest) {
			continue
		}
		title, latest = e.Title, e.StartedAt
		if title == "" {
			title = "today's workout"
		}
	}
	return title
}

// checklistDone counts the rows that are done today, hand-ticked or measured.
func checklistDone(rows []healthChecklistRow) int {
	n := 0
	for _, r := range rows {
		if r.Done {
			n++
		}
	}
	return n
}

// ChecklistPercent is the summary bar width for "n of m complete".
func (d HealthData) ChecklistPercent() int {
	if len(d.Checklist) == 0 {
		return 0
	}
	return int(math.Round(float64(d.ChecklistDone) / float64(len(d.Checklist)) * 100))
}

// BarPercent is Percent capped for a progress bar's width.
func (m HealthMacroView) BarPercent() int {
	if m.Percent > 100 {
		return 100
	}
	return m.Percent
}

// Key is the macro's CSS category ("protein", "carbs", "fat", "fiber").
func (m HealthMacroView) Key() string {
	switch m.Label {
	case "Carbohydrates":
		return "carbs"
	case "Protein":
		return "protein"
	case "Fat":
		return "fat"
	case "Fiber":
		return "fiber"
	}
	return "other"
}
