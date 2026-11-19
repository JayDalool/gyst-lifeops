package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"lifeops-tracker/internal/config"
	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services/providers"
)

func externalLinkService(t *testing.T) (*Service, models.ProjectItem) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "links.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	repo := repository.New(database)
	project := models.Project{Name: "Linked quest", Status: "active", ProjectKind: "main"}
	if err := repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Linked mission"}
	if err := repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	return New(repo), item
}

func TestBuildExternalLinkClassifiesProviders(t *testing.T) {
	service, item := externalLinkService(t)
	tests := []struct {
		url, provider, key string
	}{
		{"https://team.atlassian.net/browse/GYST-123", "jira", "GYST-123"},
		{"https://github.com/owner/repo/issues/12", "github", "owner/repo#12"},
		{"https://workspace.slack.com/archives/C0123/p1720000000000000", "slack", "C0123 · 1720000000000000"},
		{"https://linear.app/team/issue/ENG-42/fix-it", "linear", "ENG-42"},
		{"https://docs.example.com/work/7", "other", ""},
	}
	for _, tt := range tests {
		link, err := service.BuildExternalLink("project_item", item.ID, tt.url, "Context")
		if err != nil {
			t.Fatalf("build %s: %v", tt.url, err)
		}
		if link.Provider != tt.provider || link.ExternalKey != tt.key {
			t.Errorf("%s classified %s/%s, want %s/%s", tt.url, link.Provider, link.ExternalKey, tt.provider, tt.key)
		}
	}
}

func TestBuildExternalLinkRejectsUnsafeAndMissingTargets(t *testing.T) {
	service, item := externalLinkService(t)
	for _, rawURL := range []string{"/relative", "javascript:alert(1)", "ftp://example.com/file"} {
		if _, err := service.BuildExternalLink("project_item", item.ID, rawURL, ""); err == nil {
			t.Errorf("accepted unsafe URL %q", rawURL)
		}
	}
	if _, err := service.BuildExternalLink("project_item", 999999, "https://example.com", ""); err == nil {
		t.Error("accepted missing target")
	}
}

func TestSyncExternalLinkRefusesForeignJiraHost(t *testing.T) {
	service, item := externalLinkService(t)
	service.ProviderCreds = config.Providers{JiraEmail: "dev@example.test", JiraToken: "fake-token-not-real", JiraBaseURL: "https://team.atlassian.net"}
	link, err := service.BuildExternalLink("project_item", item.ID, "https://team.atlassian.net.evil.example/browse/GYST-9", "Manual title")
	if err != nil {
		t.Fatal(err)
	}
	if link.Provider != "jira" {
		t.Fatalf("provider = %s", link.Provider)
	}
	if err := service.Repo.CreateExternalLink(&link); err != nil {
		t.Fatal(err)
	}
	buf := captureLog(t)
	err = service.SyncExternalLink(context.Background(), link.ID)
	if !errors.Is(err, providers.ErrHostNotAllowed) {
		t.Fatalf("err = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, fmt.Sprintf(`"link_id":%d`, link.ID)) || !strings.Contains(out, `"reason":"host not allowed"`) ||
		!strings.Contains(out, "team.atlassian.net.evil.example") || strings.Contains(out, "/browse/") || strings.Contains(out, "fake-token-not-real") {
		t.Fatalf("log = %q", out)
	}
	after, err := service.Repo.ExternalLinkByID(link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != link.Title || after.StatusText != link.StatusText || after.LastSyncedAt != link.LastSyncedAt {
		t.Fatalf("snapshot changed: %+v -> %+v", link, after)
	}
}
