-- Multiple roadmaps: top-level journeys that group projects (trips).
-- This is an ADDITIVE migration (no table rebuild). It is unrelated to the
-- existing markdown-roadmap feature (roadmap_project_meta / RoadmapData),
-- which keeps working untouched.

CREATE TABLE IF NOT EXISTS roadmaps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    priority INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'upcoming', 'locked', 'done', 'archived')),
    privacy TEXT NOT NULL DEFAULT 'private' CHECK (privacy IN ('public', 'private', 'local_only')),
    featured INTEGER NOT NULL DEFAULT 0 CHECK (featured IN (0, 1)),
    icon TEXT NOT NULL DEFAULT 'rocket',
    color TEXT NOT NULL DEFAULT 'cyan',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at TEXT,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_roadmaps_featured ON roadmaps (featured, priority);

CREATE INDEX IF NOT EXISTS idx_roadmaps_status ON roadmaps (status);

-- Link projects to roadmaps. Nullable + NULL default, so this ADD COLUMN with a
-- REFERENCES clause is permitted by SQLite and breaks no existing rows.
ALTER TABLE projects ADD COLUMN roadmap_id INTEGER REFERENCES roadmaps (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_projects_roadmap ON projects (roadmap_id);

-- Default roadmap, featured, so existing installs have a dashboard journey.
INSERT INTO roadmaps (name, description, priority, featured, privacy, status)
VALUES ('LifeOps Roadmap', 'Default roadmap created during migration.', 1, 1, 'private', 'active');

-- Adopt every pre-existing project into the default roadmap.
UPDATE projects
SET roadmap_id = (SELECT id FROM roadmaps WHERE name = 'LifeOps Roadmap' ORDER BY id LIMIT 1)
WHERE roadmap_id IS NULL;
