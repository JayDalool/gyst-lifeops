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

func (r *Repository) AddGoalProgress(goalID int64, entryDate string, value float64, notes string) error {
	if entryDate == "" {
		return fmt.Errorf("entry date is required")
	}

	goal, err := r.GoalByID(goalID)
	if err != nil {
		return err
	}

	status := goal.Status
	if goal.TargetValue > 0 && value >= goal.TargetValue {
		status = "completed"
	} else if status == "completed" {
		status = "active"
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`INSERT INTO goal_progress_entries
		(goal_id, entry_date, value, notes)
		VALUES (?, ?, ?, ?)`, goalID, entryDate, value, notes); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec(`UPDATE goals SET
		current_value = ?,
		status = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, value, status, goalID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *Repository) GoalProgressEntries(goalID int64) ([]models.GoalProgressEntry, error) {
	rows, err := r.db.Query(`SELECT id, goal_id, entry_date, value, notes, created_at
		FROM goal_progress_entries
		WHERE goal_id = ?
		ORDER BY entry_date, created_at`, goalID)
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
