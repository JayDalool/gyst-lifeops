package repository

import (
	"context"
	"strings"
	"testing"
)

const auditTestAt = "2026-10-10T12:00:00.000000000Z"

func validAuditRow() AuditEvent {
	return AuditEvent{At: auditTestAt, ActorType: "human", ActorRef: "session:0a1b2c3d", Ingress: "public",
		Action: "login.failed", Outcome: "denied", RequestID: "abc123"}
}

func TestAuditEventsSQLChecksRejectBadRows(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	if err := r.InsertAuditEvent(ctx, validAuditRow()); err != nil {
		t.Fatalf("valid row: %v", err)
	}
	cases := map[string]func(*AuditEvent){
		"outcome":        func(e *AuditEvent) { e.Outcome = "maybe" },
		"ingress":        func(e *AuditEvent) { e.Ingress = "lan" },
		"actor_type":     func(e *AuditEvent) { e.ActorType = "robot" },
		"at short":       func(e *AuditEvent) { e.At = auditTestAt[:29] },
		"at long":        func(e *AuditEvent) { e.At = auditTestAt + "x" },
		"at no Z":        func(e *AuditEvent) { e.At = strings.TrimSuffix(auditTestAt, "Z") + "+" },
		"at wrong shape": func(e *AuditEvent) { e.At = "2026-10-10 12:00:00.000000000Z......." },
		"detail 201":     func(e *AuditEvent) { e.Detail = strings.Repeat("a", 201) },
		"actor_ref 121":  func(e *AuditEvent) { e.ActorRef = strings.Repeat("a", 121) },
		"target_ref 201": func(e *AuditEvent) { e.TargetRef = strings.Repeat("a", 201) },
		"request_id 33":  func(e *AuditEvent) { e.RequestID = strings.Repeat("a", 33) },
		"action empty":   func(e *AuditEvent) { e.Action = "" },
		"action 65":      func(e *AuditEvent) { e.Action = strings.Repeat("a", 65) },
	}
	for name, mutate := range cases {
		e := validAuditRow()
		mutate(&e)
		if err := r.InsertAuditEvent(ctx, e); err == nil {
			t.Errorf("%s: insert succeeded, want CHECK failure", name)
		}
	}
	got, err := r.RecentAuditEvents(ctx, 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("rows = %d, err %v; want only the valid row", len(got), err)
	}
}

func TestRecentAuditEventsNewestFirst(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	for i, at := range []string{"2026-10-10T12:00:00.000000000Z", "2026-10-10T13:00:00.000000000Z", "2026-10-10T13:00:00.000000000Z"} {
		e := validAuditRow()
		e.At = at
		e.TargetRef = string(rune('a' + i))
		if err := r.InsertAuditEvent(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	got, err := r.RecentAuditEvents(ctx, 2)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %d, err %v", len(got), err)
	}
	if got[0].TargetRef != "c" || got[1].TargetRef != "b" {
		t.Fatalf("order = %q, %q; want c, b", got[0].TargetRef, got[1].TargetRef)
	}
}
