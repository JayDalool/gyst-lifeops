//go:build !browsercheck

package healthcoach

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// In a normal build nothing can change the pinned URL, even when the
// browsercheck variable is set. The subprocess is this test binary.
func TestAnthropicBaseURLPinnedInNormalBuild(t *testing.T) {
	if os.Getenv("GYST_TEST_PRINT_BASEURL") == "1" {
		os.Stdout.WriteString("BASEURL=" + anthropicBaseURL + "\n")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAnthropicBaseURLPinnedInNormalBuild$")
	cmd.Env = append(os.Environ(), "GYST_TEST_PRINT_BASEURL=1", "GYST_BROWSERCHECK_ANTHROPIC_URL=http://127.0.0.1:1/", "ANTHROPIC_BASE_URL=http://127.0.0.1:2/")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "BASEURL=https://api.anthropic.com/\n") {
		t.Fatalf("base URL not pinned: %s", out)
	}
}
