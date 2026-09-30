package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
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

	// sessionRefreshInterval throttles the rolling-expiry write. Extending the
	// session on every request would mean a database write per page load - on
	// SQLite that is a write lock taken on the request path, for a value whose
	// precision does not matter. Refreshing at most once a day keeps the
	// rolling behaviour while leaving ordinary reads read-only.
	sessionRefreshInterval = 24 * time.Hour
)

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
	expiresAt = now.Add(sessionLifetime)
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
	if !now.Before(session.ExpiresAt) {
		// Clear it on the way past so lapsed rows do not accumulate between
		// sweeps. A failure here must not deny an otherwise correct decision.
		_ = s.Repo.DeleteSession(tokenHash)
		return time.Time{}, ErrNoSession
	}

	if now.Sub(session.LastSeenAt) >= sessionRefreshInterval {
		expiresAt := now.Add(sessionLifetime)
		if err := s.Repo.TouchSession(tokenHash, expiresAt, now); err != nil {
			return time.Time{}, err
		}
		return expiresAt, nil
	}
	return time.Time{}, nil
}

// RevokeSession ends one session. Revoking an unknown token succeeds: logout
// should be idempotent, and reporting "no such session" would confirm to a
// caller which tokens exist.
func (s *Service) RevokeSession(rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.Repo.DeleteSession(hashSessionToken(rawToken))
}

// SweepExpiredSessions deletes lapsed sessions and reports how many went.
func (s *Service) SweepExpiredSessions() (int64, error) {
	return s.Repo.DeleteExpiredSessions(s.Now().UTC())
}
