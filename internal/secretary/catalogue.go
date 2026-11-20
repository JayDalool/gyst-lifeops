package secretary

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// allowlist is every tool the Secretary may offer, in offer order. A tool that
// is not named here cannot be registered, however a builder is wired, so adding
// a tool is always a visible, reviewed change to this list and to the literal
// in TestRegistryEqualsAllowlist.
var allowlist = []string{
	"get_health_today", "get_health_plan", "get_weight_trend", "get_training_summary", "get_recent_workouts",
	"get_exercise_progress", "get_hevy_routines", "get_food_products", "get_daily_checklist",
	"get_day_brief", "list_tasks", "list_projects", "get_prayer_times", "get_courses",
}

// builders produce the specs. A later chunk adds its builder here (see doc.go).
var builders = []func(*Catalogue) []ToolSpec{healthBuilder, coreBuilder}

// hevyBacked are the Health tools that read the Hevy copy. On the Secretary's
// path (and only there) they declare SourceHevy, so a direct workout question
// gets the same freshness check as the brief, and they are wrapped to:
//   - say "unavailable" until a snapshot exists, instead of zeros and empty
//     lists that would read as observed (RULES 3.2);
//   - take as_of and stale from the snapshot that supplied the rows, not from
//     the turn's cached "do not retry" decision, which can predate a refresh
//     that finished late;
//   - clean and cap the workout, routine and exercise names (untrusted text).
//
// The Coach takes HealthTools instead, which has none of this and is unchanged.
var hevyBacked = map[string]bool{"get_training_summary": true, "get_recent_workouts": true, "get_exercise_progress": true, "get_hevy_routines": true}

var hevyValidators = map[string]func(json.RawMessage) error{
	"get_training_summary":  validateObject,
	"get_hevy_routines":     validateObject,
	"get_recent_workouts":   validateCount,
	"get_exercise_progress": validateWeeks,
}

func healthBuilder(c *Catalogue) []ToolSpec {
	specs := healthSpecs(c.deps.Health)
	for i := range specs {
		if !hevyBacked[specs[i].Name] {
			continue
		}
		inner := specs[i].Run
		specs[i].Sources = []string{SourceHevy}
		// Without a Validate, bad input would reach EnsureFresh and could start
		// a Hevy fetch (GYST-164). These set it on the Catalogue's copy only.
		specs[i].Validate = hevyValidators[specs[i].Name]
		specs[i].Run = func(ctx context.Context, input json.RawMessage) (any, error) {
			return c.hevyAnswer(ctx, input, inner)
		}
	}
	return specs
}

// hevyAnswer runs a Hevy-backed tool and decorates its result.
func (c *Catalogue) hevyAnswer(ctx context.Context, input json.RawMessage, inner func(context.Context, json.RawMessage) (any, error)) (any, error) {
	svc := c.deps.Service
	var cache models.TrainingCache
	var v any
	var err error
	// The rows and the timestamp must come from one snapshot: if a refresh
	// lands between reading the stamp and the rows, read again.
	for try := 0; try < 3; try++ {
		before, cerr := svc.Repo.TrainingCache("hevy")
		if cerr != nil {
			return nil, errUnavailable
		}
		if v, err = inner(ctx, input); err != nil {
			return nil, err
		}
		after, cerr := svc.Repo.TrainingCache("hevy")
		if cerr != nil {
			return nil, errUnavailable
		}
		if before.SyncedAt.Equal(after.SyncedAt) {
			cache = after
			break
		}
		cache = after
	}
	if cache.SyncedAt.IsZero() {
		return map[string]any{"available": false, "reason": "no_hevy_data", "as_of": nil, "stale": true}, nil
	}
	m, err := asObject(v)
	if err != nil {
		return nil, errUnavailable
	}
	m["as_of"] = cache.SyncedAt.In(svc.Location()).Format(minuteLayout)
	m["stale"] = svc.Now().Sub(cache.SyncedAt) >= freshLimit[SourceHevy]
	cleanNames(m)
	return m, nil
}

// asObject turns a reader's result into a fresh map[string]any by way of its
// JSON form, so cleaning works on what the model will see and not on the
// adapter's Go types (GYST-164 c5). Numbers stay byte-exact (UseNumber); the
// reader's own value is never changed. A result that is not a JSON object is an
// error: the answer needs as_of and stale, and raw text would skip the cleaning.
func asObject(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	m, ok := out.(map[string]any)
	if !ok {
		return nil, errors.New("not an object")
	}
	return m, nil
}

// cleanNames cleans and caps the free-text names in a Hevy-backed result, at any
// depth and whatever Go type produced it: values under "title" (TitleCap), under
// "name" and "exercise" and the strings of an "exercises" list (NameCap). Every
// other string only loses its hidden characters, as a backstop for a field the
// key list does not know.
func cleanNames(m map[string]any) {
	for k, v := range m {
		m[k] = cleanValue(k, v)
	}
}

