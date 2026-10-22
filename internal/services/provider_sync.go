package services

import (
	"context"
	"errors"
	"log/slog"
	"net/url"

	"lifeops-tracker/internal/services/providers"
)

func (s *Service) SyncExternalLink(ctx context.Context, id int64) error {
	link, err := s.Repo.ExternalLinkByID(id)
	if err != nil {
		return err
	}
	var provider providers.Provider
	switch link.Provider {
	case "github":
		provider = providers.GitHub{Token: s.ProviderCreds.GitHubToken}
	case "jira":
		provider = providers.Jira{Email: s.ProviderCreds.JiraEmail, Token: s.ProviderCreds.JiraToken, BaseURL: s.ProviderCreds.JiraBaseURL}
	case "slack":
		provider = providers.Slack{Token: s.ProviderCreds.SlackToken}
	default:
		return userError(link.Provider + " does not support API sync")
	}
	snapshot, err := provider.Fetch(ctx, link)
	if err != nil {
		if errors.Is(err, providers.ErrHostNotAllowed) {
			host := ""
			if u, perr := url.Parse(link.URL); perr == nil {
				host = u.Hostname()
			}
			slog.Warn("provider failure", "event", "provider.failed", "provider", link.Provider, "link_id", link.ID, "host", host, "reason", "host not allowed")
		}
		return err
	}
	return s.Repo.UpdateExternalLinkSnapshot(id, snapshot.Title, snapshot.StatusText)
}
