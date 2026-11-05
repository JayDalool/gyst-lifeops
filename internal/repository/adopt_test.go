package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

func projectRoadmap(t *testing.T, repo *Repository, id int64) int64 {
	t.Helper()
	p, err := repo.ProjectByID(id)
	if err != nil {
		t.Fatal(err)
	}
	return p.RoadmapID
}

func archiveAllRoadmaps(t *testing.T, repo *Repository) {
	t.Helper()
	if _, err := repo.db.Exec(`UPDATE roadmaps SET status = 'archived', archived_at = CURRENT_TIMESTAMP`); err != nil {
		t.Fatal(err)
	}
}

func TestCreateProjectUnassignedGetsDashboardRoadmap(t *testing.T) {
	repo := testRepository(t)
	dash, err := repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatal(err)
	}
	other := mustCreateRoadmap(t, repo, "Fictional Other", 1)
	un := models.Project{Name: "Fictional Unassigned", ProjectKind: "main", Status: "active"}
	if err := repo.CreateProject(&un); err != nil {
		t.Fatal(err)
	}
	if got := projectRoadmap(t, repo, un.ID); got != dash.ID {
		t.Fatalf("unassigned project roadmap = %d, want dashboard %d", got, dash.ID)
	}
	explicit := models.Project{Name: "Fictional Explicit", ProjectKind: "main", Status: "active", RoadmapID: other.ID}
	if err := repo.CreateProject(&explicit); err != nil {
		t.Fatal(err)
	}
	if got := projectRoadmap(t, repo, explicit.ID); got != other.ID {
		t.Fatalf("explicit roadmap = %d, want %d kept", got, other.ID)
	}
}

func TestCreateAndUpdateProjectStayUnassignedWithoutActiveRoadmap(t *testing.T) {
	repo := testRepository(t)
	archiveAllRoadmaps(t, repo)
	p := models.Project{Name: "Fictional Loose", ProjectKind: "main", Status: "active"}
	if err := repo.CreateProject(&p); err != nil {
		t.Fatal(err)
	}
	if got := projectRoadmap(t, repo, p.ID); got != 0 {
		t.Fatalf("roadmap = %d, want NULL", got)
	}
	if err := repo.UpdateProject(&p); err != nil {
		t.Fatal(err)
	}
	if got := projectRoadmap(t, repo, p.ID); got != 0 {
		t.Fatalf("roadmap after update = %d, want NULL", got)
	}
	if err := repo.AdoptOrphanProjects(); err != nil {
		t.Fatal(err)
	}
	if got := projectRoadmap(t, repo, p.ID); got != 0 {
		t.Fatalf("roadmap after no-op adopt = %d, want NULL", got)
	}
}

func TestUpdateProjectUnassignedGetsDashboardRoadmap(t *testing.T) {
	repo := testRepository(t)
	dash, _ := repo.GetDashboardRoadmap()
	other := mustCreateRoadmap(t, repo, "Fictional Other", 1)
	p := models.Project{Name: "Fictional Mover", ProjectKind: "main", Status: "active", RoadmapID: other.ID}
	if err := repo.CreateProject(&p); err != nil {
		t.Fatal(err)
	}
	p.RoadmapID = 0
	if err := repo.UpdateProject(&p); err != nil {
		t.Fatal(err)
	}
	if got := projectRoadmap(t, repo, p.ID); got != dash.ID {
		t.Fatalf("roadmap = %d, want dashboard %d", got, dash.ID)
	}
}

func TestImportRoadmapPlanProjectsGetDashboardRoadmap(t *testing.T) {
	repo := testRepository(t)
	dash, _ := repo.GetDashboardRoadmap()
	if _, err := repo.ImportRoadmapPlan(models.RoadmapImportPlan{
		SourceMarkdown: "fictional", ImportedDate: "2026-01-01",
		Projects: []models.RoadmapProjectPlan{{Name: "Orbit Alpha", ProjectKind: "main", Status: "active"}},
	}); err != nil {
		t.Fatal(err)
	}
	var got int64
	if err := repo.db.QueryRow(`SELECT roadmap_id FROM projects WHERE name = 'Orbit Alpha'`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != dash.ID {
		t.Fatalf("imported roadmap = %d, want %d", got, dash.ID)
	}
}

func TestRoadmapWritesAdoptRemainingOrphans(t *testing.T) {
	repo := testRepository(t)
	archiveAllRoadmaps(t, repo)
	orphan := models.Project{Name: "Fictional Orphan", ProjectKind: "main", Status: "active"}
	if err := repo.CreateProject(&orphan); err != nil {
		t.Fatal(err)
	}
	rm := mustCreateRoadmap(t, repo, "Fictional Fresh", 3) // CreateRoadmap adopts
	if got := projectRoadmap(t, repo, orphan.ID); got != rm.ID {
		t.Fatalf("after CreateRoadmap roadmap = %d, want %d", got, rm.ID)
	}

	archiveAllRoadmaps(t, repo)
	orphan2 := models.Project{Name: "Fictional Orphan Two", ProjectKind: "main", Status: "active"}
	if err := repo.CreateProject(&orphan2); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`UPDATE roadmaps SET archived_at = NULL WHERE id = ?`, rm.ID); err != nil {
		t.Fatal(err)
	}
	rm.Status = "active"
	if err := repo.UpdateRoadmap(&rm); err != nil { // UpdateRoadmap adopts
		t.Fatal(err)
	}
	if got := projectRoadmap(t, repo, orphan2.ID); got != rm.ID {
		t.Fatalf("after UpdateRoadmap roadmap = %d, want %d", got, rm.ID)
	}
}

func TestAdoptOrphanProjectsIdempotentAndLeavesUpdatedAt(t *testing.T) {
	repo := testRepository(t)
	if _, err := repo.db.Exec(`INSERT INTO projects (name, project_kind, status, updated_at) VALUES ('Fictional Raw Orphan', 'main', 'active', '2026-01-01 00:00:00')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := repo.AdoptOrphanProjects(); err != nil {
			t.Fatal(err)
		}
	}
	var rid int64
	var updated string
	if err := repo.db.QueryRow(`SELECT roadmap_id, updated_at FROM projects WHERE name = 'Fictional Raw Orphan'`).Scan(&rid, &updated); err != nil {
		t.Fatal(err)
	}
	dash, _ := repo.GetDashboardRoadmap()
	if rid != dash.ID || updated != "2026-01-01 00:00:00" {
		t.Fatalf("roadmap %d updated_at %q", rid, updated)
	}
}
