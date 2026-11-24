package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func hubBody(t *testing.T, h *Handler, path string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != 200 {
		t.Fatalf("%s: %d", path, rec.Code)
	}
	return rec.Body.String()
}

// GYST-186 c1, c3: the Tasks hub switch shows on Tasks and Projects (any filter), straight after the
// page head, with three links in order and exactly one current; nowhere else.
func TestTasksHubSwitch(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Fictional Orbit", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	nav := regexp.MustCompile(`(?s)<nav class="gyst-switcher gyst-hub" aria-label="Views" data-sec="(plan|work)">(.*?)</nav>`)
	for _, tt := range []struct{ path, sec, current string }{
		{"/tasks", "plan", "/tasks"}, {"/tasks?status=overdue", "plan", "/tasks"},
		{"/projects", "work", "/projects"}, {"/projects?status=completed", "work", "/projects"},
	} {
		body := hubBody(t, h, tt.path)
		if strings.Count(body, "gyst-hub") != 1 {
			t.Fatalf("%s: want one hub switch", tt.path)
		}
		m := nav.FindStringSubmatch(body)
		if m == nil || m[1] != tt.sec {
			t.Fatalf("%s: hub missing or wrong section: %v", tt.path, m)
		}
		hrefs := regexp.MustCompile(`<a href="([^"]+)"`).FindAllStringSubmatch(m[2], -1)
		if len(hrefs) != 3 || hrefs[0][1] != "/tasks" || hrefs[1][1] != "/projects" || hrefs[2][1] != "/galaxy" {
			t.Fatalf("%s: links %v", tt.path, hrefs)
		}
		if strings.Count(m[2], `aria-current="page"`) != 1 || !strings.Contains(m[2], `<a href="`+tt.current+`" aria-current="page">`) {
			t.Fatalf("%s: wrong current tab: %s", tt.path, m[2])
		}
		// directly after the page head, before the filters and the list
		head := `</header>`
		if strings.HasPrefix(tt.path, "/tasks") {
			head = "New task</a>\n</div>"
		}
		if !strings.Contains(body, head+"\n<nav class=\"gyst-switcher gyst-hub\"") {
			t.Fatalf("%s: the hub must follow the page head", tt.path)
		}
	}
	for _, path := range []string{"/tasks/new", "/projects/new", fmt.Sprintf("/projects/%d", project.ID), "/roadmap", "/roadmaps/new", "/goals", "/galaxy"} {
		if strings.Contains(hubBody(t, h, path), "gyst-hub") {
			t.Fatalf("%s must not show the hub switch", path)
		}
	}
	if tasks := hubBody(t, h, "/tasks"); strings.Contains(tasks, "Task library") || strings.Contains(tasks, "Every active task") {
		t.Fatal("the Tasks head keeps no kicker or sub-line")
	}
	if !strings.Contains(hubBody(t, h, "/projects"), "Make room for what you want to build.") {
		t.Fatal("the Projects caption stays")
	}
}

// c4: Projects pages light the Tasks tab; Roadmap stays under All.
func TestPhoneTabProjectsIsTasks(t *testing.T) {
	for page, want := range map[string]string{"tasks": "tasks", "projects": "tasks", "roadmap": "all", "galaxy": "all", "more": "all", "today": "today", "health": "health"} {
		if got := phoneTab(page); got != want {
			t.Fatalf("phoneTab(%q) = %q, want %q", page, got, want)
		}
	}
	h := testHandler(t)
	project := models.Project{Name: "Fictional Orbit", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/projects", "/projects/new", fmt.Sprintf("/projects/%d", project.ID), fmt.Sprintf("/projects/%d/edit", project.ID)} {
		bm := tabBarMarkup(t, hubBody(t, h, path))
		if !regexp.MustCompile(`<a href="/tasks"[^>]*aria-current="page"`).MatchString(bm) || strings.Count(bm, `aria-current="page"`) != 1 {
			t.Fatalf("%s: the Tasks tab must be the only current tab: %s", path, bm)
		}
	}
}

// c5: the old in-page bar is gone from Projects and Roadmaps, but its template and rules stay for GYST-177.
func TestOldProjectsSubnavOnlyInTemplateAndCSS(t *testing.T) {
	h := testHandler(t)
	for _, path := range []string{"/projects", "/roadmap"} {
		body := hubBody(t, h, path)
		if strings.Contains(body, `class="gyst-scope gyst-segments gyst-subnav gyst-projects-subnav"`) {
			t.Fatalf("%s still renders the old bar", path)
		}
	}
	tpl, _ := os.ReadFile("../../web/templates/project_components.html")
	css, _ := os.ReadFile("../../web/static/css/ui-shell.css")
	if !strings.Contains(string(tpl), `{{define "gyst.projects-subnav"}}`) || !strings.Contains(string(css), ".gyst-projects-subnav") {
		t.Fatal("GYST-177 needs the template and its CSS rules to stay")
	}
}

// c7: ui-nav.css is never linked on Galaxy, so no hub rule can reach it.
func TestGalaxyDoesNotLoadNavCSS(t *testing.T) {
	if strings.Contains(hubBody(t, testHandler(t), "/galaxy"), "ui-nav.css") {
		t.Fatal("/galaxy must not link ui-nav.css")
	}
}

// c3: the hub's icon rules use the section text colours and never accent or info.
func TestHubCSSUsesSectionColours(t *testing.T) {
	b, _ := os.ReadFile("../../web/static/css/ui-nav.css")
	css := string(b)
	for _, want := range []string{
		`nav.gyst-hub[data-sec="plan"] > a[aria-current="page"] .vn-icon { color: var(--vn-s-plan-t); }`,
		`nav.gyst-hub[data-sec="work"] > a[aria-current="page"] .vn-icon { color: var(--vn-s-work-t); }`,
		`nav.gyst-switcher.gyst-hub { max-width: 380px;`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("ui-nav.css lacks %q", want)
		}
	}
}
