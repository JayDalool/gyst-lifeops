package handlers

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"lifeops-tracker/internal/db"
)

// Most handler tests need a fresh, fully migrated database. Migrating one
// takes about a second under -race, and hundreds of tests did it, which put
// this package close to go test's 10-minute limit on the CI VM (Issue #196).
// So the migrations run once per test binary into a template, and every test
// gets its own copy of the template's bytes: the same schema and seed rows,
// and still one independent database per test. One difference from migrating
// per test: the seed rows' random uuid values and generated timestamps are
// the same in every copy within one run (still new on every run). No test
// relies on them differing between tests.
//
// Two guards keep the shortcut honest (Issue #198):
//   - Nothing re-migrates a copy. Each copy must already record every
//     migration file in schema_migrations, so an empty or partly migrated
//     template fails every test instead of quietly costing a full migration
//     per test again.
//   - The template file exists only while it is built. It is built in the
//     first caller's t.TempDir() and deleted once its bytes are in memory,
//     so no run, passing, failing or panicking, leaves it behind.

var testDBTemplate struct {
	once       sync.Once
	data       []byte
	sum        [sha256.Size]byte
	migrations []string // the migration files it holds, sorted
	err        error
}

func testMigrationsDir() string { return filepath.Join("..", "..", "migrations") }

// migratedTemplate builds the template on first use, in t's temporary
// directory, and returns its bytes and migration files.
func migratedTemplate(t *testing.T) ([]byte, []string, error) {
	testDBTemplate.once.Do(func() {
		// Set first, so a build cut short by Fatal or a panic can't leave a
		// template that looks finished.
		testDBTemplate.err = errors.New("the template build did not finish")
		data, migrations, err := templateFrom(t.TempDir(), testMigrationsDir())
		testDBTemplate.data, testDBTemplate.migrations, testDBTemplate.err = data, migrations, err
		testDBTemplate.sum = sha256.Sum256(data)
	})
	return testDBTemplate.data, testDBTemplate.migrations, testDBTemplate.err
}

// templateFrom migrates a new database in dir and returns the file's bytes
// and the migration files it applied, then deletes dir. dir should come from
// t.TempDir(), whose cleanup retries a deletion that still fails and reports
// it.
func templateFrom(dir, migrationsDir string) ([]byte, []string, error) {
	defer removeTemplateDir(dir)
	migrations, err := migrationFiles(migrationsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("list the migrations: %w", err)
	}
	path := filepath.Join(dir, "template.db")
	database, err := db.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open the template db: %w", err)
	}
	if err := db.Migrate(database, migrationsDir); err != nil {
		database.Close()
		return nil, nil, fmt.Errorf("migrate the template db: %w", err)
	}
	// Fold the WAL into the main file, which alone is read below.
	var busy, walPages, checkpointed int
	if err := database.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &walPages, &checkpointed); err != nil {
		database.Close()
		return nil, nil, fmt.Errorf("checkpoint the template db: %w", err)
	}
	if busy != 0 {
		database.Close()
		return nil, nil, errors.New("checkpoint the template db: SQLite reported busy")
	}
	if err := database.Close(); err != nil {
		return nil, nil, fmt.Errorf("close the template db: %w", err)
	}
	// Closing the last connection deletes the WAL. A WAL with content left
	// behind could hold commits the main file lacks (docs/RULES.md §3.1).
	if info, err := os.Stat(path + "-wal"); err == nil && info.Size() > 0 {
		return nil, nil, fmt.Errorf("the template db's WAL still holds %d bytes after closing", info.Size())
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("check the template db's WAL: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read the template db: %w", err)
	}
	return data, migrations, nil
}

// removeTemplateDir deletes dir, retrying for up to two seconds as the
// testing package's TempDir cleanup does: on Windows a virus scanner or the
// indexer can briefly hold a file that was just written.
func removeTemplateDir(dir string) {
	deadline := time.Now().Add(2 * time.Second)
	for wait := time.Millisecond; os.RemoveAll(dir) != nil && time.Now().Before(deadline); wait *= 2 {
		time.Sleep(wait)
	}
}

// migrationFiles lists the migrations the way db.Migrate selects them: the
// .sql files in dir, sorted by name.
func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .sql files in %s", dir)
	}
	sort.Strings(files)
	return files, nil
}

