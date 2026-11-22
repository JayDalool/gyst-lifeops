package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// GYST-174 (Issue #265, 13B DD-22): Health keeps the main tab bar on every screen and has one
// section switcher, Summary · Nutrition · Progress · Training. Plan, More and Coach are pages
// pushed from rows on Summary; they belong to no section, so nothing is marked for them.

var healthSections = []struct{ name, path, label, icon string }{
	{"summary", "/health", "Summary", "heart"},
	{"nutrition", "/health/nutrition", "Nutrition", "fork"},
	{"progress", "/health/progress", "Progress", "trend"},
	{"training", "/health/training", "Training", "gym"},
}

// healthNavHandler has a Coach provider set, so the Coach page and its row render, and one saved
// food, so the single-food page exists. It returns the pushed pages by name.
func healthNavHandler(t *testing.T) (*Handler, map[string]string) {
	t.Helper()
	h := coachHandler(t, &fakeCoach{})
	if w := cfgPost(h, "/health/foods", url.Values{"name": {"Oat bar"}, "kcal": {"190"}}, ""); w.Code != 303 {
		t.Fatalf("add a food: %d", w.Code)
	}
	foods, err := h.Service.Repo.ListHealthFoods(true)
	if err != nil || len(foods) != 1 {
		t.Fatalf("foods: %v %v", foods, err)
	}
	return h, map[string]string{
		"plan": "/health/plan", "more": "/health/more", "checklist": "/health/checklist", "foods": "/health/foods",
		"food": "/health/foods/" + idString(foods[0].ID), "profile": "/health/profile", "coach": "/health/coach",
	}
}

// healthSubMarkup is the sidebar's Health group inside the primary nav.
func healthSubMarkup(t *testing.T, body string) string {
	t.Helper()
	group := markupBetween(primaryMarkup(t, body), `<div class="gyst-health-sub"`, "</div>")
	if group == "" {
		t.Fatal("the sidebar's Health group is absent")
	}
	return group
}

var navLink = regexp.MustCompile(`<a href="([^"]+)"( aria-current="page")?>`)

// The four sections in order, and which href carries aria-current ("" for none).
func assertHealthSectionLinks(t *testing.T, where, markup, current string) {
	t.Helper()
	links := navLink.FindAllStringSubmatch(markup, -1)
	if len(links) != len(healthSections) || strings.Count(markup, "<a ") != len(healthSections) {
		t.Fatalf("%s: want the four sections, got %s", where, markup)
	}
	at := 0
	for i, s := range healthSections {
		if links[i][1] != s.path {
			t.Errorf("%s: link %d goes to %s, want %s", where, i+1, links[i][1], s.path)
		}
		if marked := links[i][2] != ""; marked != (s.path == current) {
			t.Errorf("%s: %s aria-current = %v, want %v", where, s.label, marked, s.path == current)
		}
		j := strings.Index(markup[at:], s.label)
		if j < 0 {
			t.Fatalf("%s: %q missing or out of order in %s", where, s.label, markup)
		}
		at += j
	}
}

// c1, c2: the main tab bar on every Health page, the Health-only bar on none, and exactly one
// switcher, directly under the page header, on the four section pages only.
func TestHealthKeepsMainTabBarAndHasOneSwitcher(t *testing.T) {
	h, pushed := healthNavHandler(t)
	afterHeader := regexp.MustCompile(`</header>\s*<nav class="gyst-switcher"`)
	check := func(name, path string, section bool) {
		body := cfgGet(t, h, path)
		nav := primaryMarkup(t, body)
		if i := strings.Index(nav, `<div class="gyst-health-sub"`); i >= 0 {
			nav = nav[:i] + nav[i+strings.Index(nav[i:], "</div>")+len("</div>"):]
		}
		if strings.Count(nav, "<a ") != 5 || strings.Count(nav, `aria-current="page"`) != 1 || !strings.Contains(nav, `href="/health" aria-current="page"`) {
			t.Errorf("%s: the main tab bar must have five links with Health current: %s", name, nav)
		}
		if strings.Contains(body, "hv3-tabbar") {
			t.Errorf("%s: still renders the Health-only tab bar", name)
		}
		count := strings.Count(body, `<nav class="gyst-switcher"`)
		if !section {
			if count != 0 || strings.Contains(body, "gyst-switcher") {
				t.Errorf("%s: a pushed page must have no switcher, found %d", name, count)
			}
			return
		}
		if count != 1 {
			t.Fatalf("%s: want exactly one switcher, found %d", name, count)
		}
		if !afterHeader.MatchString(body) {
			t.Errorf("%s: the switcher is not the first element after the page header", name)
		}
		sw := markupBetween(body, `<nav class="gyst-switcher"`, "</nav>")
		if !strings.Contains(sw, `aria-label="Health sections"`) {
			t.Errorf("%s: the switcher has no name", name)
		}
		assertHealthSectionLinks(t, name+" switcher", sw, path)
		for _, s := range healthSections {
			want := `<a href="` + s.path + `"`
			if s.path == path {
				want += ` aria-current="page"`
			}
			want += `><svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=r7-2#i-` + s.icon + `"/></svg><span>` + s.label + `</span></a>`
			if !strings.Contains(sw, want) {
				t.Errorf("%s: switcher lacks %s", name, want)
			}
		}
		if got := strings.Count(sw, `<svg class="vn-icon"`); got != 4 {
			t.Errorf("%s: the switcher draws %d sprite icons, want 4", name, got)
		}
	}
	for _, s := range healthSections {
		check(s.name, s.path, true)
	}
	for name, path := range pushed {
		check(name, path, false)
	}
}

