package services

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"lifeops-tracker/internal/repository"
)

// n8n automation events (GYST-VN-110). The endpoint records what n8n reports
// about a fixed list of registered workflows; nothing here acts on it. State
// is derived on read from the stored events, by occurred_at and never by
// arrival order.

const (
	AutomationStatusStarted   = "started"
	AutomationStatusSucceeded = "succeeded"
	AutomationStatusFailed    = "failed"

	AutomationApplied   = "applied"
	AutomationDuplicate = "duplicate"

	AutomationStateFailed   = "Failed"
	AutomationStateRunning  = "Running"
	AutomationStateHealthy  = "Healthy"
	AutomationStateNoEvents = "No events"

	AutomationSeverityNone     = ""
	AutomationSeverityWarning  = "warning"
	AutomationSeverityCritical = "critical"

	automationWindow       = 5 * time.Minute
	automationRunningLimit = 2 * time.Hour
	automationRetention    = 30 * 24 * time.Hour
	automationMaxSummary   = 200
	automationInstant      = "2006-01-02T15:04:05Z"
)

// AutomationWorkflow is one registered workflow from GYST_N8N_WORKFLOWS.
type AutomationWorkflow struct {
	ID   string
	Name string
}

// AutomationValidationError is a client mistake in an event (HTTP 400).
type AutomationValidationError struct{ Message string }

func (e *AutomationValidationError) Error() string { return e.Message }

func invalidAutomation(format string, args ...any) error {
	return &AutomationValidationError{Message: fmt.Sprintf(format, args...)}
}

// AutomationEventInput is the request body of POST /api/automation/events.
type AutomationEventInput struct {
	EventID      string `json:"event_id"`
	WorkflowID   string `json:"workflow_id"`
	Status       string `json:"status"`
	OccurredAt   string `json:"occurred_at"`
	ErrorSummary string `json:"error_summary"`
}

// AutomationStatus is the derived state of one registered workflow.
type AutomationStatus struct {
	ID                  string
	Name                string
	State               string
	LastRunAt           string // latest occurred_at of any event; "" when none
	LastSuccessAt       string // latest succeeded occurred_at; "" when none
	ConsecutiveFailures int
	Severity            string // "", "warning" (1 or 2 failures) or "critical" (3 or more)
	LastError           string // summary of the latest failure while failures are consecutive
}

func safeAutomationID(s string, max int) bool {
	if s == "" || len(s) > max {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// ParseWorkflows reads GYST_N8N_WORKFLOWS: comma-separated id:name entries.
// It is strict on purpose. An empty list, a bad id, a missing or unsafe name
// or a repeated id makes the whole value invalid, and the endpoint stays off
// rather than guess.
func ParseWorkflows(raw string) ([]AutomationWorkflow, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("no workflows listed")
	}
	var out []AutomationWorkflow
	seen := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		id, name, ok := strings.Cut(strings.TrimSpace(entry), ":")
		id, name = strings.TrimSpace(id), strings.TrimSpace(name)
		if !ok || !safeAutomationID(id, 64) || seen[id] {
			return nil, errors.New("invalid workflow entry")
		}
		if name == "" || utf8.RuneCountInString(name) > 80 || strings.ContainsFunc(name, func(r rune) bool { return unicode.IsControl(r) || r == ':' || r == utf8.RuneError }) {
			return nil, errors.New("invalid workflow name")
		}
		seen[id] = true
		out = append(out, AutomationWorkflow{ID: id, Name: name})
	}
	return out, nil
}

func automationSummary(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > automationMaxSummary {
		s = string([]rune(s)[:automationMaxSummary])
	}
	return strings.TrimSpace(s)
}

