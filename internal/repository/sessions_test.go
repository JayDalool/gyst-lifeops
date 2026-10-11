package repository

import (
	"testing"
	"time"
)

// RULES 1.5: probe the real format. A row inserted without created_at gets the
// SQLite DEFAULT, and GetSession must read it as the correct UTC instant.
func TestGetSessionParsesLegacyDefaultCreatedAt(t *testing.T) {
	repo := testRepository(t)
	if _, err := repo.db.Exec(
		`INSERT INTO sessions (token_hash, expires_at, last_seen_at) VALUES (?, ?, ?)`,
		"legacy", "2026-02-01T00:00:00.000000000Z", "2026-01-06T00:00:00.000000000Z"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var raw string
	if err := repo.db.QueryRow(`SELECT created_at FROM sessions WHERE token_hash = 'legacy'`).Scan(&raw); err != nil {
		t.Fatalf("read raw: %v", err)
	}
	if _, err := time.Parse(sqliteTimestampLayout, raw); err != nil {
		t.Fatalf("DEFAULT format changed, got %q: %v", raw, err)
	}
	got, err := repo.GetSession("legacy")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	want, _ := time.ParseInLocation(sqliteTimestampLayout, raw, time.UTC)
	if !got.CreatedAt.Equal(want) || got.CreatedAt.Location() != time.UTC {
		t.Errorf("created_at = %v, want %v UTC", got.CreatedAt, want)
	}
}

func TestCreateSessionWritesExplicitCreatedAt(t *testing.T) {
	repo := testRepository(t)
	created := time.Date(2026, 1, 5, 12, 0, 0, 123, time.UTC)
	if err := repo.CreateSession("h", created.Add(time.Hour), created); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetSession("h")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, created)
	}
}

func TestGetSessionRejectsUnparseableCreatedAt(t *testing.T) {
	repo := testRepository(t)
	if _, err := repo.db.Exec(
		`INSERT INTO sessions (token_hash, created_at, expires_at, last_seen_at) VALUES ('bad', 'garbage', ?, ?)`,
		"2026-02-01T00:00:00.000000000Z", "2026-01-06T00:00:00.000000000Z"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := repo.GetSession("bad"); err == nil {
		t.Fatal("garbage created_at must be an error, not a zero time")
	}
}
