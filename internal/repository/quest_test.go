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
}
