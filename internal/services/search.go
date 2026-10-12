package services

import (
	"fmt"
	"sort"
	"strings"
)

// Launcher Go search (GYST-102). Read-only: it lists projects, courses and
// goals through the same repository reads the list pages use, and never calls
// GalaxyData, which persists world positions on GET.

// Group labels.
const (
	SearchGroupTop      = "Top hit"
	SearchGroupSuggest  = "Suggestions"
	SearchGroupProjects = "Projects"
	SearchGroupLearn    = "Learn"
	SearchGroupGoals    = "Goals"
	SearchGroupHealth   = "Health"
	SearchGroupApps     = "Apps"
	SearchGroupPages    = "Pages"

	// SearchGroupLimit is the most items one group returns.
	SearchGroupLimit = 5
	// SearchQueryLimit caps the query length in characters.
	SearchQueryLimit = 100
)

var searchGroupOrder = []string{SearchGroupProjects, SearchGroupLearn, SearchGroupGoals, SearchGroupHealth, SearchGroupApps, SearchGroupPages}

type SearchItem struct {
	Label    string `json:"label"`
	Meta     string `json:"meta"`
	Href     string `json:"href"`
	External bool   `json:"external"`
}

type SearchGroup struct {
	Label string       `json:"label"`
	Items []SearchItem `json:"items"`
}

// SearchSource is one searchable thing: the item, the group it belongs to,
// extra match words (never shown) and whether it is finished.
type SearchSource struct {
	Group string
	Item  SearchItem
	Terms string
	Done  bool
}

// NormalizeSearchQuery trims q and caps it at SearchQueryLimit characters.
func NormalizeSearchQuery(q string) string {
	q = strings.TrimSpace(q)
	if r := []rune(q); len(r) > SearchQueryLimit {
		q = strings.TrimSpace(string(r[:SearchQueryLimit]))
	}
	return q
}

// SearchGo searches projects, courses and goals plus the caller's fixed
// destinations. It writes nothing.
func (s *Service) SearchGo(q string, fixed []SearchSource) ([]SearchGroup, error) {
	q = NormalizeSearchQuery(q)
	if q == "" {
		return nil, nil
	}
	var sources []SearchSource
	projects, err := s.Repo.ListProjects(false)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		if p.Project.Status == "archived" || p.Project.ArchivedAt != "" {
			continue
		}
		sources = append(sources, SearchSource{
			Group: SearchGroupProjects,
			Item:  SearchItem{Label: p.Project.Name, Meta: "Project · " + p.Project.Status, Href: fmt.Sprintf("/projects/%d", p.Project.ID)},
			Done:  p.Project.Status == "completed" || p.Project.Status == "done",
		})
	}
	courses, err := s.Repo.ListCourses(false)
	if err != nil {
		return nil, err
	}
	for _, c := range courses {
		if c.Status == "archived" || c.ArchivedAt != "" {
			continue
		}
		sources = append(sources, SearchSource{
			Group: SearchGroupLearn,
			Item:  SearchItem{Label: c.Name, Meta: "Course · " + c.Status, Href: fmt.Sprintf("/courses/%d", c.ID)},
			Done:  c.Status == "completed",
		})
	}
	goals, err := s.Repo.ListGoals(false)
	if err != nil {
		return nil, err
	}
	for _, g := range goals {
		if g.Goal.Status == "archived" || g.Goal.ArchivedAt != "" {
			continue
		}
		sources = append(sources, SearchSource{
			Group: SearchGroupGoals,
			Item:  SearchItem{Label: g.Goal.Title, Meta: "Goal · " + g.Goal.Status, Href: fmt.Sprintf("/goals/%d", g.Goal.ID)},
			Done:  g.Goal.Status == "completed",
		})
	}
	return RankSearch(q, append(sources, fixed...)), nil
}

type scoredSource struct {
	SearchSource
	kind  int // 0 title prefix, 1 title substring, 2 extra words only
	order int
}

func (a scoredSource) before(b scoredSource) bool {
	if a.kind != b.kind {
		return a.kind < b.kind
	}
	if a.Done != b.Done {
		return !a.Done
	}
	la, lb := strings.ToLower(a.Item.Label), strings.ToLower(b.Item.Label)
	if la != lb {
		return la < lb
	}
	return a.order < b.order
}

// RankSearch matches q (case-insensitive, plain Contains/HasPrefix, so there
// are no wildcard rules) against the sources. It returns a Top hit group, then
// the remaining groups ordered by their best hit, at most SearchGroupLimit
// items each. Prefix beats substring and active beats done.
func RankSearch(q string, sources []SearchSource) []SearchGroup {
	q = strings.ToLower(NormalizeSearchQuery(q))
	if q == "" {
		return nil
	}
	byGroup := map[string][]scoredSource{}
	for i, src := range sources {
		label := strings.ToLower(src.Item.Label)
		kind := -1
		switch {
		case strings.HasPrefix(label, q):
			kind = 0
		case strings.Contains(label, q):
			kind = 1
		case src.Terms != "" && strings.Contains(strings.ToLower(src.Terms), q):
			kind = 2
		}
		if kind < 0 {
			continue
		}
		byGroup[src.Group] = append(byGroup[src.Group], scoredSource{SearchSource: src, kind: kind, order: i})
	}
	type ranked struct {
		label string
		items []scoredSource
	}
	var groups []ranked
	for _, label := range searchGroupOrder {
		items := byGroup[label]
		if len(items) == 0 {
			continue
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].before(items[j]) })
		groups = append(groups, ranked{label, items})
	}
	if len(groups) == 0 {
		return nil
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].items[0].before(groups[j].items[0]) })
	top := groups[0].items[0]
	groups[0].items = groups[0].items[1:]
	out := []SearchGroup{{Label: SearchGroupTop, Items: []SearchItem{top.Item}}}
	for _, g := range groups {
		if len(g.items) == 0 {
			continue
		}
		if len(g.items) > SearchGroupLimit {
			g.items = g.items[:SearchGroupLimit]
		}
		group := SearchGroup{Label: g.label}
		for _, it := range g.items {
			group.Items = append(group.Items, it.Item)
		}
		out = append(out, group)
	}
	return out
}
