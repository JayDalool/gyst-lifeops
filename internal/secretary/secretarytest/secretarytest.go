// Package secretarytest is the fake-data helper for tests of the Secretary's
// tools, including the tools later chunks add (Issue #236). It gives a migrated
// temporary database, a pinned clock and fictional rows. Every value here is
// made up; none comes from a real account.
package secretarytest

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/secretary"
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
	env, closeDB, err := open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeDB)
	env.T = t
	return env
}

// Demo is New for Example functions, which have no *testing.T. It panics if
// the database cannot be built, and the returned function closes it. The seed
// helpers need a test, so an Example uses only the fakes.
func Demo() (*Env, func()) {
	dir, err := os.MkdirTemp("", "secretary-demo-")
	if err != nil {
		panic(err)
	}
	env, closeDB, err := open(filepath.Join(dir, "lifeops.db"))
	if err != nil {
		panic(err)
	}
	return env, func() { closeDB(); os.RemoveAll(dir) }
}

func open(path string) (*Env, func(), error) {
	template.once.Do(func() { template.data, template.err = buildTemplate() })
	if template.err != nil {
		return nil, nil, fmt.Errorf("build the template database: %w", template.err)
	}
	if err := os.WriteFile(path, template.data, 0o600); err != nil {
		return nil, nil, err
	}
	database, err := db.Open(path)
	if err != nil {
		return nil, nil, err
	}
	svc := services.New(repository.New(database))
	loc := svc.Location()
	now := Now(loc)
	svc.Clock = clock.Clock{Now: func() time.Time { return now }, Loc: loc}
	return &Env{DB: database, Svc: svc}, func() { database.Close() }, nil
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

// Sentinels are strings planted in fields a Secretary tool must never return
// (descriptions, notes, URLs). A test that finds one in a tool's output has
// found a leak.
var Sentinels = []string{"SENTINEL-DESC", "SENTINEL-NOTES", "SENTINEL-UNLOCK", "SENTINEL-URL", "SENTINEL-SUMMARY"}

// Toolset is anything that offers tools; *secretary.Catalogue is one.
type Toolset interface{ Tools() []healthcoach.Tool }

// Run calls one tool by name and fails the test if it is missing.
func Run(t *testing.T, set Toolset, name, input string) (string, error) {
	t.Helper()
	out, err, found := call(set, name, input)
	if !found {
		t.Fatalf("tool %q is not offered", name)
	}
	return out, err
}

// RunTool is Run for Example functions: a missing tool is an error.
func RunTool(set Toolset, name, input string) (string, error) {
	out, err, found := call(set, name, input)
	if !found {
		return "", fmt.Errorf("tool %q is not offered", name)
	}
	return out, err
}

func call(set Toolset, name, input string) (string, error, bool) {
	for _, tool := range set.Tools() {
		if tool.Name == name {
			out, err := tool.Run(context.Background(), json.RawMessage(input))
			return out, err, true
		}
	}
	return "", nil, false
}

func (e *Env) exec(q string, args ...any) {
	e.T.Helper()
	if _, err := e.DB.Exec(q, args...); err != nil {
		e.T.Fatal(err)
	}
}

// SeedTasks adds fictional tasks around the pinned day (Thursday 2026-11-19):
// one late, three due today (one repeating, with a reminder time), two later
// this week, one next week, one without a date, and one already done. Every
// description holds a sentinel.
func (e *Env) SeedTasks() {
	e.T.Helper()
	mk := func(title, due, prio, repeat, remind, status string) {
		task := models.Task{Title: title, Description: "SENTINEL-DESC " + title, Category: "personal", Priority: prio, Status: status, DueDate: due, RepeatType: repeat, ReminderTime: remind}
		if err := e.Svc.Repo.CreateTask(&task); err != nil {
			e.T.Fatal(err)
		}
	}
	mk("Renew library card", "2026-11-17", "high", "none", "", "pending")
	mk("Water the plants", "2026-11-10", "medium", "daily", "09:00", "pending")
	mk("Draft the <b>report</b> & send", "2026-11-19", "high", "none", "", "pending")
	mk("Pay the water bill", "2026-11-19", "low", "none", "", "completed")
	mk("Call the dentist", "2026-11-21", "medium", "none", "", "pending")
	mk("Book a table", "2026-11-24", "low", "none", "", "pending")
	mk("Plan the garden", "2026-12-05", "low", "none", "", "pending")
	mk("Sort the photos", "", "low", "none", "", "pending")
}

// Projects are the ids SeedProjects made.
type Projects struct {
	Public, OwnPrivate, OwnLocal, ChildOfLocal, InLocalRoadmap, Both, Planned, Archived int64
}

// SeedProjects adds fictional projects: public, private by its own flag,
// local-only by its own flag, a child of a local-only parent, one in a
// local-only roadmap, one with both flags, a planned one and an archived one.
func (e *Env) SeedProjects() Projects {
	e.T.Helper()
	// Migration 006 makes the default roadmap private; make it public so each
	// project's privacy comes from what this seed sets.
	e.exec(`UPDATE roadmaps SET privacy = 'public'`)
	mk := func(name, status string, parent int64, roadmap int64) int64 {
		p := models.Project{Name: name, Description: "SENTINEL-DESC " + name, Notes: "SENTINEL-NOTES", UnlockNotes: "SENTINEL-UNLOCK", Status: status, ParentProjectID: parent, RoadmapID: roadmap, TargetDate: "2026-12-31"}
		if err := e.Svc.Repo.CreateProject(&p); err != nil {
			e.T.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			item := models.ProjectItem{ProjectID: p.ID, Title: fmt.Sprintf("%s step %d", name, i+1), Description: "SENTINEL-DESC"}
			if err := e.Svc.Repo.CreateProjectItem(&item); err != nil {
				e.T.Fatal(err)
			}
		}
		return p.ID
	}
	flags := func(id int64, private, local bool) {
		e.exec(`INSERT OR IGNORE INTO roadmap_project_meta (project_id) VALUES (?)`, id)
		e.exec(`UPDATE roadmap_project_meta SET is_private = ?, local_only = ? WHERE project_id = ?`, private, local, id)
	}
	local := models.Roadmap{Name: "Quiet roadmap", Privacy: "local_only", Status: "active"}
	if err := e.Svc.Repo.CreateRoadmap(&local); err != nil {
		e.T.Fatal(err)
	}
	var p Projects
	p.Public = mk("Garden shed", "active", 0, 0)
	p.OwnPrivate = mk("Reading list", "active", 0, 0)
	flags(p.OwnPrivate, true, false)
	p.OwnLocal = mk("Boat log", "active", 0, 0)
	flags(p.OwnLocal, false, true)
	p.ChildOfLocal = mk("Boat log: sails", "active", p.OwnLocal, 0)
	p.InLocalRoadmap = mk("Letters", "active", 0, local.ID)
	p.Both = mk("Both flags", "active", 0, 0)
	flags(p.Both, true, true)
	p.Planned = mk("Someday kitchen", "upcoming", 0, 0)
	p.Archived = mk("Old project", "archived", 0, 0)
	// One finished step so open_items differs from the total.
	e.exec(`UPDATE project_items SET status = 'completed' WHERE project_id = ? AND title LIKE '% step 1'`, p.Public)
	return p
}

// SeedCourses adds a fictional course with blocks: two today (one timed), one
// overdue, one tomorrow, one skipped today. Notes and URLs hold sentinels.
func (e *Env) SeedCourses() {
	e.T.Helper()
	c := models.Course{Name: "Intro to Pottery", Provider: "Example School", URL: "https://example.invalid/SENTINEL-URL", Status: "active", Notes: "SENTINEL-NOTES"}
	if err := e.Svc.Repo.CreateCourse(&c); err != nil {
		e.T.Fatal(err)
	}
	mk := func(title, due, start, end, status string, catchup bool) {
		b := models.LearningBlock{CourseID: c.ID, Title: title, BlockType: "lesson", Status: status, DueDate: due, StartTime: start, EndTime: end, URL: "https://example.invalid/SENTINEL-URL", Notes: "SENTINEL-NOTES", IsCatchup: catchup}
		if err := e.Svc.Repo.CreateLearningBlock(&b); err != nil {
			e.T.Fatal(err)
		}
	}
	mk("Wheel basics", Today, "18:30", "19:30", "pending", false)
	mk("Glazing reading", Today, "", "", "pending", true)
	mk("Clay types quiz", "2026-11-17", "", "", "pending", false)
	mk("Firing schedules", "2026-11-20", "10:00", "11:00", "pending", false)
	mk("Skipped session", Today, "", "", "skipped", false)
}

// FakeHealth is a secretary.HealthReader with fictional, fixed answers. Err
// makes every call fail, to show that the cause never reaches the model.
type FakeHealth struct {
	Err error
	// Empty makes BriefHealth report nothing observed: every value nil.
	Empty bool
}

func (f FakeHealth) val(name string) (any, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return map[string]any{"tool": name, "calories_kcal": map[string]any{"eaten": 1450, "target": 2300}}, nil
}

// The nine Coach tools, each answering with its own name.
func (f FakeHealth) Today() (any, error)           { return f.val("today") }
func (f FakeHealth) Plan() (any, error)            { return f.val("plan") }
func (f FakeHealth) WeightTrend() (any, error)     { return f.val("weight") }
func (f FakeHealth) TrainingSummary() (any, error) { return f.val("summary") }
func (f FakeHealth) RecentWorkouts(n int) (any, error) {
	v, err := f.val("recent")
	if m, ok := v.(map[string]any); ok {
		m["count"] = n
	}
	return v, err
}
func (f FakeHealth) ExerciseProgress(w int) (any, error) {
	v, err := f.val("progress")
	if m, ok := v.(map[string]any); ok {
		m["weeks"] = w
	}
	return v, err
}
func (f FakeHealth) HevyRoutines() (any, error)   { return f.val("routines") }
func (f FakeHealth) FoodProducts() (any, error)   { return f.val("foods") }
func (f FakeHealth) DailyChecklist() (any, error) { return f.val("checklist") }

// BriefHealth is a fictional Health summary for the day brief: a synced day
// with a food budget left, one step count and an observed balance.
func (f FakeHealth) BriefHealth() (secretary.HealthBrief, error) {
	if f.Err != nil {
		return secretary.HealthBrief{}, f.Err
	}
	if f.Empty {
		return secretary.HealthBrief{}, nil
	}
	s := func(v string) *string { return &v }
	return secretary.HealthBrief{
		AsOf:          time.Date(2026, 11, 19, 16, 40, 0, 0, time.UTC), // 11:40 local
		FoodEatenKcal: s("1,450"), FoodTargetKcal: s("2,300"), FoodBudgetLeftKcal: s("850"),
		ProteinEatenG: s("96"), ProteinTargetG: s("180"),
		Steps: s("5,200"), StepsSource: "watch",
		BurnedSoFarKcal: s("1,410"), EnergyBalanceSoFarKcal: s("+40"),
		WaterLoggedL: s("1.5"), WaterTargetL: s("3.0"), WeightKg: s("82.4"),
	}, nil
}

// FakeCalendar is a CalendarReader with n fictional events on the asked day,
// 30 minutes apart from 08:00, alternating busy and free.
type FakeCalendar struct {
	N int
	// Title makes the title of event i; nil gives "Fictional meeting i".
	Title func(i int) string
	Err   error
	// SyncedAt is when the copy these events come from was refreshed; the zero
	// time means never synced.
	SyncedAt time.Time
}

// Events implements secretary.CalendarReader.
func (f FakeCalendar) Events(day secretary.Day) (secretary.CalendarSnapshot, error) {
	if f.Err != nil {
		return secretary.CalendarSnapshot{}, f.Err
	}
	out := make([]secretary.CalendarEvent, 0, f.N)
	for i := 0; i < f.N; i++ {
		start := day.Time().Add(8*time.Hour + time.Duration(i)*30*time.Minute)
		title := fmt.Sprintf("Fictional meeting %d", i+1)
		if f.Title != nil {
			title = f.Title(i)
		}
		out = append(out, secretary.CalendarEvent{Start: start, End: start.Add(25 * time.Minute), Busy: i%2 == 0, Title: title})
	}
	return secretary.CalendarSnapshot{Events: out, SyncedAt: f.SyncedAt}, nil
}
