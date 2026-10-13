package handlers

import (
	"fmt"
	"net/url"
	"testing"

	"lifeops-tracker/internal/models"
)

func postponeHandlerFixture(t *testing.T) (*Handler, models.Course, models.LearningBlock) {
	t.Helper()
	h := testHandler(t)
	course := models.Course{Name: "Fixture Course", Status: "active"}
	if err := h.Service.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("course: %v", err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Module 1", BlockType: "module", DueDate: "2026-06-30"}
	if err := h.Service.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatalf("block: %v", err)
	}
	return h, course, block
}

func catchupCount(t *testing.T, h *Handler, courseID int64) int {
	t.Helper()
	blocks, err := h.Service.Repo.ListLearningBlocks(courseID)
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

func TestPostponeRouteTwiceGivesOneCatchup(t *testing.T) {
	h, course, block := postponeHandlerFixture(t)
	path := fmt.Sprintf("/courses/%d/blocks/%d/postpone", course.ID, block.ID)
	for i := 0; i < 2; i++ {
		rec := postForm(t, h, path, url.Values{"return_to": {"/today"}})
		if rec.Code != 303 {
			t.Fatalf("post %d status = %d, want 303", i+1, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/today" {
			t.Fatalf("post %d location = %q, want /today", i+1, loc)
		}
	}
	if n := catchupCount(t, h, course.ID); n != 1 {
		t.Fatalf("catch-ups = %d, want 1", n)
	}
}

func TestPostponeRouteCompletedBlockUnchanged(t *testing.T) {
	h, course, block := postponeHandlerFixture(t)
	if err := h.Service.Repo.SetLearningBlockStatus(course.ID, block.ID, "completed"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	before, _ := h.Service.Repo.LearningBlockByID(block.ID)
	rec := postForm(t, h, fmt.Sprintf("/courses/%d/blocks/%d/postpone", course.ID, block.ID), url.Values{"return_to": {"/today"}})
	if rec.Code != 303 {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	after, _ := h.Service.Repo.LearningBlockByID(block.ID)
	if after.Status != "completed" || after.CompletedAt != before.CompletedAt {
		t.Fatalf("completed block changed: before=%+v after=%+v", before, after)
	}
	if n := catchupCount(t, h, course.ID); n != 0 {
		t.Fatalf("catch-ups = %d, want 0", n)
	}
}

func TestPostponeRouteUnknownOrForeignBlock404(t *testing.T) {
	h, course, block := postponeHandlerFixture(t)
	other := models.Course{Name: "Other Course", Status: "active"}
	if err := h.Service.Repo.CreateCourse(&other); err != nil {
		t.Fatalf("other course: %v", err)
	}

	rec := postForm(t, h, fmt.Sprintf("/courses/%d/blocks/%d/postpone", course.ID, block.ID+999), url.Values{})
	if rec.Code != 404 {
		t.Fatalf("unknown block status = %d, want 404", rec.Code)
	}
	rec = postForm(t, h, fmt.Sprintf("/courses/%d/blocks/%d/postpone", other.ID, block.ID), url.Values{})
	if rec.Code != 404 {
		t.Fatalf("foreign block status = %d, want 404", rec.Code)
	}
	got, _ := h.Service.Repo.LearningBlockByID(block.ID)
	if got.Status != "pending" {
		t.Fatalf("status = %q, want pending", got.Status)
	}
	if n := catchupCount(t, h, course.ID) + catchupCount(t, h, other.ID); n != 0 {
		t.Fatalf("catch-ups = %d, want 0", n)
	}
}
