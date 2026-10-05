package hevy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestImportPagesAndNullableSets(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("api-key") != "test-only-key" {
			t.Error("missing key")
		}
		switch r.URL.Path {
		case "/v1/exercise_templates":
			fmt.Fprint(w, `{"page":1,"page_count":1,"exercise_templates":[{"id":"bench","primary_muscle_group":"chest","secondary_muscle_groups":["triceps"]}]}`)
		case "/v1/routines":
			fmt.Fprint(w, `{"page":1,"page_count":1,"routines":[{"id":"r1","title":"Push","exercises":[{"title":"Bench","exercise_template_id":"bench","sets":[{"type":"normal","weight_kg":0,"rep_range":{"start":8,"end":12}},{"type":"warmup"}]}]}]}`)
		case "/v1/workouts":
			// Page 1 has a valid end time; page 2 ends before it starts, which must stay unknown.
			end := map[string]string{"1": "2026-09-23T12:45:00Z", "2": "2026-09-23T11:00:00Z"}[r.URL.Query().Get("page")]
			fmt.Fprintf(w, `{"page":%s,"page_count":2,"workouts":[{"id":"w%s","title":"Session","start_time":"2026-09-23T12:00:00Z","end_time":"%s","exercises":[]}]}`, r.URL.Query().Get("page"), r.URL.Query().Get("page"), end)
		default:
			t.Error("unexpected endpoint")
		}
	}))
	defer server.Close()
	c := New("test-only-key")
	c.base = server.URL
	entries, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 4 || len(entries) != 3 {
		t.Fatalf("calls=%d entries=%d", calls, len(entries))
	}
	if got := entries[1].EndedAt.Sub(entries[1].StartedAt); got != 45*time.Minute {
		t.Fatalf("workout duration %v", got)
	}
	if !entries[2].EndedAt.IsZero() || !entries[0].EndedAt.IsZero() {
		t.Fatal("invalid or routine end time stored")
	}
	ex := entries[0].Exercises[0]
	if ex.PrimaryMuscle != "chest" || ex.SecondaryMuscles[0] != "triceps" {
		t.Fatal(ex)
	}
	s := ex.Sets[0]
	if s.LoadKG == nil || *s.LoadKG != 0 || s.Reps != nil || *s.MinReps != 8 || *s.MaxReps != 12 || ex.Sets[1].LoadKG != nil {
		t.Fatal("nullable/range normalization failed")
	}
}
func TestFailuresDoNotLeakKeyOrFollowRedirect(t *testing.T) {
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer target.Close()
	for _, status := range []int{301, 401, 403, 429, 500, 200} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", target.URL)
				w.WriteHeader(status)
				fmt.Fprint(w, "private-key-and-invalid-json")
			}))
			defer server.Close()
			c := New("private-key")
			c.base = server.URL
			_, err := c.Fetch(context.Background())
			if err == nil || strings.Contains(err.Error(), "private-key") {
				t.Fatalf("unsafe error %v", err)
			}
		})
	}
	if reached {
		t.Fatal("followed redirect with API key")
	}
}
