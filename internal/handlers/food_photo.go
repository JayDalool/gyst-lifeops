package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/imageintake"
)

// Food photo analysis (Issue #217): POST /health/nutrition/photo/analyze.
// The photo is normalised in memory (imageintake), sent once to the configured
// vision provider and dropped. Nothing is stored; the response is the
// validated identification the review screen (#211) will show. No diary write.

const foodPhotoField = "photo"

func (h *Handler) healthFoodPhotoAnalyze(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		methodNotAllowed(w)
		return
	}
	if requireSameOrigin(w, r) {
		return
	}
	var analyzer healthcoach.FoodPhotoAnalyzer
	if h.Coach != nil {
		analyzer, _ = h.Coach.(healthcoach.FoodPhotoAnalyzer)
	}
	if analyzer == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Photo analysis isn't set up yet")
		return
	}
	// Rate limit before reading the body, so a rejected request costs nothing.
	if !h.photoLimit.acquire(h.Service.Now()) {
		writeJSONError(w, http.StatusTooManyRequests, healthcoach.ErrBusy.Error())
		return
	}
	defer h.photoLimit.release()

	jpeg, err := imageintake.ReadUpload(w, r, foodPhotoField)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, imageintake.ErrTooLarge):
			status = http.StatusRequestEntityTooLarge
		case errors.Is(err, imageintake.ErrUnsupported):
			status = http.StatusUnsupportedMediaType
		}
		// imageintake errors are fixed, user-safe sentences.
		h.logPhotoFailure(r, "intake")
		writeJSONError(w, status, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 75*time.Second)
	defer cancel()
	result, err := analyzer.AnalyzeFoodPhoto(ctx, jpeg)
	if err != nil {
		status, msg := http.StatusBadGateway, healthcoach.ErrUnavailable.Error()
		for _, known := range []error{healthcoach.ErrNotConfigured, healthcoach.ErrBusy, healthcoach.ErrDeclined, healthcoach.ErrPhotoUnreadable, healthcoach.ErrUnavailable} {
			if errors.Is(err, known) {
				msg = known.Error()
			}
		}
		if errors.Is(err, healthcoach.ErrBusy) {
			status = http.StatusTooManyRequests
		}
		if errors.Is(err, healthcoach.ErrNotConfigured) {
			status = http.StatusServiceUnavailable
		}
		// Only the fixed category is logged, never err, the image or output.
		h.logPhotoFailure(r, msg)
		writeJSONError(w, status, msg)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) logPhotoFailure(r *http.Request, category string) {
	h.log().LogAttrs(r.Context(), slog.LevelWarn, "photo analysis failure",
		slog.String("event", "photo.analyze.failed"),
		slog.String("category", category),
		slog.String("request_id", requestIDFrom(r.Context())))
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
