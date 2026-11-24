package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// GYST-187 (Issue #274, spec a1): All pages. Fictional data only: the test handler runs on an empty migrated
// database in t.TempDir, and the names below are the page list's own.

var tileLink = regexp.MustCompile(`<a class="gyst-tile" data-sec="([a-z]+)" data-search="([^"]*)" href="([^"]+)">`)

func moreGet(t *testing.T, h *Handler, q string) string {
	t.Helper()
	path := "/more"
	if q != "" {
		path += "?q=" + q
	}
	return cfgGet(t, h, path)
}

// sectionOf returns the markup of the element that opens with open, up to its matching close tag of the same name.
func between(body, open, end string) string {
	i := strings.Index(body, open)
	if i < 0 {
		return ""
	}
	j := strings.Index(body[i:], end)
	if j < 0 {
		return body[i:]
	}
	return body[i : i+j]
}

func hrefsIn(markup string) []string {
	var out []string
	for _, m := range tileLink.FindAllStringSubmatch(markup, -1) {
		out = append(out, m[3])
	}
	return out
}

// c1: the greeting bands, the search form, Pinned, the groups in order, Sign out last.
func TestAllPagesRendersSpec(t *testing.T) {
	loc := time.FixedZone("test", 0)
	at := func(h, m int) time.Time { return time.Date(2026, 11, 23, h, m, 0, 0, loc) }
	for _, tc := range []struct {
		h, m int
		want string
	}{{4, 59, "Good evening"}, {5, 0, "Good morning"}, {11, 59, "Good morning"}, {12, 0, "Good afternoon"}, {17, 59, "Good afternoon"}, {18, 0, "Good evening"}, {23, 59, "Good evening"}} {
		d := buildMore(livePages(allPages, true), at(tc.h, tc.m), "")
		if d.Greeting != tc.want+", "+greetingName {
			t.Errorf("%02d:%02d greeting = %q, want %q", tc.h, tc.m, d.Greeting, tc.want)
		}
	}
	if d := buildMore(allPages, at(9, 0), ""); d.DateLine != "Monday 23 November" {
		t.Errorf("date line = %q", d.DateLine)
	}

	h := testHandler(t)
	body := moreGet(t, h, "")
	if strings.Count(body, "<h1") != 1 || !regexp.MustCompile(`<h1 class="gyst-launch-title">Good (morning|afternoon|evening), Jay</h1>`).MatchString(body) {
		t.Error("want one h1 with the greeting and the name")
	}
	if !strings.Contains(body, "<title>All pages · GYST</title>") {
		t.Error("the title of the page is All pages · GYST")
	}
	form := between(body, `<form class="gyst-launch-search"`, "</form>")
	for _, want := range []string{`role="search"`, `method="get"`, `action="/more"`, `<input type="search" name="q"`, `enterkeyhint="go"`, `autocomplete="off"`, `Search pages`} {
		if !strings.Contains(form, want) {
			t.Errorf("search form lacks %s", want)
		}
	}
	if strings.Contains(form, "autofocus") {
		t.Error("the search field must not take focus on load")
	}
	groups := between(body, "<div data-launch-groups", "<section aria-labelledby=\"launch-results\"")
	pinned := between(groups, `<section aria-labelledby="launch-pinned">`, "</section>")
	if got := strings.Join(hrefsIn(pinned), ","); got != "/projects,/courses" {
		t.Errorf("Pinned = %s, want /projects,/courses", got)
	}
	var order []string
	for _, m := range regexp.MustCompile(`<h2 class="gyst-launch-sec" id="launch-(\w+)"`).FindAllStringSubmatch(groups, -1) {
		order = append(order, m[1])
	}
	if strings.Join(order, ",") != "pinned,plan,work,health,learn,system" {
		t.Errorf("group order = %v", order)
	}
	for _, g := range launchGroups {
		sec := between(groups, `<section aria-labelledby="launch-`+g.ID+`">`, "</section>")
		for _, m := range tileLink.FindAllStringSubmatch(sec, -1) {
			if m[1] != g.ID {
				t.Errorf("tile %s in group %s carries data-sec %s", m[3], g.ID, m[1])
			}
		}
	}
	// 23 built pages while the Coach is off, plus the two pinned repeats.
	if n := len(tileLink.FindAllString(groups, -1)); n != len(livePages(allPages, false))+2 {
		t.Errorf("%d tiles, want %d", n, len(livePages(allPages, false))+2)
	}
	out := strings.LastIndex(body, `<form method="post" action="/logout"`)
	if out < 0 || strings.Index(body[out:], "Sign out") < 0 || strings.LastIndex(body, "<section") > out {
		t.Error("Sign out must be the last thing on the page")
	}
	if strings.Contains(body, "gyst-tile-count") {
		t.Error("no counts in this PR (Jay's answer 28)")
	}
	h.Coach = &fakeVision{}
	all := moreGet(t, h, "")
	for _, p := range allPages {
		if !strings.Contains(all, `href="`+p.URL+`"><span class="gyst-tile-icon"><svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=`) || !strings.Contains(all, `#i-`+p.Icon+`"/>`) {
			t.Errorf("%s: its icon %q is not drawn (add it to launch.icon in more.html)", p.URL, p.Icon)
		}
	}
}

