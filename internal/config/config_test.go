package config

import (
	"errors"
	"flag"
	"os"
	"strings"
	"testing"
)

// env builds an injected getenv. Secret values in this file are fictional.
func env(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

func noFiles(string) ([]byte, error) { return nil, os.ErrNotExist }

func files(m map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		if v, ok := m[p]; ok {
			return []byte(v), nil
		}
		return nil, os.ErrNotExist
	}
}

func load(t *testing.T, args []string, getenv func(string) string) (Config, error) {
	t.Helper()
	return Load(args, getenv, noFiles)
}

func TestDefaultsEqualTheOldFlagDefaults(t *testing.T) {
	c, err := load(t, nil, env())
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":8080" || c.DBPath != "data/lifeops.db" || c.MigrationsDir != "migrations" ||
		c.SeedPath != "seeds/example.sql" || c.TemplateDir != "web/templates" ||
		c.StaticDir != "web/static" || !c.LoadSeed || c.GeneratePasswordHash {
		t.Fatalf("defaults changed: %+v", c)
	}
	if c.AllowPublicFeeds || c.AuthDisabled || c.Timezone != "" {
		t.Fatalf("env defaults changed: %+v", c)
	}
}

func TestEnvSuppliesDefaultsAndFlagsWin(t *testing.T) {
	e := env("PORT", "9090", "DB_PATH", "/app/data/lifeops.db")
	c, err := load(t, nil, e)
	if err != nil || c.Addr != ":9090" || c.DBPath != "/app/data/lifeops.db" {
		t.Fatalf("env defaults: %+v %v", c, err)
	}
	c, err = load(t, []string{"-addr", "127.0.0.1:1", "-db", "x.db", "-seed=false"}, e)
	if err != nil || c.Addr != "127.0.0.1:1" || c.DBPath != "x.db" || c.LoadSeed {
		t.Fatalf("flags must win: %+v %v", c, err)
	}
}

func TestBadPortNamesPORT(t *testing.T) {
	for _, port := range []string{"0", "-1", "65536", "70000", "abc", "80.5"} {
		_, err := load(t, nil, env("PORT", port))
		if err == nil || !strings.Contains(err.Error(), "PORT") {
			t.Errorf("PORT=%q: err = %v", port, err)
		}
	}
	for _, port := range []string{"1", "8080", "65535"} {
		if _, err := load(t, nil, env("PORT", port)); err != nil {
			t.Errorf("PORT=%q: %v", port, err)
		}
	}
}

func TestAuthDisabledOnlyForExactlyOne(t *testing.T) {
	for value, want := range map[string]bool{"1": true, "true": false, "yes": false, "": false, "0": false, " 1": false, "1 ": false} {
		c, err := load(t, nil, env("GYST_AUTH_DISABLED", value))
		if err != nil || c.AuthDisabled != want {
			t.Errorf("GYST_AUTH_DISABLED=%q = %v (%v), want %v", value, c.AuthDisabled, err, want)
		}
	}
}

func TestPublicFeedsUseParseBool(t *testing.T) {
	cases := map[string]bool{
		"1": true, "t": true, "T": true, "TRUE": true, "true": true, "True": true,
		"0": false, "f": false, "F": false, "FALSE": false, "false": false, "False": false,
		"": false, "yes": false, "on": false, " true": false, "tRuE": false,
	}
	for value, want := range cases {
		c, err := load(t, nil, env("ALLOW_PUBLIC_CALENDAR_FEEDS", value))
		if err != nil || c.AllowPublicFeeds != want {
			t.Errorf("ALLOW_PUBLIC_CALENDAR_FEEDS=%q = %v (%v), want %v", value, c.AllowPublicFeeds, err, want)
		}
	}
}

func TestInvalidTimezoneIsAnError(t *testing.T) {
	_, err := load(t, nil, env("GYST_TIMEZONE", "Mars/Base"))
	if err == nil || !strings.Contains(err.Error(), "GYST_TIMEZONE") {
		t.Fatalf("err = %v", err)
	}
}

