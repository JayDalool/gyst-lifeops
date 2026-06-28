package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

func mustCreateProject(t *testing.T, repo *Repository, p models.Project) models.Project {
	t.Helper()
	if err := repo.CreateProject(&p); err != nil {
		t.Fatalf("create project %q: %v", p.Name, err)
	}
	return p
}

func projectNames(projects []models.ProjectWithProgress) []string {
	names := make([]string, 0, len(projects))
	for _, p := range projects {
		names = append(names, p.Project.Name)
	}
	return names
}

// The galaxy route is built from TopLevelProjectsByRoadmap, so this method must
// never return subprojects: clicking a top-level destination is what opens the
// solar system that holds them.
func TestTopLevelProjectsByRoadmapExcludesSubprojects(t *testing.T) {
	repo := testRepository(t)
	rm := mustCreateRoadmap(t, repo, "Galaxy", 5)

	top := mustCreateProject(t, repo, models.Project{
		Name: "Solar System A", ProjectKind: "main", Status: "active", RoadmapID: rm.ID, AutoUnlock: true,
	})
	mustCreateProject(t, repo, models.Project{
		Name: "Planet A1", ProjectKind: "subproject", Status: "active", RoadmapID: rm.ID,
		ParentProjectID: top.ID, AutoUnlock: true,
	})
	// A locked subproject would, in the old tree, leak into the top-level
	// "Upcoming" bucket. It must not appear on the galaxy route either.
	mustCreateProject(t, repo, models.Project{
		Name: "Planet A2", ProjectKind: "subproject", Status: "locked", RoadmapID: rm.ID,
		ParentProjectID: top.ID, AutoUnlock: true,
	})

	tops, err := repo.TopLevelProjectsByRoadmap(rm.ID)
	if err != nil {
		t.Fatalf("top level: %v", err)
	}
	if len(tops) != 1 || tops[0].Project.ID != top.ID {
		t.Fatalf("galaxy route must include only top-level destinations; got %v", projectNames(tops))
	}

	kids, err := repo.SubProjects(top.ID)
	if err != nil {
		t.Fatalf("subprojects: %v", err)
	}
	if len(kids) != 2 {
		t.Fatalf("solar system must include child projects (incl. locked); got %v", projectNames(kids))
	}
}

// The list views render from the project tree. A locked/upcoming subproject
// must nest under its parent, never flatten into the top-level Upcoming bucket
// (which would show it as a peer destination on the Projects/roadmap pages).
func TestBuildProjectTreeNestsLockedSubprojects(t *testing.T) {
	repo := testRepository(t)
	rm := mustCreateRoadmap(t, repo, "Galaxy", 5)
	top := mustCreateProject(t, repo, models.Project{
		Name: "Top", ProjectKind: "main", Status: "active", RoadmapID: rm.ID, AutoUnlock: true,
	})
	mustCreateProject(t, repo, models.Project{
		Name: "Locked Child", ProjectKind: "subproject", Status: "locked", RoadmapID: rm.ID,
		ParentProjectID: top.ID, AutoUnlock: true,
	})
	mustCreateProject(t, repo, models.Project{
		Name: "Lonely Planned", ProjectKind: "main", Status: "locked", RoadmapID: rm.ID, AutoUnlock: true,
	})

	tree, err := repo.ProjectTreeByRoadmap(rm.ID)
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	if len(tree.Main) != 1 || tree.Main[0].Project.ID != top.ID {
		t.Fatalf("Main must hold only the active top-level trip; got %v", projectNames(tree.Main))
	}
	for _, u := range tree.Upcoming {
		if u.Project.ParentProjectID != 0 {
			t.Fatalf("Upcoming must never contain a subproject; got %q", u.Project.Name)
		}
	}
	if len(tree.Children[top.ID]) != 1 {
		t.Fatalf("locked subproject must nest under its parent; children=%v", projectNames(tree.Children[top.ID]))
	}
}

// TopLevelProjectsByRoadmap must be scoped: one roadmap's destinations never
// show up under another.
func TestTopLevelProjectsScopedByRoadmap(t *testing.T) {
	repo := testRepository(t)
	a := mustCreateRoadmap(t, repo, "Roadmap A", 5)
	b := mustCreateRoadmap(t, repo, "Roadmap B", 5)

	topA := mustCreateProject(t, repo, models.Project{
		Name: "A Destination", ProjectKind: "main", Status: "active", RoadmapID: a.ID, AutoUnlock: true,
	})
	mustCreateProject(t, repo, models.Project{
		Name: "B Destination", ProjectKind: "main", Status: "active", RoadmapID: b.ID, AutoUnlock: true,
	})

	tops, err := repo.TopLevelProjectsByRoadmap(a.ID)
	if err != nil {
		t.Fatalf("top level: %v", err)
	}
	if len(tops) != 1 || tops[0].Project.ID != topA.ID {
		t.Fatalf("roadmap filter must scope top-level projects; got %v", projectNames(tops))
	}
}
