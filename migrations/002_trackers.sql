PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS tracker_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('prayer', 'athkar', 'gym', 'project_signal', 'custom')),
    category TEXT NOT NULL CHECK (category IN ('deen', 'body', 'build', 'work', 'personal')),
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_tracker_items_type ON tracker_items(type);
CREATE INDEX IF NOT EXISTS idx_tracker_items_active ON tracker_items(is_active, sort_order);

CREATE TABLE IF NOT EXISTS tracker_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    tracker_item_id INTEGER NOT NULL REFERENCES tracker_items(id) ON DELETE CASCADE,
    log_date TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('done', 'not_done', 'skipped', 'rest')),
    value_numeric REAL,
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT,
    UNIQUE(tracker_item_id, log_date)
);

CREATE INDEX IF NOT EXISTS idx_tracker_logs_date ON tracker_logs(log_date);
CREATE INDEX IF NOT EXISTS idx_tracker_logs_item_date ON tracker_logs(tracker_item_id, log_date);

INSERT OR IGNORE INTO tracker_items (key, title, type, category, sort_order) VALUES
    ('prayer_fajr', 'Fajr', 'prayer', 'deen', 10),
    ('prayer_dhuhr', 'Dhuhr', 'prayer', 'deen', 20),
    ('prayer_asr', 'Asr', 'prayer', 'deen', 30),
    ('prayer_maghrib', 'Maghrib', 'prayer', 'deen', 40),
    ('prayer_isha', 'Isha', 'prayer', 'deen', 50),
    ('athkar_sabah', 'Sabah Athkar', 'athkar', 'deen', 60),
    ('athkar_masaa', 'Masaa Athkar', 'athkar', 'deen', 70),
    ('gym', 'Gym', 'gym', 'body', 80);
