package app

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

func Run() {
	// The subcommand is checked before any flag is defined: `lifeops backup`
	// and `lifeops passwordhash` take their own flag sets, and the server's
	// flags are not valid for them.
	if backupRequested(os.Args) {
		runBackupAndExit()
		return
	}
	if healthcheckRequested(os.Args) {
		runHealthcheckAndExit()
		return
	}
	if passwordHashRequested(os.Args) {
		runPasswordHashAndExit()
		return
	}

	// One JSON line per event on stdout (GYST-VN-015). Setting the default also
	// routes the remaining log.Printf lines through the same handler, so the
	// whole process logs one format. Only the server path is switched; backup
	// and passwordhash keep their plain output.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	addr := flag.String("addr", ":8080", "HTTP listen address")
	passwordHashFlag := flag.Bool("passwordhash", false, "generate a bcrypt password hash")
	dbPath := flag.String("db", "data/lifeops.db", "SQLite database path")
	migrationsDir := flag.String("migrations", "migrations", "migrations directory")
	seedPath := flag.String("seed-file", "seeds/example.sql", "example seed data file")
	templateDir := flag.String("templates", "web/templates", "HTML templates directory")
	staticDir := flag.String("static", "web/static", "static assets directory")
	loadSeed := flag.Bool("seed", true, "load example seed data when the database is empty")
	flag.Parse()

	// Check -passwordhash flag after parsing
	if *passwordHashFlag {
		runPasswordHashAndExit()
		return
	}

	// Plain (unauthenticated) .ics feeds are off unless explicitly enabled, so a
	// production deploy never exposes the schedule publicly. Tokenized feeds work
	// regardless. ParseBool treats an unset/invalid value as false.
	allowPublicFeeds, _ := strconv.ParseBool(os.Getenv("ALLOW_PUBLIC_CALENDAR_FEEDS"))

	lifeOps, err := New(Config{
		DBPath:     *dbPath,
		HevyAPIKey: os.Getenv("HEVY_API_KEY"),
		HealthAI: HealthAIConfig{
			Provider:     os.Getenv("HEALTH_AI_PROVIDER"),
			AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
			OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
			Model:        os.Getenv("HEALTH_AI_MODEL"),
		},
		MigrationsDir:    *migrationsDir,
		SeedPath:         *seedPath,
		TemplateDir:      *templateDir,
		StaticDir:        *staticDir,
		LoadSeed:         *loadSeed,
		AllowPublicFeeds: allowPublicFeeds,
		HealthSyncToken:  os.Getenv("GYST_HEALTH_SYNC_TOKEN"),
		N8NEventToken:    os.Getenv("GYST_N8N_EVENT_TOKEN"),
		N8NWorkflows:     os.Getenv("GYST_N8N_WORKFLOWS"),
		N8NURL:           os.Getenv("GYST_N8N_URL"),
		PasswordHash:     os.Getenv("GYST_PASSWORD_HASH"),
		AuthDisabled:     os.Getenv("GYST_AUTH_DISABLED") == "1",
		Timezone:         os.Getenv("GYST_TIMEZONE"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer lifeOps.Close()

	log.Printf("LifeOps Tracker running at http://localhost%s", *addr)
	if err := http.ListenAndServe(*addr, lifeOps.Handler); err != nil {
		log.Fatal(err)
	}
}
