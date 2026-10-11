package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunHealthcheck(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }))
	defer ok.Close()
	if err := RunHealthcheck([]string{"-url", ok.URL}); err != nil {
		t.Fatalf("200: %v", err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("fake-body-text"))
	}))
	defer bad.Close()
	err := RunHealthcheck([]string{"-url", bad.URL})
	if err == nil {
		t.Fatal("503 should fail")
	}
	if got := err.Error(); got != "healthz returned status 503" {
		t.Fatalf("error leaks detail: %q", got)
	}

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, ok.URL, http.StatusFound)
	}))
	defer redirect.Close()
	if err := RunHealthcheck([]string{"-url", redirect.URL}); err == nil {
		t.Fatal("redirect must not be followed")
	}

	closed := httptest.NewServer(http.NotFoundHandler())
	url := closed.URL
	closed.Close()
	if err := RunHealthcheck([]string{"-url", url}); err == nil {
		t.Fatal("closed listener should fail")
	}
}

func TestRunHealthcheckTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(4 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer slow.Close()
	start := time.Now()
	if err := RunHealthcheck([]string{"-url", slow.URL}); err == nil {
		t.Fatal("slow handler should fail")
	}
	if d := time.Since(start); d > 3500*time.Millisecond {
		t.Fatalf("took %v, want about 3s", d)
	}
}

func TestHealthcheckRequested(t *testing.T) {
	if !healthcheckRequested([]string{"lifeops", "healthcheck"}) {
		t.Fatal("healthcheck should match")
	}
	for _, args := range [][]string{
		nil, {"lifeops"}, {"lifeops", "backup"}, {"lifeops", "-passwordhash"},
		{"lifeops", "-addr", "healthcheck"}, {"lifeops", "x", "healthcheck"},
	} {
		if healthcheckRequested(args) {
			t.Fatalf("%v must not match", args)
		}
	}
}
