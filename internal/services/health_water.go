package services

import (
	"database/sql"
	"errors"
)

// Water bottle log (#60). A tap adds one bottle of the current plan's size
// to today (application timezone). Nothing here estimates water from food
// or activity: only taps count.

// maxWaterPerDayML stops a runaway double-tap loop; it is far above any
// sensible day.
const maxWaterPerDayML = 15000

// ErrWaterDailyCap is returned when a tap would pass maxWaterPerDayML.
var ErrWaterDailyCap = &HealthValidationError{Message: "That's more than 15 L today; check for a double tap"}

// LogHealthWaterBottle adds one bottle to today and returns the amount added.
func (s *Service) LogHealthWaterBottle() (int64, error) {
	today := s.TodayString()
	cycle, err := s.GetHealthCycleForDate(today)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && cycle.WaterBottleML <= 0) {
		return 0, healthInvalid("Set your bottle size in Health → Plan first")
	}
	if err != nil {
		return 0, err
	}
	stored, err := s.Repo.AddHealthWater(today, cycle.WaterBottleML, maxWaterPerDayML)
	if err != nil {
		return 0, err
	}
	if !stored {
		return 0, ErrWaterDailyCap
	}
	return cycle.WaterBottleML, nil
}

// UndoHealthWater removes today's latest tap and returns its amount;
// sql.ErrNoRows when nothing was logged today.
func (s *Service) UndoHealthWater() (int64, error) {
	return s.Repo.UndoHealthWater(s.TodayString())
}
