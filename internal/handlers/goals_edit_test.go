package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func newEditGoal(t *testing.T, h *Handler) int64 {
	t.Helper()
	g := models.Goal{Title: "Walk", Type: "yearly", Category: "health", TargetValue: 100, Unit: "walks", StartDate: "2026-01-01", EndDate: "2026-12-31", Status: "active"}
	if err := h.Service.Repo.CreateGoal(&g, "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	if err := h.Service.Repo.AddGoalProgress(g.ID, "2026-02-01", 20, ""); err != nil {
		t.Fatal(err)
	}
	return g.ID
}

func goalForm() url.Values {
	return url.Values{"title": {"Walk more"}, "description": {"daily"}, "type": {"monthly"}, "category": {"coding"},
		"target_value": {"31"}, "unit": {"km"}, "start_date": {"2026-03-01"}, "end_date": {"2026-03-31"}, "status": {"completed"}}
}

func doGoal(h http.Handler, method, path string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGoalEditGetIsPrefilled(t *testing.T) {
	h := testHandler(t)
	newEditGoal(t, h)
	rec := doGoal(h, "GET", "/goals/1/edit", nil)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Edit goal", `value="Walk"`, `value="2026-12-31"`, "Current progress", "Log progress to change it", "/goals/1/edit"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, `name="current_value"`) || strings.Contains(body, `value="archived"`) {
		t.Error("edit form must not offer current_value or the archived status")
	}
}

func TestGoalEditPostSavesAndRedirects(t *testing.T) {
	h := testHandler(t)
	id := newEditGoal(t, h)
	rec := doGoal(h, "POST", "/goals/1/edit", goalForm())
	if rec.Code != 303 || rec.Header().Get("Location") != "/goals/1" {
		t.Fatalf("status %d loc %q", rec.Code, rec.Header().Get("Location"))
	}
	g, _ := h.Service.Repo.GoalByID(id)
	if g.Title != "Walk more" || g.Type != "monthly" || g.Category != "coding" || g.TargetValue != 31 || g.Unit != "km" ||
		g.StartDate != "2026-03-01" || g.EndDate != "2026-03-31" || g.Status != "completed" || g.Description != "daily" {
		t.Fatalf("not saved: %+v", g)
	}
	if g.CurrentValue != 20 {
		t.Fatalf("current_value = %v, want 20 (unchanged, from latest entry)", g.CurrentValue)
	}
	// A forged current_value in the form is ignored.
	f := goalForm()
	f.Set("current_value", "9999")
	doGoal(h, "POST", "/goals/1/edit", f)
	if g, _ := h.Service.Repo.GoalByID(id); g.CurrentValue != 20 {
		t.Fatalf("current_value forged to %v", g.CurrentValue)
	}
	// #135 invariant after an edit: a backdated entry does not overwrite the newer value.
	if err := h.Service.Repo.AddGoalProgress(id, "2026-01-05", 99, ""); err != nil {
		t.Fatal(err)
	}
	if g, _ := h.Service.Repo.GoalByID(id); g.CurrentValue != 20 {
		t.Fatalf("after edit, backdated entry set current to %v, want 20", g.CurrentValue)
	}
}

func TestGoalEditNotFoundAndMethods(t *testing.T) {
	h := testHandler(t)
	id := newEditGoal(t, h)
	for _, m := range []string{"GET", "POST"} {
		if rec := doGoal(h, m, "/goals/999/edit", goalForm()); rec.Code != 404 {
			t.Errorf("%s unknown id: %d, want 404", m, rec.Code)
		}
	}
	for _, m := range []string{"PUT", "DELETE", "PATCH"} {
		if rec := doGoal(h, m, "/goals/1/edit", goalForm()); rec.Code != 405 {
			t.Errorf("%s: %d, want 405", m, rec.Code)
		}
	}
	if err := h.Service.Repo.ArchiveGoal(id); err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"GET", "POST"} {
		if rec := doGoal(h, m, "/goals/1/edit", goalForm()); rec.Code != 404 {
			t.Errorf("%s archived: %d, want 404", m, rec.Code)
		}
	}
	if g, _ := h.Service.Repo.GoalByID(id); g.Title != "Walk" {
		t.Error("archived goal was written")
	}
}

