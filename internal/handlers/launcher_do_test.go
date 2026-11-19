package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

type doFixtureHevy struct{}

func (doFixtureHevy) Fetch(context.Context) ([]models.TrainingEntry, error) {
	return []models.TrainingEntry{{ID: "w1", Kind: "workout", Title: "Fictional session"}}, nil
}

type doBody struct {
	Commands []services.LauncherCommand `json:"commands"`
}

func doModeCommands(t *testing.T, h *Handler, query string) []services.LauncherCommand {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/search?mode=do&q="+query, nil))
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("do mode: %d %v", rec.Code, rec.Header())
	}
	var body doBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("bad JSON %q: %v", rec.Body.String(), err)
	}
	return body.Commands
}

func TestSearchAPIDoMode(t *testing.T) {
	h := authHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/search?mode=do", nil)
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", rec.Code)
	}
	cookie := signIn(t, h)
	req = httptest.NewRequest(http.MethodGet, "/api/search?mode=do&q=bottle", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("signed in: %d %v", rec.Code, rec.Header())
	}
	var body doBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Commands) != 1 || body.Commands[0].ID != "water" {
		t.Fatalf("filtered body %q: %v", rec.Body.String(), err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/search?mode=do", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET" {
		t.Fatalf("POST: %d %v", rec.Code, rec.Header())
	}
	for _, c := range doModeCommands(t, h, "") {
		if c.ID == "sync-hevy" {
			t.Fatal("Hevy offered without Training")
		}
	}
	h.Service.Training = &services.TrainingSync{Provider: doFixtureHevy{}}
	found := false
	for _, c := range doModeCommands(t, h, "hevy") {
		found = found || c.ID == "sync-hevy"
	}
	if !found {
		t.Fatal("Hevy not offered when configured")
	}
	if got := doModeCommands(t, h, "zzzz"); len(got) != 0 {
		t.Fatalf("no match: %+v", got)
	}
	if b, r := doSearch(t, h, "", "today"); r.Code != 200 || len(b.Groups) == 0 {
		t.Fatalf("go mode changed: %d", r.Code)
	}
}

func TestLauncherDoPostsReachExistingHandlers(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)
	if err := h.Service.Repo.CreateTask(&models.Task{Title: "Fictional errand", Status: "pending", Priority: "medium", Category: "personal", RepeatType: "none", ProofType: "none", DueDate: h.Service.TodayString()}); err != nil {
		t.Fatal(err)
	}
	cycle := models.HealthCycle{Title: "Fictional plan", PrimaryGoal: "Consistency", StartDate: "2000-01-01", EndDate: "2099-12-31", WaterBottleML: 500, WaterTargetML: 2000}
	if err := h.Service.CreateHealthCycle(&cycle); err != nil {
		t.Fatal(err)
	}
	h.Service.Training = &services.TrainingSync{Provider: doFixtureHevy{}}
	cmds, err := h.Service.LauncherCommands(true)
	if err != nil {
		t.Fatal(err)
	}
	posts := 0
	for _, c := range cmds {
		if c.Kind != services.LauncherKindPost {
			continue
		}
		posts++
		if c.DisabledReason != "" {
			t.Fatalf("%s unexpectedly disabled: %s", c.ID, c.DisabledReason)
		}
		form := url.Values{}
		for k, v := range c.Fields {
			form.Set(k, v)
		}
		send := func(origin string) *httptest.ResponseRecorder {
			req := httptest.NewRequest(http.MethodPost, c.Action, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Host = "gyst.example"
			req.Header.Set("Origin", origin)
			req.AddCookie(cookie)
			rec := httptest.NewRecorder()
			h.Authenticated().ServeHTTP(rec, req)
			return rec
		}
		if rec := send("https://evil.example"); rec.Code != http.StatusForbidden {
			t.Errorf("%s cross-site POST: %d, want 403", c.ID, rec.Code)
		}
		if rec := send("https://gyst.example"); rec.Code != http.StatusSeeOther {
			t.Errorf("%s same-origin POST: %d, want 303: %s", c.ID, rec.Code, rec.Body.String())
		}
	}
	if posts != 3 {
		t.Fatalf("expected 3 POST commands, got %d", posts)
	}
}
