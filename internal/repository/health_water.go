package repository

import (
	"database/sql"
	"errors"
)

// Water bottle log (#60). One row per tap on the bottle on a local day.

// AddHealthWater records one tap of ml on date while the day's total is
// still below targetML (#78); the tap that reaches or passes the target is
// the last one. The check and insert are one statement, so two quick taps
// can't both slip past it. It reports whether the tap was stored.
func (r *Repository) AddHealthWater(date string, ml, targetML int64) (bool, error) {
	res, err := r.db.Exec(`INSERT INTO health_water_log (date, ml)
		SELECT ?, ? WHERE (SELECT COALESCE(SUM(ml), 0) FROM health_water_log WHERE date = ?) < ?`,
		date, ml, date, targetML)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// UndoHealthWater removes the latest tap on date in one statement and
// returns its amount; sql.ErrNoRows when nothing was logged that day.
func (r *Repository) UndoHealthWater(date string) (int64, error) {
	var ml int64
	err := r.db.QueryRow(`DELETE FROM health_water_log
		WHERE id = (SELECT id FROM health_water_log WHERE date = ? ORDER BY id DESC LIMIT 1)
		RETURNING ml`, date).Scan(&ml)
	return ml, err
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