// c2 (design review F3): "under the title" is under the title. On Progress the caption and the
// Manage health plan button used to sit between the title and the switcher, which put it 66 px
// lower than on the other three sections at phone width; the control jumped when Jay changed
// section. They now follow the switcher. The browser check measures the offsets.
func TestHealthProgressSwitcherSitsUnderTheTitle(t *testing.T) {
	body := cfgGet(t, testHandler(t), "/health/progress")
	main := body[strings.Index(body, "<main "):]
	header := markupBetween(main, "<header ", "</header>")
	if !strings.HasSuffix(header, `<div><p class="kicker">Health</p><h1 class="gyst-page-title">Progress</h1></div>`) {
		t.Errorf("Progress: the page header must hold the eyebrow and the title only: %s", header)
	}
	sw, button, caption := strings.Index(main, `<nav class="gyst-switcher"`), strings.Index(main, `href="/health/plan">Manage health plan</a>`), strings.Index(main, "Your recorded measurements over time.")
	if sw < 0 || button < sw || caption < sw {
		t.Errorf("Progress: the caption and the Manage health plan button must follow the switcher (switcher %d, caption %d, button %d)", sw, caption, button)
	}
}

// c4 (technical review condition b): a page outside the four sections reports no section. Nothing
// defaults to Summary or More, so no switcher tab and no sidebar item is marked for it.
func TestHealthPagesOutsideTheSectionsMarkNothing(t *testing.T) {
	for _, tc := range []struct {
		data any
		want string
	}{
		{HealthData{Page: "health"}, ""},
		{HealthData{Page: "health", HealthTab: "summary"}, "summary"},
		{nutritionData{}, "nutrition"},
		{healthProgressData{}, "progress"},
		{trainingData{HealthData: HealthData{HealthTab: "training"}}, "training"},
		{healthPlanData{}, ""}, {healthMoreData{}, ""}, {healthCoachData{}, ""},
		{healthChecklistData{}, ""}, {healthFoodsData{}, ""}, {healthProfileData{}, ""},
		{struct{ Page string }{"health"}, ""},
	} {
		if got := healthTab(tc.data); got != tc.want {
			t.Errorf("healthTab(%T) = %q, want %q", tc.data, got, tc.want)
		}
	}

	h, pushed := healthNavHandler(t)
	for name, path := range pushed {
		group := healthSubMarkup(t, cfgGet(t, h, path))
		// Condition c: with a provider set the sidebar still lists the four sections only.
		assertHealthSectionLinks(t, name+" sidebar", group, "")
		for _, gone := range []string{"/health/coach", "/health/more", "Coach", "More"} {
			if strings.Contains(group, gone) {
				t.Errorf("%s: the sidebar's Health group still has %s", name, gone)
			}
		}
	}
	for _, s := range healthSections {
		assertHealthSectionLinks(t, s.name+" sidebar", healthSubMarkup(t, cfgGet(t, h, s.path)), s.path)
	}
	// Health data that names no section marks neither navigation.
	body := renderBody(t, "health_more.html", healthMoreData{HealthData: HealthData{Page: "health"}})
	if strings.Contains(healthSubMarkup(t, body), "aria-current") || strings.Contains(body, "gyst-switcher") {
		t.Error("Health data with no section marked a section")
	}
}

