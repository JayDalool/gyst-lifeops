package services

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/repository"
)

var auditPinned = time.Date(2026, 10, 10, 18, 30, 15, 123456789, time.UTC)

func auditService(t *testing.T) *Service {
	t.Helper()
	svc := testService(t)
	svc.Now = func() time.Time { return auditPinned }
	return svc
}

func goodAudit() repository.AuditEvent {
	return repository.AuditEvent{ActorType: AuditActorHuman, ActorRef: "session:0a1b2c3d", Ingress: AuditIngressPublic,
		Action: AuditActionLoginFailed, Outcome: AuditOutcomeDenied, RequestID: "abc123"}
}

func auditRows(t *testing.T, svc *Service) []repository.AuditEvent {
	t.Helper()
	rows, err := svc.Repo.RecentAuditEvents(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestRecordAuditStampsClockAndIgnoresCallerAt(t *testing.T) {
	svc := auditService(t)
	e := goodAudit()
	e.At = "1999-01-01T00:00:00.000000000Z"
	e.ID = 77
	if err := svc.RecordAudit(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	rows := auditRows(t, svc)
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if want := clock.FormatInstant(auditPinned); rows[0].At != want {
		t.Fatalf("at = %q, want %q", rows[0].At, want)
	}
	if rows[0].ID == 77 {
		t.Fatal("caller-supplied id was used")
	}
}

func TestRecordAuditRejectsInvalidAndWritesNothing(t *testing.T) {
	svc := auditService(t)
	cases := map[string]func(*repository.AuditEvent){
		"unknown action":      func(e *repository.AuditEvent) { e.Action = "login.bypass" },
		"agent action":        func(e *repository.AuditEvent) { e.Action = "agent.queued" },
		"empty action":        func(e *repository.AuditEvent) { e.Action = "" },
		"bad actor type":      func(e *repository.AuditEvent) { e.ActorType = "robot" },
		"bad ingress":         func(e *repository.AuditEvent) { e.Ingress = "lan" },
		"bad outcome":         func(e *repository.AuditEvent) { e.Outcome = "maybe" },
		"actor space":         func(e *repository.AuditEvent) { e.ActorRef = "session:abc def" },
		"actor equals":        func(e *repository.AuditEvent) { e.ActorRef = "x=y" },
		"target quote":        func(e *repository.AuditEvent) { e.TargetRef = "a'b" },
		"actor too long":      func(e *repository.AuditEvent) { e.ActorRef = strings.Repeat("a", 121) },
		"target too long":     func(e *repository.AuditEvent) { e.TargetRef = strings.Repeat("a", 201) },
		"request id upper":    func(e *repository.AuditEvent) { e.RequestID = "ABC" },
		"request id too long": func(e *repository.AuditEvent) { e.RequestID = strings.Repeat("a", 33) },
		"actor newline":       func(e *repository.AuditEvent) { e.ActorRef = "a\nb" },
	}
	for name, mutate := range cases {
		e := goodAudit()
		mutate(&e)
		if err := svc.RecordAudit(context.Background(), e); err != ErrAuditInvalid {
			t.Errorf("%s: err = %v, want ErrAuditInvalid", name, err)
		}
	}
	if n := len(auditRows(t, svc)); n != 0 {
		t.Fatalf("rows = %d, want 0", n)
	}
}

func TestRecordAuditAcceptsEveryAllowedAction(t *testing.T) {
	svc := auditService(t)
	for a := range auditActions {
		e := goodAudit()
		e.Action = a
		if err := svc.RecordAudit(context.Background(), e); err != nil {
			t.Errorf("%s: %v", a, err)
		}
	}
	if n := len(auditRows(t, svc)); n != len(auditActions) {
		t.Fatalf("rows = %d, want %d", n, len(auditActions))
	}
}

func TestRecordAuditRedactsSecretLookingDetail(t *testing.T) {
	svc := auditService(t)
	details := []string{
		"password=hunter2-fictional",
		"Bearer test-token-fictional",
		"gyst_session=abc",
		"hash $2a$10$fictionalfictionalfictional",
		"$2b$ and $2y$ prefixes",
		"AUTHORIZATION: Basic Zm9v",
		"Cookie: a=b",
		"my API_KEY is x",
		"apikey=x",
		"client secret",
		"passwd entered",
		"pass\nword=hunter2-fictional",
		"X-Api-Key: fictional",
		"api-key fictional",
		"pass\u200bword=fictional",
		"\uff50\uff41\uff53\uff53\uff57\uff4f\uff52\uff44=fictional",
		"%70assword=fictional",
		"100% %70assword=fictional",
		"%zz %70assword=fictional",
		"%70assword=fictional %",
		"eyJa.b.c",
		"EYJa.b.c",
		"\u202etoken\u202c fictional",
		"AbCdEfGhIjKlMnOpQrStUvWxYz0123456789_-abcde",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"sk-ant-fictional",
		"key ghp_fictional",
		"gho_fictional",
		"github_pat_fictional",
		"xoxb-fictional",
		"eyJfictional.fictional.fictional",
		"Basic ZmljdGlvbmFs",
		"$2a$10$fictional",
		"private_key fictional",
		"pwd=fictional",
		"set-cookie fictional",
	}
	for _, d := range details {
		e := goodAudit()
		e.Detail = d
		if err := svc.RecordAudit(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	rows := auditRows(t, svc)
	if len(rows) != len(details) {
		t.Fatalf("rows = %d, want %d", len(rows), len(details))
	}
	for _, r := range rows {
		if r.Detail != "redacted" {
			t.Errorf("detail %q stored, want redacted", r.Detail)
		}
		if strings.Contains(r.Detail+r.ActorRef+r.TargetRef, "hunter2") {
			t.Errorf("secret leaked in %+v", r)
		}
	}
}

func TestRecordAuditKeepsNormalDetail(t *testing.T) {
	svc := auditService(t)
	want := []string{"link 42 refreshed", "host example.test", "task-12 done", "attempt 3 of 5", "caf\u00e9 \u65e5\u672c"}
	for _, d := range want {
		e := goodAudit()
		e.Detail = d
		if err := svc.RecordAudit(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	rows := auditRows(t, svc) // newest first
	for i, r := range rows {
		if w := want[len(want)-1-i]; r.Detail != w {
			t.Errorf("detail %q, want kept as %q", r.Detail, w)
		}
	}
}

func TestRecordAuditStoredDetailHasNoInvisibleRunes(t *testing.T) {
	svc := auditService(t)
	e := goodAudit()
	e.Detail = "ok\u200b\u202e\u2066value\ufeff"
	if err := svc.RecordAudit(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if got := auditRows(t, svc)[0].Detail; got != "okvalue" {
		t.Fatalf("detail = %q", got)
	}
}

func TestRecordAuditDetailControlCharsAndTruncation(t *testing.T) {
	svc := auditService(t)
	e := goodAudit()
	e.Detail = "line one\r\nline two\x00\x1b[0m"
	if err := svc.RecordAudit(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	e = goodAudit()
	e.Detail = strings.Repeat("é", 300)
	if err := svc.RecordAudit(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	rows := auditRows(t, svc) // newest first; same at, so id desc
	if got := utf8.RuneCountInString(rows[0].Detail); got != 200 || !utf8.ValidString(rows[0].Detail) {
		t.Fatalf("long detail: %d runes valid=%v", got, utf8.ValidString(rows[0].Detail))
	}
	if rows[1].Detail != "line oneline two[0m" {
		t.Fatalf("detail = %q", rows[1].Detail)
	}
}

func TestSessionActorRef(t *testing.T) {
	raw := "fictional-raw-session-token-0123456789abcdef"
	got := SessionActorRef(raw)
	hash := hashSessionToken(raw)
	if got != "session:"+hash[:8] || len(got) != len("session:")+8 {
		t.Fatalf("ref = %q", got)
	}
	if strings.Contains(raw, strings.TrimPrefix(got, "session:")) || strings.Contains(got, hash[:9]) {
		t.Fatalf("ref %q leaks token or more of the hash", got)
	}
	if SessionActorRef("") != "session:none" {
		t.Fatal("empty token must give session:none")
	}
	e := goodAudit()
	e.ActorRef = got
	if err := auditService(t).RecordAudit(context.Background(), e); err != nil {
		t.Fatalf("ref must pass the charset rule: %v", err)
	}
}

func TestRecordAuditSurvivesCancelledContext(t *testing.T) {
	svc := auditService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.RecordAudit(ctx, goodAudit()); err != nil {
		t.Fatal(err)
	}
	if n := len(auditRows(t, svc)); n != 1 {
		t.Fatalf("rows = %d, want 1", n)
	}
}
