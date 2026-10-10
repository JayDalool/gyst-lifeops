package services

import (
	"net/url"
	"path"
	"strings"

	"lifeops-tracker/internal/models"
)

// BuildExternalLink validates a manually pasted work-item URL and derives the
// compact label displayed by the UI. It performs no remote request.
func (s *Service) BuildExternalLink(entityType string, entityID int64, rawURL, summary string) (models.ExternalLink, error) {
	if entityID < 1 {
		return models.ExternalLink{}, userError("invalid link target")
	}
	exists, err := s.Repo.ExternalLinkEntityExists(entityType, entityID)
	if err != nil {
		return models.ExternalLink{}, err
	}
	if !exists {
		return models.ExternalLink{}, userError("link target not found")
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return models.ExternalLink{}, userError("external link must be an absolute http or https URL")
	}
	provider, key, title := classifyExternalURL(u)
	return models.ExternalLink{
		EntityType:  entityType,
		EntityID:    entityID,
		Provider:    provider,
		ExternalKey: key,
		URL:         u.String(),
		Title:       title,
		Summary:     strings.TrimSpace(summary),
		SyncSource:  "manual",
	}, nil
}

func classifyExternalURL(u *url.URL) (provider, key, title string) {
	host := strings.ToLower(u.Hostname())
	parts := splitURLPath(u.Path)
	provider, title = "other", u.Hostname()
	switch {
	case strings.Contains(host, "atlassian.") || strings.Contains(host, "jira"):
		provider, title = "jira", "Jira"
		key = segmentAfter(parts, "browse")
	case host == "github.com" || strings.HasSuffix(host, ".github.com"):
		provider, title = "github", "GitHub"
		if len(parts) >= 4 && (parts[2] == "issues" || parts[2] == "pull") {
			key = parts[0] + "/" + parts[1] + "#" + parts[3]
		}
	case strings.Contains(host, "slack.com"):
		provider, title = "slack", "Slack"
		if channel := segmentAfter(parts, "archives"); channel != "" {
			key = channel
			for i, part := range parts {
				if part == "archives" && i+2 < len(parts) {
					key += " · " + strings.TrimPrefix(parts[i+2], "p")
				}
			}
		}
	case strings.Contains(host, "linear.app"):
		provider, title = "linear", "Linear"
		key = segmentAfter(parts, "issue")
	case strings.Contains(host, "asana.com"):
		provider, title = "asana", "Asana"
		key = path.Base(u.Path)
	case strings.Contains(host, "notion.so") || strings.Contains(host, "notion.site"):
		provider, title = "notion", "Notion"
		key = path.Base(u.Path)
	}
	if key == "." || key == "/" {
		key = ""
	}
	return provider, key, title
}

func splitURLPath(value string) []string {
	var parts []string
	for _, part := range strings.Split(strings.Trim(value, "/"), "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func segmentAfter(parts []string, marker string) string {
	for i, part := range parts {
		if strings.EqualFold(part, marker) && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
