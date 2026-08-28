-- Tie generated achievements to their source so retrying a completion cannot
-- create duplicate rewards or duplicate activity entries.
ALTER TABLE achievements ADD COLUMN source_task_id INTEGER REFERENCES tasks (id) ON DELETE SET NULL;

ALTER TABLE achievements ADD COLUMN source_project_id INTEGER REFERENCES projects (id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_achievements_source_task
ON achievements (trigger_type, source_task_id)
WHERE source_task_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_achievements_source_project
ON achievements (trigger_type, source_project_id)
WHERE source_project_id IS NOT NULL;
