package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// String fields preserve invalid submissions for correction in the form.
type healthPlanForm struct {
	ID                      string
	Title                   string
	StartDate               string
	EndDate                 string
	PrimaryGoal             string
	DailyCalorieTarget      string
	ProteinTargetG          string
	CarbsTargetG            string
	FatTargetG              string
	FiberTargetG            string
	TargetDeficitKcal       string
	StrengthSessionsPerWeek string
	CardioSessionsPerWeek   string
	StepTarget              string
	Notes                   string
}
type healthPlanData struct {
	Page   string
	Form   healthPlanForm
	Cycles []models.HealthCycle
	Error  string
	Saved  bool
}

func healthPlanValues(c models.HealthCycle) healthPlanForm {
	return healthPlanForm{
		ID:                      strconv.FormatInt(c.ID, 10),
		Title:                   c.Title,
		StartDate:               c.StartDate,
		EndDate:                 c.EndDate,
		PrimaryGoal:             c.PrimaryGoal,
		DailyCalorieTarget:      strconv.FormatFloat(c.DailyCalorieTarget, 'f', -1, 64),
		ProteinTargetG:          strconv.FormatFloat(c.ProteinTargetG, 'f', -1, 64),
		CarbsTargetG:            strconv.FormatFloat(c.CarbsTargetG, 'f', -1, 64),
		FatTargetG:              strconv.FormatFloat(c.FatTargetG, 'f', -1, 64),
		FiberTargetG:            strconv.FormatFloat(c.FiberTargetG, 'f', -1, 64),
		TargetDeficitKcal:       strconv.FormatFloat(c.TargetDeficitKcal, 'f', -1, 64),
		StrengthSessionsPerWeek: strconv.FormatInt(c.StrengthSessionsPerWeek, 10),
		CardioSessionsPerWeek:   strconv.FormatInt(c.CardioSessionsPerWeek, 10),
		StepTarget:              strconv.FormatInt(c.StepTarget, 10),
		Notes:                   c.Notes,
	}
}
func parseHealthPlan(r *http.Request) (healthPlanForm, models.HealthCycle, error) {
	form := healthPlanForm{ID: r.PostForm.Get("cycle_id"),
		Title:                   r.PostForm.Get("title"),
		StartDate:               r.PostForm.Get("start_date"),
		EndDate:                 r.PostForm.Get("end_date"),
		PrimaryGoal:             r.PostForm.Get("primary_goal"),
		DailyCalorieTarget:      r.PostForm.Get("daily_calorie_target"),
		ProteinTargetG:          r.PostForm.Get("protein_target_g"),
		CarbsTargetG:            r.PostForm.Get("carbs_target_g"),
		FatTargetG:              r.PostForm.Get("fat_target_g"),
		FiberTargetG:            r.PostForm.Get("fiber_target_g"),
		TargetDeficitKcal:       r.PostForm.Get("target_deficit_kcal"),
		StrengthSessionsPerWeek: r.PostForm.Get("strength_sessions_per_week"),
		CardioSessionsPerWeek:   r.PostForm.Get("cardio_sessions_per_week"),
		StepTarget:              r.PostForm.Get("step_target"),
		Notes:                   r.PostForm.Get("notes"),
	}
	var c models.HealthCycle
	var err error
	c.ID, err = strconv.ParseInt(form.ID, 10, 64)
	if err != nil || c.ID < 0 {
		return form, c, &services.HealthValidationError{Message: "invalid cycle ID"}
	}
	c.Title = form.Title
	c.StartDate = form.StartDate
	c.EndDate = form.EndDate
	c.PrimaryGoal = form.PrimaryGoal
	c.DailyCalorieTarget, err = strconv.ParseFloat(form.DailyCalorieTarget, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Daily calorie target must be a valid number"}
	}
	c.ProteinTargetG, err = strconv.ParseFloat(form.ProteinTargetG, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Protein target must be a valid number"}
	}
	c.CarbsTargetG, err = strconv.ParseFloat(form.CarbsTargetG, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Carbohydrate target must be a valid number"}
	}
	c.FatTargetG, err = strconv.ParseFloat(form.FatTargetG, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Fat target must be a valid number"}
	}
	c.FiberTargetG, err = strconv.ParseFloat(form.FiberTargetG, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Fiber target must be a valid number"}
	}
	c.TargetDeficitKcal, err = strconv.ParseFloat(form.TargetDeficitKcal, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Target deficit must be a valid number"}
	}
	c.StrengthSessionsPerWeek, err = strconv.ParseInt(form.StrengthSessionsPerWeek, 10, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Strength sessions per week must be a valid whole number"}
	}
	c.CardioSessionsPerWeek, err = strconv.ParseInt(form.CardioSessionsPerWeek, 10, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Cardio sessions per week must be a valid whole number"}
	}
	c.StepTarget, err = strconv.ParseInt(form.StepTarget, 10, 64)
	if err != nil {
		return form, c, &services.HealthValidationError{Message: "Step target must be a valid whole number"}
	}
	c.Notes = form.Notes
	return form, c, nil
}
func (h *Handler) healthPlan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		methodNotAllowed(w)
		return
	}
	data := healthPlanData{Page: "health", Saved: r.URL.Query().Get("saved") == "1"}
	if r.Method == http.MethodPost {
		// Kept alongside the middleware guard, not replaced by it: this handler
		// is also exercised directly, and the check is cheap.
		if requireSameOrigin(w, r) {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid plan form", 400)
			return
		}
		form, cycle, err := parseHealthPlan(r)
		data.Form = form
		if err == nil {
			if cycle.ID == 0 {
				err = h.Service.CreateHealthCycle(&cycle)
			} else {
				err = h.Service.UpdateHealthCycle(&cycle)
			}
		}
		if err == nil {
			http.Redirect(w, r, "/health/plan?id="+strconv.FormatInt(cycle.ID, 10)+"&saved=1", http.StatusSeeOther)
			return
		}
		var invalid *services.HealthValidationError
		if errors.As(err, &invalid) {
			data.Error = invalid.Error()
		} else if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		} else {
			serverError(w, err)
			return
		}
	} else {
		date, err := h.Service.HealthTodayDate()
		if err != nil {
			serverError(w, err)
			return
		}
		start, _ := time.Parse("2006-01-02", date)
		cycle := models.HealthCycle{StartDate: date, EndDate: start.AddDate(0, 0, 84).Format("2006-01-02")}
		if id := r.URL.Query().Get("id"); id != "" {
			number, err := strconv.ParseInt(id, 10, 64)
			if err != nil || number <= 0 {
				http.NotFound(w, r)
				return
			}
			cycle, err = h.Service.GetHealthCycle(number)
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				serverError(w, err)
				return
			}
		} else if r.URL.Query().Get("new") != "1" {
			current, err := h.Service.GetHealthCycleForDate(date)
			if err == nil {
				cycle = current
			} else if !errors.Is(err, sql.ErrNoRows) {
				serverError(w, err)
				return
			}
		}
		data.Form = healthPlanValues(cycle)
	}
	cycles, err := h.Service.ListHealthCycles()
	if err != nil {
		serverError(w, err)
		return
	}
	data.Cycles = cycles
	if strings.TrimSpace(data.Error) != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
	}
	h.render(w, "health_plan.html", data)
}
