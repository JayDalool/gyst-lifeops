package secretary_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/secretary"
	"lifeops-tracker/internal/secretary/secretarytest"
)

func catalogue(t *testing.T, e *secretarytest.Env) *secretary.Catalogue {
	t.Helper()
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: secretarytest.FakeHealth{}, Budget: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cat.Wait)
	return cat
}

func run(t *testing.T, cat *secretary.Catalogue, name, in string) map[string]any {
	t.Helper()
	out, err := secretarytest.Run(t, cat, name, in)
	if err != nil {
		t.Fatalf("%s %s: %v", name, in, err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%s: not JSON: %v: %s", name, err, out)
	}
	return m
}

func noSentinels(t *testing.T, out string) {
	t.Helper()
	for _, s := range secretarytest.Sentinels {
		if strings.Contains(out, s) {
			t.Errorf("output holds %s: %s", s, out)
		}
	}
}

func titles(m map[string]any, key, field string) []string {
	var out []string
	for _, it := range m[key].([]any) {
		out = append(out, it.(map[string]any)[field].(string))
	}
	return out
}

// The tools offered are exactly this list, in this order. Adding a tool means
// changing this literal and the allowlist in catalogue.go, which a reviewer sees.
func TestRegistryEqualsAllowlist(t *testing.T) {
	want := []string{
		"get_health_today", "get_health_plan", "get_weight_trend", "get_training_summary", "get_recent_workouts",
		"get_exercise_progress", "get_hevy_routines", "get_food_products", "get_daily_checklist",
		"get_day_brief", "list_tasks", "list_projects", "get_prayer_times", "get_courses",
	}
	cat := catalogue(t, secretarytest.New(t))
	var got []string
	for _, tool := range cat.Tools() {
		got = append(got, tool.Name)
		if tool.Description == "" || tool.Run == nil {
			t.Errorf("%s incomplete", tool.Name)
		}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") || strings.Join(cat.Names(), ",") != strings.Join(want, ",") {
		t.Fatalf("registry %v", got)
	}
}

// One source: the Catalogue's Health tools are HealthTools, name for name,
// schema for schema and output for output.
func TestCatalogueHealthToolsEqualHealthTools(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-5 * time.Minute)) // a snapshot exists, so the Hevy tools answer
	fake := secretarytest.FakeHealth{}
	direct := map[string]healthcoach.Tool{}
	for _, tool := range secretary.HealthTools(fake) {
		direct[tool.Name] = tool
	}
	n := 0
	for _, tool := range catalogue(t, e).Tools() {
		d, ok := direct[tool.Name]
		if !ok {
			continue
		}
		n++
		a, _ := json.Marshal([]any{tool.Description, tool.Properties, tool.Required})
		b, _ := json.Marshal([]any{d.Description, d.Properties, d.Required})
		if string(a) != string(b) {
			t.Errorf("%s: schema differs", tool.Name)
		}
		for _, in := range []string{`{}`, `{"count":3}`, `{"weeks":26}`} {
			x, xerr := tool.Run(context.Background(), json.RawMessage(in))
			x = withoutFreshness(x)
			y, yerr := d.Run(context.Background(), json.RawMessage(in))
			if x != y || (xerr == nil) != (yerr == nil) {
				t.Errorf("%s %s: %q vs %q", tool.Name, in, x, y)
			}
		}
	}
	if n != 9 {
		t.Fatalf("%d Health tools in the Catalogue", n)
	}
	// GYST-164: out of range is an error on the Secretary path (the model can
	// retry) and a clamp on the Coach path (unchanged).
	sec, err := secretarytest.Run(t, catalogue(t, e), "get_recent_workouts", `{"count":99}`)
	if err == nil || err.Error() != "invalid input" || sec != "" {
		t.Errorf("Secretary count 99: %q %v", sec, err)
	}
	if out, err := direct["get_recent_workouts"].Run(context.Background(), json.RawMessage(`{"count":99}`)); err != nil || out == "" {
		t.Errorf("Coach count 99: %q %v", out, err)
	}
}

var edgeInputs = []string{``, `null`, `{}`, `{`, `[]`, `{"scope":"late"}`, `{"scope":"today","limit":3}`, `{"scope":"week","limit":99}`, `{"scope":"no_date","limit":-1}`, `{"scope":"nope"}`,
	`{"state":"all"}`, `{"state":"done","limit":1}`, `{"state":"x"}`, `{"date":"tomorrow"}`, `{"date":"next friday"}`, `{"date":"2026-11-25"}`, `{"date":"1999-01-01"}`, `{"date":"garbage"}`, `{"count":99}`, `{"weeks":26}`}

