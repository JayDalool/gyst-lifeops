package handlers

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

type CourseListData struct {
	Page    string
	Courses []models.CourseWithProgress
	BaseURL string
}

type CourseDetailData struct {
	Page             string
	Course           models.Course
	Blocks           []models.LearningBlock
	Catchups         []models.LearningBlock
	BlockTypeOptions []services.BlockTypeOption
	Today            string
	BaseURL          string
}

type CourseFormData struct {
	Page   string
	Title  string
	Action string
	Course models.Course
	IsEdit bool
}

// courses dispatches everything under /courses, matching the .ics sub-routes
// before parsing a segment as an ID.
func (h *Handler) courses(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		h.courseList(w, r)
		return
	}
	if len(segments) == 2 && segments[1] == "new" {
		h.newCourse(w, r)
		return
	}
	if len(segments) == 2 && segments[1] == "calendar.ics" {
		h.coursesCalendarICS(w, r)
		return
	}

	id, ok := parseID(segmentAt(segments, 1))
	if !ok {
		http.NotFound(w, r)
		return
	}

	if len(segments) == 2 {
		h.courseDetail(w, r, id)
		return
	}
	if len(segments) == 3 && segments[2] == "calendar.ics" {
		h.courseCalendarICS(w, r, id)
		return
	}

	switch segments[2] {
	case "edit":
		h.editCourse(w, r, id)
	case "status":
		h.courseStatus(w, r, id)
	case "archive":
		h.archiveCourse(w, r, id)
	case "blocks":
		h.courseBlocks(w, r, id, segments)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) courseList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	deck, err := h.Service.CourseDeck()
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "courses_list.html", CourseListData{
		Page:    "courses",
		Courses: deck,
		BaseURL: requestBaseURL(r),
	})
}

func (h *Handler) courseDetail(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	course, blocks, err := h.Service.CourseDetail(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	var catchups []models.LearningBlock
	for _, b := range blocks {
		if b.IsCatchup && b.Status == "pending" {
			catchups = append(catchups, b)
		}
	}
	h.render(w, "course_detail.html", CourseDetailData{
		Page:             "courses",
		Course:           course,
		Blocks:           blocks,
		Catchups:         catchups,
		BlockTypeOptions: services.BlockTypeOptions(),
		Today:            h.Service.TodayString(),
		BaseURL:          requestBaseURL(r),
	})
}

func (h *Handler) newCourse(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.render(w, "course_form.html", CourseFormData{
			Page:   "courses",
			Title:  "New Course",
			Action: "/courses/new",
			Course: models.Course{Status: "active"},
		})
	case http.MethodPost:
		course, err := courseFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		if err := h.Service.Repo.CreateCourse(&course); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/courses/%d", course.ID), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) editCourse(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodGet:
		course, err := h.Service.Repo.CourseByID(id)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		h.render(w, "course_form.html", CourseFormData{
			Page:   "courses",
			Title:  "Edit Course",
			Action: fmt.Sprintf("/courses/%d/edit", id),
			Course: course,
			IsEdit: true,
		})
	case http.MethodPost:
		course, err := courseFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		course.ID = id
		if err := h.Service.Repo.UpdateCourse(&course); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/courses/%d", id), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) courseStatus(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	status := cleanForm(r.FormValue("status"), "active")
	if err := h.Service.Repo.SetCourseStatus(id, status); err != nil {
		serverError(w, err)
		return
	}
	redirectAfter(w, r, fmt.Sprintf("/courses/%d", id))
}

func (h *Handler) archiveCourse(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := h.Service.Repo.ArchiveCourse(id); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/courses", http.StatusSeeOther)
}

func (h *Handler) courseBlocks(w http.ResponseWriter, r *http.Request, courseID int64, segments []string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	// POST /courses/{id}/blocks  -> create a learning block
	if len(segments) == 3 {
		block, err := learningBlockFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		block.CourseID = courseID
		if err := h.Service.Repo.CreateLearningBlock(&block); err != nil {
			serverError(w, err)
			return
		}
		redirectAfter(w, r, fmt.Sprintf("/courses/%d#block-%d", courseID, block.ID))
		return
	}

	// POST /courses/{id}/blocks/{blockID}/{action}
	if len(segments) != 5 {
		http.NotFound(w, r)
		return
	}
	blockID, ok := parseID(segments[3])
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch segments[4] {
	case "complete":
		if err := h.Service.Repo.SetLearningBlockStatus(courseID, blockID, "completed"); err != nil {
			serverError(w, err)
			return
		}
	case "reopen":
		if err := h.Service.Repo.SetLearningBlockStatus(courseID, blockID, "pending"); err != nil {
			serverError(w, err)
			return
		}
	case "skip":
		if err := h.Service.Repo.SetLearningBlockStatus(courseID, blockID, "skipped"); err != nil {
			serverError(w, err)
			return
		}
	case "postpone":
		block, err := h.Service.Repo.LearningBlockByID(blockID)
		if err != nil {
			serverError(w, err)
			return
		}
		newDue := h.Service.CatchupDueDate(block.DueDate)
		if err := h.Service.Repo.PostponeLearningBlock(courseID, blockID, newDue); err != nil {
			serverError(w, err)
			return
		}
	case "delete":
		if err := h.Service.Repo.DeleteLearningBlock(courseID, blockID); err != nil {
			serverError(w, err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	redirectAfter(w, r, fmt.Sprintf("/courses/%d#block-%d", courseID, blockID))
}

// --- ICS export handlers ---------------------------------------------------

func (h *Handler) writeICS(w http.ResponseWriter, content, filename string) {
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\""+filename+"\"")
	_, _ = w.Write([]byte(content))
}

func (h *Handler) allCalendarICS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ics, err := h.Service.AllICS(requestBaseURL(r))
	if err != nil {
		serverError(w, err)
		return
	}
	h.writeICS(w, ics, "gyst.ics")
}

func (h *Handler) coursesCalendarICS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ics, err := h.Service.CoursesICS(0, requestBaseURL(r))
	if err != nil {
		serverError(w, err)
		return
	}
	h.writeICS(w, ics, "gyst-courses.ics")
}

func (h *Handler) courseCalendarICS(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ics, err := h.Service.CoursesICS(id, requestBaseURL(r))
	if err != nil {
		serverError(w, err)
		return
	}
	h.writeICS(w, ics, fmt.Sprintf("gyst-course-%d.ics", id))
}

func (h *Handler) projectsCalendarICS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ics, err := h.Service.ProjectsICS(requestBaseURL(r))
	if err != nil {
		serverError(w, err)
		return
	}
	h.writeICS(w, ics, "gyst-projects.ics")
}