func TestGoalEditInvalidValuesReturn400AndWriteNothing(t *testing.T) {
	cases := map[string]func(url.Values){
		"empty title":     func(f url.Values) { f.Set("title", "  ") },
		"type daily":      func(f url.Values) { f.Set("type", "daily") },
		"bad category":    func(f url.Values) { f.Set("category", "fun") },
		"status archived": func(f url.Values) { f.Set("status", "archived") },
		"status junk":     func(f url.Values) { f.Set("status", "paused") },
		"target -1":       func(f url.Values) { f.Set("target_value", "-1") },
		"target NaN":      func(f url.Values) { f.Set("target_value", "NaN") },
		"target Inf":      func(f url.Values) { f.Set("target_value", "Inf") },
		"target text":     func(f url.Values) { f.Set("target_value", "lots") },
		"start 02-30":     func(f url.Values) { f.Set("start_date", "2026-02-30") },
		"end bad":         func(f url.Values) { f.Set("end_date", "2026-3-1") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			h := testHandler(t)
			id := newEditGoal(t, h)
			before, _ := h.Service.Repo.GoalByID(id)
			f := goalForm()
			f.Set("description", "typed text kept")
			mutate(f)
			rec := doGoal(h, "POST", "/goals/1/edit", f)
			if rec.Code != 400 {
				t.Fatalf("status %d, want 400", rec.Code)
			}
			body := rec.Body.String()
			if !strings.Contains(body, `role="alert"`) || !strings.Contains(body, "typed text kept") {
				t.Error("error not shown or typed values lost")
			}
			after, _ := h.Service.Repo.GoalByID(id)
			if after != before {
				t.Fatalf("goal written despite 400: %+v", after)
			}
		})
	}
}

func TestGoalEditPostIsGuarded(t *testing.T) {
	h := authHandler(t)
	newEditGoal(t, h)
	cookie := signIn(t, h)
	post := func(mutate func(*http.Request)) int {
		req := httptest.NewRequest("POST", "/goals/1/edit", strings.NewReader(goalForm().Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		mutate(req)
		rec := httptest.NewRecorder()
		h.Authenticated().ServeHTTP(rec, req)
		return rec.Code
	}
	unchanged := func(when string) {
		t.Helper()
		if g, _ := h.Service.Repo.GoalByID(1); g.Title != "Walk" {
			t.Fatalf("%s changed the goal", when)
		}
	}
	if code := post(func(r *http.Request) {}); code != http.StatusUnauthorized {
		t.Errorf("no session: %d, want 401", code)
	}
	unchanged("unauthenticated POST")
	if code := post(func(r *http.Request) { r.AddCookie(cookie); r.Header.Set("Sec-Fetch-Site", "cross-site") }); code != 403 {
		t.Errorf("cross-site: %d, want 403", code)
	}
	if code := post(func(r *http.Request) { r.AddCookie(cookie); r.Header.Set("Origin", "https://evil.example") }); code != 403 {
		t.Errorf("foreign origin: %d, want 403", code)
	}
	unchanged("rejected POST")
	if code := post(func(r *http.Request) { r.AddCookie(cookie); r.Header.Set("Sec-Fetch-Site", "same-origin") }); code != 303 {
		t.Errorf("signed-in same-origin: %d, want 303", code)
	}
}

func TestGoalsListRenders(t *testing.T) {
	h := testHandler(t)
	body := doGoal(h, "GET", "/goals", nil).Body.String()
	if !strings.Contains(body, "No goals") || !strings.Contains(body, "/goals/new") {
		t.Error("empty state missing")
	}

	newEditGoal(t, h)
	done := models.Goal{Title: "Finished thing", Type: "weekly", Category: "other", TargetValue: 5, Unit: "x", Status: "active"}
	if err := h.Service.Repo.CreateGoal(&done, "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	if err := h.Service.Repo.AddGoalProgress(done.ID, "2026-01-02", 5, ""); err != nil {
		t.Fatal(err)
	}
	rec := doGoal(h, "GET", "/goals", nil)
	body = rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "Walk") || !strings.Contains(body, "Completed (1)") || !strings.Contains(body, "<details") {
		t.Fatalf("list: status %d, completed section missing", rec.Code)
	}
	if strings.Index(body, "Finished thing") < strings.Index(body, "Completed (1)") {
		t.Error("completed goal not inside the Completed section")
	}
	if strings.Contains(body, "--xp") || strings.Contains(body, "violet") {
		t.Error("list must not use xp or violet styling")
	}
	detail := doGoal(h, "GET", "/goals/1", nil).Body.String()
	if !strings.Contains(detail, `href="/goals/1/edit"`) || !strings.Contains(detail, `id="log"`) {
		t.Error("detail edit link or #log anchor missing")
	}
}
