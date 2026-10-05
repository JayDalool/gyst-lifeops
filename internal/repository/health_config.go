package repository

import (
	"database/sql"

	"lifeops-tracker/internal/models"
)

// Daily checklist and My foods (Issue #59).

// ListHealthChecklistItems returns active items in display order, with
// DoneToday filled for the given local date.
func (r *Repository) ListHealthChecklistItems(date string) ([]models.HealthChecklistItem, error) {
	rows, err := r.db.Query(`SELECT i.id, i.name, i.kind, i.position, i.active, l.item_id IS NOT NULL
		FROM health_checklist_items i
		LEFT JOIN health_checklist_log l ON l.item_id = i.id AND l.date = ?
		WHERE i.active = 1 ORDER BY i.position, i.id`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.HealthChecklistItem{}
	for rows.Next() {
		var it models.HealthChecklistItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Kind, &it.Position, &it.Active, &it.DoneToday); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *Repository) CreateHealthChecklistItem(it *models.HealthChecklistItem) error {
	result, err := r.db.Exec(`INSERT INTO health_checklist_items (name, kind, position)
		VALUES (?, ?, COALESCE((SELECT MAX(position) + 1 FROM health_checklist_items), 0))`, it.Name, it.Kind)
	if err != nil {
		return err
	}
	it.ID, err = result.LastInsertId()
	it.Active = true
	return err
}

// DeactivateHealthChecklistItem hides an item but keeps its history.
func (r *Repository) DeactivateHealthChecklistItem(id int64) error {
	return affectedOne(r.db.Exec(`UPDATE health_checklist_items SET active = 0 WHERE id = ? AND active = 1`, id))
}

// ToggleHealthHabit flips a habit's tick for one local date and reports the
// new state. Only active habits can be ticked; anything else is ErrNoRows.
func (r *Repository) ToggleHealthHabit(id int64, date string) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var kind string
	if err := tx.QueryRow(`SELECT kind FROM health_checklist_items WHERE id = ? AND active = 1`, id).Scan(&kind); err != nil {
		return false, err
	}
	if kind != "habit" {
		return false, sql.ErrNoRows
	}
	result, err := tx.Exec(`DELETE FROM health_checklist_log WHERE item_id = ? AND date = ?`, id, date)
	if err != nil {
		return false, err
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if removed == 0 {
		if _, err := tx.Exec(`INSERT INTO health_checklist_log (item_id, date) VALUES (?, ?)`, id, date); err != nil {
			return false, err
		}
	}
	return removed == 0, tx.Commit()
}

const healthFoodColumns = `id, name, serving, kcal, protein_g, carbs_g, fat_g, fiber_g, notes, active`

func scanHealthFood(row scanner) (models.HealthFood, error) {
	var f models.HealthFood
	err := row.Scan(&f.ID, &f.Name, &f.Serving, &f.Kcal, &f.ProteinG, &f.CarbsG, &f.FatG, &f.FiberG, &f.Notes, &f.Active)
	return f, err
}

// ListHealthFoods returns foods, active first, then by name.
func (r *Repository) ListHealthFoods(includeInactive bool) ([]models.HealthFood, error) {
	query := `SELECT ` + healthFoodColumns + ` FROM health_foods`
	if !includeInactive {
		query += ` WHERE active = 1`
	}
	rows, err := r.db.Query(query + ` ORDER BY active DESC, name COLLATE NOCASE, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	foods := []models.HealthFood{}
	for rows.Next() {
		f, err := scanHealthFood(rows)
		if err != nil {
			return nil, err
		}
		foods = append(foods, f)
	}
	return foods, rows.Err()
}

func (r *Repository) GetHealthFood(id int64) (models.HealthFood, error) {
	return scanHealthFood(r.db.QueryRow(`SELECT `+healthFoodColumns+` FROM health_foods WHERE id = ?`, id))
}

func (r *Repository) CreateHealthFood(f *models.HealthFood) error {
	result, err := r.db.Exec(`INSERT INTO health_foods (name, serving, kcal, protein_g, carbs_g, fat_g, fiber_g, notes, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, f.Name, f.Serving, f.Kcal, f.ProteinG, f.CarbsG, f.FatG, f.FiberG, f.Notes, f.Active)
	if err != nil {
		return err
	}
	f.ID, err = result.LastInsertId()
	return err
}

func (r *Repository) UpdateHealthFood(f models.HealthFood) error {
	return affectedOne(r.db.Exec(`UPDATE health_foods SET name = ?, serving = ?, kcal = ?, protein_g = ?, carbs_g = ?, fat_g = ?, fiber_g = ?, notes = ?, active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		f.Name, f.Serving, f.Kcal, f.ProteinG, f.CarbsG, f.FatG, f.FiberG, f.Notes, f.Active, f.ID))
}

// affectedOne turns "no row matched" into sql.ErrNoRows.
func affectedOne(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
