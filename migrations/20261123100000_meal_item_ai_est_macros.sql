-- AI food camera: adjust before logging (Issue #269, GYST-179). Additive only.
-- Keeps the AI's own nutrient figures beside the saved ones so accuracy can be
-- measured. NULL for every item that had no AI estimate of that nutrient (all
-- existing rows, catalogue-matched items, hand-added items). Never holds photo data.
-- Ranges are copied from the item columns in 20261116120000_nutrition_diary.sql.
-- Rollback: deploy the older binary and leave these columns and this file's
-- schema_migrations row in place; an older binary ignores them. Don't delete the
-- schema_migrations row on its own: re-applying would fail on the existing columns.
ALTER TABLE nutrition_meal_items ADD COLUMN ai_est_kcal REAL
    CHECK (ai_est_kcal IS NULL OR (ai_est_kcal >= 0 AND ai_est_kcal <= 100000));
ALTER TABLE nutrition_meal_items ADD COLUMN ai_est_protein_g REAL
    CHECK (ai_est_protein_g IS NULL OR (ai_est_protein_g >= 0 AND ai_est_protein_g <= 10000));
ALTER TABLE nutrition_meal_items ADD COLUMN ai_est_carbs_g REAL
    CHECK (ai_est_carbs_g IS NULL OR (ai_est_carbs_g >= 0 AND ai_est_carbs_g <= 10000));
ALTER TABLE nutrition_meal_items ADD COLUMN ai_est_fat_g REAL
    CHECK (ai_est_fat_g IS NULL OR (ai_est_fat_g >= 0 AND ai_est_fat_g <= 10000));
