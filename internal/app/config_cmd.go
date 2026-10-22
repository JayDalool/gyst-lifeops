package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"lifeops-tracker/internal/config"
)

// configRequested reports whether the process was invoked as `lifeops config`.
// Only the first argument counts.
func configRequested(args []string) bool {
	return len(args) > 1 && args[1] == "config"
}

// RunConfig implements `lifeops config`: it loads and validates the
// configuration exactly as the server would, prints the startup summary and the
// result, and returns an error when validation fails. It opens no database and
// creates no file, so Jay can check a new image before restarting production.
// Secrets appear only as CONFIGURED / NOT CONFIGURED, and errors name
// variables, never values.
func RunConfig(args []string, getenv func(string) string, readFile func(string) ([]byte, error), out io.Writer) error {
	cfg, err := config.Load(args, getenv, readFile)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, strings.Join(cfg.Summary(), "\n"))
	fmt.Fprintln(out, "config: OK")
	return nil
}

func runConfigAndExit() {
	getenv, readFile := config.OS()
	err := RunConfig(os.Args[2:], getenv, readFile, os.Stdout)
	switch {
	case err == nil:
		return
	case errors.Is(err, flag.ErrHelp):
		return
	default:
		fmt.Fprintf(os.Stderr, "config: INVALID\n%v\n", err)
		os.Exit(1)
	}
}