// c3: Plan, More and Coach open from rows on Summary, and every pushed page links back.
func TestHealthSummaryRowsAndBackLinks(t *testing.T) {
	h, pushed := healthNavHandler(t)
	h.Service.Clock.Now = func() time.Time { return time.Date(2026, 10, 7, 15, 0, 0, 0, h.Service.Location()) }
	row := func(body, href string) string {
		return markupBetween(body, `<a class="hv3-row" href="`+href+`">`, "</a>")
	}
	body := cfgGet(t, h, "/health")
	if !strings.Contains(body, "<h1>Health</h1>") {
		t.Error("Summary's title must be Health")
	}
	if plan := row(body, "/health/plan"); !strings.Contains(plan, ">Plan<") || !strings.Contains(plan, "No plan yet") {
		t.Errorf("Summary without a cycle: Plan row = %q", plan)
	}
	if more := row(body, "/health/more"); !strings.Contains(more, ">More<") {
		t.Errorf("Summary: More row = %q", more)
	}
	if coach := row(body, "/health/coach"); !strings.Contains(coach, ">Coach<") {
		t.Errorf("Summary with a provider: Coach row = %q", coach)
	}
	// 13B DD-15 (design review F6): a row's tile icon comes from round 7's sprite, as round 7 draws it.
	for href, icon := range map[string]string{"/health/plan": "target", "/health/more": "more", "/health/coach": "sec"} {
		want := `<span class="hv3-tile sm"><svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=r7-2#i-` + icon + `"/></svg></span>`
		if !strings.Contains(row(body, href), want) {
			t.Errorf("Summary: the %s row's tile must draw the sprite's %s", href, icon)
		}
	}
	cycle := models.HealthCycle{Title: "Autumn cut", PrimaryGoal: "Weight loss", StartDate: "2026-10-01", EndDate: "2027-03-31", DailyCalorieTarget: 2250, ProteinTargetG: 180}
	if err := h.Service.CreateHealthCycle(&cycle); err != nil {
		t.Fatal(err)
	}
	if plan := row(cfgGet(t, h, "/health"), "/health/plan"); !strings.Contains(plan, "Autumn cut") || !strings.Contains(plan, "2,250 kcal a day") || strings.Contains(plan, "No plan yet") {
		t.Errorf("Summary with a cycle: Plan row = %q", plan)
	}
	// No provider: no Coach row, and the other two stay.
	plain := testHandler(t)
	body = cfgGet(t, plain, "/health")
	if strings.Contains(body, `href="/health/coach"`) || row(body, "/health/plan") == "" || row(body, "/health/more") == "" {
		t.Error("Summary without a provider must have the Plan and More rows and no Coach link")
	}

	if got := cfgGet(t, h, "/health/training"); !strings.Contains(got, "<h1>Training</h1>") {
		t.Error("Training's title must be Training")
	}
	// The link back sits in the page header, above or beside the title.
	for name, want := range map[string]string{
		"plan": `<a href="/health">Health</a>`, "more": `<a href="/health">Health</a>`, "coach": `<a href="/health">Health</a>`,
		"checklist": `<a href="/health/more">More</a>`, "foods": `<a href="/health/more">More</a>`, "profile": `<a href="/health/more">More</a>`,
		"food": `<a href="/health/foods">My foods &amp; products</a>`,
	} {
		page := cfgGet(t, h, pushed[name])
		main := page[strings.Index(page, "<main "):]
		header := markupBetween(main, "<header ", "</header>")
		if !strings.Contains(header, want) {
			t.Errorf("%s: the page header has no back link %s: %s", name, want, header)
		}
	}
}

