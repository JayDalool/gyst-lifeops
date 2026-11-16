package services

import (
	"errors"
	"math"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
)

// One counted intake source per day (Issue #207, Jay's decision 2026-11-16).
//
//   - A day with at least one live GYST diary item counts the GYST total. The
//     synced Apple Health total for that day is only a reference.
//   - Any other day counts the synced Apple Health dietary values, if any.
//   - Otherwise the day has no intake (nil values, never zero).
//
// The two sources are never added together. Every caller that needs a day's
// intake goes through ResolveDayIntake.

// snapshotNutrients are the synced dietary values of a snapshot.
func snapshotNutrients(snap *models.DailyHealthSnapshot) models.Nutrients {
	if snap == nil {
		return models.Nutrients{}
	}
	return models.Nutrients{Kcal: snap.DietaryCalories, ProteinG: snap.ProteinG, CarbsG: snap.CarbsG, FatG: snap.FatG, FiberG: snap.FiberG}
}

// ResolveDayIntake applies the source rule for one day. gyst is the day's live
// diary totals (nil or zero Items means no GYST day) and snap is the day's
// synced row (nil when nothing synced).
func ResolveDayIntake(date string, gyst *models.NutritionDayTotals, snap *models.DailyHealthSnapshot) models.DayIntake {
	out := models.DayIntake{Date: date, Source: models.IntakeSourceNone, Snapshot: snap}
	apple := snapshotNutrients(snap)
	if gyst != nil && gyst.Items > 0 {
		out.Source, out.Resolved = models.IntakeSourceGYST, true
		out.Counted, out.Missing, out.Items = gyst.Sum, gyst.Gaps, gyst.Items
		if apple.Any() {
			ref := apple
			out.Reference = &ref
		}
		return out
	}
	if apple.Any() {
		out.Source, out.Resolved, out.Counted = models.IntakeSourceAppleHealth, true, apple
	}
	return out
}

// DayIntake is the counted intake of one local day (YYYY-MM-DD).
func (s *Service) DayIntake(date string) (models.DayIntake, error) {
	if err := ValidateHealthDate(date); err != nil {
		return models.DayIntake{}, err
	}
	days, err := s.DayIntakeRange(date, date)
	if err != nil {
		return models.DayIntake{}, err
	}
	return days[0], nil
}

// DayIntakeRange returns one DayIntake per local day from..to inclusive,
// oldest first. Days with neither source have Source "none".
func (s *Service) DayIntakeRange(from, to string) ([]models.DayIntake, error) {
	if err := ValidateHealthDate(from); err != nil {
		return nil, err
	}
	if err := ValidateHealthDate(to); err != nil {
		return nil, err
	}
	start, _ := time.Parse(clock.DateLayout, from)
	end, _ := time.Parse(clock.DateLayout, to)
	if end.Before(start) {
		return nil, healthInvalid("end date is before start date")
	}
	if end.Sub(start) > 400*24*time.Hour {
		return nil, healthInvalid("date range is too long")
	}
	snaps, err := s.Repo.ListDailyHealthSnapshots(from, to)
	if err != nil {
		return nil, err
	}
	totals, err := s.Repo.NutritionDayTotals(from, to)
	if err != nil {
		return nil, err
	}
	byDate := map[string]*models.DailyHealthSnapshot{}
	for i := range snaps {
		byDate[snaps[i].Date] = &snaps[i]
	}
	var out []models.DayIntake
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		date := d.Format(clock.DateLayout)
		var gyst *models.NutritionDayTotals
		if t, ok := totals[date]; ok {
			gyst = &t
		}
		out = append(out, ResolveDayIntake(date, gyst, byDate[date]))
	}
	return out, nil
}

// ErrNutritionScale means an amount cannot be converted for a food (a unit
// that needs a serving weight the food does not have).
var ErrNutritionScale = errors.New("cannot scale this food for that unit")

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func scaleOne(v *float64, factor float64) *float64 {
	if v == nil {
		return nil
	}
	r := round2(*v * factor)
	return &r
}

// ScaleNutrients converts a logged amount of a food into snapshot nutrients and
// the weight in grams when known. Unit is "g" or "serving". A NULL nutrient
// stays NULL. A per_100g food needs serving_grams to log servings, and a
// per_serving food needs it to log grams.
func ScaleNutrients(food models.NutritionFood, unit string, amount float64) (models.Nutrients, *float64, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		return models.Nutrients{}, nil, ErrNutritionScale
	}
	servingGrams := 0.0
	if food.ServingGrams != nil && *food.ServingGrams > 0 {
		servingGrams = *food.ServingGrams
	}
	var factor float64
	var grams *float64
	switch {
	case food.Basis == "per_100g" && unit == "g":
		factor, grams = amount/100, &amount
	case food.Basis == "per_100g" && unit == "serving" && servingGrams > 0:
		g := amount * servingGrams
		factor, grams = g/100, &g
	case food.Basis == "per_serving" && unit == "serving":
		factor = amount
		if servingGrams > 0 {
			g := amount * servingGrams
			grams = &g
		}
	case food.Basis == "per_serving" && unit == "g" && servingGrams > 0:
		factor, grams = amount/servingGrams, &amount
	default:
		return models.Nutrients{}, nil, ErrNutritionScale
	}
	if grams != nil && unit != "g" { // a logged weight is stored exactly as typed
		g := round2(*grams)
		grams = &g
	}
	n := models.Nutrients{
		Kcal: scaleOne(food.Kcal, factor), ProteinG: scaleOne(food.ProteinG, factor),
		CarbsG: scaleOne(food.CarbsG, factor), FatG: scaleOne(food.FatG, factor), FiberG: scaleOne(food.FiberG, factor),
	}
	return n, grams, nil
}