func TestVapidTrioIsAllOrNothing(t *testing.T) {
	names := []string{"VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY", "VAPID_SUBSCRIBER"}
	vals := []string{"pub-fictional", "priv-fictional", "mailto:a@example.test"}
	if c, err := load(t, nil, env()); err != nil || c.Push.Configured() {
		t.Fatalf("none: %+v %v", c.Push, err)
	}
	all := []string{}
	for i := range names {
		all = append(all, names[i], vals[i])
	}
	if c, err := load(t, nil, env(all...)); err != nil || !c.Push.Configured() || c.Push.PublicKey != "pub-fictional" {
		t.Fatalf("all: %+v %v", c.Push, err)
	}
	// every one- and two-element subset fails with the missing names
	for mask := 1; mask < 7; mask++ {
		var pairs []string
		for i := range names {
			if mask&(1<<i) != 0 {
				pairs = append(pairs, names[i], vals[i])
			}
		}
		_, err := load(t, nil, env(pairs...))
		if err == nil || !strings.Contains(err.Error(), "VAPID") {
			t.Errorf("mask %d: err = %v", mask, err)
			continue
		}
		for _, v := range vals {
			if strings.Contains(err.Error(), v) {
				t.Errorf("mask %d: error shows a value: %v", mask, err)
			}
		}
	}
	// whitespace-only counts as empty (values are trimmed as before)
	if _, err := load(t, nil, env("VAPID_PUBLIC_KEY", "  ", "VAPID_PRIVATE_KEY", " ", "VAPID_SUBSCRIBER", "")); err != nil {
		t.Fatalf("blank trio: %v", err)
	}
}

func TestJiraPairAndSingleTokens(t *testing.T) {
	if _, err := load(t, nil, env("JIRA_EMAIL", "dev@example.test")); err == nil || !strings.Contains(err.Error(), "JIRA_API_TOKEN") {
		t.Fatalf("email only: %v", err)
	}
	if _, err := load(t, nil, env("JIRA_API_TOKEN", "tok-fictional")); err == nil || !strings.Contains(err.Error(), "JIRA_EMAIL") {
		t.Fatalf("token only: %v", err)
	}
	c, err := load(t, nil, env("JIRA_EMAIL", " dev@example.test ", "JIRA_API_TOKEN", " tok-fictional ", "JIRA_BASE_URL", " https://team.atlassian.net ",
		"GITHUB_TOKEN", " gh-fictional\n", "SLACK_BOT_TOKEN", " sl-fictional "))
	if err != nil {
		t.Fatal(err)
	}
	p := c.Providers
	if p.JiraEmail != "dev@example.test" || p.JiraToken != "tok-fictional" || p.JiraBaseURL != "https://team.atlassian.net" ||
		p.GitHubToken != "gh-fictional" || p.SlackToken != "sl-fictional" {
		t.Fatalf("providers: %+v", p)
	}
	// GitHub and Slack alone are fine
	if _, err := load(t, nil, env("GITHUB_TOKEN", "gh-fictional")); err != nil {
		t.Fatal(err)
	}
	if _, err := load(t, nil, env("SLACK_BOT_TOKEN", "sl-fictional")); err != nil {
		t.Fatal(err)
	}
}

func TestUntrimmedSettingsStayUntrimmed(t *testing.T) {
	c, err := load(t, nil, env("HEVY_API_KEY", " k ", "GYST_PASSWORD_HASH", " h ", "GYST_HEALTH_SYNC_TOKEN", " t ", "HEALTH_AI_PROVIDER", " Anthropic ", "HEALTH_AI_MODEL", " m "))
	if err != nil {
		t.Fatal(err)
	}
	if c.HevyAPIKey != " k " || c.PasswordHash != " h " || c.HealthSyncToken != " t " || c.HealthAI.Provider != " Anthropic " || c.HealthAI.Model != " m " {
		t.Fatalf("semantics changed: %+v", c)
	}
}

