package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lifeops-tracker/internal/models"
)

const roadmapProjectColumns = `p.id, p.uuid, COALESCE(p.parent_project_id, 0), p.name, p.description, p.category,
	p.project_kind, p.status, COALESCE(p.start_date, ''), COALESCE(p.target_date, ''), p.notes, p.unlock_notes,
	COALESCE(p.depends_on_project_id, 0), p.auto_unlock,
	p.created_at, p.updated_at, COALESCE(p.archived_at, '')`

func (r *Repository) ImportRoadmapPlan(plan models.RoadmapImportPlan) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var goalID int64
	if strings.TrimSpace(plan.Goal.Title) != "" {
		res, err := tx.Exec(`INSERT INTO goals
			(title, description, type, category, target_value, current_value, unit, start_date, end_date, status)
			VALUES (?, ?, 'yearly', 'coding', 100, 0, 'percent', ?, ?, ?)`,
			plan.Goal.Title,
			plan.Goal.Description,
			nullString(plan.ImportedDate),
			nullString(plan.Goal.Deadline),
			cleanGoalStatus(plan.Goal.Status),
		)
		if err != nil {
			return 0, err
		}
		goalID, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	}

	res, err := tx.Exec(`INSERT INTO roadmap_imports
		(goal_id, goal_title, goal_deadline, source_markdown)
		VALUES (?, ?, ?, ?)`,
		nullableInt64(goalID),
		plan.Goal.Title,
		nullString(plan.Goal.Deadline),
		plan.SourceMarkdown,
	)
	if err != nil {
		return 0, err
	}
	importID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	projectIDsByName := map[string]int64{}
	type importedProject struct {
		id   int64
		plan models.RoadmapProjectPlan
	}
	importedProjects := make([]importedProject, 0, len(plan.Projects))
	for _, project := range plan.Projects {
		if strings.TrimSpace(project.Name) == "" {
			continue
		}
		if project.ParentName != "" {
			project.ParentProjectID = projectIDsByName[strings.ToLower(project.ParentName)]
		}
		if project.DependsOn != "" {
			project.DependsOnProjectID = projectIDsByName[strings.ToLower(project.DependsOn)]
		}
		projectID, err := insertRoadmapProject(tx, importID, plan.ImportedDate, project)
		if err != nil {
			return 0, err
		}
		projectIDsByName[strings.ToLower(project.Name)] = projectID
		importedProjects = append(importedProjects, importedProject{id: projectID, plan: project})
		for _, milestone := range project.Milestones {
			if err := insertRoadmapMilestone(tx, projectID, milestone); err != nil {
				return 0, err
			}
		}
		for _, task := range project.Tasks {
			if err := insertRoadmapTask(tx, projectID, task); err != nil {
				return 0, err
			}
		}
		for _, resource := range project.Resources {
			if err := insertRoadmapResource(tx, importID, projectID, resource); err != nil {
				return 0, err
			}
		}
	}

	// Resolve relationships after every project has an ID. This supports
	// dependencies declared before their target appears in the Markdown while
	// preserving the normal parent-before-child hierarchy.
	for _, imported := range importedProjects {
		// The first pass already linked the parent that existed when this
		// project was inserted. Re-resolve by name only when it found none:
		// a later project with the same name must not become the parent.
		parentID := imported.plan.ParentProjectID
		if parentID == 0 && imported.plan.ParentName != "" {
			parentID = projectIDsByName[strings.ToLower(imported.plan.ParentName)]
		}
		cycle, err := parentCreatesCycle(tx, imported.id, parentID)
		if err != nil {
			return 0, err
		}
		if cycle {
			return 0, ErrProjectParentCycle
		}
		dependencyID := imported.plan.DependsOnProjectID
		if imported.plan.DependsOn != "" {
			dependencyID = projectIDsByName[strings.ToLower(imported.plan.DependsOn)]
		}
		if _, err := tx.Exec(`UPDATE projects
			SET parent_project_id = ?, depends_on_project_id = ?
			WHERE id = ?`, nullableInt64(parentID), nullableInt64(dependencyID), imported.id); err != nil {
			return 0, err
		}
	}

	for _, resource := range plan.Resources {
		if err := insertRoadmapResource(tx, importID, 0, resource); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return importID, nil
}

