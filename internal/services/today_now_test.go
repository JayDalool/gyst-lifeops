package services

import (
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func nowTask(id int64, priority, clock string) models.TaskOccurrence {
	return models.TaskOccurrence{Task: models.Task{ID: id, Title: "Task " + string(rune('A'+id)), Priority: priority, ReminderTime: clock}, Date: "2026-09-11", Status: "pending"}
}

func TestSelectNowTaskOrder(t *testing.T) {
	now := winnipegTime(t, "2026-09-11 08:00")
	tests := []struct {
		name    string
		pending []models.TaskOccurrence
		wantID  int64
	}{
		{"high beats medium regardless of time", []models.TaskOccurrence{nowTask(1, "medium", "09:00"), nowTask(2, "high", "18:00")}, 2},
		{"medium beats low and blank", []models.TaskOccurrence{nowTask(1, "low", "09:00"), nowTask(2, "", "09:00"), nowTask(3, "medium", "")}, 3},
		{"earlier time wins at equal priority", []models.TaskOccurrence{nowTask(1, "high", "15:00"), nowTask(2, "high", "09:30")}, 2},
		{"timed beats untimed", []models.TaskOccurrence{nowTask(1, "high", ""), nowTask(2, "high", "23:00")}, 2},
		{"lower ID wins on full tie", []models.TaskOccurrence{nowTask(7, "high", "09:00"), nowTask(3, "high", "09:00")}, 3},
		{"lower ID wins when untimed", []models.TaskOccurrence{nowTask(9, "", ""), nowTask(4, "", "")}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectNow(NowInput{Now: now, Pending: tt.pending})
			if got.Action == nil || got.Action.Kind != "task" || got.Action.Task.Task.ID != tt.wantID {
				t.Fatalf("action = %#v, want task %d", got.Action, tt.wantID)
			}
		})
	}
}

func TestSelectNowNeverChoosesFinishedTasks(t *testing.T) {
	now := winnipegTime(t, "2026-09-11 08:00")
	done, skipped, open := nowTask(1, "high", "07:00"), nowTask(2, "high", "07:00"), nowTask(3, "", "")
	done.Status, skipped.Status = "completed", "skipped"
	got := SelectNow(NowInput{Now: now, Pending: []models.TaskOccurrence{done, skipped, open}})
	if got.Action == nil || got.Action.Task.Task.ID != 3 {
		t.Fatalf("action = %#v", got.Action)
	}
	if got := SelectNow(NowInput{Now: now, Pending: []models.TaskOccurrence{done, skipped}}); got.Action != nil {
		t.Fatalf("finished tasks chosen: %#v", got.Action)
	}
}

func TestSelectNowFallbacks(t *testing.T) {
	now := winnipegTime(t, "2026-09-11 08:00")
	blocks := []models.LearningBlock{{ID: 5, Title: "Unit one", Status: "pending"}, {ID: 6, Title: "Unit two", Status: "pending"}}
	course := models.Course{ID: 2, Name: "Sample course"}

	got := SelectNow(NowInput{Now: now, Learning: models.TodayLearning{Blocks: blocks, SuggestStudy: true, SuggestCourse: course}})
	if got.Action == nil || got.Action.Kind != "session" || got.Action.Block.ID != 5 {
		t.Fatalf("block fallback = %#v", got.Action)
	}
	got = SelectNow(NowInput{Now: now, Learning: models.TodayLearning{SuggestStudy: true, SuggestCourse: course}})
	if got.Action == nil || got.Action.Kind != "continue" || got.Action.Course.ID != 2 {
		t.Fatalf("continue fallback = %#v", got.Action)
	}
	if got := SelectNow(NowInput{Now: now}); got.Action != nil || got.Timed != nil {
		t.Fatalf("empty input = %#v", got)
	}
	got = SelectNow(NowInput{Now: now, Pending: []models.TaskOccurrence{nowTask(1, "", "")}, Learning: models.TodayLearning{Blocks: blocks}})
	if got.Action.Kind != "task" {
		t.Fatalf("task should beat block: %#v", got.Action)
	}
}

// Ported from internal/handlers TestNextTodayReminder (GYST-079 deletes that copy).
func TestSelectNowTimedPortedReminderCases(t *testing.T) {
	loc := time.FixedZone("local", -5*3600)
	now := time.Date(2026, 9, 11, 14, 0, 0, 0, loc)
	rows := []models.ReminderView{
		{Title: "past", ReminderDate: "2026-09-11", ReminderTime: "09:00"},
		{Title: "later", ReminderDate: "2026-09-11", ReminderTime: "18:00"},
		{Title: "next", ReminderDate: "2026-09-11", ReminderTime: "15:00"},
		{Title: "tomorrow", ReminderDate: "2026-09-12", ReminderTime: "14:01"},
		{Title: "untimed", ReminderDate: "2026-09-11"},
		{Title: "garbled", ReminderDate: "2026-09-11", ReminderTime: "soon"},
	}
	if got := SelectNow(NowInput{Now: now, Reminders: rows}).Timed; got == nil || got.Title != "next" || got.Kind != "reminder" {
		t.Fatalf("next = %#v", got)
	}
	if got := SelectNow(NowInput{Now: now.Add(5 * time.Hour), Reminders: rows}).Timed; got != nil {
		t.Fatalf("past reminder presented as upcoming: %#v", got)
	}
}

