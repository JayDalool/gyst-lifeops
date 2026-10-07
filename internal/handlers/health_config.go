package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// Daily checklist and My foods (Issue #59). Every write is an authenticated,
// same-origin form POST that redirects; nothing needs JavaScript.

// healthChecklistRow is one row of the Today checklist card. Habits are
// ticked by hand; the protein row is measured and can't be ticked.
type healthChecklistRow struct {
	ID       int64
	Name     string
	Habit    bool
	Done     bool
	Value    string // measured rows: "132 / 180 g", or "—" when not logged
	Measured bool
	Source   string // measured rows: what the value is measured from
}

func buildChecklistRows(items []models.HealthChecklistItem, today HealthTodayView) []healthChecklistRow {
	rows := []healthChecklistRow{}
	for _, it := range items {
		row := healthChecklistRow{ID: it.ID, Name: it.Name, Habit: it.Kind == "habit", Done: it.DoneToday}
		if it.Kind == "protein" {
			row.Measured, row.Source = true, "food logged"
			var protein HealthMacroView
			for _, m := range today.Macros {
				if m.Label == "Protein" {
					protein = m
				}
			}
			row.Done = protein.HasBar && protein.State == "ok"
			row.Value = "—"
			if protein.Consumed.Present {
				row.Value = protein.Consumed.Value
			}
			if protein.Target.Present {
				row.Value += " / " + protein.Target.Value + " g"
			} else {
				row.Value += " g"
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// postGuard handles the method, origin and body-size checks shared by the
// POST routes; it reports whether the request may proceed.
func postGuard(w http.ResponseWriter, r *http.Request, allow string) bool {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost && !(allow == "GET, POST" && r.Method == http.MethodGet) {
		w.Header().Set("Allow", allow)
		methodNotAllowed(w)
		return false
	}
	if r.Method == http.MethodPost {
		if requireSameOrigin(w, r) {
			return false
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return false
		}
	}
	return true
}

type healthChecklistData struct {
	Page       string
	Items      []models.HealthChecklistItem
	HasProtein bool
	WaterName  string // the measured water row's name; "" when not added (#60)
	Error      string
	Name, Kind string
}

func (healthChecklistData) HealthTabName() string { return "more" }

func (h *Handler) healthChecklist(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "GET, POST") {
		return
	}
	data := healthChecklistData{Page: "health", Kind: "habit"}
	status := http.StatusOK
	if r.Method == http.MethodPost && r.PostFormValue("kind") == "water" {
		// The measured water row (#60) is a setting, not a checklist item.
		name := strings.TrimSpace(r.PostFormValue("name"))
		if name == "" || utf8.RuneCountInString(name) > 80 {
			data.Error, data.Name, data.Kind, status = "Give the item a name (at most 80 characters)", r.PostFormValue("name"), "water", http.StatusBadRequest
		} else if err := h.Service.Repo.SetSetting(waterChecklistKey, name); err != nil {
			serverError(w, err)
			return
		} else {
			http.Redirect(w, r, "/health/checklist", http.StatusSeeOther)
			return
		}
	} else if r.Method == http.MethodPost {
		_, err := h.Service.AddHealthChecklistItem(r.PostFormValue("name"), r.PostFormValue("kind"))
		var invalid *services.HealthValidationError
		switch {
		case err == nil:
			http.Redirect(w, r, "/health/checklist", http.StatusSeeOther)
			return
		case errors.As(err, &invalid):
			data.Error, data.Name, data.Kind, status = invalid.Error(), r.PostFormValue("name"), r.PostFormValue("kind"), http.StatusBadRequest
		default:
			serverError(w, err)
			return
		}
	}
	items, err := h.Service.Repo.ListHealthChecklistItems(h.Service.TodayString())
	if err != nil {
		serverError(w, err)
		return
	}
	data.Items = items
	for _, it := range items {
		data.HasProtein = data.HasProtein || it.Kind == "protein"
	}
	if data.WaterName, err = h.Service.Repo.GetSetting(waterChecklistKey); err != nil {
		serverError(w, err)
		return
	}
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	h.render(w, "health_checklist.html", data)
}

func (h *Handler) healthChecklistItemAction(w http.ResponseWriter, r *http.Request, rawID, action string) {
	if !postGuard(w, r, "POST") {
		return
	}
	if rawID == "water" && action == "remove" {
		if err := h.Service.Repo.DeleteSetting(waterChecklistKey); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/health/checklist", http.StatusSeeOther)
		return
	}
	id, ok := parseID(rawID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var err error
	target := "/health/checklist"
	switch action {
	case "toggle":
		_, err = h.Service.Repo.ToggleHealthHabit(id, h.Service.TodayString())
		target = "/health#checklist"
	case "remove":
		err = h.Service.Repo.DeactivateHealthChecklistItem(id)
	default:
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

type healthFoodsData struct {
	Page   string
	Foods  []models.HealthFood
	Form   services.HealthFoodForm
	Error  string
	FoodID int64 // set on the edit page
}

func (healthFoodsData) HealthTabName() string { return "more" }

func foodFormFrom(r *http.Request) services.HealthFoodForm {
	return services.HealthFoodForm{Name: r.PostFormValue("name"), Serving: r.PostFormValue("serving"), Kcal: r.PostFormValue("kcal"), Protein: r.PostFormValue("protein"),
		Carbs: r.PostFormValue("carbs"), Fat: r.PostFormValue("fat"), Fiber: r.PostFormValue("fiber"), Notes: r.PostFormValue("notes"), Active: r.PostFormValue("active") != "0"}
}

// healthFoods lists foods and adds one.
func (h *Handler) healthFoods(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "GET, POST") {
		return
	}
	data := healthFoodsData{Page: "health", Form: services.HealthFoodForm{Active: true}}
	status := http.StatusOK
	if r.Method == http.MethodPost {
		data.Form = foodFormFrom(r)
		food, err := services.ParseHealthFood(data.Form)
		if err == nil {
			err = h.Service.Repo.CreateHealthFood(&food)
		}
		var invalid *services.HealthValidationError
		switch {
		case err == nil:
			http.Redirect(w, r, "/health/foods", http.StatusSeeOther)
			return
		case errors.As(err, &invalid):
			data.Error, status = invalid.Error(), http.StatusBadRequest
		default:
			serverError(w, err)
			return
		}
	}
	foods, err := h.Service.Repo.ListHealthFoods(true)
	if err != nil {
		serverError(w, err)
		return
	}
	data.Foods = foods
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	h.render(w, "health_foods.html", data)
}

// healthFood edits one food, including making it inactive.
func (h *Handler) healthFood(w http.ResponseWriter, r *http.Request, rawID string) {
	if !postGuard(w, r, "GET, POST") {
		return
	}
	id, ok := parseID(rawID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	existing, err := h.Service.Repo.GetHealthFood(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	data := healthFoodsData{Page: "health", FoodID: id, Form: services.FoodFormOf(existing)}
	if r.Method == http.MethodPost {
		data.Form = foodFormFrom(r)
		food, err := services.ParseHealthFood(data.Form)
		if err == nil {
			food.ID = id
			err = h.Service.Repo.UpdateHealthFood(food)
		}
		var invalid *services.HealthValidationError
		switch {
		case err == nil:
			http.Redirect(w, r, "/health/foods", http.StatusSeeOther)
			return
		case errors.As(err, &invalid):
			data.Error = invalid.Error()
			w.WriteHeader(http.StatusBadRequest)
		default:
			serverError(w, err)
			return
		}
	}
	h.render(w, "health_food.html", data)
}
