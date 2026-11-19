package services

import (
	"testing"

	"lifeops-tracker/internal/config"
	"lifeops-tracker/internal/services/providers"
)

// The configured credentials must reach each provider (no network involved).
func TestProviderForInjectsConfiguredCredentials(t *testing.T) {
	s := &Service{ProviderCreds: config.Providers{
		GitHubToken: "gh-fictional", SlackToken: "sl-fictional",
		JiraEmail: "dev@example.test", JiraToken: "jira-fictional", JiraBaseURL: "https://team.atlassian.net",
	}}
	gh, ok := s.providerFor("github")
	if got, isGH := gh.(providers.GitHub); !ok || !isGH || got.Token != "gh-fictional" {
		t.Fatalf("github: %+v %v", gh, ok)
	}
	sl, ok := s.providerFor("slack")
	if got, isSl := sl.(providers.Slack); !ok || !isSl || got.Token != "sl-fictional" {
		t.Fatalf("slack: %+v %v", sl, ok)
	}
	jr, ok := s.providerFor("jira")
	if got, isJr := jr.(providers.Jira); !ok || !isJr || got.Token != "jira-fictional" || got.Email != "dev@example.test" || got.BaseURL != "https://team.atlassian.net" {
		t.Fatalf("jira: %+v %v", jr, ok)
	}
	if _, ok := s.providerFor("other"); ok {
		t.Fatal("unknown provider accepted")
	}
}
