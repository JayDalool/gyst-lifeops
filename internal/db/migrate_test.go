package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func copyMigration(t *testing.T, sourceDir, targetDir, name string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(sourceDir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, name), body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFreshDatabaseAppliesEveryMigration(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if err := Migrate(database, "../../migrations"); err != nil {
		t.Fatalf("fresh migration path: %v", err)
	}

	var applied int
	if err := database.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 11 {
		t.Fatalf("applied migrations = %d, want 11", applied)
	}
	var externalLinksTable int
	if err := database.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='external_links'`).Scan(&externalLinksTable); err != nil {
		t.Fatal(err)
	}
	if externalLinksTable != 1 {
		t.Fatal("external_links table was not created")
	}
	assertNoForeignKeyViolations(t, database)
}

func TestPreQuestDatabaseUpgradesThroughLatest(t *testing.T) {
	sourceDir := "../../migrations"
	migrationDir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(migrationDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"001_init.sql", "002_trackers.sql", "003_ai_roadmap.sql"} {
		copyMigration(t, sourceDir, migrationDir, name)
	}

	database, err := Open(filepath.Join(filepath.Dir(migrationDir), "upgrade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := Migrate(database, migrationDir); err != nil {
		t.Fatalf("apply pre-quest migrations: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO projects (id, name, status) VALUES
		(1, 'Completed legacy quest', 'completed'),
		(2, 'Paused legacy quest', 'paused')`); err != nil {
		t.Fatalf("seed legacy projects: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO project_items (project_id, title) VALUES (1, 'Legacy mission')`); err != nil {
		t.Fatalf("seed legacy mission: %v", err)
	}

	for _, name := range []string{
		"004_quest_system.sql",
		"005_fix_project_status_check.sql",
		"006_multiple_roadmaps.sql",
		"007_courses.sql",
		"008_app_settings.sql",
		"009_quest_reward_integrity.sql",
		"010_project_item_rewards.sql",
		"011_external_links.sql",
	} {
		copyMigration(t, sourceDir, migrationDir, name)
	}
	if err := Migrate(database, migrationDir); err != nil {
		t.Fatalf("upgrade pre-quest database: %v", err)
	}

	for id, want := range map[int]string{1: "done", 2: "upcoming"} {
		var got string
		if err := database.QueryRow(`SELECT status FROM projects WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("project %d status = %q, want %q", id, got, want)
		}
	}
	var items int
	if err := database.QueryRow(`SELECT COUNT(*) FROM project_items WHERE project_id = 1`).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if items != 1 {
		t.Fatalf("legacy missions after upgrade = %d, want 1", items)
	}
	var starsReward int
	var proofType string
	if err := database.QueryRow(`SELECT stars_reward, proof_type FROM project_items WHERE project_id = 1`).Scan(&starsReward, &proofType); err != nil {
		t.Fatal(err)
	}
	if starsReward != 1 || proofType != "none" {
		t.Fatalf("legacy mission reward defaults = %d/%q, want 1/none", starsReward, proofType)
	}
	if _, err := database.Exec(`INSERT INTO external_links
		(uuid, entity_type, entity_id, provider, external_key, url, title, summary)
		VALUES ('upgrade-link', 'project_item', 1, 'jira', 'LEGACY-1', 'https://example.atlassian.net/browse/LEGACY-1', 'Jira', 'Legacy reference')`); err != nil {
		t.Fatalf("external links unavailable after upgrade: %v", err)
	}
	assertNoForeignKeyViolations(t, database)
}

func assertNoForeignKeyViolations(t *testing.T, database interface {
	Query(string, ...any) (*sql.Rows, error)
}) {
	t.Helper()
	rows, err := database.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported violations")
	}
}

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

// Verifies migration 006 adds the roadmaps table additively, creates the default
// roadmap, and adopts every pre-existing project into it without FK damage.
func TestMigration006MultipleRoadmaps(t *testing.T) {
	src := "../../migrations"
	migDir := filepath.Join(t.TempDir(), "migrations")
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
	early := []string{"001_init.sql", "002_trackers.sql", "003_ai_roadmap.sql", "004_quest_system.sql", "005_fix_project_status_check.sql"}
	for _, m := range early {
		copyFile(m)
	}

	database, err := Open(filepath.Join(filepath.Dir(migDir), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := Migrate(database, migDir); err != nil {
		t.Fatalf("apply 001-005: %v", err)
	}

	// Two pre-existing projects with no roadmap concept yet.
	if _, err := database.Exec(`INSERT INTO projects (id, name) VALUES (1, 'Legacy A'), (2, 'Legacy B')`); err != nil {
		t.Fatalf("insert legacy projects: %v", err)
	}

	copyFile("006_multiple_roadmaps.sql")
	if err := Migrate(database, migDir); err != nil {
		t.Fatalf("apply 006: %v", err)
	}

	// Default roadmap created, featured.
	var name string
	var featured int
	if err := database.QueryRow(`SELECT name, featured FROM roadmaps WHERE name = 'LifeOps Roadmap'`).Scan(&name, &featured); err != nil {
		t.Fatalf("default roadmap missing: %v", err)
	}
	if featured != 1 {
		t.Fatalf("default roadmap should be featured")
	}

	// Both legacy projects adopted into the default roadmap (none left NULL).
	var unassigned int
	if err := database.QueryRow(`SELECT COUNT(*) FROM projects WHERE roadmap_id IS NULL`).Scan(&unassigned); err != nil {
		t.Fatal(err)
	}
	if unassigned != 0 {
		t.Fatalf("expected all projects assigned a roadmap, %d still NULL", unassigned)
	}

	// FK integrity intact.
	rows, err := database.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported violations after 006")
	}
}
