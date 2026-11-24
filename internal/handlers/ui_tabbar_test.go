package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// tabBarMarkup returns the phone tab bar's wrapper (GYST-178, 13B DD-18).
func tabBarMarkup(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<div class="gyst-scope gyst-tabs">`)
	if start < 0 {
		t.Fatal("phone tab bar absent")
	}
	end := strings.Index(body[start:], "</div>")
	if end < 0 {
		t.Fatal("phone tab bar not closed")
	}
	return body[start : start+end+len("</div>")]
}

// c1, c4: every layout page renders one tab bar after the old nav and before <main>, with the
// tab its page identifier names. The table is the plan's tabTable; a page it misses fails here.
func TestPhoneTabBarMarkupAndCurrentTab(t *testing.T) {
	h := testHandler(t)
	tabs := []struct{ href, label, icon string }{
		{"/today", "Today", "home"}, {"/tasks", "Tasks", "tasks"}, {"/health", "Health", "heart"}, {"/more", "All", "all"},
	}
	cases := []struct{ path, current string }{
		{"/", "/today"}, {"/today", "/today"}, {"/checklist", "/today"},
		{"/tasks", "/tasks"}, {"/tasks/new", "/tasks"},
		{"/health", "/health"}, {"/health/nutrition", "/health"}, {"/health/progress", "/health"}, {"/health/training", "/health"},
		{"/health/plan", "/health"}, {"/health/more", "/health"}, {"/health/foods", "/health"}, {"/health/profile", "/health"},
		{"/projects", "/tasks"}, {"/projects/new", "/tasks"}, {"/roadmap", "/more"}, {"/goals", "/more"},
		{"/courses", "/more"}, {"/learn", "/more"}, {"/more", "/more"}, {"/monthly", "/more"}, {"/reminders", "/more"},
		{"/analytics", "/more"}, {"/archive", "/more"}, {"/apps", "/more"}, {"/plan-import", "/more"}, {"/settings", "/more"},
	}
	for _, tt := range cases {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if w.Code == http.StatusFound || w.Code == http.StatusSeeOther || w.Code == http.StatusMovedPermanently {
			continue // a redirect page renders nothing of its own
		}
		if w.Code != 200 {
			t.Errorf("%s: %d", tt.path, w.Code)
			continue
		}
		body := w.Body.String()
		if strings.Count(body, `class="gyst-scope gyst-tabs"`) != 1 {
			t.Errorf("%s: want one div.gyst-tabs", tt.path)
			continue
		}
		oldNav, bar, main := strings.Index(body, `class="gyst-scope gyst-primary"`), strings.Index(body, `class="gyst-scope gyst-tabs"`), strings.Index(body, `<main id="main-content"`)
		if !(oldNav >= 0 && oldNav < bar && bar < main) {
			t.Errorf("%s: the tab bar must stand after nav.gyst-primary and before <main> (keyboard order: top row, tabs, +, content)", tt.path)
		}
		bm := tabBarMarkup(t, body)
		links := regexp.MustCompile(`<a href="([^"]+)"[^>]*>.*?#i-(\w+)"/></svg><span>([^<]+)</span></a>`).FindAllStringSubmatch(bm, -1)
		if len(links) != len(tabs) {
			t.Errorf("%s: %d tab links, want 4: %s", tt.path, len(links), bm)
			continue
		}
		for i, tab := range tabs {
			if links[i][1] != tab.href || links[i][2] != tab.icon || links[i][3] != tab.label {
				t.Errorf("%s: tab %d = %v, want %v", tt.path, i, links[i][1:], tab)
			}
		}
		if strings.Count(bm, `aria-current="page"`) != 1 || !regexp.MustCompile(`<a href="`+tt.current+`"[^>]*aria-current="page"`).MatchString(bm) {
			t.Errorf("%s: the current tab must be %s alone: %s", tt.path, tt.current, bm)
		}
		if !strings.Contains(bm, `<button type="button" class="gyst-plus" data-command-open data-nav-add aria-label="Add a task">`) {
			t.Errorf("%s: the + lost its attributes", tt.path)
		}
	}
}

// c6: /galaxy keeps the bytes it had: no tab bar, no nav.js, no new stylesheet, the old meta tag.
func TestGalaxyIsSentNoTabBar(t *testing.T) {
	h := testHandler(t)
	body := cfgGet(t, h, "/galaxy")
	// GYST-187: All pages adds nothing to /galaxy: no launcher markup, no All pages label, the old title and the label More.
	for _, banned := range []string{"gyst-tabs", "gyst-plus", "nav.js", "ui-nav.css", "data-light", "data-dark", "icons.svg", "All pages", "gyst-launch", "gyst-tile"} {
		if strings.Contains(body, banned) {
			t.Errorf("/galaxy contains %q", banned)
		}
	}
	if strings.Count(body, `<meta name="theme-color"`) != 1 || !strings.Contains(body, `<meta name="theme-color" content="#f8fafc">`) {
		t.Error("/galaxy must carry exactly master's theme-color meta")
	}
	if strings.Count(body, `aria-label="Primary"`) != 1 {
		t.Error("/galaxy must keep its one five-tab bar")
	}
	if !strings.Contains(body, "<span>More</span>") || !strings.Contains(body, "<title>GYST · Plan. Track. Build. Grow.</title>") {
		t.Error("/galaxy must keep the label More and its old title")
	}
}

// c3: the + opens Capture through the launcher's own attributes; the launcher is not edited.
func TestPlusUsesLauncherAttributes(t *testing.T) {
	js := readStatic(t, "js/nav.js")
	sel := regexp.MustCompile(`querySelector(?:All)?\(\s*'([^']+)'|closest\(\s*["']([^"']+)["']`).FindAllStringSubmatch(js, -1)
	var got []string
	for _, m := range sel {
		got = append(got, m[1]+m[2])
	}
	// GYST-187 widened this list on purpose: the All pages search reads its own data-launch-* hooks and tiles. The +
	// still reaches the launcher only through its two attributes, and nothing below calls a launcher function.
	want := []string{"[data-launch-filter]", ".gyst-launch", "[data-launch-groups]", "[data-launch-results]", "[data-launch-list]", ".gyst-launch-sec", "[data-launch-count]", "[data-launch-empty]", "[data-launch-query]", "[data-launch-live]", ".gyst-tile", "a", "[data-nav-add]", `[data-launcher-mode="capture"]`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("nav.js selectors = %v, want %v", got, want)
	}
	if regexp.MustCompile(`\b(open|close|setMode|launcher)\w*\(`).MatchString(strings.ReplaceAll(js, "closest(", "")) {
		t.Error("nav.js must not call a launcher function")
	}
	layout := readTemplate(t, "layout.html")
	nav := `{{if ne .Page "galaxy"}}<script defer src="/static/js/nav.js?v=nav-4"></script>{{end}}`
	i, j := strings.Index(layout, "launcher.js"), strings.Index(layout, nav)
	if j < 0 || i < 0 || j < i || strings.Count(layout, "nav.js") != 1 {
		t.Error("layout.html must load nav.js once, after launcher.js, and not on /galaxy")
	}
}

// c4 (GYST-178, rewritten by GYST-187): every page listed on All pages is two taps away (All, then the tile). The four
// create forms are not tiles (spec a1 section 11); they stay three taps away (All, the list, its New button), so each
// list page must still link its form, and each form must still answer, and the command bar still finds them (search_test.go).
func TestMoreListReachesFoodsAndProfile(t *testing.T) {
	h := testHandler(t)
	body := cfgGet(t, h, "/more")
	for _, href := range []string{"/health/foods", "/health/profile", "/health/checklist", "/settings", "/projects", "/galaxy", "/roadmap", "/goals", "/courses", "/health/nutrition", "/health/training", "/health/progress", "/health/plan", "/analytics", "/plan-import", "/archive", "/apps", "/reminders", "/monthly"} {
		if !strings.Contains(body, `href="`+href+`"`) {
			t.Errorf("/more does not link %s", href)
		}
		cfgGet(t, h, href)
	}
	for list, form := range map[string]string{"/projects": "/projects/new", "/courses": "/courses/new", "/roadmap": "/roadmaps/new", "/goals": "/goals/new"} {
		if !strings.Contains(cfgGet(t, h, list), `href="`+form+`"`) {
			t.Errorf("%s no longer links its form %s: the form would be four taps away", list, form)
		}
		cfgGet(t, h, form)
		if strings.Contains(body, `href="`+form+`"`) {
			t.Errorf("/more draws the form %s as a tile; forms are reached from their list", form)
		}
	}
	if cfgGet(t, h, "/checklist") == "" {
		t.Error("/checklist (Today's alias) stopped answering")
	}
}

// c5: the browser bar colour is the page canvas, in both themes, on layout pages; the manifest
// has the light one. login.html keeps its own meta, and Galaxy its own (TestGalaxyIsSentNoTabBar).
func TestBrowserBarColourEqualsCanvas(t *testing.T) {
	css := readStatic(t, "css/ui-accents.css")
	canvas := regexp.MustCompile(`--vn-canvas: (#[0-9A-Fa-f]{6})`)
	light := canvas.FindStringSubmatch(css)
	if light == nil {
		t.Fatal("no light canvas in ui-accents.css")
	}
	darkBlock := css[strings.Index(css, "\n[data-gyst-theme=\"dark\"] {"):]
	dark := canvas.FindStringSubmatch(darkBlock)
	if dark == nil {
		t.Fatal("no dark canvas in ui-accents.css")
	}
	h := testHandler(t)
	for _, path := range []string{"/today", "/health", "/settings"} {
		body := cfgGet(t, h, path)
		want := `<meta name="theme-color" content="` + light[1] + `" data-light="` + light[1] + `" data-dark="` + dark[1] + `">`
		if !strings.EqualFold(body[strings.Index(body, `<meta name="theme-color"`):][:len(want)], want) {
			t.Errorf("%s: theme-color meta is not %s", path, want)
		}
	}
	var manifest map[string]any
	if err := json.Unmarshal([]byte(readStatic(t, "manifest.webmanifest")), &manifest); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"theme_color", "background_color"} {
		if v, _ := manifest[k].(string); !strings.EqualFold(v, light[1]) {
			t.Errorf("manifest %s = %q, want %s", k, v, light[1])
		}
	}
	// theme.js keeps today's literals for a page that names no colour (Galaxy).
	js := readStatic(t, "js/theme.js")
	if !strings.Contains(js, `meta.getAttribute("data-dark") || "#0f172a"`) || !strings.Contains(js, `meta.getAttribute("data-light") || "#f8fafc"`) {
		t.Error("theme.js must fall back to #0f172a and #f8fafc when the meta names no colour")
	}
}