// calendarFeed serves the tokenized, subscription-safe feeds:
//
//	/calendar/feed/{token}.ics           unified
//	/calendar/feed/{token}/courses.ics   courses only
//	/calendar/feed/{token}/projects.ics  projects + tasks
//
// A wrong or absent token 404s (never reveals whether a token exists). These
// bypass app auth on purpose so an external calendar fetcher can reach them; the
// unguessable token is the only gate.
func (h *Handler) calendarFeed(w http.ResponseWriter, r *http.Request, segments []string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if len(segments) < 3 || segments[1] != "feed" {
		http.NotFound(w, r)
		return
	}

	var token, scope string
	switch len(segments) {
	case 3: // {token}.ics  -> unified
		if !strings.HasSuffix(segments[2], ".ics") {
			http.NotFound(w, r)
			return
		}
		token = strings.TrimSuffix(segments[2], ".ics")
		scope = "all"
	case 4: // {token}/{scope}.ics
		token = segments[2]
		switch segments[3] {
		case "courses.ics":
			scope = "courses"
		case "projects.ics":
			scope = "projects"
		default:
			http.NotFound(w, r)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}

	if !h.validCalendarToken(token) {
		http.NotFound(w, r)
		return
	}

	base := requestBaseURL(r)
	var ics string
	var err error
	switch scope {
	case "courses":
		ics, err = h.Service.CoursesICS(0, base)
	case "projects":
		ics, err = h.Service.ProjectsICS(base)
	default:
		ics, err = h.Service.AllICS(base)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	h.writeICS(w, ics, "gyst.ics")
}

// validCalendarToken compares the supplied token against the stored one in
// constant time. Empty/unset tokens never validate.
func (h *Handler) validCalendarToken(token string) bool {
	if token == "" {
		return false
	}
	stored, err := h.Service.CalendarToken()
	if err != nil || stored == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(token)) == 1
}

// --- form parsing + helpers ------------------------------------------------

func courseFromRequest(r *http.Request) (models.Course, error) {
	if err := r.ParseForm(); err != nil {
		return models.Course{}, err
	}
	return models.Course{
		Name:            strings.TrimSpace(r.FormValue("name")),
		Provider:        strings.TrimSpace(r.FormValue("provider")),
		Instructor:      strings.TrimSpace(r.FormValue("instructor")),
		URL:             strings.TrimSpace(r.FormValue("url")),
		Status:          cleanForm(r.FormValue("status"), "active"),
		ProgressPercent: parseIntForm(r.FormValue("progress_percent"), 0),
		TargetDate:      strings.TrimSpace(r.FormValue("target_date")),
		Notes:           strings.TrimSpace(r.FormValue("notes")),
	}, nil
}

func learningBlockFromRequest(r *http.Request) (models.LearningBlock, error) {
	if err := r.ParseForm(); err != nil {
		return models.LearningBlock{}, err
	}
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	return models.LearningBlock{
		Title:      strings.TrimSpace(r.FormValue("title")),
		BlockType:  cleanForm(r.FormValue("block_type"), "lesson"),
		Status:     "pending",
		StartTime:  strings.TrimSpace(r.FormValue("start_time")),
		EndTime:    strings.TrimSpace(r.FormValue("end_time")),
		DueDate:    strings.TrimSpace(r.FormValue("due_date")),
		URL:        strings.TrimSpace(r.FormValue("url")),
		RepeatType: cleanForm(r.FormValue("repeat_type"), "none"),
		Notes:      strings.TrimSpace(r.FormValue("notes")),
		SortOrder:  sortOrder,
	}, nil
}

// requestBaseURL reconstructs the public origin so ICS descriptions can link
// back to GYST. Honors a reverse proxy's X-Forwarded-Proto.
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
