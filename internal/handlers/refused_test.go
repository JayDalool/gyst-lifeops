package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// doneProjectWithMission returns a project marked done while one of its
// missions is still pending: the state Jay hit on 2026-09-30 (project 12).
func doneProjectWithMission(t *testing.T, h *Handler) (models.Project, models.ProjectItem) {
	t.Helper()
	project := models.Project{Name: "Finished Quest", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Leftover mission"}
	if err := h.Service.Repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	postQuestForm(t, h, fmt.Sprintf("/projects/%d/status", project.ID), url.Values{"status": {"done"}})
	return project, item
}

func postRaw(h *Handler, path string, values url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func assertRefusedPage(t *testing.T, rec *httptest.ResponseRecorder, back, wantMessage string) {
	t.Helper()
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want an HTML page the installed app can leave", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{`href="` + back + `"`, `href="/today"`, "Go back", wantMessage} {
		if !strings.Contains(body, want) {
			t.Errorf("refused page missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "only active project missions") || strings.Contains(body, "before completion") {
		t.Errorf("refused page leaks the internal error text:\n%s", body)
	}
}

func TestDoneProjectMissionCompletionIsRefusedWithAWayBack(t *testing.T) {
	h := testHandler(t)
	project, item := doneProjectWithMission(t, h)
	back := fmt.Sprintf("/projects/%d#mission-%d", project.ID, item.ID)

	for _, path := range []string{
		fmt.Sprintf("/projects/%d/items/%d/complete", project.ID, item.ID),
		fmt.Sprintf("/roadmap/projects/%d/items/%d/complete", project.ID, item.ID),
	} {
		rec := postRaw(h, path, url.Values{"return_to": {back}})
		assertRefusedPage(t, rec, back, "isn&#39;t active")
	}

	items, err := h.Service.Repo.ProjectItems(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Status != "pending" || items[0].StarsAwarded != 0 {
		t.Fatalf("refused mission changed: %+v", items[0])
	}
}

func TestRefusedPageNeverLinksOffSite(t *testing.T) {
	h := testHandler(t)
	project, item := doneProjectWithMission(t, h)
	path := fmt.Sprintf("/projects/%d/items/%d/complete", project.ID, item.ID)
	for _, hostile := range []string{"//evil.example/x", "https://evil.example/", "javascript:alert(1)"} {
		rec := postRaw(h, path, url.Values{"return_to": {hostile}})
		assertRefusedPage(t, rec, fmt.Sprintf("/projects/%d", project.ID), "isn&#39;t active")
		if strings.Contains(rec.Body.String(), "evil.example") || strings.Contains(rec.Body.String(), "javascript:") {
			t.Fatalf("refused page linked to %q", hostile)
		}
	}
}

func TestMissionWithoutRequiredProofIsRefusedWithAWayBack(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Proof Quest", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Needs evidence", ProofRequired: true, ProofType: "link"}
	if err := h.Service.Repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	rec := postRaw(h, fmt.Sprintf("/projects/%d/items/%d/complete", project.ID, item.ID), nil)
	assertRefusedPage(t, rec, fmt.Sprintf("/projects/%d", project.ID), "needs proof")
}

func TestUnknownMissionIsStillAServerError(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Real Quest", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	rec := postRaw(h, fmt.Sprintf("/projects/%d/items/999999/complete", project.ID), nil)
	if rec.Code == http.StatusConflict || rec.Code < 400 {
		t.Fatalf("missing mission answered %d; only rule refusals get the 409 page", rec.Code)
	}
}

func TestDoneProjectPageOffersNoMissionCheckbox(t *testing.T) {
	h := testHandler(t)
	project, item := doneProjectWithMission(t, h)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%d", project.ID), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("project page status %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, fmt.Sprintf("/items/%d/complete", item.ID)) {
		t.Fatal("done project still offers a complete form for its pending mission")
	}
	if !strings.Contains(body, `disabled aria-label="Project is done, can't mark done: Leftover mission"`) {
		t.Errorf("pending mission is not shown as a disabled checkbox:\n%s", body)
	}
	if !strings.Contains(body, "Reopen the project to keep working on it.") {
		t.Error("done project page doesn't explain why its missions are disabled")
	}
}

func TestTaskWithoutRequiredProofIsRefusedWithAWayBack(t *testing.T) {
	h := testHandler(t)
	task := models.Task{Title: "Needs a link", DueDate: "2026-06-24", ProofRequired: true, ProofType: "link"}
	if err := h.Service.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	rec := postRaw(h, fmt.Sprintf("/tasks/%d/complete", task.ID), url.Values{"occurrence_date": {"2026-06-24"}})
	assertRefusedPage(t, rec, fmt.Sprintf("/today#task-%d", task.ID), "needs proof")
	got, err := h.Service.Repo.TaskByID(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "pending" || got.StarsAwarded != 0 {
		t.Fatalf("refused task changed: %+v", got)
	}
}
