package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lifeops-tracker/internal/models"
)

const projectColumns = `id, uuid, COALESCE(parent_project_id, 0), name, description, category, project_kind, status,
	COALESCE(start_date, ''), COALESCE(target_date, ''), notes, unlock_notes,
	COALESCE(depends_on_project_id, 0), auto_unlock,
	created_at, updated_at, COALESCE(archived_at, ''), COALESCE(roadmap_id, 0)`

func (r *Repository) ListProjects(includeArchived bool) ([]models.ProjectWithProgress, error) {
	where := "WHERE p.archived_at IS NULL AND p.status != 'archived'"
	if includeArchived {
		where = ""
	}
	return r.listProjectsWhere(where)
}

// ListProjectsByRoadmap returns the projects (trips) that belong to one roadmap.
func (r *Repository) ListProjectsByRoadmap(roadmapID int64) ([]models.ProjectWithProgress, error) {
	return r.listProjectsWhere("WHERE p.roadmap_id = ? AND p.archived_at IS NULL AND p.status != 'archived'", roadmapID)
}

// TopLevelProjectsByRoadmap returns only the roadmap's top-level destinations
// (parent_project_id IS NULL). Subprojects never appear on the galaxy route;
// they live inside their parent's solar system. All non-archived statuses are
// included so done destinations still glow and locked ones still show dimmed.
func (r *Repository) TopLevelProjectsByRoadmap(roadmapID int64) ([]models.ProjectWithProgress, error) {
	return r.listProjectsWhere(
		"WHERE p.roadmap_id = ? AND p.parent_project_id IS NULL AND p.archived_at IS NULL AND p.status != 'archived'",
		roadmapID)
}

// SubProjects returns the child projects (planets/stations) of one parent, any
// non-archived status so locked/upcoming children still render dimmed.
func (r *Repository) SubProjects(parentID int64) ([]models.ProjectWithProgress, error) {
	return r.listProjectsWhere(
		"WHERE p.parent_project_id = ? AND p.archived_at IS NULL AND p.status != 'archived'",
		parentID)
}

func (r *Repository) listProjectsWhere(where string, args ...any) ([]models.ProjectWithProgress, error) {
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
			CASE status WHEN 'active' THEN 0 WHEN 'upcoming' THEN 1 WHEN 'locked' THEN 2 WHEN 'done' THEN 3 ELSE 4 END,
			CASE project_kind WHEN 'main' THEN 0 WHEN 'subproject' THEN 1 ELSE 2 END,
			CASE WHEN target_date IS NULL THEN 1 ELSE 0 END,
			target_date,
			created_at DESC`, projectColumns, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.ProjectWithProgress
	for rows.Next() {
		var item models.ProjectWithProgress
		var autoUnlockFlag int
		if err := rows.Scan(
			&item.Project.ID,
			&item.Project.UUID,
			&item.Project.ParentProjectID,
			&item.Project.Name,
			&item.Project.Description,
			&item.Project.Category,
			&item.Project.ProjectKind,
			&item.Project.Status,
			&item.Project.StartDate,
			&item.Project.TargetDate,
			&item.Project.Notes,
			&item.Project.UnlockNotes,
			&item.Project.DependsOnProjectID,
			&autoUnlockFlag,
			&item.Project.CreatedAt,
			&item.Project.UpdatedAt,
			&item.Project.ArchivedAt,
			&item.Project.RoadmapID,
			&item.TotalItems,
			&item.CompletedItems,
			&item.NextDueItem,
			&item.NextDueDate,
		); err != nil {
			return nil, err
		}
		item.Project.AutoUnlock = autoUnlockFlag == 1
		item.Progress = itemProgress(item.CompletedItems, item.TotalItems)
		projects = append(projects, item)
	}
	return projects, rows.Err()
}

func (r *Repository) ProjectTree() (models.ProjectTree, error) {
	projects, err := r.ListProjects(false)
	if err != nil {
		return models.ProjectTree{}, err
	}
	return buildProjectTree(projects), nil
}

// ProjectTreeByRoadmap builds the trip tree scoped to a single roadmap, used by
// the roadmap-scoped dashboard hero and the Navigation Deck filter.
func (r *Repository) ProjectTreeByRoadmap(roadmapID int64) (models.ProjectTree, error) {
	projects, err := r.ListProjectsByRoadmap(roadmapID)
	if err != nil {
		return models.ProjectTree{}, err
	}
	return buildProjectTree(projects), nil
}

func buildProjectTree(projects []models.ProjectWithProgress) models.ProjectTree {
	tree := models.ProjectTree{Children: map[int64][]models.ProjectWithProgress{}}
	for _, project := range projects {
		project.Momentum = models.ChartData{}
		// Parentage is decided before status: a subproject (any status, incl.
		// locked/upcoming) always nests under its parent and never appears as a
		// top-level destination. Only top-level planned trips reach Upcoming.
		if project.Project.ParentProjectID != 0 {
			tree.Children[project.Project.ParentProjectID] = append(tree.Children[project.Project.ParentProjectID], project)
			continue
		}
		if project.Project.Status == "locked" || project.Project.Status == "upcoming" {
			tree.Upcoming = append(tree.Upcoming, project)
			continue
		}
		tree.Main = append(tree.Main, project)
	}
	return tree
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
	project.Status = cleanProjectStatus(project.Status)
	project.ProjectKind = cleanProjectKind(project.ProjectKind)

	res, err := r.db.Exec(`INSERT INTO projects
		(parent_project_id, name, description, category, project_kind, status, start_date, target_date, notes,
		 unlock_notes, depends_on_project_id, auto_unlock, roadmap_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullableInt64(project.ParentProjectID),
		project.Name,
		project.Description,
		project.Category,
		project.ProjectKind,
		project.Status,
		nullString(project.StartDate),
		nullString(project.TargetDate),
		project.Notes,
		project.UnlockNotes,
		nullableInt64(project.DependsOnProjectID),
		boolFlag(project.AutoUnlock),
		nullableInt64(project.RoadmapID),
	)
	if err != nil {
		return err
	}
	project.ID, err = res.LastInsertId()
	return err
}

