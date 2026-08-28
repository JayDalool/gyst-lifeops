package repository

import (
	"database/sql"

	"lifeops-tracker/internal/models"
)

const externalLinkColumns = `id, uuid, entity_type, entity_id, provider,
	external_key, url, title, summary, status_text, sync_source,
	COALESCE(last_synced_at, ''), sync_version, last_modified_device_id,
	created_at, updated_at`

func scanExternalLink(s scanner) (models.ExternalLink, error) {
	var link models.ExternalLink
	err := s.Scan(&link.ID, &link.UUID, &link.EntityType, &link.EntityID,
		&link.Provider, &link.ExternalKey, &link.URL, &link.Title,
		&link.Summary, &link.StatusText, &link.SyncSource, &link.LastSyncedAt,
		&link.SyncVersion, &link.LastModifiedDeviceID, &link.CreatedAt, &link.UpdatedAt)
	return link, err
}

func (r *Repository) ExternalLinkEntityExists(entityType string, entityID int64) (bool, error) {
	var table string
	switch entityType {
	case "task":
		table = "tasks"
	case "project":
		table = "projects"
	case "project_item":
		table = "project_items"
	default:
		return false, nil
	}
	var exists int
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = ?)`, entityID).Scan(&exists)
	return exists == 1, err
}

func (r *Repository) CreateExternalLink(link *models.ExternalLink) error {
	res, err := r.db.Exec(`INSERT INTO external_links
		(uuid, entity_type, entity_id, provider, external_key, url, title, summary,
		 status_text, sync_source)
		VALUES (lower(hex(randomblob(16))), ?, ?, ?, ?, ?, ?, ?, ?, 'manual')`,
		link.EntityType, link.EntityID, link.Provider, link.ExternalKey, link.URL,
		link.Title, link.Summary, link.StatusText)
	if err != nil {
		return err
	}
	link.ID, err = res.LastInsertId()
	return err
}

func (r *Repository) ExternalLinks(entityType string, entityID int64) ([]models.ExternalLink, error) {
	rows, err := r.db.Query(`SELECT `+externalLinkColumns+` FROM external_links
		WHERE entity_type = ? AND entity_id = ? ORDER BY created_at, id`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var links []models.ExternalLink
	for rows.Next() {
		link, scanErr := scanExternalLink(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

func (r *Repository) ExternalLinkByID(id int64) (models.ExternalLink, error) {
	return scanExternalLink(r.db.QueryRow(`SELECT `+externalLinkColumns+` FROM external_links WHERE id = ?`, id))
}

func (r *Repository) UpdateExternalLinkSummary(id int64, summary string) error {
	res, err := r.db.Exec(`UPDATE external_links SET summary = ?, sync_version = sync_version + 1,
		updated_at = CURRENT_TIMESTAMP WHERE id = ?`, summary, id)
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

func (r *Repository) DeleteExternalLink(id int64) error {
	res, err := r.db.Exec(`DELETE FROM external_links WHERE id = ?`, id)
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
