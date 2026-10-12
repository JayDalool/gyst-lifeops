package services

import (
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func pinUTC(t *testing.T, svc *Service, instant string) {
	t.Helper()
	at, err := time.Parse(time.RFC3339, instant)
	if err != nil {
		t.Fatal(err)
	}
	svc.Now = func() time.Time { return at }
}

func TestTaskActionDate(t *testing.T) {
	cases := []struct {
		name    string
		instant string
		task    models.Task
		want    string
	}{
		{"daily is today", "2026-10-10T17:00:00Z", models.Task{DueDate: "2026-10-01", RepeatType: "daily"}, "2026-10-10"},
		{"weekly on an off day", "2026-10-10T17:00:00Z", models.Task{DueDate: "2026-09-28", RepeatType: "weekly"}, "2026-10-05"},
		{"weekly on its day", "2026-10-12T17:00:00Z", models.Task{DueDate: "2026-09-28", RepeatType: "weekly"}, "2026-10-12"},
		{"monthly on the 31st in a 30-day month", "2026-11-30T17:00:00Z", models.Task{DueDate: "2026-08-31", RepeatType: "monthly"}, "2026-11-30"},
		{"monthly on the 31st in February", "2027-02-28T17:00:00Z", models.Task{DueDate: "2026-12-31", RepeatType: "monthly"}, "2027-02-28"},
		{"monthly before its day", "2026-10-10T17:00:00Z", models.Task{DueDate: "2026-08-15", RepeatType: "monthly"}, "2026-09-15"},
		{"yearly Feb 29 in a non-leap year", "2027-02-28T17:00:00Z", models.Task{DueDate: "2024-02-29", RepeatType: "yearly"}, "2027-02-28"},
		{"yearly Feb 29 in a leap year", "2028-03-05T17:00:00Z", models.Task{DueDate: "2024-02-29", RepeatType: "yearly"}, "2028-02-29"},
		{"series starts after today", "2026-10-10T17:00:00Z", models.Task{DueDate: "2026-10-20", RepeatType: "weekly"}, "2026-10-20"},
		{"series starts today", "2026-10-10T17:00:00Z", models.Task{DueDate: "2026-10-10", RepeatType: "daily"}, "2026-10-10"},
		{"archived repeating task", "2026-10-10T17:00:00Z", models.Task{DueDate: "2026-10-01", RepeatType: "daily", ArchivedAt: "2026-10-05 00:00:00"}, "2026-10-10"},
		{"repeating without a start", "2026-10-10T17:00:00Z", models.Task{RepeatType: "daily"}, "2026-10-10"},
		{"one-time with a due date", "2026-10-10T17:00:00Z", models.Task{DueDate: "2026-09-01", RepeatType: "none"}, "2026-09-01"},
		{"one-time without a due date", "2026-10-10T17:00:00Z", models.Task{RepeatType: "none"}, "2026-10-10"},
		// Local-day boundaries: these instants are still the evening before in Winnipeg.
		{"DST start night", "2026-03-09T04:30:00Z", models.Task{DueDate: "2026-03-01", RepeatType: "daily"}, "2026-03-08"},
		{"DST end night", "2026-11-02T05:30:00Z", models.Task{DueDate: "2026-10-01", RepeatType: "daily"}, "2026-11-01"},
		{"Saturday night before Sunday UTC", "2026-10-11T04:30:00Z", models.Task{DueDate: "2026-09-28", RepeatType: "weekly"}, "2026-10-05"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := testService(t)
			pinUTC(t, svc, tc.instant)
			if got := svc.TaskActionDate(tc.task); got != tc.want {
				t.Fatalf("TaskActionDate = %s; want %s", got, tc.want)
			}
		})
	}
}

