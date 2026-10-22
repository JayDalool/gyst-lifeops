package services

import (
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

// fixToday pins the clock to noon of date in the application timezone. It used
// to pin 00:00 UTC, which is the previous evening in Winnipeg and only worked
// while "today" was (wrongly) the UTC day (GYST-VN-003).
func fixToday(svc *Service, date string) {
	t, _ := time.ParseInLocation("2006-01-02", date, svc.Location())
	t = t.Add(12 * time.Hour)
	svc.Now = func() time.Time { return t }
}

// A self-paced course needs no schedule: a block with no due date, no times,
// and no repeat must save and list cleanly, and show up on the Learning Deck.
func TestSelfPacedCourseWithoutSchedule(t *testing.T) {
	svc := testService(t)
	course := models.Course{Name: "Complete SQL Bootcamp", Provider: "Udemy", Status: "active", ProgressPercent: 20}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("create course: %v", err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Section 4: Joins", BlockType: "lesson"}
	if err := svc.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatalf("create self-paced block: %v", err)
	}

	blocks, err := svc.Repo.ListLearningBlocks(course.ID)
	if err != nil || len(blocks) != 1 {
		t.Fatalf("list blocks: got %d err %v", len(blocks), err)
	}
	if blocks[0].DueDate != "" || blocks[0].StartTime != "" {
		t.Fatalf("self-paced block should have no schedule, got due=%q start=%q", blocks[0].DueDate, blocks[0].StartTime)
	}

	deck, err := svc.CourseDeck()
	if err != nil || len(deck) != 1 {
		t.Fatalf("deck: got %d err %v", len(deck), err)
	}
	if !deck[0].HasNextBlock || deck[0].HasNextDeadline {
		t.Fatalf("expected a next block but no deadline for unscheduled course, got block=%v deadline=%v", deck[0].HasNextBlock, deck[0].HasNextDeadline)
	}
}

// Course URL and per-block lesson URL must round-trip.
func TestCourseAndBlockURLsSave(t *testing.T) {
	svc := testService(t)
	course := models.Course{Name: "AI Course", Status: "active", URL: "https://udemy.com/ai"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("create: %v", err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Lesson 1", BlockType: "lesson", URL: "https://udemy.com/ai/lesson-1"}
	if err := svc.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatalf("create block: %v", err)
	}
	got, err := svc.Repo.CourseByID(course.ID)
	if err != nil || got.URL != "https://udemy.com/ai" {
		t.Fatalf("course url not saved: %q err %v", got.URL, err)
	}
	gotBlock, err := svc.Repo.LearningBlockByID(block.ID)
	if err != nil || gotBlock.URL != "https://udemy.com/ai/lesson-1" {
		t.Fatalf("block url not saved: %q err %v", gotBlock.URL, err)
	}
}

// Today must surface a course block due today (with course name attached).
func TestTodayShowsLearningBlock(t *testing.T) {
	svc := testService(t)
	fixToday(svc, "2026-06-30")
	course := models.Course{Name: "API Course", Status: "active"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("create course: %v", err)
	}
	due := models.LearningBlock{CourseID: course.ID, Title: "Build endpoint", BlockType: "assignment", DueDate: "2026-06-30"}
	if err := svc.Repo.CreateLearningBlock(&due); err != nil {
		t.Fatalf("create block: %v", err)
	}

	tl, err := svc.TodayLearning()
	if err != nil {
		t.Fatalf("today learning: %v", err)
	}
	if len(tl.Blocks) != 1 || tl.Blocks[0].Title != "Build endpoint" {
		t.Fatalf("expected 1 learning block today, got %d", len(tl.Blocks))
	}
	if tl.Blocks[0].CourseName != "API Course" {
		t.Fatalf("block missing course name, got %q", tl.Blocks[0].CourseName)
	}
	if tl.SuggestStudy {
		t.Fatal("should not suggest a study block when something is due")
	}
}

// Today suggests a study block (UI hint only, no row written) when an active
// course has nothing scheduled.
func TestTodaySuggestsStudyWhenIdle(t *testing.T) {
	svc := testService(t)
	fixToday(svc, "2026-06-30")
	course := models.Course{Name: "Idle Course", Status: "active"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("create: %v", err)
	}

	tl, err := svc.TodayLearning()
	if err != nil {
		t.Fatalf("today learning: %v", err)
	}
	if len(tl.Blocks) != 0 || !tl.SuggestStudy || tl.SuggestCourse.ID != course.ID {
		t.Fatalf("expected study suggestion for idle course, got blocks=%d suggest=%v", len(tl.Blocks), tl.SuggestStudy)
	}
	// The suggestion must not have inserted a block.
	blocks, _ := svc.Repo.ListLearningBlocks(course.ID)
	if len(blocks) != 0 {
		t.Fatalf("suggestion wrote a phantom block: %d", len(blocks))
	}
}

// Postponing marks the original 'postponed' and creates exactly one pending
// catch-up block due the next day.
func TestPostponeCreatesCatchup(t *testing.T) {
	svc := testService(t)
	course := models.Course{Name: "Bootcamp", Status: "active"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("create course: %v", err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Module 2", BlockType: "module", DueDate: "2026-06-30"}
	if err := svc.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatalf("create block: %v", err)
	}

	newDue := svc.CatchupDueDate("2026-06-30")
	if newDue != "2026-07-01" {
		t.Fatalf("catch-up due date = %q, want 2026-07-01", newDue)
	}
	if ok, err := svc.Repo.PostponeLearningBlock(course.ID, block.ID, newDue); err != nil || !ok {
		t.Fatalf("postpone: %v", err)
	}

	blocks, _ := svc.Repo.ListLearningBlocks(course.ID)
	var original, catchup *models.LearningBlock
	for i := range blocks {
		if blocks[i].ID == block.ID {
			original = &blocks[i]
		} else if blocks[i].IsCatchup {
			catchup = &blocks[i]
		}
	}
	if original == nil || original.Status != "postponed" {
		t.Fatalf("original block not marked postponed: %+v", original)
	}
	if catchup == nil {
		t.Fatal("no catch-up block created")
	}
	if catchup.Status != "pending" || catchup.DueDate != "2026-07-01" || !catchup.IsCatchup {
		t.Fatalf("catch-up block wrong: %+v", catchup)
	}
}

// --- ICS structural validity (the part a content-type check misses) --------

func parseICS(t *testing.T, ics string) {
	t.Helper()
	if !strings.Contains(ics, "\r\n") {
		t.Fatal("ICS must use CRLF line endings")
	}
	if strings.Count(ics, "BEGIN:VCALENDAR") != 1 || strings.Count(ics, "END:VCALENDAR") != 1 {
		t.Fatal("VCALENDAR not balanced")
	}
	begins := strings.Count(ics, "BEGIN:VEVENT")
	if begins != strings.Count(ics, "END:VEVENT") {
		t.Fatal("VEVENT begin/end not balanced")
	}
	// Each VEVENT block must carry UID, DTSTAMP, DTSTART.
	for _, chunk := range strings.Split(ics, "BEGIN:VEVENT")[1:] {
		event := chunk[:strings.Index(chunk, "END:VEVENT")]
		for _, prop := range []string{"UID:", "DTSTAMP:", "DTSTART"} {
			if !strings.Contains(event, prop) {
				t.Fatalf("VEVENT missing %s: %q", prop, event)
			}
		}
	}
}

// The unified feed must include course blocks AND project/task deadlines, and be
// structurally valid (CRLF, UID+DTSTAMP+DTSTART per event).
func TestUnifiedICSIncludesAllSources(t *testing.T) {
	svc := testService(t)
	// Migration 006 makes the default roadmap private; this test covers feed
	// structure and project events, so make that roadmap public first.
	rm, err := svc.Repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatalf("default roadmap: %v", err)
	}
	rm.Privacy = "public"
	if err := svc.Repo.UpdateRoadmap(&rm); err != nil {
		t.Fatalf("make roadmap public: %v", err)
	}

	course := models.Course{Name: "SQL, Advanced", Status: "active"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("course: %v", err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Final project", BlockType: "assignment", DueDate: "2026-07-05"}
	if err := svc.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatalf("block: %v", err)
	}
	project := models.Project{Name: "Real Estate AI", Status: "active", TargetDate: "2026-08-01"}
	if err := svc.Repo.CreateProject(&project); err != nil {
		t.Fatalf("project: %v", err)
	}
	task := models.Task{Title: "Ship API", Status: "pending", DueDate: "2026-07-10", RepeatType: "none", StarsReward: 1}
	if err := svc.Repo.CreateTask(&task); err != nil {
		t.Fatalf("task: %v", err)
	}

	ics, err := svc.AllICS("https://gyst.example")
	if err != nil {
		t.Fatalf("AllICS: %v", err)
	}
	parseICS(t, ics)

	// Known item UID present (uuid read back from the calendar query).
	blocks, _ := svc.Repo.CalendarBlocks(0)
	if len(blocks) == 0 {
		t.Fatal("no calendar blocks")
	}
	if !strings.Contains(ics, "block-"+blocks[0].UUID+"@gyst") {
		t.Fatal("course block UID missing from unified feed")
	}
	// A dated project mission item must also appear (item-<uuid>).
	if err := svc.Repo.CreateProjectItem(&models.ProjectItem{ProjectID: project.ID, Title: "Mission deadline", DueDate: "2026-07-12"}); err != nil {
		t.Fatalf("item: %v", err)
	}
	ics, err = svc.AllICS("https://gyst.example")
	if err != nil {
		t.Fatalf("AllICS2: %v", err)
	}
	parseICS(t, ics)

	for _, want := range []string{"Final project", "Target: Real Estate AI", "Ship API", "Mission deadline", "item-"} {
		if !strings.Contains(ics, want) {
			t.Fatalf("unified feed missing %q", want)
		}
	}
	// A course named with a comma must be escaped in SUMMARY.
	if !strings.Contains(ics, "SQL\\, Advanced") {
		t.Fatal("comma in SUMMARY not escaped")
	}
}

// All-day items (due-date only) use VALUE=DATE; timed blocks use a floating
// local DTSTART with a time component.
func TestICSAllDayVsTimed(t *testing.T) {
	svc := testService(t)
	course := models.Course{Name: "Live Course", Status: "active"}
	svc.Repo.CreateCourse(&course)
	svc.Repo.CreateLearningBlock(&models.LearningBlock{CourseID: course.ID, Title: "Due only", BlockType: "assignment", DueDate: "2026-07-05"})
	svc.Repo.CreateLearningBlock(&models.LearningBlock{CourseID: course.ID, Title: "Live", BlockType: "live_session", DueDate: "2026-07-06", StartTime: "14:00", EndTime: "15:30"})

	ics, err := svc.CoursesICS(course.ID, "")
	if err != nil {
		t.Fatalf("CoursesICS: %v", err)
	}
	parseICS(t, ics)
	if !strings.Contains(ics, "DTSTART;VALUE=DATE:20260705") {
		t.Fatal("due-date-only item should be all-day VALUE=DATE")
	}
	if !strings.Contains(ics, "DTSTART:20260706T140000") || !strings.Contains(ics, "DTEND:20260706T153000") {
		t.Fatalf("timed block should use floating local DTSTART/DTEND:\n%s", ics)
	}
}

// Postponed and skipped blocks are left out of both counts (GYST-148).
func TestDecorateCourseCounts(t *testing.T) {
	blk := func(status string, catchup bool) models.LearningBlock {
		return models.LearningBlock{Status: status, IsCatchup: catchup}
	}
	cases := []struct {
		name            string
		blocks          []models.LearningBlock
		count, complete int
	}{
		{"pending only", []models.LearningBlock{blk("pending", false), blk("pending", false)}, 2, 0},
		{"completed, postponed, completed catch-up", []models.LearningBlock{blk("completed", false), blk("postponed", false), blk("completed", true)}, 2, 2},
		{"postponed with pending catch-up", []models.LearningBlock{blk("postponed", false), blk("pending", true)}, 1, 0},
		{"chained later", []models.LearningBlock{blk("postponed", false), blk("postponed", true), blk("completed", true)}, 1, 1},
		{"skipped excluded", []models.LearningBlock{blk("skipped", false), blk("completed", false), blk("pending", false)}, 2, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decorateCourse(models.Course{Name: "Fictional Course"}, tc.blocks)
			if got.BlockCount != tc.count || got.CompletedBlocks != tc.complete {
				t.Fatalf("got %d of %d, want %d of %d", got.CompletedBlocks, got.BlockCount, tc.complete, tc.count)
			}
		})
	}
}

// Later on the only block, then completing its catch-up, reads "1 of 1".
func TestCourseDeckReachesAllAfterLater(t *testing.T) {
	svc := testService(t)
	course := models.Course{Name: "Fictional Course", Status: "active"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatalf("create course: %v", err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Lesson 1", BlockType: "lesson", DueDate: "2026-03-02"}
	if err := svc.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatalf("create block: %v", err)
	}
	changed, err := svc.Repo.PostponeLearningBlock(course.ID, block.ID, "2026-03-03")
	if err != nil || !changed {
		t.Fatalf("postpone: changed=%v err=%v", changed, err)
	}
	blocks, err := svc.Repo.ListLearningBlocks(course.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var catchupID int64
	for _, b := range blocks {
		if b.IsCatchup && b.Status == "pending" {
			catchupID = b.ID
		}
	}
	if catchupID == 0 {
		t.Fatalf("no pending catch-up in %+v", blocks)
	}
	if err := svc.Repo.SetLearningBlockStatus(course.ID, catchupID, "completed"); err != nil {
		t.Fatalf("complete catch-up: %v", err)
	}
	deck, err := svc.CourseDeck()
	if err != nil || len(deck) != 1 {
		t.Fatalf("deck: got %d err %v", len(deck), err)
	}
	if deck[0].BlockCount != 1 || deck[0].CompletedBlocks != deck[0].BlockCount {
		t.Fatalf("got %d of %d, want 1 of 1", deck[0].CompletedBlocks, deck[0].BlockCount)
	}
}
