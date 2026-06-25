PRAGMA foreign_keys = OFF;
PRAGMA legacy_alter_table = ON;

ALTER TABLE projects RENAME TO projects_old;

CREATE TABLE projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    parent_project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'other' CHECK (category IN ('coding', 'sim rig', 'car', 'health', 'business', 'personal', 'other')),
    project_kind TEXT NOT NULL DEFAULT 'main' CHECK (project_kind IN ('main', 'subproject', 'phase')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'upcoming', 'locked', 'done', 'archived')),
    start_date TEXT,
    target_date TEXT,
    notes TEXT NOT NULL DEFAULT '',
    unlock_notes TEXT NOT NULL DEFAULT '',
    depends_on_project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL,
    auto_unlock INTEGER NOT NULL DEFAULT 1 CHECK (auto_unlock IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at TEXT,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

INSERT INTO projects (
    id, uuid, name, description, category, status, start_date, target_date, notes,
    created_at, updated_at, archived_at, sync_version, last_modified_device_id
)
SELECT
    id, uuid, name, description, category,
    CASE status WHEN 'completed' THEN 'done' WHEN 'paused' THEN 'upcoming' ELSE status END,
    start_date, target_date, notes, created_at, updated_at, archived_at, sync_version, last_modified_device_id
FROM projects_old;

DROP TABLE projects_old;

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_parent ON projects(parent_project_id);
CREATE INDEX IF NOT EXISTS idx_projects_dependency ON projects(depends_on_project_id);

ALTER TABLE tasks RENAME TO tasks_old;

CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'personal',
    priority TEXT NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'skipped')),
    due_date TEXT,
    repeat_type TEXT NOT NULL DEFAULT 'none' CHECK (repeat_type IN ('none', 'daily', 'weekly', 'monthly', 'yearly')),
    reminder_time TEXT,
    stars_reward INTEGER NOT NULL DEFAULT 1,
    proof_required INTEGER NOT NULL DEFAULT 0 CHECK (proof_required IN (0, 1)),
    proof_type TEXT NOT NULL DEFAULT 'none' CHECK (proof_type IN ('none', 'photo', 'file', 'link', 'text')),
    proof_url TEXT,
    proof_path TEXT,
    stars_awarded INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT,
    archived_at TEXT,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

INSERT INTO tasks (
    id, uuid, title, description, category, priority, status, due_date, repeat_type,
    reminder_time, created_at, updated_at, completed_at, archived_at, sync_version, last_modified_device_id
)
SELECT
    id, uuid, title, description, category, priority, status, due_date, repeat_type,
    reminder_time, created_at, updated_at, completed_at, archived_at, sync_version, last_modified_device_id
FROM tasks_old;

DROP TABLE tasks_old;

CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_repeat_type ON tasks(repeat_type);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

CREATE TABLE IF NOT EXISTS achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    stars_bonus INTEGER NOT NULL DEFAULT 0,
    unlocked_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    trigger_type TEXT NOT NULL DEFAULT 'manual' CHECK (trigger_type IN ('manual', 'project_done', 'task_count', 'proof_submitted')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_achievements_unlocked ON achievements(unlocked_at);
CREATE INDEX IF NOT EXISTS idx_achievements_trigger ON achievements(trigger_type);

PRAGMA foreign_keys = ON;
PRAGMA legacy_alter_table = OFF;
