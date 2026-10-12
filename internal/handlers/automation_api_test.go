package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/services"
)

// Generated test values, not real credentials.
var (
	autoTestToken   = "n8n-test-" + strings.Repeat("a1", 20)
	autoHealthToken = "health-test-" + strings.Repeat("b2", 20)
	autoNow         = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
)

const autoWorkflows = "wf_backup:Nightly backup,wf_digest:Daily digest"

func autoHandler(t *testing.T) *Handler {
	t.Helper()
	h := testHandler(t)
	h.HealthSyncToken = autoHealthToken
	h.N8NEventToken = autoTestToken
	h.N8NWorkflows = autoWorkflows
	h.Service.Now = func() time.Time { return autoNow }
	return h
}

func autoBody(id, workflow, status string, at time.Time) string {
	return fmt.Sprintf(`{"event_id":%q,"workflow_id":%q,"status":%q,"occurred_at":%q}`, id, workflow, status, at.Format(time.RFC3339))
}

func autoEventCount(t *testing.T, h *Handler) int {
	t.Helper()
	events, err := h.Service.Repo.AutomationEventsForWorkflows([]string{"wf_backup", "wf_digest", "wf_other"})
	if err != nil {
		t.Fatal(err)
	}
	return len(events)
}

func TestAutomationAPITable(t *testing.T) {
	valid := autoBody("evt-1", "wf_backup", "succeeded", autoNow)
	for _, test := range []struct {
		name, method, path, body, token, media string
		status                                 int
		wantStatus                             string
		wantRows                               int
	}{
		{"applied", "POST", "/api/automation/events", valid, autoTestToken, "application/json", 200, "applied", 1},
		{"applied with charset", "POST", "/api/automation/events", valid, autoTestToken, "application/json; charset=utf-8", 200, "applied", 1},
		{"offset time normalised", "POST", "/api/automation/events", autoBody("evt-2", "wf_backup", "started", autoNow.In(time.FixedZone("x", -6*3600))), autoTestToken, "application/json", 200, "applied", 1},
		{"unregistered workflow", "POST", "/api/automation/events", autoBody("evt-3", "wf_other", "failed", autoNow), autoTestToken, "application/json", 202, "ignored", 0},
		{"no bearer", "POST", "/api/automation/events", valid, "", "application/json", 401, "", 0},
		{"wrong bearer", "POST", "/api/automation/events", valid, "wrong-" + autoTestToken, "application/json", 401, "", 0},
		{"health token", "POST", "/api/automation/events", valid, autoHealthToken, "application/json", 401, "", 0},
		{"get without bearer", "GET", "/api/automation/events", "", "", "", 401, "", 0},
		{"stale", "POST", "/api/automation/events", autoBody("evt-4", "wf_backup", "succeeded", autoNow.Add(-5*time.Minute-time.Second)), autoTestToken, "application/json", 400, "", 0},
		{"future", "POST", "/api/automation/events", autoBody("evt-5", "wf_backup", "succeeded", autoNow.Add(5*time.Minute+time.Second)), autoTestToken, "application/json", 400, "", 0},
		{"edge of window", "POST", "/api/automation/events", autoBody("evt-6", "wf_backup", "succeeded", autoNow.Add(-5*time.Minute)), autoTestToken, "application/json", 200, "applied", 1},
		{"bad time", "POST", "/api/automation/events", `{"event_id":"e","workflow_id":"wf_backup","status":"started","occurred_at":"yesterday"}`, autoTestToken, "application/json", 400, "", 0},
		{"time without zone", "POST", "/api/automation/events", `{"event_id":"e","workflow_id":"wf_backup","status":"started","occurred_at":"2026-10-11T12:00:00"}`, autoTestToken, "application/json", 400, "", 0},
		{"bad status", "POST", "/api/automation/events", strings.Replace(valid, "succeeded", "done", 1), autoTestToken, "application/json", 400, "", 0},
		{"empty event id", "POST", "/api/automation/events", strings.Replace(valid, "evt-1", "", 1), autoTestToken, "application/json", 400, "", 0},
		{"long event id", "POST", "/api/automation/events", strings.Replace(valid, "evt-1", strings.Repeat("x", 129), 1), autoTestToken, "application/json", 400, "", 0},
		{"bad workflow id", "POST", "/api/automation/events", strings.Replace(valid, "wf_backup", "wf backup\\n", 1), autoTestToken, "application/json", 400, "", 0},
		{"unknown field", "POST", "/api/automation/events", strings.Replace(valid, "}", `,"extra":1}`, 1), autoTestToken, "application/json", 400, "", 0},
		{"array", "POST", "/api/automation/events", "[" + valid + "]", autoTestToken, "application/json", 400, "", 0},
		{"null", "POST", "/api/automation/events", "null", autoTestToken, "application/json", 400, "", 0},
		{"malformed", "POST", "/api/automation/events", "{", autoTestToken, "application/json", 400, "", 0},
		{"trailing object", "POST", "/api/automation/events", valid + " {}", autoTestToken, "application/json", 400, "", 0},
		{"non-string summary", "POST", "/api/automation/events", strings.Replace(valid, "}", `,"error_summary":5}`, 1), autoTestToken, "application/json", 400, "", 0},
		{"method", "GET", "/api/automation/events", "", autoTestToken, "", 405, "", 0},
		{"put", "PUT", "/api/automation/events", valid, autoTestToken, "application/json", 405, "", 0},
		{"media", "POST", "/api/automation/events", valid, autoTestToken, "text/plain", 415, "", 0},
		{"missing media", "POST", "/api/automation/events", valid, autoTestToken, "", 415, "", 0},
		{"oversize", "POST", "/api/automation/events", `{"event_id":"` + strings.Repeat("x", 17<<10) + `"}`, autoTestToken, "application/json", 413, "", 0},
		{"oversize trailing whitespace", "POST", "/api/automation/events", valid + strings.Repeat(" ", 17<<10), autoTestToken, "application/json", 413, "", 0},
		{"unknown route", "POST", "/api/automation/other", valid, autoTestToken, "application/json", 404, "", 0},
		{"nested route", "POST", "/api/automation/events/extra", valid, autoTestToken, "application/json", 404, "", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := autoHandler(t)
			rec := healthRequest(h, test.method, test.path, test.body, test.token, test.media)
			if rec.Code != test.status {
				t.Fatalf("status %d want %d: %s", rec.Code, test.status, rec.Body.String())
			}
			if !json.Valid(rec.Body.Bytes()) || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatal("non-JSON response")
			}
			if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Vary") != "Authorization" {
				t.Fatal("missing no-store or Vary")
			}
			if test.status == 401 && rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatal("401 without WWW-Authenticate: Bearer")
			}
			if test.status == 405 && rec.Header().Get("Allow") != "POST" {
				t.Fatal("405 without Allow")
			}
			if strings.Contains(rec.Body.String(), autoTestToken) || strings.Contains(rec.Body.String(), autoHealthToken) {
				t.Fatal("token in response")
			}
			if test.wantStatus != "" {
				var got map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got["status"] != test.wantStatus || len(got) != 1 {
					t.Fatalf("body %s want status %q only", rec.Body.String(), test.wantStatus)
				}
			}
			if n := autoEventCount(t, h); n != test.wantRows {
				t.Fatalf("rows %d want %d", n, test.wantRows)
			}
		})
	}
}

