package handlers

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

const healthTestToken = "test-only-health-token"

func healthRequest(h *Handler, method, path, body, token, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
func TestHealthAPIAuthenticationAndValidation(t *testing.T) {
	h := testHandler(t)
	h.HealthSyncToken = healthTestToken
	h.Service.Now = func() time.Time { return time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC) }
	valid := `{"date":"2026-09-10","source":"apple_shortcuts","dietary_calories":130,"protein_g":8}`
	for _, test := range []struct {
		name, method, path, body, token, media string
		status                                 int
	}{
		{"missing auth", "POST", "/api/health/sync", valid, "", "application/json", 401},
		{"wrong auth", "POST", "/api/health/sync", valid, "wrong-test-token", "application/json", 401},
		{"unauthenticated", "GET", "/api/health/sync", "", "", "", 401},
		{"valid", "POST", "/api/health/sync", valid, healthTestToken, "application/json", 200},
		{"media", "POST", "/api/health/sync", valid, healthTestToken, "text/plain", 415},
		{"missing media", "POST", "/api/health/sync", valid, healthTestToken, "", 415},
		{"malformed", "POST", "/api/health/sync", "{", healthTestToken, "application/json", 400},
		{"array", "POST", "/api/health/sync", "[]", healthTestToken, "application/json", 400},
		{"null", "POST", "/api/health/sync", "null", healthTestToken, "application/json", 400},
		{"trailing object", "POST", "/api/health/sync", valid + " {}", healthTestToken, "application/json", 400},
		{"unknown field", "POST", "/api/health/sync", `{"date":"2026-09-10","source":"apple_shortcuts","dietary_calories":1,"workout_calories":4}`, healthTestToken, "application/json", 400},
		{"invalid date", "POST", "/api/health/sync", strings.Replace(valid, "2026-09-10", "2026-02-30", 1), healthTestToken, "application/json", 400},
		{"source", "POST", "/api/health/sync", strings.Replace(valid, "apple_shortcuts", "hevy", 1), healthTestToken, "application/json", 400},
		{"empty metrics", "POST", "/api/health/sync", `{"date":"2026-09-10","source":"apple_shortcuts"}`, healthTestToken, "application/json", 400},
		{"huge number", "POST", "/api/health/sync", strings.Replace(valid, ":130", ":1e309", 1), healthTestToken, "application/json", 400},
		{"oversize", "POST", "/api/health/sync", `{"date":"` + strings.Repeat("x", 17<<10) + `"}`, healthTestToken, "application/json", 413},
		{"oversize trailing whitespace", "POST", "/api/health/sync", valid + strings.Repeat(" ", 17<<10), healthTestToken, "application/json", 413},
		{"sync method", "GET", "/api/health/sync", "", healthTestToken, "", 405},
		{"read route removed", "GET", "/api/health/today", "", healthTestToken, "", 404},
		{"read route removed for POST", "POST", "/api/health/today", valid, healthTestToken, "application/json", 404},
		{"unknown route", "GET", "/api/health/unknown", "", healthTestToken, "", 404},
		{"nested route", "GET", "/api/health/today/extra", "", healthTestToken, "", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := healthRequest(h, test.method, test.path, test.body, test.token, test.media)
			if rec.Code != test.status {
				t.Fatalf("status %d want %d: %s", rec.Code, test.status, rec.Body.String())
			}
			if !json.Valid(rec.Body.Bytes()) || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatal("non-JSON response")
			}
			if rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("private response is cacheable")
			}
			if strings.Contains(rec.Body.String(), healthTestToken) {
				t.Fatal("token in response")
			}
		})
	}
	for _, metric := range []string{"dietary_calories", "protein_g", "carbs_g", "fat_g", "fiber_g", "steps_watch", "steps_phone", "active_energy_kcal", "resting_energy_kcal", "exercise_minutes"} {
		body := fmt.Sprintf(`{"date":"2026-09-10","source":"apple_shortcuts","%s":-1}`, metric)
		if rec := healthRequest(h, "POST", "/api/health/sync", body, healthTestToken, "application/json"); rec.Code != 400 {
			t.Fatalf("negative %s accepted: %d", metric, rec.Code)
		}
	}
	h.HealthSyncToken = ""
	for _, path := range []string{"/api/health/sync", "/api/health/unknown"} {
		rec := healthRequest(h, "POST", path, valid, healthTestToken, "application/json")
		if rec.Code != 503 {
			t.Fatalf("unset token: %d", rec.Code)
		}
	}
}
func TestHealthAPIPartialSyncAndToday(t *testing.T) {
	h := testHandler(t)
	h.HealthSyncToken = healthTestToken
	h.Service.Now = func() time.Time { return time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC) }
	// The read endpoint is gone, so the merge semantics below are asserted
	// against the service directly - which is what the server-rendered
	// dashboard will call in M4. The coverage this test exists for, partial
	// merge and NULL-versus-zero, is unchanged.
	today, err := h.Service.HealthToday()
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if today.Snapshot != nil || today.HealthCycle != nil || today.Observed.ObservedTotalBurn != nil {
		t.Fatalf("empty: %+v", today)
	}
	c := models.HealthCycle{Title: "Cycle", PrimaryGoal: "Consistency", StartDate: "2026-09-10", EndDate: "2026-12-03", DailyCalorieTarget: 2300}
	if err := h.Service.CreateHealthCycle(&c); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		`{"date":"2026-09-10","source":"apple_shortcuts","dietary_calories":130,"protein_g":8}`,
		`{"date":"2026-09-10","source":"apple_shortcuts","active_energy_kcal":211,"resting_energy_kcal":1318.596,"steps_watch":1195,"steps_phone":1302}`,
		`{"date":"2026-09-10","source":"apple_shortcuts","protein_g":0,"dietary_calories":null}`,
	} {
		for i := 0; i < 2; i++ {
			if rec := healthRequest(h, "POST", "/api/health/sync", body, healthTestToken, "application/json; charset=utf-8"); rec.Code != 200 {
				t.Fatalf("sync: %s", rec.Body.String())
			}
		}
	}
	today, err = h.Service.HealthToday()
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if today.Date != "2026-09-10" || today.Snapshot == nil || today.HealthCycle == nil {
		t.Fatalf("today: %+v", today)
	}
	s := today.Snapshot
	if *s.DietaryCalories != 130 || *s.ProteinG != 0 || s.FiberG != nil || *s.StepsWatch != 1195 || *s.StepsPhone != 1302 || *today.Observed.FoodRemaining != 2170 {
		t.Fatalf("today data: %+v", today)
	}
	// Health routing must not consume or authenticate the push routes.
	push := healthRequest(h, "POST", "/api/push/subscribe", `{"endpoint":"https://push.example/health-test","keys":{"p256dh":"test-key","auth":"test-auth"}}`, "", "application/json")
	if push.Code != 201 {
		t.Fatalf("push regression %d: %s", push.Code, push.Body.String())
	}
	healthPage := healthRequest(h, "GET", "/health", "", "", "")
	if healthPage.Code != 200 || strings.Contains(healthPage.Body.String(), healthTestToken) {
		t.Fatal("health page regression or token leak")
	}
}
func healthPlanSubmission() url.Values {
	return url.Values{"cycle_id": {"0"}, "title": {"Autumn plan"}, "start_date": {"2026-09-10"}, "end_date": {"2026-12-03"}, "primary_goal": {"Consistency"}, "daily_calorie_target": {"2300"}, "protein_target_g": {"180"}, "carbs_target_g": {"220"}, "fat_target_g": {"70"}, "fiber_target_g": {"30"}, "target_deficit_kcal": {"400"}, "strength_sessions_per_week": {"3"}, "cardio_sessions_per_week": {"2"}, "step_target": {"8000"}, "notes": {"Test-only plan"}}
}
func TestHealthPlanForm(t *testing.T) {
	h := testHandler(t)
	h.Service.Now = func() time.Time { return time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC) }
	rec := healthRequest(h, "GET", "/health/plan", "", "", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `name="daily_calorie_target"`) || !strings.Contains(rec.Body.String(), `value="2026-12-03"`) {
		t.Fatalf("plan GET: %d %s", rec.Code, rec.Body.String())
	}
	form := healthPlanSubmission()
	rec = healthRequest(h, "POST", "/health/plan", form.Encode(), "", "application/x-www-form-urlencoded")
	if rec.Code != 303 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	cycles, err := h.Service.ListHealthCycles()
	if err != nil || len(cycles) != 1 || cycles[0].DailyCalorieTarget != 2300 {
		t.Fatalf("persisted: %+v %v", cycles, err)
	}
	rec = healthRequest(h, "GET", "/health/plan", "", "", "")
	if !strings.Contains(rec.Body.String(), "Edit cycle") || !strings.Contains(rec.Body.String(), "Autumn plan") {
		t.Fatal("active cycle not editable")
	}
	// Creating the same period again fails and retains the form values.
	rec = healthRequest(h, "POST", "/health/plan", form.Encode(), "", "application/x-www-form-urlencoded")
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "overlaps") || !strings.Contains(rec.Body.String(), "Autumn plan") {
		t.Fatalf("overlap: %d %s", rec.Code, rec.Body.String())
	}
	form.Set("cycle_id", fmt.Sprint(cycles[0].ID))
	form.Set("daily_calorie_target", "2400")
	rec = healthRequest(h, "POST", "/health/plan", form.Encode(), "", "application/x-www-form-urlencoded")
	if rec.Code != 303 {
		t.Fatalf("edit: %d %s", rec.Code, rec.Body.String())
	}
	got, err := h.Service.GetHealthCycle(cycles[0].ID)
	if err != nil || got.DailyCalorieTarget != 2400 {
		t.Fatalf("edit not saved: %+v %v", got, err)
	}
	for _, invalid := range []struct{ field, value string }{{"end_date", "2026-09-01"}, {"daily_calorie_target", "-1"}, {"daily_calorie_target", "not-a-number"}, {"strength_sessions_per_week", "2.5"}} {
		bad := healthPlanSubmission()
		bad.Set(invalid.field, invalid.value)
		rec = healthRequest(h, "POST", "/health/plan", bad.Encode(), "", "application/x-www-form-urlencoded")
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), `role="alert"`) {
			t.Fatalf("invalid form %s: %d", invalid.field, rec.Code)
		}
	}
	req := httptest.NewRequest("POST", "/health/plan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://untrusted.example")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatal("cross-site write accepted")
	}
	if rec := healthRequest(h, "DELETE", "/health/plan", "", "", ""); rec.Code != 405 {
		t.Fatal("plan method accepted")
	}
	if rec := healthRequest(h, "GET", "/health/plan?id=99999", "", "", ""); rec.Code != 404 {
		t.Fatal("missing cycle not 404")
	}
}
