package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

// GYST-VN-003: every stored calendar day is the application's local day
// (America/Winnipeg), never the process (UTC) day. Instants are pinned in UTC,
// the production condition. Manitoba's last fall-back was 2025-11-02 and its
// last spring forward 2026-03-08; from then on it is UTC-05 all year (Issue
// #193), so there is no fall-back on 2026-11-01.
var localDayCases = []struct {
	name    string
	instant string
	day     string
}{
	{"CST 23:30", "2026-01-16T05:30:00Z", "2026-01-15"},
	{"CST 23:59", "2026-01-16T05:59:00Z", "2026-01-15"},
	{"CST 00:00", "2026-01-16T06:00:00Z", "2026-01-16"},
	{"CST 00:30", "2026-01-16T06:30:00Z", "2026-01-16"},
	{"UTC midnight in CST (18:00)", "2026-01-16T00:00:00Z", "2026-01-15"},
	{"CDT 23:30", "2026-07-16T04:30:00Z", "2026-07-15"},
	{"CDT 23:59", "2026-07-16T04:59:00Z", "2026-07-15"},
	{"CDT 00:00", "2026-07-16T05:00:00Z", "2026-07-16"},
	{"CDT 00:30", "2026-07-16T05:30:00Z", "2026-07-16"},
	{"UTC midnight in CDT (19:00)", "2026-07-16T00:00:00Z", "2026-07-15"},
	{"spring forward, last CST", "2026-03-08T07:59:00Z", "2026-03-08"},
	{"spring forward, first CDT", "2026-03-08T08:00:00Z", "2026-03-08"},
	{"spring forward day, 23:30", "2026-03-09T04:30:00Z", "2026-03-08"},
	{"2025 fall back, first 01:30 CDT", "2025-11-02T06:30:00Z", "2025-11-02"},
	{"2025 fall back, second 01:30 CST", "2025-11-02T07:30:00Z", "2025-11-02"},
	{"2025 fall back day, 23:30 CST", "2025-11-03T05:30:00Z", "2025-11-02"},
	{"2026-11-01, 01:30 once", "2026-11-01T06:30:00Z", "2026-11-01"},
	{"2026-11-01, 23:30 UTC-05", "2026-11-02T04:30:00Z", "2026-11-01"},
	{"2026-11-02, 00:30 UTC-05 (not 23:30 CST)", "2026-11-02T05:30:00Z", "2026-11-02"},
	{"winter UTC-05, 23:30", "2027-01-16T04:30:00Z", "2027-01-15"},
	{"winter UTC-05, 00:00", "2027-01-16T05:00:00Z", "2027-01-16"},
	{"2027-03-14, 23:30 (no spring forward)", "2027-03-15T04:30:00Z", "2027-03-14"},
}

// The local days above run from 2025 to 2027; the windows below cover them.
const localDayFirst, localDayLast = "2025-01-01", "2027-12-31"

func pinClock(t *testing.T, h *Handler, instant string) {
	t.Helper()
	at, err := time.Parse(time.RFC3339, instant)
	if err != nil {
		t.Fatal(err)
	}
	h.Service.Now = func() time.Time { return at }
}

func postForm(t *testing.T, h *Handler, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func nextDay(t *testing.T, day string) string {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.AddDate(0, 0, 1).Format("2006-01-02")
}

func TestLocalDayTaskCompletionDefault(t *testing.T) {
	for _, tc := range localDayCases {
		t.Run(tc.name, func(t *testing.T) {
			h := testHandler(t)
			pinClock(t, h, tc.instant)
			task := models.Task{Title: "Daily stretch", DueDate: localDayFirst, RepeatType: "daily"}
			if err := h.Service.Repo.CreateTask(&task); err != nil {
				t.Fatal(err)
			}
			rec := postForm(t, h, fmt.Sprintf("/tasks/%d/complete", task.ID), url.Values{})
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("complete status = %d: %s", rec.Code, rec.Body.String())
			}
			completions, err := h.Service.Repo.TaskCompletionsBetween(localDayFirst, localDayLast)
			if err != nil {
				t.Fatal(err)
			}
			if len(completions) != 1 || completions[0].OccurrenceDate != tc.day {
				t.Fatalf("completion dates = %#v; want one on %s", completions, tc.day)
			}
			occurrences, err := h.Service.TaskOccurrencesOn(h.Service.Today())
			if err != nil {
				t.Fatal(err)
			}
			if len(occurrences) != 1 || occurrences[0].Date != tc.day || occurrences[0].Status != "completed" {
				t.Fatalf("today's occurrence = %#v; want completed on %s", occurrences, tc.day)
			}
		})
	}
}

