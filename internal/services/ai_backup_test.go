package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/repository"
)

// A VACUUM INTO backup carries ciphertext only: neither the API key nor the
// data key is in it, so a restore without the key file leaves the key unread.
func TestAIBackupHoldsCiphertextOnly(t *testing.T) {
	dir := t.TempDir()
	database, err := db.Open(filepath.Join(dir, "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	svc := New(repository.New(database))
	svc.AIKeyPath = filepath.Join(dir, "ai_settings.key")
	if err := svc.SaveAISettings(AISettingsInput{Provider: AIProviderOpenAI, APIKey: fakeServiceKey}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "backups", "copy.db")
	if _, err := db.Backup(filepath.Join(dir, "lifeops.db"), dest); err != nil {
		t.Fatal(err)
	}
	copyBytes, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	dataKey, err := os.ReadFile(svc.AIKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(copyBytes, []byte("FAKE0000")) || bytes.Contains(copyBytes, dataKey) {
		t.Fatal("the backup holds the key or the data key")
	}
	if _, err := os.Stat(filepath.Join(dir, "backups", "ai_settings.key")); err == nil {
		t.Fatal("the key file was copied into backups")
	}
	if _, ok := svc.AIEffective(); !ok {
		t.Fatal("setup: the saved key should decrypt")
	}
	// After a restore without the key file, the key can't be read.
	if err := os.Remove(svc.AIKeyPath); err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.AIEffective(); ok {
		t.Fatal("key read without the key file")
	}
}