func insertRoadmapProject(tx *sql.Tx, importID int64, importedDate string, project models.RoadmapProjectPlan) (int64, error) {
	res, err := tx.Exec(`INSERT INTO projects
		(parent_project_id, name, description, category, project_kind, status, start_date, target_date, notes,
		 unlock_notes, depends_on_project_id, auto_unlock, roadmap_id)
		VALUES (?, ?, ?, 'coding', ?, ?, ?, ?, ?, ?, ?, ?, (`+dashboardRoadmapIDSQL+`))`,
		nullableInt64(project.ParentProjectID),
		project.Name,
		project.Description,
		cleanProjectKind(project.ProjectKind),
		cleanProjectStatus(project.Status),
		nullString(importedDate),
		nullString(project.Deadline),
		firstNonEmpty(project.Notes, project.Description),
		project.UnlockNotes,
		nullableInt64(project.DependsOnProjectID),
		boolFlag(project.AutoUnlock),
	)
	if err != nil {
		return 0, err
	}
	projectID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`INSERT INTO roadmap_project_meta
		(project_id, source_import_id, project_type, tags, is_private, local_only, current_focus, public_summary)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		projectID,
		importID,
		models.NormalizeRoadmapProjectType(project.ProjectType),
		strings.Join(uniqueStrings(project.Tags), ", "),
		boolFlag(project.IsPrivate),
		boolFlag(project.LocalOnly),
		boolFlag(project.CurrentFocus),
		project.PublicSummary,
	)
	return projectID, err
}

func insertRoadmapMilestone(tx *sql.Tx, projectID int64, milestone models.RoadmapMilestonePlan) error {
	if strings.TrimSpace(milestone.Title) == "" {
		return nil
	}
	if milestone.SortOrder == 0 {
		milestone.SortOrder = 1
	}
	status := cleanItemStatus(milestone.Status)
	_, err := tx.Exec(`INSERT INTO roadmap_milestones
		(project_id, title, description, status, due_date, sort_order, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, CASE WHEN ? = 'completed' THEN CURRENT_TIMESTAMP ELSE NULL END)`,
		projectID,
		milestone.Title,
		milestone.Description,
		status,
		nullString(milestone.DueDate),
		milestone.SortOrder,
		status,
	)
	return err
}

func insertRoadmapTask(tx *sql.Tx, projectID int64, task models.RoadmapTaskPlan) error {
	if strings.TrimSpace(task.Title) == "" {
		return nil
	}
	if task.SortOrder == 0 {
		task.SortOrder = 1
	}
	status := cleanItemStatus(task.Status)
	_, err := tx.Exec(`INSERT INTO project_items
		(project_id, title, description, status, priority, due_date, sort_order,
		 stars_reward, proof_required, proof_type, stars_awarded, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		 CASE WHEN ? = 'completed' THEN ? ELSE 0 END,
		 CASE WHEN ? = 'completed' THEN CURRENT_TIMESTAMP ELSE NULL END)`,
		projectID,
		task.Title,
		task.Description,
		status,
		cleanPriority(task.Priority),
		nullString(task.DueDate),
		task.SortOrder,
		max(task.StarsReward, 1),
		boolFlag(task.ProofRequired),
		cleanProofType(task.ProofType),
		status,
		max(task.StarsReward, 1),
		status,
	)
	return err
}

func insertRoadmapResource(tx *sql.Tx, importID, projectID int64, resource models.RoadmapResourcePlan) error {
	if strings.TrimSpace(resource.Title) == "" && strings.TrimSpace(resource.URL) == "" {
		return nil
	}
	title := strings.TrimSpace(resource.Title)
	if title == "" {
		title = resource.URL
	}
	_, err := tx.Exec(`INSERT INTO roadmap_resources
		(project_id, source_import_id, title, url, resource_type, notes, is_private)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		nullableInt64(projectID),
		importID,
		title,
		resource.URL,
		models.NormalizeRoadmapResourceType(resource.ResourceType),
		resource.Notes,
		boolFlag(resource.IsPrivate),
	)
	return err
}

