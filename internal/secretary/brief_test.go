package secretary_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/secretary"
	"lifeops-tracker/internal/secretary/secretarytest"
)

func briefCatalogue(t *testing.T, e *secretarytest.Env, h secretarytest.FakeHealth, cal secretary.CalendarReader) *secretary.Catalogue {
	t.Helper()
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: h, Calendar: cal, Budget: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cat.Wait)
	return cat
}

func build(t *testing.T, cat *secretary.Catalogue, date string) ([]byte, map[string]any) {
	t.Helper()
	day, err := secretary.ResolveDate(date, cat.Clock())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := cat.BuildDayBrief(day)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("brief is not JSON: %v\n%s", err, raw)
	}
	return raw, m
}

func addTasks(t *testing.T, e *secretarytest.Env, n int, title func(i int) string) {
	t.Helper()
	for i := 0; i < n; i++ {
		task := models.Task{Title: title(i), Category: "personal", Priority: "medium", Status: "pending", DueDate: secretarytest.Today, RepeatType: "none"}
		if err := e.Svc.Repo.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
	}
}

func counts(t *testing.T, m map[string]any) (tasksListed, tasksTotal, tasksMore, evListed, evTotal, evMore int) {
	t.Helper()
	tk := m["tasks"].(map[string]any)
	ca := m["calendar"].(map[string]any)
	tasksListed, tasksTotal, tasksMore = len(tk["titles"].([]any)), int(tk["due"].(float64)), int(tk["more"].(float64))
	evListed, evTotal, evMore = len(ca["events"].([]any)), int(ca["total"].(float64)), int(ca["more"].(float64))
	return
}

func TestDayBriefBudget(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-10 * time.Minute))
	addTasks(t, e, 50, func(i int) string { return fmt.Sprintf("Fictional task number %d", i+1) })
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, secretarytest.FakeCalendar{N: 30})
	raw, m := build(t, cat, "today")
	if len(raw) > secretary.BriefBudget {
		t.Fatalf("%d bytes", len(raw))
	}
	tl, tt, tm, el, et, em := counts(t, m)
	if tt != 50 || tl+tm != tt || et != 30 || el+em != et || tl != 5 || el == 0 {
		t.Fatalf("tasks %d+%d/%d events %d+%d/%d", tl, tm, tt, el, em, et)
	}
	// Sections come in this fixed order, with nothing cut.
	last := -1
	for _, key := range []string{`"now"`, `"prayer"`, `"tasks"`, `"calendar"`, `"health"`, `"training"`, `"mail"`, `"sync"`} {
		i := strings.Index(string(raw), key)
		if i <= last {
			t.Fatalf("%s out of order in %s", key, raw)
		}
		last = i
	}
	if _, cut := m["cut"]; cut {
		t.Fatalf("sections cut: %v", m["cut"])
	}
}

