package repository

import (
	"database/sql"
	"errors"
)

// Water bottle log (#60). One row per "Bottle done" tap on a local day.

// AddHealthWater records one tap of ml on date.
func (r *Repository) AddHealthWater(date string, ml int64) error {
	_, err := r.db.Exec(`INSERT INTO health_water_log (date, ml) VALUES (?, ?)`, date, ml)
	return err
}

// UndoHealthWater removes the latest tap on date and returns its amount.
// It returns sql.ErrNoRows when nothing was logged that day.
func (r *Repository) UndoHealthWater(date string) (int64, error) {
	var id, ml int64
	err := r.db.QueryRow(`SELECT id, ml FROM health_water_log WHERE date = ? ORDER BY id DESC LIMIT 1`, date).Scan(&id, &ml)
	if err != nil {
		return 0, err
	}
	res, err := r.db.Exec(`DELETE FROM health_water_log WHERE id = ?`, id)
	if err != nil {
		return 0, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return 0, err
	} else if n != 1 {
		return 0, sql.ErrNoRows
	}
	return ml, nil
}

// HealthWaterDay is the day's total and number of taps; Taps 0 means
// nothing logged.
func (r *Repository) HealthWaterDay(date string) (totalML int64, taps int, err error) {
	err = r.db.QueryRow(`SELECT COALESCE(SUM(ml), 0), COUNT(*) FROM health_water_log WHERE date = ?`, date).Scan(&totalML, &taps)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return totalML, taps, err
}