// c2: only built pages; Coach only when on, and every tile opens.
func TestAllPagesOnlyBuiltPages(t *testing.T) {
	h := testHandler(t)
	off := moreGet(t, h, "")
	if strings.Contains(off, `href="/health/coach"`) {
		t.Error("the Coach tile shows while the Coach is off")
	}
	h.Coach = &fakeVision{}
	on := moreGet(t, h, "")
	if strings.Count(on, `href="/health/coach"`) != 1 {
		t.Error("the Coach tile is missing while the Coach is on")
	}
	for _, body := range []string{off, on} {
		for _, gone := range []string{"Calendar", "Overview", "Morning brief", "Achievements", "Accounts", "Automations", "Mail", "Secretary", "Connect", "gyst-tile-count"} {
			if strings.Contains(body, ">"+gone+"<") {
				t.Errorf("an unbuilt page is drawn: %s", gone)
			}
		}
		for _, href := range hrefsIn(body) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, href, nil))
			if w.Code != 200 && w.Code != 302 && w.Code != 303 {
				t.Errorf("tile %s answers %d", href, w.Code)
			}
		}
	}
}

// c3: one list feeds /more, the sidebar label and the command bar; a page added to a copy of the list reaches all.
func TestOnePageListFeedsEverywhere(t *testing.T) {
	h := testHandler(t)
	for _, coach := range []bool{false, true} {
		if coach {
			h.Coach = &fakeVision{}
		}
		want := map[string]bool{}
		for _, p := range h.pagesNow() {
			want[p.URL] = true
		}
		got := map[string]bool{}
		for _, u := range hrefsIn(between(moreGet(t, h, ""), "<div data-launch-groups", "<section aria-labelledby=\"launch-results\"")) {
			got[u] = true
		}
		if fmt.Sprint(sortedKeys(want)) != fmt.Sprint(sortedKeys(got)) {
			t.Errorf("coach=%v: /more tiles %v differ from the list %v", coach, sortedKeys(got), sortedKeys(want))
		}
		fixed, err := h.searchFixed(httptest.NewRequest(http.MethodGet, "/api/search?q=x", nil))
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]string{}
		for _, s := range fixed {
			found[s.Item.Href] = s.Item.Label + "|" + s.Group
		}
		for _, p := range h.pagesNow() {
			group := "Pages"
			if p.Group == "health" {
				group = "Health"
			}
			if found[p.URL] != p.Name+"|"+group {
				t.Errorf("coach=%v: search has %q for %s, want %s|%s", coach, found[p.URL], p.URL, p.Name, group)
			}
		}
		if (found["/health/coach"] != "") != coach {
			t.Errorf("coach=%v: the command bar's Coach item disagrees with the switch", coach)
		}
		if found["/more"] != allPagesLabel+"|Pages" {
			t.Errorf("the command bar names /more %q", found["/more"])
		}
	}
	// The sidebar label is the one constant off Galaxy and More on Galaxy.
	if bar := primaryMarkup(t, cfgGet(t, h, "/today")); !strings.Contains(bar, "<span>"+allPagesLabel+"</span>") {
		t.Error("the sidebar does not say All pages on Today")
	}
	if bar := primaryMarkup(t, cfgGet(t, h, "/galaxy")); !strings.Contains(bar, "<span>More</span>") || strings.Contains(bar, allPagesLabel) {
		t.Error("Galaxy's own bar must keep More")
	}

	// A page added to a test-local copy reaches every consumer that takes the list.
	local := append(append([]pageDef{}, allPages...), pageDef{ID: "zz", Name: "Zeta page", URL: "/zeta", Icon: "box", Group: "system", Terms: "omega"})
	d := buildMore(local, time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC), "")
	var system launchGroup
	for _, g := range d.Groups {
		if g.ID == "system" {
			system = g
		}
	}
	if last := system.Tiles[len(system.Tiles)-1]; last.URL != "/zeta" {
		t.Errorf("/more did not draw the added page: %+v", system.Tiles)
	}
	var inSearch bool
	for _, s := range pageSources(local) {
		inSearch = inSearch || s.Item.Href == "/zeta" && s.Terms == "omega"
	}
	if !inSearch {
		t.Error("the command bar did not get the added page")
	}
	if got := matchPages(local, "omega"); len(got) != 1 || got[0].URL != "/zeta" {
		t.Errorf("the server filter did not find the added page: %+v", got)
	}

	// No consumer owns a literal page name.
	for _, f := range []string{"../../internal/handlers/search.go", "../../web/templates/more.html"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range allPages {
			for _, quoted := range []string{`"` + p.Name + `"`, `>` + p.Name + `<`} {
				if strings.Contains(string(src), quoted) {
					t.Errorf("%s holds the literal page name %s; it must come from the list", f, quoted)
				}
			}
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// c4: the server filter, with no script.
func TestAllPagesServerSearch(t *testing.T) {
	h := testHandler(t)
	results := func(q string) (string, []string) {
		body := moreGet(t, h, q)
		res := between(body, `<section aria-labelledby="launch-results"`, "<form method=\"post\"")
		return body, hrefsIn(res)
	}
	for q, want := range map[string]string{"month": "/monthly", "calendar": "/monthly", "checklist": "/today", "workout": "/health/training", "TASKS": "/tasks", "%20products%20": "/health/foods"} {
		_, got := results(q)
		if !contains(got, want) {
			t.Errorf("q=%s: %v lacks %s", q, got, want)
		}
	}
	if _, got := results("checklist"); !contains(got, "/health/checklist") {
		t.Errorf("checklist must find Daily checklist too: %v", got)
	}
	body, got := results("zzz")
	if len(got) != 0 || !strings.Contains(body, "No page matches") || strings.Contains(between(body, "data-launch-empty", "</div>"), " hidden") {
		t.Errorf("zzz: want the empty block and no results, got %v", got)
	}
	if !strings.Contains(body, `<span data-launch-query>zzz</span>`) || !strings.Contains(body, `role="status"`) {
		t.Error("the empty line must name the query and a polite live region must exist")
	}
	body, got = results("month")
	if !strings.Contains(body, `value="month"`) || !strings.Contains(body, "<div data-launch-groups hidden>") || strings.Contains(body, `<section aria-labelledby="launch-results" data-launch-results hidden>`) || len(got) != 1 {
		t.Errorf("month: want the field kept, the groups hidden and one result: %v", got)
	}
	if !strings.Contains(body, ">1 page</p>") {
		t.Error("the live region should say 1 page")
	}
	// An attack string and a long one are escaped and capped.
	body = moreGet(t, h, "%3Cscript%3Ealert(1)%3C/script%3E")
	if strings.Contains(body, "<script>alert(1)") || !strings.Contains(body, "&lt;script&gt;alert(1)") {
		t.Error("the query is echoed unescaped")
	}
	body = moreGet(t, h, strings.Repeat("a", 300))
	if strings.Contains(body, strings.Repeat("a", 101)) || !strings.Contains(body, strings.Repeat("a", 100)) {
		t.Error("the query is not capped at 100 characters")
	}
	if got := matchPages(allPages, "  "); len(got) != 0 {
		t.Error("a blank query matches nothing; the page shows its groups")
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// c5: every page on All pages is at most two taps from any page: tap 1 is the All tab (or the page's own tab), tap 2 its tile.
func TestEveryListedPageWithinTwoTaps(t *testing.T) {
	h := testHandler(t)
	h.Coach = &fakeVision{}
	more := moreGet(t, h, "")
	tabs := map[string]bool{"/today": true, "/tasks": true, "/health": true}
	var table []string
	for _, p := range h.pagesNow() {
		taps := 2
		if tabs[p.URL] {
			taps = 1
		} else if !strings.Contains(more, `href="`+p.URL+`"`) {
			t.Errorf("%s is not a tile on All pages", p.URL)
		}
		table = append(table, fmt.Sprintf("%-22s %d taps", p.URL, taps))
		// Every layout page carries the way to All (tap 1): the tab bar, or Galaxy's own bar.
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p.URL, nil))
		if w.Code != 200 {
			continue
		}
		if !strings.Contains(w.Body.String(), `<a href="/more"`) {
			t.Errorf("%s has no link to All pages", p.URL)
		}
	}
	for _, f := range []string{"/projects/new", "/courses/new", "/roadmaps/new", "/goals/new"} {
		table = append(table, fmt.Sprintf("%-22s 3 taps (All, the list, its New button)", f))
	}
	t.Log("\n" + strings.Join(table, "\n"))
}

// c6: a tile takes its section's colours only; never a state colour or the accent as identity.
func TestLauncherTilesUseSectionColoursOnly(t *testing.T) {
	css := readStatic(t, "css/ui-nav.css")
	start := strings.Index(css, "/* ---------- All pages (GYST-187")
	if start < 0 {
		t.Fatal("no All pages section in ui-nav.css")
	}
	block := css[start:]
	for _, r := range parseCSS(block) {
		if !strings.Contains(r.Selector, "gyst-tile") && !strings.Contains(r.Selector, "gyst-launch-sec") {
			continue
		}
		for prop, value := range r.Decls {
			for _, m := range regexp.MustCompile(`var\((--[a-z0-9-]+)`).FindAllStringSubmatch(value, -1) {
				if regexp.MustCompile(`^--vn-(warn|info|ok|crit|idle|accent)`).MatchString(m[1]) && !(prop == "outline" && strings.Contains(r.Selector, "focus")) {
					t.Errorf("%s { %s: %s } reads %s", r.Selector, prop, value, m[1])
				}
			}
		}
	}
	if strings.Count(css, "--vn-s-on:") != 3 {
		t.Error("--vn-s-on must be declared on exactly the three theme scopes")
	}
	if strings.Contains(strings.ToLower(block), "#") && regexp.MustCompile(`#[0-9a-f]{3,6}\b`).MatchString(strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(block, "#FFFFFF", ""), "#0B0D12", ""))) {
		t.Error("the All pages rules hold a colour that is not a token")
	}
}

