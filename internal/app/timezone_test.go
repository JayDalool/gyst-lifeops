package app

import (
	"os"
	"path/filepath"
	"testing"
)

func timezoneConfig(t *testing.T, dbPath, timezone string) Config {
	t.Helper()
	return Config{
		DBPath:        dbPath,
		MigrationsDir: "../../migrations",
		TemplateDir:   "../../web/templates",
		StaticDir:     "../../web/static",
		Timezone:      timezone,
	}
}

func TestAppTimezoneDefaultsToWinnipeg(t *testing.T) {
	a, err := New(timezoneConfig(t, filepath.Join(t.TempDir(), "tz.db"), ""))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if got := a.service.Location().String(); got != "America/Winnipeg" {
		t.Fatalf("default application timezone = %s", got)
	}
}

func TestAppTimezoneUsesValidCustomZone(t *testing.T) {
	a, err := New(timezoneConfig(t, filepath.Join(t.TempDir(), "tz.db"), "America/Toronto"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if got := a.service.Location().String(); got != "America/Toronto" {
		t.Fatalf("application timezone = %s", got)
	}
	if schedule, err := a.service.PrayerConfig(); err != nil || schedule.Timezone != "America/Toronto" {
		t.Fatalf("prayer timezone = %q, %v; want the application timezone", schedule.Timezone, err)
	}
}

// An invalid explicit GYST_TIMEZONE stops startup before the database is
// opened: no file is created and nothing is migrated.
func TestAppInvalidTimezoneFailsBeforeDatabaseOpen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "never.db")
	a, err := New(timezoneConfig(t, dbPath, "Mars/Base"))
	if err == nil {
		a.Close()
		t.Fatal("New succeeded with an invalid timezone")
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("database files were created before the timezone check: %v", entries)
	}
}
