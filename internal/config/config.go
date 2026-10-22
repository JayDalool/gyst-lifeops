// Package config is the only place that reads command-line flags and the
// process environment (GYST-VN-072). Everything else receives a Config.
//
// Load is pure: the environment and the file reader are injected, so tests
// never touch the process environment. Secrets are never printed; Summary
// shows them only as CONFIGURED or NOT CONFIGURED, and no error produced here
// contains a secret value or the content of a secret file.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"lifeops-tracker/internal/clock"
)

const (
	DefaultAddr   = ":8080"
	DefaultDBPath = "data/lifeops.db"
)

// HealthAI selects the Health Coach provider (HEALTH_AI_PROVIDER,
// ANTHROPIC_API_KEY, OPENAI_API_KEY, HEALTH_AI_MODEL). Keys are never rendered
// or logged.
type HealthAI struct {
	Provider     string
	AnthropicKey string
	OpenAIKey    string
	Model        string
}

// Push is the VAPID trio for web push. All three set, or all three empty.
type Push struct{ PublicKey, PrivateKey, Subscriber string }

// Configured reports whether web push can send.
func (p Push) Configured() bool {
	return p.PublicKey != "" && p.PrivateKey != "" && p.Subscriber != ""
}

// Providers are the credentials for external-link sync.
type Providers struct {
	GitHubToken string
	JiraEmail   string
	JiraToken   string
	JiraBaseURL string
	SlackToken  string
}

type Config struct {
	Addr          string
	DBPath        string
	MigrationsDir string
	SeedPath      string
	TemplateDir   string
	StaticDir     string
	LoadSeed      bool
	// GeneratePasswordHash is the legacy -passwordhash flag.
	GeneratePasswordHash bool

	HevyAPIKey string
	// HealthAI selects the Health Coach provider (Issue #64).
	HealthAI  HealthAI
	Push      Push
	Providers Providers

	// AllowPublicFeeds enables the unauthenticated plain .ics routes. Default
	// false (production-safe); tokenized feeds are always available.
	AllowPublicFeeds bool
	HealthSyncToken  string
	// PasswordHash authenticates the web UI. Empty means nobody can sign in.
	PasswordHash string
	// AuthDisabled bypasses session enforcement entirely. Emergency use only.
	AuthDisabled bool
	// Timezone is the application timezone (GYST_TIMEZONE). Empty means the
	// clock package default. Every local calendar day is derived from it.
	Timezone string
	// Logger receives request, auth and provider lines (GYST-VN-015). Nil
	// means slog.Default().
	Logger *slog.Logger
}

// OS returns the real environment reader and file reader for production use.
func OS() (getenv func(string) string, readFile func(string) ([]byte, error)) {
	return os.Getenv, os.ReadFile
}

// DBPathFromEnv is the default database path: DB_PATH when set, else the
// built-in default. `lifeops backup -db` follows it.
func DBPathFromEnv(getenv func(string) string) string {
	if v := strings.TrimSpace(getenv("DB_PATH")); v != "" {
		return v
	}
	return DefaultDBPath
}

type loader struct {
	getenv   func(string) string
	readFile func(string) ([]byte, error)
	errs     []error
}

func (l *loader) fail(format string, args ...any) {
	l.errs = append(l.errs, fmt.Errorf(format, args...))
}

// secret reads NAME or NAME_FILE. Both set, or an unreadable or empty file, is
// an error naming the variable; neither the value nor the file content is ever
// put in an error. One trailing newline is trimmed from file content.
func (l *loader) secret(name string) string {
	value := l.getenv(name)
	fileVar := name + "_FILE"
	path := strings.TrimSpace(l.getenv(fileVar))
	if path == "" {
		return value
	}
	if value != "" {
		l.fail("%s and %s are both set; set only one", name, fileVar)
		return ""
	}
	data, err := l.readFile(path)
	if err != nil {
		l.fail("%s: the file cannot be read", fileVar)
		return ""
	}
	text := string(data)
	text = strings.TrimSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\r")
	if text == "" {
		l.fail("%s: the file is empty", fileVar)
		return ""
	}
	return text
}

func (l *loader) trimmedSecret(name string) string {
	return strings.TrimSpace(l.secret(name))
}

