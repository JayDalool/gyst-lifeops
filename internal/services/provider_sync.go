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
	provider, ok := s.providerFor(link.Provider)
	if !ok {
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

// providerFor builds the provider for a link, with the injected credentials.
func (s *Service) providerFor(name string) (providers.Provider, bool) {
	switch name {
	case "github":
		return providers.GitHub{Token: s.ProviderCreds.GitHubToken}, true
	case "jira":
		return providers.Jira{Email: s.ProviderCreds.JiraEmail, Token: s.ProviderCreds.JiraToken, BaseURL: s.ProviderCreds.JiraBaseURL}, true
	case "slack":
		return providers.Slack{Token: s.ProviderCreds.SlackToken}, true
	}
	return nil, false
}
