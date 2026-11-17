package repository

import (
	"database/sql"
	"errors"
)

// AISettingsRow is the one ai_settings row (GYST-169). KeyCiphertext and
// KeyNonce are AES-256-GCM output; the data key is never in the database.
type AISettingsRow struct {
	Provider      string
	ChatModel     string
	PhotoModel    string
	KeyCiphertext []byte // nil when no key is saved
	KeyNonce      []byte
	KeySavedAt    string // stored instant, "" when no key is saved
	UpdatedAt     string
}

// GetAISettings returns the row, or nil when nothing was ever saved.
func (r *Repository) GetAISettings() (*AISettingsRow, error) {
	var row AISettingsRow
	var saved sql.NullString
	err := r.db.QueryRow(`SELECT provider, chat_model, photo_model, key_ciphertext, key_nonce, key_saved_at, updated_at
		FROM ai_settings WHERE id = 1`).
		Scan(&row.Provider, &row.ChatModel, &row.PhotoModel, &row.KeyCiphertext, &row.KeyNonce, &saved, &row.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.KeySavedAt = saved.String
	return &row, nil
}

// SaveAISettings upserts the row. A nil KeyCiphertext stores no key.
func (r *Repository) SaveAISettings(row AISettingsRow) error {
	var saved any
	if row.KeySavedAt != "" {
		saved = row.KeySavedAt
	}
	_, err := r.db.Exec(`INSERT INTO ai_settings
			(id, provider, chat_model, photo_model, key_ciphertext, key_nonce, key_saved_at, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider = excluded.provider,
			chat_model = excluded.chat_model,
			photo_model = excluded.photo_model,
			key_ciphertext = excluded.key_ciphertext,
			key_nonce = excluded.key_nonce,
			key_saved_at = excluded.key_saved_at,
			updated_at = excluded.updated_at`,
		row.Provider, row.ChatModel, row.PhotoModel, row.KeyCiphertext, row.KeyNonce, saved, row.UpdatedAt)
	return err
}

// ClearAIKey removes the saved key and keeps the provider and models. It
// reports whether a key was there.
func (r *Repository) ClearAIKey(updatedAt string) (bool, error) {
	res, err := r.db.Exec(`UPDATE ai_settings SET key_ciphertext = NULL, key_nonce = NULL,
		key_saved_at = NULL, updated_at = ? WHERE id = 1 AND key_ciphertext IS NOT NULL`, updatedAt)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
