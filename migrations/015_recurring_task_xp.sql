-- Recurring tasks need an idempotent award per dated occurrence. Keeping the
-- award on the unique (task_id, occurrence_date) row prevents retries and
-- complete/skip toggles from duplicating XP.
ALTER TABLE task_completions ADD COLUMN stars_awarded INTEGER NOT NULL DEFAULT 0;

