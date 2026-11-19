-- AI food camera (Issue #211, GYST-162). Additive only.
-- Stores the grams the AI estimated beside the grams Jay confirmed
-- (amount/grams), so accuracy can be measured from real use. NULL for every
-- item that did not come from a photo estimate. Never holds photo data.
-- Rollback: deploy the older binary and leave this column and this file's
-- schema_migrations row in place; an older binary ignores the column. Don't
-- delete the schema_migrations row on its own: re-applying would fail on the
-- existing column.
ALTER TABLE nutrition_meal_items ADD COLUMN ai_est_grams REAL
    CHECK (ai_est_grams IS NULL OR (ai_est_grams >= 0 AND ai_est_grams <= 100000));