// Prayer check-ins (and every tracker toggle) default to the local day, which
// is also the day of the prayer schedule and of the Daily Mission shown with
// them.
func TestLocalDayPrayerCheckInDefault(t *testing.T) {
	for _, tc := range localDayCases {
		t.Run(tc.name, func(t *testing.T) {
			h := testHandler(t)
			pinClock(t, h, tc.instant)
			for _, key := range []string{"prayer_fajr", "prayer_isha"} {
				rec := postForm(t, h, "/trackers/"+key+"/toggle", url.Values{})
				if rec.Code != http.StatusSeeOther {
					t.Fatalf("toggle %s status = %d: %s", key, rec.Code, rec.Body.String())
				}
			}
			logs, err := h.Service.Repo.TrackerLogsBetween(localDayFirst, localDayLast)
			if err != nil {
				t.Fatal(err)
			}
			if len(logs) != 2 || logs[0].LogDate != tc.day || logs[1].LogDate != tc.day {
				t.Fatalf("tracker log dates = %#v; want two on %s", logs, tc.day)
			}
			schedule, err := h.Service.PrayerTimes(h.Service.Now())
			if err != nil {
				t.Fatal(err)
			}
			mission, err := h.Service.DailyMission()
			if err != nil {
				t.Fatal(err)
			}
			if schedule.Date != tc.day || mission.Date != tc.day {
				t.Fatalf("prayer day %s, mission day %s; want %s", schedule.Date, mission.Date, tc.day)
			}
			if mission.PrayerCompletion != 40 {
				t.Fatalf("prayer completion = %d; want 2 of 5 checked on the local day", mission.PrayerCompletion)
			}
		})
	}
}

func TestLocalDayHealthDay(t *testing.T) {
	for _, tc := range localDayCases {
		h := testHandler(t)
		pinClock(t, h, tc.instant)
		if day, err := h.Service.HealthTodayDate(); err != nil || day != tc.day {
			t.Errorf("%s: Health day = %s, %v; want %s", tc.name, day, err, tc.day)
		}
	}
}

func TestLocalDayQuickCaptureDefaultAndChips(t *testing.T) {
	for _, tc := range localDayCases {
		t.Run(tc.name, func(t *testing.T) {
			h := testHandler(t)
			pinClock(t, h, tc.instant)
			rec := postForm(t, h, "/tasks/quick", url.Values{"title": {"Buy stamps"}})
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("quick task status = %d: %s", rec.Code, rec.Body.String())
			}
			tasks, err := h.Service.Repo.ListTasks(false)
			if err != nil {
				t.Fatal(err)
			}
			if len(tasks) != 1 || tasks[0].DueDate != tc.day {
				t.Fatalf("quick task due dates = %#v; want %s", tasks, tc.day)
			}
			// The command palette's Today/Tomorrow chips come from the server.
			page := getProjects(t, h, "/today")
			want := fmt.Sprintf(`data-today="%s" data-tomorrow="%s"`, tc.day, nextDay(t, tc.day))
			if !strings.Contains(page, want) {
				t.Fatalf("palette dates missing %q", want)
			}
		})
	}
}

func TestLocalDayTodayOverdueAndReminder(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T00:30:00Z") // 19:30 CDT on 2026-07-15
	task := models.Task{Title: "Call the bank", DueDate: "2026-07-15", RepeatType: "none", ReminderTime: "20:00"}
	if err := h.Service.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	overdue, err := h.Service.Repo.ListActiveOneTimeOverdueTasks(h.Service.TodayString())
	if err != nil {
		t.Fatal(err)
	}
	if len(overdue) != 0 {
		t.Fatalf("a task due today is overdue in the evening: %#v", overdue)
	}
	today, _, err := h.Service.ReminderViews(h.Service.Today())
	if err != nil {
		t.Fatal(err)
	}
	if len(today) != 1 {
		t.Fatalf("today's reminders = %#v; want the 20:00 reminder", today)
	}
	if next := nextTodayReminder(today, h.Service.Now().In(h.Service.Location())); next == nil {
		t.Fatal("the next reminder card is empty in the evening")
	}
}

