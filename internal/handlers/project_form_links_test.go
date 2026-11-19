package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// linkRow reads the three link columns of one project.
func linkRow(t *testing.T, h *Handler, id int64) (parent, roadmap, dep int64) {
	t.Helper()
	p, err := h.Service.Repo.ProjectByID(id)
	if err != nil {
		t.Fatalf("project %d: %v", id, err)
	}
	return p.ParentProjectID, p.RoadmapID, p.DependsOnProjectID
}

// selectBlock returns the HTML of one named <select>.
func selectBlock(t *testing.T, body, name string) string {
	t.Helper()
	re := regexp.MustCompile(`(?s)<select name="` + name + `">.*?</select>`)
	m := re.FindString(body)
	if m == "" {
		t.Fatalf("select %q not found", name)
	}
	return m
}

// formValues rebuilds the POST a browser would send from the rendered form:
// each select posts its selected option, or its first option if none.
func formValues(t *testing.T, body, name string) url.Values {
	t.Helper()
	v := url.Values{"name": {name}, "status": {"active"}, "project_kind": {"subproject"}, "category": {"other"}}
	opt := regexp.MustCompile(`<option value="([^"]*)"([^>]*)>`)
	for _, sel := range []string{"roadmap_id", "parent_project_id", "depends_on_project_id"} {
		opts := opt.FindAllStringSubmatch(selectBlock(t, body, sel), -1)
		posted := opts[0][1]
		for _, o := range opts {
			if strings.Contains(o[2], "selected") {
				posted = o[1]
			}
		}
		v.Set(sel, posted)
	}
	return v
}

type archivedLinks struct {
	h                *Handler
	parent, dep, kid models.Project
	stranger         models.Project
	roadmap          models.Roadmap
}

