package app

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lifeops-tracker/internal/config"
	"lifeops-tracker/internal/db"
)

func testEnv(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

func noFile(string) ([]byte, error) { return nil, os.ErrNotExist }

func TestConfigRequestedOnlyMatchesTheFirstArgument(t *testing.T) {
	for args, want := range map[string]bool{"lifeops config": true, "lifeops config -addr :1": true, "lifeops": false, "lifeops -db config": false} {
		if got := configRequested(strings.Fields(args)); got != want {
			t.Errorf("configRequested(%q) = %v, want %v", args, got, want)
		}
	}
}

// A validation error is returned (the process exits 1) and no database file is
// created: the command never opens the database.
func TestRunConfigFailsWithoutOpeningTheDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "never.db")
	var out bytes.Buffer
	err := RunConfig([]string{"-db", dbPath}, testEnv("VAPID_PUBLIC_KEY", "pub-fictional"), noFile, &out)
	if err == nil || !strings.Contains(err.Error(), "VAPID_PRIVATE_KEY") {
		t.Fatalf("err = %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("lifeops config created files: %v", entries)
	}
}

func TestRunConfigPrintsSummaryAndOK(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := RunConfig([]string{"-db", filepath.Join(dir, "never.db")}, testEnv("HEVY_API_KEY", "hevy-secret-fictional"), noFile, &out); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "HEVY_API_KEY: CONFIGURED") || !strings.HasSuffix(strings.TrimSpace(text), "config: OK") {
		t.Fatalf("output:\n%s", text)
	}
	if strings.Contains(text, "hevy-secret-fictional") {
		t.Fatal("secret printed")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("lifeops config created files: %v", entries)
	}
}

// Negative log test: with fictional secrets set, nothing New or the startup
// summary logs contains a secret value.
func TestStartupLogsNeverContainSecrets(t *testing.T) {
	secrets := []string{"hevy-secret-fictional", "anth-secret-fictional", "oai-secret-fictional", "sync-secret-fictional",
		"hash-secret-fictional", "vpub-secret-fictional", "vpriv-secret-fictional", "vsub-secret@example.test",
		"gh-secret-fictional", "jira-secret@example.test", "jira-token-secret-fictional", "slack-secret-fictional"}
	getenv := testEnv("HEVY_API_KEY", secrets[0], "HEALTH_AI_PROVIDER", "anthropic", "ANTHROPIC_API_KEY", secrets[1], "OPENAI_API_KEY", secrets[2],
		"GYST_HEALTH_SYNC_TOKEN", secrets[3], "GYST_PASSWORD_HASH", secrets[4], "VAPID_PUBLIC_KEY", secrets[5], "VAPID_PRIVATE_KEY", secrets[6],
		"VAPID_SUBSCRIBER", "mailto:"+secrets[7], "GITHUB_TOKEN", secrets[8], "JIRA_EMAIL", secrets[9], "JIRA_API_TOKEN", secrets[10],
		"SLACK_BOT_TOKEN", secrets[11])
	dir := t.TempDir()
	cfg, err := config.Load([]string{"-db", filepath.Join(dir, "log.db"), "-migrations", "../../migrations",
		"-templates", "../../web/templates", "-static", "../../web/static", "-seed=false"}, getenv, noFile)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(previous) })
	for _, line := range cfg.Summary() {
		log.Printf("config: %s", line)
	}
	a, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.service.Push.PublicKey != secrets[5] || a.service.ProviderCreds.GitHubToken != secrets[8] {
		t.Fatal("credentials were not injected into the service")
	}
	if !strings.Contains(buf.String(), "config: HEVY_API_KEY: CONFIGURED") {
		t.Fatalf("no summary logged: %q", buf.String())
	}
	for _, s := range secrets {
		if strings.Contains(buf.String(), s) {
			t.Fatalf("log contains a secret value: %s", s)
		}
	}
}

// lifeops backup -db follows DB_PATH; an explicit -db still wins.
func TestRunBackupDefaultDBFollowsDBPath(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "from-env.db")
	database, err := db.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`CREATE TABLE note (id INTEGER PRIMARY KEY);`); err != nil {
		t.Fatal(err)
	}
	database.Close()

	destination := filepath.Join(dir, "out", "b.db")
	if err := runBackup([]string{"-out", destination}, testEnv("DB_PATH", source)); err != nil {
		t.Fatalf("runBackup: %v", err)
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	// Without DB_PATH the default is data/lifeops.db, which does not exist here.
	if err := runBackup([]string{"-out", filepath.Join(dir, "out", "c.db")}, testEnv()); err == nil {
		t.Fatal("expected failure on the missing default database")
	}
}
