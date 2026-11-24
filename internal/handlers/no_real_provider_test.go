package handlers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
)

// GYST-180: no test in this package may reach a real AI provider. Photo and
// Coach tests use in-process fakes on 127.0.0.1; a request to the real hosts,
// or any DNS lookup, fails the whole run so a slip cannot spend credit or send
// a photo.
var realProviderCalls atomic.Int32

type refuseRealProviders struct{ next http.RoundTripper }

func (r refuseRealProviders) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Hostname() {
	case "api.anthropic.com", "api.openai.com":
		realProviderCalls.Add(1)
		return nil, fmt.Errorf("test guard: %s is a real AI provider", req.URL.Hostname())
	}
	return r.next.RoundTrip(req)
}

func TestMain(m *testing.M) {
	http.DefaultTransport = refuseRealProviders{next: http.DefaultTransport}
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(context.Context, string, string) (net.Conn, error) {
		realProviderCalls.Add(1)
		return nil, errors.New("test guard: no DNS lookups in tests")
	}}
	code := m.Run()
	if n := realProviderCalls.Load(); n > 0 && code == 0 {
		fmt.Fprintf(os.Stderr, "FAIL: %d attempt(s) to resolve or call a real AI provider\n", n)
		code = 1
	}
	os.Exit(code)
}
