package secretary

import (
	"context"
	"encoding/json"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// The day brief is the one small summary the Secretary can read for a date
// (Issue #236, brain doc section 6). It is a hard 4096 bytes: the model gets it
// on every turn, so its size is a cost and a privacy boundary. It never holds a
// description, a note, a sender, a subject, a body, an attendee, a location or
// a URL; the only free text is cleaned task and workout titles and calendar
// titles inside an untrusted envelope.

// BriefBudget is the most bytes BuildDayBrief returns: len() of the exact
// bytes (json.Marshal, compact, UTF-8). json.Marshal escapes '<', '>' and '&'
// as six bytes, so the caps below assume the worst case of six bytes a rune.
const BriefBudget = 4096

// Section caps in bytes, each counting the section's `"key":` and its comma.
// They sum, with briefOverhead, to at most BriefBudget (TestBriefCapsFit), so
// the final guard only fires if a section breaks its own cap.
const (
	capNow      = 180
	capPrayer   = 300
	capTasks    = 700
	capCalendar = 1100
	capHealth   = 600
	capTraining = 640
	capMail     = 120
	capSync     = 260
	// briefOverhead is the object's braces and the worst `,"cut":[...]` naming
	// every droppable section.
	briefOverhead = 2 + len(`,"cut":["prayer","tasks","calendar","health","training","mail","sync"]`)
	// briefTasksListed is how many due titles the brief lists at most.
	briefTasksListed = 5
)

// HealthBrief is today's Health in the brief, read through HealthReader. Values
// are the strings the Health page shows; nil means not observed and is written
// as null, never 0 (RULES 3.3-3.5): steps are one count with its source (Watch
// and phone are never added), nothing is derived from steps, and the food
// budget and the observed energy balance are separate fields.
type HealthBrief struct {
	// AsOf is when Apple Health last synced; zero means never.
	AsOf  time.Time
	Stale bool

	FoodEatenKcal, FoodTargetKcal, FoodBudgetLeftKcal, FoodOverBudgetByKcal *string
	ProteinEatenG, ProteinTargetG                                           *string
	Steps                                                                   *string
	StepsSource                                                             string // "watch", "phone" or ""
	// BurnedSoFarKcal is observed resting plus active energy.
	BurnedSoFarKcal *string
	// EnergyBalanceSoFarKcal is observed so far, not a final deficit.
	EnergyBalanceSoFarKcal     *string
	WaterLoggedL, WaterTargetL *string
	WeightKg                   *string
}

// CalendarEvent is one event for the brief. Title is raw: the brief wraps it.
type CalendarEvent struct {
	Start, End time.Time
	Busy       bool
	Title      string
}

// CalendarReader is filled by the calendar chunks (C-1, C-4). Nil until then.
type CalendarReader interface {
	Events(day Day) ([]CalendarEvent, error)
}

type dayBrief struct {
	Now      json.RawMessage `json:"now"`
	Prayer   json.RawMessage `json:"prayer,omitempty"`
	Tasks    json.RawMessage `json:"tasks,omitempty"`
	Calendar json.RawMessage `json:"calendar,omitempty"`
	Health   json.RawMessage `json:"health,omitempty"`
	Training json.RawMessage `json:"training,omitempty"`
	Mail     json.RawMessage `json:"mail,omitempty"`
	Sync     json.RawMessage `json:"sync,omitempty"`
	Cut      []string        `json:"cut,omitempty"`
}

// BuildDayBrief builds the brief for a day. It is pure: it reads stored rows
// and the freshness state with Peek, never starts a sync and never writes, so a
// GET may call it (Issue #188). Only the Catalogue's tool wrapper syncs.
func (c *Catalogue) BuildDayBrief(day Day) ([]byte, error) {
	svc := c.deps.Service
	loc := svc.Location()
	asOf := func(t time.Time) any {
		if t.IsZero() {
			return nil
		}
		return t.In(loc).Format(minuteLayout)
	}
	now := svc.Now()
	b := dayBrief{}
	b.Now = mustJSON(map[string]any{"as_of": asOf(now), "stale": false, "date": day.Date, "weekday": day.Weekday})
	b.Prayer = c.briefPrayer(day, asOf)
	b.Tasks = c.briefTasks(day, asOf(now))
	hevy := c.fresh.Peek(SourceHevy)
	cal, mail := c.fresh.Peek(SourceCalendar), c.fresh.Peek(SourceMail)
	b.Calendar = c.briefCalendar(day, cal, asOf)
	var health HealthBrief
	healthOK := false
	if day.Offset == 0 {
		var err error
		if health, err = c.deps.Health.BriefHealth(); err == nil {
			healthOK = true
		}
	}
	b.Health = briefHealth(health, healthOK, asOf)
	b.Training = c.briefTraining(hevy, asOf)
	b.Mail = mustJSON(map[string]any{"as_of": asOf(mail.AsOf), "stale": mail.Stale, "connected": mail.Connected, "unread": nil})
	b.Sync = mustJSON(map[string]any{"hevy": asOf(hevy.AsOf), "calendar": asOf(cal.AsOf), "mail": asOf(mail.AsOf), "health": asOf(health.AsOf)})
	caps := map[string]int{"now": capNow, "prayer": capPrayer, "tasks": capTasks, "calendar": capCalendar, "health": capHealth, "training": capTraining, "mail": capMail, "sync": capSync}
	return finalize(b, caps, BriefBudget)
}

// minuteLayout is the brief's time format: the application's local clock, to
// the minute, no zone (every time in the brief is local).
const minuteLayout = "2006-01-02T15:04"

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return raw
}

