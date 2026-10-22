package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"lifeops-tracker/internal/models"
)

const courseColumns = `id, uuid, name, provider, instructor, url, status,
	progress_percent, COALESCE(target_date, ''), notes,
	created_at, updated_at, COALESCE(archived_at, '')`

// Every learning-block read joins its course so cross-course views (Today, the
// unified calendar) carry the course name without a second query.
const learningBlockColumns = `b.id, b.uuid, b.course_id, b.title, b.block_type, b.status,
	COALESCE(b.start_time, ''), COALESCE(b.end_time, ''), COALESCE(b.due_date, ''),
	b.url, b.repeat_type, b.notes, b.sort_order, b.is_catchup,
	b.created_at, b.updated_at, COALESCE(b.completed_at, ''), COALESCE(b.archived_at, ''),
	COALESCE(c.name, '')`

func scanCourse(s scanner) (models.Course, error) {
	var c models.Course
	err := s.Scan(
		&c.ID, &c.UUID, &c.Name, &c.Provider, &c.Instructor, &c.URL, &c.Status,
		&c.ProgressPercent, &c.TargetDate, &c.Notes,
		&c.CreatedAt, &c.UpdatedAt, &c.ArchivedAt,
	)
	return c, err
}

func scanLearningBlock(s scanner) (models.LearningBlock, error) {
	var b models.LearningBlock
	var isCatchup int
	err := s.Scan(
		&b.ID, &b.UUID, &b.CourseID, &b.Title, &b.BlockType, &b.Status,
		&b.StartTime, &b.EndTime, &b.DueDate, &b.URL, &b.RepeatType, &b.Notes,
		&b.SortOrder, &isCatchup,
		&b.CreatedAt, &b.UpdatedAt, &b.CompletedAt, &b.ArchivedAt,
		&b.CourseName,
	)
	b.IsCatchup = isCatchup == 1
	return b, err
}

func (r *Repository) ListCourses(includeArchived bool) ([]models.Course, error) {
	where := "WHERE archived_at IS NULL"
	if includeArchived {
		where = ""
	}
	rows, err := r.db.Query(fmt.Sprintf(`SELECT %s FROM courses %s
		ORDER BY
			CASE status WHEN 'active' THEN 0 WHEN 'paused' THEN 1 WHEN 'completed' THEN 2 ELSE 3 END,
			CASE WHEN target_date IS NULL THEN 1 ELSE 0 END, target_date,
			updated_at DESC`, courseColumns, where))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	return courses, rows.Err()
}

