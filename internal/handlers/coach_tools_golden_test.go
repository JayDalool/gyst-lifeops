package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// GYST-163 / Issue #236: the Coach's nine Health tools move behind an
// interface in internal/secretary. This golden file was written on the code
// as it was before the move; the test and the file are byte-identical in the
// move commit, so a pass proves the names, descriptions, schemas and JSON
// output did not change. Fictional values only (RULES 3.3-3.4).

var updateCoachGolden = flag.Bool("update-coach-golden", false, "rewrite testdata/coach_tools.golden.json")

const coachGoldenPath = "testdata/coach_tools.golden.json"

// goldenClock pins both the instant and the zone so the output never depends
// on the host or on the day the test runs.
func goldenClock() clock.Clock {
	loc, err := clock.Load("America/Winnipeg")
	if err != nil {
		panic(err)
	}
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, loc)
	return clock.Clock{Now: func() time.Time { return now }, Loc: loc}
}

func goldenHandler(t *testing.T) (*Handler, func()) {
	t.Helper()
	database := migratedTestDB(t)
	h := New(services.New(repository.New(database)), "../../web/templates")
	h.Service.Clock = goldenClock()
	return h, func() { database.Close() }
}

func goldenSeed(t *testing.T, h *Handler) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	now := h.Service.Now()
	date := h.Service.TodayString()
	cycle := models.HealthCycle{Title: "Autumn <plan> & más", PrimaryGoal: "Consistency", StartDate: "2026-10-01", EndDate: "2027-03-31",
		DailyCalorieTarget: 2300, ProteinTargetG: 180, CarbsTargetG: 250, FatTargetG: 70, FiberTargetG: 30, TargetDeficitKcal: 300,
		StrengthSessionsPerWeek: 3, CardioSessionsPerWeek: 2, StepTarget: 8000, WaterTargetML: 3000, WaterBottleML: 1500}
	must(h.Service.CreateHealthCycle(&cycle))
	must(h.Service.Repo.CreateHealthReferenceTarget(&models.HealthReferenceTarget{CycleID: &cycle.ID, SourceLabel: "Example <dietitian>", SourceKind: "dietitian", IssuedOn: "2026-10-02", Kcal: viewFloat(2100), ProteinG: viewFloat(150)}))
	must(h.Service.Repo.CreateHealthReferenceTarget(&models.HealthReferenceTarget{SourceLabel: "Example device", SourceKind: "device", Kcal: viewFloat(2000)}))
	_, err := h.Service.SyncHealth(models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{
		DietaryCalories: viewFloat(1450), ProteinG: viewFloat(96), CarbsG: viewFloat(150), FatG: viewFloat(40), StepsWatch: viewInt(5200), ActiveEnergyKcal: viewFloat(310), RestingEnergyKcal: viewFloat(1100), ExerciseMinutes: viewFloat(25)}})
	must(err)
	for i := 0; i < 100; i++ {
		at := now.AddDate(0, 0, -i).Add(-2 * time.Hour)
		must(h.Service.CreateHealthWeightEntry(&models.HealthWeightEntry{MeasuredAt: at.Format(time.RFC3339Nano), WeightKg: 82 + float64(i%9)*0.1, Source: "manual"}))
	}
	_, _, err = h.Service.Repo.HealthWaterDay(date)
	must(err)
	_, err = h.Service.Repo.AddHealthWater(date, 1500, 3000)
	must(err)
	for _, f := range []models.HealthFood{
		{Name: "Crème <fraîche> & miel", Serving: "100 g", Kcal: viewFloat(250), ProteinG: viewFloat(3), Notes: "naïve <b>note</b> & 日本語", Active: true},
		{Name: "Plain skyr", Serving: "150 g", ProteinG: viewFloat(16), Active: true},
	} {
		f := f
		must(h.Service.Repo.CreateHealthFood(&f))
	}
	_, err = h.Service.AddHealthChecklistItem("Stretch <5 min>", "habit")
	must(err)
	_, err = h.Service.AddHealthChecklistItem("Protein target", "protein")
	must(err)
	f := viewFloat
	entries := []models.TrainingEntry{
		{ID: "w1", Kind: "workout", Title: "Push <day> & core", StartedAt: now.AddDate(0, 0, -1), EndedAt: now.AddDate(0, 0, -1).Add(55 * time.Minute), Exercises: []models.TrainingExercise{
			{Name: "Bench Press", PrimaryMuscle: "chest", Sets: []models.TrainingSet{{Kind: "warmup", LoadKG: f(40), Reps: f(12)}, {Kind: "normal", LoadKG: f(80), Reps: f(8)}, {Kind: "failure", LoadKG: f(82.5), Reps: f(6)}}},
			{Name: "Treadmill", PrimaryMuscle: "cardio", Sets: []models.TrainingSet{{Kind: "normal", Seconds: f(1200), InclinePct: f(8), SpeedKmh: f(5)}}},
		}},
		{ID: "w2", Kind: "workout", Title: "Legs", StartedAt: now.AddDate(0, 0, -3), Exercises: []models.TrainingExercise{
			{Name: "Squat", PrimaryMuscle: "quadriceps", Sets: []models.TrainingSet{{Kind: "normal", LoadKG: f(100), Reps: f(5)}, {Kind: "dropset", LoadKG: f(80), Reps: f(10)}}},
			{Name: "Bike", PrimaryMuscle: "cardio", Sets: []models.TrainingSet{{Kind: "normal", Seconds: f(600)}}},
		}},
		{ID: "w3", Kind: "workout", Title: "Push old", StartedAt: now.AddDate(0, 0, -20), EndedAt: now.AddDate(0, 0, -20).Add(50 * time.Minute), Exercises: []models.TrainingExercise{
			{Name: "Bench Press", PrimaryMuscle: "chest", Sets: []models.TrainingSet{{Kind: "normal", LoadKG: f(70), Reps: f(8)}}},
			{Name: "Squat", PrimaryMuscle: "quadriceps", Sets: []models.TrainingSet{{Kind: "normal", LoadKG: f(90), Reps: f(5)}}},
		}},
		{ID: "r1", Kind: "routine", Title: "Routine <A>", Exercises: []models.TrainingExercise{{Name: "Bench Press"}, {Name: "Squat"}}},
	}
	for i := 4; i <= 9; i++ {
		entries = append(entries, models.TrainingEntry{ID: fmt.Sprintf("x%d", i), Kind: "workout", Title: fmt.Sprintf("Extra %d", i), StartedAt: now.AddDate(0, 0, -i*4), Exercises: []models.TrainingExercise{{Name: "Row", PrimaryMuscle: "lats", Sets: []models.TrainingSet{{Kind: "normal", LoadKG: f(50), Reps: f(10)}}}}})
	}
	must(h.Service.Repo.SaveTraining("hevy", entries, now.Add(-30*time.Minute)))
}

