package db

import (
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