// c9: the asset versions agree in the template, the worker's list and the sprite reference; the cache name moved on.
func TestLauncherAssetVersionsAgree(t *testing.T) {
	layout, icons, worker := readTemplate(t, "layout.html"), readTemplate(t, "ui_icons.html"), readStatic(t, "service-worker.js")
	for _, asset := range []string{"css/ui-nav.css", "js/nav.js", "icons.svg"} {
		ref := regexp.MustCompile(`/static/` + regexp.QuoteMeta(asset) + `\?v=([A-Za-z0-9.-]+)`)
		var versions []string
		for _, src := range []string{layout + icons, worker} {
			m := ref.FindAllStringSubmatch(src, -1)
			if len(m) == 0 {
				t.Fatalf("%s is not referenced", asset)
			}
			for _, x := range m {
				versions = append(versions, x[1])
			}
		}
		for _, v := range versions {
			if v != versions[0] {
				t.Errorf("%s has versions %v; the template and SHELL must agree", asset, versions)
			}
		}
	}
	// Later tickets move the cache again (GYST-179: v88), so this asks only that it is past v85.
	if m := regexp.MustCompile(`const CACHE = "lifeops-shell-v(\d+)`).FindStringSubmatch(worker); m == nil || func() bool { n, _ := strconv.Atoi(m[1]); return n < 86 }() {
		t.Errorf("the cache name did not move past v85: %v", m)
	}
	if strings.Contains(readTemplate(t, "more.html"), "gyst-tile-count") {
		t.Error("no count markup in this PR")
	}
}

