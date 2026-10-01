package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

func totalStars(t *testing.T, repo *Repository) int {
	t.Helper()
	stats, err := repo.PlayerStats()
	if err != nil {
		t.Fatalf("player stats: %v", err)
	}
	return stats.TotalStars
}

func completionRows(t *testing.T, repo *Repository, taskID int64) int {
	t.Helper()
	var n int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM task_completions WHERE task_id = ?`, taskID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func mustUndoTask(t *testing.T, repo *Repository, id int64, date string, want bool) {
	t.Helper()
	got, err := repo.UndoTaskCompletion(id, date)
	if err != nil {
		t.Fatalf("undo task: %v", err)
	}
	if got != want {
		t.Fatalf("undo reverted = %v, want %v", got, want)
	}
}

func TestUndoOneOffTaskCompletion(t *testing.T) {
	repo := testRepository(t)
	task := models.Task{Title: "Water the plants", DueDate: "2026-06-24", StarsReward: 4}
	if err := repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatal(err)
	}
	if got := totalStars(t, repo); got != 4 {
		t.Fatalf("after complete = %d, want 4", got)
	}
	mustUndoTask(t, repo, task.ID, "2026-06-24", true)
	got, err := repo.TaskByID(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "pending" || got.StarsAwarded != 0 || got.CompletedAt != "" {
		t.Fatalf("task after undo = %#v", got)
	}
	if completionRows(t, repo, task.ID) != 0 {
		t.Fatal("undo left a task_completions row")
	}
	if got := totalStars(t, repo); got != 0 {
		t.Fatalf("after undo = %d, want 0", got)
	}
	mustUndoTask(t, repo, task.ID, "2026-06-24", false)
	if got := totalStars(t, repo); got != 0 {
		t.Fatalf("after second undo = %d, want 0 (no negatives)", got)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatal(err)
	}
	if got := totalStars(t, repo); got != 4 {
		t.Fatalf("after re-complete = %d, want 4 (awarded once)", got)
	}
}

func TestUndoDailyTaskOnlyAffectsThatDate(t *testing.T) {
	repo := testRepository(t)
	task := models.Task{Title: "Stretch", DueDate: "2026-06-24", RepeatType: "daily", StarsReward: 10}
	if err := repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{"2026-06-24", "2026-06-25"} {
		if err := repo.SetTaskOccurrenceStatus(task.ID, date, "completed"); err != nil {
			t.Fatal(err)
		}
	}
	if got := totalStars(t, repo); got != 20 {
		t.Fatalf("after two days = %d, want 20", got)
	}
	mustUndoTask(t, repo, task.ID, "2026-06-24", true)
	if got := totalStars(t, repo); got != 10 {
		t.Fatalf("after undo = %d, want 10", got)
	}
	mustUndoTask(t, repo, task.ID, "2026-06-24", false)
	if got := totalStars(t, repo); got != 10 {
		t.Fatalf("after second undo = %d, want 10", got)
	}
	if completionRows(t, repo, task.ID) != 1 {
		t.Fatal("the other day's completion was removed")
	}
}

func TestUndoProofTaskKeepsSingleAchievement(t *testing.T) {
	repo := testRepository(t)
	task := models.Task{Title: "Proof errand", DueDate: "2026-06-24", StarsReward: 3, ProofRequired: true, ProofType: "link", ProofURL: "https://example.com/proof"}
	if err := repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatal(err)
	}
	if got := totalStars(t, repo); got != 8 {
		t.Fatalf("after complete = %d, want 8", got)
	}
	mustUndoTask(t, repo, task.ID, "2026-06-24", true)
	if got := totalStars(t, repo); got != 0 {
		t.Fatalf("after undo = %d, want 0", got)
	}
	if n, _ := repo.CountAchievements(); n != 1 {
		t.Fatalf("achievements after undo = %d, want 1", n)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatal(err)
	}
	if got := totalStars(t, repo); got != 8 {
		t.Fatalf("after re-complete = %d, want 8", got)
	}
	if n, _ := repo.CountAchievements(); n != 1 {
		t.Fatalf("achievements after re-complete = %d, want 1", n)
	}
}

func TestUndoTaskLeavesSkippedOccurrenceAlone(t *testing.T) {
	repo := testRepository(t)
	task := models.Task{Title: "Skip me", DueDate: "2026-06-24", RepeatType: "daily", StarsReward: 10}
	if err := repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "skipped"); err != nil {
		t.Fatal(err)
	}
	mustUndoTask(t, repo, task.ID, "2026-06-24", false)
	if completionRows(t, repo, task.ID) != 1 {
		t.Fatal("undo removed a skipped occurrence")
	}
}

func TestUndoProjectMission(t *testing.T) {
	repo := testRepository(t)
	project := models.Project{Name: "Two step quest", Status: "active", ProjectKind: "main"}
	if err := repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	first := models.ProjectItem{ProjectID: project.ID, Title: "Step one"}
	second := models.ProjectItem{ProjectID: project.ID, Title: "Step two"}
	for _, item := range []*models.ProjectItem{&first, &second} {
		if err := repo.CreateProjectItem(item); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.SetProjectItemStatus(project.ID, first.ID, "completed"); err != nil {
		t.Fatal(err)
	}
	if got := totalStars(t, repo); got != 20 {
		t.Fatalf("after complete = %d, want 20", got)
	}
	ok, err := repo.UndoProjectItemCompletion(project.ID, first.ID)
	if err != nil || !ok {
		t.Fatalf("undo = %v, %v", ok, err)
	}
	if got := totalStars(t, repo); got != 0 {
		t.Fatalf("after undo = %d, want 0", got)
	}
	items, _ := repo.ProjectItems(project.ID)
	for _, item := range items {
		if item.ID == first.ID && (item.Status != "pending" || item.StarsAwarded != 0) {
			t.Fatalf("mission after undo = %#v", item)
		}
	}
	if got, _ := repo.ProjectByID(project.ID); got.Status != "active" {
		t.Fatalf("project status = %q, want active", got.Status)
	}
	ok, err = repo.UndoProjectItemCompletion(project.ID, first.ID)
	if err != nil || ok {
		t.Fatalf("second undo = %v, %v; want no-op", ok, err)
	}
	if err := repo.SetProjectItemStatus(project.ID, first.ID, "completed"); err != nil {
		t.Fatal(err)
	}
	if got := totalStars(t, repo); got != 20 {
		t.Fatalf("after re-complete = %d, want 20", got)
	}
}

func TestUndoMissionThatFinishedProjectIsRefused(t *testing.T) {
	repo := testRepository(t)
	project := models.Project{Name: "One step quest", Status: "active", ProjectKind: "main"}
	if err := repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	item := models.ProjectItem{ProjectID: project.ID, Title: "Only step"}
	if err := repo.CreateProjectItem(&item); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetProjectItemStatus(project.ID, item.ID, "completed"); err != nil {
		t.Fatal(err)
	}
	ok, err := repo.UndoProjectItemCompletion(project.ID, item.ID)
	if err != nil || ok {
		t.Fatalf("undo on a finished project = %v, %v; want refused", ok, err)
	}
	if got := totalStars(t, repo); got != 270 {
		t.Fatalf("total = %d, want 270 unchanged", got)
	}
	items, _ := repo.ProjectItems(project.ID)
	if len(items) != 1 || items[0].Status != "completed" {
		t.Fatalf("mission changed: %#v", items)
	}
}

func TestUndoLearningBlockCompletion(t *testing.T) {
	repo := testRepository(t)
	course := models.Course{Name: "Sample course", Status: "active"}
	if err := repo.CreateCourse(&course); err != nil {
		t.Fatal(err)
	}
	done := models.LearningBlock{CourseID: course.ID, Title: "Lesson one", DueDate: "2026-06-24"}
	skipped := models.LearningBlock{CourseID: course.ID, Title: "Lesson two", DueDate: "2026-06-24"}
	for _, b := range []*models.LearningBlock{&done, &skipped} {
		if err := repo.CreateLearningBlock(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.SetLearningBlockStatus(course.ID, done.ID, "completed"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetLearningBlockStatus(course.ID, skipped.ID, "skipped"); err != nil {
		t.Fatal(err)
	}
	ok, err := repo.UndoLearningBlockCompletion(course.ID, done.ID)
	if err != nil || !ok {
		t.Fatalf("undo = %v, %v", ok, err)
	}
	if b, _ := repo.LearningBlockByID(done.ID); b.Status != "pending" || b.CompletedAt != "" {
		t.Fatalf("block after undo = %#v", b)
	}
	ok, err = repo.UndoLearningBlockCompletion(course.ID, done.ID)
	if err != nil || ok {
		t.Fatalf("second undo = %v, %v; want no-op", ok, err)
	}
	ok, err = repo.UndoLearningBlockCompletion(course.ID, skipped.ID)
	if err != nil || ok {
		t.Fatalf("undo on skipped = %v, %v; want no-op", ok, err)
	}
	if b, _ := repo.LearningBlockByID(skipped.ID); b.Status != "skipped" {
		t.Fatalf("skipped block changed: %#v", b)
	}
}
