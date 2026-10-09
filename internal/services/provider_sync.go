package services

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		provider = providers.GitHub{}
	case "jira":
		provider = providers.Jira{}
	case "slack":
		provider = providers.Slack{}
	default:
		return fmt.Errorf("%s does not support API sync", link.Provider)
	}
	snapshot, err := provider.Fetch(ctx, link)
	if err != nil {
		if errors.Is(err, providers.ErrHostNotAllowed) {
			host := ""
			if u, perr := url.Parse(link.URL); perr == nil {
				host = u.Hostname()
			}
			log.Printf("SECURITY: external link %d sync refused: host not allowed (%s)", link.ID, host)
		}
		return err
	}
	return s.Repo.UpdateExternalLinkSnapshot(id, snapshot.Title, snapshot.StatusText)
}
