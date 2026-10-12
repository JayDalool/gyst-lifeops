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
	h.render(w, "more.html", moreData{Page: "more", Sections: []moreSection{
		{Title: "Daily planning", Rows: []UIRow{
			{Label: "Today", URL: "/today", Icon: "house"},
			{Label: "Daily checklist", Detail: "Your daily task checklist", URL: "/checklist", Icon: "check"},
			{Label: "All tasks", URL: "/tasks", Icon: "check"},
			{Label: "Reminders", URL: "/reminders", Icon: "bell"},
			{Label: "Calendar & monthly view", Detail: "This month’s tasks and progress", URL: "/monthly", Icon: "book-open"},
		}},
		{Title: "Projects & learning", Rows: []UIRow{
			{Label: "Projects", URL: "/projects", Icon: "folder-heart"},
			{Label: "Courses", URL: "/courses", Icon: "book-open"},
			{Label: "Galaxy", Detail: "Explore your projects as worlds", URL: "/galaxy", Icon: "folder-heart"},
			{Label: "Roadmaps", URL: "/roadmap", Icon: "folder-heart"},
			{Label: "Goals", URL: "/goals", Icon: "heart"},
		}},
		{Title: "Health", Rows: []UIRow{
			{Label: "Health summary", URL: "/health", Icon: "heart"},
			{Label: "Nutrition", URL: "/health/nutrition", Icon: "heart"},
			{Label: "Training", URL: "/health/training", Icon: "heart"},
			{Label: "Progress", URL: "/health/progress", Icon: "heart"},
			{Label: "Health plan", URL: "/health/plan", Icon: "heart"},
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
	}})
}
