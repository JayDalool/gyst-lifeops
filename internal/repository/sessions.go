package repository

import (
	"database/sql"
	"time"
)

// Session is a stored login session. The raw token is deliberately absent:
// only its SHA-256 hash is ever written, so nothing read out of this table
// can be turned back into a usable cookie.
type Session struct {
	TokenHash  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastSeenAt time.Time
}

// sqliteTimestampLayout is what the column DEFAULT (CURRENT_TIMESTAMP) wrote
// before created_at was set explicitly: UTC, second precision, space separated.
// Rows from that era are still read, and capped from their real creation time.
const sqliteTimestampLayout = "2006-01-02 15:04:05"

func parseSessionCreatedAt(value string) (time.Time, error) {
	if at, err := time.Parse(sessionTimeLayout, value); err == nil {
		return at, nil
	}
	return time.ParseInLocation(sqliteTimestampLayout, value, time.UTC)
}

// sessionTimeLayout keeps stored timestamps fixed-width and UTC so TEXT
// comparison in SQL orders them chronologically.
const sessionTimeLayout = "2006-01-02T15:04:05.000000000Z"

func formatSessionTime(at time.Time) string {
	return at.UTC().Format(sessionTimeLayout)
}

// CreateSession stores a new session. A session is last seen the moment it is
// created, so lastSeenAt is also written as created_at: the service clock, not
// the column DEFAULT (wall clock), fixes the start of the absolute cap.
func (r *Repository) CreateSession(tokenHash string, expiresAt, lastSeenAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO sessions (token_hash, created_at, expires_at, last_seen_at) VALUES (?, ?, ?, ?)`,
		tokenHash, formatSessionTime(lastSeenAt), formatSessionTime(expiresAt), formatSessionTime(lastSeenAt))
	return err
}

// GetSession returns the session for a token hash. A missing row reports
// sql.ErrNoRows; expiry is the service's decision, not the repository's.
func (r *Repository) GetSession(tokenHash string) (Session, error) {
	var session Session
	var createdAt, expiresAt, lastSeenAt string
	err := r.db.QueryRow(
		`SELECT token_hash, created_at, expires_at, last_seen_at FROM sessions WHERE token_hash = ?`,
		tokenHash).Scan(&session.TokenHash, &createdAt, &expiresAt, &lastSeenAt)
	if err != nil {
		return Session{}, err
	}
	// An unreadable created_at is an error, never a zero time: the caller
	// would otherwise cap from year 1 (refuse) or skip the cap (accept).
	if session.CreatedAt, err = parseSessionCreatedAt(createdAt); err != nil {
		return Session{}, err
	}
	if session.ExpiresAt, err = time.Parse(sessionTimeLayout, expiresAt); err != nil {
		return Session{}, err
	}
	if session.LastSeenAt, err = time.Parse(sessionTimeLayout, lastSeenAt); err != nil {
		return Session{}, err
	}
	return session, nil
}

// TouchSession extends a session's rolling expiry. The service calls this at
// most once per refresh interval rather than on every request.
func (r *Repository) TouchSession(tokenHash string, expiresAt, lastSeenAt time.Time) error {
	_, err := r.db.Exec(
		`UPDATE sessions SET expires_at = ?, last_seen_at = ? WHERE token_hash = ?`,
		formatSessionTime(expiresAt), formatSessionTime(lastSeenAt), tokenHash)
	return err
}

// DeleteSession removes one session and reports whether a row was removed.
func (r *Repository) DeleteSession(tokenHash string) (bool, error) {
	result, err := r.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

// DeleteExpiredSessions removes sessions that lapsed before the given instant
// and reports how many rows went. Scoped by expiry, never a blanket delete.
func (r *Repository) DeleteExpiredSessions(before time.Time) (int64, error) {
	result, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, formatSessionTime(before))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CountSessions is used by tests and diagnostics.
func (r *Repository) CountSessions() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}
