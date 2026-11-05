package app

import (
	"database/sql"
	"path/filepath"
	"testing"

	"lifeops-tracker/internal/db"
)

func TestStartupAdoptsOrphansOnceAndIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "adopt.db")
	cfg := timezoneConfig(t, dbPath, "")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database, cfg.MigrationsDir); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO projects (name, project_kind, status, updated_at) VALUES ('Fictional Orphan', 'main', 'active', '2026-01-01 00:00:00')`); err != nil {
		t.Fatal(err)
	}
	database.Close()

	read := func() (int64, string, int) {
		a, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		var rid sql.NullInt64
		var updated string
		var version int
		if err := a.DB.QueryRow(`SELECT roadmap_id, updated_at, sync_version FROM projects WHERE name = 'Fictional Orphan'`).Scan(&rid, &updated, &version); err != nil {
			t.Fatal(err)
		}
		return rid.Int64, updated, version
	}
	rid, updated, version := read()
	if rid == 0 {
		t.Fatal("orphan not adopted at startup")
	}
	rid2, updated2, version2 := read()
	if rid2 != rid || updated2 != updated || version2 != version || updated != "2026-01-01 00:00:00" {
		t.Fatalf("second startup changed the row: %d/%q/%d -> %d/%q/%d", rid, updated, version, rid2, updated2, version2)
	}
}
