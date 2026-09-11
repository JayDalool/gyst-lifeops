package app

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/handlers"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

type Config struct {
	DBPath        string
	MigrationsDir string
	SeedPath      string
	TemplateDir   string
	StaticDir     string
	LoadSeed      bool
	// AllowPublicFeeds enables the unauthenticated plain .ics routes. Default
	// false (production-safe); tokenized feeds are always available.
	AllowPublicFeeds bool
	HealthSyncToken  string
}

type App struct {
	DB      *sql.DB
	Handler http.Handler
	cancel  context.CancelFunc
}

func (a *App) Close() error {
	if a.cancel != nil {
		a.cancel()
	}
	return a.DB.Close()
}

func New(config Config) (*App, error) {
	database, err := db.Open(config.DBPath)
	if err != nil {
		return nil, err
	}

	if err := db.Migrate(database, config.MigrationsDir); err != nil {
		database.Close()
		return nil, err
	}

	if config.LoadSeed {
		if err := db.SeedIfEmpty(database, config.SeedPath); err != nil {
			database.Close()
			return nil, err
		}
	}

	repo := repository.New(database)
	service := services.New(repo)
	pushContext, cancelPush := context.WithCancel(context.Background())
	go service.RunPushScheduler(pushContext)
	webHandler := handlers.New(service, config.TemplateDir)
	webHandler.AllowPublicFeeds = config.AllowPublicFeeds
	webHandler.HealthSyncToken = config.HealthSyncToken

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(config.StaticDir))))
	mux.HandleFunc("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		http.ServeFile(w, r, filepath.Join(config.StaticDir, "manifest.webmanifest"))
	})
	mux.HandleFunc("/service-worker.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(config.StaticDir, "service-worker.js"))
	})
	// Browsers auto-request /favicon.ico at the root; serve the themed SVG mark
	// there so it resolves (and isn't swallowed by the "/" web handler as a 404).
	faviconPath := filepath.Join(config.StaticDir, "favicon.svg")
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(w, r, faviconPath)
	})
	mux.Handle("/", webHandler)

	return &App{DB: database, Handler: mux, cancel: cancelPush}, nil
}
