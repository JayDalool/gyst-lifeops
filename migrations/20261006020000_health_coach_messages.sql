-- Health Coach conversation memory (Issue #64). Additive only.
-- GYST owns the memory; no vendor-side chat storage is used. Only visible
-- turns are stored (never hidden reasoning), and the page can delete them.
-- Rollback: DROP TABLE health_coach_messages; then remove this file's row
-- from schema_migrations. Nothing else references the table.
CREATE TABLE health_coach_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    text TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 8000),
    action TEXT NOT NULL DEFAULT '' CHECK (length(action) <= 40),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
