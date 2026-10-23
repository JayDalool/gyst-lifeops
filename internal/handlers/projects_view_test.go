package handlers

import (
	"lifeops-tracker/internal/models"
	"strings"
	"testing"
)

func TestProjectFiltersPreserveHierarchyAndStatus(t *testing.T) {
	tree := models.ProjectTree{
		Main:     []models.ProjectWithProgress{{Project: models.Project{ID: 1, Name: "Active parent", Status: "active"}, Progress: 100}, {Project: models.Project{ID: 2, Name: "Done project", Status: "done"}}, {Project: models.Project{ID: 3, Name: "Paused project", Status: "paused"}}},
		Upcoming: []models.ProjectWithProgress{{Project: models.Project{ID: 4, Name: "Locked project", Status: "locked"}}},
		Children: map[int64][]models.ProjectWithProgress{1: {{Project: models.Project{ID: 5, Name: "Completed child", Status: "completed"}}}},
	}
	for _, tt := range []struct {
		filter                   string
		main, upcoming, children int
	}{{"all", 3, 1, 1}, {"active", 2, 0, 0}, {"completed", 2, 0, 1}, {"upcoming", 0, 1, 0}, {"invalid", 3, 1, 1}} {
		d := ProjectListData{Page: "projects", StatusFilter: tt.filter}
		got := d.VisibleTree(tree)
		if len(got.Main) != tt.main || len(got.Upcoming) != tt.upcoming || len(got.Children[1]) != tt.children {
			t.Fatalf("%s: %#v", tt.filter, got)
		}
		if tt.filter == "completed" && got.Main[0].Project.Status != "active" {
			t.Fatal("context parent status was invented")
		}
	}
	if len(tree.Main) != 3 || len(tree.Children[1]) != 1 {
		t.Fatal("filter mutated source hierarchy")
	}
	d := ProjectListData{SelectedRoadmapID: 17, StatusFilter: "completed"}
	segments := d.Filters()
	selected := 0
	for _, item := range segments.Items {
		if !strings.Contains(item.URL, "roadmap=17") {
			t.Fatal("filter discarded roadmap")
		}
		if item.Selected {
			selected++
			if item.Label != "Completed" {
				t.Fatal("wrong selected status")
			}
		}
	}
	if selected != 1 {
		t.Fatal("status selection ambiguous")
	}
	if d.FilterURL("completed", 23) != "/projects?roadmap=23&status=completed" {
		t.Fatal("roadmap change discarded status")
	}
}

func TestProjectDetailStatusDotUsesOneMapping(t *testing.T) {
	data := ProjectDetailData{Page: "projects", Project: models.ProjectWithProgress{Project: models.Project{ID: 20, Name: "Fictional project", Status: "active"}}}
	for _, tt := range []struct {
		status   string
		progress int
		class    string
	}{{"paused", 0, "locked"}, {"upcoming", 0, "locked"}, {"locked", 0, "locked"}, {"done", 0, "done"}, {"active", 100, "done"}, {"active", 40, "current"}} {
		data.Project.Project.Status = tt.status
		data.Project.Progress = tt.progress
		body := renderBody(t, "project_detail.html", data)
		if !strings.Contains(body, `<span class="status `+tt.class+`"><span class="dot"></span>`+tt.status) {
			t.Errorf("%s at %d%%: want status dot %q", tt.status, tt.progress, tt.class)
		}
	}
}

func TestProjectDetailPreservesActionsAndSections(t *testing.T) {
	data := ProjectDetailData{Page: "projects", HasRoadmap: true, Roadmap: models.Roadmap{ID: 7, Name: "Real roadmap"}, Project: models.ProjectWithProgress{Project: models.Project{ID: 20, Name: "Real project", Status: "active", Notes: "Existing notes"}}, Items: []models.ProjectItem{{ID: 21, Title: "Open item", Status: "pending", ProofType: "link", ProofRequired: true}, {ID: 22, Title: "Done item", Status: "completed", ProofType: "none"}}, ExternalLinks: []models.ExternalLink{{ID: 9, Provider: "github", URL: "https://example.com/work", Summary: "Real external work"}}}
	body := renderBody(t, "project_detail.html", data)
	for _, want := range []string{`id="overview"`, `id="tasks"`, `id="roadmap"`, `id="notes"`, `href="/roadmap?roadmap=7"`, `Existing notes`, `/projects/20/items/21/complete`, `/projects/20/items/22/reopen`, `/projects/20/items/21/proof`, `/projects/20/items/21/delete`, `action="/projects/20/items"`, `/external-links/9/edit`, `/external-links/9/delete`, `/external-links/9/sync`, `action="/external-links"`, `/projects/20/status`, `/projects/20/archive`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s", want)
		}
	}
	for _, status := range []string{"locked", "upcoming", "archived"} {
		data.Project.Project.Status = status
		body = renderBody(t, "project_detail.html", data)
		if strings.Contains(body, `action="/projects/20/items/21/complete"`) {
			t.Fatalf("%s enabled item completion", status)
		}
		if !strings.Contains(body, "Unavailable until project is active") {
			t.Fatalf("%s lost unavailable state", status)
		}
	}
}