func (r *Repository) CourseByID(id int64) (models.Course, error) {
	c, err := scanCourse(r.db.QueryRow(fmt.Sprintf(`SELECT %s FROM courses WHERE id = ?`, courseColumns), id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Course{}, err
	}
	return c, err
}

func (r *Repository) CreateCourse(c *models.Course) error {
	c.Status = clean(c.Status, "active")
	if c.ProgressPercent < 0 {
		c.ProgressPercent = 0
	}
	if c.ProgressPercent > 100 {
		c.ProgressPercent = 100
	}
	res, err := r.db.Exec(`INSERT INTO courses
		(name, provider, instructor, url, status, progress_percent, target_date, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Name, c.Provider, c.Instructor, c.URL, c.Status, c.ProgressPercent,
		nullString(c.TargetDate), c.Notes,
	)
	if err != nil {
		return err
	}
	c.ID, err = res.LastInsertId()
	return err
}

func (r *Repository) UpdateCourse(c *models.Course) error {
	c.Status = clean(c.Status, "active")
	if c.ProgressPercent < 0 {
		c.ProgressPercent = 0
	}
	if c.ProgressPercent > 100 {
		c.ProgressPercent = 100
	}
	_, err := r.db.Exec(`UPDATE courses SET
		name = ?, provider = ?, instructor = ?, url = ?, status = ?,
		progress_percent = ?, target_date = ?, notes = ?,
		updated_at = CURRENT_TIMESTAMP, sync_version = sync_version + 1
		WHERE id = ?`,
		c.Name, c.Provider, c.Instructor, c.URL, c.Status, c.ProgressPercent,
		nullString(c.TargetDate), c.Notes, c.ID,
	)
	return err
}

func (r *Repository) SetCourseStatus(id int64, status string) error {
	_, err := r.db.Exec(`UPDATE courses SET status = ?, updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1 WHERE id = ?`, status, id)
	return err
}

func (r *Repository) ArchiveCourse(id int64) error {
	_, err := r.db.Exec(`UPDATE courses SET status = 'archived',
		archived_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1 WHERE id = ?`, id)
	return err
}

func (r *Repository) ListLearningBlocks(courseID int64) ([]models.LearningBlock, error) {
	rows, err := r.db.Query(fmt.Sprintf(`SELECT %s FROM learning_blocks b
		JOIN courses c ON c.id = b.course_id
		WHERE b.course_id = ? AND b.archived_at IS NULL
		ORDER BY b.sort_order, CASE WHEN b.due_date IS NULL THEN 1 ELSE 0 END, b.due_date, b.id`,
		learningBlockColumns), courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBlocks(rows)
}

func (r *Repository) LearningBlockByID(id int64) (models.LearningBlock, error) {
	b, err := scanLearningBlock(r.db.QueryRow(fmt.Sprintf(`SELECT %s FROM learning_blocks b
		JOIN courses c ON c.id = b.course_id WHERE b.id = ?`, learningBlockColumns), id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.LearningBlock{}, err
	}
	return b, err
}

func (r *Repository) CreateLearningBlock(b *models.LearningBlock) error {
	b.BlockType = clean(b.BlockType, "lesson")
	b.Status = clean(b.Status, "pending")
	b.RepeatType = clean(b.RepeatType, "none")
	res, err := r.db.Exec(`INSERT INTO learning_blocks
		(course_id, title, block_type, status, start_time, end_time, due_date, url, repeat_type, notes, sort_order, is_catchup)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.CourseID, b.Title, b.BlockType, b.Status,
		nullString(b.StartTime), nullString(b.EndTime), nullString(b.DueDate),
		b.URL, b.RepeatType, b.Notes, b.SortOrder, boolFlag(b.IsCatchup),
	)
	if err != nil {
		return err
	}
	b.ID, err = res.LastInsertId()
	return err
}

func (r *Repository) SetLearningBlockStatus(courseID, blockID int64, status string) error {
	completedAt := "NULL"
	if status == "completed" {
		completedAt = "CURRENT_TIMESTAMP"
	}
	_, err := r.db.Exec(fmt.Sprintf(`UPDATE learning_blocks SET status = ?,
		completed_at = %s, updated_at = CURRENT_TIMESTAMP, sync_version = sync_version + 1
		WHERE id = ? AND course_id = ?`, completedAt), status, blockID, courseID)
	return err
}

// UndoLearningBlockCompletion returns a completed block to pending. Blocks in
// any other state (skipped, postponed, already pending) are left alone.
func (r *Repository) UndoLearningBlockCompletion(courseID, blockID int64) (bool, error) {
	res, err := r.db.Exec(`UPDATE learning_blocks SET status = 'pending', completed_at = NULL,
		updated_at = CURRENT_TIMESTAMP, sync_version = sync_version + 1
		WHERE id = ? AND course_id = ? AND status = 'completed'`, blockID, courseID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *Repository) DeleteLearningBlock(courseID, blockID int64) error {
	_, err := r.db.Exec(`DELETE FROM learning_blocks WHERE id = ? AND course_id = ?`, blockID, courseID)
	return err
}

// PostponeLearningBlock marks a pending block 'postponed' and spawns a fresh
// pending catch-up block due on newDue. Only a pending block is postponed: any
// other status (completed, skipped, already postponed) or a block of another
// course is a no-op that returns false. The UPDATE and the catch-up INSERT run
// in one transaction, and the catch-up is created only when the UPDATE changed
// a row, so a repeated Later never duplicates it. This is the only place
// catch-ups are created.
func (r *Repository) PostponeLearningBlock(courseID, blockID int64, newDue string) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE learning_blocks SET status = 'postponed',
		updated_at = CURRENT_TIMESTAMP, sync_version = sync_version + 1
		WHERE id = ? AND course_id = ? AND status = 'pending'`, blockID, courseID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	if _, err := tx.Exec(`INSERT INTO learning_blocks
		(course_id, title, block_type, status, start_time, end_time, due_date, url, repeat_type, notes, sort_order, is_catchup)
		SELECT course_id, title, block_type, 'pending', start_time, end_time, ?, url, 'none', notes, sort_order, 1
		FROM learning_blocks WHERE id = ? AND course_id = ?`,
		nullString(newDue), blockID, courseID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// LearningBlocksThrough returns pending dated blocks due on or before date
// (today's schedule plus anything overdue, including catch-ups).
func (r *Repository) LearningBlocksThrough(date string) ([]models.LearningBlock, error) {
	rows, err := r.db.Query(fmt.Sprintf(`SELECT %s FROM learning_blocks b
		JOIN courses c ON c.id = b.course_id
		WHERE b.status = 'pending' AND b.archived_at IS NULL
			AND b.due_date IS NOT NULL AND b.due_date <= ?
			AND c.status IN ('active', 'paused')
		ORDER BY b.due_date, CASE WHEN b.start_time IS NULL THEN 1 ELSE 0 END, b.start_time, b.id`,
		learningBlockColumns), date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBlocks(rows)
}

// CalendarBlocks returns dated, non-archived, non-skipped blocks for export.
// courseID == 0 means all courses.
func (r *Repository) CalendarBlocks(courseID int64) ([]models.LearningBlock, error) {
	query := fmt.Sprintf(`SELECT %s FROM learning_blocks b
		JOIN courses c ON c.id = b.course_id
		WHERE b.archived_at IS NULL AND b.due_date IS NOT NULL
			AND b.status NOT IN ('skipped', 'postponed')
			AND c.status != 'archived'`, learningBlockColumns)
	args := []any{}
	if courseID != 0 {
		query += ` AND b.course_id = ?`
		args = append(args, courseID)
	}
	query += ` ORDER BY b.due_date, b.start_time`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBlocks(rows)
}

func collectBlocks(rows *sql.Rows) ([]models.LearningBlock, error) {
	var blocks []models.LearningBlock
	for rows.Next() {
		b, err := scanLearningBlock(rows)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

// ProjectItemDeadlines returns pending project mission items that carry a due
// date, with their project name, for the calendar feed.
func (r *Repository) ProjectItemDeadlines() ([]models.DeadlineRow, error) {
	return scanDeadlineRows(r.db.Query(`SELECT i.uuid, i.title, COALESCE(i.due_date, ''), i.project_id, p.name
		FROM project_items i JOIN projects p ON p.id = i.project_id
		WHERE i.due_date IS NOT NULL AND i.due_date != '' AND i.status = 'pending'
			AND p.archived_at IS NULL
		ORDER BY i.due_date`))
}

// MilestoneDeadlines returns pending roadmap milestones that carry a due date,
// with their project name, for the calendar feed.
func (r *Repository) MilestoneDeadlines() ([]models.DeadlineRow, error) {
	return scanDeadlineRows(r.db.Query(`SELECT m.uuid, m.title, COALESCE(m.due_date, ''), m.project_id, p.name
		FROM roadmap_milestones m JOIN projects p ON p.id = m.project_id
		WHERE m.due_date IS NOT NULL AND m.due_date != '' AND m.status = 'pending'
			AND p.archived_at IS NULL
		ORDER BY m.due_date`))
}

// ReminderDeadlines returns active standalone (non-task) reminders that carry a
// date, for the calendar feed. Task-derived reminders are excluded: they mirror
// their source task's due date, which is already emitted as a task event, so
// emitting them too would duplicate the item.
func (r *Repository) ReminderDeadlines() ([]models.DeadlineRow, error) {
	rows, err := r.db.Query(`SELECT uuid, title, COALESCE(reminder_date, ''), COALESCE(reminder_time, '')
		FROM reminders
		WHERE status = 'active'
			AND (source_type != 'task' OR source_id IS NULL)
			AND reminder_date IS NOT NULL AND reminder_date != ''
		ORDER BY reminder_date, reminder_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.DeadlineRow
	for rows.Next() {
		var d models.DeadlineRow
		if err := rows.Scan(&d.UUID, &d.Title, &d.DueDate, &d.Time); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanDeadlineRows(rows *sql.Rows, err error) ([]models.DeadlineRow, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.DeadlineRow
	for rows.Next() {
		var d models.DeadlineRow
		if err := rows.Scan(&d.UUID, &d.Title, &d.DueDate, &d.ProjectID, &d.ProjectName); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
