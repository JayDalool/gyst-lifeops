package services

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"lifeops-tracker/internal/models"
)

// Daily checklist and My foods (Issue #59). Validation mirrors the table
// constraints so users get a sentence instead of a constraint error.

func (s *Service) AddHealthChecklistItem(name, kind string) (models.HealthChecklistItem, error) {
	item := models.HealthChecklistItem{Name: strings.TrimSpace(name), Kind: kind}
	if item.Name == "" {
		return item, healthInvalid("Give the item a name")
	}
	if utf8.RuneCountInString(item.Name) > 80 {
		return item, healthInvalid("Item names can be at most 80 characters")
	}
	if kind != "habit" && kind != "protein" {
		return item, healthInvalid("Choose a habit or the protein target")
	}
	return item, s.Repo.CreateHealthChecklistItem(&item)
}

// HealthFoodForm is the food form as typed, kept for re-display on errors.
type HealthFoodForm struct {
	Name, Serving, Kcal, Protein, Carbs, Fat, Fiber, Notes string
	Active                                                 bool
}

func FoodFormOf(f models.HealthFood) HealthFoodForm {
	num := func(v *float64) string {
		if v == nil {
			return ""
		}
		return strconv.FormatFloat(*v, 'f', -1, 64)
	}
	return HealthFoodForm{Name: f.Name, Serving: f.Serving, Kcal: num(f.Kcal), Protein: num(f.ProteinG), Carbs: num(f.CarbsG), Fat: num(f.FatG), Fiber: num(f.FiberG), Notes: f.Notes, Active: f.Active}
}

// ParseHealthFood validates the form. Blank numbers stay unknown (nil).
func ParseHealthFood(form HealthFoodForm) (models.HealthFood, error) {
	food := models.HealthFood{Name: strings.TrimSpace(form.Name), Serving: strings.TrimSpace(form.Serving), Notes: strings.TrimSpace(form.Notes), Active: form.Active}
	switch {
	case food.Name == "":
		return food, healthInvalid("Give the food a name")
	case utf8.RuneCountInString(food.Name) > 120:
		return food, healthInvalid("Food names can be at most 120 characters")
	case utf8.RuneCountInString(food.Serving) > 80:
		return food, healthInvalid("Serving can be at most 80 characters")
	case utf8.RuneCountInString(food.Notes) > 500:
		return food, healthInvalid("Notes can be at most 500 characters")
	}
	fields := []struct {
		label, raw string
		max        float64
		dst        **float64
	}{
		{"Calories", form.Kcal, 10000, &food.Kcal}, {"Protein", form.Protein, 1000, &food.ProteinG},
		{"Carbs", form.Carbs, 1000, &food.CarbsG}, {"Fat", form.Fat, 1000, &food.FatG}, {"Fiber", form.Fiber, 1000, &food.FiberG},
	}
	for _, f := range fields {
		raw := strings.ReplaceAll(strings.TrimSpace(f.raw), ",", ".")
		if raw == "" {
			continue
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > f.max {
			return food, healthInvalid(f.label + " must be a number from 0 to " + strconv.FormatFloat(f.max, 'f', -1, 64))
		}
		v = math.Round(v*10) / 10
		*f.dst = &v
	}
	return food, nil
}
