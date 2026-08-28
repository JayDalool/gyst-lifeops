package repository

import (
	"path/filepath"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
)

func testRepository(t *testing.T) *Repository {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return New(database)
}

func TestLockedProjectNotesPersist(t *testing.T) {
	repo := testRepository(t)
	project := models.Project{
		Name:        "Chapter Two",
		ProjectKind: "phase",
		Status:      "locked",
		Notes:       "Remember the setup.",
		UnlockNotes: "Unlock after chapter one.",
		AutoUnlock:  true,
	}
	if err := repo.CreateProject(&project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	got, err := repo.ProjectByID(project.ID)
	if err != nil {
		t.Fatalf("project by id: %v", err)
	}
	if got.Status != "locked" || got.Notes != project.Notes || got.UnlockNotes != project.UnlockNotes {
		t.Fatalf("locked notes not persisted: %#v", got)
	}
}

func TestAutoUnlockDependentProject(t *testing.T) {
	repo := testRepository(t)
	first := models.Project{Name: "First Quest", Status: "active", ProjectKind: "main", AutoUnlock: true}
	if err := repo.CreateProject(&first); err != nil {
		t.Fatalf("create first: %v", err)
	}
	next := models.Project{Name: "Next Quest", Status: "locked", ProjectKind: "main", DependsOnProjectID: first.ID, AutoUnlock: true}
	if err := repo.CreateProject(&next); err != nil {
		t.Fatalf("create next: %v", err)
	}

	if err := repo.SetProjectStatus(first.ID, "done"); err != nil {
		t.Fatalf("set done: %v", err)
	}
	got, err := repo.ProjectByID(next.ID)
	if err != nil {
		t.Fatalf("reload next: %v", err)
	}
	if got.Status != "upcoming" {
		t.Fatalf("dependent status = %q, want upcoming", got.Status)
	}
}

func TestAutoUnlockRespectsOptOutAndManualOverride(t *testing.T) {
	repo := testRepository(t)
	dependency := models.Project{Name: "Dependency", Status: "active", ProjectKind: "main"}
	if err := repo.CreateProject(&dependency); err != nil {
		t.Fatal(err)
	}
	dependent := models.Project{
		Name:               "Manual Quest",
		Status:             "locked",
		ProjectKind:        "main",
		DependsOnProjectID: dependency.ID,
		AutoUnlock:         false,
	}
	if err := repo.CreateProject(&dependent); err != nil {
		t.Fatal(err)
	}

	if err := repo.SetProjectStatus(dependency.ID, "done"); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ProjectByID(dependent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "locked" {
		t.Fatalf("opted-out dependent status = %q, want locked", got.Status)
	}

	for _, status := range []string{"upcoming", "active", "archived"} {
		if err := repo.SetProjectStatus(dependent.ID, status); err != nil {
			t.Fatalf("manual transition to %s: %v", status, err)
		}
		got, err = repo.ProjectByID(dependent.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != status {
			t.Fatalf("manual status = %q, want %q", got.Status, status)
		}
	}
}

func TestTaskCompletionAwardsStarsOnce(t *testing.T) {
	repo := testRepository(t)
	task := models.Task{Title: "Mission", DueDate: "2026-06-24", StarsReward: 4, ProofType: "none"}
	if err := repo.CreateTask(&task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatalf("complete task again: %v", err)
	}
	stats, err := repo.PlayerStats()
	if err != nil {
		t.Fatalf("player stats: %v", err)
	}
	if stats.TotalStars != 4 {
		t.Fatalf("total stars = %d, want 4", stats.TotalStars)
	}
}

func TestProofBonusAwardsExtraStars(t *testing.T) {
	repo := testRepository(t)
	task := models.Task{
		Title:         "Proof Mission",
		DueDate:       "2026-06-24",
		StarsReward:   3,
		ProofRequired: true,
		ProofType:     "link",
		ProofURL:      "https://example.com/proof",
	}
	if err := repo.CreateTask(&task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatalf("complete task: %v", err)
	}
	stats, err := repo.PlayerStats()
	if err != nil {
		t.Fatalf("player stats: %v", err)
	}
	if stats.TotalStars != 5 {
		t.Fatalf("total stars = %d, want 5 (3 reward + 2 proof bonus)", stats.TotalStars)
	}
	if stats.ProofCount != 1 {
		t.Fatalf("proof count = %d, want 1", stats.ProofCount)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatalf("complete proof task again: %v", err)
	}
	achievements, err := repo.CountAchievements()
	if err != nil {
		t.Fatal(err)
	}
	if achievements != 1 {
		t.Fatalf("proof achievements = %d, want 1", achievements)
	}
}

func TestRequiredProofMustMatchTypeAndLocation(t *testing.T) {
	repo := testRepository(t)
	tests := []struct {
		name      string
		proofType string
		proofURL  string
		proofPath string
	}{
		{name: "link in path field", proofType: "link", proofPath: "https://example.com/proof"},
		{name: "relative link", proofType: "link", proofURL: "/proof"},
		{name: "file in URL field", proofType: "file", proofURL: "https://example.com/proof.pdf"},
		{name: "missing text", proofType: "text"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := models.Task{
				Title:         tc.name,
				DueDate:       "2026-06-24",
				StarsReward:   3,
				ProofRequired: true,
				ProofType:     tc.proofType,
				ProofURL:      tc.proofURL,
				ProofPath:     tc.proofPath,
			}
			if err := repo.CreateTask(&task); err != nil {
				t.Fatal(err)
			}
			if err := repo.SetTaskOccurrenceStatus(task.ID, task.DueDate, "completed"); err == nil {
				t.Fatal("expected completion to reject mismatched or missing proof")
			}
		})
	}
}

func TestProjectCompletionRewardIsIdempotent(t *testing.T) {
	repo := testRepository(t)
	project := models.Project{Name: "Quest", Status: "active", ProjectKind: "main"}
	if err := repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Mission"}
	if err := repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := repo.SetProjectItemStatus(project.ID, item.ID, "completed"); err != nil {
			t.Fatalf("completion %d: %v", i+1, err)
		}
		if i == 0 {
			if err := repo.SetProjectItemStatus(project.ID, item.ID, "pending"); err != nil {
				t.Fatalf("reopen: %v", err)
			}
		}
	}

	stats, err := repo.PlayerStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalStars != 10 {
		t.Fatalf("quest reward stars = %d, want 10", stats.TotalStars)
	}
	var achievements int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM achievements WHERE trigger_type = 'project_done'`).Scan(&achievements); err != nil {
		t.Fatal(err)
	}
	if achievements != 1 {
		t.Fatalf("quest completion achievements = %d, want 1", achievements)
	}
}