// noCap is "no cap" for Clean: no real string is this long.
const noCap = 1 << 30

func cleanValue(key string, v any) any {
	switch x := v.(type) {
	case string:
		switch key {
		case "title":
			t, _ := Clean(x, TitleCap)
			return t
		case "name", "exercise", "exercises":
			t, _ := Clean(x, NameCap)
			return t
		}
		t, _ := Clean(x, noCap)
		return t
	case map[string]any:
		for k, val := range x {
			x[k] = cleanValue(k, val)
		}
		return x
	case []any:
		for i, val := range x {
			x[i] = cleanValue(key, val) // a list keeps its key: "exercises": ["Squat", ...]
		}
		return x
	}
	return v
}

// Deps is what a Catalogue reads. Every field is read-only data access.
type Deps struct {
	Service *services.Service
	// Health is required: the Health tools and the brief's health section read it.
	Health HealthReader
	// Calendar is filled by the calendar chunks; nil means connected:false.
	Calendar CalendarReader
	// Syncers fills the freshness slots for sources beyond Hevy ("calendar",
	// "mail"). Empty slots report connected:false and never sync.
	Syncers map[string]Syncer
	// Budget is the freshness budget per EnsureFresh call; 0 means WebBudget.
	Budget time.Duration
}

// Catalogue is one turn's set of tools. Build one per turn: freshness is
// checked once per source per Catalogue.
type Catalogue struct {
	deps  Deps
	fresh *Freshener
	specs []ToolSpec
}

// New builds a Catalogue and checks it against the allowlist.
func New(d Deps) (*Catalogue, error) {
	if d.Service == nil || d.Health == nil {
		return nil, errors.New("secretary: Service and Health are required")
	}
	slots := map[string]Syncer{SourceHevy: HevySyncer(d.Service)}
	for name, s := range d.Syncers {
		if !validSource(name) {
			return nil, fmt.Errorf("secretary: unknown freshness source %q", name)
		}
		slots[name] = s
	}
	c := &Catalogue{deps: d}
	c.fresh = NewFreshener(slots, d.Budget, d.Service.Now)
	var all []ToolSpec
	for _, build := range builders {
		all = append(all, build(c)...)
	}
	specs, err := register(allowlist, all)
	if err != nil {
		return nil, err
	}
	c.specs = specs
	return c, nil
}

// register checks every spec and returns them in allowlist order.
func register(allow []string, specs []ToolSpec) ([]ToolSpec, error) {
	seen := map[string]bool{}
	for _, s := range specs {
		switch {
		case !slices.Contains(allow, s.Name):
			return nil, fmt.Errorf("secretary: tool %q is not on the allowlist", s.Name)
		case seen[s.Name]:
			return nil, fmt.Errorf("secretary: tool %q registered twice", s.Name)
		case s.Description == "":
			return nil, fmt.Errorf("secretary: tool %q has no description", s.Name)
		case s.Run == nil:
			return nil, fmt.Errorf("secretary: tool %q has no Run", s.Name)
		}
		for _, src := range s.Sources {
			if !validSource(src) {
				return nil, fmt.Errorf("secretary: tool %q names unknown source %q", s.Name, src)
			}
		}
		// A tool with Sources is refreshed before it runs, so unchecked input
		// could start a sync (GYST-164). Every such tool must validate.
		if len(s.Sources) > 0 && s.Validate == nil {
			return nil, fmt.Errorf("secretary: tool %q has Sources but no Validate", s.Name)
		}
		seen[s.Name] = true
	}
	out := make([]ToolSpec, 0, len(specs))
	for _, name := range allow {
		for _, s := range specs {
			if s.Name == name {
				out = append(out, s)
			}
		}
	}
	return out, nil
}

// Tools are the tools for one turn, in allowlist order. Each one first checks
// the freshness of the sources it reads (which can start a sync), so Tools is
// for POST paths only: a GET must never call it (Issue #188). Errors the model
// sees are fixed strings.
func (c *Catalogue) Tools() []healthcoach.Tool {
	before := func(ctx context.Context, sources []string) { c.fresh.EnsureFresh(ctx, sources...) }
	out := make([]healthcoach.Tool, 0, len(c.specs))
	for _, s := range c.specs {
		out = append(out, s.tool(before))
	}
	return out
}

// Freshness is the turn's freshness checker; BuildDayBrief reads it with Peek.
func (c *Catalogue) Freshness() *Freshener { return c.fresh }

// Wait blocks until background syncs this Catalogue started have finished.
func (c *Catalogue) Wait() { c.fresh.Wait() }

// Names lists the registered tools, in offer order.
func (c *Catalogue) Names() []string {
	out := make([]string, 0, len(c.specs))
	for _, s := range c.specs {
		out = append(out, s.Name)
	}
	return out
}

// Clock is the application clock the Catalogue reads dates from.
func (c *Catalogue) Clock() clock.Clock { return c.deps.Service.Clock }
