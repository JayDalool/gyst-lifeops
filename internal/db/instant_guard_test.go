package db

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// GYST-VN-004: instants are written by the app through clock.FormatInstant.
// Migrations added after the cutoff must not use SQLite now-defaults, which
// store 'YYYY-MM-DD HH:MM:SS' with no offset. Older files are exempt.
const instantGuardCutoff = "20261007010000_health_water.sql"

// Matched against the whole file so a call split across lines is still caught
// ([^)] spans newlines). The no-argument forms, such as datetime(), also mean now.
var bannedNow = regexp.MustCompile(`(?i)\bcurrent_(timestamp|date|time)\b|\b(datetime|date|time|strftime|julianday|unixepoch)\s*\([^)]*'now'|\b(datetime|date|time|julianday|unixepoch)\s*\(\s*\)`)

func TestBannedNowPatterns(t *testing.T) {
	for _, s := range []string{
		"created_at TEXT DEFAULT CURRENT_TIMESTAMP",
		"d DEFAULT current_date",
		"t DEFAULT (CURRENT_TIME)",
		"DEFAULT (datetime('now'))",
		"DEFAULT (strftime('%s','now'))",
		"SELECT julianday('now')",
		"DATE( 'now', 'localtime')",
		"DEFAULT (datetime(\n'now'))",
		"DEFAULT (strftime('%s',\n  'now'))",
		"DEFAULT (datetime())",
		"x DEFAULT (date( ))",
		"time()",
		"SELECT julianday()",
		"DEFAULT (unixepoch())",
		"unixepoch('now')",
	} {
		if !bannedNow.MatchString(s) {
			t.Errorf("should flag %q", s)
		}
	}
	for _, s := range []string{
		"current_total INTEGER NOT NULL",
		"-- the first of 'now' and later",
		"at TEXT NOT NULL",
	} {
		if bannedNow.MatchString(s) {
			t.Errorf("should not flag %q", s)
		}
	}
}

func TestNewMigrationsAvoidSQLiteNowDefaults(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	sort.Strings(files)
	for _, f := range files {
		name := filepath.Base(f)
		if name <= instantGuardCutoff {
			continue
		}
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, loc := range bannedNow.FindAllStringIndex(text, -1) {
			line := strings.Count(text[:loc[0]], "\n") + 1
			t.Errorf("%s:%d uses a SQLite now-default; write clock.FormatInstant(Clock.Now()) from the app (GYST-VN-004): %s", name, line, strings.Join(strings.Fields(text[loc[0]:loc[1]]), " "))
		}
	}
}