// The checked-in table nav.js is tested against must equal what the Go filter says today, so the two cannot drift.
func TestLauncherFilterFixtureIsCurrent(t *testing.T) {
	type fixturePage struct{ URL, Haystack string }
	type fixtureCase struct {
		Query  string
		Expect []string
	}
	type fixture struct {
		Pages []fixturePage
		Cases []fixtureCase
	}
	pages := livePages(allPages, true)
	f := fixture{}
	for _, p := range pages {
		f.Pages = append(f.Pages, fixturePage{p.URL, pageHaystack(p)})
	}
	for _, q := range []string{"month", "calendar", "  Checklist ", "workout", "ZZZ", "", "   ", "food", "health", "a", "Éclair", strings.Repeat("a", 150), "xp", "ai"} {
		c := fixtureCase{Query: q, Expect: []string{}}
		for _, p := range matchPages(pages, q) {
			c.Expect = append(c.Expect, p.URL)
		}
		f.Cases = append(f.Cases, c)
	}
	want, _ := json.MarshalIndent(f, "", " ")
	want = append(want, '\n')
	const path = "../../scripts/fixtures/launch-search.json"
	if os.Getenv("GYST_UPDATE_LAUNCH_FIXTURE") == "1" {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatal("scripts/fixtures/launch-search.json is out of date: run GYST_UPDATE_LAUNCH_FIXTURE=1 go test ./internal/handlers -run LauncherFilterFixture, then node scripts/check-ui-shell.cjs")
	}
}
