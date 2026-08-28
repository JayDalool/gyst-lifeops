-- Project-scoped missions need the same structured reward/proof behavior as
-- standalone tasks. Additive columns keep every existing mission valid.
ALTER TABLE project_items ADD COLUMN stars_reward INTEGER NOT NULL DEFAULT 1;

ALTER TABLE project_items
ADD COLUMN proof_required INTEGER NOT NULL DEFAULT 0 CHECK (proof_required IN (0, 1));

ALTER TABLE project_items
ADD COLUMN proof_type TEXT NOT NULL DEFAULT 'none' CHECK (
    proof_type IN ('none', 'photo', 'file', 'link', 'text')
);

ALTER TABLE project_items ADD COLUMN proof_url TEXT;

ALTER TABLE project_items ADD COLUMN proof_path TEXT;

ALTER TABLE project_items ADD COLUMN stars_awarded INTEGER NOT NULL DEFAULT 0;

ALTER TABLE achievements ADD COLUMN source_project_item_id INTEGER REFERENCES project_items (id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_achievements_source_project_item
ON achievements (trigger_type, source_project_item_id)
WHERE source_project_item_id IS NOT NULL;
