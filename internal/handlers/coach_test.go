package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
)

// fakeCoach calls the tools the test asks for, records what it saw and
// returns a canned reply or error.
type fakeCoach struct {
	call     []string
	args     map[string]string
	reply    string
	err      error
	seen     healthcoach.Request
	toolData map[string]string
}

func (f *fakeCoach) Name() string { return "fake" }

func (f *fakeCoach) Respond(ctx context.Context, req healthcoach.Request) (healthcoach.Response, error) {
	f.seen = req
	f.toolData = map[string]string{}
	byName := map[string]healthcoach.Tool{}
	for _, t := range req.Tools {
		byName[t.Name] = t
	}
	for _, name := range f.call {
		tool, ok := byName[name]
		if !ok {
			return healthcoach.Response{}, errors.New("tool missing: " + name)
		}
		in := f.args[name]
		if in == "" {
			in = "{}"
		}
		out, err := tool.Run(ctx, json.RawMessage(in))
		if err != nil {
			f.toolData[name] = "ERR " + err.Error()
			continue
		}
		f.toolData[name] = out
	}
	if f.err != nil {
		return healthcoach.Response{}, f.err
	}
	return healthcoach.Response{Text: f.reply}, nil
}

func coachHandler(t *testing.T, fc *fakeCoach) *Handler {
	t.Helper()
	h := testHandler(t)
	if fc != nil {
		h.Coach = fc
	}
	return h
}

func ask(h *Handler, q, origin string) *httptest.ResponseRecorder {
	return cfgPost(h, "/health/coach", url.Values{"q": {q}}, origin)
}

func TestCoachDisabledWithoutProvider(t *testing.T) {
	h := coachHandler(t, nil)
	body := cfgGet(t, h, "/health/coach")
	if !strings.Contains(body, "AI provider not configured") || strings.Contains(body, `name="q"`) {
		t.Fatal("disabled state wrong")
	}
	if health := cfgGet(t, h, "/health"); strings.Contains(health, `href="/health/coach"`) {
		t.Fatal("Coach tab shown without a provider")
	}
	if w := ask(h, "Hi", ""); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("ask while disabled: %d", w.Code)
	}
}

func TestCoachAnswersWithHealthOnlyContext(t *testing.T) {
	fc := &fakeCoach{
		call:  []string{"get_health_today", "get_health_plan", "get_food_products", "get_daily_checklist", "get_training_summary", "get_recent_workouts", "get_exercise_progress", "get_hevy_routines", "get_weight_trend", "suggest_gyst_page"},
		args:  map[string]string{"suggest_gyst_page": `{"destination":"health_workout_progress"}`, "get_recent_workouts": `{"count":99}`},
		reply: "You have 48 g of protein left.\n<script>alert(1)</script>",
	}
	h := coachHandler(t, fc)
	h.Service.Clock.Now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, h.Service.Location()) }
	// Unrelated LifeOps data that must never reach the provider.
	if err := h.Service.Repo.CreateTask(&models.Task{Title: "SECRET-TASK-TITLE", Category: "work", Status: "pending", Priority: "medium", RepeatType: "none"}); err != nil {
		t.Fatal(err)
	}
	if err := h.Service.Repo.SetSetting("health.profile.food_rules", "Halal, no alcohol"); err != nil {
		t.Fatal(err)
	}
	food := 24.0
	if err := h.Service.Repo.CreateHealthFood(&models.HealthFood{Name: "Lean turkey", ProteinG: &food, Active: true}); err != nil {
		t.Fatal(err)
	}
	if w := ask(h, "How much protein do I have left?", "http://example.com"); w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/health/coach#latest" {
		t.Fatalf("ask: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(fc.seen.System, "Halal, no alcohol") || !strings.Contains(fc.seen.System, "not a final deficit") {
		t.Fatal("system prompt lacks the profile or the so-far rule")
	}
	everything := fc.seen.System
	for name, out := range fc.toolData {
		if strings.HasPrefix(out, "ERR") {
			t.Fatalf("%s failed: %s", name, out)
		}
		everything += out
	}
	if strings.Contains(everything, "SECRET-TASK-TITLE") {
		t.Fatal("non-Health data reached the provider")
	}
	if !strings.Contains(fc.toolData["get_food_products"], "Lean turkey") {
		t.Fatal("foods tool missing data")
	}
	last := fc.seen.History[len(fc.seen.History)-1]
	if last.Role != "user" || last.Text != "How much protein do I have left?" {
		t.Fatalf("question not last in history: %+v", last)
	}
	page := cfgGet(t, h, "/health/coach")
	if !strings.Contains(page, "&lt;script&gt;alert(1)&lt;/script&gt;") || strings.Contains(page, "<script>alert(1)") {
		t.Fatal("reply not escaped")
	}
	if !strings.Contains(page, `href="/health/training?view=progress">Open Workout progress</a>`) || !strings.Contains(page, `id="latest"`) {
		t.Fatal("allowlisted action missing")
	}
	if !strings.Contains(cfgGet(t, h, "/health"), `href="/health/coach"`) {
		t.Fatal("Coach tab missing when configured")
	}

	// The next question carries the stored history back.
	h.Service.Clock.Now = func() time.Time { return time.Date(2026, 10, 7, 12, 1, 0, 0, h.Service.Location()) }
	fc.call = nil
	ask(h, "And carbs?", "")
	if len(fc.seen.History) != 3 || fc.seen.History[0].Text != "How much protein do I have left?" {
		t.Fatalf("history %+v", fc.seen.History)
	}

	// Clear deletes the conversation.
	if w := cfgPost(h, "/health/coach/clear", nil, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("clear: %d", w.Code)
	}
	if msgs, _ := h.Service.Repo.ListCoachMessages(40); len(msgs) != 0 {
		t.Fatal("clear left messages")
	}
}

