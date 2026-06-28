package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

// unfeatureAll clears featured flags so a test starts from a known state (the
// migration creates a featured default roadmap).
func unfeatureAll(t *testing.T, repo *Repository) {
	t.Helper()
	if _, err := repo.db.Exec(`UPDATE roadmaps SET featured = 0`); err != nil {
		t.Fatalf("unfeature: %v", err)
	}
}

func mustCreateRoadmap(t *testing.T, repo *Repository, name string, priority int) models.Roadmap {
	t.Helper()
	rm := models.Roadmap{Name: name, Priority: priority, Status: "active", Privacy: "private"}
	if err := repo.CreateRoadmap(&rm); err != nil {
		t.Fatalf("create roadmap %s: %v", name, err)
	}
	return rm
}

func TestDefaultRoadmapExistsAfterMigration(t *testing.T) {
	repo := testRepository(t)
	roadmaps, err := repo.ListRoadmaps(false)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(roadmaps) != 1 || roadmaps[0].Name != "LifeOps Roadmap" || !roadmaps[0].Featured {
		t.Fatalf("expected one featured default roadmap, got %+v", roadmaps)
	}
}

func TestFeaturedOverridesPriority(t *testing.T) {
	repo := testRepository(t)
	unfeatureAll(t, repo)
	mustCreateRoadmap(t, repo, "High Priority", 10)
	low := mustCreateRoadmap(t, repo, "Featured Low", 5)
	if err := repo.SetFeaturedRoadmap(low.ID); err != nil {
		t.Fatalf("set featured: %v", err)
	}
	got, err := repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatalf("dashboard roadmap: %v", err)
	}
	if got.ID != low.ID {
		t.Fatalf("featured roadmap should win over higher priority: got %q", got.Name)
	}
}

func TestHighestPriorityWhenNoneFeatured(t *testing.T) {
	repo := testRepository(t)
	unfeatureAll(t, repo)
	mustCreateRoadmap(t, repo, "Mid", 5)
	top := mustCreateRoadmap(t, repo, "Top", 9)
	got, err := repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatalf("dashboard roadmap: %v", err)
	}
	if got.ID != top.ID {
		t.Fatalf("highest priority should be selected when none featured: got %q", got.Name)
	}
}

func TestUpdatedAtTieBreak(t *testing.T) {
	repo := testRepository(t)
	unfeatureAll(t, repo)
	older := mustCreateRoadmap(t, repo, "Older", 7)
	newer := mustCreateRoadmap(t, repo, "Newer", 7)
	// Force distinct, controlled timestamps (same-second CURRENT_TIMESTAMP would
	// otherwise tie). Newer has the later updated_at.
	if _, err := repo.db.Exec(`UPDATE roadmaps SET updated_at = '2026-01-01 00:00:01' WHERE id = ?`, older.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`UPDATE roadmaps SET updated_at = '2026-01-01 00:00:09' WHERE id = ?`, newer.ID); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatalf("dashboard roadmap: %v", err)
	}
	if got.ID != newer.ID {
		t.Fatalf("tie should break on most recent updated_at: got %q", got.Name)
	}
}

func TestSetFeaturedRoadmapUnsetsOthers(t *testing.T) {
	repo := testRepository(t)
	a := mustCreateRoadmap(t, repo, "Alpha", 3)
	b := mustCreateRoadmap(t, repo, "Beta", 3)
	if err := repo.SetFeaturedRoadmap(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetFeaturedRoadmap(b.ID); err != nil {
		t.Fatal(err)
	}
	roadmaps, err := repo.ListRoadmaps(false)
	if err != nil {
		t.Fatal(err)
	}
	featured := 0
	var featuredID int64
	for _, rm := range roadmaps {
		if rm.Featured {
			featured++
			featuredID = rm.ID
		}
	}
	if featured != 1 || featuredID != b.ID {
		t.Fatalf("exactly one (Beta) should be featured, got count=%d id=%d", featured, featuredID)
	}
}

func TestListProjectsByRoadmap(t *testing.T) {
	repo := testRepository(t)
	rm := mustCreateRoadmap(t, repo, "Trip Set", 2)
	project := models.Project{Name: "Scoped Trip", ProjectKind: "main", Status: "active", AutoUnlock: true}
	if err := repo.CreateProject(&project); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := repo.db.Exec(`UPDATE projects SET roadmap_id = ? WHERE id = ?`, rm.ID, project.ID); err != nil {
		t.Fatal(err)
	}
	scoped, err := repo.ListProjectsByRoadmap(rm.ID)
	if err != nil {
		t.Fatalf("list by roadmap: %v", err)
	}
	if len(scoped) != 1 || scoped[0].Project.ID != project.ID {
		t.Fatalf("expected the scoped project, got %+v", scoped)
	}
	// A different roadmap sees none of it.
	other, err := repo.ListProjectsByRoadmap(rm.ID + 999)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Fatalf("unrelated roadmap should have no projects, got %d", len(other))
	}
}
