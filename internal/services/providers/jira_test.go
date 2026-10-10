package providers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func jiraEnv(t *testing.T, base string) {
	t.Helper()
	t.Setenv("JIRA_EMAIL", "dev@example.test")
	t.Setenv("JIRA_API_TOKEN", "fake-token-not-real")
	t.Setenv("JIRA_BASE_URL", base)
}

func forbidClient(t *testing.T) (*http.Client, *int) {
	n := new(int)
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		*n++
		t.Errorf("unexpected request to %s", r.URL)
		return nil, errors.New("blocked")
	})}, n
}

func TestJiraRefusesForeignHostsBeforeAnyRequest(t *testing.T) {
	jiraEnv(t, "https://team.atlassian.net")
	urls := []string{
		"https://team.atlassian.net.evil.example/browse/GYST-1",
		"https://evil-jira.example/browse/GYST-1",
		"https://team.atlassian.net@evil.example/browse/GYST-1",
		"https://evil.example/team.atlassian.net/browse/GYST-1",
		"http://team.atlassian.net/browse/GYST-1",
		"https://team.atlassian.net:8443/browse/GYST-1",
		"https://team.atlassian.net./browse/GYST-1",
		"https://xn--team-atlassian-net.example/browse/GYST-1",
		"https://tеam.atlassian.net/browse/GYST-1", // Cyrillic e
		"https://sub.team.atlassian.net/browse/GYST-1",
		"https://eam.atlassian.net/browse/GYST-1",
		"ftp://team.atlassian.net/browse/GYST-1",
		"//team.atlassian.net/browse/GYST-1",
		"::not a url",
	}
	client, n := forbidClient(t)
	for _, raw := range urls {
		_, err := (Jira{Client: client}).Fetch(context.Background(), models.ExternalLink{URL: raw, ExternalKey: "GYST-1"})
		if !errors.Is(err, ErrHostNotAllowed) {
			t.Errorf("%q: err = %v, want ErrHostNotAllowed", raw, err)
		}
	}
	if *n != 0 {
		t.Fatalf("%d requests sent", *n)
	}
}

func TestJiraRefusesEmptyKey(t *testing.T) {
	jiraEnv(t, "https://team.atlassian.net")
	client, _ := forbidClient(t)
	_, err := (Jira{Client: client}).Fetch(context.Background(), models.ExternalLink{URL: "https://team.atlassian.net/browse/X"})
	if !errors.Is(err, ErrHostNotAllowed) {
		t.Fatalf("err = %v", err)
	}
}

func TestJiraNeedsValidBaseURL(t *testing.T) {
	for _, base := range []string{"", "  ", "http://team.atlassian.net", "https://user:pw@team.atlassian.net",
		"https://team.atlassian.net?x=1", "https://team.atlassian.net/#frag", "team.atlassian.net", "https://", "https://team.atlassian.net?"} {
		jiraEnv(t, base)
		client, n := forbidClient(t)
		_, err := (Jira{Client: client}).Fetch(context.Background(), models.ExternalLink{URL: "https://team.atlassian.net/browse/GYST-1", ExternalKey: "GYST-1"})
		if !errors.Is(err, ErrNotConfigured) || *n != 0 {
			t.Errorf("base %q: err=%v requests=%d", base, err, *n)
		}
	}
}

func TestJiraBuildsRequestFromConfiguredBase(t *testing.T) {
	jiraEnv(t, "https://Team.Atlassian.net/")
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if got := r.URL.String(); got != "https://Team.Atlassian.net/rest/api/3/issue/GYST-1?fields=summary,status" {
			t.Errorf("URL = %s", got)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "dev@example.test" || pass != "fake-token-not-real" {
			t.Errorf("basic auth = %q %q %v", user, pass, ok)
		}
		body := `{"fields":{"summary":"Fix it","status":{"name":"Done"}}}`
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	snap, err := (Jira{Client: client}).Fetch(context.Background(), models.ExternalLink{URL: "https://team.atlassian.net/browse/GYST-1", ExternalKey: "GYST-1"})
	if err != nil || calls != 1 || snap.Title != "Fix it" || snap.StatusText != "Done" {
		t.Fatalf("snap=%+v err=%v calls=%d", snap, err, calls)
	}
}

func TestJiraKeepsBasePathAndEscapesKey(t *testing.T) {
	jiraEnv(t, "https://jira.example.test/jira/")
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.String(); got != "https://jira.example.test/jira/rest/api/3/issue/A%2F..%2FB?fields=summary,status" {
			t.Errorf("URL = %s", got)
		}
		return &http.Response{StatusCode: 404, Status: "404 Not Found", Body: http.NoBody, Header: make(http.Header)}, nil
	})}
	_, err := (Jira{Client: client}).Fetch(context.Background(), models.ExternalLink{URL: "https://jira.example.test/browse/A", ExternalKey: "A/../B"})
	if err == nil {
		t.Fatal("404 should be an error")
	}
}

func TestJiraDoesNotFollowRedirects(t *testing.T) {
	jiraEnv(t, "https://team.atlassian.net")
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "team.atlassian.net" {
			t.Errorf("redirect target requested: %s", r.URL)
		}
		h := make(http.Header)
		h.Set("Location", "https://evil.example/collect")
		return &http.Response{StatusCode: 302, Status: "302 Found", Body: http.NoBody, Header: h, Request: r}, nil
	})}
	_, err := (Jira{Client: client}).Fetch(context.Background(), models.ExternalLink{URL: "https://team.atlassian.net/browse/GYST-1", ExternalKey: "GYST-1"})
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
	if client.CheckRedirect != nil {
		t.Fatal("caller's client must not be mutated")
	}
}
