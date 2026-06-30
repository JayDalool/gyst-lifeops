package repository

import (
	"database/sql"
	"errors"
)

// GetSetting returns the value for key, or "" when the key is unset.
func (r *Repository) GetSetting(key string) (string, error) {
	var value string
	err := r.db.QueryRow(`SELECT setting_value FROM app_settings WHERE setting_key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SetSetting upserts a key/value.
func (r *Repository) SetSetting(key, value string) error {
	_, err := r.db.Exec(`INSERT INTO app_settings (setting_key, setting_value)
		VALUES (?, ?)
		ON CONFLICT(setting_key) DO UPDATE SET
			setting_value = excluded.setting_value,
			updated_at = CURRENT_TIMESTAMP`, key, value)
	return err
}

// DeleteSetting removes a key.
func (r *Repository) DeleteSetting(key string) error {
	_, err := r.db.Exec(`DELETE FROM app_settings WHERE setting_key = ?`, key)
	return err
}
