-- Health water target and bottle log (Issue #60). Additive only.
-- Rollback: an older binary ignores the new table and columns. To remove:
-- DROP TABLE health_water_log; the two health_cycles columns can stay (they
-- default to 0 = not set); DELETE FROM schema_migrations WHERE filename =
-- this file.

-- Water target and bottle size per plan, in millilitres. 0 means not set,
-- so an existing plan shows "Set a water target in Plan" rather than an
-- invented goal.
ALTER TABLE health_cycles ADD COLUMN water_target_ml INTEGER NOT NULL DEFAULT 0
    CHECK (water_target_ml BETWEEN 0 AND 10000);
ALTER TABLE health_cycles ADD COLUMN water_bottle_ml INTEGER NOT NULL DEFAULT 0
    CHECK (water_bottle_ml BETWEEN 0 AND 3000);

-- One row per "Bottle done" tap on a local calendar day (application
-- timezone). The amount is copied from the plan at the time of the tap, so a
-- later change of bottle size doesn't rewrite history. No rows = nothing
-- logged, never "0 L drunk".
CREATE TABLE health_water_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    ml INTEGER NOT NULL CHECK (ml BETWEEN 1 AND 3000),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_health_water_log_date ON health_water_log(date);
