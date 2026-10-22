package services

import (
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
)

// GoalPace says whether an active goal is on track. Expected progress is
// target x elapsed days / total days on the application's calendar (today is
// the app day, YYYY-MM-DD). It returns ok=false, never a guess, when the goal
// is not active, has no target above 0, lacks either date, the dates are not
// a forward range, or today is before the start date.
func GoalPace(goal models.Goal, today string) (label string, ok bool) {
	if goal.Status != "active" || goal.TargetValue <= 0 || goal.StartDate == "" || goal.EndDate == "" {
		return "", false
	}
	start, err := time.Parse(clock.DateLayout, goal.StartDate)
	if err != nil {
		return "", false
	}
	end, err := time.Parse(clock.DateLayout, goal.EndDate)
	if err != nil {
		return "", false
	}
	now, err := time.Parse(clock.DateLayout, today)
	if err != nil {
		return "", false
	}
	total := end.Sub(start).Hours() / 24
	if total <= 0 || now.Before(start) {
		return "", false
	}
	elapsed := now.Sub(start).Hours() / 24
	if elapsed > total {
		elapsed = total
	}
	if goal.CurrentValue >= goal.TargetValue*elapsed/total {
		return "On track", true
	}
	return "Behind", true
}
