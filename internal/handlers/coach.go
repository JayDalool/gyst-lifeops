package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/secretary"
	"lifeops-tracker/internal/services"
	"math"
)

// Health Coach (Issue #64). GYST builds a small context and offers the model
// allowlisted, read-only tools over Health data only. No SQL, files, shell,
// URLs or other LifeOps areas (tasks, projects, prayer, courses) are exposed.
// The Secretary's wider catalogue lives in internal/secretary (Issue #236);
// the Coach stays Health-only and takes only secretary.HealthTools.

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

// coachWater is today's water for the Coach. No taps is null, never 0, and the
// target and bottle follow HasTarget/HasBottle, not the formatted strings.
func coachWater(v healthWaterView) map[string]any {
	w := map[string]any{"logged_l": nil, "taps_today": v.Taps, "target_l": nil, "bottle_l": nil, "target_reached": nil,
		"note": "Logged by bottle taps on Today; taps_today counts taps, not litres. null means nothing logged, not zero. GYST stops adding bottles at the target, so the real amount may be higher. Nothing is estimated from food or Apple Health."}
	if v.Logged {
		w["logged_l"] = v.Total
	}
	if v.HasTarget {
		w["target_l"], w["target_reached"] = v.Target, v.Done
	}
	if v.HasBottle {
		w["bottle_l"] = v.Bottle
	}
	return w
}

// coachToday is the compact Today summary, also embedded in the system prompt.
func coachToday(data HealthData) map[string]any {
	t := data.Today
	macros := []map[string]any{}
	for _, m := range t.Nutrients {
		macros = append(macros, map[string]any{"name": m.Short(), "eaten_g": metricOrNil(m.Consumed), "target_g": metricOrNil(m.Target), "note": m.Note, "partial": m.Partial})
	}
	cal := map[string]any{"eaten": metricOrNil(t.Calories.Consumed), "target": metricOrNil(t.CalorieTarget), "partial": t.Calories.Partial}
	if t.Calories.HasBar {
		key := "budget_left"
		if t.CaloriesOver {
			key = "over_budget_by"
		}
		cal[key] = t.BudgetAbs
	}
	// Only when the day's food includes AI estimates; absent otherwise.
	if est := t.Estimated.aiEstimated(); est != nil {
		cal["ai_estimated"] = est
	}
	return map[string]any{
		"date":                          t.Date,
		"apple_health_last_sync":        map[bool]string{true: t.LastSynced, false: "none today"}[t.HasLastSynced],
		"sync_stale":                    t.LastSyncedStale,
		"intake_source":                 t.IntakeSource,
		"apple_health_kcal_not_counted": map[bool]any{true: metricOrNil(t.ReferenceKcal), false: nil}[t.IntakeSource == models.IntakeSourceGYST],
		"calories_kcal":                 cal,
		"burned_so_far_kcal":            metricOrNil(t.ObservedBurn),
		"energy_balance_so_far":         metricOrNil(t.ObservedEnergy),
		"note_on_balance":               "Observed so far today; the day isn't over, so this is not a final deficit.",
		"macros":                        macros,
		"steps":                         map[string]any{"count": metricOrNil(t.Steps), "source": t.StepsSource, "target": metricOrNil(t.StepTarget)},
		"exercise_minutes":              metricOrNil(t.ExerciseMinutes),
		"latest_weight_kg":              map[bool]any{true: data.Weight.Latest, false: nil}[data.Weight.Present],
		"weight_change_since_plan":      map[bool]any{true: data.Weight.Change, false: nil}[data.Weight.HasChange],
		"water":                         coachWater(data.Water),
	}
}

// coachTools are the only things the model can read: the nine Health tools
// (internal/secretary.HealthTools, over coachHealth) and the navigation action.
// The Coach gets no other Secretary tool and no freshness sync (Issue #236).
func (h *Handler) coachTools(data HealthData, action *string) []healthcoach.Tool {
	return append(secretary.HealthTools(coachHealth{h: h, data: data, now: h.Service.Now(), loc: h.Service.Location()}), healthcoach.ActionTool(action))
}

func metricOrNilString(ok bool, v string) any {
	if !ok {
		return nil
	}
	return v
}

// addCardioFields adds per-set cardio data only when some timed set has a parsed
// incline or speed. The exercise-level values are kept only when every timed set
// agrees; otherwise they are null so a mixed session is never summarised by its
// first set.
func addCardioFields(item map[string]any, sets []models.TrainingSet) {
	var per []map[string]any
	var incline, speed []*float64
	parsed := false
	for _, s := range sets {
		if s.Seconds == nil {
			continue
		}
		parsed = parsed || s.InclinePct != nil || s.SpeedKmh != nil
		set := map[string]any{"duration_min": math.Round(*s.Seconds/60*10) / 10, "incline_pct": nil, "speed_kmh": nil}
		if s.InclinePct != nil {
			set["incline_pct"] = *s.InclinePct
		}
		if s.SpeedKmh != nil {
			set["speed_kmh"] = *s.SpeedKmh
		}
		per = append(per, set)
		incline, speed = append(incline, s.InclinePct), append(speed, s.SpeedKmh)
	}
	if !parsed {
		return
	}
	item["sets"] = per
	item["incline_pct"], item["speed_kmh"] = commonValue(incline), commonValue(speed)
}

// commonValue is the shared value when every entry is present and equal, else nil.
func commonValue(v []*float64) any {
	for _, x := range v {
		if x == nil || *x != *v[0] {
			return nil
		}
	}
	return *v[0]
}

func bestSetText(sets []models.TrainingSet) string {
	best, text := -1.0, ""
	for _, s := range sets {
		if s.Kind == "warmup" || s.LoadKG == nil || s.Reps == nil {
			continue
		}
		if v := *s.LoadKG * (1 + *s.Reps/30); v > best {
			best, text = v, fmt.Sprintf("%s kg × %g", services.FormatLoadKG(*s.LoadKG), *s.Reps)
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
- Food eaten today comes from one source, intake_source (gyst diary, apple_health or none). apple_health_kcal_not_counted is a reference only: never add it to the eaten values. A partial value means some logged food has no value for it, so say it is at least that much.
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
	coachProvider := h.coach()
	base, ok := h.healthBase(w)
	if !ok {
		return
	}
	data := healthCoachData{HealthData: base, Configured: coachProvider != nil, Question: question, Error: errText, Prompts: coachPrompts}
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
	provider := h.coach() // resolved once per request
	switch {
	case provider == nil:
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
	reply, err := provider.Respond(ctx, req)
	if err != nil {
		// Only fixed messages are shown; nothing about the request is logged.
		status, msg := http.StatusBadGateway, healthcoach.ErrUnavailable.Error()
		for _, known := range []error{healthcoach.ErrNotConfigured, healthcoach.ErrBusy, healthcoach.ErrDeclined, healthcoach.ErrCreditExhausted, healthcoach.ErrSpendLimit, healthcoach.ErrUnavailable} {
			if errors.Is(err, known) {
				msg = known.Error()
			}
		}
		if errors.Is(err, healthcoach.ErrBusy) {
			status = http.StatusTooManyRequests
		}
		// The vendor error can echo the prompt, which holds health values, so
		// only the fixed category is logged, never err itself.
		h.log().LogAttrs(r.Context(), slog.LevelWarn, "provider failure",
			slog.String("event", "provider.failed"),
			slog.String("provider", "health-coach"),
			slog.String("category", msg),
			slog.String("request_id", requestIDFrom(r.Context())))
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