// sectionOrder is the fixed order of the brief, and, reversed, the order in
// which the final guard drops sections. now is never dropped.
var sectionOrder = []string{"now", "prayer", "tasks", "calendar", "health", "training", "mail", "sync"}

func (b *dayBrief) slot(name string) *json.RawMessage {
	switch name {
	case "now":
		return &b.Now
	case "prayer":
		return &b.Prayer
	case "tasks":
		return &b.Tasks
	case "calendar":
		return &b.Calendar
	case "health":
		return &b.Health
	case "training":
		return &b.Training
	case "mail":
		return &b.Mail
	}
	return &b.Sync
}

// finalize drops any section over its own cap, then, while the whole is over
// limit, drops whole sections from the bottom, naming each in "cut". A section
// is kept entire or dropped entire: no value is ever cut partway.
func finalize(b dayBrief, caps map[string]int, limit int) ([]byte, error) {
	for _, name := range sectionOrder[1:] {
		raw := b.slot(name)
		if len(*raw) > 0 && len(name)+len(*raw)+4 > caps[name] {
			*raw = nil
			b.Cut = append(b.Cut, name)
		}
	}
	out, err := json.Marshal(b)
	for i := len(sectionOrder) - 1; err == nil && len(out) > limit && i > 0; i-- {
		raw := b.slot(sectionOrder[i])
		if len(*raw) == 0 {
			continue
		}
		*raw = nil
		b.Cut = append(b.Cut, sectionOrder[i])
		out, err = json.Marshal(b)
	}
	return out, err
}

// fitList marshals a list section and adds whole items while it stays inside
// the cap. build gets the items that fit and how many were left out.
func fitList[T any](key string, cap int, items []T, build func(listed []T, more int) any) json.RawMessage {
	var best json.RawMessage
	for n := 0; n <= len(items); n++ {
		raw := mustJSON(build(items[:n], len(items)-n))
		if len(key)+len(raw)+4 > cap {
			break
		}
		best = raw
	}
	return best // nil (dropped by finalize's cap check) only if even zero items do not fit
}

func (c *Catalogue) briefPrayer(day Day, asOf func(time.Time) any) json.RawMessage {
	svc := c.deps.Service
	schedule, err := svc.PrayerTimes(day.Time())
	if err != nil {
		return mustJSON(map[string]any{"as_of": nil, "stale": true, "times": nil, "next": nil})
	}
	times := make([][2]string, 0, len(schedule.Times))
	for _, t := range schedule.Times {
		times = append(times, [2]string{t.Name, t.At.Format("15:04")})
	}
	var next any
	if day.Offset == 0 {
		if n, err := svc.NextPrayer(svc.Now()); err == nil {
			next = map[string]any{"name": n.Prayer.Name, "time": n.Prayer.At.Format("15:04"), "date": svc.DateOf(n.Prayer.At)}
		}
	}
	return mustJSON(map[string]any{"as_of": asOf(svc.Now()), "stale": false, "times": times, "next": next})
}

func (c *Catalogue) briefTasks(day Day, nowAsOf any) json.RawMessage {
	svc := c.deps.Service
	null := mustJSON(map[string]any{"as_of": nil, "stale": true, "late": nil, "due": nil, "no_date": nil, "titles": nil, "more": nil})
	late, err := svc.OverdueTasks()
	if err != nil {
		return null
	}
	occ, err := svc.TaskOccurrencesOn(day.Time())
	if err != nil {
		return null
	}
	all, err := svc.Repo.ListTasks(false)
	if err != nil {
		return null
	}
	noDate := 0
	for _, t := range all {
		if t.DueDate == "" && t.Status == "pending" {
			noDate++
		}
	}
	var titles []string
	for _, o := range occ {
		if o.Status == "pending" {
			t, _ := Clean(o.Task.Title, TitleCap)
			titles = append(titles, t)
		}
	}
	total := len(titles)
	titles = titles[:min(len(titles), briefTasksListed)]
	return fitList("tasks", capTasks, titles, func(listed []string, more int) any {
		if listed == nil {
			listed = []string{}
		}
		return map[string]any{"as_of": nowAsOf, "stale": false, "late": len(late), "due": total, "no_date": noDate, "titles": listed, "more": total - len(listed)}
	})
}

