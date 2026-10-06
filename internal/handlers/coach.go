package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// Health Coach (Issue #64). GYST builds a small context and offers the model
// allowlisted, read-only tools over Health data only. No SQL, files, shell,
// URLs or other LifeOps areas (tasks, projects, prayer, courses) are exposed.

const (
	coachHistoryTurns = 8    // visible turns sent back to the provider
	coachShownTurns   = 40   // turns shown on the page
	coachMaxQuestion  = 1000 // characters
)

// coachLimiter is a small in-process guard against runaway cost: one request
// at a time, a short gap between requests and an hourly cap.
type coachLimiter struct {
	mu     sync.Mutex
	busy   bool
	last   time.Time
	window []time.Time
}

func (l *coachLimiter) acquire(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := now.Add(-time.Hour)
	kept := l.window[:0]
	for _, t := range l.window {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	l.window = kept
	if l.busy || now.Sub(l.last) < 3*time.Second || len(l.window) >= 40 {
		return false
	}
	l.busy, l.last = true, now
	l.window = append(l.window, now)
	return true
}

func (l *coachLimiter) release() {
	l.mu.Lock()
	l.busy = false
	l.mu.Unlock()
}

// Profile keys in app_settings. Values are what Jay typed; nothing is inferred.
var coachProfileFields = []struct{ Key, Label, Hint string }{
	{"health.profile.age", "Age", "e.g. 24"},
	{"health.profile.sex", "Sex", "e.g. Male"},
	{"health.profile.height_cm", "Height (cm)", "e.g. 181"},
	{"health.profile.goal_weight_kg", "Goal weight (kg)", "e.g. 110"},
	{"health.profile.food_rules", "Food rules", "e.g. Halal, no alcohol"},
	{"health.profile.notes", "Notes for the coach", "Anything else to keep in mind"},
}

func (h *Handler) coachProfile() map[string]string {
	out := map[string]string{}
	for _, f := range coachProfileFields {
		if v, err := h.Service.Repo.GetSetting(f.Key); err == nil && strings.TrimSpace(v) != "" {
			out[strings.TrimPrefix(f.Key, "health.profile.")] = v
		}
	}
	return out
}

func toJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

func metricOrNil(m HealthMetricView) any {
	if !m.Present {
		return nil
	}
	return m.Value
}

// coachToday is the compact Today summary, also embedded in the system prompt.
func coachToday(data HealthData) map[string]any {
	t := data.Today
	macros := []map[string]any{}
	for _, m := range t.Nutrients {
		macros = append(macros, map[string]any{"name": m.Short(), "eaten_g": metricOrNil(m.Consumed), "target_g": metricOrNil(m.Target), "note": m.Note})
	}
	cal := map[string]any{"eaten": metricOrNil(t.Calories.Consumed), "target": metricOrNil(t.CalorieTarget)}
	if t.Calories.HasBar {
		key := "budget_left"
		if t.CaloriesOver {
			key = "over_budget_by"
		}
		cal[key] = t.BudgetAbs
	}
	return map[string]any{
		"date":                     t.Date,
		"apple_health_last_sync":   map[bool]string{true: t.LastSynced, false: "none today"}[t.HasLastSynced],
		"sync_stale":               t.LastSyncedStale,
		"calories_kcal":            cal,
		"burned_so_far_kcal":       metricOrNil(t.ObservedBurn),
		"energy_balance_so_far":    metricOrNil(t.ObservedEnergy),
		"note_on_balance":          "Observed so far today; the day isn't over, so this is not a final deficit.",
		"macros":                   macros,
		"steps":                    map[string]any{"count": metricOrNil(t.Steps), "source": t.StepsSource, "target": metricOrNil(t.StepTarget)},
		"exercise_minutes":         metricOrNil(t.ExerciseMinutes),
		"latest_weight_kg":         map[bool]any{true: data.Weight.Latest, false: nil}[data.Weight.Present],
		"weight_change_since_plan": map[bool]any{true: data.Weight.Change, false: nil}[data.Weight.HasChange],
	}
}

// coachTools are the only things the model can read. Each returns compact JSON.
func (h *Handler) coachTools(data HealthData, action *string) []healthcoach.Tool {
	now, loc := h.Service.Now(), h.Service.Location()
	training := func() []models.TrainingEntry {
		cache, err := h.Service.Repo.TrainingCache("hevy")
		if err != nil {
			return nil
		}
		return cache.Entries
	}
	none := func(f func() (any, error)) func(context.Context, json.RawMessage) (string, error) {
		return func(context.Context, json.RawMessage) (string, error) {
			v, err := f()
			if err != nil {
				return "", errors.New("data unavailable")
			}
			return toJSON(v)
		}
	}
	return []healthcoach.Tool{
		{Name: "get_health_today", Description: "Today's calories (eaten, target, budget left), burned so far, energy balance so far, macros, steps, exercise minutes and latest weight.",
			Run: none(func() (any, error) { return coachToday(data), nil })},
		{Name: "get_health_plan", Description: "The current health plan: dates and daily targets (calories, macros, steps, strength and cardio sessions per week).",
			Run: none(func() (any, error) {
				c, err := h.Service.GetHealthCycleForDate(data.Today.Date)
				if err != nil {
					return map[string]any{"plan": nil}, nil
				}
				return map[string]any{"title": c.Title, "goal": c.PrimaryGoal, "start": c.StartDate, "end": c.EndDate, "calories": c.DailyCalorieTarget, "protein_g": c.ProteinTargetG,
					"carbs_g": c.CarbsTargetG, "fat_g": c.FatTargetG, "fiber_g": c.FiberTargetG, "target_daily_deficit_kcal": c.TargetDeficitKcal,
					"strength_sessions_per_week": c.StrengthSessionsPerWeek, "cardio_sessions_per_week": c.CardioSessionsPerWeek, "step_target": c.StepTarget}, nil
			})},
		{Name: "get_weight_trend", Description: "Weight readings from the last 90 days (date, kg), latest reading and change since the plan started.",
			Run: none(func() (any, error) {
				entries, err := h.Service.ListHealthWeightEntries()
				if err != nil {
					return nil, err
				}
				readings := []map[string]any{}
				cut := now.AddDate(0, 0, -90)
				for _, e := range entries {
					at, err := time.Parse(time.RFC3339Nano, e.MeasuredAt)
					if err == nil && at.After(cut) && !at.After(now) {
						readings = append(readings, map[string]any{"date": at.In(loc).Format("2006-01-02"), "kg": e.WeightKg})
					}
				}
				if len(readings) > 30 {
					readings = readings[:30]
				}
				return map[string]any{"latest_kg": metricOrNilString(data.Weight.Present, data.Weight.Latest), "change_since": data.Weight.Since, "change": data.Weight.Change, "readings_newest_first": readings}, nil
			})},
		{Name: "get_training_summary", Description: "This week's workouts against the plan target, volume, training time, workouts per week for 8 weeks, and muscle-group shares over 30 days.",
			Run: none(func() (any, error) {
				e := training()
				week := services.SummarizeTrainingWeek(e, now, loc)
				target := 0
				if c, err := h.Service.GetHealthCycleForDate(data.Today.Date); err == nil {
					target = int(c.StrengthSessionsPerWeek)
				}
				groups := []map[string]any{}
				for _, g := range services.MuscleGroupShares(e, now, 30) {
					groups = append(groups, map[string]any{"group": g.Name, "percent": g.Percent, "sets": g.Sets})
				}
				return map[string]any{"this_week_workouts": week.Workouts, "weekly_target": target, "volume_kg": week.VolumeKG, "training_minutes": week.Minutes,
					"workouts_without_end_time": week.UnknownTime, "workouts_per_week_oldest_first": services.WorkoutsPerWeek(e, now, loc, 8), "muscle_groups_30_days": groups}, nil
			})},
		{Name: "get_recent_workouts", Description: "The most recent Hevy workouts with each exercise's best set. Input: count (1-5).",
			Properties: map[string]any{"count": map[string]any{"type": "integer", "minimum": 1, "maximum": 5}},
			Run: func(_ context.Context, input json.RawMessage) (string, error) {
				var in struct {
					Count int `json:"count"`
				}
				_ = json.Unmarshal(input, &in)
				in.Count = min(max(in.Count, 1), 5)
				e := training()
				sort.SliceStable(e, func(i, j int) bool { return e[i].StartedAt.After(e[j].StartedAt) })
				out := []map[string]any{}
				for _, w := range e {
					if w.Kind != "workout" || len(out) == in.Count {
						continue
					}
					exercises := []map[string]any{}
					for _, ex := range w.Exercises {
						exercises = append(exercises, map[string]any{"name": ex.Name, "muscle": services.MuscleName(ex.PrimaryMuscle), "best_set": bestSetText(ex.Sets), "work_sets": workSetCount(ex.Sets)})
					}
					item := map[string]any{"title": w.Title, "date": w.StartedAt.In(loc).Format("2006-01-02"), "exercises": exercises}
					if !w.EndedAt.IsZero() {
						item["minutes"] = int(w.EndedAt.Sub(w.StartedAt).Minutes())
					}
					out = append(out, item)
				}
				return toJSON(map[string]any{"workouts": out})
			}},
		{Name: "get_exercise_progress", Description: "Change in best estimated 1-rep max per exercise (exercises with 2+ sessions). Input: weeks (4, 12 or 26).",
			Properties: map[string]any{"weeks": map[string]any{"type": "integer", "enum": []int{4, 12, 26}}},
			Run: func(_ context.Context, input json.RawMessage) (string, error) {
				var in struct {
					Weeks int `json:"weeks"`
				}
				_ = json.Unmarshal(input, &in)
				if in.Weeks != 4 && in.Weeks != 26 {
					in.Weeks = 12
				}
				out := []map[string]any{}
				for _, t := range services.ExerciseProgress(training(), now, in.Weeks*7) {
					out = append(out, map[string]any{"exercise": t.Name, "change_percent": t.ChangePct, "sessions": t.Sessions, "best_set": t.BestSet})
				}
				return toJSON(map[string]any{"weeks": in.Weeks, "exercises": out, "note": "Only exercises logged in two or more sessions in the window appear."})
			}},
		{Name: "get_hevy_routines", Description: "Jay's Hevy routines and their exercises.",
			Run: none(func() (any, error) {
				out := []map[string]any{}
				for _, r := range training() {
					if r.Kind != "routine" {
						continue
					}
					names := []string{}
					for _, ex := range r.Exercises {
						names = append(names, ex.Name)
					}
					out = append(out, map[string]any{"title": r.Title, "exercises": names})
				}
				return map[string]any{"routines": out}, nil
			})},
		{Name: "get_food_products", Description: "Foods and products Jay usually has, with per-serving nutrition (null = unknown).",
			Run: none(func() (any, error) {
				foods, err := h.Service.Repo.ListHealthFoods(false)
				if err != nil {
					return nil, err
				}
				out := []map[string]any{}
				for _, f := range foods {
					out = append(out, map[string]any{"name": f.Name, "serving": f.Serving, "kcal": f.Kcal, "protein_g": f.ProteinG, "carbs_g": f.CarbsG, "fat_g": f.FatG, "fiber_g": f.FiberG, "notes": f.Notes})
				}
				return map[string]any{"foods": out}, nil
			})},
		{Name: "get_daily_checklist", Description: "Today's checklist: Jay's own habits (done or not) and measured targets.",
			Run: none(func() (any, error) {
				out := []map[string]any{}
				for _, r := range data.Checklist {
					item := map[string]any{"name": r.Name, "done": r.Done}
					if r.Measured {
						item["measured"] = r.Value
					}
					out = append(out, item)
				}
				return map[string]any{"items": out}, nil
			})},
		healthcoach.ActionTool(action),
	}
}

func metricOrNilString(ok bool, v string) any {
	if !ok {
		return nil
	}
	return v
}

func bestSetText(sets []models.TrainingSet) string {
	best, text := -1.0, ""
	for _, s := range sets {
		if s.Kind == "warmup" || s.LoadKG == nil || s.Reps == nil {
			continue
		}
		if v := *s.LoadKG * (1 + *s.Reps/30); v > best {
			best, text = v, fmt.Sprintf("%g kg × %g", *s.LoadKG, *s.Reps)
		}
	}
	return text
}

func workSetCount(sets []models.TrainingSet) int {
	n := 0
	for _, s := range sets {
		if s.Kind == "normal" || s.Kind == "failure" || s.Kind == "dropset" {
			n++
		}
	}
	return n
}

// coachSystemPrompt is stable rules plus the profile and today's summary.
func coachSystemPrompt(profile map[string]string, today map[string]any) string {
	p, _ := json.Marshal(profile)
	t, _ := json.Marshal(today)
	return `You are the Health Coach inside GYST, Jay's personal app. You help with nutrition, training and using GYST.

Style: short and practical. Numbers first. One to three relevant numbers, then at most a short recommendation. No long essays unless asked. Plain text only, no Markdown tables or headings.

Rules:
- Use only the data below and your tools. If data is missing, say so plainly; never invent values, meals or workouts.
- The day isn't over: call calorie balance "so far", never a final deficit.
- Never claim spot fat reduction. Never turn steps into calories.
- Respect Jay's food rules in the profile (for example halal, no alcohol) in every food suggestion.
- Prefer foods from get_food_products when suggesting meals or ways to close a protein gap.
- You are not a doctor. For symptoms, pain, injuries, medication or abnormal readings, give general guidance only and suggest a professional.
- To point Jay to a GYST page, call suggest_gyst_page; never write URLs.

Profile (typed by Jay): ` + string(p) + `
Today (observed so far): ` + string(t)
}

type coachTurn struct {
	Role, Text  string
	Last        bool
	ActionLabel string
	ActionPath  string
}

type healthCoachData struct {
	HealthData
	Configured bool
	Turns      []coachTurn
	Question   string
	Error      string
	Prompts    []string
}

var coachPrompts = []string{
	"What should I eat for breakfast?",
	"How much protein do I have left?",
	"Should I train today?",
	"How has my training been?",
	"What have I neglected in the gym?",
	"Make me dinner using foods I have.",
}

func (h *Handler) renderCoach(w http.ResponseWriter, status int, question, errText string) {
	base, ok := h.healthBase(w)
	if !ok {
		return
	}
	base.HealthTab = "coach"
	data := healthCoachData{HealthData: base, Configured: h.Coach != nil, Question: question, Error: errText, Prompts: coachPrompts}
	msgs, err := h.Service.Repo.ListCoachMessages(coachShownTurns)
	if err != nil {
		serverError(w, err)
		return
	}
	for _, m := range msgs {
		turn := coachTurn{Role: m.Role, Text: m.Text}
		if d, ok := healthcoach.Destinations[m.Action]; ok {
			turn.ActionLabel, turn.ActionPath = d.Label, d.Path
		}
		data.Turns = append(data.Turns, turn)
	}
	if n := len(data.Turns); n > 0 {
		data.Turns[n-1].Last = true
	}
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	h.render(w, "health_coach.html", data)
}

func (h *Handler) healthCoach(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "GET, POST") {
		return
	}
	if r.Method == http.MethodGet {
		h.renderCoach(w, http.StatusOK, "", "")
		return
	}
	question := strings.TrimSpace(r.PostFormValue("q"))
	switch {
	case h.Coach == nil:
		h.renderCoach(w, http.StatusServiceUnavailable, question, healthcoach.ErrNotConfigured.Error())
		return
	case question == "":
		h.renderCoach(w, http.StatusBadRequest, question, "Type a question first")
		return
	case utf8.RuneCountInString(question) > coachMaxQuestion:
		h.renderCoach(w, http.StatusBadRequest, question, "Questions can be at most "+strconv.Itoa(coachMaxQuestion)+" characters")
		return
	}
	if !h.coachLimit.acquire(h.Service.Now()) {
		h.renderCoach(w, http.StatusTooManyRequests, question, healthcoach.ErrBusy.Error())
		return
	}
	defer h.coachLimit.release()

	base, ok := h.healthBase(w)
	if !ok {
		return
	}
	history, err := h.Service.Repo.ListCoachMessages(coachHistoryTurns)
	if err != nil {
		serverError(w, err)
		return
	}
	req := healthcoach.Request{System: coachSystemPrompt(h.coachProfile(), coachToday(base))}
	for _, m := range history {
		req.History = append(req.History, healthcoach.Message{Role: m.Role, Text: m.Text})
	}
	req.History = append(req.History, healthcoach.Message{Role: "user", Text: question})
	var action string
	req.Tools = h.coachTools(base, &action)

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	reply, err := h.Coach.Respond(ctx, req)
	if err != nil {
		// Only fixed messages are shown; nothing about the request is logged.
		status, msg := http.StatusBadGateway, healthcoach.ErrUnavailable.Error()
		for _, known := range []error{healthcoach.ErrNotConfigured, healthcoach.ErrBusy, healthcoach.ErrDeclined, healthcoach.ErrUnavailable} {
			if errors.Is(err, known) {
				msg = known.Error()
			}
		}
		if errors.Is(err, healthcoach.ErrBusy) {
			status = http.StatusTooManyRequests
		}
		h.renderCoach(w, status, question, msg)
		return
	}
	if _, ok := healthcoach.Destinations[action]; !ok {
		action = ""
	}
	text := reply.Text
	if utf8.RuneCountInString(text) > 8000 {
		text = string([]rune(text)[:8000])
	}
	if err := h.Service.Repo.AddCoachMessage("user", question, ""); err != nil {
		serverError(w, err)
		return
	}
	if err := h.Service.Repo.AddCoachMessage("assistant", text, action); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/health/coach#latest", http.StatusSeeOther)
}

