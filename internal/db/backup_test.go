package db

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// seedBackupSource builds a database shaped like the real one - the actual
// migrations, plus a foreign key and rows of its own - so a copy that silently
// loses schema or data is detectable.
func seedBackupSource(t *testing.T, path string, rows int) *sql.DB {
	t.Helper()

	database, err := Open(path)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate source: %v", err)
	}

	if _, err := database.Exec(`
		CREATE TABLE owner (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE item (
			id       INTEGER PRIMARY KEY,
			owner_id INTEGER NOT NULL REFERENCES owner(id) ON DELETE CASCADE,
			label    TEXT NOT NULL
		);
		INSERT INTO owner (id, name) VALUES (1, 'jay');
	`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	for index := 0; index < rows; index++ {
		if _, err := database.Exec(`INSERT INTO item (owner_id, label) VALUES (1, ?)`, "row"); err != nil {
			t.Fatalf("seed row %d: %v", index, err)
		}
	}
	return database
}

func countRows(t *testing.T, path, table string) int {
	t.Helper()

	dsn, err := readOnlyDSN(path)
	if err != nil {
		t.Fatalf("read-only dsn: %v", err)
	}
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer database.Close()

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count %s in %s: %v", table, path, err)
	}
	return count
}

func TestBackupCopiesEveryRow(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "lifeops.db")
	seedBackupSource(t, source, 250)

	destination := filepath.Join(dir, "out", "backup.db")
	result, err := Backup(source, destination)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}

	if got := countRows(t, destination, "item"); got != 250 {
		t.Fatalf("backup has %d items, want 250", got)
	}
	if got := countRows(t, destination, "owner"); got != 1 {
		t.Fatalf("backup has %d owners, want 1", got)
	}
	if result.Bytes <= 0 {
		t.Fatalf("result reports %d bytes", result.Bytes)
	}
	if result.Tables < 3 {
		// owner, item, schema_migrations, and everything the migrations create.
		t.Fatalf("result reports %d tables, want at least 3", result.Tables)
	}

	applied, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Migrations != int64(len(applied)) {
		t.Fatalf("result reports %d migrations, want %d (one per file in migrations/)", result.Migrations, len(applied))
	}
	if !filepath.IsAbs(result.Destination) || !filepath.IsAbs(result.Source) {
		t.Fatalf("result paths are not absolute: %q, %q", result.Source, result.Destination)
	}
}

// The copy must be a self-contained file. VACUUM INTO writes it in
// journal_mode=delete, so a backup can be moved elsewhere on its own without
// leaving committed data behind in a -wal file nobody copied.
func TestBackupIsSelfContained(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "lifeops.db")
	seedBackupSource(t, source, 10)

	destination := filepath.Join(dir, "backup.db")
	if _, err := Backup(source, destination); err != nil {
		t.Fatalf("backup: %v", err)
	}

	for _, companion := range []string{destination + "-wal", destination + "-shm"} {
		if _, err := os.Stat(companion); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("backup left %s behind (err=%v)", companion, err)
		}
	}

	dsn, err := readOnlyDSN(destination)
	if err != nil {
		t.Fatal(err)
	}
	copyDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()

	var foreignKeys int
	if err := copyDB.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign_keys from the backup: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("the backup opened with foreign_keys=%d; the DSN guarantees 1", foreignKeys)
	}
}

// The source must come through untouched. A backup that checkpoints, changes
// the journal mode, or writes anything at all is not a backup of a live
// production database - it is a change to one.
func TestBackupDoesNotModifyTheSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "lifeops.db")
	database := seedBackupSource(t, source, 50)

	var modeBefore string
	if err := database.QueryRow(`PRAGMA journal_mode`).Scan(&modeBefore); err != nil {
		t.Fatal(err)
	}
	if modeBefore != "wal" {
		t.Fatalf("source journal_mode is %q, want wal", modeBefore)
	}

	before, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	// mtime has coarse granularity on some filesystems; make any write visible.
	time.Sleep(10 * time.Millisecond)

	if _, err := Backup(source, filepath.Join(dir, "backup.db")); err != nil {
		t.Fatalf("backup: %v", err)
	}

	after, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("backup modified the source: mtime %s -> %s", before.ModTime(), after.ModTime())
	}
	if before.Size() != after.Size() {
		t.Fatalf("backup modified the source: size %d -> %d", before.Size(), after.Size())
	}

	var modeAfter string
	if err := database.QueryRow(`PRAGMA journal_mode`).Scan(&modeAfter); err != nil {
		t.Fatal(err)
	}
	if modeAfter != modeBefore {
		t.Fatalf("backup changed journal_mode from %q to %q", modeBefore, modeAfter)
	}
}

