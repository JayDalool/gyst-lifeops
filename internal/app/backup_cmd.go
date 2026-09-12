package app

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"lifeops-tracker/internal/db"
)

// RunBackup implements `lifeops backup`. It lives on the server binary rather
// than in its own command because that is the binary the Docker image ships:
// a separate tool would not exist inside the running container, which is
// exactly where the database needing a backup is.
//
//	lifeops backup
//	lifeops backup -db data/lifeops.db -out backups/before-migration.db
//
// Scheduling, rotation and off-site copying are deliberately absent. They are
// one cron line around this command, and a backup tool that runs unattended is
// a daemon with its own failure modes.
func RunBackup(args []string) error {
	flags := flag.NewFlagSet("backup", flag.ContinueOnError)
	dbPath := flags.String("db", "data/lifeops.db", "SQLite database path to back up")
	out := flags.String("out", "", "destination file (default: backups/<name>-<UTC timestamp>.db)")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: lifeops backup [options]\n\n"+
			"Writes a consistent copy of the database using VACUUM INTO.\n"+
			"The source is opened read-only and is never modified; the server\n"+
			"may keep running.\n\nOptions:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}

	destination := *out
	if destination == "" {
		destination = db.DefaultBackupPath(*dbPath, time.Now())
	}

	result, err := db.Backup(*dbPath, destination)
	if err != nil {
		return err
	}

	migrations := "unknown (no schema_migrations table)"
	if result.Migrations >= 0 {
		migrations = fmt.Sprintf("%d", result.Migrations)
	}
	fmt.Printf("backup written: %s\n", result.Destination)
	fmt.Printf("  source:      %s\n", result.Source)
	fmt.Printf("  size:        %s\n", humanBytes(result.Bytes))
	fmt.Printf("  tables:      %d\n", result.Tables)
	fmt.Printf("  migrations:  %s\n", migrations)
	fmt.Printf("  took:        %s\n", result.Elapsed.Round(time.Millisecond))
	fmt.Printf("  verified:    integrity_check ok\n")
	return nil
}

func humanBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value, exponent := float64(bytes), 0
	for value >= unit && exponent < 3 {
		value /= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", value, "KMG"[exponent-1])
}

// backupRequested reports whether the process was invoked as `lifeops backup`.
// Only the first argument counts, so a future -flag named backup cannot be
// mistaken for the subcommand.
func backupRequested(args []string) bool {
	return len(args) > 1 && args[1] == "backup"
}

func runBackupAndExit() {
	err := RunBackup(os.Args[2:])
	switch {
	case err == nil:
		return
	case errors.Is(err, flag.ErrHelp):
		// -h printed the usage text; that is the command working, not failing.
		return
	default:
		// The underlying error is printed whole: a SQLite message names the
		// path and the reason, and summarising it would lose both.
		fmt.Fprintf(os.Stderr, "backup failed: %v\n", err)
		os.Exit(1)
	}
}
