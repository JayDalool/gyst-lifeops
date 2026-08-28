PRAGMA foreign_keys = ON;

ALTER TABLE projects
ADD COLUMN parent_project_id INTEGER REFERENCES projects (id) ON DELETE SET NULL;

ALTER TABLE projects
ADD COLUMN project_kind TEXT NOT NULL DEFAULT 'main' CHECK (
    project_kind IN ('main', 'subproject', 'phase')
);

ALTER TABLE projects
ADD COLUMN unlock_notes TEXT NOT NULL DEFAULT '';

ALTER TABLE projects
ADD COLUMN depends_on_project_id INTEGER REFERENCES projects (id) ON DELETE SET NULL;

ALTER TABLE projects
ADD COLUMN auto_unlock INTEGER NOT NULL DEFAULT 1 CHECK (auto_unlock IN (0, 1));

-- Status values are normalized by migration 005 while it rebuilds the table.
-- Updating them here would violate the original 001 CHECK constraint on a
-- populated pre-quest database.

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects (status);

CREATE INDEX IF NOT EXISTS idx_projects_parent ON projects (parent_project_id);

CREATE INDEX IF NOT EXISTS idx_projects_dependency ON projects (depends_on_project_id);

ALTER TABLE tasks ADD COLUMN stars_reward INTEGER NOT NULL DEFAULT 1;

ALTER TABLE tasks
ADD COLUMN proof_required INTEGER NOT NULL DEFAULT 0 CHECK (proof_required IN (0, 1));

ALTER TABLE tasks
ADD COLUMN proof_type TEXT NOT NULL DEFAULT 'none' CHECK (
    proof_type IN (
        'none',
        'photo',
        'file',
        'link',
        'text'
    )
);

ALTER TABLE tasks ADD COLUMN proof_url TEXT;

ALTER TABLE tasks ADD COLUMN proof_path TEXT;

ALTER TABLE tasks
ADD COLUMN stars_awarded INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks (due_date);

CREATE INDEX IF NOT EXISTS idx_tasks_repeat_type ON tasks (repeat_type);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);

CREATE TABLE IF NOT EXISTS achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT(lower(hex(randomblob (16)))),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    stars_bonus INTEGER NOT NULL DEFAULT 0,
    unlocked_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    trigger_type TEXT NOT NULL DEFAULT 'manual' CHECK (
        trigger_type IN (
            'manual',
            'project_done',
            'task_count',
            'proof_submitted'
        )
    ),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_achievements_unlocked ON achievements (unlocked_at);

CREATE INDEX IF NOT EXISTS idx_achievements_trigger ON achievements (trigger_type);
