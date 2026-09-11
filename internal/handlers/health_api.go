package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

type healthAPIError struct {
	Error string `json:"error"`
}

func healthJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func healthJSONError(w http.ResponseWriter, status int, message string) {
	healthJSON(w, status, healthAPIError{Error: message})
}

// healthAPI is deliberately write-only: POST /api/health/sync is the entire
// surface. That makes GYST_HEALTH_SYNC_TOKEN a write credential by
// construction rather than by convention, so the Apple Health Shortcut - which
// lives in iCloud and is shareable from the share sheet with one tap - cannot
// be used to read back a health history. A read API, when one is wanted, should
// sit behind the session added in A4 rather than behind this token.
func (h *Handler) healthAPI(w http.ResponseWriter, r *http.Request, segments []string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Authorization")
	if strings.TrimSpace(h.HealthSyncToken) == "" {
		healthJSONError(w, 503, "health API is not configured")
		return
	}
	auth := strings.Fields(r.Header.Get("Authorization"))
	if len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") {
		w.Header().Set("WWW-Authenticate", "Bearer")
		healthJSONError(w, 401, "unauthorized")
		return
	}
	expected, supplied := sha256.Sum256([]byte(h.HealthSyncToken)), sha256.Sum256([]byte(auth[1]))
	if subtle.ConstantTimeCompare(expected[:], supplied[:]) != 1 {
		w.Header().Set("WWW-Authenticate", "Bearer")
		healthJSONError(w, 401, "unauthorized")
		return
	}
	if len(segments) != 3 {
		healthJSONError(w, 404, "health endpoint not found")
		return
	}
	switch segments[2] {
	case "sync":
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			healthJSONError(w, 405, "method not allowed")
			return
		}
		media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "application/json" {
			healthJSONError(w, 415, "Content-Type must be application/json")
			return
		}
		var input models.HealthSyncRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&input)
		if err == nil {
			var extra any
			err = decoder.Decode(&extra)
			if err == io.EOF {
				err = nil
			} else if err == nil {
				err = errors.New("expected a single JSON object")
			}
		}
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				healthJSONError(w, 413, "request body too large")
			} else {
				healthJSONError(w, 400, "invalid health JSON")
			}
			return
		}
		snapshot, err := h.Service.SyncHealth(input)
		if err != nil {
			h.healthServiceError(w, err)
			return
		}
		healthJSON(w, 200, snapshot)
	default:
		healthJSONError(w, 404, "health endpoint not found")
	}
}
func (h *Handler) healthServiceError(w http.ResponseWriter, err error) {
	var invalid *services.HealthValidationError
	if errors.As(err, &invalid) {
		healthJSONError(w, 400, invalid.Error())
		return
	}
	healthJSONError(w, 500, "health data unavailable")
}
