package handlers

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func primaryMarkup(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<nav aria-label="Primary"`)
	if start < 0 {
		t.Fatal("primary nav absent")
	}
	end := strings.Index(body[start:], "</nav>")
	if end < 0 {
		t.Fatal("primary nav not closed")
	}
	return body[start : start+end]
}
func TestUIShellFiveLinksAndNestedSelection(t *testing.T) {
	h := testHandler(t)
	for _, tt := range []struct{ path, current string }{{"/", "today"}, {"/today", "today"}, {"/projects", "projects"}, {"/projects/new", "projects"}, {"/galaxy", "projects"}, {"/roadmap", "projects"}, {"/health", "health"}, {"/health/plan", "health"}, {"/learn", "learn"}, {"/courses", "learn"}, {"/courses/new", "learn"}, {"/more", "more"}, {"/settings", "more"}, {"/tasks/new", "more"}, {"/plan-import", "more"}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if rec.Code != 200 {
			t.Fatalf("%s: %d", tt.path, rec.Code)
		}
		nav := primaryMarkup(t, rec.Body.String())
		// Inside Health and Projects the sidebar also lists the area's own
		// sections in a labelled group (#53, approved design C5; #124; Health's
		// four since GYST-174, none current on /health/plan). The five primary
		// destinations and their single current item are checked without it.
		for _, group := range []string{`<div class="gyst-health-sub"`, `<div class="gyst-projects-sub"`} {
			if i := strings.Index(nav, group); i >= 0 {
				end := strings.Index(nav[i:], "</div>")
				nav = nav[:i] + nav[i+end+len("</div>"):]
			}
		}
		if strings.Count(nav, "<a ") != 5 || strings.Count(nav, `aria-current="page"`) != 1 {
			t.Fatalf("invalid nav: %s", nav)
		}
		for _, path := range []string{"today", "projects", "health", "learn", "more"} {
			if !strings.Contains(nav, `href="/`+path+`"`) {
				t.Fatalf("missing destination %s", path)
			}
		}
		if !strings.Contains(nav, `href="/`+tt.current+`" aria-current="page"`) {
			t.Fatalf("wrong active state at %s: %s", tt.path, nav)
		}
		if strings.Count(nav, "<svg") != 5 {
			t.Fatal("navigation must use five SVG icons")
		}
		for _, hook := range []string{`data-command-open`, `data-quick-task`, `href="#main-content"`} {
			if !strings.Contains(rec.Body.String(), hook) {
				t.Fatalf("%s lost action %s", tt.path, hook)
			}
		}
	}
}
func TestMoreMethodAndReachability(t *testing.T) {
	h := testHandler(t)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, "/more", nil))
		if rec.Code != 405 || rec.Header().Get("Allow") != "GET" {
			t.Fatalf("More accepted %s: %d", method, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/more/not-real", nil))
	if rec.Code != 404 {
		t.Fatal("More accepted unknown subroute")
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/more", nil))
	for _, path := range []string{"/health/checklist", "/health/foods", "/health/profile", "/health/more", "/projects", "/courses", "/health", "/health/nutrition", "/health/training", "/health/progress", "/health/plan", "/today", "/tasks", "/reminders", "/monthly", "/galaxy", "/roadmap", "/goals", "/analytics", "/plan-import", "/archive", "/apps", "/settings"} {
		if !strings.Contains(rec.Body.String(), `href="`+path+`"`) {
			t.Fatalf("More missing %s", path)
		}
		dest := httptest.NewRecorder()
		h.ServeHTTP(dest, httptest.NewRequest(http.MethodGet, path, nil))
		if dest.Code != 200 {
			t.Fatalf("More links to broken destination %s", path)
		}
	}
	denied := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/more", nil))
	if denied.Code != 303 || !strings.HasPrefix(denied.Header().Get("Location"), "/login") {
		t.Fatalf("More bypassed authentication: %d", denied.Code)
	}
}

// Creation belongs to the corresponding page, while Menu covers destinations.
func TestNavigationCleanupAndContextActions(t *testing.T) {
	h := testHandler(t)
	for _, tc := range []struct{ page, action string }{
		{"/today", "/tasks/new"}, {"/projects", "/projects/new"}, {"/courses", "/courses/new"}, {"/learn", "/courses/new"}, {"/more", "/plan-import"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.page, nil))
		body := rec.Body.String()
		start := strings.Index(body, `<header class="gyst-scope gyst-topbar">`)
		end := strings.Index(body[start:], "</header>")
		header := body[start : start+end]
		if strings.Contains(header, "gyst-add-menu") || strings.Contains(header, "gyst-xp") || !strings.Contains(header, "data-command-open") {
			t.Fatalf("top bar not simplified: %s", header)
		}
		mainStart := strings.Index(body, `<main `)
		mainEnd := strings.Index(body[mainStart:], "</main>")
		if !strings.Contains(body[mainStart:mainStart+mainEnd], `href="`+tc.action+`"`) {
			t.Fatalf("%s lost contextual action %s", tc.page, tc.action)
		}
		if !strings.Contains(primaryMarkup(t, body), "<span>"+allPagesLabel+"</span>") {
			t.Fatal("More label missing")
		}
	}
}

// markupBetween returns the element that starts with open, up to its close tag.
func markupBetween(body, open, close string) string {
	start := strings.Index(body, open)
	if start < 0 {
		return ""
	}
	end := strings.Index(body[start:], close)
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}

func TestProjectsArea(t *testing.T) {
	for page, want := range map[string]string{"projects": "projects", "galaxy": "galaxy", "roadmap": "roadmaps", "roadmaps": "roadmaps", "health": "projects", "": "projects"} {
		if got := projectsArea(page); got != want {
			t.Fatalf("projectsArea(%q) = %q, want %q", page, got, want)
		}
	}
}

// GYST-VN-039 (#124): Projects, Galaxy and Roadmaps are one tap apart. The
// in-page control is on the Galaxy page only (GYST-177, after GYST-186 moved
// Projects and Roadmaps to the Tasks hub). The sidebar group is on every Projects page.
func TestProjectsAreaSubNavigation(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Fictional Orbit", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		path, current string
		inPage        bool
	}{
		{"/projects", "/projects", false},
		{"/projects?status=completed", "/projects", false},
		{"/roadmap", "/roadmap", false},
		{fmt.Sprintf("/projects/%d", project.ID), "/projects", false},
		{"/projects/new", "/projects", false},
		{"/roadmaps/new", "/roadmap", false},
		{"/galaxy", "/galaxy", true},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if rec.Code != 200 {
			t.Fatalf("%s: %d", tt.path, rec.Code)
		}
		body := rec.Body.String()
		groups := map[string]string{"sidebar": markupBetween(body, `<div class="gyst-projects-sub"`, "</div>")}
		if tt.inPage {
			groups["in-page"] = markupBetween(body, `<nav class="gyst-scope gyst-segments gyst-subnav gyst-projects-subnav" aria-label="Projects areas">`, "</nav>")
		} else if strings.Contains(body, "gyst-projects-subnav") {
			t.Fatalf("%s must not render the in-page Projects control", tt.path)
		}
		for name, group := range groups {
			if group == "" {
				t.Fatalf("%s: %s Projects areas group absent", tt.path, name)
			}
			if strings.Count(group, "<a ") != 3 || strings.Count(group, `aria-current="page"`) != 1 {
				t.Fatalf("%s: %s group needs three links and one current: %s", tt.path, name, group)
			}
			for _, href := range []string{"/projects", "/galaxy", "/roadmap"} {
				if !strings.Contains(group, `href="`+href+`"`) {
					t.Fatalf("%s: %s group missing %s", tt.path, name, href)
				}
			}
			if !strings.Contains(group, `href="`+tt.current+`" aria-current="page"`) {
				t.Fatalf("%s: %s group marks the wrong area: %s", tt.path, name, group)
			}
			if strings.Contains(group, "Development") {
				t.Fatalf("%s: %s group shows Development before it has a destination", tt.path, name)
			}
		}
		if strings.Count(body, `aria-label="Projects areas"`) != len(groups) {
			t.Fatalf("%s: want %d Projects areas groups", tt.path, len(groups))
		}
	}
	// The status filter is no longer a second segmented control. The launcher
	// renders its own segments on every page, so count only what /projects adds.
	base := httptest.NewRecorder()
	h.ServeHTTP(base, httptest.NewRequest(http.MethodGet, "/today", nil))
	shellSegments := strings.Count(base.Body.String(), `class="gyst-segment"`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/projects", nil))
	if !strings.Contains(rec.Body.String(), `<nav class="projects-roadmaps" aria-label="Project status">`) || strings.Count(rec.Body.String(), `class="gyst-segment"`)-shellSegments != 0 {
		t.Fatal("projects status filter must use the pill links, leaving no segmented control (GYST-186 moved the switch to the Tasks hub)")
	}
	for _, path := range []string{"/health", "/today", "/courses"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if strings.Contains(rec.Body.String(), "Projects areas") {
			t.Fatalf("%s must not render the Projects areas navigation", path)
		}
	}
}

// GYST-177: the Galaxy bar is static and outside the stage; the stage gives back its height; scene code is untouched.
func TestGalaxyProjectsBarRulesAndSceneFiles(t *testing.T) {
	css, _ := os.ReadFile("../../web/static/css/ui-shell.css")
	s := string(css)
	if !strings.Contains(s, "--gyst-subnav-clearance: 0px") {
		t.Fatal("root clearance must default to 0px so other pages are unchanged")
	}
	if strings.Count(s, "--gyst-subnav-clearance: 52px") != 1 || !strings.Contains(s, ":root body.galaxy-page { --gyst-subnav-clearance: 52px; }") {
		t.Fatal("52px clearance belongs only under body.galaxy-page")
	}
	// the 52px rule must sit inside the below-1024px query
	i := strings.Index(s, "--gyst-subnav-clearance: 52px")
	q := strings.LastIndex(s[:i], "@media")
	if !strings.HasPrefix(s[q:], "@media (max-width: 1023px)") {
		t.Fatalf("clearance is outside the phone and tablet query: %q", s[q:q+40])
	}
	if !strings.Contains(s, ":root body.galaxy-page .gyst-projects-subnav { position: static; margin-inline: max(16px,var(--gyst-safe-left)) max(16px,var(--gyst-safe-right)); }") {
		t.Fatal("Galaxy bar must be static with the safe-area gutter")
	}
	// The screen height in ui-galaxy.css is what actually sizes the stage on phone and tablet.
	gcss, _ := os.ReadFile("../../web/static/css/ui-galaxy.css")
	if !strings.Contains(string(gcss), "max(calc(420px - var(--gyst-subnav-clearance,0px)), calc(100svh - var(--gyst-header-clearance) - var(--gyst-nav-clearance) - var(--gyst-subnav-clearance,0px)))") {
		t.Fatal("Galaxy screen height must subtract the bar clearance")
	}
	if !strings.Contains(string(gcss), "min-height:calc(280px - var(--gyst-subnav-clearance,0px))") {
		t.Fatal("stage floor must give the bar's height back too, or the stage overflows the screen on short viewports")
	}
	if strings.Count(s, "var(--gyst-subnav-clearance)") != 2 {
		t.Fatal("both stage height lines (vh and dvh) must subtract the bar")
	}
	for _, f := range []string{"galaxy.js", "galaxy-camera.js", "galaxy-gestures.js", "app.js"} {
		got, err := os.ReadFile("../../web/static/js/" + f)
		if err != nil {
			t.Fatal(err)
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(got))
		if want := galaxySceneHashes[f]; sum != want {
			t.Errorf("%s changed (GYST-177 must not touch it): %s", f, sum)
		}
	}
}

// Hashes of master 17ece2f. A later ticket that edits one of these updates its line on purpose.
var galaxySceneHashes = map[string]string{
	"galaxy.js":          "3dd964566261729bb157c7814065973f62bb126229b47d84697c8727e298541f",
	"galaxy-camera.js":   "1560d2b97f6a34e03a3cf7a3b8212cdcadd8f70af8a83f202cd46fc3fecb05d9",
	"galaxy-gestures.js": "b874c2b7cc445fa0f074873eaa75bb196d4b888f83dea0fb0b43444142b55fda",
	"app.js":             "b85d40a2516c25f25c2e69a47d313ccc9e3092870df0c1e457f10b2c917498c6",
}