// The whole reason for VACUUM INTO: the server keeps serving during a backup.
func TestBackupSucceedsWhileTheDatabaseIsBeingWritten(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "lifeops.db")
	database := seedBackupSource(t, source, 500)

	stop := make(chan struct{})
	failures := make(chan error, 1)
	var writers sync.WaitGroup
	writers.Add(1)
	go func() {
		defer writers.Done()
		for {
			select {
			case <-stop:
				return
			default:
				if _, err := database.Exec(`INSERT INTO item (owner_id, label) VALUES (1, 'during-backup')`); err != nil {
					select {
					case failures <- err:
					default:
					}
					return
				}
			}
		}
	}()

	// Let the writer get going, so the vacuum genuinely overlaps it.
	time.Sleep(25 * time.Millisecond)

	destination := filepath.Join(dir, "backup.db")
	result, err := Backup(source, destination)
	close(stop)
	writers.Wait()

	if err != nil {
		t.Fatalf("backup under concurrent writes: %v", err)
	}
	select {
	case writeErr := <-failures:
		t.Fatalf("the backup blocked a writer: %v", writeErr)
	default:
	}

	// Backup verifies integrity itself, so reaching here means the copy is
	// sound. What this asserts is that it is also a plausible point-in-time
	// snapshot: at least what was there before the writer started, and no more
	// than what is there now.
	copied := countRows(t, destination, "item")
	live := countRows(t, source, "item")
	if copied < 500 {
		t.Fatalf("backup has %d items, want at least the 500 present before writes began", copied)
	}
	if copied > live {
		t.Fatalf("backup has %d items but the source only has %d", copied, live)
	}
	if result.Tables < 3 {
		t.Fatalf("result reports %d tables", result.Tables)
	}
}

// An existing destination is never overwritten: the most likely thing at a
// backup path is an earlier backup.
func TestBackupRefusesAnExistingDestination(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "lifeops.db")
	seedBackupSource(t, source, 5)

	destination := filepath.Join(dir, "backup.db")
	existing := []byte("an earlier backup")
	if err := os.WriteFile(destination, existing, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Backup(source, destination); err == nil {
		t.Fatal("backup overwrote an existing destination")
	} else if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error does not explain the refusal: %v", err)
	}

	after, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(existing) {
		t.Fatalf("the existing file was modified: %q", after)
	}
}

func TestBackupRefusesToWriteOverTheSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "lifeops.db")
	seedBackupSource(t, source, 5)

	if _, err := Backup(source, source); err == nil {
		t.Fatal("backup accepted the source as its own destination")
	}
	if countRows(t, source, "item") != 5 {
		t.Fatal("the source was damaged")
	}
}

// A mistyped source must fail loudly. Opening read-only is what guarantees it:
// SQLite will not create a database it is not allowed to write.
func TestBackupRefusesAMissingSource(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "not-here.db")

	if _, err := Backup(missing, filepath.Join(dir, "backup.db")); err == nil {
		t.Fatal("backup accepted a source that does not exist")
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup created the missing source (err=%v)", err)
	}
}

func TestBackupRejectsADSNSource(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "backup.db")

	for _, source := range []string{":memory:", "file:" + filepath.Join(dir, "x.db") + "?mode=rw"} {
		if _, err := Backup(source, destination); err == nil {
			t.Fatalf("backup accepted the DSN source %q", source)
		}
	}
}

// An unwritable destination directory must produce an error and no file, not a
// half-written database that looks like a backup.
func TestBackupLeavesNoFileWhenTheDestinationIsUnwritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission bits are not enforced")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "lifeops.db")
	seedBackupSource(t, source, 5)

	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o700) })

	destination := filepath.Join(locked, "backup.db")
	if _, err := Backup(source, destination); err == nil {
		t.Fatal("backup succeeded into an unwritable directory")
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a partial backup was left behind (err=%v)", err)
	}
}

// A corrupt copy must not be reported as a backup. The check cannot be
// triggered through Backup - a fresh VACUUM INTO is always sound - so it is
// exercised directly on the file that verification actually reads.
func TestVerifyBackupRejectsAFileThatIsNotADatabase(t *testing.T) {
	dir := t.TempDir()
	corrupt := filepath.Join(dir, "corrupt.db")
	if err := os.WriteFile(corrupt, []byte("SQLite format 3\x00 but not really"), 0o600); err != nil {
		t.Fatal(err)
	}

	var result BackupResult
	if err := verifyBackup(corrupt, &result); err == nil {
		t.Fatal("verification accepted a file that is not a database")
	}
}

// A database with no schema_migrations table is unusual but not corrupt. It
// reports -1, which is distinguishable from 0 applied migrations; the command
// prints "unknown" rather than a number that would read as "none applied".
func TestBackupReportsUnknownMigrationsWhenTheTableIsAbsent(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "plain.db")

	database, err := Open(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`CREATE TABLE note (id INTEGER PRIMARY KEY);`); err != nil {
		t.Fatal(err)
	}
	database.Close()

	result, err := Backup(source, filepath.Join(dir, "backup.db"))
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if result.Migrations != -1 {
		t.Fatalf("result reports %d migrations, want -1 for an absent table", result.Migrations)
	}
	if result.Tables != 1 {
		t.Fatalf("result reports %d tables, want 1", result.Tables)
	}
}

func TestDefaultBackupPathIsSortableAndNamedAfterTheSource(t *testing.T) {
	moment := time.Date(2026, 9, 12, 18, 4, 5, 0, time.FixedZone("CDT", -5*60*60))

	got := DefaultBackupPath("data/lifeops.db", moment)
	want := filepath.Join("data", "backups", "lifeops-20260912T230405Z.db")
	if got != want {
		t.Fatalf("DefaultBackupPath = %q, want %q", got, want)
	}

	// Beside the database, not beside the process: in the container only the
	// data directory is a mounted volume.
	if filepath.Dir(filepath.Dir(got)) != filepath.Dir("data/lifeops.db") {
		t.Fatalf("DefaultBackupPath %q is not beside the database", got)
	}

	earlier := DefaultBackupPath("data/lifeops.db", moment.Add(-time.Hour))
	if !(earlier < got) {
		t.Fatalf("backup names do not sort chronologically: %q then %q", earlier, got)
	}
}
