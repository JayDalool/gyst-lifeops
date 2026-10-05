package repository

import (
	"database/sql"
	"errors"

	"lifeops-tracker/internal/models"
)

// The conditional writes repeat the service overlap check atomically so two
// requests cannot both insert/update conflicting date ranges after validation.
var ErrHealthCycleOverlap = errors.New("health cycle overlaps an existing cycle")

const healthCycleColumns = `id, title, start_date, end_date, primary_goal, daily_calorie_target, protein_target_g, carbs_target_g, fat_target_g, fiber_target_g, target_deficit_kcal, strength_sessions_per_week, cardio_sessions_per_week, step_target, notes, created_at, updated_at`

func scanHealthCycle(row scanner) (models.HealthCycle, error) {
	var c models.HealthCycle
	err := row.Scan(&c.ID, &c.Title, &c.StartDate, &c.EndDate, &c.PrimaryGoal, &c.DailyCalorieTarget, &c.ProteinTargetG, &c.CarbsTargetG, &c.FatTargetG, &c.FiberTargetG, &c.TargetDeficitKcal, &c.StrengthSessionsPerWeek, &c.CardioSessionsPerWeek, &c.StepTarget, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}
func (r *Repository) GetHealthCycle(id int64) (models.HealthCycle, error) {
	return scanHealthCycle(r.db.QueryRow(`SELECT `+healthCycleColumns+` FROM health_cycles WHERE id = ?`, id))
}
func (r *Repository) GetHealthCycleForDate(date string) (models.HealthCycle, error) {
	return scanHealthCycle(r.db.QueryRow(`SELECT `+healthCycleColumns+` FROM health_cycles WHERE start_date <= ? AND end_date >= ? ORDER BY start_date DESC LIMIT 1`, date, date))
}
func (r *Repository) ListHealthCycles() ([]models.HealthCycle, error) {
	rows, err := r.db.Query(`SELECT ` + healthCycleColumns + ` FROM health_cycles ORDER BY start_date DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cycles := []models.HealthCycle{}
	for rows.Next() {
		c, err := scanHealthCycle(rows)
		if err != nil {
			return nil, err
		}
		cycles = append(cycles, c)
	}
	return cycles, rows.Err()
}
func (r *Repository) CreateHealthCycle(c *models.HealthCycle) error {
	result, err := r.db.Exec(`INSERT INTO health_cycles (title, start_date, end_date, primary_goal, daily_calorie_target, protein_target_g, carbs_target_g, fat_target_g, fiber_target_g, target_deficit_kcal, strength_sessions_per_week, cardio_sessions_per_week, step_target, notes) SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
 WHERE NOT EXISTS (SELECT 1 FROM health_cycles WHERE start_date <= ? AND end_date >= ?)`, c.Title, c.StartDate, c.EndDate, c.PrimaryGoal, c.DailyCalorieTarget, c.ProteinTargetG, c.CarbsTargetG, c.FatTargetG, c.FiberTargetG, c.TargetDeficitKcal, c.StrengthSessionsPerWeek, c.CardioSessionsPerWeek, c.StepTarget, c.Notes, c.EndDate, c.StartDate)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrHealthCycleOverlap
	}
	c.ID, err = result.LastInsertId()
	return err
}
func (r *Repository) UpdateHealthCycle(c *models.HealthCycle) error {
	result, err := r.db.Exec(`UPDATE health_cycles SET title = ?, start_date = ?, end_date = ?, primary_goal = ?, daily_calorie_target = ?, protein_target_g = ?, carbs_target_g = ?, fat_target_g = ?, fiber_target_g = ?, target_deficit_kcal = ?, strength_sessions_per_week = ?, cardio_sessions_per_week = ?, step_target = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
 WHERE id = ? AND NOT EXISTS (SELECT 1 FROM health_cycles WHERE id != ? AND start_date <= ? AND end_date >= ?)`, c.Title, c.StartDate, c.EndDate, c.PrimaryGoal, c.DailyCalorieTarget, c.ProteinTargetG, c.CarbsTargetG, c.FatTargetG, c.FiberTargetG, c.TargetDeficitKcal, c.StrengthSessionsPerWeek, c.CardioSessionsPerWeek, c.StepTarget, c.Notes, c.ID, c.ID, c.EndDate, c.StartDate)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := r.GetHealthCycle(c.ID); err != nil {
			return err
		}
		return ErrHealthCycleOverlap
	}
	return nil
}

const healthSnapshotColumns = `date, dietary_calories, protein_g, carbs_g, fat_g, fiber_g, steps_watch, steps_phone, active_energy_kcal, resting_energy_kcal, exercise_minutes, last_synced_at`

func scanHealthSnapshot(row scanner) (models.DailyHealthSnapshot, error) {
	var s models.DailyHealthSnapshot
	err := row.Scan(&s.Date, &s.DietaryCalories, &s.ProteinG, &s.CarbsG, &s.FatG, &s.FiberG, &s.StepsWatch, &s.StepsPhone, &s.ActiveEnergyKcal, &s.RestingEnergyKcal, &s.ExerciseMinutes, &s.LastSyncedAt)
	return s, err
}
func (r *Repository) GetDailyHealthSnapshot(date string) (models.DailyHealthSnapshot, error) {
	return scanHealthSnapshot(r.db.QueryRow(`SELECT `+healthSnapshotColumns+` FROM daily_health_snapshots WHERE date = ?`, date))
}

// ListDailyHealthSnapshots returns the stored days from..to inclusive
// (YYYY-MM-DD), oldest first. Days with no row are simply absent.
func (r *Repository) ListDailyHealthSnapshots(from, to string) ([]models.DailyHealthSnapshot, error) {
	rows, err := r.db.Query(`SELECT `+healthSnapshotColumns+` FROM daily_health_snapshots WHERE date >= ? AND date <= ? ORDER BY date`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.DailyHealthSnapshot{}
	for rows.Next() {
		s, err := scanHealthSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

const upsertHealthSnapshotSQL = `INSERT INTO daily_health_snapshots (` + healthSnapshotColumns + `)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
 ON CONFLICT(date) DO UPDATE SET
 dietary_calories = COALESCE(excluded.dietary_calories, daily_health_snapshots.dietary_calories),
 protein_g = COALESCE(excluded.protein_g, daily_health_snapshots.protein_g),
 carbs_g = COALESCE(excluded.carbs_g, daily_health_snapshots.carbs_g),
 fat_g = COALESCE(excluded.fat_g, daily_health_snapshots.fat_g),
 fiber_g = COALESCE(excluded.fiber_g, daily_health_snapshots.fiber_g),
 steps_watch = COALESCE(excluded.steps_watch, daily_health_snapshots.steps_watch),
 steps_phone = COALESCE(excluded.steps_phone, daily_health_snapshots.steps_phone),
 active_energy_kcal = COALESCE(excluded.active_energy_kcal, daily_health_snapshots.active_energy_kcal),
 resting_energy_kcal = COALESCE(excluded.resting_energy_kcal, daily_health_snapshots.resting_energy_kcal),
 exercise_minutes = COALESCE(excluded.exercise_minutes, daily_health_snapshots.exercise_minutes),
 last_synced_at = excluded.last_synced_at`

func healthSnapshotArgs(s models.DailyHealthSnapshot) []any {
	return []any{s.Date, s.DietaryCalories, s.ProteinG, s.CarbsG, s.FatG, s.FiberG, s.StepsWatch, s.StepsPhone, s.ActiveEnergyKcal, s.RestingEnergyKcal, s.ExerciseMinutes, s.LastSyncedAt}
}
func (r *Repository) UpsertDailyHealthSnapshot(s models.DailyHealthSnapshot) error {
	_, err := r.db.Exec(upsertHealthSnapshotSQL, healthSnapshotArgs(s)...)
	return err
}

const upsertHealthSyncSQL = `INSERT INTO health_sync_states (source, last_synced_at) VALUES (?, ?)
 ON CONFLICT(source) DO UPDATE SET last_synced_at = excluded.last_synced_at, updated_at = CURRENT_TIMESTAMP`

func (r *Repository) GetHealthSyncState(source string) (models.HealthSyncState, error) {
	var s models.HealthSyncState
	err := r.db.QueryRow(`SELECT source, last_synced_at, updated_at FROM health_sync_states WHERE source = ?`, source).Scan(&s.Source, &s.LastSyncedAt, &s.UpdatedAt)
	return s, err
}
func (r *Repository) UpsertHealthSyncState(s models.HealthSyncState) error {
	_, err := r.db.Exec(upsertHealthSyncSQL, s.Source, s.LastSyncedAt)
	return err
}

// ApplyHealthSync returns the merged row and commits both changes together.
func (r *Repository) ApplyHealthSync(s models.DailyHealthSnapshot, source string) (models.DailyHealthSnapshot, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return models.DailyHealthSnapshot{}, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(upsertHealthSnapshotSQL, healthSnapshotArgs(s)...); err != nil {
		return models.DailyHealthSnapshot{}, err
	}
	if _, err = tx.Exec(upsertHealthSyncSQL, source, s.LastSyncedAt); err != nil {
		return models.DailyHealthSnapshot{}, err
	}
	merged, err := scanHealthSnapshot(tx.QueryRow(`SELECT `+healthSnapshotColumns+` FROM daily_health_snapshots WHERE date = ?`, s.Date))
	if err != nil {
		return models.DailyHealthSnapshot{}, err
	}
	return merged, tx.Commit()
}
func (r *Repository) CreateHealthWeightEntry(w *models.HealthWeightEntry) error {
	result, err := r.db.Exec(`INSERT INTO health_weight_entries (measured_at, weight_kg, source) VALUES (?, ?, ?)`, w.MeasuredAt, w.WeightKg, w.Source)
	if err != nil {
		return err
	}
	w.ID, err = result.LastInsertId()
	return err
}
func (r *Repository) GetHealthWeightEntry(id int64) (models.HealthWeightEntry, error) {
	var w models.HealthWeightEntry
	err := r.db.QueryRow(`SELECT id, measured_at, weight_kg, source, created_at FROM health_weight_entries WHERE id = ?`, id).Scan(&w.ID, &w.MeasuredAt, &w.WeightKg, &w.Source, &w.CreatedAt)
	return w, err
}

// DeleteManualHealthWeightEntry removes one manually entered reading. Synced
// readings are never deleted here; a missing or non-manual id is sql.ErrNoRows.
func (r *Repository) DeleteManualHealthWeightEntry(id int64) error {
	result, err := r.db.Exec(`DELETE FROM health_weight_entries WHERE id = ? AND source = 'manual'`, id)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *Repository) ListHealthWeightEntries() ([]models.HealthWeightEntry, error) {
	rows, err := r.db.Query(`SELECT id, measured_at, weight_kg, source, created_at FROM health_weight_entries ORDER BY measured_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []models.HealthWeightEntry{}
	for rows.Next() {
		var w models.HealthWeightEntry
		if err := rows.Scan(&w.ID, &w.MeasuredAt, &w.WeightKg, &w.Source, &w.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, w)
	}
	return entries, rows.Err()
}
