package handlers

import (
	"net/http"
	"strings"

	"lifeops-tracker/internal/services"
)

type appRow struct {
	Name     string
	Monogram string
	URL      string // empty for a tailnet app on the public host
	Tailnet  bool
}
type appGroup struct {
	Title string
	Rows  []appRow
}
type appsData struct {
	Page   string
	Groups []appGroup
}

var appGroupTitles = map[string]string{
	services.AppGroupWork:     "Work",
	services.AppGroupOperate:  "Operate",
	services.AppGroupLearning: "Learning",
}

// apps lists the external tools in the registry (GYST-101). Tailnet apps are
// links only on a .ts.net host, where the placeholder becomes the request
// host. On every other host (fail safe) their URL is cleared here, so it never
// reaches the template or the response body.
func (h *Handler) apps(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/apps" && r.URL.Path != "/apps/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	load := h.appRegistry
	if load == nil {
		load = services.AppRegistry
	}
	apps, err := load()
	if err != nil {
		serverError(w, err)
		return
	}
	onTailnet := classifyIngress(r) == bucketTailnet
	host := normalizeHost(r.Host)
	data := appsData{Page: "apps"}
	for _, group := range services.AppGroups {
		g := appGroup{Title: appGroupTitles[group]}
		for _, a := range apps {
			if a.Group != group {
				continue
			}
			row := appRow{Name: a.Name, Monogram: a.Monogram, URL: a.URL}
			if a.Environment == services.AppEnvTailnet {
				row.Tailnet = true
				if onTailnet {
					row.URL = strings.ReplaceAll(a.URL, services.TailnetHostPlaceholder, host)
				} else {
					row.URL = ""
				}
			}
			g.Rows = append(g.Rows, row)
		}
		if len(g.Rows) > 0 {
			data.Groups = append(data.Groups, g)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	h.render(w, "apps.html", data)
}
