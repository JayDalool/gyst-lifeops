package secretary

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/healthcoach"
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

func healthBuilder(c *Catalogue) []ToolSpec { return healthSpecs(c.deps.Health) }

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