func (r *Repository) UpdateProject(project *models.Project) error {
	project.Category = clean(project.Category, "other")
	project.Status = cleanProjectStatus(project.Status)
	project.ProjectKind = cleanProjectKind(project.ProjectKind)

	_, err := r.db.Exec(`UPDATE projects SET
		parent_project_id = ?,
		name = ?,
		description = ?,
		category = ?,
		project_kind = ?,
		status = ?,
		start_date = ?,
		target_date = ?,
		notes = ?,
		unlock_notes = ?,
		depends_on_project_id = ?,
		auto_unlock = ?,
		roadmap_id = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`,
		nullableInt64(project.ParentProjectID),
		project.Name,
		project.Description,
		project.Category,
		project.ProjectKind,
		project.Status,
		nullString(project.StartDate),
		nullString(project.TargetDate),
		project.Notes,
		project.UnlockNotes,
		nullableInt64(project.DependsOnProjectID),
		boolFlag(project.AutoUnlock),
		nullableInt64(project.RoadmapID),
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

func (r *Repository) SetProjectStatus(id int64, status string) error {
	status = cleanProjectStatus(status)
	if _, err := r.db.Exec(`UPDATE projects SET
		status = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, status, id); err != nil {
		return err
	}
	if status == "done" {
		return r.AutoUnlockDependents(id, "upcoming")
	}
	return nil
}

func (r *Repository) MoveProject(id, parentID int64, kind string) error {
	_, err := r.db.Exec(`UPDATE projects SET
		parent_project_id = ?,
		project_kind = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`,
		nullableInt64(parentID),
		cleanProjectKind(kind),
		id,
	)
	return err
}

func (r *Repository) UpdateProjectNotes(id int64, notes, unlockNotes string) error {
	_, err := r.db.Exec(`UPDATE projects SET
		notes = ?,
		unlock_notes = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, notes, unlockNotes, id)
	return err
}

// CountUnlockableDependents reports how many locked, auto-unlock projects
// depend on the given project and would unlock if it were marked done. Used
// only to decide whether to show the "New Destination Unlocked" cue.
func (r *Repository) CountUnlockableDependents(projectID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM projects
		WHERE depends_on_project_id = ?
			AND auto_unlock = 1
			AND status = 'locked'`, projectID).Scan(&count)
	return count, err
}

func (r *Repository) AutoUnlockDependents(doneProjectID int64, targetStatus string) error {
	targetStatus = cleanProjectStatus(targetStatus)
	if targetStatus == "done" || targetStatus == "archived" {
		targetStatus = "upcoming"
	}
	_, err := r.db.Exec(`UPDATE projects SET
		status = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE depends_on_project_id = ?
			AND auto_unlock = 1
			AND status = 'locked'`,
		targetStatus,
		doneProjectID,
	)
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
	if status == "archived" || status == "upcoming" || status == "locked" {
		return nil
	}

	nextStatus := "active"
	if total > 0 && total == completed {
		nextStatus = "done"
	}
	if status == nextStatus {
		return nil
	}

	_, err := r.db.Exec(`UPDATE projects SET
		status = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, nextStatus, projectID)
	if err != nil {
		return err
	}
	if nextStatus == "done" {
		if _, err := r.db.Exec(`INSERT OR IGNORE INTO achievements
			(title, description, stars_bonus, trigger_type, source_project_id)
			VALUES ('Quest complete', (SELECT name FROM projects WHERE id = ?), 10, 'project_done', ?)`, projectID, projectID); err != nil {
			return err
		}
		return r.AutoUnlockDependents(projectID, "upcoming")
	}
	return nil
}

func cleanProjectKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "subproject", "phase":
		return strings.TrimSpace(strings.ToLower(kind))
	default:
		return "main"
	}
}
