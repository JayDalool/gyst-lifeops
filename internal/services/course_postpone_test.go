package services

import (
	"sync"
	"testing"

	"lifeops-tracker/internal/models"
)

func postponeFixture(t *testing.T) (*Service, models.Course, models.LearningBlock) {
	t.Helper()
	svc := testService(t)
	course := models.Course{Name: "Fixture Course", Status: "active"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("create course: %v", err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Module 1", BlockType: "module", DueDate: "2026-06-30"}
	if err := svc.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatalf("create block: %v", err)
	}
	return svc, course, block
}

func countCatchups(t *testing.T, svc *Service, courseID int64) int {
	t.Helper()
	blocks, err := svc.Repo.ListLearningBlocks(courseID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	n := 0
	for _, b := range blocks {
		if b.IsCatchup {
			n++
		}
	}
	return n
}

func TestPostponeTwiceCreatesOneCatchup(t *testing.T) {
	svc, course, block := postponeFixture(t)
	ok, err := svc.Repo.PostponeLearningBlock(course.ID, block.ID, "2026-07-01")
	if err != nil || !ok {
		t.Fatalf("first postpone ok=%v err=%v", ok, err)
	}
	ok, err = svc.Repo.PostponeLearningBlock(course.ID, block.ID, "2026-07-01")
	if err != nil || ok {
		t.Fatalf("second postpone ok=%v err=%v, want false,nil", ok, err)
	}
	if n := countCatchups(t, svc, course.ID); n != 1 {
		t.Fatalf("catch-ups = %d, want 1", n)
	}
	got, _ := svc.Repo.LearningBlockByID(block.ID)
	if got.Status != "postponed" {
		t.Fatalf("original status = %q, want postponed", got.Status)
	}
}

func TestPostponeCompletedBlockIsNoOp(t *testing.T) {
	svc, course, block := postponeFixture(t)
	if err := svc.Repo.SetLearningBlockStatus(course.ID, block.ID, "completed"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	before, _ := svc.Repo.LearningBlockByID(block.ID)
	if before.CompletedAt == "" {
		t.Fatal("fixture: completed_at not set")
	}
	ok, err := svc.Repo.PostponeLearningBlock(course.ID, block.ID, "2026-07-01")
	if err != nil || ok {
		t.Fatalf("postpone ok=%v err=%v, want false,nil", ok, err)
	}
	after, _ := svc.Repo.LearningBlockByID(block.ID)
	if after.Status != "completed" || after.CompletedAt != before.CompletedAt {
		t.Fatalf("completed block changed: before=%+v after=%+v", before, after)
	}
	if n := countCatchups(t, svc, course.ID); n != 0 {
		t.Fatalf("catch-ups = %d, want 0", n)
	}
	through, err := svc.Repo.LearningBlocksThrough("2026-12-31")
	if err != nil {
		t.Fatalf("through: %v", err)
	}
	for _, b := range through {
		if b.IsCatchup {
			t.Fatalf("catch-up appeared on Today: %+v", b)
		}
	}
}

func TestPostponeSkippedOrPostponedIsNoOp(t *testing.T) {
	for _, status := range []string{"skipped", "postponed"} {
		svc, course, block := postponeFixture(t)
		if err := svc.Repo.SetLearningBlockStatus(course.ID, block.ID, status); err != nil {
			t.Fatalf("%s: %v", status, err)
		}
		ok, err := svc.Repo.PostponeLearningBlock(course.ID, block.ID, "2026-07-01")
		if err != nil || ok {
			t.Fatalf("%s: postpone ok=%v err=%v, want false,nil", status, ok, err)
		}
		got, _ := svc.Repo.LearningBlockByID(block.ID)
		if got.Status != status {
			t.Fatalf("%s: status became %q", status, got.Status)
		}
		if n := countCatchups(t, svc, course.ID); n != 0 {
			t.Fatalf("%s: catch-ups = %d, want 0", status, n)
		}
	}
}

func TestPostponeWrongCourseWritesNothing(t *testing.T) {
	svc, course, block := postponeFixture(t)
	ok, err := svc.Repo.PostponeLearningBlock(course.ID+999, block.ID, "2026-07-01")
	if err != nil || ok {
		t.Fatalf("postpone ok=%v err=%v, want false,nil", ok, err)
	}
	got, _ := svc.Repo.LearningBlockByID(block.ID)
	if got.Status != "pending" {
		t.Fatalf("status = %q, want pending", got.Status)
	}
	blocks, _ := svc.Repo.ListLearningBlocks(course.ID)
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
}

func TestPostponeConcurrent(t *testing.T) {
	svc, course, block := postponeFixture(t)
	const workers = 8
	var wg sync.WaitGroup
	results := make(chan bool, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := svc.Repo.PostponeLearningBlock(course.ID, block.ID, "2026-07-01")
			if err != nil {
				errs <- err
				return
			}
			results <- ok
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("postpone error: %v", err)
	}
	changed := 0
	for ok := range results {
		if ok {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("%d calls changed the block, want 1", changed)
	}
	if n := countCatchups(t, svc, course.ID); n != 1 {
		t.Fatalf("catch-ups = %d, want 1", n)
	}
}
