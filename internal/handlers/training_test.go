package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTrainingRoute(t *testing.T) {
	h := testHandler(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health/training", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("GET: %d %s", r.Code, r.Body.String())
	}
	for _, want := range []string{"Workout history unavailable", "Hevy · Not connected", "gym-day check-ins", `action="/trackers/gym/toggle"`, `href="/health/training" aria-current="page"`} {
		if !strings.Contains(r.Body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if count := strings.Count(r.Body.String(), `name="log_date"`); count != 7 {
		t.Errorf("want 7 gym days, got %d", count)
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/health/training", nil))
	if r.Code != http.StatusMethodNotAllowed || r.Header().Get("Allow") != "GET" {
		t.Fatalf("POST: %d", r.Code)
	}
	if authExemptPath("/health/training") {
		t.Fatal("training must require a session")
	}
}
