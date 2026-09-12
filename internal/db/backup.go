package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Backups are taken with VACUUM INTO rather than by copying the file.
//
// Copying data/lifeops.db while the server runs can produce a torn backup: the
// database is in WAL mode, so committed transactions live in the -wal file
// until a checkpoint moves them. The main file on its own may be missing recent
// commits, and a copy made mid-checkpoint may be inconsistent with both.
//
// VACUUM INTO takes its own read transaction and writes a single, complete,
// already-compacted database file. Writers carry on during it: measured against
// this driver, a backup taken while another connection inserted continuously
// succeeded and produced a copy consistent with the snapshot it started from.
// The copy comes out in journal_mode=delete - one self-contained file with no
// -wal companion, which is what you want for something you are about to move
// elsewhere.

// BackupResult describes a completed backup. Everything in it is measured from
// the finished file, not predicted from the source, so it can be printed as
// evidence that the backup is real.
type BackupResult struct {
	Source      string
	Destination string
	Bytes       int64
	Elapsed     time.Duration

	// Tables counts tables in the copy. Migrations counts rows in
	// schema_migrations, or is -1 when that table is absent - a database that
	// predates the migration runner is unusual but not corrupt, so it is
	// reported rather than rejected.
	Tables     int64
	Migrations int64
}

// DefaultBackupPath names a backup after the source and the moment it was
// taken, in UTC with a sortable layout so `ls` orders them correctly wherever
// the machine thinks it is.
//
// It sits in a backups/ directory beside the database rather than beside the
// process. In the container only ./data is a mounted volume, so a backup
// written relative to the working directory would live inside the container
// and disappear with it - which is the one thing a backup must not do.
func DefaultBackupPath(sourcePath string, now time.Time) string {
	name := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	if name == "" {
		name = "lifeops"
	}
	return filepath.Join(filepath.Dir(sourcePath), "backups", fmt.Sprintf("%s-%s.db", name, now.UTC().Format("20060102T150405Z")))
}

// readOnlyDSN opens a database for reading and nothing else. mode=ro makes the
// guarantee structural instead of relying on the caller never issuing a write:
// SQLite refuses writes on the connection, and it will not create the file if
// the path is wrong, so a mistyped source cannot leave an empty database behind.
func readOnlyDSN(path string) (string, error) {
	if isDSNForm(path) {
		// A caller-supplied DSN may already carry its own mode=, and appending
		// a second one leaves which wins undefined. A backup is not the place
		// to find out, and it has no use for an exotic DSN in any case.
		return "", fmt.Errorf("sqlite: backup needs a filesystem path, not a DSN (%q)", path)
	}
	dsn, err := sqliteDSN(path)
	if err != nil {
		return "", err
	}
	return dsn + "&mode=ro", nil
}

// Backup writes a consistent copy of sourcePath to destinationPath.
//
// The source is opened read-only and is never modified: no checkpoint is
// forced, no pragma is set, and the journal mode is left exactly as it was.
// The server may keep running throughout.
func Backup(sourcePath, destinationPath string) (BackupResult, error) {
	var result BackupResult

	if strings.TrimSpace(destinationPath) == "" {
		return result, errors.New("sqlite: backup destination must not be empty")
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return result, fmt.Errorf("sqlite: cannot read the database to back up: %w", err)
	}
	if sourceInfo.IsDir() {
		return result, fmt.Errorf("sqlite: backup source %q is a directory", sourcePath)
	}

	// SQLite is given an absolute destination: it resolves the INTO path
	// relative to the process, and being explicit keeps the path that is
	// reported identical to the path that was written.
	destination, err := filepath.Abs(destinationPath)
	if err != nil {
		return result, err
	}
	source, err := filepath.Abs(sourcePath)
	if err != nil {
		return result, err
	}
	if destination == source {
		return result, fmt.Errorf("sqlite: backup destination %q is the database itself", destinationPath)
	}

	// Checked before the vacuum purely for the error message. SQLite refuses an
	// existing output file itself, and that refusal - not this check - is what
	// actually protects an earlier backup from being overwritten.
	if _, err := os.Stat(destination); err == nil {
		return result, fmt.Errorf("sqlite: backup destination %q already exists; refusing to overwrite it", destinationPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, err
	}

	// VACUUM INTO will not create the directory holding its output; it fails
	// with "unable to open database file" and leaves nothing behind.
	if dir := filepath.Dir(destination); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return result, err
		}
	}

	dsn, err := readOnlyDSN(sourcePath)
	if err != nil {
		return result, err
	}
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return result, err
	}
	defer database.Close()

	started := time.Now()
	// A bound parameter, not string concatenation: the INTO clause takes an
	// expression, so the path never has to be quoted or escaped by hand.
	if _, err := database.Exec(`VACUUM INTO ?`, destination); err != nil {
		return result, fmt.Errorf("sqlite: VACUUM INTO %q: %w", destinationPath, err)
	}
	result.Elapsed = time.Since(started)

	info, err := os.Stat(destination)
	if err != nil {
		return result, fmt.Errorf("sqlite: backup reported success but %q is not there: %w", destinationPath, err)
	}

	result.Source = source
	result.Destination = destination
	result.Bytes = info.Size()

	if err := verifyBackup(destination, &result); err != nil {
		// A backup that does not verify is worse than no backup, because it
		// invites the confidence of having one. Remove it - this function
		// created the file seconds ago, so nothing else can own it - and say
		// why, leaving the path free for an immediate retry.
		if removeErr := os.Remove(destination); removeErr != nil {
			return result, fmt.Errorf("%w (and the unusable copy at %q could not be removed: %v)", err, destinationPath, removeErr)
		}
		return result, fmt.Errorf("%w (the unusable copy was removed)", err)
	}

	return result, nil
}

// verifyBackup proves the copy is a readable database before the backup is
// reported as done. Read-only again, so verification cannot leave -wal or -shm
// files sitting next to a backup that is about to be moved somewhere else.
func verifyBackup(destination string, result *BackupResult) error {
	dsn, err := readOnlyDSN(destination)
	if err != nil {
		return err
	}
	copyDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("sqlite: cannot reopen the backup: %w", err)
	}
	defer copyDB.Close()

	rows, err := copyDB.Query(`PRAGMA integrity_check`)
	if err != nil {
		return fmt.Errorf("sqlite: integrity_check on the backup failed: %w", err)
	}
	var problems []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			rows.Close()
			return err
		}
		if line != "ok" {
			problems = append(problems, line)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(problems) > 0 {
		return fmt.Errorf("sqlite: the backup failed integrity_check: %s", strings.Join(problems, "; "))
	}

	if err := copyDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table'`).Scan(&result.Tables); err != nil {
		return fmt.Errorf("sqlite: cannot read the backup's schema: %w", err)
	}

	result.Migrations = -1
	var present int
	if err := copyDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&present); err != nil {
		return err
	}
	if present > 0 {
		if err := copyDB.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&result.Migrations); err != nil {
			return fmt.Errorf("sqlite: cannot count applied migrations in the backup: %w", err)
		}
	}

	return nil
}