var malformedDates = []string{"2026-2-3", "2026-02-30", "2026-13-01", "0000-01-01", "x", "2026-01-15T00:00:00Z", "15/01/2026"}

func TestMalformedDatesAreRejectedWithoutWriting(t *testing.T) {
	for _, bad := range malformedDates {
		t.Run(bad, func(t *testing.T) {
			h := testHandler(t)
			pinClock(t, h, "2026-07-16T04:30:00Z")
			task := models.Task{Title: "Daily stretch", DueDate: "2026-01-01", RepeatType: "daily"}
			if err := h.Service.Repo.CreateTask(&task); err != nil {
				t.Fatal(err)
			}
			goal := models.Goal{Title: "Read", Type: "yearly", TargetValue: 12, StartDate: "2026-01-01", EndDate: "2026-12-31"}
			if err := h.Service.Repo.CreateGoal(&goal, h.Service.TodayString()); err != nil {
				t.Fatal(err)
			}
			requests := []struct {
				path   string
				values url.Values
			}{
				{fmt.Sprintf("/tasks/%d/complete", task.ID), url.Values{"occurrence_date": {bad}}},
				{fmt.Sprintf("/tasks/%d/skip", task.ID), url.Values{"occurrence_date": {bad}}},
				{"/trackers/prayer_fajr/toggle", url.Values{"log_date": {bad}}},
				{fmt.Sprintf("/goals/%d/progress", goal.ID), url.Values{"entry_date": {bad}, "value": {"1"}}},
				{"/tasks/quick", url.Values{"title": {"Quick"}, "due_date": {bad}}},
				{"/tasks/new", url.Values{"title": {"New"}, "due_date": {bad}}},
				{fmt.Sprintf("/tasks/%d/edit", task.ID), url.Values{"title": {"Edited"}, "due_date": {bad}, "repeat_type": {"daily"}}},
			}
			for _, req := range requests {
				if rec := postForm(t, h, req.path, req.values); rec.Code != http.StatusBadRequest {
					t.Errorf("POST %s with %q: status %d; want 400", req.path, bad, rec.Code)
				}
			}
			completions, _ := h.Service.Repo.TaskCompletionsBetween("0001-01-01", "9999-12-31")
			logs, _ := h.Service.Repo.TrackerLogsBetween("0001-01-01", "9999-12-31")
			entries, _ := h.Service.Repo.GoalProgressEntries(goal.ID)
			tasks, _ := h.Service.Repo.ListTasks(true)
			stored, _ := h.Service.Repo.TaskByID(task.ID)
			if len(completions) != 0 || len(logs) != 0 || len(entries) != 0 || len(tasks) != 1 || stored.Title != "Daily stretch" || stored.DueDate != "2026-01-01" {
				t.Fatalf("a malformed date wrote data: completions=%d logs=%d entries=%d tasks=%d stored=%q %q",
					len(completions), len(logs), len(entries), len(tasks), stored.Title, stored.DueDate)
			}
		})
	}
}

