package db

import (
	"os"
	"path/filepath"
	"testing"
)

// Verifies migration 005 rebuilds projects without losing child rows to a
// foreign-key cascade and that the new status CHECK accepts quest states.
func TestMigration005RebuildIsDataSafe(t *testing.T) {
	src := "../../migrations"
	tmp := t.TempDir()

	// Stage migrations 001-004 only, run them, seed data, THEN add 005 and
	// re-run -- mirroring how an existing populated DB upgrades.
	early := []string{"001_init.sql", "002_trackers.sql", "003_ai_roadmap.sql", "004_quest_system.sql"}
	migDir := filepath.Join(tmp, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile := func(name string) {
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(migDir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, m := range early {
		copyFile(m)
	}

	dbPath := filepath.Join(tmp, "test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if err := Migrate(database, migDir); err != nil {
		t.Fatalf("apply 001-004: %v", err)
	}

	// Parent project + a child project + a project_item + a dependent project.
	if _, err := database.Exec(`INSERT INTO projects (id, name, project_kind) VALUES (1, 'Parent', 'main')`); err != nil {
		t.Fatalf("insert parent: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO projects (id, name, project_kind, parent_project_id) VALUES (2, 'Child', 'subproject', 1)`); err != nil {
		t.Fatalf("insert child: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO projects (id, name, depends_on_project_id) VALUES (3, 'Dependent', 1)`); err != nil {
		t.Fatalf("insert dependent: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO project_items (id, project_id, title) VALUES (1, 1, 'Mission A'), (2, 2, 'Mission B')`); err != nil {
		t.Fatalf("insert items: %v", err)
	}
	// roadmap_project_meta also references projects ON DELETE CASCADE; the
	// roadmap page's JOIN depends on these rows surviving the rebuild.
	if _, err := database.Exec(`INSERT INTO roadmap_project_meta (project_id, project_type) VALUES (1, 'real_estate_ai'), (2, 'learning')`); err != nil {
		t.Fatalf("insert roadmap meta: %v", err)
	}

	// Before 005 the old CHECK must reject 'locked'.
	if _, err := database.Exec(`INSERT INTO projects (name, status) VALUES ('Locked', 'locked')`); err == nil {
		t.Fatal("expected old CHECK to reject status 'locked', but insert succeeded")
	}

	// Apply 005.
	copyFile("005_fix_project_status_check.sql")
	if err := Migrate(database, migDir); err != nil {
		t.Fatalf("apply 005: %v", err)
	}

	// Children must survive the rebuild (no cascade wipe).
	var items int
	if err := database.QueryRow(`SELECT COUNT(*) FROM project_items`).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if items != 2 {
		t.Fatalf("project_items lost in rebuild: got %d, want 2", items)
	}

	var meta int
	if err := database.QueryRow(`SELECT COUNT(*) FROM roadmap_project_meta`).Scan(&meta); err != nil {
		t.Fatal(err)
	}
	if meta != 2 {
		t.Fatalf("roadmap_project_meta lost in rebuild: got %d, want 2", meta)
	}

	var projects int
	if err := database.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&projects); err != nil {
		t.Fatal(err)
	}
	if projects != 3 {
		t.Fatalf("projects lost in rebuild: got %d, want 3", projects)
	}

	// Relationships preserved.
	var parent int64
	if err := database.QueryRow(`SELECT parent_project_id FROM projects WHERE id = 2`).Scan(&parent); err != nil {
		t.Fatal(err)
	}
	if parent != 1 {
		t.Fatalf("child parent_project_id lost: got %d, want 1", parent)
	}

	// FK integrity intact after rename.
	rows, err := database.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported violations after rebuild")
	}

	// New CHECK now accepts quest states.
	for _, s := range []string{"locked", "upcoming", "done"} {
		if _, err := database.Exec(`INSERT INTO projects (name, status) VALUES (?, ?)`, "P-"+s, s); err != nil {
			t.Fatalf("new CHECK rejected status %q: %v", s, err)
		}
	}

	// project_items FK still cascades on parent delete (ON DELETE CASCADE).
	if _, err := database.Exec(`DELETE FROM projects WHERE id = 1`); err != nil {
		t.Fatalf("delete parent: %v", err)
	}
	var remaining int
	if err := database.QueryRow(`SELECT COUNT(*) FROM project_items WHERE project_id = 1`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("ON DELETE CASCADE not re-established: %d orphan items", remaining)
	}
}
