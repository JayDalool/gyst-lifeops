package config

import (
	"bytes"
	"encoding/json"
	"errors"
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

type holder struct {
	Name string
	Cfg  Config
	AI   HealthAI
}

type wrapErr struct{ cfg Config }

func (e wrapErr) Error() string { return fmt.Sprintf("load failed: %+v", e.cfg) }

// Covered: fmt verbs %v %+v %#v %s %d %t on a value and a pointer, Sprint,
// json.Marshal of each type and of a holding struct, slog text and JSON
// handlers (directly and inside a struct), and an error wrapped with %w.
// Not covered, and still unsafe: %p, and %w on a value that is not an error
// (fmt never calls Format for them and prints the fields by reflection; go vet
// flags %w but not %p), and reflection that reads fields directly.
func TestConfigNeverPrintsSecrets(t *testing.T) {
	c := fullConfig(t)
	var text, jsonLog bytes.Buffer
	slog.New(slog.NewTextHandler(&text, nil)).Info("x", "cfg", c, "push", c.Push, "providers", c.Providers, "ai", c.HealthAI)
	slog.New(slog.NewJSONHandler(&jsonLog, nil)).Info("x", "cfg", c, "holder", holder{"h", c, c.HealthAI})

	outputs := map[string]string{"slog text": text.String(), "slog json": jsonLog.String()}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%d", "%t"} {
		outputs["Config "+verb] = fmt.Sprintf(verb, c)
		outputs["*Config "+verb] = fmt.Sprintf(verb, &c)
		outputs["Push "+verb] = fmt.Sprintf(verb, c.Push)
		outputs["Providers "+verb] = fmt.Sprintf(verb, c.Providers)
		outputs["HealthAI "+verb] = fmt.Sprintf(verb, c.HealthAI)
		outputs["holder "+verb] = fmt.Sprintf(verb, holder{"h", c, c.HealthAI})
	}
	outputs["Sprint"] = fmt.Sprint(c, c.Push, c.Providers, c.HealthAI)
	wrapped := fmt.Errorf("start: %w", wrapErr{c})
	outputs["wrapped error"] = wrapped.Error()
	var target wrapErr
	if !errors.As(wrapped, &target) {
		t.Fatal("the wrapped error lost its type")
	}
	for name, value := range map[string]any{"Config": c, "Push": c.Push, "Providers": c.Providers, "HealthAI": c.HealthAI, "holder": holder{"h", c, c.HealthAI}} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("json %s: %v", name, err)
		}
		outputs["json "+name] = string(raw)
	}

	for name, out := range outputs {
		for _, secret := range fictionalSecrets {
			if strings.Contains(out, secret) {
				t.Errorf("%s contains a secret value (%s)", name, secret)
			}
		}
	}
	if !strings.Contains(outputs["Config %v"], "CONFIGURED") {
		t.Error("the redacted form should still say which settings are configured")
	}
}

func TestVisionModelShowsInTheSummaryButNoKey(t *testing.T) {
	c, err := Load(nil, env("HEALTH_AI_VISION_MODEL", "vision-model-fictional", "OPENAI_API_KEY", "oai-secret-fictional"), noFiles)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(c.Summary(), "\n")
	if !strings.Contains(got, "HEALTH_AI_VISION_MODEL: vision-model-fictional") || strings.Contains(got, "oai-secret-fictional") {
		t.Errorf("summary = %q", got)
	}
	c, _ = Load(nil, env(), noFiles)
	if !strings.Contains(strings.Join(c.Summary(), "\n"), "HEALTH_AI_VISION_MODEL: default") {
		t.Error("unset vision model should read default")
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