// migratedTestDB returns an open database of the test's own: a copy of the
// migrated template at t.TempDir()/lifeops.db, closed when the test ends.
func migratedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	data, migrations, err := migratedTemplate(t)
	if err != nil {
		t.Fatalf("build the template db: %v", err)
	}
	database, err := openTemplateCopy(t.TempDir(), data, migrations)
	if err != nil {
		t.Fatalf("copy the template db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// openTemplateCopy writes a template's bytes to dir/lifeops.db, opens it and
// checks it holds every migration. db.Migrate isn't run on the copy: it would
// find nothing to do, and on a template missing migrations it would quietly
// apply them in every test, hiding the fault.
func openTemplateCopy(dir string, data []byte, migrations []string) (*sql.DB, error) {
	path := filepath.Join(dir, "lifeops.db")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, err
	}
	database, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	if err := checkFullyMigrated(database, migrations); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

// checkFullyMigrated requires schema_migrations to record exactly the given
// migration files.
func checkFullyMigrated(database *sql.DB, migrations []string) error {
	rows, err := database.Query(`SELECT filename FROM schema_migrations ORDER BY filename`)
	if err != nil {
		return fmt.Errorf("not a migrated database: %w", err)
	}
	defer rows.Close()
	var applied []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		applied = append(applied, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if slices.Equal(applied, migrations) {
		return nil
	}
	var missing, unexpected []string
	for _, name := range migrations {
		if !slices.Contains(applied, name) {
			missing = append(missing, name)
		}
	}
	for _, name := range applied {
		if !slices.Contains(migrations, name) {
			unexpected = append(unexpected, name)
		}
	}
	return fmt.Errorf("not fully migrated: schema_migrations has %d of %d migrations; missing %q, unexpected %q",
		len(applied)-len(unexpected), len(migrations), missing, unexpected)
}

// A copy holds exactly what a fresh migration produces: the same schema
// objects, the same seed rows and the same settings db.Open applies. Two
// kinds of value can't repeat between migrations, so they are checked for
// form instead of equality: timestamps the database generated (columns whose
// default is the current time) must still look like timestamps, and random
// uuid defaults must still be 32 lowercase hex digits. Everything else,
// including timestamps set to fixed values, is compared exactly. The
// negative cases show the comparison catches a corrupted copy.
func TestMigratedTestDBMatchesFreshMigration(t *testing.T) {
	fresh, err := db.Open(filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fresh.Close() })
	if err := db.Migrate(fresh, testMigrationsDir()); err != nil {
		t.Fatal(err)
	}

	t.Run("copy matches", func(t *testing.T) {
		if diffs := freshMigrationDifferences(t, fresh, migratedTestDB(t)); len(diffs) > 0 {
			t.Fatalf("a copy differs from a fresh migration:\n%s", strings.Join(diffs, "\n"))
		}
	})
	for _, tc := range []struct{ name, corrupt, want string }{
		{"malformed generated timestamp", `UPDATE tracker_items SET created_at = 'garbage' WHERE rowid = (SELECT min(rowid) FROM tracker_items)`, "table tracker_items"},
		{"missing seed row", `DELETE FROM roadmaps`, "table roadmaps"},
		{"changed seed value", `UPDATE tracker_items SET title = 'Changed' WHERE key = 'prayer_fajr'`, "table tracker_items"},
		{"malformed uuid", `UPDATE roadmaps SET uuid = 'not-a-uuid'`, "table roadmaps"},
		{"extra schema object", `CREATE INDEX zz_extra ON tracker_items(title)`, "schema"},
	} {
		t.Run("detects "+tc.name, func(t *testing.T) {
			copied := migratedTestDB(t)
			if _, err := copied.Exec(tc.corrupt); err != nil {
				t.Fatal(err)
			}
			diffs := freshMigrationDifferences(t, fresh, copied)
			found := false
			for _, d := range diffs {
				found = found || strings.HasPrefix(d, tc.want)
			}
			if !found {
				t.Fatalf("corruption %q went unnoticed (differences: %q)", tc.corrupt, diffs)
			}
		})
	}
}

// freshMigrationDifferences compares the schema, every table's rows and the
// connection settings of two databases; see TestMigratedTestDBMatchesFreshMigration.
func freshMigrationDifferences(t *testing.T, fresh, copied *sql.DB) []string {
	t.Helper()
	var diffs []string
	const schema = `SELECT type, name, tbl_name, IFNULL(sql, '') FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name`
	if a, b := queryRows(t, fresh, schema), queryRows(t, copied, schema); !equalRows(a, b) {
		diffs = append(diffs, fmt.Sprintf("schema:\n  fresh:  %q\n  copied: %q", a, b))
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
			diffs = append(diffs, fmt.Sprintf("table %s:\n  fresh:  %q\n  copied: %q", table, a, b))
		}
		seeded += len(a)
	}
	if seeded == 0 {
		t.Fatal("no seed rows compared; the check would pass on empty databases")
	}
	for _, pragma := range []string{"journal_mode", "foreign_keys", "user_version", "page_size", "auto_vacuum", "encoding"} {
		q := "PRAGMA " + pragma
		if a, b := queryRows(t, fresh, q), queryRows(t, copied, q); !equalRows(a, b) {
			diffs = append(diffs, fmt.Sprintf("PRAGMA %s: fresh %q, copied %q", pragma, a, b))
		}
	}
	return diffs
}

// Copies are independent: a write to one database is invisible to another
// copy and to every later copy, and the template's bytes never change (they
// exist only in memory; TestTemplateIsDeletedOnceRead covers the file). The
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
}

