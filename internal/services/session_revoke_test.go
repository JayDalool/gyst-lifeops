package services

import (
	"regexp"
	"strings"
	"testing"
)

// GYST-VN-078. The hashes below are fictional strings, not real bcrypt output.
const (
	revokeHashA = "$2a$04$fictionalhashAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	revokeHashB = "$2a$04$fictionalhashBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
)

func makeSessions(t *testing.T, svc *Service, n int) []string {
	t.Helper()
	var tokens []string
	for i := 0; i < n; i++ {
		tok, _, err := svc.CreateSession()
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, tok)
	}
	return tokens
}

func TestRevokeAllSessionsRemovesEveryRow(t *testing.T) {
	svc, _, _ := capFixture(t)
	tokens := makeSessions(t, svc, 3)
	n, err := svc.RevokeAllSessions()
	if err != nil || n != 3 {
		t.Fatalf("revoked %d, %v; want 3", n, err)
	}
	for _, tok := range tokens {
		if err := svc.ValidateSession(tok); err != ErrNoSession {
			t.Fatalf("old token still valid: %v", err)
		}
	}
}

func TestPasswordChangeFirstRunRecordsSilently(t *testing.T) {
	svc, _, _ := capFixture(t)
	tokens := makeSessions(t, svc, 2)
	n, changed, err := svc.RevokeSessionsOnPasswordChange(revokeHashA)
	if err != nil || changed || n != 0 {
		t.Fatalf("first run = %d, %v, %v; want 0, false, nil", n, changed, err)
	}
	if err := svc.ValidateSession(tokens[0]); err != nil {
		t.Fatalf("first run signed someone out: %v", err)
	}
	stored, _ := svc.Repo.GetSetting(passwordFingerprintKey)
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(stored) {
		t.Fatalf("stored fingerprint %q is not 64 hex characters", stored)
	}
	if stored == revokeHashA || strings.Contains(stored, revokeHashA) {
		t.Fatal("fingerprint equals or contains the hash")
	}
}

func TestPasswordChangeSameHashKeepsSessions(t *testing.T) {
	svc, _, _ := capFixture(t)
	svc.RevokeSessionsOnPasswordChange(revokeHashA)
	tokens := makeSessions(t, svc, 2)
	// Whitespace around the hash is trimmed, as the env loader may add it.
	n, changed, err := svc.RevokeSessionsOnPasswordChange(" " + revokeHashA + "\n")
	if err != nil || changed || n != 0 {
		t.Fatalf("same hash = %d, %v, %v", n, changed, err)
	}
	if err := svc.ValidateSession(tokens[1]); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordChangeDifferentHashRevokesAll(t *testing.T) {
	svc, _, _ := capFixture(t)
	svc.RevokeSessionsOnPasswordChange(revokeHashA)
	tokens := makeSessions(t, svc, 3)
	n, changed, err := svc.RevokeSessionsOnPasswordChange(revokeHashB)
	if err != nil || !changed || n != 3 {
		t.Fatalf("changed hash = %d, %v, %v; want 3, true", n, changed, err)
	}
	for _, tok := range tokens {
		if svc.ValidateSession(tok) != ErrNoSession {
			t.Fatal("session survived a password change")
		}
	}
	stored, _ := svc.Repo.GetSetting(passwordFingerprintKey)
	if stored != passwordFingerprint(revokeHashB) {
		t.Fatal("new fingerprint not stored")
	}
	// A second start with the new hash is a no-op.
	if _, changed, _ := svc.RevokeSessionsOnPasswordChange(revokeHashB); changed {
		t.Fatal("same new hash reported as changed")
	}
}

func TestPasswordChangeEmptyHashHasOwnFingerprintAndRevokes(t *testing.T) {
	if passwordFingerprint("") == passwordFingerprint(revokeHashA) {
		t.Fatal("empty hash shares a fingerprint")
	}
	svc, _, _ := capFixture(t)
	svc.RevokeSessionsOnPasswordChange(revokeHashA)
	makeSessions(t, svc, 1)
	n, changed, err := svc.RevokeSessionsOnPasswordChange("  ")
	if err != nil || !changed || n != 1 {
		t.Fatalf("removing the hash = %d, %v, %v; want 1, true", n, changed, err)
	}
}

// Delete first, then store: if storing fails the old fingerprint remains, so
// the next start revokes again rather than keeping stale sessions.
func TestPasswordChangeStoreFailureKeepsOldFingerprint(t *testing.T) {
	svc, database, _ := capFixture(t)
	svc.RevokeSessionsOnPasswordChange(revokeHashA)
	makeSessions(t, svc, 1)
	if _, err := database.Exec(`CREATE TRIGGER block_fp BEFORE UPDATE ON app_settings
		BEGIN SELECT RAISE(ABORT, 'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.RevokeSessionsOnPasswordChange(revokeHashB); err == nil {
		t.Fatal("want an error when the fingerprint cannot be stored")
	}
	if n, _ := svc.Repo.CountSessions(); n != 0 {
		t.Fatalf("sessions = %d; deletion must happen before the store", n)
	}
	if stored, _ := svc.Repo.GetSetting(passwordFingerprintKey); stored != passwordFingerprint(revokeHashA) {
		t.Fatal("old fingerprint must remain so the next start revokes again")
	}
}

func TestPasswordChangeReadAndDeleteErrorsFailClosed(t *testing.T) {
	svc, database, _ := capFixture(t)
	svc.RevokeSessionsOnPasswordChange(revokeHashA)
	makeSessions(t, svc, 1)
	if _, err := database.Exec(`CREATE TRIGGER block_del BEFORE DELETE ON sessions
		BEGIN SELECT RAISE(ABORT, 'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.RevokeSessionsOnPasswordChange(revokeHashB); err == nil {
		t.Fatal("want an error when sessions cannot be deleted")
	}
	if stored, _ := svc.Repo.GetSetting(passwordFingerprintKey); stored != passwordFingerprint(revokeHashA) {
		t.Fatal("fingerprint must not advance when revocation failed")
	}
	if _, err := database.Exec(`ALTER TABLE app_settings RENAME TO app_settings_gone`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.RevokeSessionsOnPasswordChange(revokeHashB); err == nil {
		t.Fatal("want an error when the fingerprint cannot be read")
	}
}
