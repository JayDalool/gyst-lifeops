package handlers

import (
	"net/http"
	"strings"
)

type nutritionMetric struct {
	HealthMacroView
	Progress        UIProgress
	Difference      string
	DifferenceLabel string
}

type nutritionData struct {
	Page         string
	Today        HealthTodayView
	Metrics      []nutritionMetric
	HasNutrition bool
}

func buildNutritionData(today HealthTodayView) nutritionData {
	data := nutritionData{Page: "health", Today: today}
	for _, macro := range today.Macros {
		metric := nutritionMetric{HealthMacroView: macro, Progress: UIProgress{Present: macro.HasBar, Percent: macro.Percent, Label: macro.ProgressAria}, Difference: macro.Remaining.Value, DifferenceLabel: "remaining"}
		if strings.HasPrefix(metric.Difference, "−") {
			metric.Difference = strings.TrimPrefix(metric.Difference, "−")
			metric.DifferenceLabel = "over target"
		}
		data.HasNutrition = data.HasNutrition || macro.Consumed.Present
		data.Metrics = append(data.Metrics, metric)
	}
	return data
}

func (h *Handler) nutrition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	today, err := h.Service.HealthToday()
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "nutrition.html", buildNutritionData(buildHealthTodayView(today, h.Service.Now())))
}
