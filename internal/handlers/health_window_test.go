package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func windowHandlerSetup(t *testing.T) *Handler {
	h := testHandler(t)
	h.HealthSyncToken = healthTestToken
	h.Service.Now = func() time.Time { return time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC) }
	return h
}

func windowBody(date string) string {
	return `{"date":"` + date + `","source":"apple_shortcuts","dietary_calories":1357,"protein_g":7654.25}`
}

func TestHealthSyncWindowHandler(t *testing.T) {
	h := windowHandlerSetup(t)
	for _, date := range []string{"2026-10-11", "2026-10-04"} {
		if rec := healthRequest(h, "POST", "/api/health/sync", windowBody(date), healthTestToken, "application/json"); rec.Code != 200 {
			t.Fatalf("%s: %d %s", date, rec.Code, rec.Body.String())
		}
	}
	before, err := h.Service.GetHealthSyncState("apple_shortcuts")
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	origOut, origFlags := log.Writer(), log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	defer func() { log.SetOutput(origOut); log.SetFlags(origFlags) }()

	for _, date := range []string{"2026-10-03", "2026-10-12"} {
		rec := healthRequest(h, "POST", "/api/health/sync", windowBody(date), healthTestToken, "application/json")
		if rec.Code != 422 {
			t.Fatalf("%s: %d %s", date, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Cache-Control") != "no-store" || !strings.Contains(rec.Body.String(), "date must be between 2026-10-04 and 2026-10-11") {
			t.Fatalf("%s: headers/body: %v %s", date, rec.Header(), rec.Body.String())
		}
		if _, err := h.Service.GetDailyHealthSnapshot(date); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("%s: snapshot written: %v", date, err)
		}
	}
	after, err := h.Service.GetHealthSyncState("apple_shortcuts")
	if err != nil || after != before {
		t.Fatalf("sync state changed: %+v -> %+v (%v)", before, after, err)
	}
	if strings.Count(logs.String(), "SECURITY: health sync refused: date outside window") != 2 {
		t.Fatalf("log: %q", logs.String())
	}
	for _, bad := range []string{"7654", "1357", healthTestToken} {
		if strings.Contains(logs.String(), bad) {
			t.Fatalf("log contains %q", bad)
		}
	}

	events, err := h.Service.Repo.RecentAuditEvents(context.Background(), 10)
	if err != nil || len(events) != 1 { // two refusals, one row: the audit gate allows one per client per period
		t.Fatalf("audit rows: %d %v", len(events), err)
	}
	for _, e := range events {
		if e.Action != "health.sync_refused" || e.Outcome != "denied" || e.ActorType != "machine" || e.TargetRef != "health:sync" {
			t.Fatalf("audit row: %+v", e)
		}
		if e.Detail != "date=2026-10-03 window=2026-10-04..2026-10-11" && e.Detail != "date=2026-10-12 window=2026-10-04..2026-10-11" {
			t.Fatalf("audit detail: %q", e.Detail)
		}
		for _, bad := range []string{"7654", "1357", healthTestToken} {
			if strings.Contains(e.Detail+e.ActorRef+e.TargetRef, bad) {
				t.Fatalf("audit row contains %q", bad)
			}
		}
	}
}

func TestHealthSyncWindowOrderOfChecks(t *testing.T) {
	h := windowHandlerSetup(t)
	old := windowBody("2026-09-01")
	for _, token := range []string{"", "wrong-test-token"} {
		if rec := healthRequest(h, "POST", "/api/health/sync", old, token, "application/json"); rec.Code != 401 {
			t.Fatalf("token %q: %d", token, rec.Code)
		}
	}
	if rec := healthRequest(h, "POST", "/api/health/sync", old, healthTestToken, "text/plain"); rec.Code != 415 {
		t.Fatalf("media: %d", rec.Code)
	}
	if rec := healthRequest(h, "POST", "/api/health/sync", windowBody("2026-02-30"), healthTestToken, "application/json"); rec.Code != 400 {
		t.Fatalf("malformed date: %d", rec.Code)
	}
	if rec := healthRequest(h, "POST", "/api/health/sync", old, healthTestToken, "application/json"); rec.Code != 422 {
		t.Fatalf("old date: %d", rec.Code)
	}
	events, err := h.Service.Repo.RecentAuditEvents(context.Background(), 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("only the authenticated refusal is audited: %d %v", len(events), err)
	}
}

func TestHealthSyncRefusalBurstWritesOneAuditRowButEveryLogLine(t *testing.T) {
	h := windowHandlerSetup(t)
	var logs bytes.Buffer
	origOut, origFlags := log.Writer(), log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	defer func() { log.SetOutput(origOut); log.SetFlags(origFlags) }()
	const n = 50
	for i := 0; i < n; i++ {
		if rec := healthRequest(h, "POST", "/api/health/sync", windowBody("2026-09-01"), healthTestToken, "application/json"); rec.Code != 422 {
			t.Fatalf("request %d: %d", i, rec.Code)
		}
	}
	events, err := h.Service.Repo.RecentAuditEvents(context.Background(), 100)
	if err != nil || len(events) != 1 {
		t.Fatalf("audit rows after %d refusals: %d %v", n, len(events), err)
	}
	if got := strings.Count(logs.String(), "SECURITY: health sync refused: date outside window"); got != n {
		t.Fatalf("SECURITY lines: %d want %d", got, n)
	}
	// After the period the next refusal may write again.
	now := h.Service.Now()
	h.Service.Now = func() time.Time { return now.Add(16 * time.Minute) }
	healthRequest(h, "POST", "/api/health/sync", windowBody("2026-09-01"), healthTestToken, "application/json")
	if events, _ := h.Service.Repo.RecentAuditEvents(context.Background(), 100); len(events) != 2 {
		t.Fatalf("audit rows after the period: %d", len(events))
	}
}

func TestHealthSyncRefusalThroughRequestLogIsNotAnError(t *testing.T) {
	rig := newLogRig(t)
	rig.h.HealthSyncToken = healthTestToken
	rig.h.Service.Now = func() time.Time { return time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC) }
	var std bytes.Buffer
	origOut, origFlags := log.Writer(), log.Flags()
	log.SetOutput(&std)
	log.SetFlags(0)
	defer func() { log.SetOutput(origOut); log.SetFlags(origFlags) }()
	rig.buf.Reset()

	req := httptest.NewRequest("POST", "/api/health/sync", strings.NewReader(windowBody("2026-09-01")))
	req.Header.Set("Authorization", "Bearer "+healthTestToken)
	req.Header.Set("Content-Type", "application/json")
	if w := rig.do(req); w.Code != 422 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(rig.buf.String(), "request failed") || strings.Contains(rig.buf.String(), `"level":"ERROR"`) {
		t.Fatalf("422 filed as an error: %s", rig.buf.String())
	}
	if got := strings.Count(std.String(), "SECURITY: health sync refused: date outside window"); got != 1 {
		t.Fatalf("SECURITY lines: %d", got)
	}
}