func TestIsTaskOccurrence(t *testing.T) {
	svc := testService(t)
	pinUTC(t, svc, "2026-10-10T17:00:00Z")
	weekly := models.Task{DueDate: "2026-09-28", RepeatType: "weekly"}
	for date, want := range map[string]bool{
		"2026-09-28": true, "2026-10-05": true, "2026-10-12": true,
		"2026-10-06": false, "2026-09-21": false, "2026-09-27": false,
	} {
		if got := svc.IsTaskOccurrence(weekly, date); got != want {
			t.Errorf("weekly %s = %v; want %v", date, got, want)
		}
	}
	if !svc.IsTaskOccurrence(models.Task{DueDate: "2026-09-01", RepeatType: "none"}, "2026-10-06") {
		t.Error("a one-time task accepts any date")
	}
	if !svc.IsTaskOccurrence(models.Task{RepeatType: "daily"}, "2026-10-06") {
		t.Error("a repeating task with no start has no series to check")
	}
}

// Sunday 23:30 CDT is 04:30 UTC on Monday: a weekly task anchored on a Sunday
// is due today, one anchored on a Monday is not (GYST-VN-003).
func TestLocalDayWeeklyRecurrenceAtSundayNight(t *testing.T) {
	svc := testService(t)
	pinUTC(t, svc, "2026-03-09T04:30:00Z") // Sun 2026-03-08 23:30 CDT
	for _, task := range []models.Task{
		{Title: "Sunday review", DueDate: "2026-03-01", RepeatType: "weekly"},
		{Title: "Monday planning", DueDate: "2026-03-02", RepeatType: "weekly"},
	} {
		if err := svc.Repo.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
	}
	occurrences, err := svc.TaskOccurrencesOn(svc.Today())
	if err != nil {
		t.Fatal(err)
	}
	if len(occurrences) != 1 || occurrences[0].Task.Title != "Sunday review" || occurrences[0].Date != "2026-03-08" {
		t.Fatalf("occurrences on Sunday night = %#v", occurrences)
	}
}

// A 30-day window ending after a DST change still yields 30 distinct,
// consecutive local days.
func TestLocalDayStatsWindowAcrossDST(t *testing.T) {
	for _, instant := range []string{"2026-03-20T17:00:00Z", "2026-11-10T17:00:00Z"} {
		svc := testService(t)
		pinUTC(t, svc, instant)
		today := svc.Today()
		stats, err := svc.DailyCompletionStats(today.AddDate(0, 0, -29), today)
		if err != nil {
			t.Fatal(err)
		}
		if len(stats) != 30 {
			t.Fatalf("%s: %d days; want 30", instant, len(stats))
		}
		for i := 1; i < len(stats); i++ {
			prev, _ := time.Parse("2006-01-02", stats[i-1].Date)
			if stats[i].Date != prev.AddDate(0, 0, 1).Format("2006-01-02") {
				t.Fatalf("%s: day %d is %s after %s", instant, i, stats[i].Date, stats[i-1].Date)
			}
		}
	}
}

func TestLocalDayEveningDefaults(t *testing.T) {
	svc := testService(t)
	pinUTC(t, svc, "2026-01-16T05:30:00Z") // 2026-01-15 23:30 CST
	if got := svc.CatchupDueDate(""); got != "2026-01-16" {
		t.Fatalf("catch-up due date = %s; want the next local day", got)
	}
	goal := models.Goal{Title: "Read", Type: "yearly", TargetValue: 12, CurrentValue: 2}
	if err := svc.Repo.CreateGoal(&goal, svc.TodayString()); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Repo.GoalProgressEntries(goal.ID)
	if err != nil || len(entries) != 1 || entries[0].EntryDate != "2026-01-15" {
		t.Fatalf("initial goal progress = %#v, %v; want 2026-01-15", entries, err)
	}
	// 23:30 on Jul 31 CDT is already Aug 1 in UTC; the month is still July.
	pinUTC(t, svc, "2026-08-01T04:30:00Z")
	if today := svc.Today(); today.Month() != time.July || today.Day() != 31 {
		t.Fatalf("month-end evening local day = %s", today)
	}
}
