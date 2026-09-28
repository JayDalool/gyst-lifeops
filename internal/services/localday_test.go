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
