// Package secretarytest is the fake-data helper for tests of the Secretary's
// tools, including the tools later chunks add (Issue #236). It gives a migrated
// temporary database, a pinned clock and fictional rows. Every value here is
// made up; none comes from a real account.
package secretarytest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// Env is one test's world: its own database and a clock that stands still.
type Env struct {
	T   *testing.T
	DB  *sql.DB
	Svc *services.Service
}

// Now is the pinned instant: Thursday 2026-11-19, 12:00 local.
func Now(loc *time.Location) time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, loc) }

// Today is the pinned local day.
const Today = "2026-11-19"

// template is the migrated schema, built once per test binary. Migrating takes
// about a second under -race, and every Secretary test needs a database of its
// own (the same shortcut the handlers tests use, Issue #196).
var template struct {
	once       sync.Once
	data       []byte
	err        error
	migrations string
}

func buildTemplate() ([]byte, error) {
	_, here, _, _ := runtime.Caller(0)
	template.migrations = filepath.Join(filepath.Dir(here), "..", "..", "..", "migrations")
	dir, err := os.MkdirTemp("", "secretary-template-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "template.db")
	database, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	defer database.Close()
	if err := db.Migrate(database, template.migrations); err != nil {
		return nil, err
	}
	// Fold the WAL into the main file, which alone is read below.
	var busy, pages, done int
	if err := database.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &pages, &done); err != nil || busy != 0 {
		return nil, fmt.Errorf("checkpoint the template: busy=%d err=%v", busy, err)
	}
	if err := database.Close(); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// New returns an Env on a migrated temporary database with the clock pinned to
// Now in the application zone.
func New(t *testing.T) *Env {
	t.Helper()
	template.once.Do(func() { template.data, template.err = buildTemplate() })
	if template.err != nil {
		t.Fatalf("build the template database: %v", template.err)
	}
	path := filepath.Join(t.TempDir(), "lifeops.db")
	if err := os.WriteFile(path, template.data, 0o600); err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	svc := services.New(repository.New(database))
	loc := svc.Location()
	now := Now(loc)
	svc.Clock = clock.Clock{Now: func() time.Time { return now }, Loc: loc}
	return &Env{T: t, DB: database, Svc: svc}
}

// SetNow moves the pinned clock.
func (e *Env) SetNow(now time.Time) {
	loc := e.Svc.Location()
	e.Svc.Clock = clock.Clock{Now: func() time.Time { return now }, Loc: loc}
}

// DumpTables returns every row of every table except schema_migrations, one
// string per table, so a test can prove which tables a call changed. Columns
// such as updated_at are included.
func DumpTables(t *testing.T, database *sql.DB) map[string]string {
	t.Helper()
	names, err := database.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name <> 'schema_migrations' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for names.Next() {
		var n string
		if err := names.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	}
	names.Close()
	out := map[string]string{}
	for _, table := range tables {
		rows, err := database.Query(fmt.Sprintf(`SELECT * FROM "%s"`, table))
		if err != nil {
			t.Fatalf("dump %s: %v", table, err)
		}
		cols, _ := rows.Columns()
		var lines []string
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			lines = append(lines, fmt.Sprintf("%v", vals))
		}
		rows.Close()
		sort.Strings(lines)
		out[table] = strings.Join(lines, "\n")
	}
	return out
}

// Changed names the tables that differ between two dumps.
func Changed(before, after map[string]string) []string {
	var out []string
	for name, rows := range after {
		if before[name] != rows {
			out = append(out, name)
		}
	}
	for name := range before {
		if _, ok := after[name]; !ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// FakeHevy is a training provider that counts its calls and can be held open.
type FakeHevy struct {
	mu      sync.Mutex
	fetches int
	Entries []models.TrainingEntry
	Err     error
	gate    chan struct{}
	honour  bool
}

// Fetch implements services.TrainingProvider.
func (f *FakeHevy) Fetch(ctx context.Context) ([]models.TrainingEntry, error) {
	f.mu.Lock()
	f.fetches++
	gate := f.gate
	f.mu.Unlock()
	if gate != nil {
		if f.honour {
			select {
			case <-gate:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		} else {
			<-gate
		}
	}
	return f.Entries, f.Err
}

// Fetches is how many times Fetch was called.
func (f *FakeHevy) Fetches() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fetches
}

// SlowHevy blocks every Fetch until Release. honourCtx says whether it also
// returns when its context ends (a well-behaved provider) or ignores it.
func SlowHevy(honourCtx bool) *FakeHevy {
	return &FakeHevy{gate: make(chan struct{}), honour: honourCtx}
}

// Release lets every blocked Fetch, now and later, return.
func (f *FakeHevy) Release() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gate != nil {
		close(f.gate)
		f.gate = nil
	}
}

// ConnectHevy wires the provider into the service, as app.go does.
func (e *Env) ConnectHevy(p services.TrainingProvider) {
	e.Svc.Training = &services.TrainingSync{Provider: p}
}

// SeedHevy stores a fictional Hevy copy last refreshed at syncedAt: one
// finished workout the day before the pinned day, and a routine.
func (e *Env) SeedHevy(syncedAt time.Time) {
	e.T.Helper()
	loc := e.Svc.Location()
	day := Now(loc).AddDate(0, 0, -1)
	w := 60.0
	entries := []models.TrainingEntry{
		{ID: "w1", Kind: "workout", Title: "Push day", StartedAt: day, EndedAt: day.Add(55 * time.Minute), Exercises: []models.TrainingExercise{
			{Name: "Bench Press", PrimaryMuscle: "chest", Sets: []models.TrainingSet{{Kind: "normal", LoadKG: &w, Reps: ptr(8.0)}}}}},
		{ID: "r1", Kind: "routine", Title: "Routine A", Exercises: []models.TrainingExercise{{Name: "Bench Press"}}},
	}
	if err := e.Svc.Repo.SaveTraining("hevy", entries, syncedAt); err != nil {
		e.T.Fatal(err)
	}
}

func ptr[T any](v T) *T { return &v }