func TestSelectNowTimedBoundaries(t *testing.T) {
	now := winnipegTime(t, "2026-09-11 14:00")
	exactly := []models.ReminderView{{Title: "exactly now", ReminderDate: "2026-09-11", ReminderTime: "14:00"}}
	if got := SelectNow(NowInput{Now: now, Reminders: exactly}).Timed; got != nil {
		t.Fatalf("reminder at exactly now chosen: %#v", got)
	}
	if got := SelectNow(NowInput{Now: now.Add(-time.Minute), Reminders: exactly}).Timed; got == nil {
		t.Fatal("reminder one minute ahead skipped")
	}
}

func TestSelectNowTimedEarlierOfReminderAndBlock(t *testing.T) {
	now := winnipegTime(t, "2026-09-11 08:00")
	reminders := []models.ReminderView{{SourceType: "reminder", SourceID: 4, Title: "Call back", ReminderDate: "2026-09-11", ReminderTime: "13:00"}}
	blocks := []models.LearningBlock{{ID: 8, Title: "Lecture", Status: "pending", DueDate: "2026-09-11", StartTime: "11:30"}}
	got := SelectNow(NowInput{Now: now, Reminders: reminders, Learning: models.TodayLearning{Blocks: blocks}})
	// The block is the fallback action, so it is excluded; the reminder is Timed.
	if got.Action == nil || got.Action.Kind != "session" || got.Timed == nil || got.Timed.Title != "Call back" {
		t.Fatalf("got action %#v timed %#v", got.Action, got.Timed)
	}
	// With a task as the action the earlier block is Timed.
	got = SelectNow(NowInput{Now: now, Pending: []models.TaskOccurrence{nowTask(1, "", "")}, Reminders: reminders, Learning: models.TodayLearning{Blocks: blocks}})
	if got.Timed == nil || got.Timed.Kind != "session" || got.Timed.SourceID != 8 || got.Timed.SourceType != "learning_block" {
		t.Fatalf("timed = %#v", got.Timed)
	}
	// An overdue catch-up block (earlier due date) is never Timed.
	blocks[0].DueDate = "2026-09-10"
	got = SelectNow(NowInput{Now: now, Pending: []models.TaskOccurrence{nowTask(1, "", "")}, Learning: models.TodayLearning{Blocks: blocks}})
	if got.Timed != nil {
		t.Fatalf("overdue block timed: %#v", got.Timed)
	}
}

func TestSelectNowChosenTaskIsNeverTimed(t *testing.T) {
	now := winnipegTime(t, "2026-09-11 08:00")
	task := nowTask(1, "high", "10:00")
	other := nowTask(2, "medium", "11:00")
	reminders := []models.ReminderView{
		{SourceType: "task", SourceID: 1, Title: "Task B", ReminderDate: "2026-09-11", ReminderTime: "10:00"},
		{SourceType: "task", SourceID: 2, Title: "Task C", ReminderDate: "2026-09-11", ReminderTime: "11:00"},
	}
	got := SelectNow(NowInput{Now: now, Pending: []models.TaskOccurrence{other, task}, Reminders: reminders})
	if got.Action.Task.Task.ID != 1 || got.Timed == nil || got.Timed.SourceID != 2 {
		t.Fatalf("action %#v timed %#v", got.Action, got.Timed)
	}
	// A standalone reminder that shares the task's ID is not excluded.
	got = SelectNow(NowInput{Now: now, Pending: []models.TaskOccurrence{task}, Reminders: []models.ReminderView{{SourceType: "reminder", SourceID: 1, Title: "Other", ReminderDate: "2026-09-11", ReminderTime: "12:00"}}})
	if got.Timed == nil || got.Timed.Title != "Other" {
		t.Fatalf("standalone excluded: %#v", got.Timed)
	}
}

