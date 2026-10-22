package repository

import (
	"fmt"

	"lifeops-tracker/internal/models"
)

const roadmapColumns = `id, uuid, name, description, priority, status, privacy, featured,
	icon, color, created_at, updated_at, COALESCE(archived_at, '')`

func scanRoadmap(s scanner) (models.Roadmap, error) {
	var roadmap models.Roadmap
	var featured int
	err := s.Scan(
		&roadmap.ID,
		&roadmap.UUID,
		&roadmap.Name,
		&roadmap.Description,
		&roadmap.Priority,
		&roadmap.Status,
		&roadmap.Privacy,
		&featured,
		&roadmap.Icon,
		&roadmap.Color,
		&roadmap.CreatedAt,
		&roadmap.UpdatedAt,
		&roadmap.ArchivedAt,
	)
	roadmap.Featured = featured == 1
	return roadmap, err
}

// ListRoadmaps returns roadmaps ordered for the sidebar: featured first, then by
// priority, then most-recently updated.
func (r *Repository) ListRoadmaps(includeArchived bool) ([]models.Roadmap, error) {
	where := "WHERE archived_at IS NULL AND status != 'archived'"
	if includeArchived {
		where = ""
	}
	rows, err := r.db.Query(fmt.Sprintf(`SELECT %s FROM roadmaps %s
		ORDER BY featured DESC, priority DESC, updated_at DESC, id DESC`, roadmapColumns, where))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roadmaps []models.Roadmap
	for rows.Next() {
		roadmap, err := scanRoadmap(rows)
		if err != nil {
			return nil, err
		}
		roadmaps = append(roadmaps, roadmap)
	}
	return roadmaps, rows.Err()
}

func (r *Repository) GetRoadmap(id int64) (models.Roadmap, error) {
	return scanRoadmap(r.db.QueryRow(fmt.Sprintf(`SELECT %s FROM roadmaps WHERE id = ?`, roadmapColumns), id))
}

func (r *Repository) CreateRoadmap(roadmap *models.Roadmap) error {
	res, err := r.db.Exec(`INSERT INTO roadmaps
		(name, description, priority, status, privacy, featured, icon, color)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		roadmap.Name,
		roadmap.Description,
		roadmap.Priority,
		cleanRoadmapStatus(roadmap.Status),
		cleanRoadmapPrivacy(roadmap.Privacy),
		boolFlag(roadmap.Featured),
		cleanRoadmapIcon(roadmap.Icon),
		cleanRoadmapColor(roadmap.Color),
	)
	if err != nil {
		return err
	}
	roadmap.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	if roadmap.Featured {
		if err := r.SetFeaturedRoadmap(roadmap.ID); err != nil {
			return err
		}
	}
	return r.AdoptOrphanProjects()
}

func (r *Repository) UpdateRoadmap(roadmap *models.Roadmap) error {
	_, err := r.db.Exec(`UPDATE roadmaps SET
		name = ?, description = ?, priority = ?, status = ?, privacy = ?,
		icon = ?, color = ?, updated_at = CURRENT_TIMESTAMP, sync_version = sync_version + 1
		WHERE id = ?`,
		roadmap.Name,
		roadmap.Description,
		roadmap.Priority,
		cleanRoadmapStatus(roadmap.Status),
		cleanRoadmapPrivacy(roadmap.Privacy),
		cleanRoadmapIcon(roadmap.Icon),
		cleanRoadmapColor(roadmap.Color),
		roadmap.ID,
	)
	if err != nil {
		return err
	}
	return r.AdoptOrphanProjects()
}

func (r *Repository) ArchiveRoadmap(id int64) error {
	_, err := r.db.Exec(`UPDATE roadmaps SET
		status = 'archived', archived_at = CURRENT_TIMESTAMP,
		featured = 0, updated_at = CURRENT_TIMESTAMP, sync_version = sync_version + 1
		WHERE id = ?`, id)
	return err
}

// SetFeaturedRoadmap makes exactly one roadmap featured, atomically.
func (r *Repository) SetFeaturedRoadmap(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE roadmaps SET featured = 0, updated_at = CURRENT_TIMESTAMP WHERE featured = 1 AND id != ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`UPDATE roadmaps SET featured = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// dashboardRoadmapIDSQL is a scalar subquery giving the id of the roadmap
// GetDashboardRoadmap picks (NULL when no active roadmap exists). Keep its
// filter and ORDER BY identical to GetDashboardRoadmap; EffectiveProjectPrivacy
// (roadmap.go) and the project and roadmap writes reuse it.
const dashboardRoadmapIDSQL = `SELECT id FROM roadmaps
	WHERE status = 'active' AND archived_at IS NULL
	ORDER BY featured DESC, priority DESC, updated_at DESC, id DESC LIMIT 1`

// GetDashboardRoadmap selects the single roadmap shown on the bridge:
// featured beats priority; among equals, the most recently updated wins.
// Steps 1-4 of the spec collapse to one ordered query over active roadmaps.
// dashboardRoadmapIDSQL repeats this selection as SQL; keep the two in sync.
func (r *Repository) GetDashboardRoadmap() (models.Roadmap, error) {
	return scanRoadmap(r.db.QueryRow(fmt.Sprintf(`SELECT %s FROM roadmaps
		WHERE status = 'active' AND archived_at IS NULL
		ORDER BY featured DESC, priority DESC, updated_at DESC, id DESC
		LIMIT 1`, roadmapColumns)))
}

// AdoptOrphanProjects assigns projects with no roadmap (created before this
// feature, via seed, or while no active roadmap existed) to the dashboard
// roadmap. It is a no-op without an active roadmap and idempotent. It runs at
// startup and after roadmap writes, never on a GET.
func (r *Repository) AdoptOrphanProjects() error {
	_, err := r.db.Exec(`UPDATE projects SET roadmap_id = (` + dashboardRoadmapIDSQL + `)
		WHERE roadmap_id IS NULL AND EXISTS (
			SELECT 1 FROM roadmaps WHERE status = 'active' AND archived_at IS NULL)`)
	return err
}

// HasRoadmap reports whether any non-archived roadmap exists (empty-state check).
func (r *Repository) HasRoadmap() (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM roadmaps WHERE archived_at IS NULL AND status != 'archived'`).Scan(&count)
	return count > 0, err
}

func cleanRoadmapStatus(value string) string {
	switch value {
	case "active", "upcoming", "locked", "done", "archived":
		return value
	default:
		return "active"
	}
}

func cleanRoadmapPrivacy(value string) string {
	switch value {
	case "public", "private", "local_only":
		return value
	default:
		return "private"
	}
}

func cleanRoadmapIcon(value string) string {
	if value == "" {
		return "rocket"
	}
	return value
}

func cleanRoadmapColor(value string) string {
	if value == "" {
		return "cyan"
	}
	return value
}
