package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-VN-038: mission and project-status use cases are one transaction each.
// These tests create real SQLite triggers (visible to every pooled connection)
// that RAISE(ABORT) part-way through an operation and assert nothing was left
// behind.

type questWorld struct {
	repo        *Repository
	quest       models.Project
	dependent   models.Project
	firstItem   models.ProjectItem
	secondItem  models.ProjectItem
	startedStar int
}

// newQuestWorld builds an active Quest with one pending mission and a locked
// dependent Quest that auto-unlocks when the first is done.
func newQuestWorld(t *testing.T, missions int) *questWorld {
	t.Helper()
	repo := testRepository(t)
	w := &questWorld{repo: repo}
	w.quest = models.Project{Name: "Atomic Quest", Status: "active", ProjectKind: "main", AutoUnlock: true}
	if err := repo.CreateProject(&w.quest); err != nil {
		t.Fatalf("create quest: %v", err)
	}
	w.dependent = models.Project{Name: "Next Atomic Quest", Status: "locked", ProjectKind: "main", DependsOnProjectID: w.quest.ID, AutoUnlock: true}
	if err := repo.CreateProject(&w.dependent); err != nil {
		t.Fatalf("create dependent: %v", err)
	}
	w.firstItem = models.ProjectItem{ProjectID: w.quest.ID, Title: "First mission", Priority: "medium", StarsReward: 4}
	if err := repo.CreateProjectItem(&w.firstItem); err != nil {
		t.Fatalf("create first item: %v", err)
	}
	if missions > 1 {
		w.secondItem = models.ProjectItem{ProjectID: w.quest.ID, Title: "Second mission", Priority: "medium", StarsReward: 4}
		if err := repo.CreateProjectItem(&w.secondItem); err != nil {
			t.Fatalf("create second item: %v", err)
		}
	}
	return w
}

type worldState struct {
	itemStatus   string
	itemStars    int
	itemCount    int
	questStatus  string
	depStatus    string
	projectDone  int
	proofAwards  int
	totalStars   int
	questUpdated int
}

func (w *questWorld) snapshot(t *testing.T, itemID int64) worldState {
	t.Helper()
	var s worldState
	if itemID != 0 {
		if err := w.repo.db.QueryRow(`SELECT status, stars_awarded FROM project_items WHERE id = ?`, itemID).Scan(&s.itemStatus, &s.itemStars); err != nil {
			t.Fatalf("snapshot item: %v", err)
		}
	}
	mustScan(t, w.repo, `SELECT COUNT(*) FROM project_items WHERE project_id = ?`, []any{w.quest.ID}, &s.itemCount)
	mustScan(t, w.repo, `SELECT status FROM projects WHERE id = ?`, []any{w.quest.ID}, &s.questStatus)
	mustScan(t, w.repo, `SELECT status FROM projects WHERE id = ?`, []any{w.dependent.ID}, &s.depStatus)
	mustScan(t, w.repo, `SELECT COUNT(*) FROM achievements WHERE trigger_type = 'project_done'`, nil, &s.projectDone)
	mustScan(t, w.repo, `SELECT COUNT(*) FROM achievements WHERE trigger_type = 'proof_submitted'`, nil, &s.proofAwards)
	mustScan(t, w.repo, `SELECT sync_version FROM projects WHERE id = ?`, []any{w.quest.ID}, &s.questUpdated)
	stats, err := w.repo.PlayerStats()
	if err != nil {
		t.Fatalf("snapshot stats: %v", err)
	}
	s.totalStars = stats.TotalStars
	return s
}

func mustScan(t *testing.T, repo *Repository, query string, args []any, dest any) {
	t.Helper()
	if err := repo.db.QueryRow(query, args...).Scan(dest); err != nil {
		t.Fatalf("scan %q: %v", query, err)
	}
}

func (w *questWorld) trigger(t *testing.T, name, ddl string) {
	t.Helper()
	if _, err := w.repo.db.Exec(ddl); err != nil {
		t.Fatalf("create trigger %s: %v", name, err)
	}
}

func (w *questWorld) dropTrigger(t *testing.T, name string) {
	t.Helper()
	if _, err := w.repo.db.Exec(`DROP TRIGGER ` + name); err != nil {
		t.Fatalf("drop trigger %s: %v", name, err)
	}
}

const (
	blockUnlockTrigger = `CREATE TRIGGER inject_block_unlock BEFORE UPDATE OF status ON projects
		WHEN NEW.status = 'upcoming' BEGIN SELECT RAISE(ABORT, 'injected'); END`
	blockQuestBonusTrigger = `CREATE TRIGGER inject_block_bonus BEFORE INSERT ON achievements
		WHEN NEW.trigger_type = 'project_done' BEGIN SELECT RAISE(ABORT, 'injected'); END`
)

func TestCompleteLastMissionRollsBackWhenUnlockFails(t *testing.T) {
	w := newQuestWorld(t, 1)
	w.trigger(t, "inject_block_unlock", blockUnlockTrigger)
	before := w.snapshot(t, w.firstItem.ID)

	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "completed"); err == nil {
		t.Fatal("expected injected failure, got nil")
	}
	after := w.snapshot(t, w.firstItem.ID)
	if after != before {
		t.Fatalf("partial state after failed completion:\nbefore %+v\nafter  %+v", before, after)
	}
	if after.itemStatus != "pending" || after.itemStars != 0 || after.questStatus != "active" || after.projectDone != 0 || after.depStatus != "locked" {
		t.Fatalf("unexpected state: %+v", after)
	}

	// Without the failure the same operation succeeds (the rollback left a
	// clean slate and the trigger was the only blocker).
	w.dropTrigger(t, "inject_block_unlock")
	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "completed"); err != nil {
		t.Fatalf("retry after trigger removed: %v", err)
	}
}