func TestSelectNowTimedTieOrder(t *testing.T) {
	now := winnipegTime(t, "2026-09-11 08:00")
	reminders := []models.ReminderView{
		{SourceType: "reminder", SourceID: 9, Title: "Zed", ReminderDate: "2026-09-11", ReminderTime: "10:00"},
		{SourceType: "reminder", SourceID: 7, Title: "Alpha", ReminderDate: "2026-09-11", ReminderTime: "10:00"},
		{SourceType: "reminder", SourceID: 3, Title: "Alpha", ReminderDate: "2026-09-11", ReminderTime: "10:00"},
	}
	blocks := []models.LearningBlock{{ID: 1, Title: "Aardvark session", Status: "pending", DueDate: "2026-09-11", StartTime: "10:00"}}
	in := NowInput{Now: now, Pending: []models.TaskOccurrence{nowTask(50, "", "")}, Reminders: reminders, Learning: models.TodayLearning{Blocks: blocks}}

	if got := SelectNow(in).Timed; got.Kind != "reminder" || got.Title != "Alpha" || got.SourceID != 3 {
		t.Fatalf("tie order = %#v", got)
	}
	in.Reminders = []models.ReminderView{reminders[2], reminders[0], reminders[1]}
	if got := SelectNow(in).Timed; got.SourceID != 3 {
		t.Fatalf("order-dependent: %#v", got)
	}
	in.Reminders = nil
	if got := SelectNow(in).Timed; got.Kind != "session" || got.SourceID != 1 {
		t.Fatalf("session alone = %#v", got)
	}
	// Same kind, same title: lower source ID; different titles: title order.
	in.Reminders = reminders[:2]
	if got := SelectNow(in).Timed; got.Title != "Alpha" || got.SourceID != 7 {
		t.Fatalf("title then ID = %#v", got)
	}
}

func TestSelectNowMidnight(t *testing.T) {
	svc := testService(t)
	late, early := winnipegTime(t, "2026-09-11 23:59"), winnipegTime(t, "2026-09-11 00:01")
	latePrayer, err := svc.NextPrayer(late)
	if err != nil {
		t.Fatal(err)
	}
	earlyPrayer, err := svc.NextPrayer(early)
	if err != nil {
		t.Fatal(err)
	}
	tomorrow := []models.ReminderView{{Title: "Early tomorrow", ReminderDate: "2026-09-12", ReminderTime: "00:05"}}
	got := SelectNow(NowInput{Now: late, Reminders: tomorrow, Prayer: latePrayer})
	if got.Timed != nil {
		t.Fatalf("tomorrow item at 23:59: %#v", got.Timed)
	}
	if !got.Prayer.Tomorrow || got.Prayer.Prayer.Name != "Fajr" || got.Prayer.Prayer.At.In(late.Location()).Format("2006-01-02") != "2026-09-12" {
		t.Fatalf("23:59 prayer = %#v", got.Prayer)
	}
	got = SelectNow(NowInput{Now: early, Prayer: earlyPrayer})
	if got.Prayer.Tomorrow || got.Prayer.Prayer.Name != "Fajr" || got.Prayer.Prayer.At.In(early.Location()).Format("2006-01-02") != "2026-09-11" {
		t.Fatalf("00:01 prayer = %#v", got.Prayer)
	}
}

func TestTodayNowPrayerBoundary(t *testing.T) {
	svc := testService(t)
	schedule, err := svc.PrayerTimes(winnipegTime(t, "2026-09-11 12:00"))
	if err != nil {
		t.Fatal(err)
	}
	var dhuhr time.Time
	for _, prayer := range schedule.Times {
		if prayer.Name == "Dhuhr" {
			dhuhr = prayer.At
		}
	}
	if dhuhr.IsZero() {
		t.Fatal("no Dhuhr in schedule")
	}
	pinUTC(t, svc, dhuhr.Add(-time.Second).UTC().Format(time.RFC3339Nano))
	got, err := svc.TodayNow()
	if err != nil {
		t.Fatal(err)
	}
	if got.Prayer.Prayer.Name != "Dhuhr" || got.Prayer.Remaining != time.Second {
		t.Fatalf("1s before Dhuhr = %#v", got.Prayer)
	}
	pinUTC(t, svc, dhuhr.UTC().Format(time.RFC3339Nano))
	got, err = svc.TodayNow()
	if err != nil {
		t.Fatal(err)
	}
	if got.Prayer.Prayer.Name != "Asr" {
		t.Fatalf("exactly Dhuhr = %#v", got.Prayer)
	}
}

func TestTodayNowGathersFromDatabase(t *testing.T) {
	svc := testService(t)
	pinUTC(t, svc, "2026-09-11T14:00:00Z") // 09:00 CDT
	for _, task := range []models.Task{
		{Title: "Sample low", DueDate: "2026-09-11", Priority: "low"},
		{Title: "Sample high", DueDate: "2026-09-11", Priority: "high", ReminderTime: "17:00"},
		{Title: "Sample overdue", DueDate: "2026-09-01", Priority: "high"},
	} {
		if err := svc.Repo.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.TodayNow()
	if err != nil {
		t.Fatal(err)
	}
	if got.Action == nil || got.Action.Task.Task.Title != "Sample high" {
		t.Fatalf("action = %#v", got.Action)
	}
	if got.Timed != nil {
		t.Fatalf("chosen task also timed: %#v", got.Timed)
	}
}
