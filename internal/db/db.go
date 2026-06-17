package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
	}
	for _, pragma := range pragmas {
		if _, err := database.Exec(pragma); err != nil {
			database.Close()
			return nil, err
		}
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

	for _, file := range files {
		var applied int
		if err := database.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE filename = ?`, file).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}

		body, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return err
		}

		tx, err := database.Begin()
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
