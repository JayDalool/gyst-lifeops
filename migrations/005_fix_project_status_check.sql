-- Rebuild the projects table so the status CHECK accepts the quest-system
-- states introduced in 004 (upcoming, locked, done). SQLite cannot ALTER a
-- CHECK constraint in place, so the table is recreated and data copied over.
--
-- foreign_keys is turned OFF for the rebuild so that DROP TABLE projects does
-- not cascade-delete project_items; the project_items FK resolves back to the
-- new table by name after the rename. The self-referential parent/dependency
-- FKs likewise re-resolve to the renamed table.

PRAGMA foreign_keys = OFF;

CREATE TABLE projects_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'other' CHECK (category IN ('coding', 'sim rig', 'car', 'health', 'business', 'personal', 'other')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'upcoming', 'locked', 'completed', 'done', 'archived')),
    start_date TEXT,
    target_date TEXT,
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at TEXT,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT,
    parent_project_id INTEGER REFERENCES projects (id) ON DELETE SET NULL,
    project_kind TEXT NOT NULL DEFAULT 'main' CHECK (project_kind IN ('main', 'subproject', 'phase')),
    unlock_notes TEXT NOT NULL DEFAULT '',
    depends_on_project_id INTEGER REFERENCES projects (id) ON DELETE SET NULL,
    auto_unlock INTEGER NOT NULL DEFAULT 1 CHECK (auto_unlock IN (0, 1))
);

INSERT INTO projects_new (
    id, uuid, name, description, category, status, start_date, target_date,
    notes, created_at, updated_at, archived_at, sync_version, last_modified_device_id,
    parent_project_id, project_kind, unlock_notes, depends_on_project_id, auto_unlock
)
SELECT
    id, uuid, name, description, category,
    CASE status
        WHEN 'completed' THEN 'done'
        WHEN 'paused' THEN 'upcoming'
        ELSE status
    END,
    start_date, target_date,
    notes, created_at, updated_at, archived_at, sync_version, last_modified_device_id,
    parent_project_id, project_kind, unlock_notes, depends_on_project_id, auto_unlock
FROM projects;

DROP TABLE projects;

ALTER TABLE projects_new RENAME TO projects;

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects (status);

CREATE INDEX IF NOT EXISTS idx_projects_parent ON projects (parent_project_id);

CREATE INDEX IF NOT EXISTS idx_projects_dependency ON projects (depends_on_project_id);

PRAGMA foreign_keys = ON;
