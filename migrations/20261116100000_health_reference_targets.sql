-- Health reference targets (Issue #205, GYST-156). Additive only.
-- Outside recommendations (for example a dietitian's numbers) kept beside the
-- active plan. They never change health_cycles and are never used as targets.
-- NULL means "not given", never 0. Rows are archived, not deleted.
-- created_at is written by the application (clock.FormatInstant).
-- Rollback: DROP TABLE health_reference_targets; then remove this file's row
-- from schema_migrations. Nothing else references the table. An older binary
-- ignores the table, so deploying it with the table left in place is also safe.
CREATE TABLE health_reference_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cycle_id INTEGER REFERENCES health_cycles(id),
    source_label TEXT NOT NULL CHECK (length(source_label) BETWEEN 1 AND 200),
    source_kind TEXT NOT NULL CHECK (source_kind IN ('dietitian', 'device', 'ai', 'other')),
    issued_on TEXT CHECK (issued_on IS NULL OR issued_on GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    kcal REAL CHECK (kcal IS NULL OR (kcal >= 0 AND kcal <= 100000)),
    protein_g REAL CHECK (protein_g IS NULL OR (protein_g >= 0 AND protein_g <= 10000)),
    carbs_g REAL CHECK (carbs_g IS NULL OR (carbs_g >= 0 AND carbs_g <= 10000)),
    fat_g REAL CHECK (fat_g IS NULL OR (fat_g >= 0 AND fat_g <= 10000)),
    fiber_g REAL CHECK (fiber_g IS NULL OR (fiber_g >= 0 AND fiber_g <= 10000)),
    notes TEXT NOT NULL DEFAULT '' CHECK (length(notes) <= 2000),
    created_at TEXT NOT NULL,
    archived_at TEXT
);
CREATE INDEX idx_health_reference_targets_cycle ON health_reference_targets(cycle_id);