func TestCoachRejectsUnlistedActionsAndHidesVendorErrors(t *testing.T) {
	fc := &fakeCoach{call: []string{"suggest_gyst_page"}, args: map[string]string{"suggest_gyst_page": `{"destination":"https://evil.example"}`}, reply: "OK"}
	h := coachHandler(t, fc)
	ask(h, "Go somewhere", "")
	if fc.toolData["suggest_gyst_page"] == "" || !strings.HasPrefix(fc.toolData["suggest_gyst_page"], "ERR") {
		t.Fatal("unlisted destination accepted")
	}
	if page := cfgGet(t, h, "/health/coach"); strings.Contains(page, "evil.example") {
		t.Fatal("model-written destination rendered")
	}

	for _, tc := range []struct {
		err    error
		status int
		want   string
	}{
		{errors.New("vendor: prompt echo 134.5 kg sk-secret"), http.StatusBadGateway, "couldn&#39;t answer right now"},
		{healthcoach.ErrBusy, http.StatusTooManyRequests, "busy"},
		{healthcoach.ErrDeclined, http.StatusBadGateway, "can&#39;t help with that"},
	} {
		h := coachHandler(t, &fakeCoach{err: tc.err})
		w := ask(h, "Hi there", "")
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.want) {
			t.Fatalf("%v: %d", tc.err, w.Code)
		}
		if strings.Contains(w.Body.String(), "sk-secret") || strings.Contains(w.Body.String(), "prompt echo") {
			t.Fatal("vendor error text leaked")
		}
		if !strings.Contains(w.Body.String(), ">Hi there</textarea>") {
			t.Fatal("question not kept for retry")
		}
		if msgs, _ := h.Service.Repo.ListCoachMessages(40); len(msgs) != 0 {
			t.Fatal("failed exchange stored")
		}
	}
}

func TestCoachGuards(t *testing.T) {
	h := coachHandler(t, &fakeCoach{reply: "OK"})
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, h.Service.Location())
	h.Service.Clock.Now = func() time.Time { return now }
	for _, q := range []string{"", "   ", strings.Repeat("q", 1001)} {
		if w := ask(h, q, ""); w.Code != http.StatusBadRequest {
			t.Fatalf("%d chars: %d", len(q), w.Code)
		}
	}
	if w := ask(h, "Hi", "https://evil.example"); w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin: %d", w.Code)
	}
	if w := ask(h, "First", ""); w.Code != http.StatusSeeOther {
		t.Fatalf("first: %d", w.Code)
	}
	if w := ask(h, "Too soon", ""); w.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit: %d", w.Code)
	}
	for _, tc := range []struct{ method, path string }{{"GET", "/health/coach/clear"}, {"PUT", "/health/coach"}, {"DELETE", "/health/profile"}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s: %d", tc.method, tc.path, w.Code)
		}
	}
	for _, p := range []string{"/health/coach", "/health/coach/clear", "/health/profile"} {
		if authExemptPath(p) {
			t.Fatalf("%s must require a session", p)
		}
	}
}

func TestCoachProfile(t *testing.T) {
	h := coachHandler(t, nil)
	form := url.Values{"age": {"24"}, "sex": {"Male"}, "height_cm": {"181"}, "goal_weight_kg": {"110"}, "food_rules": {"Halal, <b>no</b> alcohol"}}
	if w := cfgPost(h, "/health/profile", form, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("save: %d", w.Code)
	}
	page := cfgGet(t, h, "/health/profile?saved=1")
	if !strings.Contains(page, `value="Halal, &lt;b&gt;no&lt;/b&gt; alcohol"`) || !strings.Contains(page, "Saved.") {
		t.Fatal("profile not saved or not escaped")
	}
	if p := h.coachProfile(); p["age"] != "24" || p["notes"] != "" {
		t.Fatalf("profile %+v", p)
	}
	if w := cfgPost(h, "/health/profile", url.Values{"notes": {strings.Repeat("n", 301)}}, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("long notes: %d", w.Code)
	}
}