func (h *Handler) healthCoachClear(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "POST") {
		return
	}
	if err := h.Service.Repo.ClearCoachMessages(); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/health/coach", http.StatusSeeOther)
}

type healthProfileData struct {
	Page   string
	Fields []struct{ Key, Name, Label, Hint, Value string }
	Saved  bool
	Error  string
}

func (healthProfileData) HealthTabName() string { return "more" }

// healthProfile edits the few facts the coach uses. Values are stored as typed.
func (h *Handler) healthProfile(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "GET, POST") {
		return
	}
	data := healthProfileData{Page: "health", Saved: r.URL.Query().Get("saved") == "1"}
	values := map[string]string{}
	if r.Method == http.MethodPost {
		for _, f := range coachProfileFields {
			name := strings.TrimPrefix(f.Key, "health.profile.")
			v := strings.TrimSpace(r.PostFormValue(name))
			if utf8.RuneCountInString(v) > 300 {
				data.Error = f.Label + " can be at most 300 characters"
			}
			values[f.Key] = v
		}
		if data.Error == "" {
			for k, v := range values {
				var err error
				if v == "" {
					err = h.Service.Repo.DeleteSetting(k)
				} else {
					err = h.Service.Repo.SetSetting(k, v)
				}
				if err != nil {
					serverError(w, err)
					return
				}
			}
			http.Redirect(w, r, "/health/profile?saved=1", http.StatusSeeOther)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}
	for _, f := range coachProfileFields {
		v, ok := values[f.Key]
		if !ok {
			v, _ = h.Service.Repo.GetSetting(f.Key)
		}
		data.Fields = append(data.Fields, struct{ Key, Name, Label, Hint, Value string }{f.Key, strings.TrimPrefix(f.Key, "health.profile."), f.Label, f.Hint, v})
	}
	h.render(w, "health_profile.html", data)
}
