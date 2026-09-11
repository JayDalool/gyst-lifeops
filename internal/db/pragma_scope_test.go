package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// openConnections holds several pooled connections open at once so the pool is
// forced to create new ones rather than handing back the single connection any
// earlier statement happened to configure.
func openConnections(t *testing.T, database *sql.DB, count int) []*sql.Conn {
	t.Helper()
	connections := make([]*sql.Conn, 0, count)
	for index := 0; index < count; index++ {
		connection, err := database.Conn(context.Background())
		if err != nil {
			t.Fatalf("open connection %d: %v", index, err)
		}
		connections = append(connections, connection)
	}
	t.Cleanup(func() {
		for _, connection := range connections {
			connection.Close()
		}
	})
	return connections
}

// TestConnectionPragmasApplyToEveryPooledConnection fails against the previous
// implementation, which executed the connection-scoped PRAGMAs once against the
// *sql.DB and so configured only the first connection of the pool.
func TestConnectionPragmasApplyToEveryPooledConnection(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "pragma.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	for index, connection := range openConnections(t, database, 4) {
		var foreignKeys, busyTimeout int
		var journalMode string
		if err := connection.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			t.Fatalf("connection %d foreign_keys: %v", index, err)
		}
		if err := connection.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
			t.Fatalf("connection %d busy_timeout: %v", index, err)
		}
		if err := connection.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
			t.Fatalf("connection %d journal_mode: %v", index, err)
		}
		if foreignKeys != 1 {
			t.Errorf("connection %d: foreign_keys = %d, want 1", index, foreignKeys)
		}
		if busyTimeout != 5000 {
			t.Errorf("connection %d: busy_timeout = %d, want 5000", index, busyTimeout)
		}
		if journalMode != "wal" {
			t.Errorf("connection %d: journal_mode = %q, want \"wal\"", index, journalMode)
		}
	}
}

// TestForeignKeysAreEnforcedOnEveryConnection is the behavioural counterpart:
// it does not ask what the PRAGMA reports, it tries to insert a row referencing
// a parent that does not exist and requires SQLite to refuse on every
// connection. Under the previous implementation only the first connection did.
func TestForeignKeysAreEnforcedOnEveryConnection(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "foreignkeys.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `
		CREATE TABLE parent (id INTEGER PRIMARY KEY);
		CREATE TABLE child (
			id        INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL REFERENCES parent(id)
		);`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	for index, connection := range openConnections(t, database, 4) {
		_, err := connection.ExecContext(ctx,
			`INSERT INTO child (id, parent_id) VALUES (?, 999)`, index+1)
		if err == nil {
			t.Errorf("connection %d accepted a row referencing a missing parent", index)
			continue
		}
		if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			t.Errorf("connection %d rejected the row for the wrong reason: %v", index, err)
		}
	}

	var orphans int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM child`).Scan(&orphans); err != nil {
		t.Fatalf("count child rows: %v", err)
	}
	if orphans != 0 {
		t.Errorf("%d orphan rows were written, want 0", orphans)
	}
}

// TestMigratedAndSeededSchemaHasNoForeignKeyViolations runs the real migrations,
// loads the real seed through the production SeedIfEmpty path, and then applies
// the same read-only integrity check that should be run against the deployed
// database before this change is released. Seeding is what gives the check
// something to find: the seed carries rows on both sides of several foreign
// keys, so an empty database would make this test pass vacuously.
func TestMigratedAndSeededSchemaHasNoForeignKeyViolations(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "integrity.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := Migrate(database, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := SeedIfEmpty(database, "../../seeds/example.sql"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Prove the seed actually loaded before trusting the check below. Each of
	// these tables holds one side of at least one foreign key in the schema.
	for _, table := range []string{
		"projects", "project_items", "tasks", "goals", "goal_progress_entries", "reminders",
	} {
		var count int
		if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count == 0 {
			t.Fatalf("seed loaded no rows into %s, so the integrity check would be vacuous", table)
		}
	}

	// The seed must exercise a real parent/child relationship, not just fill
	// unrelated tables, or the check still proves nothing about enforcement.
	var linkedItems int
	if err := database.QueryRow(`SELECT COUNT(*) FROM project_items
		WHERE project_id IN (SELECT id FROM projects)`).Scan(&linkedItems); err != nil {
		t.Fatalf("count linked project_items: %v", err)
	}
	if linkedItems == 0 {
		t.Fatal("seed contains no project_items referencing a seeded project")
	}

	violations, err := foreignKeyViolations(database)
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("seeded database reports %d foreign-key violations: %v", len(violations), violations)
	}
}

// foreignKeyViolations runs PRAGMA foreign_key_check and returns one
// description per offending row. It reads only; it never repairs anything.
func foreignKeyViolations(database *sql.DB) ([]string, error) {
	rows, err := database.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []string
	for rows.Next() {
		// table, rowid, referenced table, index of the failing foreign key.
		var table, parent string
		var rowID sql.NullInt64
		var keyIndex int
		if err := rows.Scan(&table, &rowID, &parent, &keyIndex); err != nil {
			return nil, err
		}
		rowIDText := "null"
		if rowID.Valid {
			rowIDText = strconv.FormatInt(rowID.Int64, 10)
		}
		violations = append(violations,
			fmt.Sprintf("%s rowid=%s -> %s (fk #%d)", table, rowIDText, parent, keyIndex))
	}
	return violations, rows.Err()
}

