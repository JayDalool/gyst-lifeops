package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCredentials covers both a wrong password and an unconfigured
// password hash. The caller must not distinguish them to the user: telling an
// attacker which of the two happened is free information.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

// ErrNoSession means the cookie was absent, unknown, or expired. It is not an
// error worth logging; it is the ordinary state of a logged-out browser.
var ErrNoSession = errors.New("auth: no valid session")

const (
	// sessionTokenBytes is the entropy behind a session cookie. 32 bytes is
	// far beyond guessing range and keeps the cookie short enough to be
	// comfortable in a header.
	sessionTokenBytes = 32

	// sessionLifetime is how long a session stays valid without use. It rolls
	// forward while the session is being used.
	sessionLifetime = 30 * 24 * time.Hour

	// sessionAbsoluteLifetime caps a session's whole life from creation, however
	// often it is used (10A D-S5). Rolling renewal never reaches past it.
	sessionAbsoluteLifetime = 90 * 24 * time.Hour

	// sessionRefreshInterval throttles the rolling-expiry write. Extending the
	// session on every request would mean a database write per page load - on
	// SQLite that is a write lock taken on the request path, for a value whose
	// precision does not matter. Refreshing at most once a day keeps the
	// rolling behaviour while leaving ordinary reads read-only.
	sessionRefreshInterval = 24 * time.Hour
)

// sessionExpiry is the earlier of the rolling idle expiry and the absolute
// cap. Used for every write of expires_at so the sweeper sees the cap too.
func sessionExpiry(createdAt, now time.Time) time.Time {
	idle := now.Add(sessionLifetime)
	if capAt := createdAt.Add(sessionAbsoluteLifetime); capAt.Before(idle) {
		return capAt
	}
	return idle
}

// AuthConfig carries the server-side secret. The hash is supplied through the
// environment and is never rendered, logged, or written to the database.
type AuthConfig struct {
	// PasswordHash is a bcrypt hash, as produced by cmd/passwordhash.
	PasswordHash string
}

// hashSessionToken maps a raw token to what is stored. SHA-256 is the right
// tool here and bcrypt is not: the input is 32 bytes of cryptographic
// randomness, not a guessable human password, so there is nothing for a slow
// hash to defend against and it would add latency to every request.
func hashSessionToken(rawToken string) string {
	digest := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(digest[:])
}