func allToolNames(cat *secretary.Catalogue) []string { return cat.Names() }

// Every tool is read-only: with Hevy fresh, nothing in any table changes after
// valid, edge and invalid input.
func TestEveryToolIsReadOnly(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedTasks()
	e.SeedProjects()
	e.SeedCourses()
	e.SeedHevy(e.Svc.Now().Add(-5 * time.Minute))
	fake := &secretarytest.FakeHevy{}
	e.ConnectHevy(fake)
	cat := catalogue(t, e)
	before := secretarytest.DumpTables(t, e.DB)
	for _, name := range allToolNames(cat) {
		for _, in := range edgeInputs {
			_, _ = secretarytest.Run(t, cat, name, in)
		}
	}
	cat.Wait()
	if changed := secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB)); len(changed) != 0 || fake.Fetches() != 0 {
		t.Fatalf("tables changed: %v (%d fetches)", changed, fake.Fetches())
	}
}

// The one side effect a tool may have is a Hevy refresh, and it writes only
// training_snapshots. get_day_brief and the four Hevy-backed Health tools name
// the Hevy source, so running them is what starts it.
func TestFreshSyncWritesOnlyTrainingSnapshots(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedTasks()
	e.SeedProjects()
	e.SeedCourses()
	e.SeedHevy(e.Svc.Now().Add(-3 * time.Hour))
	fake := &secretarytest.FakeHevy{Entries: nil}
	e.ConnectHevy(fake)
	cat := catalogue(t, e)
	before := secretarytest.DumpTables(t, e.DB)
	run(t, cat, "get_day_brief", `{}`)
	cat.Wait()
	changed := secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB))
	if fake.Fetches() != 1 || len(changed) != 1 || changed[0] != "training_snapshots" {
		t.Fatalf("fetches=%d changed=%v", fake.Fetches(), changed)
	}
}

// On the Secretary's path the four Hevy-backed Health tools check freshness and
// say how old their data is; the other five do not. The Coach's HealthTools
// stay free of both (covered by the golden test in internal/handlers).
func TestHevyBackedHealthToolsCarryFreshness(t *testing.T) {
	e := secretarytest.New(t)
	stored := e.Svc.Now().Add(-3 * time.Hour)
	e.SeedHevy(stored)
	fake := &secretarytest.FakeHevy{}
	e.ConnectHevy(fake)
	cat := catalogue(t, e)
	m := run(t, cat, "get_recent_workouts", `{"count":2}`)
	cat.Wait()
	if fake.Fetches() != 1 || m["stale"] != false || m["as_of"] != "2026-11-19T12:00" || m["count"] != 2.0 {
		t.Fatalf("fetches %d: %v", fake.Fetches(), m)
	}
	for _, name := range []string{"get_training_summary", "get_exercise_progress", "get_hevy_routines"} {
		if m := run(t, cat, name, `{}`); m["as_of"] == nil || m["stale"] == nil {
			t.Errorf("%s lacks as_of/stale: %v", name, m)
		}
	}
	for _, name := range []string{"get_health_today", "get_health_plan", "get_weight_trend", "get_food_products", "get_daily_checklist"} {
		if m := run(t, cat, name, `{}`); m["as_of"] != nil || m["stale"] != nil {
			t.Errorf("%s has freshness fields: %v", name, m)
		}
	}
	for _, tool := range secretary.HealthTools(secretarytest.FakeHealth{}) {
		out, _ := tool.Run(context.Background(), nil)
		if strings.Contains(out, "as_of") || strings.Contains(out, "stale") {
			t.Errorf("Coach tool %s carries freshness: %s", tool.Name, out)
		}
	}
}

