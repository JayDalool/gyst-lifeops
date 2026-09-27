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

func TestTrainingSyncRequiresSessionAndSameOrigin(t *testing.T) {
	h := testHandler(t)
	r := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(r, httptest.NewRequest("POST", "/health/training/sync", nil))
	if r.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated sync reached handler: %d", r.Code)
	}
	request := httptest.NewRequest("POST", "/health/training/sync", nil)
	request.Header.Set("Origin", "https://attacker.example")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	r = httptest.NewRecorder()
	h.ServeHTTP(r, request)
	if r.Code != 403 {
		t.Fatalf("cross-site sync: %d", r.Code)
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/health/training/sync", nil))
	if r.Code != 405 {
		t.Fatal("GET can sync")
	}
}
