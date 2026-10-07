package services

import (
	"database/sql"
	"errors"
)

// Water bottle log (#60, #78). A tap adds one bottle of the current plan's
// size to today (application timezone, so each day starts empty at local
// midnight). Taps stop at the plan's daily target. Nothing here estimates
// water from food or activity: only taps count.

// ErrWaterTargetReached is returned for a tap once today's total has
// reached the plan's water target (#78).
var ErrWaterTargetReached = &HealthValidationError{Message: "Today's water target is reached"}

// LogHealthWaterBottle adds one bottle to today and returns the amount added.
func (s *Service) LogHealthWaterBottle() (int64, error) {
	today := s.TodayString()
	cycle, err := s.GetHealthCycleForDate(today)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (cycle.WaterBottleML <= 0 || cycle.WaterTargetML <= 0)) {
		return 0, healthInvalid("Set your water target and bottle size in Health → Plan first")
	}
	if err != nil {
		return 0, err
	}
	stored, err := s.Repo.AddHealthWater(today, cycle.WaterBottleML, cycle.WaterTargetML)
	if err != nil {
		return 0, err
	}
	if !stored {
		return 0, ErrWaterTargetReached
	}
	return cycle.WaterBottleML, nil
}

// UndoHealthWater removes today's latest tap and returns its amount;
// sql.ErrNoRows when nothing was logged today.
func (s *Service) UndoHealthWater() (int64, error) {
	return s.Repo.UndoHealthWater(s.TodayString())
}
