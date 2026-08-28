package services

import (
	"path/filepath"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
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
