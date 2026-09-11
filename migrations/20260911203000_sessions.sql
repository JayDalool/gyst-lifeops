-- Login sessions for the single-user web UI. Additive; no existing table is
-- touched and nothing is seeded.
--
-- The primary key is a SHA-256 hash of the session token, never the token
-- itself. The raw token exists only in the user's cookie, so a reader of this
-- database - a backup, a stolen disk, a support copy - cannot mint a cookie
-- from what is stored here.
--
-- This is the first migration to use a timestamp prefix. Sequence numbers are
-- allocated from a counter neither of two parallel branches can see, which is
-- how the historical 017 collision happened; timestamps remove the shared
-- counter. The name still sorts after 018_ because '2' > '0'.
CREATE TABLE sessions (
    token_hash   TEXT PRIMARY KEY NOT NULL,
    created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at   TEXT NOT NULL,
    last_seen_at TEXT NOT NULL
);

-- Supports expiry sweeps.
CREATE INDEX idx_sessions_expires ON sessions (expires_at);
