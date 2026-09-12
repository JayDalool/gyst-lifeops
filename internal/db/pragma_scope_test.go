package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
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
	t.Cleanup(func() { releaseConnections(connections) })
	return connections
}

// releaseConnections hands the connections back to the pool. A test that runs
// a further query after inspecting every connection must call this first: the
// pool is unbounded today, so asking for one more connection while N are held
// happens to succeed, but it would block until the test timed out if a maximum
// were ever configured. Closing twice is harmless; the second Close reports
// sql.ErrConnDone, which the cleanup ignores.
func releaseConnections(connections []*sql.Conn) {
	for _, connection := range connections {
		connection.Close()
	}
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

	connections := openConnections(t, database, 4)
	for index, connection := range connections {
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
	// Release before the count below so it cannot be starved of a connection.
	releaseConnections(connections)

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
		{"file uri", "file:/tmp/x.db", "file:/tmp/x.db?" + connectionPragmas},
		{"file uri with query", "file:/tmp/x.db?mode=ro", "file:/tmp/x.db?mode=ro&" + connectionPragmas},
		{"unrelated pragma is allowed", "file:/tmp/x.db?_pragma=synchronous(1)",
			"file:/tmp/x.db?_pragma=synchronous(1)&" + connectionPragmas},
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

// TestOpenRejectsInMemoryDatabases covers every spelling of an in-memory
// database. Each pooled connection to one gets its own private, empty copy, so
// a schema created on the first connection is missing from the second. The
// previous version of this test asked only for PRAGMA values through a single
// implicit connection, which passed while the form remained unusable.
func TestOpenRejectsInMemoryDatabases(t *testing.T) {
	for _, testCase := range []struct {
		name string
		path string
	}{
		{"bare", ":memory:"},
		{"file uri", "file::memory:"},
		{"shared cache", "file::memory:?cache=shared"},
		{"mode parameter", "file:/tmp/ignored.db?mode=memory"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			database, err := Open(testCase.path)
			if err == nil {
				database.Close()
				t.Fatalf("Open(%q) succeeded; want rejection", testCase.path)
			}
			if !errors.Is(err, ErrInMemoryUnsupported) {
				t.Errorf("Open(%q) = %v, want ErrInMemoryUnsupported", testCase.path, err)
			}
		})
	}
}

// TestUppercaseMemorySpellingIsAnOrdinaryFile pins the case sensitivity the
// rejection above deliberately mirrors. SQLite compares ":memory:" exactly, so
// ":MEMORY:" asks for a file with that name and must keep working; rejecting it
// would refuse a legitimate, if unusual, filename.
func TestUppercaseMemorySpellingIsAnOrdinaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ":MEMORY:")

	database, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q): %v", path, err)
	}
	defer database.Close()

	if _, err := database.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected an ordinary file at %q: %v", path, err)
	}
}

// TestInMemoryDatabasesWouldNotShareState documents why the rejection above
// exists, by reproducing the defect directly against the driver: the pool hands
// out independent databases. If SQLite or the driver ever changes so that this
// test fails, the rejection can be revisited.
func TestInMemoryDatabasesWouldNotShareState(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	first, err := database.Conn(ctx)
	if err != nil {
		t.Fatalf("first connection: %v", err)
	}
	defer first.Close()
	second, err := database.Conn(ctx)
	if err != nil {
		t.Fatalf("second connection: %v", err)
	}
	defer second.Close()

	if _, err := first.ExecContext(ctx, `CREATE TABLE shared (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create on first connection: %v", err)
	}
	var count int
	err = second.QueryRowContext(ctx, `SELECT COUNT(*) FROM shared`).Scan(&count)
	if err == nil {
		t.Fatal("the second connection saw the first connection's table; in-memory isolation may no longer apply")
	}
	t.Logf("second connection cannot see the first connection's table: %v", err)
}

// TestOpenRejectsPragmaOverrides is the H1 guard. The driver executes every
// _pragma in lexicographic order and the last one wins, so "foreign_keys(off)"
// sorts after the "foreign_keys(1)" this package appends and would silently
// disable enforcement. Verified spellings that beat it before this guard
// existed: off, false and OFF. "foreign_keys(0)" happened to lose, but is
// refused too rather than relying on collation.
func TestOpenRejectsPragmaOverrides(t *testing.T) {
	directory := t.TempDir()
	for _, override := range []string{
		"foreign_keys(off)",
		"foreign_keys(false)",
		"foreign_keys(OFF)",
		"foreign_keys(0)",
		"FOREIGN_KEYS(off)",
		"foreign_keys = off",
		"busy_timeout(0)",
	} {
		t.Run(override, func(t *testing.T) {
			dsn := "file://" + filepath.ToSlash(filepath.Join(directory, "x.db")) +
				"?_pragma=" + url.QueryEscape(override)
			database, err := Open(dsn)
			if err == nil {
				database.Close()
				t.Fatalf("Open accepted a DSN overriding %q", override)
			}
			if !strings.Contains(err.Error(), "_pragma=") {
				t.Errorf("error does not explain the rejected pragma: %v", err)
			}
		})
	}
}

// TestOpenAllowsUnrelatedPragmas keeps the H1 guard narrow: only the pragmas
// this package guarantees are refused.
func TestOpenAllowsUnrelatedPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unrelated.db")
	dsn := "file://" + filepath.ToSlash(path) + "?_pragma=" + url.QueryEscape("synchronous(1)")

	database, err := Open(dsn)
	if err != nil {
		t.Fatalf("open with an unrelated pragma: %v", err)
	}
	defer database.Close()

	var synchronous, foreignKeys int
	if err := database.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatalf("synchronous: %v", err)
	}
	if err := database.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if synchronous != 1 {
		t.Errorf("caller pragma not applied: synchronous = %d, want 1", synchronous)
	}
	if foreignKeys != 1 {
		t.Errorf("guaranteed pragma lost: foreign_keys = %d, want 1", foreignKeys)
	}
}