// TestSQLiteDSNHandlesAwkwardPaths covers the reason the DSN is built as a
// percent-encoded file: URI rather than by concatenating a query onto the path.
// The driver splits a DSN at the first '?', so a naive path+"?"+params would
// truncate any path containing '?' and open a different database than asked for.
func TestSQLiteDSNHandlesAwkwardPaths(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		filename string
	}{
		{"plain", "lifeops.db"},
		{"spaces", "my health data.db"},
		{"question mark", "what?now.db"},
		{"hash", "release#3.db"},
		{"percent", "100%.db"},
		{"unicode", "santé.db"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), testCase.filename)
			database, err := Open(path)
			if err != nil {
				t.Fatalf("open %q: %v", path, err)
			}
			defer database.Close()

			if _, err := database.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY)`); err != nil {
				t.Fatalf("write to %q: %v", path, err)
			}

			// The database must exist at exactly the requested path, proving the
			// path was neither truncated nor percent-decoded into something else.
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("database not created at requested path %q: %v", path, err)
			}

			// PRAGMAs must still reach every connection for these paths too.
			var foreignKeys int
			if err := database.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
				t.Fatalf("foreign_keys: %v", err)
			}
			if foreignKeys != 1 {
				t.Errorf("foreign_keys = %d, want 1", foreignKeys)
			}
		})
	}
}

// TestSQLiteDSNPreservesCallerSuppliedForms keeps ":memory:" and explicit file:
// URIs usable, so a future test can reach for an in-memory database without
// having to work around DSN construction.
func TestSQLiteDSNPreservesCallerSuppliedForms(t *testing.T) {
	for _, testCase := range []struct {
		name string
		path string
		want string
	}{
		{"memory", ":memory:", ":memory:?" + connectionPragmas},
		{"file uri", "file:/tmp/x.db", "file:/tmp/x.db?" + connectionPragmas},
		{"file uri with query", "file:/tmp/x.db?mode=ro", "file:/tmp/x.db?mode=ro&" + connectionPragmas},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := sqliteDSN(testCase.path)
			if err != nil {
				t.Fatalf("sqliteDSN(%q): %v", testCase.path, err)
			}
			if got != testCase.want {
				t.Errorf("sqliteDSN(%q) = %q, want %q", testCase.path, got, testCase.want)
			}
		})
	}

	if _, err := sqliteDSN(""); err == nil {
		t.Error("sqliteDSN(\"\") should reject an empty path")
	}
}

// TestOpenAcceptsCallerSuppliedFileURI is the end-to-end counterpart to the
// sqliteDSN unit test: it drives Open itself, so it also covers the fact that a
// file: URI must bypass filesystem directory creation. Running filepath over
// "file:///tmp/x/db" yields "file:/tmp/x", and creating that would leave a
// literal "file:" directory behind in the working directory.
func TestOpenAcceptsCallerSuppliedFileURI(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "uri.db")
	dsn := "file://" + filepath.ToSlash(path) + "?mode=rwc"

	database, err := Open(dsn)
	if err != nil {
		t.Fatalf("open %q: %v", dsn, err)
	}
	defer database.Close()

	if _, err := database.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("write through file: URI: %v", err)
	}

	// The URI must have opened the database the caller named, not some other file.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database not created at %q: %v", path, err)
	}

	// No stray directory may have been created from the URI scheme.
	if _, err := os.Stat("file:"); !os.IsNotExist(err) {
		t.Errorf(`a literal "file:" directory was created in the working directory (stat err: %v)`, err)
	}

	// The caller's own query parameter must survive alongside the appended
	// pragmas, and the pragmas must still reach every pooled connection.
	ctx := context.Background()
	for index, connection := range openConnections(t, database, 3) {
		var foreignKeys, busyTimeout int
		if err := connection.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			t.Fatalf("connection %d foreign_keys: %v", index, err)
		}
		if err := connection.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
			t.Fatalf("connection %d busy_timeout: %v", index, err)
		}
		if foreignKeys != 1 {
			t.Errorf("file: URI connection %d: foreign_keys = %d, want 1", index, foreignKeys)
		}
		if busyTimeout != 5000 {
			t.Errorf("file: URI connection %d: busy_timeout = %d, want 5000", index, busyTimeout)
		}
	}
}

// TestInMemoryDatabaseGetsConnectionPragmas proves the ":memory:" form is not
// just accepted but correctly configured.
func TestInMemoryDatabaseGetsConnectionPragmas(t *testing.T) {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open in-memory: %v", err)
	}
	defer database.Close()

	var foreignKeys, busyTimeout int
	if err := database.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if err := database.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("busy_timeout: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("in-memory foreign_keys = %d, want 1", foreignKeys)
	}
	if busyTimeout != 5000 {
		t.Errorf("in-memory busy_timeout = %d, want 5000", busyTimeout)
	}
}