func TestAutomationAPISessionCookieIsNotEnough(t *testing.T) {
	h := autoHandler(t)
	req := httptest.NewRequest("POST", "/api/automation/events", strings.NewReader(autoBody("evt-1", "wf_backup", "started", autoNow)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "any-session-value"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 401 || rec.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("cookie only: %d", rec.Code)
	}
}

func TestAutomationAPIDuplicateChangesNothing(t *testing.T) {
	h := autoHandler(t)
	body := autoBody("evt-1", "wf_backup", "failed", autoNow)
	if rec := healthRequest(h, "POST", "/api/automation/events", body, autoTestToken, "application/json"); rec.Code != 200 {
		t.Fatalf("first: %d", rec.Code)
	}
	// Same id, different content: still a duplicate, the stored row stays.
	again := autoBody("evt-1", "wf_backup", "succeeded", autoNow.Add(time.Minute))
	rec := healthRequest(h, "POST", "/api/automation/events", again, autoTestToken, "application/json")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"duplicate"`) {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body.String())
	}
	statuses, err := h.Service.AutomationStatuses([]services.AutomationWorkflow{{ID: "wf_backup", Name: "Nightly backup"}})
	if err != nil || autoEventCount(t, h) != 1 || statuses[0].State != services.AutomationStateFailed {
		t.Fatalf("state changed: %v %+v", err, statuses)
	}
}

func TestAutomationAPIFailsClosed(t *testing.T) {
	valid := autoBody("evt-1", "wf_backup", "started", autoNow)
	for _, test := range []struct {
		name  string
		setup func(h *Handler)
	}{
		{"empty token", func(h *Handler) { h.N8NEventToken = "" }},
		{"short token", func(h *Handler) { h.N8NEventToken = strings.Repeat("s", 31) }},
		{"same as health token", func(h *Handler) { h.N8NEventToken = autoHealthToken }},
		{"empty workflows", func(h *Handler) { h.N8NWorkflows = "" }},
		{"unparsable workflows", func(h *Handler) { h.N8NWorkflows = "wf_backup" }},
		{"bad workflow entry", func(h *Handler) { h.N8NWorkflows = "wf_backup:Backup,bad id:Oops" }},
		{"duplicate workflow", func(h *Handler) { h.N8NWorkflows = "wf_backup:A,wf_backup:B" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := autoHandler(t)
			test.setup(h)
			// 503 before any credential check, whatever the request carries.
			for _, token := range []string{autoTestToken, "", autoHealthToken} {
				rec := healthRequest(h, "POST", "/api/automation/events", valid, token, "application/json")
				if rec.Code != 503 || !strings.Contains(rec.Body.String(), "not configured") {
					t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
				}
			}
			if n := autoEventCount(t, h); n != 0 {
				t.Fatalf("rows %d", n)
			}
		})
	}
}

func TestAutomationAPIRateLimit(t *testing.T) {
	h := autoHandler(t)
	for i := 0; i < 60; i++ {
		rec := healthRequest(h, "POST", "/api/automation/events", autoBody(fmt.Sprintf("evt-%d", i), "wf_backup", "started", autoNow), autoTestToken, "application/json")
		if rec.Code != 200 {
			t.Fatalf("event %d: %d", i, rec.Code)
		}
	}
	rec := healthRequest(h, "POST", "/api/automation/events", autoBody("evt-60", "wf_backup", "started", autoNow), autoTestToken, "application/json")
	if rec.Code != 429 {
		t.Fatalf("61st event: %d", rec.Code)
	}
	// A bad token never spends the budget.
	if rec := healthRequest(h, "POST", "/api/automation/events", "{}", "wrong", "application/json"); rec.Code != 401 {
		t.Fatalf("unauthorised after limit: %d", rec.Code)
	}
	h.Service.Now = func() time.Time { return autoNow.Add(61 * time.Second) }
	rec = healthRequest(h, "POST", "/api/automation/events", autoBody("evt-61", "wf_backup", "started", autoNow.Add(61*time.Second)), autoTestToken, "application/json")
	if rec.Code != 200 {
		t.Fatalf("after the window: %d", rec.Code)
	}
}

func TestAutomationTokensAreScoped(t *testing.T) {
	h := autoHandler(t)
	// Inside the Health sync window (today-7..today, GYST-116) on the test clock.
	health := `{"date":"` + autoNow.Format("2006-01-02") + `","source":"apple_shortcuts","dietary_calories":130}`
	if rec := healthRequest(h, "POST", "/api/health/sync", health, autoTestToken, "application/json"); rec.Code != 401 {
		t.Fatalf("automation token on health sync: %d", rec.Code)
	}
	if rec := healthRequest(h, "POST", "/api/health/sync", health, autoHealthToken, "application/json"); rec.Code != 200 {
		t.Fatalf("health token on health sync: %d", rec.Code)
	}
}

func TestAutomationEndpointStillNeedsBearerWhenAuthDisabled(t *testing.T) {
	h := autoHandler(t)
	h.AuthDisabled = true
	req := httptest.NewRequest("POST", "/api/automation/events", strings.NewReader(autoBody("evt-1", "wf_backup", "started", autoNow)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("auth disabled, no bearer: %d", rec.Code)
	}
}

func TestAutomationExemptEndpointIsServedWithoutASession(t *testing.T) {
	h := autoHandler(t)
	req := httptest.NewRequest("POST", "/api/automation/events", strings.NewReader(autoBody("evt-1", "wf_backup", "started", autoNow)))
	req.Header.Set("Authorization", "Bearer "+autoTestToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("through the session middleware: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAutomationLogsCarryNoBodyTokenOrSummary(t *testing.T) {
	h := autoHandler(t)
	var logs bytes.Buffer
	h.Logger = slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h.LogAutomationConfig()
	body := `{"event_id":"evt-secret-id","workflow_id":"wf_other","status":"failed","occurred_at":"` + autoNow.Format(time.RFC3339) + `","error_summary":"sensitive-summary-text"}`
	if rec := healthRequest(h, "POST", "/api/automation/events", body, autoTestToken, "application/json"); rec.Code != 202 {
		t.Fatalf("ignored: %d", rec.Code)
	}
	h.N8NEventToken = ""
	h.LogAutomationConfig()
	out := logs.String()
	if strings.Count(out, "automation.ignored") != 1 {
		t.Fatalf("want exactly one ignored line: %s", out)
	}
	for _, banned := range []string{autoTestToken, autoHealthToken, "sensitive-summary-text", "evt-secret-id", "occurred_at"} {
		if strings.Contains(out, banned) {
			t.Fatalf("log leaks %q: %s", banned, out)
		}
	}
	if !strings.Contains(out, "GYST_N8N_EVENT_TOKEN") {
		t.Fatalf("startup line should name the variable: %s", out)
	}
}

func TestAutomationErrorSummaryIsSanitisedBeforeStorage(t *testing.T) {
	h := autoHandler(t)
	summary := "line one\\nline\\u0000two " + strings.Repeat("é", 300)
	body := `{"event_id":"evt-1","workflow_id":"wf_backup","status":"failed","occurred_at":"` + autoNow.Format(time.RFC3339) + `","error_summary":"` + summary + `"}`
	if rec := healthRequest(h, "POST", "/api/automation/events", body, autoTestToken, "application/json"); rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	events, _ := h.Service.Repo.AutomationEventsForWorkflows([]string{"wf_backup"})
	got := events[0].ErrorSummary
	if strings.ContainsAny(got, "\n\x00") || len([]rune(got)) != 200 {
		t.Fatalf("stored summary %q (%d runes)", got, len([]rune(got)))
	}
}