// archivedLinkFixture builds a child linked to a parent, dependency and
// roadmap that are all archived (by archived_at, or by status alone).
func archivedLinkFixture(t *testing.T, statusOnly bool) archivedLinks {
	t.Helper()
	h, database := cycleHandler(t)
	f := archivedLinks{h: h}
	f.parent = cycleProject(t, h, "Fictional Parent Atlas", 0)
	f.dep = cycleProject(t, h, "Fictional Dependency Dock", 0)
	f.stranger = cycleProject(t, h, "Fictional Archived Stranger", 0)
	f.roadmap = models.Roadmap{Name: "Fictional Roadmap Row", Status: "active", Privacy: "private"}
	if err := h.Service.Repo.CreateRoadmap(&f.roadmap); err != nil {
		t.Fatal(err)
	}
	f.kid = models.Project{Name: "Fictional Child Orbit", ProjectKind: "subproject", Status: "active",
		ParentProjectID: f.parent.ID, RoadmapID: f.roadmap.ID, DependsOnProjectID: f.dep.ID, AutoUnlock: true}
	if err := h.Service.Repo.CreateProject(&f.kid); err != nil {
		t.Fatal(err)
	}
	if statusOnly {
		if _, err := database.Exec(`UPDATE projects SET status='archived' WHERE id IN (?, ?, ?)`, f.parent.ID, f.dep.ID, f.stranger.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`UPDATE roadmaps SET status='archived' WHERE id=?`, f.roadmap.ID); err != nil {
			t.Fatal(err)
		}
		return f
	}
	for _, id := range []int64{f.parent.ID, f.dep.ID, f.stranger.ID} {
		if err := h.Service.Repo.ArchiveProject(id); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.Service.Repo.ArchiveRoadmap(f.roadmap.ID); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestEditProjectKeepsArchivedLinks(t *testing.T) {
	for _, statusOnly := range []bool{false, true} {
		t.Run(fmt.Sprintf("statusOnly=%v", statusOnly), func(t *testing.T) {
			f := archivedLinkFixture(t, statusOnly)
			path := fmt.Sprintf("/projects/%d/edit", f.kid.ID)
			rec := serve(f.h, http.MethodGet, path, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET status %d", rec.Code)
			}
			body := rec.Body.String()
			for sel, want := range map[string]string{
				"roadmap_id":            fmt.Sprintf(`<option value="%d" selected>Fictional Roadmap Row (archived)</option>`, f.roadmap.ID),
				"parent_project_id":     fmt.Sprintf(`<option value="%d" selected>Fictional Parent Atlas (archived)</option>`, f.parent.ID),
				"depends_on_project_id": fmt.Sprintf(`<option value="%d" selected>Fictional Dependency Dock (archived)</option>`, f.dep.ID),
			} {
				if !strings.Contains(selectBlock(t, body, sel), want) {
					t.Errorf("%s select missing %q", sel, want)
				}
			}
			if strings.Contains(body, "Fictional Archived Stranger") {
				t.Error("unlinked archived project is offered")
			}
			self := fmt.Sprintf(`value="%d"`, f.kid.ID)
			if strings.Contains(selectBlock(t, body, "parent_project_id"), self) ||
				strings.Contains(selectBlock(t, body, "depends_on_project_id"), self) {
				t.Error("project offered as its own parent or dependency")
			}

			// A rename posts the rendered values and must keep every link.
			rec = serve(f.h, http.MethodPost, path, formValues(t, body, "Fictional Child Renamed"))
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("POST status %d: %s", rec.Code, rec.Body.String())
			}
			p, r, d := linkRow(t, f.h, f.kid.ID)
			if p != f.parent.ID || r != f.roadmap.ID || d != f.dep.ID {
				t.Fatalf("links after save = %d/%d/%d, want %d/%d/%d", p, r, d, f.parent.ID, f.roadmap.ID, f.dep.ID)
			}
		})
	}
}

func TestEditProjectArchivedLinkStillClearsOnNone(t *testing.T) {
	f := archivedLinkFixture(t, false)
	rec := serve(f.h, http.MethodPost, fmt.Sprintf("/projects/%d/edit", f.kid.ID), url.Values{
		"name": {"Fictional Child Orbit"}, "status": {"active"}, "project_kind": {"subproject"},
		"parent_project_id": {""}, "roadmap_id": {""}, "depends_on_project_id": {""},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	// Unassigned is adopted into the dashboard roadmap in the same write (GYST-VN-028).
	dash, err := f.h.Service.Repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatalf("dashboard roadmap: %v", err)
	}
	if p, r, d := linkRow(t, f.h, f.kid.ID); p != 0 || r != dash.ID || d != 0 {
		t.Fatalf("links after None = %d/%d/%d, want 0/%d/0", p, r, d, dash.ID)
	}
}

// With no active roadmap there is nothing to adopt into: None leaves it NULL.
func TestEditProjectNoneStaysUnassignedWithoutActiveRoadmap(t *testing.T) {
	h, database := cycleHandler(t)
	if _, err := database.Exec(`UPDATE roadmaps SET status='archived', archived_at=CURRENT_TIMESTAMP`); err != nil {
		t.Fatal(err)
	}
	parent := cycleProject(t, h, "Fictional Parent Atlas", 0)
	kid := cycleProject(t, h, "Fictional Child Orbit", parent.ID)
	rec := serve(h, http.MethodPost, fmt.Sprintf("/projects/%d/edit", kid.ID), url.Values{
		"name": {"Fictional Child Orbit"}, "status": {"active"}, "project_kind": {"subproject"},
		"parent_project_id": {""}, "roadmap_id": {""}, "depends_on_project_id": {""},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	if p, r, d := linkRow(t, h, kid.ID); p != 0 || r != 0 || d != 0 {
		t.Fatalf("links after None = %d/%d/%d, want all cleared", p, r, d)
	}
}

func TestEditProjectArchivedParentKeepsPrivacyInheritance(t *testing.T) {
	h, database := cycleHandler(t)
	parent := cycleProject(t, h, "Fictional Private Parent", 0)
	child := cycleProject(t, h, "Fictional Private Child", parent.ID)
	if _, err := database.Exec(`INSERT INTO roadmap_project_meta (project_id, is_private) VALUES (?, 1)`, parent.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.Service.Repo.ArchiveProject(parent.ID); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/projects/%d/edit", child.ID)
	body := serve(h, http.MethodGet, path, nil).Body.String()
	if rec := serve(h, http.MethodPost, path, formValues(t, body, "Fictional Private Child Renamed")); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	priv, err := h.Service.Repo.EffectiveProjectPrivacy()
	if err != nil {
		t.Fatal(err)
	}
	if !priv[child.ID].Hidden() {
		t.Fatal("child lost inherited privacy after an unchanged edit")
	}
	pub, err := h.Service.ExportRoadmap(true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pub, "Fictional Private Child") {
		t.Fatal("public export shows the child of a private archived parent")
	}
}

func TestEditProjectWithoutLinksOffersNoArchivedOption(t *testing.T) {
	h, _ := cycleHandler(t)
	p := cycleProject(t, h, "Fictional Loner", 0)
	if strings.Contains(serve(h, http.MethodGet, fmt.Sprintf("/projects/%d/edit", p.ID), nil).Body.String(), "(archived)") {
		t.Error("archived option shown for a project with no links")
	}
	if strings.Contains(serve(h, http.MethodGet, "/projects/new", nil).Body.String(), "(archived)") {
		t.Error("new-project form shows an archived option")
	}
}
