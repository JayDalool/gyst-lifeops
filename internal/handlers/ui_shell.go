package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/services"
)

// Page identifiers already supplied by handlers determine ownership for detail
// pages as well as list pages. No URL matching or client-side navigation state.
func primarySection(page string) string {
	switch page {
	case "today", "dashboard":
		return "today"
	case "projects", "galaxy", "roadmap":
		return "projects"
	case "health":
		return "health"
	case "courses", "learn":
		return "learn"
	default:
		return "more"
	}
}

// phoneTab names the tab of the phone tab bar a page belongs to (GYST-178, 13B DD-18). Only
// today, tasks and health have a tab of their own, and Projects pages (list, detail, forms) count
// as Tasks because Projects lives inside the Tasks hub (GYST-186); roadmap stays under All; All is the fallback because it is always
// last and reaches every other page through the More list. Galaxy never renders the bar.
func phoneTab(page string) string {
	switch page {
	case "today", "dashboard":
		return "today"
	case "tasks", "projects":
		return "tasks"
	case "health":
		return "health"
	default:
		return "all"
	}
}

// projectsArea names the Projects sub-navigation item a page belongs to
// (GYST-VN-039). Detail pages and forms belong to their list's area.
func projectsArea(page string) string {
	switch page {
	case "galaxy":
		return "galaxy"
	case "roadmap", "roadmaps":
		return "roadmaps"
	default:
		return "projects"
	}
}

// allPagesLabel is the name of the page behind the All tab (GYST-187, spec a1). It is the sidebar label
// off Galaxy and the command bar's item for /more; Galaxy's own bar keeps "More" so /galaxy is sent the bytes it had.
const allPagesLabel = "All pages"

// greetingName is the one place the name sits until chunk R5 lets Jay edit it (Jay's answer 34 keeps it
// on one small greeting row; handlers.go greeting() hard-codes the same name for Today).
const greetingName = "Jay"

// pageDef is one built page. This list is the only place a page's name, address, icon, group and search
// words are written: /more draws it, the sidebar label and the command bar's Go search read it (GYST-187, c3).
type pageDef struct {
	ID, Name, URL, Icon, Group, Terms string
	Pinned                            bool // also drawn in the Pinned row (fixed until R5, Jay's answer 29)
	NeedsCoach                        bool // drawn only while the Health Coach is on
}

// launchGroups is the group order of All pages. A group with no built page is not drawn.
var launchGroups = []struct{ ID, Title string }{
	{"plan", "Plan"}, {"work", "Work"}, {"health", "Health"}, {"learn", "Learn"}, {"system", "System"},
}

// allPages lists the pages that exist today, in spec a1 section 4 order. Terms are the "also found by"
// words plus the hidden words the command bar already knew. A page is added by the ticket that builds it.
var allPages = []pageDef{
	{ID: "today", Name: "Today", URL: "/today", Icon: "home", Group: "plan", Terms: "daily checklist focus now home missions"},
	{ID: "tasks", Name: "Tasks", URL: "/tasks", Icon: "tasks", Group: "plan", Terms: "all tasks to do late new task add missions"},
	{ID: "reminders", Name: "Reminders", URL: "/reminders", Icon: "bell", Group: "plan", Terms: "alerts"},
	{ID: "monthly", Name: "Monthly view", URL: "/monthly", Icon: "cal", Group: "plan", Terms: "calendar month"},
	{ID: "projects", Name: "Projects", URL: "/projects", Icon: "folder", Group: "work", Terms: "worlds new project add create", Pinned: true},
	{ID: "goals", Name: "Goals", URL: "/goals", Icon: "flag", Group: "work", Terms: "milestones new goal add"},
	{ID: "galaxy", Name: "Galaxy", URL: "/galaxy", Icon: "orbit", Group: "work", Terms: "map space planets projects worlds"},
	{ID: "roadmap", Name: "Roadmaps", URL: "/roadmap", Icon: "ship", Group: "work", Terms: "journey new roadmap add"},
	{ID: "analytics", Name: "Analytics", URL: "/analytics", Icon: "bars", Group: "work", Terms: "stats xp charts"},
	{ID: "health", Name: "Health", URL: "/health", Icon: "heart", Group: "health", Terms: "summary"},
	{ID: "nutrition", Name: "Nutrition", URL: "/health/nutrition", Icon: "fork", Group: "health", Terms: "food diary calories protein"},
	{ID: "foods", Name: "My foods", URL: "/health/foods", Icon: "apple", Group: "health", Terms: "foods products saved meals"},
	{ID: "training", Name: "Training", URL: "/health/training", Icon: "gym", Group: "health", Terms: "workout hevy routines gym"},
	{ID: "progress", Name: "Progress", URL: "/health/progress", Icon: "trend", Group: "health", Terms: "weight steps"},
	{ID: "healthplan", Name: "Health plan", URL: "/health/plan", Icon: "target", Group: "health", Terms: "targets cycles cut"},
	{ID: "checklist", Name: "Daily checklist", URL: "/health/checklist", Icon: "check", Group: "health", Terms: "habits"},
	{ID: "profile", Name: "Coach profile", URL: "/health/profile", Icon: "user", Group: "health", Terms: "age height goal weight food rules"},
	{ID: "coach", Name: "Coach", URL: "/health/coach", Icon: "spark", Group: "health", Terms: "ask ai health coach", NeedsCoach: true},
	{ID: "healthextras", Name: "Health extras", URL: "/health/more", Icon: "more", Group: "health", Terms: "more health integrations"},
	{ID: "courses", Name: "Courses", URL: "/courses", Icon: "book", Group: "learn", Terms: "learn lessons study learning add course", Pinned: true},
	{ID: "settings", Name: "Settings", URL: "/settings", Icon: "gear", Group: "system", Terms: "appearance prayer calendar feed preferences theme"},
	{ID: "planimport", Name: "Import a plan", URL: "/plan-import", Icon: "upload", Group: "system", Terms: "upload plan file ai chatgpt markdown tasks checklist"},
	{ID: "archive", Name: "Archive", URL: "/archive", Icon: "box", Group: "system", Terms: "done old restore"},
	{ID: "apps", Name: "Apps", URL: "/apps", Icon: "ext", Group: "system", Terms: "other apps links tools"},
}

