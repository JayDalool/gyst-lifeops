package app

import (
	"path/filepath"
	"testing"

	"lifeops-tracker/internal/db"
)

func TestBackupRequestedOnlyMatchesTheFirstArgument(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"lifeops", "backup"}, true},
		{[]string{"lifeops", "backup", "-out", "x.db"}, true},
		{[]string{"lifeops"}, false},
		{[]string{"lifeops", "-addr", ":8080"}, false},
		{[]string{"lifeops", "-db", "backup"}, false},
	}
	for _, testCase := range cases {
		if got := backupRequested(testCase.args); got != testCase.want {
			t.Fatalf("backupRequested(%q) = %v, want %v", testCase.args, got, testCase.want)
		}
	}
}

// End-to-end through the flag set, so a wiring mistake between the command and
// db.Backup is caught. -out is always passed: without it the command would
// write into a backups/ directory beside this package's working directory,
// leaving an artefact in the repository.
func TestRunBackupWritesTheRequestedFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "lifeops.db")

	database, err := db.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`CREATE TABLE note (id INTEGER PRIMARY KEY);`); err != nil {
		t.Fatal(err)
	}
	database.Close()

	destination := filepath.Join(dir, "backups", "manual.db")
	if err := RunBackup([]string{"-db", source, "-out", destination}); err != nil {
		t.Fatalf("RunBackup: %v", err)
	}

	reopened, err := db.Open(destination)
	if err != nil {
		t.Fatalf("the backup is not openable: %v", err)
	}
	defer reopened.Close()
	var tables int
	if err := reopened.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='note'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 1 {
		t.Fatalf("the backup is missing the note table")
	}
}

func TestRunBackupReportsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := RunBackup([]string{"-db", filepath.Join(dir, "absent.db"), "-out", filepath.Join(dir, "out.db")}); err == nil {
		t.Fatal("RunBackup reported success for a database that does not exist")
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:          "0 B",
		999:        "999 B",
		1024:       "1.0 KiB",
		1536:       "1.5 KiB",
		1048576:    "1.0 MiB",
		1073741824: "1.0 GiB",
	}
	for bytes, want := range cases {
		if got := humanBytes(bytes); got != want {
			t.Fatalf("humanBytes(%d) = %q, want %q", bytes, got, want)
		}
	}
}