// RecordAutomationEvent validates and stores one event. It returns
// AutomationApplied or AutomationDuplicate; ignored is true when the workflow
// is not registered (nothing is stored). Validation failures are
// *AutomationValidationError.
func (s *Service) RecordAutomationEvent(in AutomationEventInput, workflows []AutomationWorkflow) (result string, ignored bool, err error) {
	if id := in.EventID; id == "" || len(id) > 128 || strings.ContainsFunc(id, func(r rune) bool { return r < 0x21 || r > 0x7e }) {
		return "", false, invalidAutomation("event_id must be 1 to 128 printable characters")
	}
	if !safeAutomationID(in.WorkflowID, 64) {
		return "", false, invalidAutomation("workflow_id is invalid")
	}
	switch in.Status {
	case AutomationStatusStarted, AutomationStatusSucceeded, AutomationStatusFailed:
	default:
		return "", false, invalidAutomation("status must be started, succeeded or failed")
	}
	occurred, perr := time.Parse(time.RFC3339, in.OccurredAt)
	if perr != nil {
		return "", false, invalidAutomation("occurred_at must be an RFC 3339 time")
	}
	now := s.Now().UTC()
	if d := occurred.Sub(now); d > automationWindow || d < -automationWindow {
		return "", false, invalidAutomation("occurred_at is more than 5 minutes from server time")
	}
	registered := false
	for _, w := range workflows {
		if w.ID == in.WorkflowID {
			registered = true
			break
		}
	}
	if !registered {
		return "", true, nil
	}
	inserted, err := s.Repo.InsertAutomationEvent(repository.AutomationEvent{
		EventID:      in.EventID,
		WorkflowID:   in.WorkflowID,
		Status:       in.Status,
		OccurredAt:   occurred.UTC().Format(automationInstant),
		ReceivedAt:   now.Format(automationInstant),
		ErrorSummary: automationSummary(in.ErrorSummary),
	})
	if err != nil {
		return "", false, err
	}
	if !inserted {
		return AutomationDuplicate, false, nil
	}
	if _, err := s.Repo.PruneAutomationEventsBefore(now.Add(-automationRetention).Format(automationInstant)); err != nil {
		return "", false, err
	}
	return AutomationApplied, false, nil
}

// AutomationStatuses derives the state of each registered workflow. Failed
// workflows come first, then the registration order.
//
// State comes from the latest event by occurred_at: a start within the last 2
// hours means Running. Otherwise the latest finished run decides: failed is
// Failed, succeeded is Healthy. A workflow with only a stale start, or no
// events, is "No events".
func (s *Service) AutomationStatuses(workflows []AutomationWorkflow) ([]AutomationStatus, error) {
	ids := make([]string, len(workflows))
	for i, w := range workflows {
		ids[i] = w.ID
	}
	events, err := s.Repo.AutomationEventsForWorkflows(ids)
	if err != nil {
		return nil, err
	}
	byWorkflow := map[string][]repository.AutomationEvent{}
	for _, e := range events {
		byWorkflow[e.WorkflowID] = append(byWorkflow[e.WorkflowID], e)
	}
	now := s.Now().UTC()
	out := make([]AutomationStatus, 0, len(workflows))
	for _, w := range workflows {
		out = append(out, deriveAutomationStatus(w, byWorkflow[w.ID], now))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].State == AutomationStateFailed && out[j].State != AutomationStateFailed
	})
	return out, nil
}

func deriveAutomationStatus(w AutomationWorkflow, events []repository.AutomationEvent, now time.Time) AutomationStatus {
	st := AutomationStatus{ID: w.ID, Name: w.Name, State: AutomationStateNoEvents}
	// Newest first by occurred_at. At the same instant a finish counts as
	// later than a start, so arrival order never decides.
	sorted := append([]repository.AutomationEvent(nil), events...)
	rank := func(e repository.AutomationEvent) int {
		if e.Status == AutomationStatusStarted {
			return 0
		}
		return 1
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.OccurredAt != b.OccurredAt {
			return a.OccurredAt > b.OccurredAt
		}
		if rank(a) != rank(b) {
			return rank(a) > rank(b)
		}
		return a.ReceivedAt > b.ReceivedAt
	})
	if len(sorted) == 0 {
		return st
	}
	st.LastRunAt = sorted[0].OccurredAt
	counting := true
	var latestFinish *repository.AutomationEvent
	for i := range sorted {
		e := &sorted[i]
		if e.Status == AutomationStatusStarted {
			continue
		}
		if latestFinish == nil {
			latestFinish = e
		}
		if e.Status == AutomationStatusSucceeded {
			if st.LastSuccessAt == "" {
				st.LastSuccessAt = e.OccurredAt
			}
			counting = false
		} else if counting {
			if st.ConsecutiveFailures == 0 {
				st.LastError = e.ErrorSummary
			}
			st.ConsecutiveFailures++
		}
	}
	switch {
	case st.ConsecutiveFailures >= 3:
		st.Severity = AutomationSeverityCritical
	case st.ConsecutiveFailures >= 1:
		st.Severity = AutomationSeverityWarning
	}
	if latest := sorted[0]; latest.Status == AutomationStatusStarted {
		if started, err := time.Parse(automationInstant, latest.OccurredAt); err == nil && now.Sub(started) <= automationRunningLimit {
			st.State = AutomationStateRunning
			return st
		}
	}
	switch {
	case latestFinish == nil:
	case latestFinish.Status == AutomationStatusFailed:
		st.State = AutomationStateFailed
	default:
		st.State = AutomationStateHealthy
	}
	return st
}
