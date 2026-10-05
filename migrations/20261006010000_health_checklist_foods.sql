-- Health daily checklist and My foods & products (Issue #59). Additive only.
-- Rollback: these three tables are new and nothing else references them, so
-- an older binary simply ignores them. To remove: DROP TABLE
-- health_checklist_log; DROP TABLE health_checklist_items; DROP TABLE
-- health_foods; DELETE FROM schema_migrations WHERE filename = this file.

-- A checklist item is configured by the user. "habit" is ticked by hand per
-- day; "protein" is a measured target filled from the day's snapshot and the
-- plan, and is never ticked. No doses: GYST gives no supplement advice.
CREATE TABLE health_checklist_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 80),
    kind TEXT NOT NULL CHECK (kind IN ('habit', 'protein')),
    position INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One row per habit ticked on a local calendar day (application timezone).
CREATE TABLE health_checklist_log (
    item_id INTEGER NOT NULL REFERENCES health_checklist_items(id) ON DELETE CASCADE,
    date TEXT NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    done_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (item_id, date)
);

-- Foods Jay usually has. Nutrition values are per serving and nullable:
-- unknown is not zero.
CREATE TABLE health_foods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 120),
    serving TEXT NOT NULL DEFAULT '' CHECK (length(serving) <= 80),
    kcal REAL CHECK (kcal IS NULL OR (kcal >= 0 AND kcal <= 10000)),
    protein_g REAL CHECK (protein_g IS NULL OR (protein_g >= 0 AND protein_g <= 1000)),
    carbs_g REAL CHECK (carbs_g IS NULL OR (carbs_g >= 0 AND carbs_g <= 1000)),
    fat_g REAL CHECK (fat_g IS NULL OR (fat_g >= 0 AND fat_g <= 1000)),
    fiber_g REAL CHECK (fiber_g IS NULL OR (fiber_g >= 0 AND fiber_g <= 1000)),
    notes TEXT NOT NULL DEFAULT '' CHECK (length(notes) <= 500),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_health_foods_active_name ON health_foods(active, name);