func TestListTasksScopes(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedTasks()
	cat := catalogue(t, e)

	today := run(t, cat, "list_tasks", ``)
	if today["scope"] != "today" || today["date"] != secretarytest.Today || today["weekday"] != "Thursday" {
		t.Fatalf("defaults: %v", today)
	}
	// The daily task and the report are due today; the paid bill is done.
	if got := titles(today, "tasks", "title"); strings.Join(got, "|") != "Draft the <b>report</b> & send|Water the plants" || today["total"] != 2.0 || today["more"] != 0.0 {
		t.Fatalf("today: %v", got)
	}
	first := today["tasks"].([]any)[1].(map[string]any)
	if first["time"] != "09:00" || first["repeat"] != "daily" || first["due"] != secretarytest.Today || first["status"] != "pending" || first["priority"] != "medium" {
		t.Fatalf("task fields: %v", first)
	}

	late := run(t, cat, "list_tasks", `{"scope":"late"}`)
	if got := titles(late, "tasks", "title"); len(got) != 1 || got[0] != "Renew library card" || late["tasks"].([]any)[0].(map[string]any)["due"] != "2026-11-17" {
		t.Fatalf("late: %v", late)
	}
	week := run(t, cat, "list_tasks", `{"scope":"week","limit":3}`)
	// Seven days from Thursday: the daily task each day, plus the one-offs.
	if week["total"].(float64) < 8 || len(week["tasks"].([]any)) != 3 || week["more"].(float64) != week["total"].(float64)-3 {
		t.Fatalf("week limit/more: total %v more %v", week["total"], week["more"])
	}
	nodate := run(t, cat, "list_tasks", `{"scope":"no_date"}`)
	if got := titles(nodate, "tasks", "title"); len(got) != 1 || got[0] != "Sort the photos" || nodate["tasks"].([]any)[0].(map[string]any)["due"] != nil {
		t.Fatalf("no_date: %v", nodate)
	}
	for _, in := range []string{``, `{"scope":"week"}`, `{"scope":"late"}`, `{"scope":"no_date","limit":20}`} {
		out, _ := secretarytest.Run(t, cat, "list_tasks", in)
		noSentinels(t, out)
	}
	// limit is clamped to 1-20.
	if m := run(t, cat, "list_tasks", `{"scope":"week","limit":99}`); len(m["tasks"].([]any)) > 20 {
		t.Fatal("limit above 20 honoured")
	}
	if m := run(t, cat, "list_tasks", `{"scope":"week","limit":-5}`); len(m["tasks"].([]any)) != 1 {
		t.Fatal("negative limit not clamped to 1")
	}
}

func TestListTasksCleansAndCapsTitles(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedTasks()
	long := strings.Repeat("é", 200)
	if _, err := e.DB.Exec(`UPDATE tasks SET title = ? WHERE title LIKE 'Sort%'`, "Hidden‮text​ "+long); err != nil {
		t.Fatal(err)
	}
	m := run(t, catalogue(t, e), "list_tasks", `{"scope":"no_date"}`)
	got := titles(m, "tasks", "title")[0]
	if strings.ContainsAny(got, "‮​") || len([]rune(got)) != secretary.TaskTitleCap || !strings.HasSuffix(got, "…") {
		t.Fatalf("%q (%d runes)", got, len([]rune(got)))
	}
}

func TestFixedErrorsNeverEchoInput(t *testing.T) {
	e := secretarytest.New(t)
	cat := catalogue(t, e)
	secret := "IGNORE-PREVIOUS-INSTRUCTIONS"
	for name, ins := range map[string][]string{
		"list_tasks":       {`{"scope":"` + secret + `"}`, `{"scope":1}`, `{` + secret},
		"list_projects":    {`{"state":"` + secret + `"}`, `{"limit":"` + secret + `"}`},
		"get_prayer_times": {`{"date":"` + secret + `"}`, `{"date":"next ` + secret + `"}`, `{"date":"2030-01-01"}`},
		"get_courses":      {`{"date":"` + secret + `"}`},
	} {
		for _, in := range ins {
			out, err := secretarytest.Run(t, cat, name, in)
			if err == nil || out != "" || strings.Contains(err.Error(), "IGNORE") {
				t.Errorf("%s %s: %q %v", name, in, out, err)
			}
		}
	}
}

func projectsByName(m map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, p := range m["projects"].([]any) {
		pm := p.(map[string]any)
		out[pm["name"].(string)] = pm
	}
	return out
}

