package app

import (
	"database/sql"
	"net/http"

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
}

type App struct {
	DB      *sql.DB
	Handler http.Handler
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
	webHandler := handlers.New(service, config.TemplateDir)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(config.StaticDir))))
	mux.Handle("/", webHandler)

	return &App{DB: database, Handler: mux}, nil
}
