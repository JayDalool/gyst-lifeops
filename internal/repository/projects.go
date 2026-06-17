package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"lifeops-tracker/internal/models"
)

const projectColumns = `id, uuid, name, description, category, status,
	COALESCE(start_date, ''), COALESCE(target_date, ''), notes,
	created_at, updated_at, COALESCE(archived_at, '')`

func (r *Repository) ListProjects(includeArchived bool) ([]models.ProjectWithProgress, error) {
	where := "WHERE archived_at IS NULL AND status != 'archived'"
	if includeArchived {
		where = ""
	}

	rows, err := r.db.Query(fmt.Sprintf(`SELECT %s,
		(SELECT COUNT(*) FROM project_items i WHERE i.project_id = p.id) AS total_items,
		(SELECT COUNT(*) FROM project_items i WHERE i.project_id = p.id AND i.status = 'completed') AS completed_items,
		COALESCE((SELECT i.title FROM project_items i
			WHERE i.project_id = p.id AND i.status = 'pending'
			ORDER BY CASE WHEN i.due_date IS NULL THEN 1 ELSE 0 END, i.due_date, i.sort_order
			LIMIT 1), '') AS next_due_item,
		COALESCE((SELECT i.due_date FROM project_items i
			WHERE i.project_id = p.id AND i.status = 'pending' AND i.due_date IS NOT NULL
			ORDER BY i.due_date, i.sort_order
			LIMIT 1), '')
		FROM projects p
		%s
		ORDER BY
			CASE status WHEN 'active' THEN 0 WHEN 'paused' THEN 1 WHEN 'completed' THEN 2 ELSE 3 END,
			CASE WHEN target_date IS NULL THEN 1 ELSE 0 END,
			target_date,
			created_at DESC`, projectColumns, where))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.ProjectWithProgress
	for rows.Next() {
		var item models.ProjectWithProgress
		if err := rows.Scan(
			&item.Project.ID,
			&item.Project.UUID,
			&item.Project.Name,
			&item.Project.Description,
			&item.Project.Category,
			&item.Project.Status,
			&item.Project.StartDate,
			&item.Project.TargetDate,
			&item.Project.Notes,
			&item.Project.CreatedAt,
			&item.Project.UpdatedAt,
			&item.Project.ArchivedAt,
			&item.TotalItems,
			&item.CompletedItems,
			&item.NextDueItem,
			&item.NextDueDate,
		); err != nil {
			return nil, err
		}
		item.Progress = itemProgress(item.CompletedItems, item.TotalItems)
		projects = append(projects, item)
	}
	return projects, rows.Err()
}

