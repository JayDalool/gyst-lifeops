package handlers

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"lifeops-tracker/internal/db"
)

// Most handler tests need a fresh, fully migrated database. Migrating one
// takes about a second under -race, and hundreds of tests did it, which put
// this package close to go test's 10-minute limit on the CI VM (Issue #196).
// So the migrations run once per test binary into a template, and every test
// gets its own copy of that file: the same schema and seed rows, and still
// one independent database per test. One difference from migrating per test:
// the seed rows' random uuid values and migration timestamps are the same in
// every copy within one run (still new on every run). No test relies on them
// differing between tests.

var testDBTemplate struct {
	once sync.Once
	dir  string
	data []byte
	sum  [sha256.Size]byte
	err  error
}

func TestMain(m *testing.M) {
	code := m.Run()
	if testDBTemplate.dir != "" {
		// The template is read-only; Windows won't delete it as it is.
		os.Chmod(filepath.Join(testDBTemplate.dir, "template.db"), 0o600)
		os.RemoveAll(testDBTemplate.dir)
	}
	os.Exit(code)
}

func testMigrationsDir() string { return filepath.Join("..", "..", "migrations") }

// migratedTemplate migrates one database and keeps its bytes. The file is
// closed (its WAL folded in) before it is read, and SQLite never opens it
// again.
func migratedTemplate() ([]byte, error) {
	testDBTemplate.once.Do(func() {
		testDBTemplate.err = func() error {
			dir, err := os.MkdirTemp("", "gyst-handlers-testdb-")
			if err != nil {
				return err
			}
			testDBTemplate.dir = dir
			path := filepath.Join(dir, "template.db")
			database, err := db.Open(path)
			if err != nil {
				return err
			}
			if err := db.Migrate(database, testMigrationsDir()); err != nil {
				database.Close()
				return err
			}
			if _, err := database.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
				database.Close()
				return err
			}
			if err := database.Close(); err != nil {
				return err
			}
			if err := os.Chmod(path, 0o400); err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			testDBTemplate.data = data
			testDBTemplate.sum = sha256.Sum256(data)
			return nil
		}()
	})
	return testDBTemplate.data, testDBTemplate.err
}

// migratedTestDB returns an open database of the test's own: a copy of the
// migrated template at t.TempDir()/lifeops.db, closed when the test ends.
// db.Migrate still runs on the copy. It finds nothing to do, and it would
// apply anything the copy lacked, so the database is fully migrated either
// way.
func migratedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	data, err := migratedTemplate()
	if err != nil {
		t.Fatalf("migrate template db: %v", err)
	}
	path := filepath.Join(t.TempDir(), "lifeops.db")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("copy template db: %v", err)
	}
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(database, testMigrationsDir()); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return database
}

// A copy holds exactly what a fresh migration produces: the same schema
// objects, the same seed rows and the same settings db.Open applies. Two
// kinds of value can't repeat between migrations and are compared by form
// only: timestamps (columns ending in _at) and the random uuid defaults.
func TestMigratedTestDBMatchesFreshMigration(t *testing.T) {
	fresh, err := db.Open(filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fresh.Close() })
	if err := db.Migrate(fresh, testMigrationsDir()); err != nil {
		t.Fatal(err)
	}
	copied := migratedTestDB(t)

	const schema = `SELECT type, name, tbl_name, IFNULL(sql, '') FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name`
	if a, b := queryRows(t, fresh, schema), queryRows(t, copied, schema); !equalRows(a, b) {
		t.Fatalf("schema differs:\nfresh:  %q\ncopied: %q", a, b)
	}
	var tables []string
	for _, row := range queryRows(t, fresh, `SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`) {
		tables = append(tables, row[0])
	}
	if len(tables) < 20 {
		t.Fatalf("only %d tables after migrating; want the full schema", len(tables))
	}
	seeded := 0
	for _, table := range tables {
		query := fmt.Sprintf(`SELECT * FROM %q`, table)
		a, b := maskVolatile(t, fresh, table, queryRows(t, fresh, query)), maskVolatile(t, copied, table, queryRows(t, copied, query))
		if !equalRows(a, b) {
			t.Errorf("table %s differs:\nfresh:  %q\ncopied: %q", table, a, b)
		}
		seeded += len(a)
	}
	if seeded == 0 {
		t.Fatal("no seed rows compared; the check would pass on empty databases")
	}
	for _, pragma := range []string{"journal_mode", "foreign_keys", "user_version", "page_size", "auto_vacuum", "encoding"} {
		q := "PRAGMA " + pragma
		if a, b := queryRows(t, fresh, q), queryRows(t, copied, q); !equalRows(a, b) {
			t.Errorf("PRAGMA %s: fresh %q, copied %q", pragma, a, b)
		}
	}
}