// allPagesTerms are the command bar's extra words for the All pages item itself ("menu" found More before).
const allPagesTerms = "more menu tools"

// createShortcuts are forms, not places: they are not tiles (spec a1 section 11) but the command bar still
// finds them, as it did when More listed them (GYST-178 c4). Only search.go reads this.
var createShortcuts = []pageDef{
	{Name: "New project", URL: "/projects/new", Terms: "create add"},
	{Name: "Add course", URL: "/courses/new", Terms: "create new"},
	{Name: "New roadmap", URL: "/roadmaps/new", Terms: "create add journey"},
	{Name: "New goal", URL: "/goals/new", Terms: "create add milestones"},
}

// suggestionIDs are the command bar's four suggestions; the labels come from allPages.
var suggestionIDs = []string{"today", "projects", "galaxy", "health"}

// livePages drops the pages that are off. The Coach tile and its search item depend on the same switch the
// Coach page itself reads, so no page is offered that would not open.
func livePages(pages []pageDef, coachOn bool) []pageDef {
	out := make([]pageDef, 0, len(pages))
	for _, p := range pages {
		if p.NeedsCoach && !coachOn {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (h *Handler) pagesNow() []pageDef { return livePages(allPages, h.coach() != nil) }

// pageHaystack is the text a query is matched against: name, group and extra words, lower-cased. The server
// filter, nav.js (through data-search) and the command bar's ranking all use it, so the same word finds the same
// page with or without script.
func pageHaystack(p pageDef) string {
	group := p.Group
	for _, g := range launchGroups {
		if g.ID == p.Group {
			group = g.Title
		}
	}
	return strings.ToLower(p.Name + " " + group + " " + p.Terms)
}

// matchPages keeps the pages whose haystack holds the query (trimmed, capped, lower-cased).
func matchPages(pages []pageDef, q string) []pageDef {
	q = strings.ToLower(services.NormalizeSearchQuery(q))
	var out []pageDef
	for _, p := range pages {
		if q != "" && strings.Contains(pageHaystack(p), q) {
			out = append(out, p)
		}
	}
	return out
}

// pageSources turns the list into command bar entries: Health pages in the Health group, the rest under Pages.
func pageSources(pages []pageDef) []services.SearchSource {
	out := make([]services.SearchSource, 0, len(pages))
	for _, p := range pages {
		group, meta := services.SearchGroupPages, "Page"
		if p.Group == "health" {
			group, meta = services.SearchGroupHealth, "Health"
		}
		out = append(out, services.SearchSource{Group: group, Item: services.SearchItem{Label: p.Name, Meta: meta, Href: p.URL}, Terms: p.Terms})
	}
	return out
}

type launchTile struct{ Name, URL, Icon, Sec, Search string }
type launchGroup struct {
	ID, Title string
	Tiles     []launchTile
}
type moreData struct {
	Page, DateLine, Greeting, Query, Live string
	Searching                             bool
	Pinned                                []launchTile
	Groups                                []launchGroup
	Results                               []launchTile
}

func tileOf(p pageDef) launchTile {
	return launchTile{Name: p.Name, URL: p.URL, Icon: p.Icon, Sec: p.Group, Search: pageHaystack(p)}
}

// greetingBand names the time of day the spec's way: 05:00-11:59, 12:00-17:59, then evening through 04:59.
func greetingBand(hour int) string {
	switch {
	case hour >= 5 && hour < 12:
		return "Good morning"
	case hour >= 12 && hour < 18:
		return "Good afternoon"
	default:
		return "Good evening"
	}
}

// buildMore lays the list out for the template. now is already in the app's time zone.
func buildMore(pages []pageDef, now time.Time, q string) moreData {
	q = services.NormalizeSearchQuery(q)
	d := moreData{Page: "more", DateLine: now.Format("Monday 2 January"), Greeting: greetingBand(now.Hour()) + ", " + greetingName, Query: q, Searching: q != ""}
	// The groups are always drawn (hidden while a query shows Results) so nav.js can filter them without a request.
	if d.Searching {
		for _, p := range matchPages(pages, q) {
			d.Results = append(d.Results, tileOf(p))
		}
		switch n := len(d.Results); n {
		case 0:
			d.Live = "No page matches"
		case 1:
			d.Live = "1 page"
		default:
			d.Live = strconv.Itoa(n) + " pages"
		}
	}
	for _, p := range pages {
		if p.Pinned {
			d.Pinned = append(d.Pinned, tileOf(p))
		}
	}
	for _, g := range launchGroups {
		lg := launchGroup{ID: g.ID, Title: g.Title}
		for _, p := range pages {
			if p.Group == g.ID {
				lg.Tiles = append(lg.Tiles, tileOf(p))
			}
		}
		if len(lg.Tiles) > 0 {
			d.Groups = append(d.Groups, lg)
		}
	}
	return d
}

func (h *Handler) more(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/more" && r.URL.Path != "/more/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	c := h.appClock()
	h.render(w, "more.html", buildMore(h.pagesNow(), c.Now().In(c.Location()), r.URL.Query().Get("q")))
}