func TestCompleteLastMissionRollsBackWhenQuestBonusFails(t *testing.T) {
	w := newQuestWorld(t, 1)
	w.trigger(t, "inject_block_bonus", blockQuestBonusTrigger)
	before := w.snapshot(t, w.firstItem.ID)

	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "completed"); err == nil {
		t.Fatal("expected injected failure, got nil")
	}
	if after := w.snapshot(t, w.firstItem.ID); after != before {
		t.Fatalf("partial state after failed completion:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestSetProjectStatusDoneRollsBackWhenUnlockFails(t *testing.T) {
	w := newQuestWorld(t, 1)
	w.trigger(t, "inject_block_unlock", blockUnlockTrigger)
	before := w.snapshot(t, w.firstItem.ID)

	if err := w.repo.SetProjectStatus(w.quest.ID, "done"); err == nil {
		t.Fatal("expected injected failure, got nil")
	}
	after := w.snapshot(t, w.firstItem.ID)
	if after != before || after.questStatus != "active" || after.projectDone != 0 {
		t.Fatalf("partial state after failed SetProjectStatus:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestDeleteProjectItemRollsBackWhenReconcileFails(t *testing.T) {
	w := newQuestWorld(t, 2)
	// Completing the first mission leaves the quest active (second pending).
	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "completed"); err != nil {
		t.Fatalf("complete first: %v", err)
	}
	// Deleting the pending mission would make the quest done.
	w.trigger(t, "inject_block_done", `CREATE TRIGGER inject_block_done BEFORE UPDATE OF status ON projects
		WHEN NEW.status = 'done' BEGIN SELECT RAISE(ABORT, 'injected'); END`)
	before := w.snapshot(t, w.firstItem.ID)

	if err := w.repo.DeleteProjectItem(w.quest.ID, w.secondItem.ID); err == nil {
		t.Fatal("expected injected failure, got nil")
	}
	after := w.snapshot(t, w.firstItem.ID)
	if after != before || after.itemCount != 2 {
		t.Fatalf("delete was not rolled back:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestUndoProjectItemCompletionRollsBackWhenReconcileFails(t *testing.T) {
	w := newQuestWorld(t, 2)
	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "completed"); err != nil {
		t.Fatalf("complete first: %v", err)
	}
	// A side effect of the undo flips the project to 'done' so that reconcile
	// has to write 'active' back; that write is the one we make fail.
	w.trigger(t, "inject_flip", `CREATE TRIGGER inject_flip AFTER UPDATE ON project_items
		WHEN NEW.status = 'pending' BEGIN UPDATE projects SET status = 'done' WHERE id = NEW.project_id; END`)
	w.trigger(t, "inject_block_active", `CREATE TRIGGER inject_block_active BEFORE UPDATE OF status ON projects
		WHEN NEW.status = 'active' BEGIN SELECT RAISE(ABORT, 'injected'); END`)
	before := w.snapshot(t, w.firstItem.ID)

	ok, err := w.repo.UndoProjectItemCompletion(w.quest.ID, w.firstItem.ID)
	if err == nil || ok {
		t.Fatalf("expected injected failure, got ok=%v err=%v", ok, err)
	}
	after := w.snapshot(t, w.firstItem.ID)
	if after != before || after.itemStatus != "completed" || after.itemStars == 0 || after.questStatus != "active" {
		t.Fatalf("undo was not rolled back:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestCreateProjectItemRollsBackWhenReopenFails(t *testing.T) {
	w := newQuestWorld(t, 1)
	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "completed"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if got := w.snapshot(t, w.firstItem.ID); got.questStatus != "done" {
		t.Fatalf("quest should be done, got %q", got.questStatus)
	}
	w.trigger(t, "inject_block_active", `CREATE TRIGGER inject_block_active BEFORE UPDATE OF status ON projects
		WHEN NEW.status = 'active' BEGIN SELECT RAISE(ABORT, 'injected'); END`)
	before := w.snapshot(t, w.firstItem.ID)

	extra := models.ProjectItem{ProjectID: w.quest.ID, Title: "Late mission", Priority: "medium"}
	if err := w.repo.CreateProjectItem(&extra); err == nil {
		t.Fatal("expected injected failure, got nil")
	}
	after := w.snapshot(t, w.firstItem.ID)
	if after != before || after.itemCount != 1 {
		t.Fatalf("item insert was not rolled back:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestCompleteLastMissionPositivePathAwardsOnce(t *testing.T) {
	w := newQuestWorld(t, 1)
	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "completed"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got := w.snapshot(t, w.firstItem.ID)
	if got.itemStatus != "completed" || got.questStatus != "done" || got.depStatus != "upcoming" || got.projectDone != 1 {
		t.Fatalf("unexpected completed state: %+v", got)
	}
	starsAfterFirst := got.totalStars

	// Reopen then complete again: the Quest bonus row exists once and the
	// mission's stars are not awarded a second time.
	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "pending"); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if err := w.repo.SetProjectItemStatus(w.quest.ID, w.firstItem.ID, "completed"); err != nil {
		t.Fatalf("recomplete: %v", err)
	}
	again := w.snapshot(t, w.firstItem.ID)
	if again.projectDone != 1 {
		t.Fatalf("project_done achievements = %d, want 1", again.projectDone)
	}
	if again.totalStars != starsAfterFirst {
		t.Fatalf("total stars changed on recomplete: %d -> %d", starsAfterFirst, again.totalStars)
	}
}
