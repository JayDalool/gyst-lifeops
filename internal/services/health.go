package services

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

// HealthValidationError separates user input errors from storage failures.
type HealthValidationError struct{ Message string }

func (e *HealthValidationError) Error() string { return e.Message }
func healthInvalid(message string) error       { return &HealthValidationError{Message: message} }
func validHealthNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1e9
}
func ValidateHealthDate(date string) error {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil || parsed.Format("2006-01-02") != date || parsed.Year() < 1 {
		return healthInvalid("date must be YYYY-MM-DD")
	}
	return nil
}
func validateHealthSource(source string) error {
	if source != "apple_shortcuts" {
		return healthInvalid("unsupported health source")
	}
	return nil
}
func ValidateHealthCycle(c models.HealthCycle) error {
	if strings.TrimSpace(c.Title) == "" || len(c.Title) > 200 {
		return healthInvalid("title is required and must be at most 200 characters")
	}
	if strings.TrimSpace(c.PrimaryGoal) == "" || len(c.PrimaryGoal) > 2000 || len(c.Notes) > 10000 {
		return healthInvalid("primary goal is required (maximum 2000 characters); notes must be at most 10000 characters")
	}
	if err := ValidateHealthDate(c.StartDate); err != nil {
		return err
	}
	if err := ValidateHealthDate(c.EndDate); err != nil {
		return err
	}
	if c.EndDate <= c.StartDate {
		return healthInvalid("end date must be after start date")
	}
	for _, value := range []float64{c.DailyCalorieTarget, c.ProteinTargetG, c.CarbsTargetG, c.FatTargetG, c.FiberTargetG, c.TargetDeficitKcal, float64(c.StrengthSessionsPerWeek), float64(c.CardioSessionsPerWeek), float64(c.StepTarget)} {
		if !validHealthNumber(value) {
			return healthInvalid("targets must be finite, non-negative, and no greater than 1 billion")
		}
	}
	return nil
}
func (s *Service) validateHealthCycleOverlap(c models.HealthCycle) error {
	cycles, err := s.Repo.ListHealthCycles()
	if err != nil {
		return err
	}
	for _, other := range cycles {
		if other.ID != c.ID && c.StartDate <= other.EndDate && c.EndDate >= other.StartDate {
			return healthInvalid("health cycle overlaps an existing cycle (dates include both start and end)")
		}
	}
	return nil
}
func healthCycleWriteError(err error) error {
	if errors.Is(err, repository.ErrHealthCycleOverlap) {
		return healthInvalid("health cycle overlaps an existing cycle")
	}
	return err
}
func (s *Service) CreateHealthCycle(c *models.HealthCycle) error {
	c.ID = 0
	c.Title = strings.TrimSpace(c.Title)
	c.PrimaryGoal = strings.TrimSpace(c.PrimaryGoal)
	if err := ValidateHealthCycle(*c); err != nil {
		return err
	}
	if err := s.validateHealthCycleOverlap(*c); err != nil {
		return err
	}
	return healthCycleWriteError(s.Repo.CreateHealthCycle(c))
}
func (s *Service) UpdateHealthCycle(c *models.HealthCycle) error {
	if c.ID <= 0 {
		return healthInvalid("invalid health cycle ID")
	}
	c.Title = strings.TrimSpace(c.Title)
	c.PrimaryGoal = strings.TrimSpace(c.PrimaryGoal)
	if err := ValidateHealthCycle(*c); err != nil {
		return err
	}
	if _, err := s.Repo.GetHealthCycle(c.ID); err != nil {
		return err
	}
	if err := s.validateHealthCycleOverlap(*c); err != nil {
		return err
	}
	return healthCycleWriteError(s.Repo.UpdateHealthCycle(c))
}
func (s *Service) GetHealthCycle(id int64) (models.HealthCycle, error) {
	return s.Repo.GetHealthCycle(id)
}
func (s *Service) ListHealthCycles() ([]models.HealthCycle, error) { return s.Repo.ListHealthCycles() }
func (s *Service) GetHealthCycleForDate(date string) (models.HealthCycle, error) {
	if err := ValidateHealthDate(date); err != nil {
		return models.HealthCycle{}, err
	}
	return s.Repo.GetHealthCycleForDate(date)
}
func ValidateHealthSync(input models.HealthSyncRequest) error {
	if err := ValidateHealthDate(input.Date); err != nil {
		return err
	}
	if err := validateHealthSource(input.Source); err != nil {
		return err
	}
	received := false
	for _, value := range []*float64{input.DietaryCalories, input.ProteinG, input.CarbsG, input.FatG, input.FiberG, input.ActiveEnergyKcal, input.RestingEnergyKcal, input.ExerciseMinutes} {
		if value != nil {
			received = true
			if !validHealthNumber(*value) {
				return healthInvalid("metrics must be finite, non-negative, and no greater than 1 billion")
			}
		}
	}
	for _, value := range []*int64{input.StepsWatch, input.StepsPhone} {
		if value != nil {
			received = true
			if *value < 0 || *value > 1e9 {
				return healthInvalid("steps must be between 0 and 1 billion")
			}
		}
	}
	if !received {
		return healthInvalid("at least one numeric metric is required")
	}
	return nil
}
func (s *Service) SyncHealth(input models.HealthSyncRequest) (models.DailyHealthSnapshot, error) {
	if err := ValidateHealthSync(input); err != nil {
		return models.DailyHealthSnapshot{}, err
	}
	snapshot := models.DailyHealthSnapshot{Date: input.Date, HealthMetrics: input.HealthMetrics, LastSyncedAt: s.Now().UTC().Format(time.RFC3339Nano)}
	return s.Repo.ApplyHealthSync(snapshot, input.Source)
}
func (s *Service) GetDailyHealthSnapshot(date string) (models.DailyHealthSnapshot, error) {
	if err := ValidateHealthDate(date); err != nil {
		return models.DailyHealthSnapshot{}, err
	}
	return s.Repo.GetDailyHealthSnapshot(date)
}
func (s *Service) GetHealthSyncState(source string) (models.HealthSyncState, error) {
	if err := validateHealthSource(source); err != nil {
		return models.HealthSyncState{}, err
	}
	return s.Repo.GetHealthSyncState(source)
}
func (s *Service) CreateHealthWeightEntry(w *models.HealthWeightEntry) error {
	if !validHealthNumber(w.WeightKg) || w.WeightKg == 0 {
		return healthInvalid("weight must be positive kilograms, no greater than 1 billion")
	}
	if w.Source != "manual" {
		if err := validateHealthSource(w.Source); err != nil {
			return err
		}
	}
	measured, err := time.Parse(time.RFC3339Nano, w.MeasuredAt)
	if err != nil {
		return healthInvalid("measured_at must be an RFC3339 timestamp with timezone")
	}
	// Fixed precision makes TEXT sorting chronological even with fractional seconds.
	w.MeasuredAt = measured.UTC().Format("2006-01-02T15:04:05.000000000Z")
	return s.Repo.CreateHealthWeightEntry(w)
}
func (s *Service) ListHealthWeightEntries() ([]models.HealthWeightEntry, error) {
	return s.Repo.ListHealthWeightEntries()
}

