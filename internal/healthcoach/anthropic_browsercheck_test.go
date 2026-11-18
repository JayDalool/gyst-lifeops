//go:build browsercheck

package healthcoach

import "testing"

func TestBrowsercheckBaseURLLoopbackOnly(t *testing.T) {
	if u, err := browsercheckBaseURL("http://127.0.0.1:8099/"); err != nil || u != "http://127.0.0.1:8099/" {
		t.Fatalf("loopback refused: %q %v", u, err)
	}
	for _, bad := range []string{"", "https://api.anthropic.com/", "http://localhost:80/", "http://127.0.0.1/", "http://127.0.0.1:80", "http://127.0.0.1:80/x", "http://10.0.0.1:80/", "http://127.0.0.1:80@evil.example/", "https://127.0.0.1:80/"} {
		if _, err := browsercheckBaseURL(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