func (c *Catalogue) briefCalendar(day Day, f Freshness, asOf func(time.Time) any) json.RawMessage {
	reader := c.deps.Calendar
	if reader == nil {
		return mustJSON(map[string]any{"as_of": nil, "stale": true, "connected": false, "events": nil, "total": nil, "more": nil})
	}
	events, err := reader.Events(day)
	if err != nil {
		return mustJSON(map[string]any{"as_of": asOf(f.AsOf), "stale": true, "connected": true, "events": nil, "total": nil, "more": nil})
	}
	loc := c.deps.Service.Location()
	type item struct {
		Start string   `json:"start"`
		End   string   `json:"end"`
		Busy  bool     `json:"busy"`
		Title Envelope `json:"title"`
	}
	items := make([]item, 0, len(events))
	for _, e := range events {
		env, _ := Wrap("calendar", e.Title, TitleCap)
		items = append(items, item{e.Start.In(loc).Format(minuteLayout), e.End.In(loc).Format(minuteLayout), e.Busy, env})
	}
	return fitList("calendar", capCalendar, items, func(listed []item, more int) any {
		if listed == nil {
			listed = []item{}
		}
		return map[string]any{"as_of": asOf(f.AsOf), "stale": f.Stale, "connected": true, "events": listed, "total": len(items), "more": more}
	})
}

func briefHealth(h HealthBrief, ok bool, asOf func(time.Time) any) json.RawMessage {
	if !ok {
		return json.RawMessage("null") // not today, or unreadable: null stays null
	}
	var source any
	if h.StepsSource != "" {
		source = h.StepsSource
	}
	return mustJSON(map[string]any{
		"as_of": asOf(h.AsOf), "stale": h.Stale || h.AsOf.IsZero(),
		"food":    map[string]any{"eaten": h.FoodEatenKcal, "target": h.FoodTargetKcal, "left": h.FoodBudgetLeftKcal, "over_by": h.FoodOverBudgetByKcal},
		"protein": map[string]any{"eaten_g": h.ProteinEatenG, "target_g": h.ProteinTargetG},
		"steps":   map[string]any{"count": h.Steps, "source": source},
		"burned":  h.BurnedSoFarKcal, "balance": h.EnergyBalanceSoFarKcal,
		"water":  map[string]any{"logged_l": h.WaterLoggedL, "target_l": h.WaterTargetL},
		"weight": h.WeightKg,
	})
}

func (c *Catalogue) briefTraining(f Freshness, asOf func(time.Time) any) json.RawMessage {
	svc := c.deps.Service
	cache, err := svc.Repo.TrainingCache("hevy")
	if err != nil {
		return mustJSON(map[string]any{"as_of": nil, "stale": true, "last": nil, "week": nil})
	}
	var last any
	var newest *models.TrainingEntry
	for i := range cache.Entries {
		e := &cache.Entries[i]
		if e.Kind == "workout" && (newest == nil || e.StartedAt.After(newest.StartedAt)) {
			newest = e
		}
	}
	if newest != nil {
		title, _ := Clean(newest.Title, TitleCap)
		last = map[string]any{"date": svc.DateOf(newest.StartedAt), "title": title}
	}
	week := services.SummarizeTrainingWeek(cache.Entries, svc.Now(), svc.Location())
	var target any
	if cycle, err := svc.GetHealthCycleForDate(svc.TodayString()); err == nil && cycle.StrengthSessionsPerWeek > 0 {
		target = cycle.StrengthSessionsPerWeek
	}
	return mustJSON(map[string]any{"as_of": asOf(f.AsOf), "stale": f.Stale, "last": last, "week": map[string]any{"workouts": week.Workouts, "target": target}})
}

// getDayBrief is the get_day_brief tool.
func (c *Catalogue) getDayBrief(_ context.Context, input json.RawMessage) (any, error) {
	var in struct {
		Date string `json:"date"`
	}
	if err := decode(input, &in); err != nil {
		return nil, err
	}
	day, err := ResolveDate(in.Date, c.deps.Service.Clock)
	if err != nil {
		return nil, err
	}
	raw, err := c.BuildDayBrief(day)
	if err != nil {
		return nil, errUnavailable
	}
	return json.RawMessage(raw), nil
}
