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

type Slack struct{ Client *http.Client }

func (p Slack) Fetch(ctx context.Context, link models.ExternalLink) (LinkSnapshot, error) {
	token := strings.TrimSpace(os.Getenv("SLACK_BOT_TOKEN"))
	if token == "" {
		return LinkSnapshot{}, ErrNotConfigured
	}
	u, err := url.Parse(link.URL)
	if err != nil {
		return LinkSnapshot{}, err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 3 {
		return LinkSnapshot{}, fmt.Errorf("unsupported Slack URL")
	}
	tsRaw := strings.TrimPrefix(parts[2], "p")
	if len(tsRaw) < 7 {
		return LinkSnapshot{}, fmt.Errorf("invalid Slack timestamp")
	}
	ts := tsRaw[:len(tsRaw)-6] + "." + tsRaw[len(tsRaw)-6:]
	endpoint := "https://slack.com/api/conversations.replies?channel=" + url.QueryEscape(parts[1]) + "&ts=" + url.QueryEscape(ts) + "&limit=1"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := p.Client
	if client == nil {
		client = Client()
	}
	resp, err := client.Do(req)
	if err != nil {
		return LinkSnapshot{}, err
	}
	defer resp.Body.Close()
	var result struct {
		OK       bool   `json:"ok"`
		Error    string `json:"error"`
		Messages []struct {
			Text string `json:"text"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return LinkSnapshot{}, err
	}
	if !result.OK {
		return LinkSnapshot{}, fmt.Errorf("Slack: %s", result.Error)
	}
	title := "Slack message"
	if len(result.Messages) > 0 {
		title = result.Messages[0].Text
	}
	return LinkSnapshot{Title: title, StatusText: "synced"}, nil
}
