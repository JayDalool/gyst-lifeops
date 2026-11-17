-- AI provider settings (GYST-169, Issue #218). Additive only.
-- One row (id = 1): the provider, the chat and photo model names, and the API
-- key as AES-256-GCM ciphertext plus its random nonce. The 32-byte data key is
-- NOT in this database: it lives in a 0600 file beside it, so a VACUUM INTO
-- backup carries ciphertext only.
-- Rollback: deploy the older binary and leave the table and this file's
-- schema_migrations row in place. An older binary ignores the table. Don't
-- delete the schema_migrations row on its own: re-applying this file would
-- then fail on the existing table.
--
-- Instants are written by the app through clock.FormatInstant: no SQL default.
CREATE TABLE ai_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    provider TEXT NOT NULL CHECK (provider IN ('openai', 'anthropic')),
    chat_model TEXT NOT NULL DEFAULT '' CHECK (length(chat_model) <= 64),
    photo_model TEXT NOT NULL DEFAULT '' CHECK (length(photo_model) <= 64),
    key_ciphertext BLOB,
    key_nonce BLOB,
    key_saved_at TEXT
        CHECK (key_saved_at IS NULL OR (length(key_saved_at) = 30
            AND key_saved_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z')),
    updated_at TEXT NOT NULL
        CHECK (length(updated_at) = 30
            AND updated_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z'),
    CHECK ((key_ciphertext IS NULL AND key_nonce IS NULL AND key_saved_at IS NULL)
        OR (key_ciphertext IS NOT NULL AND key_nonce IS NOT NULL AND key_saved_at IS NOT NULL
            AND length(key_nonce) = 12))
);
