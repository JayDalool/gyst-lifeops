package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lifeops-tracker/internal/models"
)

const taskColumns = `id, uuid, title, description, category, priority, status,
	COALESCE(due_date, ''), repeat_type, COALESCE(reminder_time, ''),
	stars_reward, proof_required, proof_type, COALESCE(proof_url, ''), COALESCE(proof_path, ''), stars_awarded,
	created_at, updated_at, COALESCE(completed_at, ''), COALESCE(archived_at, '')`

func (r *Repository) ListTasks(includeArchived bool) ([]models.Task, error) {
	where := "WHERE archived_at IS NULL"
	if includeArchived {
		where = ""
	}

	rows, err := r.db.Query(fmt.Sprintf(`SELECT %s FROM tasks %s
		ORDER BY
			CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END,
			CASE WHEN due_date IS NULL THEN 1 ELSE 0 END,
			due_date,
			created_at DESC`, taskColumns, where))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *Repository) ListActiveOneTimeOverdueTasks(today string) ([]models.Task, error) {
	rows, err := r.db.Query(fmt.Sprintf(`SELECT %s FROM tasks
		WHERE archived_at IS NULL
			AND repeat_type = 'none'
			AND status = 'pending'
			AND due_date IS NOT NULL
			AND due_date < ?
		ORDER BY due_date, CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END`, taskColumns), today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *Repository) TaskByID(id int64) (models.Task, error) {
	task, err := scanTask(r.db.QueryRow(fmt.Sprintf(`SELECT %s FROM tasks WHERE id = ?`, taskColumns), id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Task{}, err
	}
	return task, err
}

func (r *Repository) CreateTask(task *models.Task) error {
	task.Category = clean(task.Category, "personal")
	task.Priority = clean(task.Priority, "medium")
	task.Status = clean(task.Status, "pending")
	task.RepeatType = clean(task.RepeatType, "none")
	task.ProofType = cleanProofType(task.ProofType)
	if task.StarsReward <= 0 {
		task.StarsReward = 1
	}

	res, err := r.db.Exec(`INSERT INTO tasks
		(title, description, category, priority, status, due_date, repeat_type, reminder_time,
		 stars_reward, proof_required, proof_type, proof_url, proof_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.Title,
		task.Description,
		task.Category,
		task.Priority,
		task.Status,
		nullString(task.DueDate),
		task.RepeatType,
		nullString(task.ReminderTime),
		task.StarsReward,
		boolFlag(task.ProofRequired),
		task.ProofType,
		nullString(task.ProofURL),
		nullString(task.ProofPath),
	)
	if err != nil {
		return err
	}
	task.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	return r.syncTaskReminder(task.ID)
}

func (r *Repository) UpdateTask(task *models.Task) error {
	task.Category = clean(task.Category, "personal")
	task.Priority = clean(task.Priority, "medium")
	task.Status = clean(task.Status, "pending")
	task.RepeatType = clean(task.RepeatType, "none")
	task.ProofType = cleanProofType(task.ProofType)
	if task.StarsReward <= 0 {
		task.StarsReward = 1
	}

	_, err := r.db.Exec(`UPDATE tasks SET
		title = ?,
		description = ?,
		category = ?,
		priority = ?,
		status = ?,
		due_date = ?,
		repeat_type = ?,
		reminder_time = ?,
		stars_reward = ?,
		proof_required = ?,
		proof_type = ?,
		proof_url = ?,
		proof_path = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`,
		task.Title,
		task.Description,
		task.Category,
		task.Priority,
		task.Status,
		nullString(task.DueDate),
		task.RepeatType,
		nullString(task.ReminderTime),
		task.StarsReward,
		boolFlag(task.ProofRequired),
		task.ProofType,
		nullString(task.ProofURL),
		nullString(task.ProofPath),
		task.ID,
	)
	if err != nil {
		return err
	}
	return r.syncTaskReminder(task.ID)
}

func (r *Repository) ArchiveTask(id int64) error {
	_, err := r.db.Exec(`UPDATE tasks SET
		archived_at = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, id)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`UPDATE reminders SET
		status = 'dismissed',
		dismissed_at = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE source_type = 'task' AND source_id = ?`, id)
	return err
}

func (r *Repository) SetTaskOccurrenceStatus(taskID int64, occurrenceDate, status string) error {
	if status != "completed" && status != "skipped" {
		return fmt.Errorf("unsupported task status %q", status)
	}

	task, err := r.TaskByID(taskID)
	if err != nil {
		return err
	}
	if occurrenceDate == "" {
		occurrenceDate = task.DueDate
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO task_completions
		(task_id, occurrence_date, status, completed_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(task_id, occurrence_date) DO UPDATE SET
			status = excluded.status,
			completed_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			sync_version = task_completions.sync_version + 1`,
		taskID,
		occurrenceDate,
		status,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	if task.RepeatType == "none" {
		completedAt := "CURRENT_TIMESTAMP"
		if status == "skipped" {
			completedAt = "NULL"
		}
		award := 0
		if status == "completed" && task.StarsAwarded == 0 {
			award = task.StarsReward
			if taskHasProof(task) {
				award += proofBonus(task)
			}
		}
		if _, err := tx.Exec(fmt.Sprintf(`UPDATE tasks SET
			status = ?,
			completed_at = %s,
			stars_awarded = CASE WHEN ? = 'completed' AND stars_awarded = 0 THEN ? ELSE stars_awarded END,
			updated_at = CURRENT_TIMESTAMP,
			sync_version = sync_version + 1
			WHERE id = ?`, completedAt), status, status, award, taskID); err != nil {
			tx.Rollback()
			return err
		}
		if award > 0 && taskHasProof(task) {
			if _, err := tx.Exec(`INSERT INTO achievements
				(title, description, stars_bonus, trigger_type)
				VALUES (?, ?, ?, 'proof_submitted')`,
				"Proof submitted",
				task.Title,
				0,
			); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) PlayerStats() (models.PlayerStats, error) {
	var stats models.PlayerStats
	if err := r.db.QueryRow(`SELECT
		COALESCE((SELECT SUM(stars_awarded) FROM tasks), 0) +
		COALESCE((SELECT SUM(stars_bonus) FROM achievements), 0),
		(SELECT COUNT(*) FROM tasks WHERE status = 'completed'),
		(SELECT COUNT(*) FROM tasks WHERE proof_type != 'none' AND (COALESCE(proof_url, '') != '' OR COALESCE(proof_path, '') != ''))
	`).Scan(&stats.TotalStars, &stats.TaskDoneCount, &stats.ProofCount); err != nil {
		return models.PlayerStats{}, err
	}
	stats.CurrentLevel = (stats.TotalStars / 100) + 1
	return stats, nil
}

func (r *Repository) CountAchievements() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM achievements`).Scan(&count)
	return count, err
}

func (r *Repository) RecentAchievements(limit int) ([]models.Achievement, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := r.db.Query(`SELECT id, title, description, stars_bonus, unlocked_at, trigger_type, created_at
		FROM achievements
		ORDER BY unlocked_at DESC, id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []models.Achievement
	for rows.Next() {
		var achievement models.Achievement
		if err := rows.Scan(
			&achievement.ID,
			&achievement.Title,
			&achievement.Description,
			&achievement.StarsBonus,
			&achievement.UnlockedAt,
			&achievement.TriggerType,
			&achievement.CreatedAt,
		); err != nil {
			return nil, err
		}
		achievements = append(achievements, achievement)
	}
	return achievements, rows.Err()
}

func cleanProofType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "photo", "file", "link", "text":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "none"
	}
}

func taskHasProof(task models.Task) bool {
	return task.ProofType != "none" && (strings.TrimSpace(task.ProofURL) != "" || strings.TrimSpace(task.ProofPath) != "")
}

func proofBonus(task models.Task) int {
	if task.ProofRequired || taskHasProof(task) {
		return 2
	}
	return 0
}

func (r *Repository) TaskCompletionsBetween(startDate, endDate string) ([]models.TaskCompletion, error) {
	rows, err := r.db.Query(`SELECT id, task_id, occurrence_date, status, completed_at, notes
		FROM task_completions
		WHERE occurrence_date >= ? AND occurrence_date <= ?
		ORDER BY occurrence_date`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var completions []models.TaskCompletion
	for rows.Next() {
		var completion models.TaskCompletion
		if err := rows.Scan(
			&completion.ID,
			&completion.TaskID,
			&completion.OccurrenceDate,
			&completion.Status,
			&completion.CompletedAt,
			&completion.Notes,
		); err != nil {
			return nil, err
		}
		completions = append(completions, completion)
	}
	return completions, rows.Err()
}

func (r *Repository) syncTaskReminder(taskID int64) error {
	task, err := r.TaskByID(taskID)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM reminders WHERE source_type = 'task' AND source_id = ?`, taskID); err != nil {
		tx.Rollback()
		return err
	}

	if task.ReminderTime != "" && task.DueDate != "" && task.ArchivedAt == "" {
		if _, err := tx.Exec(`INSERT INTO reminders
			(source_type, source_id, title, reminder_date, reminder_time)
			VALUES ('task', ?, ?, ?, ?)`,
			task.ID,
			task.Title,
			task.DueDate,
			task.ReminderTime,
		); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