// c5 (technical review condition d): Nutrition's and Training's eyebrows are not edited. For the
// same data they read as they do on master: the date, and the Hevy state. Summary's shows the
// date alone, as round 7 draws it: its title now says Health, and "Health · date" over "Health"
// said the word twice (design review F4; the lead lifted condition d for that word).
func TestHealthEyebrowsKeepTheirText(t *testing.T) {
	h := testHandler(t)
	day := time.Date(2026, 10, 7, 15, 0, 0, 0, h.Service.Location())
	h.Service.Clock.Now = func() time.Time { return day }
	if body := cfgGet(t, h, "/health"); strings.Contains(markupBetween(body[strings.Index(body, "<main "):], "<header ", "</header>"), "Health ·") {
		t.Error("Summary's eyebrow repeats the title's word")
	}
	for path, want := range map[string]string{
		"/health":           `<p class="hv3-eyebrow">Oct 7, 2026</p>`,
		"/health/nutrition": `<p class="hv3-eyebrow">Oct 7, 2026</p>`,
		"/health/training":  `<p class="hv3-eyebrow">Hevy · Not connected</p>`,
	} {
		if body := cfgGet(t, h, path); !strings.Contains(body, want) {
			t.Errorf("%s: eyebrow is not %s", path, want)
		}
	}
	h.Service.Training = &services.TrainingSync{Provider: doFixtureHevy{}}
	if body := cfgGet(t, h, "/health/training"); !strings.Contains(body, `<p class="hv3-eyebrow">Hevy</p>`) {
		t.Error("Training connected, nothing imported: eyebrow is not Hevy")
	}
	synced := time.Date(2026, 10, 7, 19, 30, 0, 0, time.UTC)
	if err := h.Service.Repo.SaveTraining("hevy", []models.TrainingEntry{{ID: "w1", Kind: "workout", Title: "Leg Day", StartedAt: day.Add(-3 * time.Hour)}}, synced); err != nil {
		t.Fatal(err)
	}
	if body := cfgGet(t, h, "/health/training"); !strings.Contains(body, `<p class="hv3-eyebrow">Hevy · synced Oct 7, 19:30 UTC</p>`) {
		t.Error("Training connected and synced: eyebrow lost the Hevy sync time")
	}
}

// c2, c8: the duplicate Health row is gone from Progress and Plan; the Health stylesheet no
// longer hides the main bar, and still hides the sidebar group on phone (condition a); the
// switcher's tabs are 44 px targets and read no accent, info or state token (lead rule from R1).
func TestHealthNavStyles(t *testing.T) {
	h := testHandler(t)
	for _, path := range []string{"/health/progress", "/health/plan"} {
		body := cfgGet(t, h, path)
		if strings.Contains(body, `class="health-links" aria-label="Health sections"`) || strings.Count(body, `aria-label="Health sections"`) != strings.Count(body, `class="gyst-health-sub"`)+strings.Count(body, `class="gyst-switcher"`) {
			t.Errorf("%s: still has its own row of Health links", path)
		}
	}
	rng := markupBetween(cfgGet(t, h, "/health/progress"), `<nav class="health-links" aria-label="Weight history range">`, "</nav>")
	if strings.Count(rng, "<a ") != 3 {
		t.Errorf("Weight history range must keep its three links: %q", rng)
	}

	hidden := false
	for _, r := range parseCSS(readStatic(t, "css/ui-health-v3.css")) {
		if strings.Contains(r.Selector, ".hv3-tabbar") || strings.Contains(r.Selector, ".is-health .gyst-primary") {
			t.Errorf("ui-health-v3.css still has the Health-only bar rule %q", r.Selector)
		}
		// Without this base rule the group becomes a sixth item in the phone tab bar's grid and wraps it.
		if r.Media == "" && r.has(".gyst-health-sub") && r.Decls["display"] == "none" {
			hidden = true
		}
	}
	if !hidden {
		t.Error("ui-health-v3.css must keep .gyst-health-sub { display: none } outside any media query")
	}
	// Design review F1: .hv3-group clips what is drawn outside a row, so a row's focus ring is drawn inside it.
	ring := false
	for _, r := range parseCSS(readStatic(t, "css/ui-health-v3.css")) {
		if r.has(".hv3-group a.hv3-row:focus-visible") && r.Decls["outline-offset"] == "-2px" {
			ring = true
		}
	}
	if !ring {
		t.Error("ui-health-v3.css must draw the focus ring of a row in a group inside the row (outline-offset: -2px)")
	}

	state := regexp.MustCompile(`^--(vn-(info|ok|warn|crit|idle)|gyst-(green|orange|red|positive|warning|danger))`)
	ref := regexp.MustCompile(`var\((--[a-z0-9-]+)`)
	// Today the file holds only the switcher, so the whole file is held to the rule.
	base, rules := "", 0
	for _, r := range parseCSS(readStatic(t, "css/ui-nav.css")) {
		if strings.Contains(r.Selector, "gyst-switcher") {
			rules++
			// 11 section P: layouts survive 200% text. A label that cannot wrap spills into the next tab.
			if r.Decls["white-space"] == "nowrap" {
				t.Errorf("ui-nav.css: %s sets white-space: nowrap; at 200%% text size the label must be able to wrap", r.Selector)
			}
		}
		if r.has("nav.gyst-switcher > a") {
			if v, ok := r.Decls["min-height"]; ok && r.Media == "" {
				base = v
			} else if ok && v != "44px" {
				t.Errorf("ui-nav.css: %s %s sets min-height %s on a switcher tab", r.Media, r.Selector, v)
			}
		}
		for prop, value := range r.Decls {
			for _, m := range ref.FindAllStringSubmatch(value, -1) {
				if accentNames[m[1]] || state.MatchString(m[1]) {
					t.Errorf("ui-nav.css: %s { %s: %s } reads %s; the current tab is a selection on the neutral surface", r.Selector, prop, value, m[1])
				}
			}
		}
	}
	if base != "44px" {
		t.Errorf("ui-nav.css: nav.gyst-switcher > a min-height = %q, want 44px", base)
	}
	if rules == 0 {
		t.Fatal("ui-nav.css has no switcher rules")
	}

	// Never on Galaxy: its page must not gain a stylesheet.
	layout := readTemplate(t, "layout.html")
	link := `{{if ne .Page "galaxy"}}<link rel="stylesheet" href="/static/css/ui-nav.css?v=nav-1">{{end}}`
	if !strings.Contains(layout, link) || strings.Count(layout, "ui-nav.css") != 1 {
		t.Errorf("layout.html must link ui-nav.css once, as %s", link)
	}
	if strings.Contains(cfgGet(t, h, "/galaxy"), "ui-nav.css") {
		t.Error("/galaxy links ui-nav.css")
	}
	for _, path := range []string{"/health", "/today"} {
		if !strings.Contains(cfgGet(t, h, path), `<link rel="stylesheet" href="/static/css/ui-nav.css?v=nav-1">`) {
			t.Errorf("%s does not link ui-nav.css", path)
		}
	}
}