// VerifyPassword reports whether the supplied password matches the configured
// hash. It fails closed: with no hash configured, nothing authenticates.
func (c AuthConfig) VerifyPassword(password string) error {
	if strings.TrimSpace(c.PasswordHash) == "" {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

// Configured reports whether a password hash is present. Handlers use this to
// fail closed loudly at startup rather than silently refusing every login.
func (c AuthConfig) Configured() bool {
	return strings.TrimSpace(c.PasswordHash) != ""
}

// CreateSession mints a session and returns the raw token exactly once. Only
// its hash is stored, so this return value is the sole copy: it goes into the
// cookie and is never recoverable from the database afterwards.
func (s *Service) CreateSession() (rawToken string, expiresAt time.Time, err error) {
	buffer := make([]byte, sessionTokenBytes)
	if _, err = rand.Read(buffer); err != nil {
		return "", time.Time{}, err
	}
	rawToken = base64.RawURLEncoding.EncodeToString(buffer)

	now := s.Now().UTC()
	expiresAt = sessionExpiry(now, now)
	if err = s.Repo.CreateSession(hashSessionToken(rawToken), expiresAt, now); err != nil {
		return "", time.Time{}, err
	}
	return rawToken, expiresAt, nil
}

// ValidateSession checks a raw token from a cookie. It returns ErrNoSession
// for anything that is not a live session, and extends the rolling expiry at
// most once per sessionRefreshInterval.
func (s *Service) ValidateSession(rawToken string) error {
	_, err := s.CheckSession(rawToken)
	return err
}

// CheckSession is ValidateSession for callers that own the cookie. When it
// rolls the expiry forward it returns the new expiry, so the caller can renew
// the browser's cookie to match; otherwise the returned time is zero. Without
// this the server-side session keeps rolling while the cookie still expires
// 30 days after login, and the browser signs out a session that is live.
func (s *Service) CheckSession(rawToken string) (renewedUntil time.Time, err error) {
	if rawToken == "" {
		return time.Time{}, ErrNoSession
	}
	tokenHash := hashSessionToken(rawToken)

	session, err := s.Repo.GetSession(tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, ErrNoSession
	}
	if err != nil {
		return time.Time{}, err
	}

	now := s.Now().UTC()
	if !now.Before(session.ExpiresAt) || !now.Before(session.CreatedAt.Add(sessionAbsoluteLifetime)) {
		// Idle-expired or past the absolute cap, even if used yesterday.
		// Clear it on the way past so lapsed rows do not accumulate between
		// sweeps. A failure here must not deny an otherwise correct decision.
		_, _ = s.Repo.DeleteSession(tokenHash)
		return time.Time{}, ErrNoSession
	}

	if now.Sub(session.LastSeenAt) >= sessionRefreshInterval {
		expiresAt := sessionExpiry(session.CreatedAt, now)
		if err := s.Repo.TouchSession(tokenHash, expiresAt, now); err != nil {
			return time.Time{}, err
		}
		return expiresAt, nil
	}
	return time.Time{}, nil
}

// RevokeSession ends one session. Revoking an unknown token succeeds: logout
// should be idempotent, and reporting "no such session" would confirm to a
// caller which tokens exist. The revoked flag is for the server's own audit
// log only; it must never change what the browser is told.
func (s *Service) RevokeSession(rawToken string) (revoked bool, err error) {
	if rawToken == "" {
		return false, nil
	}
	return s.Repo.DeleteSession(hashSessionToken(rawToken))
}

// SweepExpiredSessions deletes lapsed sessions and reports how many went.
func (s *Service) SweepExpiredSessions() (int64, error) {
	return s.Repo.DeleteExpiredSessions(s.Now().UTC())
}

// PasswordHashCost is the bcrypt cost for every newly generated password hash
// (09 section C, S2-12). It is deliberately above the library default: login
// runs on a machine serving one person, so a slower hash costs nothing that
// matters and buys resistance to offline cracking if a hash ever leaks.
// Verification reads the cost from the stored hash, so hashes made at any
// other cost keep working until they are regenerated.
const PasswordHashCost = 12

// ErrEmptyPassword means the password was blank or only whitespace.
var ErrEmptyPassword = errors.New("auth: password must not be empty")

// HashPassword is the only place a password hash is generated. A blank or
// whitespace-only password is rejected. The password is hashed exactly as
// given (no trimming), so it matches what is typed at login. bcrypt's own
// errors, such as input over 72 bytes, are returned rather than truncated.
func HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", ErrEmptyPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), PasswordHashCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// sessionSweepInterval is how often lapsed sessions are removed.
const sessionSweepInterval = time.Hour

// SweepSessionsAtStart runs one sweep and logs the result. app.New calls it
// synchronously before starting the loop, so frequent restarts cannot starve
// the hourly ticker and the sweep cannot race App.Close.
func (s *Service) SweepSessionsAtStart() {
	s.sweepSessionsAndLog(context.Background())
}

// RunSessionSweeper sweeps every sessionSweepInterval until ctx is cancelled.
// Validation already refuses a lapsed session; this only stops dead rows
// piling up.
func (s *Service) RunSessionSweeper(ctx context.Context) {
	ticker := time.NewTicker(sessionSweepInterval)
	defer ticker.Stop()
	s.sweepSessionsOnTick(ctx, ticker.C)
}

// sweepSessionsOnTick sweeps once per tick. The tick value is ignored: the
// cutoff always comes from the injected clock (s.Now).
func (s *Service) sweepSessionsOnTick(ctx context.Context, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			s.sweepSessionsAndLog(ctx)
		}
	}
}

// sweepSessionsAndLog logs counts and errors only, never token material, and
// stays silent when nothing was removed. Once ctx is cancelled the app is
// shutting down and the database may already be closed, so it logs nothing.
func (s *Service) sweepSessionsAndLog(ctx context.Context) {
	n, err := s.SweepExpiredSessions()
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		log.Printf("sessions: sweep failed: %v", err)
		return
	}
	if n > 0 {
		log.Printf("sessions: removed %d expired", n)
	}
}
