package app

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lifeops-tracker/internal/services"
)

// GYST-VN-078 at startup. Hashes are fictional strings.
const (
	startHashA = "$2a$04$fictionalstartAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	startHashB = "$2a$04$fictionalstartBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
)

func startWith(t *testing.T, dbPath, hash string) *App {
	t.Helper()
	cfg := timezoneConfig(t, dbPath, "")
	cfg.PasswordHash = hash
	a, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestStartupRevokesAllSessionsWhenHashChanges(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "r.db")
	a := startWith(t, dbPath, startHashA)
	if _, _, err := a.service.CreateSession(); err != nil {
		t.Fatal(err)
	}
	a.Close()

	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)
	b := startWith(t, dbPath, startHashB)
	defer b.Close()
	if n, _ := b.service.Repo.CountSessions(); n != 0 {
		t.Fatalf("sessions after hash change = %d, want 0", n)
	}
	out := logs.String()
	if !strings.Contains(out, "session.revoked_all, 1 sessions") {
		t.Fatalf("missing security log line: %q", out)
	}
	if strings.Contains(out, startHashA) || strings.Contains(out, startHashB) {
		t.Fatal("log contains a password hash")
	}
	rows, err := b.service.Repo.RecentAuditEvents(context.Background(), 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("audit rows = %d, %v; want 1", len(rows), err)
	}
	e := rows[0]
	if e.Action != services.AuditActionSessionRevokeAll || e.ActorType != services.AuditActorSystem ||
		e.Outcome != services.AuditOutcomeSuccess {
		t.Fatalf("audit row = %+v", e)
	}
	if text := e.ActorRef + e.TargetRef + e.Detail; strings.Contains(text, "$2") {
		t.Fatal("audit row carries hash text")
	}
}

func TestStartupFirstRunAndSameHashKeepSessions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "r.db")
	// Sessions created before any fingerprint exists (the first deploy).
	a := startWith(t, dbPath, startHashA)
	if _, err := a.DB.Exec(`DELETE FROM app_settings WHERE setting_key = 'auth.password_fingerprint'`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.service.CreateSession(); err != nil {
		t.Fatal(err)
	}
	a.Close()

	b := startWith(t, dbPath, startHashA) // no stored fingerprint: silent
	if n, _ := b.service.Repo.CountSessions(); n != 1 {
		t.Fatalf("first-run sessions = %d, want 1", n)
	}
	if rows, _ := b.service.Repo.RecentAuditEvents(context.Background(), 10); len(rows) != 0 {
		t.Fatalf("first run wrote %d audit rows", len(rows))
	}
	b.Close()

	c := startWith(t, dbPath, startHashA) // unchanged
	defer c.Close()
	if n, _ := c.service.Repo.CountSessions(); n != 1 {
		t.Fatalf("unchanged-hash sessions = %d, want 1", n)
	}
}

func TestStartupFailsClosedWhenFingerprintCannotBeStored(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "r.db")
	a := startWith(t, dbPath, startHashA)
	if _, err := a.DB.Exec(`CREATE TRIGGER block_fp BEFORE UPDATE ON app_settings
		BEGIN SELECT RAISE(ABORT, 'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	a.Close()
	cfg := timezoneConfig(t, dbPath, "")
	cfg.PasswordHash = startHashB
	if b, err := New(cfg); err == nil {
		b.Close()
		t.Fatal("startup must fail when the new fingerprint cannot be stored")
	}
}