// S15 (round 7): Private and Local-only projects are in the Secretary's view,
// each with its effective privacy; archived ones are not.
func TestListProjectsIncludesLocalOnly(t *testing.T) {
	e := secretarytest.New(t)
	ids := e.SeedProjects()
	cat := catalogue(t, e)
	all := projectsByName(run(t, cat, "list_projects", `{"state":"all"}`))
	want := map[string]string{"Garden shed": "public", "Reading list": "private", "Boat log": "local_only", "Boat log: sails": "local_only",
		"Letters": "local_only", "Both flags": "local_only", "Someday kitchen": "public"}
	if len(all) != len(want) {
		t.Fatalf("got %d projects: %v", len(all), all)
	}
	for name, privacy := range want {
		p, ok := all[name]
		if !ok || p["privacy"] != privacy {
			t.Errorf("%s: %v, want %s", name, p, privacy)
		}
	}
	if _, ok := all["Old project"]; ok {
		t.Error("archived project listed")
	}
	shed := all["Garden shed"]
	if shed["open_items"] != 2.0 || shed["state"] != "active" || shed["parent_id"] != nil || shed["target_date"] != "2026-12-31" || shed["id"] != float64(ids.Public) {
		t.Fatalf("shed: %v", shed)
	}
	if all["Boat log: sails"]["parent_id"] != float64(ids.OwnLocal) {
		t.Fatalf("child: %v", all["Boat log: sails"])
	}
	if all["Someday kitchen"]["state"] != "planned" {
		t.Fatalf("planned: %v", all["Someday kitchen"])
	}
	// Default state is active: not the planned one.
	def := projectsByName(run(t, cat, "list_projects", ``))
	if _, ok := def["Someday kitchen"]; ok || len(def) != 6 {
		t.Fatalf("default state: %v", def)
	}
	if got := projectsByName(run(t, cat, "list_projects", `{"state":"planned"}`)); len(got) != 1 {
		t.Fatalf("planned: %v", got)
	}
	limited := run(t, cat, "list_projects", `{"state":"all","limit":2}`)
	if len(limited["projects"].([]any)) != 2 || limited["more"] != 5.0 {
		t.Fatalf("limit/more: %v", limited)
	}
	for _, in := range []string{``, `{"state":"all"}`} {
		out, _ := secretarytest.Run(t, cat, "list_projects", in)
		noSentinels(t, out)
		for _, banned := range []string{`"description"`, `"notes"`, `"unlock"`, `"summary"`, `"next"`} {
			if strings.Contains(out, banned) {
				t.Errorf("output has %s", banned)
			}
		}
	}
}

func TestGetPrayerTimes(t *testing.T) {
	e := secretarytest.New(t)
	cat := catalogue(t, e)
	m := run(t, cat, "get_prayer_times", ``)
	times := m["times"].([]any)
	if m["date"] != secretarytest.Today || m["weekday"] != "Thursday" || len(times) != 5 {
		t.Fatalf("%v", m)
	}
	last := ""
	for _, tm := range times {
		hhmm := tm.(map[string]any)["time"].(string)
		if len(hhmm) != 5 || hhmm[2] != ':' || hhmm <= last {
			t.Fatalf("time %q after %q", hhmm, last)
		}
		last = hhmm
	}
	next := m["next"].(map[string]any)
	if next["date"] != secretarytest.Today { // noon: the next prayer is later today
		t.Fatalf("next: %v", next)
	}
	// After the last prayer the next one is tomorrow's first, with its date.
	e.SetNow(time.Date(2026, 11, 19, 23, 50, 0, 0, e.Svc.Location()))
	m = run(t, catalogue(t, e), "get_prayer_times", `{"date":"today"}`)
	if n := m["next"].(map[string]any); n["date"] != "2026-11-20" || n["name"] != times[0].(map[string]any)["name"] {
		t.Fatalf("after the last prayer: %v", n)
	}
	// Another day: named date and weekday, no next.
	m = run(t, cat, "get_prayer_times", `{"date":"2026-11-23"}`)
	if m["date"] != "2026-11-23" || m["weekday"] != "Monday" || m["next"] != nil || len(m["times"].([]any)) != 5 {
		t.Fatalf("%v", m)
	}
	out, _ := secretarytest.Run(t, cat, "get_prayer_times", ``)
	for _, banned := range []string{"latitude", "longitude", "Winnipeg", "location", "place"} {
		if strings.Contains(strings.ToLower(out), strings.ToLower(banned)) {
			t.Errorf("output has %q", banned)
		}
	}
}

func TestGetCourses(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedCourses()
	cat := catalogue(t, e)
	m := run(t, cat, "get_courses", ``)
	blocks := m["blocks"].([]any)
	// The skipped block is not scheduled; the overdue one belongs to another day.
	if m["date"] != secretarytest.Today || m["weekday"] != "Thursday" || len(blocks) != 2 || m["overdue"] != 1.0 {
		t.Fatalf("%v", m)
	}
	timed := blocks[1].(map[string]any)
	if timed["title"] != "Wheel basics" || timed["start"] != "18:30" || timed["end"] != "19:30" || timed["course"] != "Intro to Pottery" || timed["type"] != "lesson" || timed["status"] != "pending" || timed["catch_up"] != false {
		t.Fatalf("timed block: %v", timed)
	}
	untimed := blocks[0].(map[string]any)
	if untimed["start"] != nil || untimed["end"] != nil || untimed["catch_up"] != true {
		t.Fatalf("untimed block: %v", untimed)
	}
	tomorrow := run(t, cat, "get_courses", `{"date":"tomorrow"}`)
	if tomorrow["overdue"] != nil || tomorrow["weekday"] != "Friday" || len(tomorrow["blocks"].([]any)) != 1 {
		t.Fatalf("%v", tomorrow)
	}
	for _, in := range []string{``, `{"date":"tomorrow"}`} {
		out, _ := secretarytest.Run(t, cat, "get_courses", in)
		noSentinels(t, out)
		for _, banned := range []string{"notes", "url", "http"} {
			if strings.Contains(out, banned) {
				t.Errorf("output has %q: %s", banned, out)
			}
		}
	}
}

