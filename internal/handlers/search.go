package handlers

import (
	"encoding/json"
	"net/http"

	"lifeops-tracker/internal/services"
)

// GET /api/search?q= feeds the launcher's Go mode (GYST-102). The titles it
// returns are private, so it is protected like the pages: Authenticated()
// answers a signed-out fetch with 401 JSON, and the response is never cached.
// It is read-only (no GalaxyData) and returns Health page names, never values.

// searchTerms adds hidden match words to a few destinations.
var searchTerms = map[string]string{
	"/today":        "home daily missions",
	"/galaxy":       "projects planets worlds",
	"/more":         "menu tools",
	"/projects/new": "create add",
	"/plan-import":  "ai chatgpt markdown plan tasks checklist",
	"/courses":      "study learning",
	"/settings":     "prayer preferences appearance theme",
	"/tasks":        "missions all",
}

type searchResponse struct {
	Groups []services.SearchGroup `json:"groups"`
}

type doResponse struct {
	Commands []services.LauncherCommand `json:"commands"`
}

func (h *Handler) searchAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	q := services.NormalizeSearchQuery(r.URL.Query().Get("q"))
	if r.URL.Query().Get("mode") == "do" {
		cmds, err := h.Service.LauncherCommands(h.Service.Training != nil)
		if err != nil {
			serverError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(doResponse{Commands: services.FilterLauncherCommands(cmds, q)})
		return
	}
	groups := []services.SearchGroup{}
	if q == "" {
		groups = append(groups, searchSuggestions())
	} else {
		fixed, err := h.searchFixed(r)
		if err != nil {
			serverError(w, err)
			return
		}
		found, err := h.Service.SearchGo(q, fixed)
		if err != nil {
			serverError(w, err)
			return
		}
		if found != nil {
			groups = found
		}
	}
	_ = json.NewEncoder(w).Encode(searchResponse{Groups: groups})
}

func searchSuggestions() services.SearchGroup {
	g := services.SearchGroup{Label: services.SearchGroupSuggest}
	for _, s := range []struct{ label, href string }{{"Today", "/today"}, {"Projects", "/projects"}, {"Galaxy", "/galaxy"}, {"Health", "/health"}} {
		g.Items = append(g.Items, services.SearchItem{Label: s.label, Meta: "Page", Href: s.href})
	}
	return g
}

// searchFixed lists the destinations that are not rows in the database: the
// More pages, Health pages by name, Galaxy and the apps this host may open.
func (h *Handler) searchFixed(r *http.Request) ([]services.SearchSource, error) {
	var out []services.SearchSource
	seen := map[string]bool{}
	add := func(group, label, href, meta string) {
		if seen[href] {
			return
		}
		seen[href] = true
		out = append(out, services.SearchSource{Group: group, Item: services.SearchItem{Label: label, Meta: meta, Href: href}, Terms: searchTerms[href]})
	}
	for _, section := range moreSections() {
		for _, row := range section.Rows {
			if section.Title == "Health" {
				add(services.SearchGroupHealth, row.Label, row.URL, "Health")
			} else {
				add(services.SearchGroupPages, row.Label, row.URL, "Page")
			}
		}
	}
	add(services.SearchGroupHealth, "Foods", "/health/foods", "Health")
	add(services.SearchGroupHealth, "More health", "/health/more", "Health")
	if h.Coach != nil {
		add(services.SearchGroupHealth, "Health coach", "/health/coach", "Health")
	}
	add(services.SearchGroupPages, "Menu", "/more", "Page")
	add(services.SearchGroupPages, "New project", "/projects/new", "Page")

	load := h.appRegistry
	if load == nil {
		load = services.AppRegistry
	}
	apps, err := load()
	if err != nil {
		return nil, err
	}
	onTailnet := classifyIngress(r) == bucketTailnet
	host := normalizeHost(r.Host)
	for _, a := range apps {
		url := appURL(a, onTailnet, host)
		if url == "" {
			continue // a tailnet app on the public host never appears
		}
		out = append(out, services.SearchSource{Group: services.SearchGroupApps, Item: services.SearchItem{Label: a.Name, Meta: "App", Href: url, External: true}})
	}
	return out, nil
}
