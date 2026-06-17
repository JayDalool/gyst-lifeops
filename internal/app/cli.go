package app

import (
	"flag"
	"log"
	"net/http"
)

func Run() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "data/lifeops.db", "SQLite database path")
	migrationsDir := flag.String("migrations", "migrations", "migrations directory")
	seedPath := flag.String("seed-file", "seeds/example.sql", "example seed data file")
	templateDir := flag.String("templates", "web/templates", "HTML templates directory")
	staticDir := flag.String("static", "web/static", "static assets directory")
	loadSeed := flag.Bool("seed", true, "load example seed data when the database is empty")
	flag.Parse()

	lifeOps, err := New(Config{
		DBPath:        *dbPath,
		MigrationsDir: *migrationsDir,
		SeedPath:      *seedPath,
		TemplateDir:   *templateDir,
		StaticDir:     *staticDir,
		LoadSeed:      *loadSeed,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer lifeOps.DB.Close()

	log.Printf("LifeOps Tracker running at http://localhost%s", *addr)
	if err := http.ListenAndServe(*addr, lifeOps.Handler); err != nil {
		log.Fatal(err)
	}
}
