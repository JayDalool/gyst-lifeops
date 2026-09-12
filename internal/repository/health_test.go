package repository

import (
	"database/sql"
	"errors"
	"testing"

	"lifeops-tracker/internal/models"
)

func healthFloat(v float64) *float64 { return &v }
func healthInt(v int64) *int64       { return &v }
func TestHealthSnapshotPartialIdempotentAndZero(t *testing.T) {
	r := newTestRepo(t)
	if _, err := r.GetDailyHealthSnapshot("2026-09-10"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing snapshot: %v", err)
	}
	first := models.DailyHealthSnapshot{Date: "2026-09-10", HealthMetrics: models.HealthMetrics{DietaryCalories: healthFloat(130), ProteinG: healthFloat(8), StepsWatch: healthInt(1195), StepsPhone: healthInt(1302)}, LastSyncedAt: "2026-09-10T12:00:00Z"}
	for i := 0; i < 2; i++ {
		if err := r.UpsertDailyHealthSnapshot(first); err != nil {
			t.Fatal(err)
		}
	}
	second := models.DailyHealthSnapshot{Date: first.Date, HealthMetrics: models.HealthMetrics{ActiveEnergyKcal: healthFloat(211), RestingEnergyKcal: healthFloat(1318.596), ProteinG: healthFloat(0)}, LastSyncedAt: "2026-09-10T14:00:00Z"}
	merged, err := r.ApplyHealthSync(second, "apple_shortcuts")
	if err != nil {
		t.Fatal(err)
	}
	if *merged.DietaryCalories != 130 || *merged.ProteinG != 0 || *merged.ActiveEnergyKcal != 211 || *merged.RestingEnergyKcal != 1318.596 {
		t.Fatalf("partial merge: %+v", merged)
	}
	if merged.FiberG != nil || merged.ExerciseMinutes != nil || merged.CarbsG != nil {
		t.Fatal("missing metric became zero")
	}
	if *merged.StepsWatch != 1195 || *merged.StepsPhone != 1302 {
		t.Fatal("steps were merged or overwritten")
	}
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM daily_health_snapshots`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("snapshot count %d: %v", count, err)
	}
	state, err := r.GetHealthSyncState("apple_shortcuts")
	if err != nil || state.LastSyncedAt != second.LastSyncedAt {
		t.Fatalf("sync state: %+v %v", state, err)
	}
	// A failing sync-state write must roll back the snapshot write too.
	if _, err := r.db.Exec(`CREATE TRIGGER test_reject_sync BEFORE UPDATE ON health_sync_states BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	second.DietaryCalories = healthFloat(999)
	if _, err := r.ApplyHealthSync(second, "apple_shortcuts"); err == nil {
		t.Fatal("expected transaction failure")
	}
	after, err := r.GetDailyHealthSnapshot(first.Date)
	if err != nil || *after.DietaryCalories != 130 {
		t.Fatalf("snapshot not rolled back: %+v %v", after, err)
	}
}
func TestHealthCycleRepositoryAndWeight(t *testing.T) {
	r := newTestRepo(t)
	c := models.HealthCycle{Title: "Cycle", StartDate: "2026-09-10", EndDate: "2026-12-03", PrimaryGoal: "Consistency", DailyCalorieTarget: 2300}
	if err := r.CreateHealthCycle(&c); err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{c.StartDate, "2026-10-01", c.EndDate} {
		got, err := r.GetHealthCycleForDate(date)
		if err != nil || got.ID != c.ID {
			t.Fatalf("cycle for %s: %+v %v", date, got, err)
		}
	}
	if _, err := r.GetHealthCycleForDate("2026-12-04"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("outside cycle: %v", err)
	}
	c.DailyCalorieTarget = 2400
	if err := r.UpdateHealthCycle(&c); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetHealthCycle(c.ID)
	if err != nil || got.DailyCalorieTarget != 2400 {
		t.Fatalf("get cycle: %+v %v", got, err)
	}
	conflict := c
	conflict.ID = 0
	if err := r.CreateHealthCycle(&conflict); !errors.Is(err, ErrHealthCycleOverlap) {
		t.Fatalf("atomic overlap guard: %v", err)
	}
	cycles, err := r.ListHealthCycles()
	if err != nil || len(cycles) != 1 {
		t.Fatalf("list cycles: %v %v", cycles, err)
	}
	for _, stamp := range []string{"2026-09-10T09:00:00.000000000Z", "2026-09-10T18:00:00.000000000Z"} {
		w := models.HealthWeightEntry{MeasuredAt: stamp, WeightKg: 80.25, Source: "manual"}
		if err := r.CreateHealthWeightEntry(&w); err != nil || w.ID == 0 {
			t.Fatalf("weight insert: %v", err)
		}
	}
	weights, err := r.ListHealthWeightEntries()
	if err != nil || len(weights) != 2 || weights[0].MeasuredAt < weights[1].MeasuredAt || weights[0].WeightKg != 80.25 {
		t.Fatalf("weights: %+v %v", weights, err)
	}
	if err := r.UpsertHealthSyncState(models.HealthSyncState{Source: "apple_shortcuts", LastSyncedAt: "2026-09-10T15:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	state, err := r.GetHealthSyncState("apple_shortcuts")
	if err != nil || state.UpdatedAt == "" {
		t.Fatalf("state: %+v %v", state, err)
	}
}
