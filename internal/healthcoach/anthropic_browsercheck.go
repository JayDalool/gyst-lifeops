//go:build browsercheck

package healthcoach

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

// Browser-check hook (#243). Compiled only with -tags browsercheck, which the
// Dockerfile, Makefile, compose file and CI never use. It lets a local browser
// check point the adapter at cmd/fakeanthropic. Anything other than a
// loopback http URL stops the program rather than sending the key or a photo
// somewhere else.
const browsercheckEnv = "GYST_BROWSERCHECK_ANTHROPIC_URL"

var loopbackURL = regexp.MustCompile(`^http://127\.0\.0\.1:[0-9]{1,5}/$`)

func browsercheckBaseURL(raw string) (string, error) {
	if !loopbackURL.MatchString(raw) {
		return "", fmt.Errorf("%s must be http://127.0.0.1:<port>/", browsercheckEnv)
	}
	return raw, nil
}

func init() {
	raw, set := os.LookupEnv(browsercheckEnv)
	if !set {
		return
	}
	u, err := browsercheckBaseURL(raw)
	if err != nil {
		panic(err.Error())
	}
	anthropicBaseURL = u
	// Fixed line, loopback URL only: a tagged build must never be mistaken
	// for a production one.
	log.Printf("browsercheck build: Anthropic base URL overridden to %s", u)
}
