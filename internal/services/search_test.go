package services

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

func searchLabels(groups []SearchGroup) map[string][]string {
	out := map[string][]string{}
	for _, g := range groups {
		for _, it := range g.Items {
			out[g.Label] = append(out[g.Label], it.Label)
		}
	}
	return out
}

func TestSearchGoRanking(t *testing.T) {
	svc := testService(t)
	for _, p := range []models.Project{
		{Name: "Alpha Garden", ProjectKind: "main", Status: "completed"},
		{Name: "Garden Plan", ProjectKind: "main", Status: "active"},
		{Name: "Back Garden", ProjectKind: "main", Status: "active"},
		{Name: "Garden Done", ProjectKind: "main", Status: "done"},
		{Name: "Hidden GARDEN", ProjectKind: "main", Status: "active"},
	} {
		p := p
		if err := svc.Repo.CreateProject(&p); err != nil {
			t.Fatal(err)
		}
		if p.Name == "Hidden GARDEN" {
			if err := svc.Repo.ArchiveProject(p.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 7; i++ {
		c := models.Course{Name: fmt.Sprintf("Garden course %d", i), Status: "active"}
		if err := svc.Repo.CreateCourse(&c); err != nil {
			t.Fatal(err)
		}
	}
	g := models.Goal{Title: "Grow a garden", Type: "yearly", TargetValue: 4}
	if err := svc.Repo.CreateGoal(&g, "2026-01-01"); err != nil {
		t.Fatal(err)
	}

	groups, err := svc.SearchGo("  gArDeN ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if groups[0].Label != SearchGroupTop || len(groups[0].Items) != 1 {
		t.Fatalf("first group must be one Top hit: %+v", groups)
	}
	got := searchLabels(groups)
	// Prefix before substring; active before done; archived excluded.
	if strings.Join(got[SearchGroupProjects], "|") != "Garden Plan|Garden Done|Back Garden|Alpha Garden" {
		t.Fatalf("projects = %v", got[SearchGroupProjects])
	}
	if strings.Join(got[SearchGroupTop], "") != "Garden course 0" {
		t.Fatalf("top = %v", got[SearchGroupTop])
	}
	for _, label := range got[SearchGroupProjects] {
		if label == "Hidden GARDEN" {
			t.Fatal("archived project returned")
		}
	}
	for _, g := range groups {
		if g.Label != SearchGroupTop && len(g.Items) > SearchGroupLimit {
			t.Fatalf("%s has %d items", g.Label, len(g.Items))
		}
	}
	if n := len(got[SearchGroupLearn]); n > SearchGroupLimit {
		t.Fatalf("learn group has %d", n)
	}
	// Groups are ordered by best hit: Learn and Projects have prefix hits and
	// Goals only a substring hit, so Goals comes last.
	var order []string
	for _, g := range groups {
		order = append(order, g.Label)
	}
	if strings.Join(order, ",") != "Top hit,Learn,Projects,Goals" {
		t.Fatalf("groups = %v", order)
	}
	if len(got[SearchGroupLearn]) != SearchGroupLimit {
		t.Fatalf("learn = %v", got[SearchGroupLearn])
	}
}

func TestRankSearchOrdering(t *testing.T) {
	src := []SearchSource{
		{Group: SearchGroupProjects, Item: SearchItem{Label: "Beta sky"}, Done: false},
		{Group: SearchGroupProjects, Item: SearchItem{Label: "Sky done"}, Done: true},
		{Group: SearchGroupProjects, Item: SearchItem{Label: "Sky active"}},
		{Group: SearchGroupPages, Item: SearchItem{Label: "Settings"}, Terms: "prefs sky"},
		{Group: SearchGroupPages, Item: SearchItem{Label: "100% sky"}},
	}
	groups := RankSearch("SKY", src)
	if groups[0].Label != SearchGroupTop || groups[0].Items[0].Label != "Sky active" {
		t.Fatalf("top hit = %+v", groups[0])
	}
	if got := searchLabels(groups)[SearchGroupProjects]; strings.Join(got, "|") != "Sky done|Beta sky" {
		t.Fatalf("projects = %v", got)
	}
	if got := searchLabels(groups)[SearchGroupPages]; strings.Join(got, "|") != "100% sky|Settings" {
		t.Fatalf("pages = %v", got)
	}
	// No wildcard behaviour: % and _ are plain characters.
	if len(RankSearch("%", src)) == 0 || len(RankSearch("_", src)) != 0 {
		t.Fatal("wildcard characters must match literally")
	}
	if RankSearch("   ", src) != nil || RankSearch("zzz", src) != nil {
		t.Fatal("empty and unmatched queries return nothing")
	}
}

func TestNormalizeSearchQueryCaps(t *testing.T) {
	long := strings.Repeat("é", 150)
	if got := NormalizeSearchQuery("  " + long + "  "); len([]rune(got)) != SearchQueryLimit {
		t.Fatalf("cap = %d runes", len([]rune(got)))
	}
}

func TestSearchGoDoesNotWrite(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	svc := New(repository.New(database))
	if err := svc.Repo.CreateProject(&models.Project{Name: "Orbit", ProjectKind: "main", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		var s string
		row := database.QueryRow(`SELECT (SELECT COUNT(*) FROM projects) || '/' || (SELECT COUNT(*) FROM courses) || '/' || (SELECT COUNT(*) FROM goals) || '/' ||
			COALESCE((SELECT group_concat(COALESCE(pos_x,'n') || ',' || COALESCE(pos_y,'n') || ',' || COALESCE(pos_z,'n') || ',' || updated_at) FROM projects), '')`)
		if err := row.Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := snapshot()
	for _, q := range []string{"orb", "", "zzz", "o"} {
		if _, err := svc.SearchGo(q, nil); err != nil {
			t.Fatal(err)
		}
	}
	if snapshot() != before {
		t.Fatal("search changed stored rows or world positions")
	}
}
