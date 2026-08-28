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

type Jira struct{ Client *http.Client }

func (p Jira) Fetch(ctx context.Context, link models.ExternalLink) (LinkSnapshot, error) {
	token, email := strings.TrimSpace(os.Getenv("JIRA_API_TOKEN")), strings.TrimSpace(os.Getenv("JIRA_EMAIL"))
	if token == "" || email == "" {
		return LinkSnapshot{}, ErrNotConfigured
	}
	u, err := url.Parse(link.URL)
	if err != nil {
		return LinkSnapshot{}, err
	}
	endpoint := u.Scheme + "://" + u.Host + "/rest/api/3/issue/" + url.PathEscape(link.ExternalKey) + "?fields=summary,status"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")
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
		return LinkSnapshot{}, fmt.Errorf("Jira returned %s", resp.Status)
	}
	var result struct {
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return LinkSnapshot{}, err
	}
	return LinkSnapshot{Title: result.Fields.Summary, StatusText: result.Fields.Status.Name}, nil
}