func TestDayBriefWorstCaseStaysInBudget(t *testing.T) {
	for name, title := range map[string]string{"escaped": strings.Repeat("<", 200), "four-byte": strings.Repeat("\U0001F600", 200), "quotes": strings.Repeat(`"`, 200), "backslash": strings.Repeat(`\`, 200)} {
		t.Run(name, func(t *testing.T) {
			e := secretarytest.New(t)
			e.SeedHevy(e.Svc.Now().Add(-10 * time.Minute))
			if _, err := e.DB.Exec(`UPDATE training_snapshots SET entries_json = ?`, fmt.Sprintf(`[{"ID":"w","Kind":"workout","Title":%q,"StartedAt":"2026-11-18T10:00:00Z"}]`, title)); err != nil {
				t.Fatal(err)
			}
			addTasks(t, e, 50, func(int) string { return title })
			cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, secretarytest.FakeCalendar{N: 30, Title: func(int) string { return title }})
			raw, m := build(t, cat, "today")
			if len(raw) > secretary.BriefBudget {
				t.Fatalf("%d bytes", len(raw))
			}
			tl, tt, tm, el, et, em := counts(t, m)
			if tl+tm != tt || el+em != et || tl == 0 && tt > 0 && m["cut"] == nil {
				t.Fatalf("tasks %d+%d/%d events %d+%d/%d cut %v", tl, tm, tt, el, em, et, m["cut"])
			}
			// The worst-case training title must have been kept whole or the section named in cut.
			if tr, ok := m["training"].(map[string]any); ok {
				if last, _ := tr["last"].(map[string]any); last != nil {
					if got := last["title"].(string); utf8.RuneCountInString(got) > secretary.TitleCap {
						t.Fatalf("training title %d runes", utf8.RuneCountInString(got))
					}
				}
			} else if m["cut"] == nil {
				t.Fatal("training missing and not named in cut")
			}
		})
	}
}

var hiddenAlphabet = []rune("ab é日<>&\"\\ ​‮\U0001F600 \t")

func randomTitle(r *rand.Rand) string {
	n := r.Intn(130)
	out := make([]rune, n)
	for i := range out {
		out[i] = hiddenAlphabet[r.Intn(len(hiddenAlphabet))]
	}
	return string(out)
}

// 500 seeded random inputs: always valid JSON within budget, and every title
// listed is the cleaned whole title, never a piece of one.
func TestDayBriefRandomInputs(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-10 * time.Minute))
	addTasks(t, e, 12, func(i int) string { return fmt.Sprint("t", i) })
	r := rand.New(rand.NewSource(163))
	for round := 0; round < 500; round++ {
		tasks := 12
		want := map[string]bool{}
		tx, err := e.DB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < tasks; i++ {
			title := randomTitle(r)
			if _, err := tx.Exec(`UPDATE tasks SET title = ? WHERE id = (SELECT id FROM tasks ORDER BY id LIMIT 1 OFFSET ?)`, title, i); err != nil {
				t.Fatal(err)
			}
			clean, _ := secretary.Clean(title, secretary.TitleCap)
			want[clean] = true
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		titles := make([]string, r.Intn(45))
		for i := range titles {
			titles[i] = randomTitle(r)
		}
		cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, secretarytest.FakeCalendar{N: len(titles), Title: func(i int) string { return titles[i] }})
		raw, m := build(t, cat, "today")
		if len(raw) > secretary.BriefBudget || !json.Valid(raw) {
			t.Fatalf("round %d: %d bytes", round, len(raw))
		}
		for _, v := range m["tasks"].(map[string]any)["titles"].([]any) {
			if !want[v.(string)] {
				t.Fatalf("round %d: task title %q is not a whole cleaned title", round, v)
			}
		}
		for i, v := range m["calendar"].(map[string]any)["events"].([]any) {
			env := v.(map[string]any)["title"].(map[string]any)
			clean, trunc := secretary.Clean(titles[i], secretary.TitleCap)
			if env["text"] != clean || env["truncated"] != trunc || env["untrusted"] != true || env["source"] != "calendar" {
				t.Fatalf("round %d event %d: %v", round, i, env)
			}
		}
	}
}

func TestDayBriefNullStaysNull(t *testing.T) {
	e := secretarytest.New(t) // no plan, no Hevy, no calendar, no mail
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{Empty: true}, nil)
	_, m := build(t, cat, "today")
	health := m["health"].(map[string]any)
	if health["as_of"] != nil || health["stale"] != true || health["weight"] != nil || health["burned"] != nil || health["balance"] != nil {
		t.Fatalf("health: %v", health)
	}
	for _, group := range []string{"food", "protein", "steps", "water"} {
		for k, v := range health[group].(map[string]any) {
			if v != nil {
				t.Errorf("health.%s.%s = %v, want null", group, k, v)
			}
		}
	}
	tr := m["training"].(map[string]any)
	// Hevy never synced: the week is unknown (null), not "0 workouts" (RULES 3.2).
	if tr["as_of"] != nil || tr["last"] != nil || tr["week"] != nil {
		t.Fatalf("training: %v", tr)
	}
	cal := m["calendar"].(map[string]any)
	if cal["connected"] != false || cal["events"] != nil || cal["as_of"] != nil || cal["total"] != nil {
		t.Fatalf("calendar: %v", cal)
	}
	mail := m["mail"].(map[string]any)
	if mail["connected"] != false || mail["unread"] != nil || mail["as_of"] != nil {
		t.Fatalf("mail: %v", mail)
	}
	sync := m["sync"].(map[string]any)
	for _, k := range []string{"hevy", "calendar", "mail", "health"} {
		if v, ok := sync[k]; !ok || v != nil {
			t.Errorf("sync.%s = %v", k, v)
		}
	}
	// Another day: Health is today only. The section says so, with its as_of,
	// instead of being a bare null.
	_, tomorrow := build(t, cat, "tomorrow")
	h, ok := tomorrow["health"].(map[string]any)
	if _, hasAsOf := h["as_of"]; !ok || !hasAsOf || h["available"] != false || h["reason"] != "today_only" || h["as_of"] != nil || h["stale"] != true {
		t.Fatalf("tomorrow's health: %v", tomorrow["health"])
	}
}

// A synced snapshot with no workouts this week is a real zero; before any
// snapshot the week is null.
func TestDayBriefTrainingWeekZeroVersusNull(t *testing.T) {
	e := secretarytest.New(t)
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{Empty: true}, nil)
	_, m := build(t, cat, "today")
	if m["training"].(map[string]any)["week"] != nil {
		t.Fatalf("never synced: %v", m["training"])
	}
	if err := e.Svc.Repo.SaveTraining("hevy", []models.TrainingEntry{}, e.Svc.Now().Add(-5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	_, m = build(t, cat, "today")
	week, ok := m["training"].(map[string]any)["week"].(map[string]any)
	if !ok || week["workouts"] != 0.0 || week["target"] != nil {
		t.Fatalf("synced, no workouts: %v", m["training"])
	}
}

// Every section says when its data is from and whether that is stale.
func TestDayBriefAsOf(t *testing.T) {
	e := secretarytest.New(t)
	stored := e.Svc.Now().Add(-90 * time.Minute)
	e.SeedHevy(stored)
	e.ConnectHevy(&secretarytest.FakeHevy{})
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, secretarytest.FakeCalendar{N: 2, SyncedAt: e.Svc.Now().Add(-20 * time.Minute)})
	_, m := build(t, cat, "today")
	for _, key := range []string{"now", "prayer", "tasks", "calendar", "health", "training", "mail", "sync"} {
		sec := m[key].(map[string]any)
		if _, ok := sec["as_of"]; !ok {
			t.Errorf("%s has no as_of", key)
		}
		if _, ok := sec["stale"]; !ok {
			t.Errorf("%s has no stale", key)
		}
	}
	if m["now"].(map[string]any)["as_of"] != "2026-11-19T12:00" || m["now"].(map[string]any)["stale"] != false {
		t.Fatalf("now: %v", m["now"])
	}
	tr := m["training"].(map[string]any)
	if tr["as_of"] != "2026-11-19T10:30" || tr["stale"] != true {
		t.Fatalf("training as_of %v stale %v: a 90 minute old copy is stale", tr["as_of"], tr["stale"])
	}
	if m["health"].(map[string]any)["as_of"] != "2026-11-19T11:40" || m["sync"].(map[string]any)["hevy"] != "2026-11-19T10:30" || m["sync"].(map[string]any)["health"] != "2026-11-19T11:40" {
		t.Fatalf("health/sync: %v / %v", m["health"], m["sync"])
	}
	// The calendar's as_of is the reader's own synced-at: 20 minutes is past the 15 minute limit.
	if c := m["calendar"].(map[string]any); c["as_of"] != "2026-11-19T11:40" || c["stale"] != true || m["sync"].(map[string]any)["calendar"] != "2026-11-19T11:40" {
		t.Fatalf("calendar: %v", c)
	}
	if m["tasks"].(map[string]any)["stale"] != false || m["prayer"].(map[string]any)["stale"] != false {
		t.Fatal("local data reported stale")
	}
}

func walkKeys(v any, f func(string)) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			f(k)
			walkKeys(val, f)
		}
	case []any:
		for _, val := range x {
			walkKeys(val, f)
		}
	}
}

func TestDayBriefNoDescriptionsSendersSubjects(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedTasks()
	e.SeedProjects()
	e.SeedCourses()
	e.SeedHevy(e.Svc.Now().Add(-10 * time.Minute))
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, secretarytest.FakeCalendar{N: 5})
	raw, m := build(t, cat, "today")
	banned := map[string]bool{"description": true, "notes": true, "sender": true, "from": true, "subject": true, "body": true, "attendees": true, "location": true, "url": true}
	walkKeys(m, func(k string) {
		if banned[strings.ToLower(k)] {
			t.Errorf("brief has key %q", k)
		}
	})
	noSentinels(t, string(raw))
	if !strings.Contains(string(raw), "Draft the") {
		t.Fatalf("a due task title is missing: %s", raw)
	}
}

// #188: building the brief never starts a sync and never writes.
func TestBriefNeverSyncs(t *testing.T) {
	e := secretarytest.New(t)
	stored := e.Svc.Now().Add(-5 * time.Hour)
	e.SeedHevy(stored)
	fake := &secretarytest.FakeHevy{}
	e.ConnectHevy(fake)
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, nil)
	before := secretarytest.DumpTables(t, e.DB)
	_, m := build(t, cat, "today")
	cat.Wait()
	if fake.Fetches() != 0 || len(secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB))) != 0 {
		t.Fatalf("%d fetches, tables changed", fake.Fetches())
	}
	tr := m["training"].(map[string]any)
	if tr["stale"] != true || tr["as_of"] != "2026-11-19T07:00" || tr["last"].(map[string]any)["title"] != "Push day" {
		t.Fatalf("training: %v", tr)
	}
}

// With a slow provider the tool still answers inside the budget, from the
// stored workout, and says it is stale.
func TestGetDayBriefToolWithSlowHevy(t *testing.T) {
	e := secretarytest.New(t)
	stored := e.Svc.Now().Add(-2 * time.Hour)
	e.SeedHevy(stored)
	slow := secretarytest.SlowHevy(false)
	e.ConnectHevy(slow)
	t.Cleanup(slow.Release)
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: secretarytest.FakeHealth{}, Budget: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cat.Wait)
	start := time.Now()
	out, err := secretarytest.Run(t, cat, "get_day_brief", `{}`)
	if err != nil || time.Since(start) > 3*time.Second {
		t.Fatalf("%v after %v", err, time.Since(start))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil || len(out) > secretary.BriefBudget {
		t.Fatalf("%v, %d bytes", err, len(out))
	}
	tr := m["training"].(map[string]any)
	if tr["stale"] != true || tr["as_of"] != "2026-11-19T10:00" || tr["last"].(map[string]any)["title"] != "Push day" {
		t.Fatalf("training: %v", tr)
	}
	slow.Release()
	cat.Wait()
	if c, _ := e.Svc.Repo.TrainingCache("hevy"); c.Error != "" {
		t.Fatalf("failure row %q", c.Error)
	}
}

func TestGetDayBriefTool(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedTasks()
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, nil)
	for _, in := range []string{``, `{}`, `{"date":"today"}`, `{"date":"tomorrow"}`, `{"date":"2026-12-03"}`, `{"date":"next friday"}`} {
		out, err := secretarytest.Run(t, cat, "get_day_brief", in)
		if err != nil || len(out) > secretary.BriefBudget || !json.Valid([]byte(out)) {
			t.Errorf("%s: %v (%d bytes)", in, err, len(out))
		}
	}
	m := run(t, cat, "get_day_brief", `{"date":"friday"}`)
	if n := m["now"].(map[string]any); n["date"] != "2026-11-20" || n["weekday"] != "Friday" {
		t.Fatalf("now: %v", n)
	}
	secret := "IGNORE-PREVIOUS-INSTRUCTIONS"
	for _, in := range []string{`{"date":"2026-12-04"}`, `{"date":"2026-11-18"}`, `{"date":"` + secret + `"}`, `{`} {
		out, err := secretarytest.Run(t, cat, "get_day_brief", in)
		if err == nil || out != "" || strings.Contains(err.Error(), "IGNORE") {
			t.Errorf("%s: %q %v", in, out, err)
		}
	}
	// A calendar that fails gives a calendar section of nulls, not an error.
	bad := briefCatalogue(t, e, secretarytest.FakeHealth{Err: context.Canceled}, secretarytest.FakeCalendar{Err: context.Canceled})
	_, bm := build(t, bad, "today")
	h, _ := bm["health"].(map[string]any)
	if _, hasAsOf := h["as_of"]; !hasAsOf || h["available"] != false || h["reason"] != "unavailable" || bm["calendar"].(map[string]any)["events"] != nil || bm["calendar"].(map[string]any)["connected"] != true || bm["calendar"].(map[string]any)["as_of"] != nil {
		t.Fatalf("failures: health %v calendar %v", bm["health"], bm["calendar"])
	}
}

// Malformed or out-of-range input answers with its fixed error before any
// freshness check: no fetch, no write, even with a stale Hevy copy.
func TestGetDayBriefValidatesBeforeRefreshing(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-5 * time.Hour))
	fake := &secretarytest.FakeHevy{}
	e.ConnectHevy(fake)
	cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, nil)
	before := secretarytest.DumpTables(t, e.DB)
	for _, in := range []string{`{"date":"2026-12-25"}`, `{"date":"2020-01-01"}`, `{"date":"nonsense"}`, `{"date":1}`, `{`} {
		if out, err := secretarytest.Run(t, cat, "get_day_brief", in); err == nil || out != "" {
			t.Errorf("%s accepted", in)
		}
	}
	cat.Wait()
	if fake.Fetches() != 0 || len(secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB))) != 0 {
		t.Fatalf("%d fetches or a write on invalid input", fake.Fetches())
	}
}

// movingCalendar is a CalendarReader whose stored copy can change between reads.
type movingCalendar struct {
	mu     sync.Mutex
	at     time.Time
	titles []string
}

func (m *movingCalendar) set(at time.Time, titles ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.at, m.titles = at, titles
}

func (m *movingCalendar) Events(day secretary.Day) (secretary.CalendarSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := secretary.CalendarSnapshot{SyncedAt: m.at}
	for i, title := range m.titles {
		start := day.Time().Add(8*time.Hour + time.Duration(i)*time.Hour)
		snap.Events = append(snap.Events, secretary.CalendarEvent{Start: start, End: start.Add(30 * time.Minute), Busy: true, Title: title})
	}
	return snap, nil
}

// GYST-164 c4: the calendar's as_of, stale and sync.calendar come from the
// snapshot the events were read from, not from the turn's cached decision. A
// sync that finishes after the wait budget used to leave the new events under
// the old as_of with stale true.
func TestBriefCalendarFreshnessFollowsTheSnapshotRead(t *testing.T) {
	e := secretarytest.New(t)
	now := e.Svc.Now()
	cal := &movingCalendar{}
	cal.set(now.Add(-2*time.Hour), "Old event")
	release := make(chan struct{})
	var once sync.Once
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: secretarytest.FakeHealth{}, Calendar: cal, Budget: 100 * time.Millisecond,
		Syncers: map[string]secretary.Syncer{secretary.SourceCalendar: {
			LastSync: func() (time.Time, error) { return now.Add(-2 * time.Hour), nil },
			Sync:     func(context.Context) error { <-release; return nil },
		}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cat.Wait)
	t.Cleanup(func() { once.Do(func() { close(release) }) }) // runs before cat.Wait
	out, err := secretarytest.Run(t, cat, "get_day_brief", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatal(err)
	}
	if m["calendar"]["stale"] != true || m["calendar"]["as_of"] != "2026-11-19T10:00" {
		t.Fatalf("while the sync is slow: %v", m["calendar"])
	}
	// The sync finishes after the budget and the reader's copy moves on.
	cal.set(now, "New event")
	once.Do(func() { close(release) })
	cat.Wait()
	_, m2 := build(t, cat, "today")
	c := m2["calendar"].(map[string]any)
	if c["stale"] != false || c["as_of"] != "2026-11-19T12:00" || m2["sync"].(map[string]any)["calendar"] != "2026-11-19T12:00" {
		t.Fatalf("after the sync: %v / %v", c, m2["sync"])
	}
	if ev := c["events"].([]any); len(ev) != 1 || ev[0].(map[string]any)["title"].(map[string]any)["text"] != "New event" {
		t.Fatalf("events: %v", c["events"])
	}
	// A reader older than the Syncer's own LastSync shows its own time.
	cal.set(now.Add(-3*time.Hour), "Older")
	_, m3 := build(t, cat, "today")
	if c := m3["calendar"].(map[string]any); c["as_of"] != "2026-11-19T09:00" || c["stale"] != true {
		t.Fatalf("older reader: %v", c)
	}
}

// GYST-164 c4: as_of and stale follow the reader's synced-at; a failing or
// never-synced reader reports null and stale, never a time it did not read.
func TestDayBriefCalendarAsOf(t *testing.T) {
	e := secretarytest.New(t)
	now := e.Svc.Now()
	for name, tc := range map[string]struct {
		cal            secretarytest.FakeCalendar
		asOf           any
		stale          bool
		syncedCalendar any
	}{
		"20 min old":   {secretarytest.FakeCalendar{N: 1, SyncedAt: now.Add(-20 * time.Minute)}, "2026-11-19T11:40", true, "2026-11-19T11:40"},
		"5 min old":    {secretarytest.FakeCalendar{N: 1, SyncedAt: now.Add(-5 * time.Minute)}, "2026-11-19T11:55", false, "2026-11-19T11:55"},
		"never synced": {secretarytest.FakeCalendar{N: 1}, nil, true, nil},
		"read fails":   {secretarytest.FakeCalendar{Err: context.Canceled, SyncedAt: now}, nil, true, nil},
	} {
		cat := briefCatalogue(t, e, secretarytest.FakeHealth{}, tc.cal)
		_, m := build(t, cat, "today")
		c := m["calendar"].(map[string]any)
		if c["as_of"] != tc.asOf || c["stale"] != tc.stale || c["connected"] != true || m["sync"].(map[string]any)["calendar"] != tc.syncedCalendar {
			t.Errorf("%s: %v / sync %v", name, c, m["sync"])
		}
	}
}

// #188: with a stale calendar Syncer, a stale Hevy copy and a calendar reader,
// building the brief starts no sync, no fetch and writes nothing.
func TestBriefNeverSyncsCalendar(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-5 * time.Hour))
	fake := &secretarytest.FakeHevy{}
	e.ConnectHevy(fake)
	var syncs atomic.Int32
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: secretarytest.FakeHealth{}, Budget: time.Second,
		Calendar: secretarytest.FakeCalendar{N: 2, SyncedAt: e.Svc.Now().Add(-5 * time.Hour)},
		Syncers: map[string]secretary.Syncer{secretary.SourceCalendar: {
			LastSync: func() (time.Time, error) { return e.Svc.Now().Add(-5 * time.Hour), nil },
			Sync:     func(context.Context) error { syncs.Add(1); return nil },
		}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cat.Wait)
	before := secretarytest.DumpTables(t, e.DB)
	build(t, cat, "today")
	cat.Wait()
	if syncs.Load() != 0 || fake.Fetches() != 0 || len(secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB))) != 0 {
		t.Fatalf("%d calendar syncs, %d fetches, or tables changed", syncs.Load(), fake.Fetches())
	}
}

// estimatedHealth is FakeHealth whose food total includes AI photo estimates
// (GYST-180). Fictional numbers.
type estimatedHealth struct{ secretarytest.FakeHealth }

func (f estimatedHealth) BriefHealth() (secretary.HealthBrief, error) {
	b, err := f.FakeHealth.BriefHealth()
	items, kcal, protein := 2, "410", "21"
	b.FoodAIEstimatedItems, b.FoodAIEstimatedKcal, b.FoodAIEstimatedProteinG = &items, &kcal, &protein
	return b, err
}

// food.ai_estimated is present only with estimated items, holds exactly these
// aggregate keys, and changes nothing else in food.
func TestDayBriefFoodAIEstimatedKeys(t *testing.T) {
	e := secretarytest.New(t)
	plain := briefCatalogue(t, e, secretarytest.FakeHealth{}, nil)
	_, m := build(t, plain, "today")
	food := m["health"].(map[string]any)["food"].(map[string]any)
	if _, ok := food["ai_estimated"]; ok {
		t.Fatalf("ai_estimated without estimates: %v", food)
	}
	base := len(food)
	est, err := secretary.New(secretary.Deps{Service: e.Svc, Health: estimatedHealth{}, Budget: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(est.Wait)
	raw, m := build(t, est, "today")
	food = m["health"].(map[string]any)["food"].(map[string]any)
	ai, ok := food["ai_estimated"].(map[string]any)
	if !ok || len(food) != base+1 {
		t.Fatalf("food: %v", food)
	}
	if len(ai) != 4 || ai["items"] != float64(2) || ai["kcal"] != "410" || ai["protein_g"] != "21" || ai["note"] == "" {
		t.Fatalf("ai_estimated: %v", ai)
	}
	if len(raw) > secretary.BriefBudget {
		t.Fatalf("brief %d bytes over budget %d", len(raw), secretary.BriefBudget)
	}
}
