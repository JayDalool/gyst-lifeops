CREATE TABLE training_snapshots (
 provider TEXT PRIMARY KEY,
 entries_json TEXT NOT NULL,
 synced_at TEXT NOT NULL,
 last_error TEXT NOT NULL DEFAULT ''
);
