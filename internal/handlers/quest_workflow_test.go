package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func postQuestForm(t *testing.T, h *Handler, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST %s status %d: %s", path, rec.Code, rec.Body.String())
	}
	return rec
}

func TestManualUnlockHandler(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Locked Quest", Status: "locked", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}

	postQuestForm(t, h, fmt.Sprintf("/projects/%d/status", project.ID), url.Values{"status": {"active"}})
	got, err := h.Service.Repo.ProjectByID(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "active" {
		t.Fatalf("manual unlock status = %q, want active", got.Status)
	}
}

func TestEditProjectMovesItUnderParent(t *testing.T) {
	h := testHandler(t)
	parent := models.Project{Name: "Main Quest", Status: "active", ProjectKind: "main"}
	child := models.Project{Name: "Chapter", Status: "upcoming", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&parent); err != nil {
		t.Fatal(err)
	}
	if err := h.Service.Repo.CreateProject(&child); err != nil {
		t.Fatal(err)
	}

	postQuestForm(t, h, fmt.Sprintf("/projects/%d/edit", child.ID), url.Values{
		"name":              {child.Name},
		"status":            {"upcoming"},
		"project_kind":      {"phase"},
		"parent_project_id": {strconv.FormatInt(parent.ID, 10)},
	})
	got, err := h.Service.Repo.ProjectByID(child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ParentProjectID != parent.ID || got.ProjectKind != "phase" {
		t.Fatalf("moved project parent/kind = %d/%q, want %d/phase", got.ParentProjectID, got.ProjectKind, parent.ID)
	}
}

func TestProjectMissionCompletionHandlerDoesNotDuplicateReward(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Reward Quest", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Ship it"}
	if err := h.Service.Repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/projects/%d/items/%d", project.ID, item.ID)
	first := postQuestForm(t, h, base+"/complete", nil)
	if location := first.Header().Get("Location"); !strings.Contains(location, "celebrate=project") || !strings.Contains(location, "celebrate_xp=250") {
		t.Fatalf("project completion redirect = %q, want project celebration and XP", location)
	}
	postQuestForm(t, h, base+"/reopen", nil)
	postQuestForm(t, h, base+"/complete", nil)

	stats, err := h.Service.Repo.PlayerStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalStars != 270 {
		t.Fatalf("total XP = %d, want one 20-XP mission and one 250-XP quest reward", stats.TotalStars)
	}
}

func TestRoadmapFinalMissionRedirectsToProjectCelebration(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Roadmap Finale", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Final roadmap mission", StarsReward: 35}
	if err := h.Service.Repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/roadmap/projects/%d/items/%d/complete", project.ID, item.ID)
	rec := postQuestForm(t, h, path, nil)
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "celebrate=project") || !strings.Contains(location, "celebrate_xp=250") {
		t.Fatalf("roadmap completion redirect = %q, want project celebration", location)
	}
}

func TestProjectMissionProofHandler(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Proof Quest", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Evidence", ProofRequired: true, ProofType: "link", StarsReward: 3}
	if err := h.Service.Repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/projects/%d/items/%d", project.ID, item.ID)
	postQuestForm(t, h, base+"/proof", url.Values{
		"proof_type": {"link"},
		"proof_url":  {"https://example.com/evidence"},
	})
	postQuestForm(t, h, base+"/complete", nil)

	items, err := h.Service.Repo.ProjectItems(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].StarsAwarded != 8 || items[0].ProofURL == "" {
		t.Fatalf("proof workflow result: %#v", items)
	}
}

func TestAttachJiraLinkToMissionRendersChipWithoutReward(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Integration Quest", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Tracked mission", StarsReward: 5}
	if err := h.Service.Repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	before, err := h.Service.Repo.PlayerStats()
	if err != nil {
		t.Fatal(err)
	}
	projectPath := fmt.Sprintf("/projects/%d", project.ID)
	postQuestForm(t, h, "/external-links", url.Values{
		"entity_type": {"project_item"},
		"entity_id":   {strconv.FormatInt(item.ID, 10)},
		"url":         {"https://team.atlassian.net/browse/GYST-88"},
		"summary":     {"Review the imported ticket"},
		"return_to":   {projectPath},
	})
	req := httptest.NewRequest(http.MethodGet, projectPath, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"GYST-88", "Review the imported ticket", `target="_blank"`, `rel="noopener noreferrer"`} {
		if !strings.Contains(body, want) {
			t.Errorf("project page missing %q", want)
		}
	}
	after, err := h.Service.Repo.PlayerStats()
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("attaching external work changed player stats: before=%#v after=%#v", before, after)
	}
}
