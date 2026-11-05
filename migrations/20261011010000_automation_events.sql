-- n8n automation events (GYST-VN-110, Issue #137). Additive only.
-- Rollback: deploy the older binary and leave everything in place (the
-- table, the index and this file's schema_migrations row). An older binary
-- ignores them. Don't delete the schema_migrations row on its own:
-- re-applying this file would then fail on the existing table.

-- One row per event n8n reports for a registered workflow. event_id is the
-- primary key, so a repeated delivery is a no-op (deduplication). Instants are
-- UTC text in the form 2026-01-02T03:04:05Z. error_summary is a short,
-- sanitised note, never a payload. Rows older than 30 days are pruned by the
-- service when a new event is stored.
CREATE TABLE automation_events (
    event_id TEXT PRIMARY KEY CHECK (length(event_id) BETWEEN 1 AND 128),
    workflow_id TEXT NOT NULL CHECK (length(workflow_id) BETWEEN 1 AND 64),
    status TEXT NOT NULL CHECK (status IN ('started', 'succeeded', 'failed')),
    occurred_at TEXT NOT NULL CHECK (occurred_at LIKE '____-__-__T__:__:__Z'),
    received_at TEXT NOT NULL CHECK (received_at LIKE '____-__-__T__:__:__Z'),
    error_summary TEXT NOT NULL DEFAULT '' CHECK (length(error_summary) <= 200)
);

CREATE INDEX idx_automation_events_workflow_time
    ON automation_events (workflow_id, occurred_at);
