package handlers

import "net/http"

func (h *Handler) training(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	mission, err := h.Service.DailyMission()
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "training.html", HealthData{Page: "health", Mission: mission})
}