// withoutFreshness removes the as_of and stale keys the Secretary path adds to
// the Hevy-backed tools, so the rest can be compared with the Coach's output.
func withoutFreshness(out string) string {
	var m map[string]any
	if json.Unmarshal([]byte(out), &m) != nil {
		return out
	}
	delete(m, "as_of")
	delete(m, "stale")
	b, _ := json.Marshal(m)
	return string(b)
}

// Before any Hevy snapshot the four Hevy-backed tools say "unavailable" with
// null freshness; a synced, empty copy answers normally (a real zero).
func TestHevyToolsUnavailableBeforeFirstSync(t *testing.T) {
	e := secretarytest.New(t)
	cat := catalogue(t, e)
	names := []string{"get_training_summary", "get_recent_workouts", "get_exercise_progress", "get_hevy_routines"}
	for _, name := range names {
		m := run(t, cat, name, `{}`)
		if m["available"] != false || m["reason"] != "no_hevy_data" || m["stale"] != true {
			t.Errorf("%s: %v", name, m)
		}
		if v, has := m["as_of"]; !has || v != nil {
			t.Errorf("%s as_of: %v", name, m)
		}
		if len(m) != 4 {
			t.Errorf("%s shows more than the unavailable marker: %v", name, m)
		}
	}
	if err := e.Svc.Repo.SaveTraining("hevy", nil, e.Svc.Now().Add(-5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		m := run(t, cat, name, `{}`)
		if _, marker := m["available"]; marker || m["stale"] != false || m["as_of"] != "2026-11-19T11:55" {
			t.Errorf("%s with a synced empty copy: %v", name, m)
		}
	}
}

// A refresh that outlives the budget and finishes before a later call in the
// same turn: that call's as_of and stale come from the new snapshot, not from
// the turn's cached decision.
func TestHevyToolFreshnessFollowsTheSnapshotRead(t *testing.T) {
	e := secretarytest.New(t)
	stored := e.Svc.Now().Add(-2 * time.Hour)
	e.SeedHevy(stored)
	slow := secretarytest.SlowHevy(false)
	slow.Entries = nil
	e.ConnectHevy(slow)
	t.Cleanup(slow.Release)
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: secretarytest.FakeHealth{}, Budget: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cat.Wait)
	first := run(t, cat, "get_recent_workouts", `{}`)
	if first["stale"] != true || first["as_of"] != "2026-11-19T10:00" {
		t.Fatalf("first call: %v", first)
	}
	slow.Release()
	cat.Wait()
	second := run(t, cat, "get_hevy_routines", `{}`)
	if second["stale"] != false || second["as_of"] != "2026-11-19T12:00" {
		t.Fatalf("second call in the same turn: %v", second)
	}
	day, _ := secretary.ResolveDate("today", cat.Clock())
	raw, _ := cat.BuildDayBrief(day)
	var b map[string]map[string]any
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	if b["training"]["as_of"] != "2026-11-19T12:00" || b["training"]["stale"] != false || b["sync"]["hevy"] != "2026-11-19T12:00" {
		t.Fatalf("brief: %v / %v", b["training"], b["sync"])
	}
}

