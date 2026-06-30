package services

import (
	"strconv"

	"lifeops-tracker/internal/models"
)

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

// BlockTypeLabel maps the stored block_type to the broader self-paced UI
// language (away from university "lecture" framing).
func BlockTypeLabel(t string) string {
	switch t {
	case "lesson":
		return "Lesson"
	case "module":
		return "Module"
	case "study":
		return "Study Block"
	case "assignment":
		return "Deadline"
	case "review":
		return "Review Block"
	case "exam":
		return "Test / Exam"
	case "live_session":
		return "Live Session"
	default:
		return "Learning Block"
	}
}

// BlockTypeOption is one choice in the type-driven add-block form.
type BlockTypeOption struct {
	Value string
	Label string
}

// BlockTypeOptions powers the add-block form; one form, type-driven.
func BlockTypeOptions() []BlockTypeOption {
	return []BlockTypeOption{
		{"lesson", "Lesson / Module item"},
		{"module", "Module"},
		{"study", "Study Block"},
		{"review", "Review Block"},
		{"assignment", "Deadline / Deliverable"},
		{"exam", "Test / Exam"},
		{"live_session", "Live Session"},
	}
}

// CourseDeck returns non-archived courses decorated with block counts and the
// next pending block / next deadline for the Learning Deck.
func (s *Service) CourseDeck() ([]models.CourseWithProgress, error) {
	courses, err := s.Repo.ListCourses(false)
	if err != nil {
		return nil, err
	}
	deck := make([]models.CourseWithProgress, 0, len(courses))
	for _, c := range courses {
		blocks, err := s.Repo.ListLearningBlocks(c.ID)
		if err != nil {
			return nil, err
		}
		deck = append(deck, decorateCourse(c, blocks))
	}
	return deck, nil
}

func decorateCourse(c models.Course, blocks []models.LearningBlock) models.CourseWithProgress {
	cw := models.CourseWithProgress{Course: c}
	for _, b := range blocks {
		cw.BlockCount++
		if b.Status == "completed" {
			cw.CompletedBlocks++
		}
		if b.Status != "pending" {
			continue
		}
		if !cw.HasNextBlock {
			cw.NextBlock = b
			cw.HasNextBlock = true
		}
		if b.DueDate != "" {
			if !cw.HasNextDeadline || b.DueDate < cw.NextDeadline.DueDate {
				cw.NextDeadline = b
				cw.HasNextDeadline = true
			}
		}
	}
	return cw
}

// CourseDetail returns a course with all its (non-archived) learning blocks.
func (s *Service) CourseDetail(id int64) (models.Course, []models.LearningBlock, error) {
	course, err := s.Repo.CourseByID(id)
	if err != nil {
		return models.Course{}, nil, err
	}
	blocks, err := s.Repo.ListLearningBlocks(id)
	if err != nil {
		return models.Course{}, nil, err
	}
	return course, blocks, nil
}

// TodayLearning is the compact Learning Missions payload: blocks due today or
// overdue (incl. catch-ups). When nothing is due but an active course exists, it
// flags a study-block suggestion — a UI hint only, never an inserted row.
func (s *Service) TodayLearning() (models.TodayLearning, error) {
	blocks, err := s.Repo.LearningBlocksThrough(s.TodayString())
	if err != nil {
		return models.TodayLearning{}, err
	}
	tl := models.TodayLearning{Blocks: blocks}
	if len(blocks) == 0 {
		courses, err := s.Repo.ListCourses(false)
		if err != nil {
			return models.TodayLearning{}, err
		}
		for _, c := range courses {
			if c.Status == "active" {
				tl.SuggestStudy = true
				tl.SuggestCourse = c
				break
			}
		}
	}
	return tl, nil
}

// CatchupDueDate computes where a postponed block lands: the day after its
// original due date, or tomorrow when it had none.
func (s *Service) CatchupDueDate(originalDue string) string {
	if originalDue != "" {
		return nextDay(originalDue)
	}
	return formatDate(s.Today().AddDate(0, 0, 1))
}
