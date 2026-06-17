package repository

import (
	"database/sql"
	"fmt"

	"lifeops-tracker/internal/models"
)

func (r *Repository) ListTrackerItems() ([]models.TrackerItem, error) {
	rows, err := r.db.Query(`SELECT id, uuid, key, title, type, category, sort_order, is_active, created_at, updated_at
		FROM tracker_items
		WHERE is_active = 1
		ORDER BY sort_order, title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.TrackerItem
	for rows.Next() {
		var item models.TrackerItem
		var active int
		if err := rows.Scan(
			&item.ID,
			&item.UUID,
			&item.Key,
			&item.Title,
			&item.Type,
			&item.Category,
			&item.SortOrder,
			&active,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.IsActive = active == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) TrackerLogsBetween(startDate, endDate string) ([]models.TrackerLog, error) {
	rows, err := r.db.Query(`SELECT id, tracker_item_id, log_date, status, value_numeric, note, created_at, updated_at
		FROM tracker_logs
		WHERE log_date >= ? AND log_date <= ?
		ORDER BY log_date, tracker_item_id`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.TrackerLog
	for rows.Next() {
		var log models.TrackerLog
		var value sql.NullFloat64
		if err := rows.Scan(
			&log.ID,
			&log.TrackerItemID,
			&log.LogDate,
			&log.Status,
			&value,
			&log.Note,
			&log.CreatedAt,
			&log.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if value.Valid {
			log.ValueNumeric = value.Float64
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *Repository) SetTrackerLogByKey(key, logDate, status, note string, valueNumeric *float64) error {
	if status != "done" && status != "not_done" && status != "skipped" && status != "rest" {
		return fmt.Errorf("unsupported tracker status %q", status)
	}

	var itemID int64
	if err := r.db.QueryRow(`SELECT id FROM tracker_items WHERE key = ? AND is_active = 1`, key).Scan(&itemID); err != nil {
		return err
	}

	var value any
	if valueNumeric != nil {
		value = *valueNumeric
	}

	_, err := r.db.Exec(`INSERT INTO tracker_logs
		(tracker_item_id, log_date, status, value_numeric, note)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(tracker_item_id, log_date) DO UPDATE SET
			status = excluded.status,
			value_numeric = excluded.value_numeric,
			note = excluded.note,
			updated_at = CURRENT_TIMESTAMP,
			sync_version = tracker_logs.sync_version + 1`,
		itemID,
		logDate,
		status,
		value,
		note,
	)
	return err
}

func (r *Repository) ToggleTrackerLogByKey(key, logDate string) (string, error) {
	items, err := r.ListTrackerItems()
	if err != nil {
		return "", err
	}

	var item models.TrackerItem
	found := false
	for _, candidate := range items {
		if candidate.Key == key {
			item = candidate
			found = true
			break
		}
	}
	if !found {
		return "", sql.ErrNoRows
	}

	logs, err := r.TrackerLogsBetween(logDate, logDate)
	if err != nil {
		return "", err
	}
	current := defaultTrackerStatus(item)
	for _, log := range logs {
		if log.TrackerItemID == item.ID {
			current = log.Status
			break
		}
	}

	next := "done"
	if item.Type == "gym" {
		switch current {
		case "done":
			next = "not_done"
		case "not_done":
			next = "rest"
		default:
			next = "done"
		}
	} else if current == "done" {
		next = "not_done"
	}

	if err := r.SetTrackerLogByKey(key, logDate, next, "", nil); err != nil {
		return "", err
	}
	return next, nil
}

func defaultTrackerStatus(item models.TrackerItem) string {
	if item.Type == "gym" {
		return "rest"
	}
	return "not_done"
}