func (r *Repository) ListRoadmapProjects(includeArchived bool) ([]models.RoadmapProject, error) {
	where := "WHERE p.archived_at IS NULL AND p.status != 'archived'"
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
			LIMIT 1), '') AS next_due_date,
		m.id,
		COALESCE(m.source_import_id, 0),
		m.project_type,
		m.tags,
		m.is_private,
		m.local_only,
		m.current_focus,
		m.public_summary,
		m.created_at,
		m.updated_at
		FROM roadmap_project_meta m
		JOIN projects p ON p.id = m.project_id
		%s
		ORDER BY
			m.current_focus DESC,
			CASE p.status WHEN 'active' THEN 0 WHEN 'upcoming' THEN 1 WHEN 'locked' THEN 2 WHEN 'done' THEN 3 ELSE 4 END,
			CASE p.project_kind WHEN 'main' THEN 0 WHEN 'subproject' THEN 1 ELSE 2 END,
			CASE WHEN p.target_date IS NULL THEN 1 ELSE 0 END,
			p.target_date,
			p.created_at DESC`, roadmapProjectColumns, where))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.RoadmapProject
	for rows.Next() {
		var item models.RoadmapProject
		var privateFlag, localOnlyFlag, currentFocusFlag, autoUnlockFlag int
		if err := rows.Scan(
			&item.Project.Project.ID,
			&item.Project.Project.UUID,
			&item.Project.Project.ParentProjectID,
			&item.Project.Project.Name,
			&item.Project.Project.Description,
			&item.Project.Project.Category,
			&item.Project.Project.ProjectKind,
			&item.Project.Project.Status,
			&item.Project.Project.StartDate,
			&item.Project.Project.TargetDate,
			&item.Project.Project.Notes,
			&item.Project.Project.UnlockNotes,
			&item.Project.Project.DependsOnProjectID,
			&autoUnlockFlag,
			&item.Project.Project.CreatedAt,
			&item.Project.Project.UpdatedAt,
			&item.Project.Project.ArchivedAt,
			&item.Project.TotalItems,
			&item.Project.CompletedItems,
			&item.Project.NextDueItem,
			&item.Project.NextDueDate,
			&item.Meta.ID,
			&item.Meta.SourceImportID,
			&item.Meta.ProjectType,
			&item.Meta.TagsText,
			&privateFlag,
			&localOnlyFlag,
			&currentFocusFlag,
			&item.Meta.PublicSummary,
			&item.Meta.CreatedAt,
			&item.Meta.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Meta.ProjectID = item.Project.Project.ID
		item.Meta.ProjectType = models.NormalizeRoadmapProjectType(item.Meta.ProjectType)
		item.Meta.ProjectTypeLabel = models.RoadmapProjectTypeLabel(item.Meta.ProjectType)
		item.Meta.Tags = splitCSV(item.Meta.TagsText)
		item.Meta.IsPrivate = privateFlag == 1
		item.Meta.LocalOnly = localOnlyFlag == 1
		item.Meta.CurrentFocus = currentFocusFlag == 1
		item.Project.Project.AutoUnlock = autoUnlockFlag == 1
		item.Project.Progress = itemProgress(item.Project.CompletedItems, item.Project.TotalItems)
		projects = append(projects, item)
	}
	return projects, rows.Err()
}

func (r *Repository) RoadmapMilestones(projectID int64) ([]models.RoadmapMilestone, error) {
	rows, err := r.db.Query(`SELECT id, project_id, title, description, status,
		COALESCE(due_date, ''), sort_order, created_at, updated_at, COALESCE(completed_at, '')
		FROM roadmap_milestones
		WHERE project_id = ?
		ORDER BY status = 'completed', sort_order, due_date, created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var milestones []models.RoadmapMilestone
	for rows.Next() {
		var milestone models.RoadmapMilestone
		if err := rows.Scan(
			&milestone.ID,
			&milestone.ProjectID,
			&milestone.Title,
			&milestone.Description,
			&milestone.Status,
			&milestone.DueDate,
			&milestone.SortOrder,
			&milestone.CreatedAt,
			&milestone.UpdatedAt,
			&milestone.CompletedAt,
		); err != nil {
			return nil, err
		}
		milestones = append(milestones, milestone)
	}
	return milestones, rows.Err()
}