// On the Secretary path, workout, routine and exercise names are cleaned and
// capped; the reader's own values are not changed.
func TestHevyToolsCleanNames(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-5 * time.Minute))
	dirty := "Push\u202E day\u200B " + strings.Repeat("x", 200)
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: dirtyHealth{dirty}, Budget: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cat.Wait)
	w := run(t, cat, "get_recent_workouts", `{}`)["workouts"].([]any)[0].(map[string]any)
	title, ex := w["title"].(string), w["exercises"].([]any)[0].(map[string]any)["name"].(string)
	r := run(t, cat, "get_hevy_routines", `{}`)["routines"].([]any)[0].(map[string]any)
	prog := run(t, cat, "get_exercise_progress", `{}`)["exercises"].([]any)[0].(map[string]any)["exercise"].(string)
	for label, got := range map[string]string{"workout title": title, "routine title": r["title"].(string)} {
		if strings.ContainsAny(got, "\u202E\u200B") || utf8.RuneCountInString(got) != secretary.TitleCap || !strings.HasSuffix(got, "…") {
			t.Errorf("%s %q (%d runes)", label, got, utf8.RuneCountInString(got))
		}
	}
	for label, got := range map[string]string{"exercise": ex, "routine exercise": r["exercises"].([]any)[0].(string), "progress": prog} {
		if strings.ContainsAny(got, "\u202E\u200B") || utf8.RuneCountInString(got) != secretary.NameCap {
			t.Errorf("%s %q (%d runes)", label, got, utf8.RuneCountInString(got))
		}
	}
	// The Coach's tool, over the same reader, is untouched.
	for _, tool := range secretary.HealthTools(dirtyHealth{dirty}) {
		if tool.Name == "get_recent_workouts" {
			out, _ := tool.Run(context.Background(), nil)
			if !strings.Contains(out, "\\u202e") && !strings.Contains(out, "\u202e") {
				t.Errorf("Coach output was cleaned: %s", out)
			}
		}
	}
}

// dirtyHealth answers the Hevy tools with one hidden-character, over-long name.
type dirtyHealth struct{ name string }

func (d dirtyHealth) Today() (any, error)       { return nil, nil }
func (d dirtyHealth) Plan() (any, error)        { return nil, nil }
func (d dirtyHealth) WeightTrend() (any, error) { return nil, nil }
func (d dirtyHealth) TrainingSummary() (any, error) {
	return map[string]any{"this_week_workouts": 0}, nil
}
func (d dirtyHealth) RecentWorkouts(int) (any, error) {
	return map[string]any{"workouts": []map[string]any{{"title": d.name, "exercises": []map[string]any{{"name": d.name}}}}}, nil
}
func (d dirtyHealth) ExerciseProgress(int) (any, error) {
	return map[string]any{"exercises": []map[string]any{{"exercise": d.name}}}, nil
}
func (d dirtyHealth) HevyRoutines() (any, error) {
	return map[string]any{"routines": []map[string]any{{"title": d.name, "exercises": []string{d.name}}}}, nil
}
func (d dirtyHealth) FoodProducts() (any, error)   { return nil, nil }
func (d dirtyHealth) DailyChecklist() (any, error) { return nil, nil }
func (d dirtyHealth) BriefHealth() (secretary.HealthBrief, error) {
	return secretary.HealthBrief{}, nil
}

// GYST-164 c1: input is checked before any refresh. Each invalid input gets the
// fixed error, starts no Hevy fetch, writes nothing and never appears in the
// error. A fresh Catalogue per input: freshness is decided once per turn.
func TestHevyToolsValidateBeforeRefreshing(t *testing.T) {
	invalid := map[string][]string{
		"get_recent_workouts":   {`{`, `[1]`, `"x"`, `{"count":"three"}`, `{"count":2.5}`, `{"count":3.0}`, `{"count":true}`, `{"count":0}`, `{"count":6}`, `{"count":-4}`, `{"count":99}`, `{"count":1e0}`, `{"count":[3]}`},
		"get_exercise_progress": {`{`, `[1]`, `"x"`, `{"weeks":"4"}`, `{"weeks":7}`, `{"weeks":0}`, `{"weeks":12.0}`, `{"weeks":false}`},
		"get_training_summary":  {`{`, `[1]`, `"x"`, `5`},
		"get_hevy_routines":     {`{`, `[1]`, `"x"`, `true`},
	}
	valid := map[string][]string{
		"get_recent_workouts":   {``, `null`, `{}`, `{"count":null}`, `{"count":1}`, `{"count":3}`, `{"count":5}`, `{"count":3,"extra":"ignored"}`},
		"get_exercise_progress": {``, `null`, `{}`, `{"weeks":null}`, `{"weeks":4}`, `{"weeks":12}`, `{"weeks":26}`},
		"get_training_summary":  {``, `null`, `{}`, `{"anything":1}`},
		"get_hevy_routines":     {``, `null`, `{}`},
	}
	build := func() (*secretarytest.Env, *secretarytest.FakeHevy, *secretary.Catalogue) {
		e := secretarytest.New(t)
		e.SeedHevy(e.Svc.Now().Add(-2 * time.Hour))
		fake := &secretarytest.FakeHevy{}
		e.ConnectHevy(fake)
		return e, fake, catalogue(t, e)
	}
	for name, inputs := range invalid {
		for _, in := range inputs {
			e, fake, cat := build()
			before := secretarytest.DumpTables(t, e.DB)
			out, err := secretarytest.Run(t, cat, name, in)
			cat.Wait()
			if err == nil || err.Error() != "invalid input" || out != "" {
				t.Errorf("%s %s: %q %v", name, in, out, err)
			}
			if fake.Fetches() != 0 || len(secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB))) != 0 {
				t.Errorf("%s %s: fetches %d, tables changed", name, in, fake.Fetches())
			}
		}
	}
	for name, inputs := range valid {
		for _, in := range inputs {
			e, fake, cat := build()
			out, err := secretarytest.Run(t, cat, name, in)
			cat.Wait()
			if err != nil || out == "" || fake.Fetches() != 1 {
				t.Errorf("%s %q: %q %v, fetches %d", name, in, out, err, fake.Fetches())
			}
			_ = e
		}
	}
}

