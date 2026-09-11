package db

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHealthMigrationPreservesExistingData(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "017_health_core.sql" {
			copyMigration(t, "../../migrations", dir, entry.Name())
		}
	}
	database, err := Open(filepath.Join(t.TempDir(), "upgrade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := Migrate(database, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO tasks (title) VALUES ('Preserved task'); INSERT INTO projects (name) VALUES ('Preserved project'); INSERT INTO goals (title) VALUES ('Preserved goal')`); err != nil {
		t.Fatal(err)
	}
	// Compare all existing table rows and schema definitions, not just counts.
	var tables []string
	rows, err := database.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	capture := func() map[string]any {
		out := map[string]any{}
		for _, table := range tables {
			var schema string
			if err := database.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&schema); err != nil {
				t.Fatal(err)
			}
			out[table+" schema"] = schema
			records, err := database.Query(`SELECT * FROM "` + table + `" ORDER BY rowid`)
			if err != nil {
				t.Fatal(err)
			}
			columns, err := records.Columns()
			if err != nil {
				t.Fatal(err)
			}
			values := [][]any{}
			for records.Next() {
				row := make([]any, len(columns))
				dest := make([]any, len(columns))
				for i := range row {
					dest[i] = &row[i]
				}
				if err := records.Scan(dest...); err != nil {
					t.Fatal(err)
				}
				values = append(values, row)
			}
			if err := records.Err(); err != nil {
				t.Fatal(err)
			}
			records.Close()
			out[table] = values
		}
		return out
	}
	before := capture()
	copyMigration(t, "../../migrations", dir, "017_health_core.sql")
	for i := 0; i < 2; i++ {
		if err := Migrate(database, dir); err != nil {
			t.Fatal(err)
		}
	}
	if after := capture(); !reflect.DeepEqual(before, after) {
		t.Fatal("existing data or schema changed during Health migration")
	}
	for _, table := range []string{"health_cycles", "daily_health_snapshots", "health_weight_entries", "health_sync_states"} {
		var count int
		if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("table %s missing or seeded: %d %v", table, count, err)
		}
	}
	assertNoForeignKeyViolations(t, database)
}
