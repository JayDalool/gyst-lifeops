package repository

import (
	"database/sql"

	"lifeops-tracker/internal/models"
)

// Reference targets (#205): outside recommendations kept beside the plan.
// Nothing here reads or writes health_cycles.

const healthReferenceColumns = `id, cycle_id, source_label, source_kind, issued_on, kcal, protein_g, carbs_g, fat_g, fiber_g, notes, created_at, archived_at`

func scanHealthReference(row scanner) (models.HealthReferenceTarget, error) {
	var t models.HealthReferenceTarget
	var cycle sql.NullInt64
	var issued, archived sql.NullString
	err := row.Scan(&t.ID, &cycle, &t.SourceLabel, &t.SourceKind, &issued, &t.Kcal, &t.ProteinG, &t.CarbsG, &t.FatG, &t.FiberG, &t.Notes, &t.CreatedAt, &archived)
	if cycle.Valid {
		t.CycleID = &cycle.Int64
	}
	t.IssuedOn, t.ArchivedAt = issued.String, archived.String
	return t, err
}

func (r *Repository) CreateHealthReferenceTarget(t *models.HealthReferenceTarget) error {
	var issued any
	if t.IssuedOn != "" {
		issued = t.IssuedOn
	}
	result, err := r.db.Exec(`INSERT INTO health_reference_targets (cycle_id, source_label, source_kind, issued_on, kcal, protein_g, carbs_g, fat_g, fiber_g, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, t.CycleID, t.SourceLabel, t.SourceKind, issued, t.Kcal, t.ProteinG, t.CarbsG, t.FatG, t.FiberG, t.Notes, t.CreatedAt)
	if err != nil {
		return err
	}
	t.ID, err = result.LastInsertId()
	return err
}

// ListHealthReferenceTargets returns the active (not archived) targets,
// newest first.
func (r *Repository) ListHealthReferenceTargets() ([]models.HealthReferenceTarget, error) {
	rows, err := r.db.Query(`SELECT ` + healthReferenceColumns + ` FROM health_reference_targets WHERE archived_at IS NULL ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.HealthReferenceTarget{}
	for rows.Next() {
		t, err := scanHealthReference(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ArchiveHealthReferenceTarget hides a target; sql.ErrNoRows when it does not
// exist or is already archived. Rows are never deleted.
func (r *Repository) ArchiveHealthReferenceTarget(id int64, at string) error {
	return affectedOne(r.db.Exec(`UPDATE health_reference_targets SET archived_at = ? WHERE id = ? AND archived_at IS NULL`, at, id))
}
