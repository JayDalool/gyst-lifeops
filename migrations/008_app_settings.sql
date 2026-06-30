PRAGMA foreign_keys = ON;

-- Small key/value store for app-level config. First use: the tokenized calendar
-- feed secret, so an external calendar (Apple/Google) can subscribe to a private
-- feed URL without exposing the plain /calendar.ics publicly.
CREATE TABLE IF NOT EXISTS app_settings (
    setting_key TEXT PRIMARY KEY,
    setting_value TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