// Run gets exactly the number the model sent; a default fills only an absent
// or null field, and nothing accepted is silently replaced.
func TestHevyValidatedInputReachesRunUnchanged(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-5 * time.Minute))
	h := &argHealth{}
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: h, Budget: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cat.Wait)
	for _, tc := range []struct {
		name, in string
		count    int
		weeks    int
	}{
		{"get_recent_workouts", ``, 1, 0}, {"get_recent_workouts", `{"count":null}`, 1, 0}, {"get_recent_workouts", `{"count":1}`, 1, 0},
		{"get_recent_workouts", `{"count":3}`, 3, 0}, {"get_recent_workouts", `{"count":5}`, 5, 0},
		{"get_exercise_progress", `{}`, 0, 12}, {"get_exercise_progress", `{"weeks":4}`, 0, 4}, {"get_exercise_progress", `{"weeks":26}`, 0, 26}, {"get_exercise_progress", `{"weeks":12}`, 0, 12},
	} {
		h.count, h.weeks = 0, 0
		if _, err := secretarytest.Run(t, cat, tc.name, tc.in); err != nil {
			t.Fatalf("%s %s: %v", tc.name, tc.in, err)
		}
		if h.count != tc.count || h.weeks != tc.weeks {
			t.Errorf("%s %s: Run got count %d weeks %d, want %d %d", tc.name, tc.in, h.count, h.weeks, tc.count, tc.weeks)
		}
	}
}

// argHealth records the clamped arguments the Hevy tools pass to the reader.
type argHealth struct {
	secretarytest.FakeHealth
	count, weeks int
}

func (a *argHealth) RecentWorkouts(n int) (any, error) {
	a.count = n
	return map[string]any{"workouts": []map[string]any{}}, nil
}
func (a *argHealth) ExerciseProgress(w int) (any, error) {
	a.weeks = w
	return map[string]any{"exercises": []map[string]any{}}, nil
}

// shapeHealth answers all four Hevy tools with one value, whatever Go type the
// test builds it from.
type shapeHealth struct {
	secretarytest.FakeHealth
	v any
}

func (s shapeHealth) TrainingSummary() (any, error)     { return s.v, nil }
func (s shapeHealth) RecentWorkouts(int) (any, error)   { return s.v, nil }
func (s shapeHealth) ExerciseProgress(int) (any, error) { return s.v, nil }
func (s shapeHealth) HevyRoutines() (any, error)        { return s.v, nil }

type shapeEx struct {
	Name string  `json:"name"`
	Kg   float64 `json:"kg"`
}
type shapeWorkout struct {
	Title     string    `json:"title"`
	Note      string    `json:"note"`
	Exercises []shapeEx `json:"exercises"`
}
type shapeRoutine struct {
	Title     string   `json:"title"`
	Exercises []string `json:"exercises"`
}
type shapeProgress struct {
	Exercise string `json:"exercise"`
}
type shapeResult struct {
	Workouts  []shapeWorkout  `json:"workouts"`
	Routines  []shapeRoutine  `json:"routines"`
	Exercises []shapeProgress `json:"exercises"`
}

