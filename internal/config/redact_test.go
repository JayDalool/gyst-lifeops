package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// Fictional values only.
var fictionalSecrets = []string{"hevy-secret-fictional", "anth-secret-fictional", "oai-secret-fictional",
	"sync-secret-fictional", "hash-secret-fictional", "vpub-secret-fictional", "vpriv-secret-fictional",
	"gh-secret-fictional", "jira-token-secret-fictional", "slack-secret-fictional"}

func fullConfig(t *testing.T) Config {
	t.Helper()
	s := fictionalSecrets
	c, err := Load(nil, env("HEVY_API_KEY", s[0], "HEALTH_AI_PROVIDER", "anthropic", "ANTHROPIC_API_KEY", s[1],
		"OPENAI_API_KEY", s[2], "GYST_HEALTH_SYNC_TOKEN", s[3], "GYST_PASSWORD_HASH", s[4],
		"VAPID_PUBLIC_KEY", s[5], "VAPID_PRIVATE_KEY", s[6], "VAPID_SUBSCRIBER", "mailto:a@example.test",
		"GITHUB_TOKEN", s[7], "JIRA_EMAIL", "j@example.test", "JIRA_API_TOKEN", s[8],
		"JIRA_BASE_URL", "https://example.test", "SLACK_BOT_TOKEN", s[9]), noFiles)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// A stray print, log or JSON of the struct must not dump the secrets.
func TestPrintingTheConfigNeverShowsSecrets(t *testing.T) {
	c := fullConfig(t)
	var logged bytes.Buffer
	slog.New(slog.NewTextHandler(&logged, nil)).Info("x", "cfg", c, "push", c.Push, "providers", c.Providers, "ai", c.HealthAI)
	outputs := []string{
		fmt.Sprintf("%v", c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c), fmt.Sprint(c),
		fmt.Sprintf("%+v", c.Push), fmt.Sprintf("%#v", c.Providers), fmt.Sprintf("%+v", c.HealthAI),
		fmt.Sprintf("%+v", &c), logged.String(),
	}
	for _, out := range outputs {
		for _, secret := range fictionalSecrets {
			if strings.Contains(out, secret) {
				t.Fatalf("a printed form contains a secret value (%s)", secret)
			}
		}
	}
	if !strings.Contains(outputs[0], "CONFIGURED") {
		t.Error("the redacted form should still say which settings are configured")
	}
}

func TestJiraSyncLineNamesAnIncompleteSetup(t *testing.T) {
	cases := []struct{ name, url, want string }{
		{"no credentials", "", "JIRA sync: off (not configured)"},
		{"missing base url", "", "JIRA sync: INCOMPLETE, sync off"},
		{"http base url", "http://example.test", "JIRA sync: INCOMPLETE, sync off"},
		{"valid", "https://example.test", "JIRA sync: on"},
	}
	for _, tc := range cases {
		pairs := []string{"JIRA_BASE_URL", tc.url}
		if tc.name != "no credentials" {
			pairs = append(pairs, "JIRA_EMAIL", "j@example.test", "JIRA_API_TOKEN", "jira-token-secret-fictional")
		}
		c, err := Load(nil, env(pairs...), noFiles)
		if err != nil {
			t.Fatalf("%s: a missing or invalid JIRA_BASE_URL must not stop startup: %v", tc.name, err)
		}
		if got := strings.Join(c.Summary(), "\n"); !strings.Contains(got, tc.want) {
			t.Errorf("%s: summary lacks %q", tc.name, tc.want)
		}
	}
}

func TestLoadKeepsPositionalArguments(t *testing.T) {
	c, err := Load([]string{"-db", "x.db", "extra"}, env(), noFiles)
	if err != nil || len(c.Args) != 1 || c.Args[0] != "extra" {
		t.Fatalf("args = %v, err = %v", c.Args, err)
	}
}