func CalculateObservedHealthEnergy(snapshot *models.DailyHealthSnapshot, cycle *models.HealthCycle) models.ObservedHealthEnergy {
	var out models.ObservedHealthEnergy
	if snapshot == nil {
		return out
	}
	if snapshot.ActiveEnergyKcal != nil && snapshot.RestingEnergyKcal != nil {
		burn := *snapshot.ActiveEnergyKcal + *snapshot.RestingEnergyKcal
		out.ObservedTotalBurn = &burn
		if snapshot.DietaryCalories != nil {
			balance := *snapshot.DietaryCalories - burn
			out.ObservedEnergyBalance = &balance
		}
	}
	if cycle != nil && snapshot.DietaryCalories != nil {
		remaining := cycle.DailyCalorieTarget - *snapshot.DietaryCalories
		out.FoodRemaining = &remaining
	}
	return out
}

// Use the existing configured local timezone; incoming dates remain literal dates.
func (s *Service) HealthTodayDate() (string, error) {
	config, err := s.PrayerConfig()
	if err != nil {
		return "", err
	}
	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		return "", fmt.Errorf("health timezone: %w", err)
	}
	return s.Now().In(location).Format("2006-01-02"), nil
}
func (s *Service) HealthToday() (models.HealthTodayResponse, error) {
	date, err := s.HealthTodayDate()
	if err != nil {
		return models.HealthTodayResponse{}, err
	}
	out := models.HealthTodayResponse{Date: date}
	snapshot, err := s.GetDailyHealthSnapshot(date)
	if err == nil {
		out.Snapshot = &snapshot
	} else if !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	cycle, err := s.GetHealthCycleForDate(date)
	if err == nil {
		out.HealthCycle = &cycle
	} else if !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	out.Observed = CalculateObservedHealthEnergy(out.Snapshot, out.HealthCycle)
	return out, nil
}
