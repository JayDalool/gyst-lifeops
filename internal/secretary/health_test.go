package secretary

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// recordingHealth is a HealthReader that remembers the arguments it got.
type recordingHealth struct {
	count, weeks int
	fail         bool
}

func (r *recordingHealth) val(name string) (any, error) {
	if r.fail {
		return nil, errors.New("SECRET-CAUSE db path /data/x.db")
	}
	return map[string]string{"tool": name}, nil
}
func (r *recordingHealth) Today() (any, error)           { return r.val("today") }
func (r *recordingHealth) Plan() (any, error)            { return r.val("plan") }
func (r *recordingHealth) WeightTrend() (any, error)     { return r.val("weight") }
func (r *recordingHealth) TrainingSummary() (any, error) { return r.val("summary") }
func (r *recordingHealth) RecentWorkouts(n int) (any, error) {
	r.count = n
	return r.val("recent")
}
func (r *recordingHealth) ExerciseProgress(w int) (any, error) {
	r.weeks = w
	return r.val("progress")
}
func (r *recordingHealth) HevyRoutines() (any, error)   { return r.val("routines") }
func (r *recordingHealth) FoodProducts() (any, error)   { return r.val("foods") }
func (r *recordingHealth) DailyChecklist() (any, error) { return r.val("checklist") }

func TestHealthToolsNamesAndOrder(t *testing.T) {
	var got []string
	for _, tool := range HealthTools(&recordingHealth{}) {
		got = append(got, tool.Name)
		if tool.Description == "" || tool.Run == nil {
			t.Errorf("%s incomplete", tool.Name)
		}
	}
	want := "get_health_today,get_health_plan,get_weight_trend,get_training_summary,get_recent_workouts,get_exercise_progress,get_hevy_routines,get_food_products,get_daily_checklist"
	if strings.Join(got, ",") != want {
		t.Fatalf("names %v", got)
	}
	for _, s := range healthSpecs(&recordingHealth{}) {
		if len(s.Sources) != 0 {
			t.Errorf("%s has sources %v; Health tools read no synced copy", s.Name, s.Sources)
		}
	}
}

func TestHealthToolsClampInputs(t *testing.T) {
	for _, tc := range []struct {
		tool, in string
		count    int
		weeks    int
	}{
		{"get_recent_workouts", ``, 1, 0}, {"get_recent_workouts", `{"count":0}`, 1, 0}, {"get_recent_workouts", `{"count":3}`, 3, 0},
		{"get_recent_workouts", `{"count":99}`, 5, 0}, {"get_recent_workouts", `{"count":-4}`, 1, 0}, {"get_recent_workouts", `{`, 1, 0},
		{"get_exercise_progress", ``, 0, 12}, {"get_exercise_progress", `{"weeks":4}`, 0, 4}, {"get_exercise_progress", `{"weeks":7}`, 0, 12},
		{"get_exercise_progress", `{"weeks":26}`, 0, 26}, {"get_exercise_progress", `{"weeks":12}`, 0, 12}, {"get_exercise_progress", `{"weeks":"4"}`, 0, 12},
	} {
		r := &recordingHealth{}
		for _, tool := range HealthTools(r) {
			if tool.Name != tc.tool {
				continue
			}
			if _, err := tool.Run(context.Background(), json.RawMessage(tc.in)); err != nil {
				t.Fatal(err)
			}
		}
		if r.count != tc.count || r.weeks != tc.weeks {
			t.Errorf("%s %q: count %d weeks %d, want %d and %d", tc.tool, tc.in, r.count, r.weeks, tc.count, tc.weeks)
		}
	}
}

func TestHealthToolsHideTheCause(t *testing.T) {
	for _, tool := range HealthTools(&recordingHealth{fail: true}) {
		out, err := tool.Run(context.Background(), json.RawMessage(`{}`))
		if err == nil || err.Error() != "data unavailable" || out != "" {
			t.Errorf("%s: %q %v", tool.Name, out, err)
		}
	}
}

func TestHealthToolsEncodeCompactJSON(t *testing.T) {
	out, err := HealthTools(&recordingHealth{})[0].Run(context.Background(), nil)
	if err != nil || out != `{"tool":"today"}` {
		t.Fatalf("%q %v", out, err)
	}
}
