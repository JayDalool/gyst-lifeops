package handlers

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// The ICS endpoints must return 200 with a text/calendar content type and a
// real VCALENDAR body (not HTML).
func TestCalendarICSResponse(t *testing.T) {
	h := testHandler(t)

	course := models.Course{Name: "SQL Course", Status: "active"}
	if err := h.Service.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("course: %v", err)
	}
	if err := h.Service.Repo.CreateLearningBlock(&models.LearningBlock{CourseID: course.ID, Title: "Deadline", BlockType: "assignment", DueDate: "2026-07-05"}); err != nil {
		t.Fatalf("block: %v", err)
	}

	for _, path := range []string{"/calendar.ics", "/courses/calendar.ics", "/projects/calendar.ics"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("%s status %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/calendar") {
			t.Fatalf("%s content-type = %q, want text/calendar", path, ct)
		}
		if !strings.HasPrefix(rec.Body.String(), "BEGIN:VCALENDAR") {
			t.Fatalf("%s body is not an ICS calendar", path)
		}
	}
}

// /courses/{id}/calendar.ics resolves to a single course (the .ics suffix must
// not be parsed as an action).
func TestSingleCourseICSRoute(t *testing.T) {
	h := testHandler(t)
	course := models.Course{Name: "One", Status: "active"}
	h.Service.Repo.CreateCourse(&course)
	h.Service.Repo.CreateLearningBlock(&models.LearningBlock{CourseID: course.ID, Title: "L1", BlockType: "lesson", DueDate: "2026-07-05"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/courses/1/calendar.ics", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.HasPrefix(rec.Body.String(), "BEGIN:VCALENDAR") {
		t.Fatalf("single-course ics failed: status %d", rec.Code)
	}
}

func TestRenderCoursesList(t *testing.T) {
	renderOK(t, "courses_list.html", CourseListData{
		Page:    "courses",
		BaseURL: "https://gyst.example",
		Courses: []models.CourseWithProgress{
			{
				Course:          models.Course{ID: 1, Name: "SQL Bootcamp", Provider: "Udemy", Instructor: "Jose", Status: "active", ProgressPercent: 40, URL: "https://udemy.com/sql"},
				BlockCount:      6,
				CompletedBlocks: 2,
				HasNextBlock:    true,
				NextBlock:       models.LearningBlock{Title: "Joins", BlockType: "lesson"},
				HasNextDeadline: true,
				NextDeadline:    models.LearningBlock{Title: "Project", DueDate: "2026-07-10"},
			},
		},
	})
	renderOK(t, "courses_list.html", CourseListData{Page: "courses"})
}

func TestRenderCourseDetail(t *testing.T) {
	renderOK(t, "course_detail.html", CourseDetailData{
		Page:             "courses",
		Today:            "2026-06-30",
		BaseURL:          "https://gyst.example",
		BlockTypeOptions: services.BlockTypeOptions(),
		Course:           models.Course{ID: 1, Name: "AI Course", Provider: "Coursera", Status: "active", ProgressPercent: 25, URL: "https://x", Notes: "deep learning", TargetDate: "2026-09-01"},
		Blocks: []models.LearningBlock{
			{ID: 1, Title: "Lesson 1", BlockType: "lesson", Status: "completed", URL: "https://x/1"},
			{ID: 2, Title: "Study session", BlockType: "study", Status: "pending", DueDate: "2026-07-01", StartTime: "14:00", EndTime: "15:00", RepeatType: "weekly"},
			{ID: 3, Title: "Final exam", BlockType: "exam", Status: "pending", DueDate: "2026-07-20"},
			{ID: 4, Title: "Postponed bit", BlockType: "module", Status: "postponed"},
		},
		Catchups: []models.LearningBlock{
			{ID: 5, Title: "Catch up module", BlockType: "module", Status: "pending", IsCatchup: true, DueDate: "2026-07-02"},
		},
	})
	renderOK(t, "course_detail.html", CourseDetailData{Page: "courses", Course: models.Course{ID: 9, Name: "Bare", Status: "active"}, BlockTypeOptions: services.BlockTypeOptions()})
}

func TestRenderCourseForm(t *testing.T) {
	renderOK(t, "course_form.html", CourseFormData{Page: "courses", Title: "New Course", Action: "/courses/new", Course: models.Course{Status: "active"}})
	renderOK(t, "course_form.html", CourseFormData{Page: "courses", Title: "Edit Course", Action: "/courses/1/edit", IsEdit: true, Course: models.Course{ID: 1, Name: "X", Status: "paused", ProgressPercent: 50}})
}

func TestRenderTodayWithLearning(t *testing.T) {
	renderOK(t, "today.html", TodayData{
		Page:  "today",
		Today: "2026-06-30",
		Learning: models.TodayLearning{
			Blocks: []models.LearningBlock{
				{ID: 1, CourseID: 2, Title: "Watch lesson", BlockType: "lesson", CourseName: "SQL", DueDate: "2026-06-30"},
			},
		},
	})
	// suggestion path
	renderOK(t, "today.html", TodayData{
		Page:     "today",
		Today:    "2026-06-30",
		Learning: models.TodayLearning{SuggestStudy: true, SuggestCourse: models.Course{ID: 3, Name: "AI"}},
	})
}

// Guard: this feature must not pull in Google OAuth / API client dependencies.
func TestNoOAuthDependencies(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, banned := range []string{"golang.org/x/oauth2", "google.golang.org/api", "cloud.google.com"} {
		if strings.Contains(string(body), banned) {
			t.Fatalf("unexpected OAuth/Google dependency present: %s", banned)
		}
	}
}
