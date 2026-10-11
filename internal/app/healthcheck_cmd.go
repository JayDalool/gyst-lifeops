package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	// healthcheckURL is the fixed loopback address of the in-container server.
	healthcheckURL     = "http://127.0.0.1:8080/healthz"
	healthcheckTimeout = 3 * time.Second
)

// healthcheckRequested reports whether the process was invoked as
// `lifeops healthcheck`. Only the first argument counts.
func healthcheckRequested(args []string) bool {
	return len(args) > 1 && args[1] == "healthcheck"
}

// RunHealthcheck GETs /healthz and returns nil only for HTTP 200. It opens no
// database, reads no environment and never prints the response body. Redirects
// are not followed. It exists because the runtime image has no curl or wget.
func RunHealthcheck(args []string) error {
	fs := flag.NewFlagSet("healthcheck", flag.ContinueOnError)
	url := fs.String("url", healthcheckURL, "healthz URL to probe")
	if err := fs.Parse(args); err != nil {
		return err
	}
	client := &http.Client{
		Timeout: healthcheckTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(*url)
	if err != nil {
		return errors.New("healthz request failed")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned status %d", resp.StatusCode)
	}
	return nil
}

func runHealthcheckAndExit() {
	err := RunHealthcheck(os.Args[2:])
	switch {
	case err == nil:
		return
	case errors.Is(err, flag.ErrHelp):
		return
	default:
		fmt.Fprintf(os.Stderr, "unhealthy: %v\n", err)
		os.Exit(1)
	}
}
