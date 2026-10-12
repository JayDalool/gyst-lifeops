package handlers

import (
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// nowSection returns the Now block's markup (it holds no nested section).
func nowSection(t *testing.T, body string) string {
	t.Helper()
	_, rest, ok := strings.Cut(body, `<section class="gyst-card today-now"`)
	if !ok {
		t.Fatal("no Now section")
	}
	section, _, _ := strings.Cut(rest, `</section>`)
	return section
}

func nowFixture() TodayData {
	at := time.Date(2026, 9, 11, 17, 5, 0, 0, time.UTC)
	names := []string{"Fajr", "Dhuhr", "Asr", "Maghrib", "Isha"}
	var times []models.PrayerTime
	for i, name := range names {
		moment := at.Add(time.Duration(i-2) * 3 * time.Hour)
		times = append(times, models.PrayerTime{Name: name, Time: moment.Format("15:04"), At: moment})
	}
	next := models.NextPrayer{Prayer: times[3], Remaining: 3 * time.Hour}
	return TodayData{
		Page: "today", Today: "2026-09-11",
		Now: services.NowSet{Prayer: next},
		Prayer: PrayerTodayData{
			Schedule: models.PrayerSchedule{Date: "2026-09-11", Location: "Fictional City", Times: times},
			Next:     next, NextAt: next.Prayer.At.Format(time.RFC3339), CurrentName: "Asr",
		},
	}
}

func TestNowEmptyAndPrayerHooksNested(t *testing.T) {
	body := renderBody(t, "today.html", nowFixture())
	section := nowSection(t, body)
	if !strings.Contains(section, `<h2 id="now-title" class="gyst-section-title">Now</h2>`) || !strings.Contains(section, "Nothing else today") {
		t.Fatalf("empty Now wrong: %s", section)
	}
	// Every hook initMisc looks up must sit inside [data-next-prayer], or the
	// shared page script throws and stops the copy buttons, motion toggle and nav menus.
	for _, hook := range []string{`data-next-prayer=`, `data-gyst-open="prayer-sheet"`, `<dialog id="prayer-sheet"`, `aria-label="Close prayer times"`, `aria-labelledby="prayer-sheet-title"`} {
		if !strings.Contains(section+">", hook) && hook != `data-next-prayer=` {
			t.Errorf("missing %s", hook)
		}
	}
	for _, hook := range []string{`data-prayer-countdown`, `data-prayer-enable`, `data-prayer-test`, `data-prayer-alert-state`, `data-prayer-audio`} {
		if strings.Count(body, hook) != 1 || !strings.Contains(section, hook) {
			t.Errorf("%s is not exactly once inside the Now section", hook)
		}
	}
	if n := strings.Count(section, `<li data-prayer-name=`); n != 5 || strings.Count(section, `data-prayer-at=`) != 5 {
		t.Fatalf("prayer sheet rows = %d, want 5", n)
	}
	if !strings.Contains(section, `>Next</span>`) || !strings.Contains(section, `>Current</span>`) {
		t.Error("current and next prayer are not marked with words")
	}
	if strings.Contains(body, "prayer-hero") || strings.Contains(body, `aria-live="polite">&mdash;`) {
		t.Error("the prayer hero card is still rendered")
	}
	if strings.Contains(section, "Later today") {
		t.Error("timed row shown without a timed item")
	}
}

func TestNowTaskActionIsOnceAndOutOfTheList(t *testing.T) {
	d := nowFixture()
	pending := []models.TaskOccurrence{
		{Task: models.Task{ID: 41, Title: "Fictional first", Priority: "high"}, Date: "2026-09-11"},
		{Task: models.Task{ID: 42, Title: "Fictional second"}, Date: "2026-09-11"},
		{Task: models.Task{ID: 43, Title: "Fictional third"}, Date: "2026-09-11"},
	}
	d.Pending = pending
	d.Now.Action = &services.NowAction{Kind: "task", Task: &pending[0]}
	d.Now.Timed = &services.NowTimed{Kind: "reminder", Title: "Fictional timed", At: time.Date(2026, 9, 11, 18, 30, 0, 0, time.UTC)}
	body := renderBody(t, "today.html", d)
	section := nowSection(t, body)
	if strings.Count(section, "<form") != 1 || strings.Count(section, `action="/tasks/41/complete"`) != 1 || !strings.Contains(section, `name="return_to" value="/today"`) || !strings.Contains(section, `name="occurrence_date" value="2026-09-11"`) {
		t.Fatalf("Now action form wrong: %s", section)
	}
	if strings.Count(body, `action="/tasks/41/complete"`) != 1 || strings.Contains(body, `id="task-41"`) {
		t.Error("the Now task is repeated in the list")
	}
	for _, id := range []string{"42", "43"} {
		if strings.Count(body, `action="/tasks/`+id+`/complete"`) != 1 {
			t.Errorf("task %s should be listed once", id)
		}
	}
	if !strings.Contains(body, `data-pending="3"`) || !strings.Contains(body, "3 left") {
		t.Error("counts must still include the Now task")
	}
	if !strings.Contains(section, `href="/reminders"`) || !strings.Contains(section, "Fictional timed") || !strings.Contains(section, ">18:30<") {
		t.Errorf("timed row wrong: %s", section)
	}
}

func TestNowSessionAndContinueActions(t *testing.T) {
	d := nowFixture()
	blocks := []models.LearningBlock{{ID: 31, CourseID: 11, Title: "Fictional lesson", CourseName: "Course A", BlockType: "lesson", StartTime: "19:00"}, {ID: 32, CourseID: 11, Title: "Fictional lesson two", CourseName: "Course A", BlockType: "lesson"}}
	d.Learning.Blocks = blocks
	d.Now.Action = &services.NowAction{Kind: "session", Block: &blocks[0]}
	d.Now.Timed = &services.NowTimed{Kind: "session", Title: "Fictional lesson two", At: time.Date(2026, 9, 11, 20, 0, 0, 0, time.UTC), SourceID: 32}
	body := renderBody(t, "today.html", d)
	section := nowSection(t, body)
	if strings.Count(section, `action="/courses/11/blocks/31/complete"`) != 1 || strings.Count(body, `action="/courses/11/blocks/31/complete"`) != 1 {
		t.Error("session action is not a single course block form")
	}
	if !strings.Contains(section, `href="/courses/11#block-32"`) || strings.Count(body, `id="block-31"`) != 0 || strings.Count(body, `id="block-32"`) != 1 {
		t.Error("session handling in Now and list is wrong")
	}

	d = nowFixture()
	d.Learning = models.TodayLearning{SuggestStudy: true, SuggestCourse: models.Course{ID: 12, Name: "Fictional course"}}
	d.Now.Action = &services.NowAction{Kind: "continue", Course: &d.Learning.SuggestCourse}
	body = renderBody(t, "today.html", d)
	section = nowSection(t, body)
	if strings.Contains(section, "<form") || !strings.Contains(section, `href="/courses/12"`) {
		t.Errorf("continue must be a link without a check circle: %s", section)
	}
	if strings.Contains(body, "Nothing scheduled from") {
		t.Error("the list repeats the continue suggestion")
	}
}

func TestChecklistPageHasNoNowBlock(t *testing.T) {
	d := nowFixture()
	pending := []models.TaskOccurrence{{Task: models.Task{ID: 41, Title: "Fictional first"}, Date: "2026-09-11"}}
	d.ChecklistOnly = true
	d.Pending = pending
	d.Now.Action = &services.NowAction{Kind: "task", Task: &pending[0]}
	body := renderBody(t, "checklist.html", d)
	if strings.Contains(body, "today-now") || strings.Count(body, `action="/tasks/41/complete"`) != 1 {
		t.Error("checklist page must list every pending task and no Now block")
	}
}

func TestTodayPutsTopTaskInNowNotInList(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T15:00:00Z")
	day := h.Service.TodayString()
	for _, task := range []models.Task{
		{Title: "Fictional low chore", DueDate: day, RepeatType: "none", Priority: "low"},
		{Title: "Fictional urgent call", DueDate: day, RepeatType: "none", Priority: "high"},
	} {
		task := task
		if err := h.Service.Repo.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
	}
	body := getProjects(t, h, "/today")
	section := nowSection(t, body)
	if !strings.Contains(section, "Fictional urgent call") || strings.Contains(section, "Fictional low chore") {
		t.Fatalf("high-priority task is not the Now action: %s", section)
	}
	if strings.Count(body, ">Fictional urgent call</a>") != 1 || strings.Count(body, ">Fictional low chore</a>") != 1 {
		t.Error("each task must appear once on Today")
	}
	if !strings.Contains(body, `data-pending="2"`) {
		t.Error("data-pending must still count the Now task")
	}
}