func (r *Repository) RoadmapResources(projectID int64) ([]models.RoadmapResource, error) {
	rows, err := r.db.Query(`SELECT id, COALESCE(project_id, 0), COALESCE(milestone_id, 0),
		COALESCE(project_item_id, 0), COALESCE(source_import_id, 0), title, url,
		resource_type, notes, is_private, created_at, updated_at
		FROM roadmap_resources
		WHERE project_id = ?
		ORDER BY resource_type, title`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRoadmapResources(rows)
}

func (r *Repository) GlobalRoadmapResources() ([]models.RoadmapResource, error) {
	rows, err := r.db.Query(`SELECT id, COALESCE(project_id, 0), COALESCE(milestone_id, 0),
		COALESCE(project_item_id, 0), COALESCE(source_import_id, 0), title, url,
		resource_type, notes, is_private, created_at, updated_at
		FROM roadmap_resources
		WHERE project_id IS NULL
		ORDER BY resource_type, title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRoadmapResources(rows)
}

func scanRoadmapResources(rows *sql.Rows) ([]models.RoadmapResource, error) {
	var resources []models.RoadmapResource
	for rows.Next() {
		var resource models.RoadmapResource
		var privateFlag int
		if err := rows.Scan(
			&resource.ID,
			&resource.ProjectID,
			&resource.MilestoneID,
			&resource.ProjectItemID,
			&resource.SourceImportID,
			&resource.Title,
			&resource.URL,
			&resource.ResourceType,
			&resource.Notes,
			&privateFlag,
			&resource.CreatedAt,
			&resource.UpdatedAt,
		); err != nil {
			return nil, err
		}
		resource.IsPrivate = privateFlag == 1
		resources = append(resources, resource)
	}
	return resources, rows.Err()
}

func (r *Repository) LatestRoadmapImport() (models.RoadmapImport, error) {
	var item models.RoadmapImport
	err := r.db.QueryRow(`SELECT id, uuid, COALESCE(goal_id, 0), goal_title,
		COALESCE(goal_deadline, ''), source_markdown, imported_at, created_at, updated_at
		FROM roadmap_imports
		ORDER BY imported_at DESC, id DESC
		LIMIT 1`).Scan(
		&item.ID,
		&item.UUID,
		&item.GoalID,
		&item.GoalTitle,
		&item.GoalDeadline,
		&item.SourceMarkdown,
		&item.ImportedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.RoadmapImport{}, nil
	}
	return item, err
}

func (r *Repository) ListRoadmapWeeklyReviews(limit int) ([]models.RoadmapWeeklyReview, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := r.db.Query(`SELECT id, week_start, finished, blocked, next_steps, created_at, updated_at
		FROM roadmap_weekly_reviews
		ORDER BY week_start DESC, created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.RoadmapWeeklyReview
	for rows.Next() {
		var review models.RoadmapWeeklyReview
		if err := rows.Scan(
			&review.ID,
			&review.WeekStart,
			&review.Finished,
			&review.Blocked,
			&review.NextSteps,
			&review.CreatedAt,
			&review.UpdatedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func (r *Repository) CreateRoadmapWeeklyReview(review *models.RoadmapWeeklyReview) error {
	res, err := r.db.Exec(`INSERT INTO roadmap_weekly_reviews
		(week_start, finished, blocked, next_steps)
		VALUES (?, ?, ?, ?)`,
		review.WeekStart,
		review.Finished,
		review.Blocked,
		review.NextSteps,
	)
	if err != nil {
		return err
	}
	review.ID, err = res.LastInsertId()
	return err
}

func (r *Repository) SetRoadmapProjectMeta(meta models.RoadmapProjectMeta) error {
	_, err := r.db.Exec(`UPDATE roadmap_project_meta SET
		project_type = ?,
		tags = ?,
		is_private = ?,
		local_only = ?,
		current_focus = ?,
		public_summary = ?,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE project_id = ?`,
		models.NormalizeRoadmapProjectType(meta.ProjectType),
		meta.TagsText,
		boolFlag(meta.IsPrivate),
		boolFlag(meta.LocalOnly),
		boolFlag(meta.CurrentFocus),
		meta.PublicSummary,
		meta.ProjectID,
	)
	return err
}

// EffectiveProjectPrivacy returns, for every project that is private or
// local-only itself, through any ancestor, or through its roadmap, its
// effective flags. Projects missing from the map are public. Roadmap privacy
// ('private' or 'local_only') seeds every project of that roadmap, and the
// walk carries it to all descendants whatever roadmap they point at. Orphan
// projects (roadmap_id NULL) use the roadmap GetDashboardRoadmap would pick;
// with no active roadmap they are not hidden by roadmap privacy. Archived
// rows and archived roadmaps are not filtered, and UNION (not UNION ALL)
// makes the walk terminate on parent cycles.
func (r *Repository) EffectiveProjectPrivacy() (map[int64]models.ProjectPrivacy, error) {
	// The orphan fallback subquery is dashboardRoadmapIDSQL, which mirrors
	// GetDashboardRoadmap (internal/repository/roadmaps.go).
	rows, err := r.db.Query(`WITH RECURSIVE f(id, priv, loc) AS (
		SELECT id, priv, loc FROM (
			SELECT project_id AS id, is_private AS priv, local_only AS loc FROM roadmap_project_meta WHERE is_private = 1 OR local_only = 1
			UNION ALL
			SELECT p.id, r.privacy = 'private', r.privacy = 'local_only'
			FROM projects p
			JOIN roadmaps r ON r.id = COALESCE(p.roadmap_id, (` + dashboardRoadmapIDSQL + `))
			WHERE r.privacy IN ('private', 'local_only')
		)
		UNION
		SELECT p.id, f.priv, f.loc FROM projects p JOIN f ON p.parent_project_id = f.id
	) SELECT id, MAX(priv), MAX(loc) FROM f GROUP BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]models.ProjectPrivacy)
	for rows.Next() {
		var id int64
		var priv, loc int
		if err := rows.Scan(&id, &priv, &loc); err != nil {
			return nil, err
		}
		out[id] = models.ProjectPrivacy{Private: priv == 1, LocalOnly: loc == 1}
	}
	return out, rows.Err()
}

func (r *Repository) SetRoadmapMilestoneStatus(id int64, status string) error {
	status = cleanItemStatus(status)
	completedAt := "NULL"
	if status == "completed" {
		completedAt = "CURRENT_TIMESTAMP"
	}
	_, err := r.db.Exec(fmt.Sprintf(`UPDATE roadmap_milestones SET
		status = ?,
		completed_at = %s,
		updated_at = CURRENT_TIMESTAMP,
		sync_version = sync_version + 1
		WHERE id = ?`, completedAt), status, id)
	return err
}

func boolFlag(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func cleanGoalStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "completed", "archived":
		return strings.TrimSpace(strings.ToLower(status))
	default:
		return "active"
	}
}

func cleanProjectStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "upcoming", "locked", "done", "archived":
		return strings.TrimSpace(strings.ToLower(status))
	case "paused":
		return "upcoming"
	case "completed", "complete":
		return "done"
	default:
		return "active"
	}
}

func cleanItemStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "completed", "done", "complete", "x":
		return "completed"
	default:
		return "pending"
	}
}

func cleanPriority(priority string) string {
	switch strings.TrimSpace(strings.ToLower(priority)) {
	case "low", "high":
		return strings.TrimSpace(strings.ToLower(priority))
	default:
		return "medium"
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	return uniqueStrings(parts)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var cleaned []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