// A copy must already hold every migration: an empty or partly migrated
// template is refused, with the missing migrations named, before any test
// uses it. The real template is the positive control.
func TestMigratedTestDBRejectsIncompleteTemplate(t *testing.T) {
	data, migrations, err := migratedTemplate(t)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) < 2 {
		t.Fatalf("only %d migrations; want enough to leave one out", len(migrations))
	}
	t.Run("complete template accepted", func(t *testing.T) {
		database, err := openTemplateCopy(t.TempDir(), data, migrations)
		if err != nil {
			t.Fatalf("the real template was refused: %v", err)
		}
		database.Close()
	})
	t.Run("empty template refused", func(t *testing.T) {
		database, err := openTemplateCopy(t.TempDir(), nil, migrations)
		if err == nil {
			database.Close()
			t.Fatal("an empty template was accepted")
		}
		if !strings.Contains(err.Error(), "schema_migrations") {
			t.Fatalf("the error doesn't say the copy isn't migrated: %v", err)
		}
	})
	t.Run("partly migrated template refused", func(t *testing.T) {
		newest := migrations[len(migrations)-1]
		partial, applied, err := templateFrom(t.TempDir(), copyMigrations(t, len(migrations)-1))
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(applied, newest) || len(applied) != len(migrations)-1 {
			t.Fatalf("the partial template applied %q; want all but %s", applied, newest)
		}
		database, err := openTemplateCopy(t.TempDir(), partial, migrations)
		if err == nil {
			database.Close()
			t.Fatalf("a template without %s was accepted", newest)
		}
		if !strings.Contains(err.Error(), newest) {
			t.Fatalf("the error doesn't name the missing migration %s: %v", newest, err)
		}
	})
}

// The template exists on disk only while it is built: its directory is gone
// once the bytes are read. So a test binary killed later without running
// cleanups (a go test timeout) leaves no template behind; only the build
// itself, about a second, is exposed.
func TestTemplateIsDeletedOnceRead(t *testing.T) {
	dir := t.TempDir()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("positive control: the build directory must exist first: %v", err)
	}
	// One migration is enough to build a database file.
	data, _, err := templateFrom(dir, copyMigrations(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("the template has no bytes")
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the template directory %s is still there after the build (stat: %v)", dir, err)
	}
}

// copyMigrations copies the first n migration files into a new directory and
// returns it.
func copyMigrations(t *testing.T, n int) string {
	t.Helper()
	files, err := migrationFiles(testMigrationsDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range files[:n] {
		body, err := os.ReadFile(filepath.Join(testMigrationsDir(), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
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

var (
	randomHex = regexp.MustCompile(`^[0-9a-f]{32}$`)
	// SQLite's CURRENT_TIMESTAMP / datetime('now') form and strftime ISO forms.
	generatedTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(\.\d{1,9})?(Z|[+-]\d{2}:?\d{2})?$`)
	// Column defaults that take the current time.
	nowDefault = regexp.MustCompile(`(?i)current_timestamp|'now'`)
)

// maskVolatile replaces the values a migration cannot repeat, and only when
// they have the expected form: timestamps the database generated (the
// column's default takes the current time) and the random uuid defaults,
// lower(hex(randomblob(16))). A malformed value is left as it is, so it
// still differs.
func maskVolatile(t *testing.T, database *sql.DB, table string, rows [][]string) [][]string {
	t.Helper()
	var timeColumns, randomColumns []int
	for _, column := range queryRows(t, database, fmt.Sprintf(`SELECT cid, name, IFNULL(dflt_value, '') FROM pragma_table_info('%s')`, table)) {
		cid, err := strconv.Atoi(column[0])
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case nowDefault.MatchString(column[2]):
			timeColumns = append(timeColumns, cid)
		case strings.Contains(column[2], "randomblob"):
			randomColumns = append(randomColumns, cid)
		}
	}
	for _, row := range rows {
		for _, i := range timeColumns {
			if i < len(row) && generatedTime.MatchString(row[i]) {
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