// Copies are independent: a write to one database is invisible to another
// copy and to every later copy, and the template's bytes never change. The
// positive control shows the check would see a shared file.
func TestMigratedTestDBsAreIndependent(t *testing.T) {
	a := migratedTestDB(t)
	b := migratedTestDB(t)
	pathA := databaseFile(t, a)
	if pathA == databaseFile(t, b) {
		t.Fatalf("two tests share %s", pathA)
	}
	if _, err := a.Exec(`CREATE TABLE zz_isolation_probe (x INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Exec(`INSERT INTO zz_isolation_probe VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	again, err := db.Open(pathA)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { again.Close() })
	if !hasProbe(t, again) {
		t.Fatal("positive control: a second handle on the same file must see the write")
	}
	if hasProbe(t, b) {
		t.Fatal("a write to one test's database reached another test's database")
	}
	if hasProbe(t, migratedTestDB(t)) {
		t.Fatal("a write to one test's database reached the template")
	}
	if sha256.Sum256(testDBTemplate.data) != testDBTemplate.sum {
		t.Fatal("the template bytes changed")
	}
	onDisk, err := os.ReadFile(filepath.Join(testDBTemplate.dir, "template.db"))
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(onDisk) != testDBTemplate.sum {
		t.Fatal("the template file changed on disk")
	}
}

func databaseFile(t *testing.T, database *sql.DB) string {
	t.Helper()
	for _, row := range queryRows(t, database, `PRAGMA database_list`) {
		if row[1] == "main" {
			return row[2]
		}
	}
	t.Fatal("no main database")
	return ""
}

func hasProbe(t *testing.T, database *sql.DB) bool {
	t.Helper()
	var n int
	if err := database.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name = 'zz_isolation_probe'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

// queryRows returns every row as strings, sorted, so tables without a
// natural order still compare.
func queryRows(t *testing.T, database *sql.DB, query string) [][]string {
	t.Helper()
	rows, err := database.Query(query)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var out [][]string
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			t.Fatal(err)
		}
		row := make([]string, len(columns))
		for i, v := range values {
			switch v := v.(type) {
			case nil:
				row[i] = "<NULL>"
			case []byte:
				row[i] = string(v)
			default:
				row[i] = fmt.Sprint(v)
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Slice(out, func(i, j int) bool { return strings.Join(out[i], "\x00") < strings.Join(out[j], "\x00") })
	return out
}

var randomHex = regexp.MustCompile(`^[0-9a-f]{32}$`)

// maskVolatile replaces the values a migration cannot repeat: timestamps in
// columns ending in _at (kept as present or empty) and the random uuid
// defaults, lower(hex(randomblob(16))) (kept only if well formed).
func maskVolatile(t *testing.T, database *sql.DB, table string, rows [][]string) [][]string {
	t.Helper()
	var timeColumns, randomColumns []int
	for _, column := range queryRows(t, database, fmt.Sprintf(`SELECT cid, name, IFNULL(dflt_value, '') FROM pragma_table_info('%s')`, table)) {
		cid, err := strconv.Atoi(column[0])
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case strings.HasSuffix(column[1], "_at"):
			timeColumns = append(timeColumns, cid)
		case strings.Contains(column[2], "randomblob"):
			randomColumns = append(randomColumns, cid)
		}
	}
	for _, row := range rows {
		for _, i := range timeColumns {
			if i < len(row) && row[i] != "" && row[i] != "<NULL>" {
				row[i] = "<time>"
			}
		}
		for _, i := range randomColumns {
			if i < len(row) && randomHex.MatchString(row[i]) {
				row[i] = "<random>"
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool { return strings.Join(rows[i], "\x00") < strings.Join(rows[j], "\x00") })
	return rows
}

func equalRows(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.Join(a[i], "\x00") != strings.Join(b[i], "\x00") {
			return false
		}
	}
	return true
}