// c8: the sprite is in production use for the first time, so the offline shell holds it at the
// exact URL the helper emits; the new stylesheet is listed; the worker changes nowhere else.
func TestHealthNavAssets(t *testing.T) {
	m := regexp.MustCompile(`<use href="(/static/icons\.svg\?v=[\w.-]+)#i-\{\{\.\}\}"/>`).FindStringSubmatch(readTemplate(t, "ui_icons.html"))
	if m == nil || m[1] != "/static/icons.svg?v=r7-2" {
		t.Fatalf("vn.icon emits %v, want /static/icons.svg?v=r7-2", m)
	}
	layout := readTemplate(t, "layout.html")
	cache, assets := shellAssets(t)
	count := map[string]int{}
	for _, a := range assets {
		path, _, _ := strings.Cut(a, "?")
		count[path]++
		count[a]++
	}
	for _, a := range []string{m[1], "/static/css/ui-nav.css?v=nav-1"} {
		path, _, _ := strings.Cut(a, "?")
		if count[a] != 1 || count[path] != 1 {
			t.Errorf("SHELL must hold %s exactly once and under no other version", a)
		}
	}
	v := regexp.MustCompile(`/static/css/ui-health-v3\.css\?v=[\w.-]+`).FindAllString(layout+" "+strings.Join(assets, " "), -1)
	if len(v) != 2 || v[0] != v[1] || v[0] == "/static/css/ui-health-v3.css?v=hv3-11" {
		t.Errorf("ui-health-v3.css changed: it needs one new ?v= in layout.html and SHELL, got %v", v)
	}
	if cache == "lifeops-shell-v82-aurora" {
		t.Error("CACHE must move, so installed apps drop the old shell")
	}
	// The fetch, push and notification handlers are as GYST-168 left them: only the cache name and
	// the list may differ. Update this hash only with a change the ticket asks for.
	var rest []string
	for _, line := range strings.Split(readStatic(t, "service-worker.js"), "\n") {
		if !strings.HasPrefix(line, "const CACHE = ") && !strings.HasPrefix(line, "const SHELL = ") {
			rest = append(rest, line)
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(rest, "\n")))
	if got := hex.EncodeToString(sum[:]); got != "6d3cf41526bbc0ce188ae0730e211d41ee1a17d66022ae546e3d1945ebe25d24" {
		t.Errorf("service-worker.js changed outside CACHE and SHELL (sha256 %s)", got)
	}
}
