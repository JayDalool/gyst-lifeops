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

// jiraBase parses JIRA_BASE_URL. It must be https with a host, and carry no
// userinfo, query or fragment. Any other value means Jira is not configured.
func jiraBase() (*url.URL, error) {
	raw := strings.TrimSpace(os.Getenv("JIRA_BASE_URL"))
	if raw == "" {
		return nil, ErrNotConfigured
	}
	base, err := url.Parse(raw)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil ||
		base.RawQuery != "" || base.ForceQuery || base.Fragment != "" || base.Opaque != "" {
		return nil, ErrNotConfigured
	}
	base.Path = strings.TrimRight(base.Path, "/")
	base.RawPath = ""
	return base, nil
}

func (p Jira) Fetch(ctx context.Context, link models.ExternalLink) (LinkSnapshot, error) {
	token, email := strings.TrimSpace(os.Getenv("JIRA_API_TOKEN")), strings.TrimSpace(os.Getenv("JIRA_EMAIL"))
	if token == "" || email == "" {
		return LinkSnapshot{}, ErrNotConfigured
	}
	base, err := jiraBase()
	if err != nil {
		return LinkSnapshot{}, err
	}
	// Check the link host before any request or credential exists. u.Host
	// excludes userinfo, so "https://good@evil.example" yields evil.example.
	u, err := url.Parse(link.URL)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, base.Host) || link.ExternalKey == "" {
		return LinkSnapshot{}, ErrHostNotAllowed
	}
	// The endpoint comes from the configured base, never from the link.
	endpoint := "https://" + base.Host + base.Path + "/rest/api/3/issue/" + url.PathEscape(link.ExternalKey) + "?fields=summary,status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return LinkSnapshot{}, err
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")
	client := p.Client
	if client == nil {
		client = Client()
	}
	// Never follow redirects: a 3xx is an error and the credential is not replayed.
	noRedirect := *client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client = &noRedirect
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
