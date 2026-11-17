package app

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"lifeops-tracker/internal/config"
)

func Run() {
	// The subcommand is checked before any flag is defined: `lifeops backup`,
	// `lifeops config` and `lifeops passwordhash` take their own flag sets, and the server's
	// flags are not valid for them.
	if backupRequested(os.Args) {
		runBackupAndExit()
		return
	}
	if healthcheckRequested(os.Args) {
		runHealthcheckAndExit()
		return
	}
	if configRequested(os.Args) {
		runConfigAndExit()
		return
	}
	if passwordHashRequested(os.Args) {
		runPasswordHashAndExit()
		return
	}
	if tzinfoRequested(os.Args) {
		runTzinfoAndExit()
		return
	}

	// One JSON line per event on stdout (GYST-VN-015). Setting the default also
	// routes the remaining log.Printf lines through the same handler, so the
	// whole process logs one format. Only the server path is switched; backup
	// and passwordhash keep their plain output.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	getenv, readFile := config.OS()
	cfg, err := config.Load(os.Args[1:], getenv, readFile)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "usage: lifeops [-addr] [-db] [-migrations] [-seed-file] [-templates] [-static] [-seed]")
			fmt.Fprintln(os.Stderr, "       lifeops config | backup | healthcheck | passwordhash")
			return
		}
		// Named errors only: they never contain a secret value.
		log.Fatalf("configuration error: %v", err)
	}

	// Check -passwordhash flag after parsing
	if cfg.GeneratePasswordHash {
		runPasswordHashAndExit()
		return
	}

	for _, line := range cfg.Summary() {
		log.Printf("config: %s", line)
	}

	lifeOps, err := New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer lifeOps.Close()

	log.Printf("LifeOps Tracker running at http://localhost%s", cfg.Addr)
	// Timeouts bound slow or stalled clients. Photo upload (up to 8 MB over a
	// mobile link) plus one vision call (75 s) is the longest request, so the
	// read and write budgets are sized for it; headers must arrive quickly.
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           lifeOps.Handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      4 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
