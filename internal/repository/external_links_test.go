package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

func TestExternalLinkCRUDDoesNotChangeRewardsOrCompletion(t *testing.T) {
	repo := testRepository(t)
	project := models.Project{Name: "External quest", Status: "active", ProjectKind: "main"}
	if err := repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "External mission", StarsReward: 7}
	if err := repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	before, err := repo.PlayerStats()
	if err != nil {
		t.Fatal(err)
	}
	link := models.ExternalLink{EntityType: "project_item", EntityID: item.ID, Provider: "jira", ExternalKey: "GYST-1", URL: "https://example.atlassian.net/browse/GYST-1", Title: "Jira", Summary: "First"}
	if err := repo.CreateExternalLink(&link); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateExternalLinkSummary(link.ID, "Updated"); err != nil {
		t.Fatal(err)
	}
	links, err := repo.ExternalLinks("project_item", item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].Summary != "Updated" || links[0].SyncVersion != 2 {
		t.Fatalf("updated links = %#v", links)
	}
	after, err := repo.PlayerStats()
	if err != nil {
		t.Fatal(err)
	}
	items, err := repo.ProjectItems(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before != after || len(items) != 1 || items[0].Status != "pending" || items[0].StarsAwarded != 0 {
		t.Fatalf("external link changed quest state: before=%#v after=%#v item=%#v", before, after, items)
	}
	if err := repo.DeleteExternalLink(link.ID); err != nil {
		t.Fatal(err)
	}
}