// Load parses the server flags in args (without the program name) and reads
// the environment. Flag defaults come from the environment where it supplies
// one (PORT for -addr, DB_PATH for -db); an explicit flag wins. Every
// validation failure is returned together, each naming its variable.
func Load(args []string, getenv func(string) string, readFile func(string) ([]byte, error)) (Config, error) {
	l := &loader{getenv: getenv, readFile: readFile}

	addrDefault := DefaultAddr
	if port := strings.TrimSpace(getenv("PORT")); port != "" {
		if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
			l.fail("PORT must be an integer from 1 to 65535")
		} else {
			addrDefault = ":" + strconv.Itoa(n)
		}
	}

	var c Config
	fs := flag.NewFlagSet("lifeops", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&c.Addr, "addr", addrDefault, "HTTP listen address")
	fs.BoolVar(&c.GeneratePasswordHash, "passwordhash", false, "generate a bcrypt password hash")
	fs.StringVar(&c.DBPath, "db", DBPathFromEnv(getenv), "SQLite database path")
	fs.StringVar(&c.MigrationsDir, "migrations", "migrations", "migrations directory")
	fs.StringVar(&c.SeedPath, "seed-file", "seeds/example.sql", "example seed data file")
	fs.StringVar(&c.TemplateDir, "templates", "web/templates", "HTML templates directory")
	fs.StringVar(&c.StaticDir, "static", "web/static", "static assets directory")
	fs.BoolVar(&c.LoadSeed, "seed", true, "load example seed data when the database is empty")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// ParseBool treats an unset or invalid value as false.
	c.AllowPublicFeeds, _ = strconv.ParseBool(getenv("ALLOW_PUBLIC_CALENDAR_FEEDS"))
	c.AuthDisabled = getenv("GYST_AUTH_DISABLED") == "1"
	c.Timezone = getenv("GYST_TIMEZONE")

	c.HevyAPIKey = l.secret("HEVY_API_KEY")
	c.HealthAI = HealthAI{
		Provider:     getenv("HEALTH_AI_PROVIDER"),
		AnthropicKey: l.secret("ANTHROPIC_API_KEY"),
		OpenAIKey:    l.secret("OPENAI_API_KEY"),
		Model:        getenv("HEALTH_AI_MODEL"),
	}
	c.HealthSyncToken = l.secret("GYST_HEALTH_SYNC_TOKEN")
	c.PasswordHash = l.secret("GYST_PASSWORD_HASH")
	c.Push = Push{
		PublicKey:  l.trimmedSecret("VAPID_PUBLIC_KEY"),
		PrivateKey: l.trimmedSecret("VAPID_PRIVATE_KEY"),
		Subscriber: strings.TrimSpace(getenv("VAPID_SUBSCRIBER")),
	}
	c.Providers = Providers{
		GitHubToken: l.trimmedSecret("GITHUB_TOKEN"),
		JiraEmail:   strings.TrimSpace(getenv("JIRA_EMAIL")),
		JiraToken:   l.trimmedSecret("JIRA_API_TOKEN"),
		JiraBaseURL: strings.TrimSpace(getenv("JIRA_BASE_URL")),
		SlackToken:  l.trimmedSecret("SLACK_BOT_TOKEN"),
	}

	l.validate(c)
	if len(l.errs) > 0 {
		return Config{}, errors.Join(l.errs...)
	}
	return c, nil
}

func (l *loader) validate(c Config) {
	// The timezone is checked here and again by app.New before the database
	// opens, so an invalid zone can never reach a migration.
	if _, err := clock.Load(c.Timezone); err != nil {
		l.errs = append(l.errs, err)
	}
	l.pair([]string{"VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY", "VAPID_SUBSCRIBER"},
		[]bool{c.Push.PublicKey != "", c.Push.PrivateKey != "", c.Push.Subscriber != ""})
	l.pair([]string{"JIRA_EMAIL", "JIRA_API_TOKEN"},
		[]bool{c.Providers.JiraEmail != "", c.Providers.JiraToken != ""})
}

// pair requires the named variables to be all set or all empty.
func (l *loader) pair(names []string, set []bool) {
	var have, missing []string
	for i, name := range names {
		if set[i] {
			have = append(have, name)
		} else {
			missing = append(missing, name)
		}
	}
	if len(have) > 0 && len(missing) > 0 {
		l.fail("%s must be all set or all empty: %s set, %s missing",
			strings.Join(names, ", "), strings.Join(have, ", "), strings.Join(missing, ", "))
	}
}

func state(set bool) string {
	if set {
		return "CONFIGURED"
	}
	return "NOT CONFIGURED"
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

// Summary is the startup report: secrets as CONFIGURED / NOT CONFIGURED only,
// everything else as its value. It never contains a secret.
func (c Config) Summary() []string {
	zone := c.Timezone
	if loc, err := clock.Load(c.Timezone); err == nil {
		zone = loc.String()
	}
	aiProvider := strings.TrimSpace(c.HealthAI.Provider)
	if aiProvider == "" {
		aiProvider = "none"
	}
	aiModel := c.HealthAI.Model
	if aiModel == "" {
		aiModel = "default"
	}
	return []string{
		"GYST_TIMEZONE: " + zone,
		"addr: " + c.Addr,
		"db: " + c.DBPath,
		"migrations: " + c.MigrationsDir,
		"templates: " + c.TemplateDir,
		"static: " + c.StaticDir,
		"seed on empty database: " + onOff(c.LoadSeed),
		"ALLOW_PUBLIC_CALENDAR_FEEDS: " + onOff(c.AllowPublicFeeds),
		"GYST_AUTH_DISABLED: " + onOff(c.AuthDisabled),
		"HEALTH_AI_PROVIDER: " + aiProvider,
		"HEALTH_AI_MODEL: " + aiModel,
		"GYST_PASSWORD_HASH: " + state(c.PasswordHash != ""),
		"GYST_HEALTH_SYNC_TOKEN: " + state(c.HealthSyncToken != ""),
		"HEVY_API_KEY: " + state(c.HevyAPIKey != ""),
		"ANTHROPIC_API_KEY: " + state(c.HealthAI.AnthropicKey != ""),
		"OPENAI_API_KEY: " + state(c.HealthAI.OpenAIKey != ""),
		"VAPID (web push): " + state(c.Push.Configured()),
		"GITHUB_TOKEN: " + state(c.Providers.GitHubToken != ""),
		"JIRA (email, token): " + state(c.Providers.JiraEmail != "" && c.Providers.JiraToken != ""),
		"JIRA_BASE_URL: " + state(c.Providers.JiraBaseURL != ""),
		"SLACK_BOT_TOKEN: " + state(c.Providers.SlackToken != ""),
	}
}
