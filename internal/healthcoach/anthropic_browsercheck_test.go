//go:build browsercheck

package healthcoach

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestBrowsercheckBaseURLLoopbackOnly(t *testing.T) {
	if u, err := browsercheckBaseURL("http://127.0.0.1:8099/"); err != nil || u != "http://127.0.0.1:8099/" {
		t.Fatalf("loopback refused: %q %v", u, err)
	}
	for _, bad := range []string{"", "https://api.anthropic.com/", "http://localhost:80/", "http://127.0.0.1/", "http://127.0.0.1:80", "http://127.0.0.1:80/x", "http://10.0.0.1:80/", "http://127.0.0.1:80@evil.example/", "https://127.0.0.1:80/", "http://[::1]:80/", "http://127.0.0.2:80/", "http://0.0.0.0:80/", "http://127.0.0.1:80/?x=1", "http://127.0.0.1:80/#f", "http://127.0.0.1.example.com:80/", "http://127.0.0.1:http/", "http://user@127.0.0.1:80/"} {
		if _, err := browsercheckBaseURL(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// A rejected value stops the program at start-up: the test binary itself is
// the subprocess, and its init panics before any test runs.
func TestBrowsercheckBadURLStopsStartup(t *testing.T) {
	if os.Getenv("GYST_TEST_NEVER_REACHED") == "1" {
		t.Fatal("start-up did not stop")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBrowsercheckBadURLStopsStartup$")
	cmd.Env = append(os.Environ(), "GYST_TEST_NEVER_REACHED=1", "GYST_BROWSERCHECK_ANTHROPIC_URL=https://api.anthropic.com/")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "must be http://127.0.0.1:<port>/") {
		t.Fatalf("subprocess err=%v out=%s", err, out)
	}
}
