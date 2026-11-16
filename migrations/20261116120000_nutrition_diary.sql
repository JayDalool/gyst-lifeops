-- Food diary core (Issue #207, GYST-158). Additive only.
-- Rollback: deploy the older binary and leave these tables, their indexes,
-- triggers and this file's schema_migrations row in place. An older binary
-- ignores them and counts Apple Health intake for every day, as before.
-- Don't delete the schema_migrations row on its own: re-applying this file
-- would then fail on the existing tables.
--
-- Timestamps have no SQL default: the app writes clock.FormatInstant. The
-- CHECKs below only enforce that stored instants have that exact shape.
-- Provider names are generic (docs/RULES.md 3.6). Nutrient columns are
-- nullable: unknown is not zero.

-- Catalogue foods are immutable and versioned. A change inserts a new row
-- with the next version and sets superseded_at on the old one, so a logged
-- item can always point at the exact row it used. Nutrient values are per
-- 100 g when basis is per_100g and per serving when basis is per_serving
-- (serving_grams is optional: My foods carry no gram weight).
CREATE TABLE nutrition_foods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL CHECK (provider IN ('user', 'usda_fdc', 'cnf', 'off', 'recipe')),
    provider_id TEXT NOT NULL CHECK (length(provider_id) BETWEEN 1 AND 120),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 120),
    brand TEXT NOT NULL DEFAULT '' CHECK (length(brand) <= 120),
    normalized_name TEXT NOT NULL CHECK (length(normalized_name) BETWEEN 1 AND 120),
    basis TEXT NOT NULL CHECK (basis IN ('per_100g', 'per_serving')),
    serving_label TEXT NOT NULL DEFAULT '' CHECK (length(serving_label) <= 80),
    serving_grams REAL CHECK (serving_grams IS NULL OR (serving_grams > 0 AND serving_grams <= 10000)),
    kcal REAL CHECK (kcal IS NULL OR (kcal >= 0 AND kcal <= 10000)),
    protein_g REAL CHECK (protein_g IS NULL OR (protein_g >= 0 AND protein_g <= 1000)),
    carbs_g REAL CHECK (carbs_g IS NULL OR (carbs_g >= 0 AND carbs_g <= 1000)),
    fat_g REAL CHECK (fat_g IS NULL OR (fat_g >= 0 AND fat_g <= 1000)),
    fiber_g REAL CHECK (fiber_g IS NULL OR (fiber_g >= 0 AND fiber_g <= 1000)),
    data_provenance TEXT NOT NULL CHECK (data_provenance IN ('label', 'database', 'user', 'ai_estimate')),
    dataset_version TEXT NOT NULL DEFAULT '' CHECK (length(dataset_version) <= 40),
    created_at TEXT NOT NULL CHECK (length(created_at) = 30 AND created_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z'),
    superseded_at TEXT CHECK (superseded_at IS NULL OR (length(superseded_at) = 30 AND superseded_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z')),
    archived_at TEXT CHECK (archived_at IS NULL OR (length(archived_at) = 30 AND archived_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z')),
    UNIQUE (provider, provider_id, version),
    UNIQUE (id, version)
);

-- At most one current (not superseded) row per food.
CREATE UNIQUE INDEX idx_nutrition_foods_current ON nutrition_foods(provider, provider_id) WHERE superseded_at IS NULL;
CREATE INDEX idx_nutrition_foods_search ON nutrition_foods(normalized_name);

-- Only archived_at may change after insert, and superseded_at may be set once.
CREATE TRIGGER nutrition_foods_immutable BEFORE UPDATE ON nutrition_foods
FOR EACH ROW WHEN
    NEW.id IS NOT OLD.id OR NEW.provider IS NOT OLD.provider OR NEW.provider_id IS NOT OLD.provider_id
    OR NEW.version IS NOT OLD.version OR NEW.name IS NOT OLD.name OR NEW.brand IS NOT OLD.brand
    OR NEW.normalized_name IS NOT OLD.normalized_name OR NEW.basis IS NOT OLD.basis
    OR NEW.serving_label IS NOT OLD.serving_label OR NEW.serving_grams IS NOT OLD.serving_grams
    OR NEW.kcal IS NOT OLD.kcal OR NEW.protein_g IS NOT OLD.protein_g OR NEW.carbs_g IS NOT OLD.carbs_g
    OR NEW.fat_g IS NOT OLD.fat_g OR NEW.fiber_g IS NOT OLD.fiber_g
    OR NEW.data_provenance IS NOT OLD.data_provenance OR NEW.dataset_version IS NOT OLD.dataset_version
    OR NEW.created_at IS NOT OLD.created_at
    OR (OLD.superseded_at IS NOT NULL AND NEW.superseded_at IS NOT OLD.superseded_at)
BEGIN
    SELECT RAISE(ABORT, 'nutrition_foods rows are immutable');
END;

CREATE TRIGGER nutrition_foods_no_delete BEFORE DELETE ON nutrition_foods
BEGIN
    SELECT RAISE(ABORT, 'nutrition_foods rows are never deleted');
END;

-- One row per meal. idempotency_key is 32 lowercase hex characters made by the
-- form, so a double submit inserts one meal.
CREATE TABLE nutrition_meals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    local_date TEXT NOT NULL CHECK (local_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    meal_slot TEXT NOT NULL CHECK (meal_slot IN ('breakfast', 'lunch', 'dinner', 'snack')),
    eaten_at TEXT CHECK (eaten_at IS NULL OR (length(eaten_at) = 30 AND eaten_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z')),
    created_via TEXT NOT NULL CHECK (created_via IN ('manual', 'camera', 'chat', 'barcode')),
    idempotency_key TEXT NOT NULL UNIQUE CHECK (length(idempotency_key) = 32 AND idempotency_key NOT GLOB '*[^0-9a-f]*'),
    created_at TEXT NOT NULL CHECK (length(created_at) = 30 AND created_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z'),
    archived_at TEXT CHECK (archived_at IS NULL OR (length(archived_at) = 30 AND archived_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z'))
);
CREATE INDEX idx_nutrition_meals_date ON nutrition_meals(local_date);

-- Items snapshot the name, quantity and nutrients at log time, so later
-- catalogue edits never change history. food_id points at the exact catalogue
-- row used and food_version repeats its version (both NULL for a free-form
-- item). unit is g or serving; grams is the weight when known (equal to
-- amount for g). Removing an item sets archived_at; nothing is deleted.
CREATE TABLE nutrition_meal_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    meal_id INTEGER NOT NULL REFERENCES nutrition_meals(id),
    food_id INTEGER,
    food_version INTEGER,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 120),
    unit TEXT NOT NULL CHECK (unit IN ('g', 'serving')),
    amount REAL NOT NULL CHECK (amount > 0 AND amount <= 100000),
    grams REAL CHECK (grams IS NULL OR (grams > 0 AND grams <= 100000)),
    quantity_provenance TEXT NOT NULL CHECK (quantity_provenance IN ('user_confirmed', 'ai_estimate', 'usual_portion', 'recipe_serving')),
    nutrition_provenance TEXT NOT NULL CHECK (nutrition_provenance IN ('label', 'database', 'user', 'ai_estimate')),
    kcal REAL CHECK (kcal IS NULL OR (kcal >= 0 AND kcal <= 100000)),
    protein_g REAL CHECK (protein_g IS NULL OR (protein_g >= 0 AND protein_g <= 10000)),
    carbs_g REAL CHECK (carbs_g IS NULL OR (carbs_g >= 0 AND carbs_g <= 10000)),
    fat_g REAL CHECK (fat_g IS NULL OR (fat_g >= 0 AND fat_g <= 10000)),
    fiber_g REAL CHECK (fiber_g IS NULL OR (fiber_g >= 0 AND fiber_g <= 10000)),
    created_at TEXT NOT NULL CHECK (length(created_at) = 30 AND created_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z'),
    archived_at TEXT CHECK (archived_at IS NULL OR (length(archived_at) = 30 AND archived_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9].[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]Z')),
    CHECK ((food_id IS NULL) = (food_version IS NULL)),
    CHECK (unit <> 'g' OR (grams IS NOT NULL AND grams = amount)),
    FOREIGN KEY (food_id, food_version) REFERENCES nutrition_foods(id, version)
);
CREATE INDEX idx_nutrition_meal_items_meal ON nutrition_meal_items(meal_id);
CREATE INDEX idx_nutrition_meal_items_live ON nutrition_meal_items(meal_id) WHERE archived_at IS NULL;
