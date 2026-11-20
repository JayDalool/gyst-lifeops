package secretary_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
		"list_tasks", "list_projects", "get_prayer_times", "get_courses",
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
		for _, in := range []string{`{}`, `{"count":99}`, `{"weeks":26}`} {
			x, xerr := tool.Run(context.Background(), json.RawMessage(in))
			y, yerr := d.Run(context.Background(), json.RawMessage(in))
			if x != y || (xerr == nil) != (yerr == nil) {
				t.Errorf("%s %s: %q vs %q", tool.Name, in, x, y)
			}
		}
	}
	if n != 9 {
		t.Fatalf("%d Health tools in the Catalogue", n)
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
// training_snapshots. No Catalogue tool names a source yet, so the check goes
// through EnsureFresh itself, which is what a tool with Sources calls.
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
	got := cat.Freshness().EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
	cat.Wait()
	changed := secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB))
	if got.Stale || fake.Fetches() != 1 || len(changed) != 1 || changed[0] != "training_snapshots" {
		t.Fatalf("stale=%v fetches=%d changed=%v", got.Stale, fake.Fetches(), changed)
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