// The files the plan says stay as master has them are not edited here.
func TestGalaxyAndLauncherFilesUntouched(t *testing.T) {
	if _, err := os.Stat("../../web/static/js/nav.js"); err != nil {
		t.Fatal(err)
	}
	css := readStatic(t, "css/ui-nav.css")
	if !strings.Contains(css, "body:not(.galaxy-page) nav.gyst-primary { display: none; }") {
		t.Error("the old bar must be hidden only off Galaxy, so a later change to the stylesheet's link cannot hide Galaxy's bar")
	}
}

// Codex P3: between 1023 and 1024 px (browser zoom) neither max-width: 1023px nor min-width: 1024px matches, so a phone
// rule written as max-width would leave both the bar and the old in-flow nav. Every phone query in ui-nav.css is the exact
// complement of ui-shell.css's desktop query instead.
func TestPhoneQueriesAreTheComplementOfTheDesktopQuery(t *testing.T) {
	if !strings.Contains(readStatic(t, "css/ui-shell.css"), "@media (min-width: 1024px)") {
		t.Fatal("ui-shell.css no longer has the desktop query this test pairs with")
	}
	css := readStatic(t, "css/ui-nav.css")
	for _, m := range regexp.MustCompile(`@media [^{]*`).FindAllString(css, -1) {
		if strings.Contains(m, "max-width: 1023") {
			t.Errorf("ui-nav.css has %q; use the complement of (min-width: 1024px)", strings.TrimSpace(m))
		}
	}
	if !strings.Contains(css, "@media not all and (min-width: 1024px) {") || !strings.Contains(css, "@media (not (min-width: 1024px)) and (hover: hover) {") {
		t.Error("ui-nav.css lost its phone queries")
	}
}
