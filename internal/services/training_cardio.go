package services

import (
	"time"

	"lifeops-tracker/internal/models"
)

// CardioWeek is one local week of duration-type cardio (for example a treadmill).
type CardioWeek struct {
	Start    time.Time
	Minutes  int
	Sessions int
	// Averages over the sessions that have the value; nil when none do.
	InclinePct, SpeedKmh *float64
}

// cardioSeconds returns the timed seconds of a cardio exercise: one logged as
// cardio, or with parsed incline or speed. Sets without a duration are skipped.
func cardioSeconds(ex models.TrainingExercise) (seconds float64, incline, speed []float64) {
	cardio := ex.PrimaryMuscle == "cardio"
	for _, s := range ex.Sets {
		if s.InclinePct != nil || s.SpeedKmh != nil {
			cardio = true
		}
	}
	if !cardio {
		return 0, nil, nil
	}
	for _, s := range ex.Sets {
		if s.Seconds == nil || *s.Seconds <= 0 || s.Kind == "warmup" {
			continue
		}
		seconds += *s.Seconds
		if s.InclinePct != nil {
			incline = append(incline, *s.InclinePct)
		}
		if s.SpeedKmh != nil {
			speed = append(speed, *s.SpeedKmh)
		}
	}
	return seconds, incline, speed
}

func mean(v []float64) *float64 {
	if len(v) == 0 {
		return nil
	}
	var sum float64
	for _, x := range v {
		sum += x
	}
	m := sum / float64(len(v))
	return &m
}

// CardioPerWeek returns the last n local weeks, oldest first. Incline and speed
// are the mean of each session's own mean, so a long session does not outweigh
// a short one.
func CardioPerWeek(entries []models.TrainingEntry, now time.Time, loc *time.Location, n int) []CardioWeek {
	out := make([]CardioWeek, n)
	current := weekStart(now, loc)
	for i := range out {
		from := current.AddDate(0, 0, -7*(n-1-i))
		week := CardioWeek{Start: from}
		var seconds float64
		var inclines, speeds []float64
		for _, e := range workoutsBetween(entries, from, from.AddDate(0, 0, 7)) {
			if e.StartedAt.After(now) {
				continue
			}
			var sessionSeconds float64
			var sessionIncline, sessionSpeed []float64
			for _, ex := range e.Exercises {
				s, inc, spd := cardioSeconds(ex)
				sessionSeconds += s
				sessionIncline = append(sessionIncline, inc...)
				sessionSpeed = append(sessionSpeed, spd...)
			}
			if sessionSeconds == 0 {
				continue
			}
			seconds += sessionSeconds
			week.Sessions++
			if m := mean(sessionIncline); m != nil {
				inclines = append(inclines, *m)
			}
			if m := mean(sessionSpeed); m != nil {
				speeds = append(speeds, *m)
			}
		}
		week.Minutes = int(seconds/60 + 0.5)
		week.InclinePct, week.SpeedKmh = mean(inclines), mean(speeds)
		out[i] = week
	}
	return out
}
