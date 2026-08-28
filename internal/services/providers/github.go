package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"lifeops-tracker/internal/models"
)

type GitHub struct{ Client *http.Client }

func (p GitHub) Fetch(ctx context.Context, link models.ExternalLink) (LinkSnapshot, error) {
	u, err := url.Parse(link.URL)
	if err != nil {
		return LinkSnapshot{}, err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return LinkSnapshot{}, fmt.Errorf("unsupported GitHub work item URL")
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s", parts[0], parts[1], parts[3])
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := p.Client
	if client == nil {
		client = Client()
	}
	resp, err := client.Do(req)
	if err != nil {
		return LinkSnapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return LinkSnapshot{}, fmt.Errorf("GitHub returned %s", resp.Status)
	}
	var result struct {
		Title, State string
		PullRequest  json.RawMessage `json:"pull_request"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return LinkSnapshot{}, err
	}
	status := result.State
	if len(result.PullRequest) > 0 {
		status = "pull request · " + status
	}
	return LinkSnapshot{Title: result.Title, StatusText: status}, nil
}
