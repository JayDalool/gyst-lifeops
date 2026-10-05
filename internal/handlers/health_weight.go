package handlers

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// Manual entries outside this range are almost certainly typos (13.45, 1345).
const (
	minManualWeightKg = 20
	maxManualWeightKg = 400
)

// localDateTime is the value format of <input type="datetime-local">.
const localDateTime = "2006-01-02T15:04"

type loggedWeight struct {
	ID    int64
	Value string
}

// parseManualWeight validates the Log weight form. measured is interpreted in
// the application timezone; empty means now.
func parseManualWeight(rawWeight, rawMeasured string, now time.Time, location *time.Location) (models.HealthWeightEntry, error) {
	entry := models.HealthWeightEntry{Source: "manual"}
	text := strings.ReplaceAll(strings.TrimSpace(rawWeight), ",", ".")
	if text == "" {
		return entry, &services.HealthValidationError{Message: "Enter your weight in kilograms"}
	}
	kg, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(kg) || math.IsInf(kg, 0) {
		return entry, &services.HealthValidationError{Message: "Weight must be a number, for example 134.5"}
	}
	if kg <= minManualWeightKg || kg >= maxManualWeightKg {
		return entry, &services.HealthValidationError{Message: "Weight must be between 20 and 400 kg"}
	}
	entry.WeightKg = math.Round(kg*100) / 100
	measured := now
	if raw := strings.TrimSpace(rawMeasured); raw != "" {
		measured, err = time.ParseInLocation(localDateTime, raw, location)
		if err != nil {
			return entry, &services.HealthValidationError{Message: "Measured time must be a valid date and time"}
		}
		if measured.After(now.Add(5 * time.Minute)) {
			return entry, &services.HealthValidationError{Message: "Measured time can't be in the future"}
		}
	}
	entry.MeasuredAt = measured.Format(time.RFC3339Nano)
	return entry, nil
}

// healthWeight handles POST /health/weight from the Progress page form.
func (h *Handler) healthWeight(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		methodNotAllowed(w)
		return
	}
	if requireSameOrigin(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid weight form", http.StatusBadRequest)
		return
	}
	rawWeight, rawMeasured := r.PostFormValue("weight_kg"), r.PostFormValue("measured_at")
	entry, err := parseManualWeight(rawWeight, rawMeasured, h.Service.Now(), h.Service.Location())
	if err == nil {
		err = h.Service.CreateHealthWeightEntry(&entry)
	}
	var invalid *services.HealthValidationError
	switch {
	case err == nil:
		http.Redirect(w, r, "/health/progress?logged="+strconv.FormatInt(entry.ID, 10)+"#log", http.StatusSeeOther)
	case errors.As(err, &invalid):
		h.renderHealthProgress(w, "30", http.StatusBadRequest, func(data *healthProgressData) {
			data.Error, data.WeightValue, data.MeasuredValue = invalid.Error(), rawWeight, rawMeasured
		})
	default:
		serverError(w, err)
	}
}

// healthWeightUndo handles POST /health/weight/{id}/undo, offered only in the
// confirmation shown right after logging.
func (h *Handler) healthWeightUndo(w http.ResponseWriter, r *http.Request, rawID string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		methodNotAllowed(w)
		return
	}
	if requireSameOrigin(w, r) {
		return
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	if err := h.Service.UndoManualWeightEntry(id); errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/health/progress?undone=1#log", http.StatusSeeOther)
}
