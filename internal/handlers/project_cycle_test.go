package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

func cycleHandler(t *testing.T) (*Handler, *sql.DB) {
	t.Helper()
	database := migratedTestDB(t)
	h := New(services.New(repository.New(database)), "../../web/templates")
	h.AllowPublicFeeds = true
	return h, database
}

func cycleProject(t *testing.T, h *Handler, name string, parent int64) models.Project {
	t.Helper()
	p := models.Project{Name: name, ProjectKind: "main", Status: "active", ParentProjectID: parent, AutoUnlock: true}
	if err := h.Service.Repo.CreateProject(&p); err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	return p
}

func serve(h *Handler, method, path string, form url.Values) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestEditProjectRefusesDescendantParentWith422(t *testing.T) {
	h, database := cycleHandler(t)
	a := cycleProject(t, h, "Fictional Atlas", 0)
	b := cycleProject(t, h, "Orbit Alpha", a.ID)

	rec := serve(h, http.MethodPost, fmt.Sprintf("/projects/%d/edit", a.ID), url.Values{
		"name":              {"Fictional Atlas Renamed"},
		"parent_project_id": {fmt.Sprint(b.ID)},
		"status":            {"active"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Not saved", "A project can&#39;t sit inside its own sub-project. Choose another parent, or None.", `role="alert"`, "Fictional Atlas Renamed"} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
	for _, leak := range []string{"SQL", "sqlite", "constraint", "UPDATE projects"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks %q", leak)
		}
	}
	var name string
	var parent sql.NullInt64
	if err := database.QueryRow(`SELECT name, parent_project_id FROM projects WHERE id = ?`, a.ID).Scan(&name, &parent); err != nil {
		t.Fatal(err)
	}
	if name != "Fictional Atlas" || parent.Valid {
		t.Fatalf("refused edit changed the row: %q %v", name, parent)
	}
}

func TestEditProjectValidParentStillSavesAndRedirects(t *testing.T) {
	h, _ := cycleHandler(t)
	a := cycleProject(t, h, "Fictional Atlas", 0)
	other := cycleProject(t, h, "Orbit Gamma", 0)
	rec := serve(h, http.MethodPost, fmt.Sprintf("/projects/%d/edit", a.ID), url.Values{
		"name": {"Fictional Atlas"}, "parent_project_id": {fmt.Sprint(other.ID)}, "status": {"active"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	got, _ := h.Service.Repo.ProjectByID(a.ID)
	if got.ParentProjectID != other.ID {
		t.Fatalf("parent = %d", got.ParentProjectID)
	}
}

func parentSelectOptions(body string) string {
	start := strings.Index(body, `name="parent_project_id"`)
	end := strings.Index(body[start:], "</select>")
	return body[start : start+end]
}

func TestEditFormOmitsOwnSubProjectsFromParentPicker(t *testing.T) {
	h, _ := cycleHandler(t)
	a := cycleProject(t, h, "Fictional Atlas", 0)
	b := cycleProject(t, h, "Orbit Alpha", a.ID)
	c := cycleProject(t, h, "Orbit Beta", b.ID)
	other := cycleProject(t, h, "Orbit Gamma", 0)

	rec := serve(h, http.MethodGet, fmt.Sprintf("/projects/%d/edit", a.ID), nil)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	opts := parentSelectOptions(rec.Body.String())
	for _, gone := range []int64{a.ID, b.ID, c.ID} {
		if strings.Contains(opts, fmt.Sprintf(`value="%d"`, gone)) {
			t.Errorf("picker still offers project %d", gone)
		}
	}
	if !strings.Contains(opts, fmt.Sprintf(`value="%d"`, other.ID)) {
		t.Errorf("picker lost the unrelated project")
	}

	// A project whose current parent is (already) one of its descendants keeps
	// that parent in the picker so the form never silently changes data.
	h2, database := cycleHandler(t)
	x := cycleProject(t, h2, "Fictional Atlas", 0)
	y := cycleProject(t, h2, "Orbit Alpha", x.ID)
	if _, err := database.Exec(`UPDATE projects SET parent_project_id = ? WHERE id = ?`, y.ID, x.ID); err != nil {
		t.Fatal(err)
	}
	rec = serve(h2, http.MethodGet, fmt.Sprintf("/projects/%d/edit", x.ID), nil)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(parentSelectOptions(rec.Body.String()), fmt.Sprintf(`value="%d" selected`, y.ID)) {
		t.Errorf("current parent not kept selected")
	}
}

func TestProjectsListShowsBothProjectsOfAParentCycle(t *testing.T) {
	h, database := cycleHandler(t)
	a := cycleProject(t, h, "Fictional Atlas", 0)
	b := cycleProject(t, h, "Orbit Alpha", a.ID)
	if _, err := database.Exec(`UPDATE projects SET parent_project_id = ? WHERE id = ?`, b.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	body := getProjects(t, h, "/projects")
	for _, name := range []string{"Fictional Atlas", "Orbit Alpha"} {
		if !strings.Contains(body, name) {
			t.Errorf("/projects hides %q", name)
		}
	}
}
