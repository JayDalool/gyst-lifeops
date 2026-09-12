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
	case "courses":
		return "courses"
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
		{Title: "Your day", Rows: []UIRow{
			{Label: "Daily checklist", Detail: "Your tasks and routines in Today", URL: "/today", Icon: "check"},
			{Label: "All tasks", URL: "/tasks", Icon: "check"},
			{Label: "Reminders", URL: "/reminders", Icon: "bell"},
			{Label: "Monthly", Detail: "This month’s tasks and progress", URL: "/monthly", Icon: "book-open"},
		}},
		{Title: "Build & grow", Rows: []UIRow{
			{Label: "Galaxy", Detail: "Explore your projects as worlds", URL: "/galaxy", Icon: "folder-heart"},
			{Label: "Roadmaps", URL: "/roadmap", Icon: "folder-heart"},
			{Label: "Goals", URL: "/goals", Icon: "heart"},
			{Label: "Analytics", URL: "/analytics", Icon: "book-open"},
		}},
		{Title: "Tools & preferences", Rows: []UIRow{
			{Label: "Import a plan", URL: "/plan-import", Icon: "plus"},
			{Label: "Archive", URL: "/archive", Icon: "inbox"},
			{Label: "Settings", Detail: "Appearance, prayer times and calendar feeds", URL: "/settings", Icon: "settings"},
		}},
	}})
}
