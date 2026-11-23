package handlers

import "net/http"

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
// today, tasks and health have a tab of their own; All is the fallback because it is always
// last and reaches every other page through the More list. Galaxy never renders the bar.
func phoneTab(page string) string {
	switch page {
	case "today", "dashboard":
		return "today"
	case "tasks":
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

type moreSection struct {
	Title string
	Rows  []UIRow
}
type moreData struct {
	Page     string
	Sections []moreSection
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
	h.render(w, "more.html", moreData{Page: "more", Sections: moreSections()})
}

// moreSections is the fixed destination list. More renders it and the
// launcher's Go search (search.go) reads the same rows, so they cannot drift.
func moreSections() []moreSection {
	return []moreSection{
		{Title: "Daily planning", Rows: []UIRow{
			{Label: "Today", URL: "/today", Icon: "house"},
			{Label: "Daily checklist", Detail: "Your daily task checklist", URL: "/checklist", Icon: "check"},
			{Label: "All tasks", URL: "/tasks", Icon: "check"},
			{Label: "Reminders", URL: "/reminders", Icon: "bell"},
			{Label: "Calendar & monthly view", Detail: "This month’s tasks and progress", URL: "/monthly", Icon: "book-open"},
		}},
		{Title: "Projects & learning", Rows: []UIRow{
			{Label: "Projects", URL: "/projects", Icon: "folder-heart"},
			// GYST-178 (c4, Codex P2): with Projects and Learn no longer tabs, these create forms would be three taps away.
			{Label: "New project", URL: "/projects/new", Icon: "plus"},
			{Label: "Courses", URL: "/courses", Icon: "book-open"},
			{Label: "Add course", URL: "/courses/new", Icon: "plus"},
			{Label: "Galaxy", Detail: "Explore your projects as worlds", URL: "/galaxy", Icon: "folder-heart"},
			{Label: "Roadmaps", URL: "/roadmap", Icon: "folder-heart"},
			{Label: "New roadmap", URL: "/roadmaps/new", Icon: "plus"},
			{Label: "Goals", URL: "/goals", Icon: "heart"},
			{Label: "New goal", URL: "/goals/new", Icon: "plus"},
		}},
		{Title: "Health", Rows: []UIRow{
			{Label: "Health summary", URL: "/health", Icon: "heart"},
			{Label: "Nutrition", URL: "/health/nutrition", Icon: "heart"},
			{Label: "Training", URL: "/health/training", Icon: "heart"},
			{Label: "Progress", URL: "/health/progress", Icon: "heart"},
			{Label: "Health plan", URL: "/health/plan", Icon: "heart"},
			// GYST-178 (c4): without these two rows both pages are three taps from outside Health.
			{Label: "My foods & products", Detail: "What you usually have, per serving", URL: "/health/foods", Icon: "apple"},
			{Label: "Profile for the coach", Detail: "Age, height, goal weight, food rules", URL: "/health/profile", Icon: "heart"},
		}},
		{Title: "Review & tools", Rows: []UIRow{
			{Label: "Analytics", URL: "/analytics", Icon: "book-open"},
			{Label: "Import a plan", URL: "/plan-import", Icon: "plus"},
			{Label: "Archive", URL: "/archive", Icon: "inbox"},
		}},
		{Title: "Operate", Rows: []UIRow{
			{Label: "Apps", Detail: "Open your other tools", URL: "/apps", Icon: "external-link"},
		}},
		{Title: "Preferences", Rows: []UIRow{
			{Label: "Settings", Detail: "Appearance, prayer times and calendar feeds", URL: "/settings", Icon: "settings"},
		}},
	}
}
