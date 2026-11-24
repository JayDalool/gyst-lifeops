package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The food diary schema (Issue #207): catalogue rows are immutable, items
// reference the exact version they used, and timestamps must be instants.
// Fictional values only.

const nutInstant = "2026-11-16T15:00:00.000000000Z"

func nutritionDB(t *testing.T) (exec func(q string, args ...any) error) {
	t.Helper()
	database, err := Open(filepath.Join(t.TempDir(), "nutrition.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if err := Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	return func(q string, args ...any) error {
		_, err := database.Exec(q, args...)
		return err
	}
}

const insertFood = `INSERT INTO nutrition_foods (provider, provider_id, version, name, normalized_name, basis, kcal, data_provenance, created_at)
 VALUES ('usda_fdc', 'oats-1', ?, 'Test oats', 'test oats', 'per_100g', 380, 'database', ?)`

func TestNutritionFoodsAreImmutableAndVersioned(t *testing.T) {
	exec := nutritionDB(t)
	if err := exec(insertFood, 1, nutInstant); err != nil {
		t.Fatal(err)
	}
	// Any content column is frozen.
	for _, col := range []string{"name = 'Other'", "kcal = 1", "basis = 'per_serving'", "normalized_name = 'x'", "data_provenance = 'user'", "version = 5", "provider_id = 'z'", "created_at = '" + strings.Replace(nutInstant, "15", "16", 1) + "'"} {
		err := exec(`UPDATE nutrition_foods SET ` + col + ` WHERE id = 1`)
		if err == nil || !strings.Contains(err.Error(), "immutable") {
			t.Errorf("UPDATE %s: err = %v, want the immutable trigger", col, err)
		}
	}
	// A second current version for the same food is refused until the first is superseded.
	if err := exec(insertFood, 2, nutInstant); err == nil {
		t.Fatal("two current versions of one food accepted")
	}
	// archived_at may change; superseded_at may be set once.
	if err := exec(`UPDATE nutrition_foods SET archived_at = ? WHERE id = 1`, nutInstant); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if err := exec(`UPDATE nutrition_foods SET superseded_at = ? WHERE id = 1`, nutInstant); err != nil {
		t.Fatalf("supersede: %v", err)
	}
	if err := exec(`UPDATE nutrition_foods SET superseded_at = NULL WHERE id = 1`); err == nil {
		t.Fatal("superseded_at was cleared")
	}
	if err := exec(`UPDATE nutrition_foods SET superseded_at = ? WHERE id = 1`, "2026-11-17T15:00:00.000000000Z"); err == nil {
		t.Fatal("superseded_at was changed a second time")
	}
	if err := exec(insertFood, 2, nutInstant); err != nil {
		t.Fatalf("next version after supersede: %v", err)
	}
	if err := exec(insertFood, 2, nutInstant); err == nil {
		t.Fatal("duplicate (provider, provider_id, version) accepted")
	}
	if err := exec(`DELETE FROM nutrition_foods WHERE id = 1`); err == nil {
		t.Fatal("catalogue row deleted")
	}
}

func TestNutritionItemsReferenceTheLoggedVersion(t *testing.T) {
	exec := nutritionDB(t)
	if err := exec(insertFood, 1, nutInstant); err != nil {
		t.Fatal(err)
	}
	if err := exec(`INSERT INTO nutrition_meals (local_date, meal_slot, created_via, idempotency_key, created_at) VALUES ('2026-11-16', 'breakfast', 'manual', ?, ?)`, strings.Repeat("a", 32), nutInstant); err != nil {
		t.Fatal(err)
	}
	item := func(foodID, version any, unit string, amount, grams any, kcal any, created string) error {
		return exec(`INSERT INTO nutrition_meal_items (meal_id, food_id, food_version, name, unit, amount, grams, quantity_provenance, nutrition_provenance, kcal, created_at)
 VALUES (1, ?, ?, 'Test oats', ?, ?, ?, 'user_confirmed', 'database', ?, ?)`, foodID, version, unit, amount, grams, kcal, created)
	}
	if err := item(1, 1, "g", 40, 40, 152, nutInstant); err != nil {
		t.Fatalf("valid item: %v", err)
	}
	// NULL nutrients are stored as NULL (unknown is not zero).
	if err := item(1, 1, "g", 40, 40, nil, nutInstant); err != nil {
		t.Fatalf("NULL kcal: %v", err)
	}
	if err := item(nil, nil, "serving", 1, nil, 90, nutInstant); err != nil {
		t.Fatalf("free-form item: %v", err)
	}
	bad := map[string]error{
		"wrong version":        item(1, 2, "g", 40, 40, 1, nutInstant),
		"food without version": item(1, nil, "g", 40, 40, 1, nutInstant),
		"unknown food":         item(9, 1, "g", 40, 40, 1, nutInstant),
		"g without grams":      item(1, 1, "g", 40, nil, 1, nutInstant),
		"grams differ":         item(1, 1, "g", 40, 41, 1, nutInstant),
		"zero amount":          item(1, 1, "g", 0, 0, 1, nutInstant),
		"negative kcal":        item(1, 1, "g", 40, 40, -1, nutInstant),
		"unknown unit":         item(1, 1, "oz", 40, 40, 1, nutInstant),
		"not an instant":       item(1, 1, "g", 40, 40, 1, "2026-11-16 15:00:00"),
	}
	for name, err := range bad {
		if err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	// Meals: the key is 32 lowercase hex and unique; the date is a calendar date.
	meal := func(key, slot, date string) error {
		return exec(`INSERT INTO nutrition_meals (local_date, meal_slot, created_via, idempotency_key, created_at) VALUES (?, ?, 'manual', ?, ?)`, date, slot, key, nutInstant)
	}
	if err := meal(strings.Repeat("a", 32), "lunch", "2026-11-16"); err == nil {
		t.Error("duplicate idempotency key accepted")
	}
	for name, err := range map[string]error{
		"short key":    meal("abc", "lunch", "2026-11-16"),
		"upper key":    meal(strings.Repeat("A", 32), "lunch", "2026-11-16"),
		"unknown slot": meal(strings.Repeat("b", 32), "brunch", "2026-11-16"),
		"bad date":     meal(strings.Repeat("c", 32), "lunch", "16/11/2026"),
	} {
		if err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

// GYST-179: the AI's own nutrient figures sit beside the saved ones. Two passes,
// as a deploy does it: a database at the previous schema holds a row written
// the old way, then the new migration runs on it. The row must survive with the
// new columns NULL, and the new columns must hold the item columns' ranges.
func TestMealItemAIEstimateMacroColumns(t *testing.T) {
	const newest = "20261123100000_meal_item_ai_est_macros.sql"
	src := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != newest {
			copyMigration(t, src, dir, e.Name())
		}
	}
	database, err := Open(filepath.Join(t.TempDir(), "aiest.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	// Pass 1: the schema as it was before this migration.
	if err := Migrate(database, dir); err != nil {
		t.Fatal(err)
	}
	var has int
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('nutrition_meal_items') WHERE name LIKE 'ai_est_%' AND name <> 'ai_est_grams'`).Scan(&has); err != nil || has != 0 {
		t.Fatalf("the previous schema already has the new columns (%d, %v)", has, err)
	}
	exec := func(q string, args ...any) error { _, err := database.Exec(q, args...); return err }
	if err := exec(`INSERT INTO nutrition_meals (local_date, meal_slot, created_via, idempotency_key, created_at) VALUES ('2026-11-16', 'lunch', 'camera', ?, ?)`, strings.Repeat("a", 32), nutInstant); err != nil {
		t.Fatal(err)
	}
	if err := exec(`INSERT INTO nutrition_meal_items (meal_id, name, unit, amount, grams, quantity_provenance, nutrition_provenance, kcal, protein_g, ai_est_grams, created_at)
 VALUES (1, 'Old row', 'g', 100, 100, 'user_confirmed', 'user', 130, 2.7, 120, ?)`, nutInstant); err != nil {
		t.Fatal(err)
	}
	// Pass 2: the new migration runs on the database that already holds data.
	copyMigration(t, src, dir, newest)
	if err := Migrate(database, dir); err != nil {
		t.Fatalf("new migration on existing data: %v", err)
	}
	var kcal, protein, estGrams float64
	var nulls int
	if err := database.QueryRow(`SELECT kcal, protein_g, ai_est_grams, (ai_est_kcal IS NULL) + (ai_est_protein_g IS NULL) + (ai_est_carbs_g IS NULL) + (ai_est_fat_g IS NULL)
 FROM nutrition_meal_items WHERE name = 'Old row'`).Scan(&kcal, &protein, &estGrams, &nulls); err != nil {
		t.Fatalf("old row lost: %v", err)
	}
	if kcal != 130 || protein != 2.7 || estGrams != 120 || nulls != 4 {
		t.Fatalf("old row changed: kcal %v protein %v ai_est_grams %v, NULL estimate columns %d", kcal, protein, estGrams, nulls)
	}
	var check string
	if err := database.QueryRow(`PRAGMA integrity_check`).Scan(&check); err != nil || check != "ok" {
		t.Fatalf("integrity_check = %q, %v", check, err)
	}
	insert := func(col string, v any) error {
		return exec(`INSERT INTO nutrition_meal_items (meal_id, name, unit, amount, grams, quantity_provenance, nutrition_provenance, `+col+`, created_at)
 VALUES (1, 'Test rice', 'g', 100, 100, 'user_confirmed', 'user', ?, ?)`, v, nutInstant)
	}
	for _, col := range []string{"ai_est_kcal", "ai_est_protein_g", "ai_est_carbs_g", "ai_est_fat_g"} {
		max := 10000.0
		if col == "ai_est_kcal" {
			max = 100000
		}
		for name, v := range map[string]any{"NULL": nil, "zero": 0, "maximum": max} {
			if err := insert(col, v); err != nil {
				t.Errorf("%s %s: %v", col, name, err)
			}
		}
		for name, v := range map[string]float64{"above max": max + 1, "negative": -0.1} {
			if err := insert(col, v); err == nil || !strings.Contains(err.Error(), "CHECK") {
				t.Errorf("%s %s accepted: %v", col, name, err)
			}
		}
	}
}
