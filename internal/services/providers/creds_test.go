package providers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"lifeops-tracker/internal/models"
)

func TestGitHubSendsAuthorizationOnlyWithAToken(t *testing.T) {
	for token, want := range map[string]string{"": "", "  ": "", " gh-fictional ": "Bearer gh-fictional"} {
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if got := request.Header.Get("Authorization"); got != want {
				t.Errorf("token %q: Authorization = %q, want %q", token, got, want)
			}
			return &http.Response{StatusCode: 404, Status: "404 Not Found", Body: http.NoBody, Header: make(http.Header)}, nil
		})}
		_, _ = (GitHub{Client: client, Token: token}).Fetch(context.Background(), models.ExternalLink{URL: "https://github.com/acme/app/issues/42"})
	}
}

func TestJiraAndSlackAreNotConfiguredWithoutCredentials(t *testing.T) {
	link := models.ExternalLink{URL: "https://team.atlassian.net/browse/GYST-1", ExternalKey: "GYST-1"}
	for _, j := range []Jira{
		{},
		{Email: "dev@example.test", BaseURL: "https://team.atlassian.net"},
		{Token: "tok-fictional", BaseURL: "https://team.atlassian.net"},
		{Email: "dev@example.test", Token: "tok-fictional"},
	} {
		if _, err := j.Fetch(context.Background(), link); !errors.Is(err, ErrNotConfigured) {
			t.Errorf("Jira %+v: err = %v", j, err)
		}
	}
	slackLink := models.ExternalLink{URL: "https://x.slack.com/archives/C1/p1700000000000100"}
	if _, err := (Slack{}).Fetch(context.Background(), slackLink); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Slack: err = %v", err)
	}
}