// Valid supplied dates are stored as given: back-dating is allowed and there is
// no future-date clamp. Exception: goal progress
// entries (GYST-150).
func TestValidSuppliedDatesAreKept(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T04:30:00Z") // 2026-07-15 local
	task := models.Task{Title: "Daily stretch", DueDate: "2026-01-01", RepeatType: "daily"}
	if err := h.Service.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	for path, values := range map[string]url.Values{
		fmt.Sprintf("/tasks/%d/complete", task.ID): {"occurrence_date": {"2026-07-10"}},
		"/trackers/prayer_isha/toggle":             {"log_date": {"2026-08-20"}},
		"/tasks/quick":                             {"title": {"Later"}, "due_date": {"2027-02-01"}},
		"/tasks/new":                               {"title": {"New"}, "due_date": {"2026-12-24"}},
	} {
		if rec := postForm(t, h, path, values); rec.Code != http.StatusSeeOther {
			t.Fatalf("POST %s status %d: %s", path, rec.Code, rec.Body.String())
		}
	}
	completions, _ := h.Service.Repo.TaskCompletionsBetween("0001-01-01", "9999-12-31")
	logs, _ := h.Service.Repo.TrackerLogsBetween("0001-01-01", "9999-12-31")
	if len(completions) != 1 || completions[0].OccurrenceDate != "2026-07-10" {
		t.Fatalf("completion = %#v", completions)
	}
	if len(logs) != 1 || logs[0].LogDate != "2026-08-20" {
		t.Fatalf("tracker log = %#v", logs)
	}
	due := map[string]string{}
	tasks, _ := h.Service.Repo.ListTasks(false)
	for _, task := range tasks {
		due[task.Title] = task.DueDate
	}
	if due["Later"] != "2027-02-01" || due["New"] != "2026-12-24" {
		t.Fatalf("task due dates = %v", due)
	}
}

// A new or edited task may still have no due date; only quick capture defaults
// an empty due date to today.
func TestTaskFormEmptyDueDateStaysEmpty(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-07-16T04:30:00Z")
	if rec := postForm(t, h, "/tasks/new", url.Values{"title": {"Someday"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("new task status %d: %s", rec.Code, rec.Body.String())
	}
	tasks, _ := h.Service.Repo.ListTasks(false)
	if len(tasks) != 1 || tasks[0].DueDate != "" {
		t.Fatalf("new task due date = %#v; want empty", tasks)
	}
	if rec := postForm(t, h, fmt.Sprintf("/tasks/%d/edit", tasks[0].ID), url.Values{"title": {"Someday soon"}, "due_date": {"2026-07-20"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("edit status %d: %s", rec.Code, rec.Body.String())
	}
	edited, _ := h.Service.Repo.TaskByID(tasks[0].ID)
	if edited.DueDate != "2026-07-20" {
		t.Fatalf("edited due date = %q", edited.DueDate)
	}
}

// The prayer timezone is the application timezone: it is shown read-only and a
// posted value is ignored and not stored.
func TestPrayerTimezoneIsTheApplicationTimezone(t *testing.T) {
	h := testHandler(t)
	values := url.Values{
		"latitude": {"49.8951"}, "longitude": {"-97.1384"}, "timezone": {"Asia/Tokyo"},
		"method": {"isna"}, "asr_method": {"shafi"}, "alert_sound": {"athan"},
		"adjust_fajr": {"0"}, "adjust_dhuhr": {"0"}, "adjust_asr": {"0"}, "adjust_maghrib": {"0"}, "adjust_isha": {"0"},
	}
	// Saved through the service; the real /settings/prayer route is covered in
	// prayer_settings_test.go.
	if err := h.Service.SavePrayerConfig(values); err != nil {
		t.Fatalf("save prayer settings: %v", err)
	}
	config, err := h.Service.PrayerConfig()
	if err != nil || config.Timezone != "America/Winnipeg" {
		t.Fatalf("prayer timezone = %q, %v", config.Timezone, err)
	}
	if stored, _ := h.Service.Repo.GetSetting("prayer.timezone"); stored != "" {
		t.Fatalf("posted prayer timezone was stored: %q", stored)
	}
	// An old stored value is ignored, not deleted.
	if err := h.Service.Repo.SetSetting("prayer.timezone", "Asia/Tokyo"); err != nil {
		t.Fatal(err)
	}
	if config, _ := h.Service.PrayerConfig(); config.Timezone != "America/Winnipeg" {
		t.Fatalf("old prayer.timezone row is still used: %q", config.Timezone)
	}
	if stored, _ := h.Service.Repo.GetSetting("prayer.timezone"); stored != "Asia/Tokyo" {
		t.Fatalf("old prayer.timezone row was changed: %q", stored)
	}
	page := getProjects(t, h, "/settings")
	if !strings.Contains(page, `<input value="America/Winnipeg" readonly>`) || strings.Contains(page, `name="timezone"`) {
		t.Fatal("settings must show the application timezone read-only and not submit it")
	}
}