// GYST-164 c5: names are cleaned from the JSON form, so the adapter's Go types
// do not matter. Any string anywhere loses hidden characters; titles and names
// keep their caps; a result that is not an object is "data unavailable".
func TestHevyToolsCleanNamesAnyShape(t *testing.T) {
	d := "Push‮ day​ " + strings.Repeat("x", 200)
	shapes := map[string]any{
		"maps": map[string]any{
			"workouts":  []map[string]any{{"title": d, "note": d, "exercises": []map[string]any{{"name": d, "kg": 102.5}}}},
			"routines":  []map[string]any{{"title": d, "exercises": []string{d}}},
			"exercises": []map[string]any{{"exercise": d}},
		},
		"anys": map[string]any{
			"workouts":  []any{map[string]any{"title": d, "note": d, "exercises": []any{map[string]any{"name": d, "kg": 102.5}}}},
			"routines":  []any{map[string]any{"title": d, "exercises": []any{d}}},
			"exercises": []any{map[string]any{"exercise": d}},
		},
		"tagged structs": map[string]any{
			"workouts":  []shapeWorkout{{Title: d, Note: d, Exercises: []shapeEx{{Name: d, Kg: 102.5}}}},
			"routines":  []shapeRoutine{{Title: d, Exercises: []string{d}}},
			"exercises": []shapeProgress{{Exercise: d}},
		},
		"string maps": map[string]any{
			"workouts":  []map[string]string{{"title": d, "note": d}},
			"routines":  []map[string]string{{"title": d}},
			"exercises": []map[string]string{{"exercise": d}},
		},
		"struct pointer": &shapeResult{
			Workouts:  []shapeWorkout{{Title: d, Note: d, Exercises: []shapeEx{{Name: d, Kg: 102.5}}}},
			Routines:  []shapeRoutine{{Title: d, Exercises: []string{d}}},
			Exercises: []shapeProgress{{Exercise: d}},
		},
	}
	var walk func(key string, v any, t *testing.T, shape string)
	walk = func(key string, v any, t *testing.T, shape string) {
		switch x := v.(type) {
		case string:
			if strings.ContainsAny(x, "‮​") {
				t.Errorf("%s: %s holds a hidden rune: %q", shape, key, x)
			}
			if key == "title" && utf8.RuneCountInString(x) > secretary.TitleCap {
				t.Errorf("%s: title over cap: %d runes", shape, utf8.RuneCountInString(x))
			}
			if (key == "name" || key == "exercise" || key == "exercises") && utf8.RuneCountInString(x) > secretary.NameCap {
				t.Errorf("%s: %s over cap: %d runes", shape, key, utf8.RuneCountInString(x))
			}
		case map[string]any:
			for k, val := range x {
				walk(k, val, t, shape)
			}
		case []any:
			for _, val := range x {
				walk(key, val, t, shape)
			}
		}
	}
	for shape, v := range shapes {
		e := secretarytest.New(t)
		e.SeedHevy(e.Svc.Now().Add(-5 * time.Minute))
		cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: shapeHealth{v: v}, Budget: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(cat.Wait)
		for _, name := range []string{"get_training_summary", "get_recent_workouts", "get_exercise_progress", "get_hevy_routines"} {
			out, err := secretarytest.Run(t, cat, name, `{}`)
			if err != nil {
				t.Fatalf("%s %s: %v", shape, name, err)
			}
			if strings.ContainsAny(out, "‮​") || strings.Contains(out, `‮`) || strings.Contains(out, `​`) {
				t.Errorf("%s %s: raw output holds a hidden rune", shape, name)
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(out), &m); err != nil {
				t.Fatal(err)
			}
			walk("", m, t, shape+" "+name)
			if m["as_of"] == nil || m["stale"] == nil {
				t.Errorf("%s %s: no freshness: %s", shape, name, out)
			}
			if shape != "string maps" && name == "get_recent_workouts" {
				if !strings.Contains(out, `"kg":102.5`) {
					t.Errorf("%s: number not byte-exact: %s", shape, out)
				}
			}
		}
	}
	// The reader's value is not changed by cleaning.
	src := map[string]any{"workouts": []map[string]any{{"title": d}}}
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-5 * time.Minute))
	cat, _ := secretary.New(secretary.Deps{Service: e.Svc, Health: shapeHealth{v: src}, Budget: time.Second})
	t.Cleanup(cat.Wait)
	if _, err := secretarytest.Run(t, cat, "get_recent_workouts", `{}`); err != nil {
		t.Fatal(err)
	}
	if src["workouts"].([]map[string]any)[0]["title"] != d {
		t.Error("the reader's value was changed")
	}
	// A result that is not an object is unavailable, not passed through raw.
	for _, bad := range []any{[]string{d}, "raw " + d, 7.0, nil} {
		cat, _ := secretary.New(secretary.Deps{Service: e.Svc, Health: shapeHealth{v: bad}, Budget: time.Second})
		t.Cleanup(cat.Wait)
		if out, err := secretarytest.Run(t, cat, "get_recent_workouts", `{}`); err == nil || err.Error() != "data unavailable" || out != "" {
			t.Errorf("non-object %T: %q %v", bad, out, err)
		}
	}
}
