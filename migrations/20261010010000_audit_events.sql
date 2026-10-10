-- Minimal audit event store (GYST-VN-005, Issue #111). Additive only.
-- Rollback: deploy the older binary and leave the table, its index and this
-- file's schema_migrations row in place. An older binary ignores the table.
-- Don't delete the schema_migrations row on its own: re-applying this file
-- would then fail on the existing table.
--
-- `at` has no SQL default: the app writes clock.FormatInstant(Clock.Now()).
-- The action list is enforced in Go, not here, so later tickets can add
-- actions without rebuilding the table. The enums below are complete in
-- docs/audit/vnext/09_SECURITY_FOUNDATION.md section M.
CREATE TABLE audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    at TEXT NOT NULL
        CHECK (length(at) = 30
            AND at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z'),
    actor_type TEXT NOT NULL CHECK (actor_type IN ('human', 'machine', 'system')),
    actor_ref TEXT NOT NULL DEFAULT '' CHECK (length(actor_ref) <= 120),
    ingress TEXT NOT NULL CHECK (ingress IN ('public', 'tailnet', 'internal', 'job')),
    action TEXT NOT NULL CHECK (length(action) BETWEEN 1 AND 64),
    target_ref TEXT NOT NULL DEFAULT '' CHECK (length(target_ref) <= 200),
    outcome TEXT NOT NULL CHECK (outcome IN ('success', 'denied', 'failed')),
    detail TEXT NOT NULL DEFAULT '' CHECK (length(detail) <= 200),
    request_id TEXT NOT NULL DEFAULT '' CHECK (length(request_id) <= 32)
);

CREATE INDEX idx_audit_events_at ON audit_events(at);