func TestSecretFiles(t *testing.T) {
	fs := files(map[string]string{
		"/run/secrets/hevy":   "hevy-fictional\n",
		"/run/secrets/empty":  "\n",
		"/run/secrets/github": "gh-fictional\r\n",
		"/run/secrets/two":    "two-fictional\n\n",
	})
	c, err := Load(nil, env("HEVY_API_KEY_FILE", "/run/secrets/hevy", "GITHUB_TOKEN_FILE", "/run/secrets/github"), fs)
	if err != nil || c.HevyAPIKey != "hevy-fictional" || c.Providers.GitHubToken != "gh-fictional" {
		t.Fatalf("read: %+v %v", c, err)
	}
	// only one trailing newline is removed
	c, err = Load(nil, env("HEVY_API_KEY_FILE", "/run/secrets/two"), fs)
	if err != nil || c.HevyAPIKey != "two-fictional\n" {
		t.Fatalf("one newline: %q %v", c.HevyAPIKey, err)
	}

	cases := []struct {
		name string
		e    func(string) string
		want string
	}{
		{"both", env("HEVY_API_KEY", "inline-fictional", "HEVY_API_KEY_FILE", "/run/secrets/hevy"), "HEVY_API_KEY"},
		{"unreadable", env("OPENAI_API_KEY_FILE", "/run/secrets/missing"), "OPENAI_API_KEY_FILE"},
		{"empty", env("GYST_PASSWORD_HASH_FILE", "/run/secrets/empty"), "GYST_PASSWORD_HASH_FILE"},
	}
	for _, tc := range cases {
		_, err := Load(nil, tc.e, fs)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v", tc.name, err)
			continue
		}
		for _, leak := range []string{"inline-fictional", "hevy-fictional", "/run/secrets"} {
			if strings.Contains(err.Error(), leak) {
				t.Errorf("%s: error leaks %q: %v", tc.name, leak, err)
			}
		}
	}
	// every secret honours NAME_FILE
	for _, name := range []string{"HEVY_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GYST_HEALTH_SYNC_TOKEN", "GYST_PASSWORD_HASH",
		"VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY", "GITHUB_TOKEN", "JIRA_API_TOKEN", "SLACK_BOT_TOKEN"} {
		_, err := Load(nil, env(name+"_FILE", "/run/secrets/missing"), fs)
		if err == nil || !strings.Contains(err.Error(), name+"_FILE") {
			t.Errorf("%s_FILE not honoured: %v", name, err)
		}
	}
}

func TestSummaryAndErrorsNeverShowSecrets(t *testing.T) {
	secrets := map[string]string{
		"HEVY_API_KEY": "hevy-secret-fictional", "ANTHROPIC_API_KEY": "anth-secret-fictional", "OPENAI_API_KEY": "oai-secret-fictional",
		"GYST_HEALTH_SYNC_TOKEN": "sync-secret-fictional", "GYST_PASSWORD_HASH": "$2a$10$hash-secret-fictional",
		"VAPID_PUBLIC_KEY": "vpub-secret-fictional", "VAPID_PRIVATE_KEY": "vpriv-secret-fictional", "VAPID_SUBSCRIBER": "mailto:vsub-secret@example.test",
		"GITHUB_TOKEN": "gh-secret-fictional", "JIRA_EMAIL": "jira-secret@example.test", "JIRA_API_TOKEN": "jira-secret-fictional",
		"JIRA_BASE_URL": "https://user:pw-secret-fictional@team.atlassian.net", "SLACK_BOT_TOKEN": "slack-secret-fictional",
	}
	var pairs []string
	for k, v := range secrets {
		pairs = append(pairs, k, v)
	}
	c, err := load(t, nil, env(pairs...))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(c.Summary(), "\n")
	for k, v := range secrets {
		if strings.Contains(text, v) {
			t.Errorf("summary shows %s", k)
		}
	}
	for _, want := range []string{"HEVY_API_KEY: CONFIGURED", "VAPID (web push): CONFIGURED", "SLACK_BOT_TOKEN: CONFIGURED"} {
		if !strings.Contains(text, want) {
			t.Errorf("summary lacks %q:\n%s", want, text)
		}
	}
	empty, _ := load(t, nil, env())
	if t2 := strings.Join(empty.Summary(), "\n"); !strings.Contains(t2, "GITHUB_TOKEN: NOT CONFIGURED") || !strings.Contains(t2, "GYST_TIMEZONE: ") {
		t.Errorf("empty summary:\n%s", t2)
	}

	// half-set pair errors show names only
	pairs = append(pairs, "VAPID_SUBSCRIBER", "")
	_, err = load(t, nil, env(pairs...))
	if err == nil {
		t.Fatal("half-set VAPID should fail")
	}
	for k, v := range secrets {
		if strings.Contains(err.Error(), v) {
			t.Errorf("error shows %s", k)
		}
	}
}

func TestHelpAndBadFlagAreErrors(t *testing.T) {
	if _, err := load(t, []string{"-h"}, env()); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("-h: %v", err)
	}
	if _, err := load(t, []string{"-nope"}, env()); err == nil {
		t.Fatal("unknown flag accepted")
	}
}

func TestDBPathFromEnv(t *testing.T) {
	if DBPathFromEnv(env()) != "data/lifeops.db" || DBPathFromEnv(env("DB_PATH", " /x/y.db ")) != "/x/y.db" {
		t.Fatal("DB_PATH default")
	}
}
