package services

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func testAuthConfig(t *testing.T, password string) AuthConfig {
	t.Helper()
	// MinCost keeps the suite fast; production hashes come from
	// cmd/passwordhash, which uses a deliberately higher cost.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return AuthConfig{PasswordHash: string(hash)}
}

func TestVerifyPassword(t *testing.T) {
	config := testAuthConfig(t, "correct horse battery staple")

	if err := config.VerifyPassword("correct horse battery staple"); err != nil {
		t.Errorf("correct password rejected: %v", err)
	}
	if err := config.VerifyPassword("wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password: got %v, want ErrInvalidCredentials", err)
	}
	if err := config.VerifyPassword(""); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("empty password: got %v, want ErrInvalidCredentials", err)
	}
}

// TestUnconfiguredPasswordFailsClosed matches the precedent set by the Health
// sync token: with no secret configured, nothing authenticates.
func TestUnconfiguredPasswordFailsClosed(t *testing.T) {
	for _, hash := range []string{"", "   "} {
		config := AuthConfig{PasswordHash: hash}
		if config.Configured() {
			t.Errorf("Configured() = true for %q, want false", hash)
		}
		if err := config.VerifyPassword("anything"); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("unconfigured hash %q accepted a password: %v", hash, err)
		}
	}
}

func TestSessionLifecycle(t *testing.T) {
	service := testService(t)

	rawToken, expiresAt, err := service.CreateSession()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if rawToken == "" {
		t.Fatal("created session returned an empty token")
	}
	if !expiresAt.After(service.Now()) {
		t.Errorf("expiry %v is not in the future", expiresAt)
	}

	if err := service.ValidateSession(rawToken); err != nil {
		t.Errorf("fresh session rejected: %v", err)
	}

	if err := service.RevokeSession(rawToken); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := service.ValidateSession(rawToken); !errors.Is(err, ErrNoSession) {
		t.Errorf("revoked session: got %v, want ErrNoSession", err)
	}
}

func TestValidateSessionRejectsUnknownAndEmptyTokens(t *testing.T) {
	service := testService(t)

	if err := service.ValidateSession(""); !errors.Is(err, ErrNoSession) {
		t.Errorf("empty token: got %v, want ErrNoSession", err)
	}
	if err := service.ValidateSession("not-a-real-token"); !errors.Is(err, ErrNoSession) {
		t.Errorf("unknown token: got %v, want ErrNoSession", err)
	}
}

// TestRawTokenIsNeverStored is the point of hashing session tokens: a reader
// of the database - a backup, a stolen disk - must not be able to mint a
// cookie from what is stored.
func TestRawTokenIsNeverStored(t *testing.T) {
	service := testService(t)
	repo := service.Repo

	rawToken, _, err := service.CreateSession()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := repo.GetSession(rawToken); err == nil {
		t.Fatal("the raw token is usable as a primary key; it must be stored hashed")
	}

	stored, err := repo.GetSession(hashSessionToken(rawToken))
	if err != nil {
		t.Fatalf("session not found under its hash: %v", err)
	}
	if stored.TokenHash == rawToken {
		t.Error("stored token_hash equals the raw token")
	}
	if len(stored.TokenHash) != 64 {
		t.Errorf("token_hash is %d chars, want 64 (hex SHA-256)", len(stored.TokenHash))
	}
}

func TestSessionsAreUnique(t *testing.T) {
	service := testService(t)

	seen := map[string]bool{}
	for index := 0; index < 20; index++ {
		rawToken, _, err := service.CreateSession()
		if err != nil {
			t.Fatalf("create session %d: %v", index, err)
		}
		if seen[rawToken] {
			t.Fatalf("session token repeated after %d sessions", index)
		}
		seen[rawToken] = true
	}
}

func TestExpiredSessionIsRejectedAndRemoved(t *testing.T) {
	service := testService(t)
	repo := service.Repo

	rawToken, _, err := service.CreateSession()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Step past the session lifetime.
	base := service.Now()
	service.Now = func() time.Time { return base.Add(sessionLifetime + time.Minute) }

	if err := service.ValidateSession(rawToken); !errors.Is(err, ErrNoSession) {
		t.Errorf("expired session: got %v, want ErrNoSession", err)
	}

	count, err := repo.CountSessions()
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 0 {
		t.Errorf("expired session row remained: %d rows", count)
	}
}

// TestRollingExpiryIsThrottled pins the behaviour the review asked for: the
// session rolls forward, but not with a database write on every request.
func TestRollingExpiryIsThrottled(t *testing.T) {
	service := testService(t)
	repo := service.Repo

	rawToken, _, err := service.CreateSession()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	hash := hashSessionToken(rawToken)

	original, err := repo.GetSession(hash)
	if err != nil {
		t.Fatalf("read session: %v", err)
	}

	// Well inside the refresh interval: validating must not rewrite the row.
	base := service.Now()
	service.Now = func() time.Time { return base.Add(time.Hour) }
	for request := 0; request < 5; request++ {
		if err := service.ValidateSession(rawToken); err != nil {
			t.Fatalf("validate within interval: %v", err)
		}
	}
	unchanged, err := repo.GetSession(hash)
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	if !unchanged.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("expiry moved inside the refresh interval: %v -> %v",
			original.ExpiresAt, unchanged.ExpiresAt)
	}
	if !unchanged.LastSeenAt.Equal(original.LastSeenAt) {
		t.Errorf("last_seen_at moved inside the refresh interval: %v -> %v",
			original.LastSeenAt, unchanged.LastSeenAt)
	}

	// Past the refresh interval: the session rolls forward once.
	service.Now = func() time.Time { return base.Add(sessionRefreshInterval + time.Minute) }
	if err := service.ValidateSession(rawToken); err != nil {
		t.Fatalf("validate past interval: %v", err)
	}
	refreshed, err := repo.GetSession(hash)
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	if !refreshed.ExpiresAt.After(original.ExpiresAt) {
		t.Errorf("expiry did not roll forward: %v -> %v",
			original.ExpiresAt, refreshed.ExpiresAt)
	}
	if !refreshed.LastSeenAt.After(original.LastSeenAt) {
		t.Errorf("last_seen_at did not advance: %v -> %v",
			original.LastSeenAt, refreshed.LastSeenAt)
	}
}

func TestSweepExpiredSessionsLeavesLiveOnes(t *testing.T) {
	service := testService(t)
	repo := service.Repo

	expiring, _, err := service.CreateSession()
	if err != nil {
		t.Fatalf("create expiring session: %v", err)
	}

	// Create the second session far enough ahead that it is still live when
	// the first has lapsed.
	base := service.Now()
	service.Now = func() time.Time { return base.Add(sessionLifetime - time.Hour) }
	live, _, err := service.CreateSession()
	if err != nil {
		t.Fatalf("create live session: %v", err)
	}

	service.Now = func() time.Time { return base.Add(sessionLifetime + time.Minute) }
	removed, err := service.SweepExpiredSessions()
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 1 {
		t.Errorf("swept %d sessions, want 1", removed)
	}

	if err := service.ValidateSession(live); err != nil {
		t.Errorf("live session was swept: %v", err)
	}
	if err := service.ValidateSession(expiring); !errors.Is(err, ErrNoSession) {
		t.Errorf("expired session survived: %v", err)
	}

	count, err := repo.CountSessions()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("sessions remaining = %d, want 1", count)
	}
}
