package providers

import (
	"context"
	"net/http"
	"testing"

	"lifeops-tracker/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestGitHubBuildsIssueRequest(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://api.github.com/repos/acme/app/issues/42" {
			t.Fatalf("URL = %s", request.URL)
		}
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: http.NoBody, Header: make(http.Header)}, nil
	})}
	_, err := (GitHub{Client: client}).Fetch(context.Background(), models.ExternalLink{URL: "https://github.com/acme/app/issues/42"})
	if err == nil {
		t.Fatal("empty response body should fail decoding")
	}
}
