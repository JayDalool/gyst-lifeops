package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// connectionPragmas are scoped to one SQLite connection, so the driver must
// apply them as it opens each pooled connection. Running them once against the
// *sql.DB is not equivalent: database/sql serves each call from an arbitrary
// connection, so a single Exec configures whichever connection happened to
// answer it and leaves every later connection on SQLite's defaults - foreign
// keys off and no busy timeout.
const connectionPragmas = "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

// escapeSQLiteURIPath escapes the three characters SQLite treats specially in
// the path component of a URI filename. '?' and '#' terminate the path and '%'
// introduces an escape sequence, so each must be percent-encoded to survive the
// round trip. Every other byte, spaces included, is passed through untouched.
func escapeSQLiteURIPath(path string) string {
	var escaped strings.Builder
	for index := 0; index < len(path); index++ {
		switch character := path[index]; character {
		case '%':
			escaped.WriteString("%25")
		case '?':
			escaped.WriteString("%3f")
		case '#':
			escaped.WriteString("%23")
		default:
			escaped.WriteByte(character)
		}
	}
	return escaped.String()
}

// isDSNForm reports whether path is a DSN the caller expressed directly rather
// than an ordinary filesystem path. ":memory:" names no file at all, and a
// file: URI names its file through SQLite's own URI resolution, so neither can
// be taken apart with filepath or handed to directory creation.
func isDSNForm(path string) bool {
	return path == ":memory:" || strings.HasPrefix(path, "file:")
}

// sqliteDSN turns a database path into a driver DSN carrying connectionPragmas.
//
// The driver splits a DSN at the first '?' and, unless the DSN is prefixed
// "file:", uses everything before that as a literal filename. Appending a query
// directly to a path would therefore silently truncate any path containing '?'.
// Filesystem paths are converted to a percent-encoded file: URI instead, so
// SQLite parses the query itself and decodes the path back to its exact bytes.
// A caller that already supplied ":memory:" or its own file: URI keeps that
// form, with the pragmas merged into whatever query it carries.
func sqliteDSN(path string) (string, error) {
	if path == "" {
		return "", errors.New("sqlite: empty database path")
	}

	if isDSNForm(path) {
		separator := "?"
		if strings.ContainsRune(path, '?') {
			separator = "&"
		}
		return path + separator + connectionPragmas, nil
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	uriPath := filepath.ToSlash(absolute)
	if !strings.HasPrefix(uriPath, "/") {
		// Windows volume-rooted paths such as C:/gyst/lifeops.db.
		uriPath = "/" + uriPath
	}
	return "file://" + escapeSQLiteURIPath(uriPath) + "?" + connectionPragmas, nil
}

func Open(path string) (*sql.DB, error) {
	dsn, err := sqliteDSN(path)
	if err != nil {
		return nil, err
	}

	// Only an ordinary filesystem path has a parent directory to create.
	// Running filepath over a DSN form would treat the scheme as a path
	// segment and create a literal "file:" directory, so those are left alone;
	// a file: URI's directory must already exist, because SQLite will open a
	// database file but never create the directory holding it.
	if !isDSNForm(path) {
		if dir := filepath.Dir(path); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
	}

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// journal_mode is a persistent property of the database file rather than of
	// the connection, so setting it once here is enough: every connection the
	// pool opens afterwards reads WAL mode back out of the file.
	if _, err := database.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

func Migrate(database *sql.DB, migrationsDir string) error {
	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	// Run every migration on a single dedicated connection with foreign-key
	// enforcement disabled. This is the SQLite-recommended procedure for
	// schema changes that rebuild a table (DROP/recreate): with enforcement on,
	// dropping a parent table would cascade-delete child rows. The PRAGMA must
	// be toggled outside a transaction, so it is set on the connection before
	// each migration's transaction begins and restored when Migrate returns.
	ctx := context.Background()
	conn, err := database.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")

	for _, file := range files {
		var applied int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE filename = ?`, file).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}

		body, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return err
		}

		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, file); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func SeedIfEmpty(database *sql.DB, seedPath string) error {
	var records int
	if err := database.QueryRow(`SELECT
		(SELECT COUNT(*) FROM tasks) +
		(SELECT COUNT(*) FROM projects) +
		(SELECT COUNT(*) FROM goals)
	`).Scan(&records); err != nil {
		return err
	}
	if records > 0 {
		return nil
	}

	body, err := os.ReadFile(seedPath)
	if err != nil {
		return err
	}

	_, err = database.Exec(string(body))
	return err
}
