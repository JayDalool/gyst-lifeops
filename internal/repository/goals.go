package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"lifeops-tracker/internal/models"
)

const goalColumns = `id, uuid, title, description, type, category,
	target_value, current_value, unit,
	COALESCE(start_date, ''), COALESCE(end_date, ''), status,
	created_at, updated_at, COALESCE(archived_at, '')`

func (r *Repository) ListGoals(includeArchived bool) ([]models.GoalWithProgress, error) {
	where := "WHERE archived_at IS NULL AND status != 'archived'"
	if includeArchived {
		where = ""
	}

	rows, err := r.db.Query(fmt.Sprintf(`SELECT %s FROM goals %s
		ORDER BY
			CASE status WHEN 'active' THEN 0 WHEN 'completed' THEN 1 ELSE 2 END,
			CASE type WHEN 'yearly' THEN 0 WHEN 'monthly' THEN 1 ELSE 2 END,
			CASE WHEN end_date IS NULL THEN 1 ELSE 0 END,
			end_date,
			created_at DESC`, goalColumns, where))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.GoalWithProgress
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		goals = append(goals, models.GoalWithProgress{
			Goal:     goal,
			Progress: progressPercent(goal.CurrentValue, goal.TargetValue),
		})
	}
	return goals, rows.Err()
}

func (r *Repository) GoalByID(id int64) (models.Goal, error) {
	goal, err := scanGoal(r.db.QueryRow(fmt.Sprintf(`SELECT %s FROM goals WHERE id = ?`, goalColumns), id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Goal{}, err
	}
	return goal, err
}

func (r *Repository) GoalWithProgress(id int64) (models.GoalWithProgress, error) {
	goal, err := r.GoalByID(id)
	if err != nil {
		return models.GoalWithProgress{}, err
	}
	return models.GoalWithProgress{
		Goal:     goal,
		Progress: progressPercent(goal.CurrentValue, goal.TargetValue),
	}, nil
}

// CreateGoal stores a goal. today is the application's local day, used as the
// initial progress entry date when the goal has no start date.
func (r *Repository) CreateGoal(goal *models.Goal, today string) error {
	goal.Type = clean(goal.Type, "yearly")
	goal.Category = clean(goal.Category, "personal")
	goal.Unit = clean(goal.Unit, "tasks")
	goal.Status = clean(goal.Status, "active")

	res, err := r.db.Exec(`INSERT INTO goals
		(title, description, type, category, target_value, current_value, unit, start_date, end_date, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		goal.Title,
		goal.Description,
		goal.Type,
		goal.Category,
		goal.TargetValue,
		goal.CurrentValue,
		goal.Unit,
		nullString(goal.StartDate),
		nullString(goal.EndDate),
		goal.Status,
	)
	if err != nil {
		return err
	}
	goal.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	if goal.CurrentValue > 0 {
		entryDate := goal.StartDate
		if entryDate == "" {
			entryDate = today
		}
		return r.AddGoalProgress(goal.ID, entryDate, goal.CurrentValue, "Initial progress")
	}
	return nil
}

// UpdateGoal saves the editable fields of a live (not archived) goal. It never
// touches current_value or the progress entries: only AddGoalProgress writes
// those. Returns sql.ErrNoRows when the goal is missing or archived.
func (r *Repository) UpdateGoal(goal *models.Goal) error {
	res, err := r.db.Exec(`UPDATE goals SET
		title = ?, description = ?, type = ?, category = ?, target_value = ?,
		unit = ?, start_date = ?, end_date = ?, status = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ? AND archived_at IS NULL AND status != 'archived'`,
		goal.Title, goal.Description, goal.Type, goal.Category, goal.TargetValue,
		goal.Unit, nullString(goal.StartDate), nullString(goal.EndDate), goal.Status,
		goal.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// AddGoalProgress records a progress entry and recomputes the goal's current
// value from the latest-dated entry, so a backdated entry never overwrites a
// newer value.
func (r *Repository) AddGoalProgress(goalID int64, entryDate string, value float64, notes string) error {
	if entryDate == "" {
		return fmt.Errorf("entry date is required")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRow(`SELECT 1 FROM goals WHERE id = ?`, goalID).Scan(&exists); err != nil {
		return err
	}

	if _, err := tx.Exec(`INSERT INTO goal_progress_entries
		(goal_id, entry_date, value, notes)
		VALUES (?, ?, ?, ?)`, goalID, entryDate, value, notes); err != nil {
		return err
	}

	if err := recomputeGoalProgressTx(tx, goalID); err != nil {
		return err
	}
	return tx.Commit()
}

// recomputeGoalProgressTx sets current_value (and status) from the entry with
// the latest entry_date, ties broken by the highest id.
func recomputeGoalProgressTx(tx *sql.Tx, goalID int64) error {
	var target float64
	var status string
	if err := tx.QueryRow(`SELECT target_value, status FROM goals WHERE id = ?`, goalID).Scan(&target, &status); err != nil {
		return err
	}

	var value float64
	err := tx.QueryRow(`SELECT value FROM goal_progress_entries
		WHERE goal_id = ?
		ORDER BY entry_date DESC, id DESC
		LIMIT 1`, goalID).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if status != "archived" {
		if target > 0 && value >= target {
			status = "completed"
		} else if status == "completed" {
			status = "active"
		}
	}

	_, err = tx.Exec(`UPDATE goals SET
		current_value = ?,
		status = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, value, status, goalID)
	return err
}

func (r *Repository) GoalProgressEntries(goalID int64) ([]models.GoalProgressEntry, error) {
	rows, err := r.db.Query(`SELECT id, goal_id, entry_date, value, notes, created_at
		FROM goal_progress_entries
		WHERE goal_id = ?
		ORDER BY entry_date, id`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.GoalProgressEntry
	for rows.Next() {
		var entry models.GoalProgressEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.GoalID,
			&entry.EntryDate,
			&entry.Value,
			&entry.Notes,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *Repository) ArchiveGoal(id int64) error {
	_, err := r.db.Exec(`UPDATE goals SET
		status = 'archived',
		archived_at = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, id)
	return err
}
