package repository

import (
	"database/sql"
	"encoding/json"
	"lifeops-tracker/internal/models"
	"time"
)

func (r *Repository) TrainingCache(provider string) (models.TrainingCache, error) {
	var c models.TrainingCache
	var body, stamp string
	err := r.db.QueryRow(`SELECT entries_json, synced_at, last_error FROM training_snapshots WHERE provider=?`, provider).Scan(&body, &stamp, &c.Error)
	if err == sql.ErrNoRows {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	c.SyncedAt, _ = time.Parse(time.RFC3339Nano, stamp)
	err = json.Unmarshal([]byte(body), &c.Entries)
	return c, err
}
func (r *Repository) SaveTraining(provider string, entries []models.TrainingEntry, now time.Time) error {
	body, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`INSERT INTO training_snapshots(provider,entries_json,synced_at) VALUES(?,?,?) ON CONFLICT(provider) DO UPDATE SET entries_json=excluded.entries_json,synced_at=excluded.synced_at,last_error=''`, provider, string(body), now.UTC().Format(time.RFC3339Nano))
	return err
}
func (r *Repository) TrainingFailure(provider, message string) error {
	_, err := r.db.Exec(`INSERT INTO training_snapshots(provider,entries_json,synced_at,last_error) VALUES(?,'[]','',?) ON CONFLICT(provider) DO UPDATE SET last_error=excluded.last_error`, provider, message)
	return err
}