func (r *Repository) ProjectByID(id int64) (models.Project, error) {
	project, err := scanProject(r.db.QueryRow(fmt.Sprintf(`SELECT %s FROM projects WHERE id = ?`, projectColumns), id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Project{}, err
	}
	return project, err
}

func (r *Repository) ProjectWithProgress(id int64) (models.ProjectWithProgress, error) {
	project, err := r.ProjectByID(id)
	if err != nil {
		return models.ProjectWithProgress{}, err
	}

	var total, completed int
	var nextDueItem, nextDueDate string
	if err := r.db.QueryRow(`SELECT
		COUNT(*),
		COUNT(CASE WHEN status = 'completed' THEN 1 END),
		COALESCE((SELECT title FROM project_items WHERE project_id = ? AND status = 'pending'
			ORDER BY CASE WHEN due_date IS NULL THEN 1 ELSE 0 END, due_date, sort_order LIMIT 1), ''),
		COALESCE((SELECT due_date FROM project_items WHERE project_id = ? AND status = 'pending' AND due_date IS NOT NULL
			ORDER BY due_date, sort_order LIMIT 1), '')
		FROM project_items
		WHERE project_id = ?`, id, id, id).Scan(&total, &completed, &nextDueItem, &nextDueDate); err != nil {
		return models.ProjectWithProgress{}, err
	}

	return models.ProjectWithProgress{
		Project:        project,
		TotalItems:     total,
		CompletedItems: completed,
		Progress:       itemProgress(completed, total),
		NextDueItem:    nextDueItem,
		NextDueDate:    nextDueDate,
	}, nil
}

func (r *Repository) CreateProject(project *models.Project) error {
	project.Category = clean(project.Category, "other")
	project.Status = clean(project.Status, "active")

	res, err := r.db.Exec(`INSERT INTO projects
		(name, description, category, status, start_date, target_date, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		project.Name,
		project.Description,
		project.Category,
		project.Status,
		nullString(project.StartDate),
		nullString(project.TargetDate),
		project.Notes,
	)
	if err != nil {
		return err
	}
	project.ID, err = res.LastInsertId()
	return err
}

func (r *Repository) UpdateProject(project *models.Project) error {
	project.Category = clean(project.Category, "other")
	project.Status = clean(project.Status, "active")

	_, err := r.db.Exec(`UPDATE projects SET
		name = ?,
		description = ?,
		category = ?,
		status = ?,
		start_date = ?,
		target_date = ?,
		notes = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`,
		project.Name,
		project.Description,
		project.Category,
		project.Status,
		nullString(project.StartDate),
		nullString(project.TargetDate),
		project.Notes,
		project.ID,
	)
	return err
}

func (r *Repository) ArchiveProject(id int64) error {
	_, err := r.db.Exec(`UPDATE projects SET
		status = 'archived',
		archived_at = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, id)
	return err
}

func (r *Repository) ProjectItems(projectID int64) ([]models.ProjectItem, error) {
	rows, err := r.db.Query(`SELECT id, project_id, title, description, status, priority,
		COALESCE(due_date, ''), sort_order, created_at, updated_at, COALESCE(completed_at, '')
		FROM project_items
		WHERE project_id = ?
		ORDER BY status = 'completed', sort_order, due_date, created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ProjectItem
	for rows.Next() {
		var item models.ProjectItem
		if err := rows.Scan(
			&item.ID,
			&item.ProjectID,
			&item.Title,
			&item.Description,
			&item.Status,
			&item.Priority,
			&item.DueDate,
			&item.SortOrder,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.CompletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateProjectItem(item *models.ProjectItem) error {
	item.Priority = clean(item.Priority, "medium")
	item.Status = clean(item.Status, "pending")
	if item.SortOrder == 0 {
		if err := r.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM project_items WHERE project_id = ?`, item.ProjectID).Scan(&item.SortOrder); err != nil {
			return err
		}
	}

	res, err := r.db.Exec(`INSERT INTO project_items
		(project_id, title, description, status, priority, due_date, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ProjectID,
		item.Title,
		item.Description,
		item.Status,
		item.Priority,
		nullString(item.DueDate),
		item.SortOrder,
	)
	if err != nil {
		return err
	}
	item.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	return r.reconcileProjectStatus(item.ProjectID)
}

func (r *Repository) SetProjectItemStatus(projectID, itemID int64, status string) error {
	if status != "pending" && status != "completed" {
		return fmt.Errorf("unsupported project item status %q", status)
	}

	completedAt := "NULL"
	if status == "completed" {
		completedAt = "CURRENT_TIMESTAMP"
	}

	if _, err := r.db.Exec(fmt.Sprintf(`UPDATE project_items SET
		status = ?,
		completed_at = %s,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ? AND project_id = ?`, completedAt), status, itemID, projectID); err != nil {
		return err
	}
	return r.reconcileProjectStatus(projectID)
}

func (r *Repository) DeleteProjectItem(projectID, itemID int64) error {
	if _, err := r.db.Exec(`DELETE FROM project_items WHERE id = ? AND project_id = ?`, itemID, projectID); err != nil {
		return err
	}
	return r.reconcileProjectStatus(projectID)
}

func (r *Repository) reconcileProjectStatus(projectID int64) error {
	var total, completed int
	var status string
	if err := r.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM project_items WHERE project_id = ?),
		(SELECT COUNT(*) FROM project_items WHERE project_id = ? AND status = 'completed'),
		(SELECT status FROM projects WHERE id = ?)`,
		projectID, projectID, projectID).Scan(&total, &completed, &status); err != nil {
		return err
	}
	if status == "archived" || status == "paused" {
		return nil
	}

	nextStatus := "active"
	if total > 0 && total == completed {
		nextStatus = "completed"
	}
	if status == nextStatus {
		return nil
	}

	_, err := r.db.Exec(`UPDATE projects SET
		status = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, nextStatus, projectID)
	return err
}
