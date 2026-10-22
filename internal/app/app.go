package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/config"
	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/handlers"
	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/providers/hevy"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// Config is the single configuration struct, built by config.Load. The alias
// keeps existing callers and tests compiling unchanged.
type Config = config.Config

type App struct {
	DB      *sql.DB
	Handler http.Handler
	cancel  context.CancelFunc
	service *services.Service
}

func (a *App) Close() error {
	if a.cancel != nil {
		a.cancel()
	}
	return a.DB.Close()
}

func New(cfg Config) (*App, error) {
	// Resolve the timezone first: an invalid explicit GYST_TIMEZONE must stop
	// startup before the database is opened, created or migrated.
	location, err := clock.Load(cfg.Timezone)
	if err != nil {
		return nil, err
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	if err := db.Migrate(database, cfg.MigrationsDir); err != nil {
		database.Close()
		return nil, err
	}

	if cfg.LoadSeed {
		if err := db.SeedIfEmpty(database, cfg.SeedPath); err != nil {
			database.Close()
			return nil, err
		}
	}

	repo := repository.New(database)
	service := services.New(repo)
	// Set before any goroutine reads the clock.
	service.Loc = location
	service.Push = cfg.Push
	service.ProviderCreds = cfg.Providers
	service.BackupStatusPath = filepath.Join(filepath.Dir(cfg.DBPath), "backups", "last_run.json")
	log.Printf("Application timezone: %s", location)
	// Before any goroutine, handler or listener: a changed GYST_PASSWORD_HASH
	// must end every old session, and a failure here stops startup rather
	// than serve sessions the new password never approved.
	if revoked, changed, err := service.RevokeSessionsOnPasswordChange(cfg.PasswordHash); err != nil {
		database.Close()
		return nil, fmt.Errorf("revoke sessions on password change: %w", err)
	} else if changed {
		log.Printf("SECURITY: GYST_PASSWORD_HASH changed; session.revoked_all, %d sessions", revoked)
		if err := service.RecordAudit(context.Background(), repository.AuditEvent{
			ActorType: services.AuditActorSystem, ActorRef: "system:startup",
			Ingress: services.AuditIngressInternal, Action: services.AuditActionSessionRevokeAll,
			TargetRef: "account:owner", Outcome: services.AuditOutcomeSuccess,
			Detail: "credential-changed",
		}); err != nil {
			log.Printf("audit write failed: session.revoked_all")
		}
	}
	pushContext, cancelPush := context.WithCancel(context.Background())
	// pushContext is cancelled by Close and drives every background loop.
	go service.RunPushScheduler(pushContext)
	// Synchronous, before the loop: it cannot race Close.
	service.SweepSessionsAtStart()
	go service.RunSessionSweeper(pushContext)
	service.PruneAuditAtStart(pushContext)
	go service.RunAuditPruneLoop(pushContext)
	if cfg.HevyAPIKey != "" {
		service.Training = &services.TrainingSync{Provider: hevy.New(cfg.HevyAPIKey)}
		go service.RunTrainingSync(pushContext)
	}
	webHandler := handlers.New(service, cfg.TemplateDir)
	webHandler.AllowPublicFeeds = cfg.AllowPublicFeeds
	webHandler.HealthSyncToken = cfg.HealthSyncToken
	webHandler.Auth = services.AuthConfig{PasswordHash: cfg.PasswordHash}
	webHandler.AuthDisabled = cfg.AuthDisabled
	webHandler.Coach = healthCoachProvider(cfg.HealthAI)
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	webHandler.Logger = logger

	if cfg.AuthDisabled {
		log.Println("=======================================================")
		log.Println("SECURITY: GYST_AUTH_DISABLED is set.")
		log.Println("Every route is served without authentication.")
		log.Println("Unset it as soon as you can sign in again.")
		log.Println("=======================================================")
	} else if cfg.PasswordHash == "" {
		log.Println("SECURITY: GYST_PASSWORD_HASH is not set; nobody can sign in.")
		log.Println("Generate one with: go run ./cmd/passwordhash")
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir))))
	mux.HandleFunc("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "manifest.webmanifest"))
	})
	mux.HandleFunc("/service-worker.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "service-worker.js"))
	})
	// Browsers auto-request /favicon.ico at the root; serve the themed SVG mark
	// there so it resolves (and isn't swallowed by the "/" web handler as a 404).
	faviconPath := filepath.Join(cfg.StaticDir, "favicon.svg")
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(w, r, faviconPath)
	})
	// Liveness probe: unauthenticated by design, exact path only. Other
	// spellings (/healthz/, /HEALTHZ) fall through to the session middleware.
	mux.Handle("/healthz", healthzHandler(database))
	// Everything the mux has not already claimed goes through session
	// enforcement. Static assets, the manifest, the service worker and the
	// favicon are matched above and so stay reachable while signed out.
	mux.Handle("/", handlers.RequestLog(logger, webHandler.Authenticated()))

	return &App{DB: database, Handler: handlers.SecurityHeaders(mux), cancel: cancelPush, service: service}, nil
}

// HealthAIConfig is the Health Coach part of Config.
type HealthAIConfig = config.HealthAI

// healthCoachProvider returns nil (Coach disabled) unless a supported provider
// and its key are both set. Each provider only ever uses its own key.
func healthCoachProvider(c HealthAIConfig) healthcoach.Provider {
	switch strings.ToLower(strings.TrimSpace(c.Provider)) {
	case "anthropic":
		if c.AnthropicKey != "" {
			log.Printf("Health Coach: provider anthropic enabled")
			return healthcoach.NewAnthropic(c.AnthropicKey, c.Model)
		}
		log.Printf("Health Coach: disabled (ANTHROPIC_API_KEY not set)")
	case "openai":
		if c.OpenAIKey != "" {
			log.Printf("Health Coach: provider openai enabled")
			return healthcoach.NewOpenAI(c.OpenAIKey, c.Model)
		}
		log.Printf("Health Coach: disabled (OPENAI_API_KEY not set)")
	case "":
	default:
		log.Printf("Health Coach: disabled (unknown HEALTH_AI_PROVIDER)")
	}
	return nil
}