type goldenCall struct {
	Input  string `json:"input"`
	Output any    `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
	// Chosen is the page suggest_gyst_page stored for the handler.
	Chosen string `json:"chosen,omitempty"`
}

var goldenInputs = map[string][]string{
	"get_recent_workouts":   {``, `null`, `{}`, `{`, `{"count":0}`, `{"count":1}`, `{"count":3}`, `{"count":5}`, `{"count":99}`, `{"count":-2}`, `{"count":1.5}`, `{"count":"3"}`, `{"count":null}`, `[]`},
	"get_exercise_progress": {``, `null`, `{}`, `{`, `{"weeks":0}`, `{"weeks":4}`, `{"weeks":7}`, `{"weeks":12}`, `{"weeks":26}`, `{"weeks":-4}`, `{"weeks":4.5}`, `{"weeks":"26"}`, `{"weeks":null}`, `[]`},
}

// record runs every tool and returns what the model would see. Output is
// stored as the raw string (not re-parsed) so key order and number text count.
func goldenRecord(h *Handler, data HealthData) map[string]any {
	var action string
	tools := h.coachTools(data, &action)
	out := map[string]any{}
	for _, tool := range tools {
		if tool.Name == "suggest_gyst_page" {
			continue
		}
		inputs := goldenInputs[tool.Name]
		if inputs == nil {
			inputs = []string{`{}`, ``, `null`}
		}
		var calls []goldenCall
		for _, in := range inputs {
			c := goldenCall{Input: in}
			got, err := tool.Run(context.Background(), json.RawMessage(in))
			if err != nil {
				c.Error = err.Error()
			} else {
				c.Output = got
			}
			calls = append(calls, c)
		}
		out[tool.Name] = calls
	}
	return out
}

func TestCoachToolsGolden(t *testing.T) {
	doc := map[string]any{}

	// Schemas, order and the suggest_gyst_page tool come from a seeded handler.
	h, closeDB := goldenHandler(t)
	goldenSeed(t, h)
	rec := httptest.NewRecorder()
	data, ok := h.healthBase(rec)
	if !ok {
		t.Fatal("healthBase")
	}
	var action string
	tools := h.coachTools(data, &action)
	order := []string{}
	schemas := map[string]any{}
	for _, tool := range tools {
		order = append(order, tool.Name)
		props := tool.Properties
		if tool.Name == "suggest_gyst_page" {
			// The enum is built from map iteration in healthcoach.ActionTool, so
			// its order changes between runs; the sorted list is what is stable.
			// Fixing the order is a separate follow-up (GYST-163 plan review).
			dest := props["destination"].(map[string]any)
			enum := append([]string(nil), dest["enum"].([]string)...)
			sort.Strings(enum)
			props = map[string]any{"destination": map[string]any{"type": dest["type"], "enum": enum}}
		}
		schemas[tool.Name] = map[string]any{"name": tool.Name, "description": tool.Description, "properties": props, "required": tool.Required}
	}
	doc["tool_order"] = order
	doc["schemas"] = schemas

	var suggest []goldenCall
	for _, in := range []string{`{"destination":"health_today"}`, `{"destination":"health_workout_progress"}`, `{"destination":"https://evil.example"}`, `{"destination":""}`, `{}`, `{`, ``, `null`} {
		tool := tools[len(tools)-1]
		c := goldenCall{Input: in}
		action = ""
		got, err := tool.Run(context.Background(), json.RawMessage(in))
		c.Chosen = action
		if err != nil {
			c.Error = err.Error()
		} else {
			c.Output = got
		}
		suggest = append(suggest, c)
	}
	doc["suggest_gyst_page_runs"] = suggest

	fixtures := map[string]any{}
	fixtures["seeded"] = goldenRecord(h, data)
	// The data was read before the database closes, as in a request; only the
	// tools that read the database again fail.
	closeDB()
	fixtures["unavailable"] = goldenRecord(h, data)

	empty, closeEmpty := goldenHandler(t)
	defer closeEmpty()
	emptyData, ok := empty.healthBase(httptest.NewRecorder())
	if !ok {
		t.Fatal("healthBase on empty")
	}
	fixtures["empty"] = goldenRecord(empty, emptyData)
	doc["fixtures"] = fixtures

	got, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(bytes.TrimRight(got, "\n"), '\n')
	if *updateCoachGolden {
		if err := os.MkdirAll(filepath.Dir(coachGoldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(coachGoldenPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(coachGoldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Coach tools differ from %s; run with -update-coach-golden only if the change is intended", coachGoldenPath)
	}
}
