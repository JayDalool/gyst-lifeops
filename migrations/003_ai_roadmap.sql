PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS roadmap_imports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    goal_id INTEGER REFERENCES goals(id) ON DELETE SET NULL,
    goal_title TEXT NOT NULL DEFAULT '',
    goal_deadline TEXT,
    source_markdown TEXT NOT NULL,
    imported_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_roadmap_imports_imported ON roadmap_imports(imported_at);

CREATE TABLE IF NOT EXISTS roadmap_project_meta (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    project_id INTEGER NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    source_import_id INTEGER REFERENCES roadmap_imports(id) ON DELETE SET NULL,
    project_type TEXT NOT NULL DEFAULT 'learning' CHECK (project_type IN (
        'real_estate_ai',
        'healthcare_ai',
        'ml_model',
        'certification',
        'portfolio',
        'learning'
    )),
    tags TEXT NOT NULL DEFAULT '',
    is_private INTEGER NOT NULL DEFAULT 0 CHECK (is_private IN (0, 1)),
    local_only INTEGER NOT NULL DEFAULT 0 CHECK (local_only IN (0, 1)),
    current_focus INTEGER NOT NULL DEFAULT 0 CHECK (current_focus IN (0, 1)),
    public_summary TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_roadmap_project_meta_type ON roadmap_project_meta(project_type);
CREATE INDEX IF NOT EXISTS idx_roadmap_project_meta_flags ON roadmap_project_meta(is_private, local_only, current_focus);

CREATE TABLE IF NOT EXISTS roadmap_milestones (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
    due_date TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_roadmap_milestones_project ON roadmap_milestones(project_id);
CREATE INDEX IF NOT EXISTS idx_roadmap_milestones_due ON roadmap_milestones(due_date);

CREATE TABLE IF NOT EXISTS roadmap_resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    milestone_id INTEGER REFERENCES roadmap_milestones(id) ON DELETE SET NULL,
    project_item_id INTEGER REFERENCES project_items(id) ON DELETE SET NULL,
    source_import_id INTEGER REFERENCES roadmap_imports(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT 'other' CHECK (resource_type IN (
        'youtube',
        'course',
        'github',
        'documentation',
        'notes',
        'other'
    )),
    notes TEXT NOT NULL DEFAULT '',
    is_private INTEGER NOT NULL DEFAULT 0 CHECK (is_private IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_roadmap_resources_project ON roadmap_resources(project_id);
CREATE INDEX IF NOT EXISTS idx_roadmap_resources_type ON roadmap_resources(resource_type);

CREATE TABLE IF NOT EXISTS roadmap_weekly_reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    week_start TEXT NOT NULL,
    finished TEXT NOT NULL DEFAULT '',
    blocked TEXT NOT NULL DEFAULT '',
    next_steps TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_roadmap_weekly_reviews_week ON roadmap_weekly_reviews(week_start);
