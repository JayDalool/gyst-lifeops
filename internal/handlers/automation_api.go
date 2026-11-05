package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"lifeops-tracker/internal/services"
)

// Environment variable names, used in startup logs. Values are never logged.
const (
	envN8NEventToken = "GYST_N8N_EVENT_TOKEN"
	envN8NWorkflows  = "GYST_N8N_WORKFLOWS"

	automationMinTokenBytes = 32
	automationRatePerMinute = 60
)

// automationLimiter is a fixed one-minute window. The zero value is ready.
type automationLimiter struct {
	mu    sync.Mutex
	start time.Time
	count int
}

func (l *automationLimiter) allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.start.IsZero() || now.Sub(l.start) >= time.Minute || now.Before(l.start) {
		l.start, l.count = now, 0
	}
	if l.count >= automationRatePerMinute {
		return false
	}
	l.count++
	return true
}

// AutomationConfigProblem names the environment variable that keeps the n8n
// event endpoint off, or returns "" when it is configured. The endpoint fails
// closed: an empty or short token, a token equal to the Health sync token, or
// an unusable workflow list all switch it off (503).
func (h *Handler) AutomationConfigProblem() string {
	token := h.N8NEventToken
	if len(token) < automationMinTokenBytes {
		return envN8NEventToken
	}
	if h.HealthSyncToken != "" {
		a, b := sha256.Sum256([]byte(token)), sha256.Sum256([]byte(h.HealthSyncToken))
		if subtle.ConstantTimeCompare(a[:], b[:]) == 1 {
			return envN8NEventToken
		}
	}
	if _, err := services.ParseWorkflows(h.N8NWorkflows); err != nil {
		return envN8NWorkflows
	}
	return ""
}

// LogAutomationConfig writes one startup line naming the variable that
// disables the endpoint, if any. It never includes a value.
func (h *Handler) LogAutomationConfig() {
	if name := h.AutomationConfigProblem(); name != "" {
		h.log().Warn("n8n event endpoint disabled", "event", "automation.disabled", "env", name)
		return
	}
	h.log().Info("n8n event endpoint enabled", "event", "automation.enabled")
}

// automationAPI serves POST /api/automation/events. It carries its own bearer
// token (GYST_N8N_EVENT_TOKEN), separate from the Health sync token, so the
// session middleware exempts exactly this path. It is write-only: it stores
// events and returns a fixed status, never stored data.
func (h *Handler) automationAPI(w http.ResponseWriter, r *http.Request, segments []string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Authorization")
	if h.AutomationConfigProblem() != "" {
		healthJSONError(w, 503, "automation events are not configured")
		return
	}
	auth := strings.Fields(r.Header.Get("Authorization"))
	if len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") {
		w.Header().Set("WWW-Authenticate", "Bearer")
		healthJSONError(w, 401, "unauthorized")
		return
	}
	expected, supplied := sha256.Sum256([]byte(h.N8NEventToken)), sha256.Sum256([]byte(auth[1]))
	if subtle.ConstantTimeCompare(expected[:], supplied[:]) != 1 {
		w.Header().Set("WWW-Authenticate", "Bearer")
		healthJSONError(w, 401, "unauthorized")
		return
	}
	if len(segments) != 3 || segments[2] != "events" {
		healthJSONError(w, 404, "automation endpoint not found")
		return
	}
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
	if !h.automationLimit.allow(h.Service.Now()) {
		w.Header().Set("Retry-After", "60")
		healthJSONError(w, 429, "too many events")
		return
	}
	var input services.AutomationEventInput
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
			healthJSONError(w, 400, "invalid event JSON")
		}
		return
	}
	workflows, _ := services.ParseWorkflows(h.N8NWorkflows) // validated above
	result, ignored, err := h.Service.RecordAutomationEvent(input, workflows)
	if err != nil {
		var invalid *services.AutomationValidationError
		if errors.As(err, &invalid) {
			healthJSONError(w, 400, invalid.Error())
			return
		}
		noteError(w, err)
		healthJSONError(w, 500, "automation events unavailable")
		return
	}
	if ignored {
		// One line, no body. workflow_id already passed the id check.
		h.log().LogAttrs(r.Context(), slog.LevelInfo, "automation event ignored",
			slog.String("event", "automation.ignored"),
			slog.String("workflow_id", input.WorkflowID),
			slog.String("request_id", requestIDFrom(r.Context())),
		)
		healthJSON(w, 202, map[string]string{"status": "ignored"})
		return
	}
	healthJSON(w, 200, map[string]string{"status": result})
}
