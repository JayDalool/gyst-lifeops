package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-133: repeating tasks ticked from /tasks or /tasks/{id} book the current
// occurrence, not the series start. Fictional data only.

const occurrenceClock = "2026-10-10T17:00:00Z" // Saturday 2026-10-10 local

func makeSeries(t *testing.T, h *Handler, title, due, repeat string) models.Task {
	t.Helper()
	task := models.Task{Title: title, DueDate: due, RepeatType: repeat, StarsReward: 3}
	if err := h.Service.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	return task
}

func getPage(t *testing.T, h *Handler, path string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status %d: %s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func completionRows(t *testing.T, h *Handler) []models.TaskCompletion {
	t.Helper()
	rows, err := h.Service.Repo.TaskCompletionsBetween("0001-01-01", "9999-12-31")
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// postedDate pulls the occurrence_date the page would post for one task form.
func postedDates(t *testing.T, body, action string) []string {
	t.Helper()
	var dates []string
	for _, chunk := range strings.Split(body, "<form")[1:] {
		if !strings.Contains(chunk, `action="`+action+`"`) {
			continue
		}
		const marker = `name="occurrence_date" value="`
		i := strings.Index(chunk, marker)
		if i < 0 {
			t.Fatalf("no occurrence_date in %s form", action)
		}
		rest := chunk[i+len(marker):]
		dates = append(dates, rest[:strings.Index(rest, `"`)])
	}
	return dates
}

func TestRepeatingTaskPagesPostTheCurrentOccurrence(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, occurrenceClock)
	daily := makeSeries(t, h, "Daily stretch", "2026-10-01", "daily")
	weekly := makeSeries(t, h, "Weekly review", "2026-09-28", "weekly")
	once := makeSeries(t, h, "One-time errand", "2026-10-03", "none")

	list := getPage(t, h, "/tasks")
	for _, tc := range []struct {
		task models.Task
		want string
	}{{daily, "2026-10-10"}, {weekly, "2026-10-05"}, {once, "2026-10-03"}} {
		for _, verb := range []string{"complete", "skip"} {
			got := postedDates(t, list, fmt.Sprintf("/tasks/%d/%s", tc.task.ID, verb))
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("/tasks %s for %q posts %v; want %s", verb, tc.task.Title, got, tc.want)
			}
		}
		detail := getPage(t, h, fmt.Sprintf("/tasks/%d", tc.task.ID))
		for _, verb := range []string{"complete", "skip"} {
			got := postedDates(t, detail, fmt.Sprintf("/tasks/%d/%s", tc.task.ID, verb))
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("detail %s for %q posts %v; want %s", verb, tc.task.Title, got, tc.want)
			}
		}
	}

	pinClock(t, h, "2026-10-12T17:00:00Z") // Monday, the weekly task's day
	got := postedDates(t, getPage(t, h, "/tasks"), fmt.Sprintf("/tasks/%d/complete", weekly.ID))
	if len(got) != 1 || got[0] != "2026-10-12" {
		t.Errorf("weekly on its day posts %v; want 2026-10-12", got)
	}
}

func TestCompleteAndSkipFromTaskPagesBookTodayNotSeriesStart(t *testing.T) {
	for _, page := range []string{"list", "detail"} {
		for _, verb := range []string{"complete", "skip"} {
			t.Run(page+"/"+verb, func(t *testing.T) {
				h := testHandler(t)
				pinClock(t, h, occurrenceClock)
				task := makeSeries(t, h, "Daily stretch", "2026-10-01", "daily")
				before := stars(t, h)

				var body string
				if page == "list" {
					body = getPage(t, h, "/tasks")
				} else {
					body = getPage(t, h, fmt.Sprintf("/tasks/%d", task.ID))
				}
				action := fmt.Sprintf("/tasks/%d/%s", task.ID, verb)
				date := postedDates(t, body, action)[0]
				rec := postQuestForm(t, h, action, url.Values{"occurrence_date": {date}})

				rows := completionRows(t, h)
				if len(rows) != 1 || rows[0].OccurrenceDate != "2026-10-10" {
					t.Fatalf("rows = %#v; want exactly one for 2026-10-10", rows)
				}
				if verb == "complete" {
					if rows[0].Status != "completed" || stars(t, h) != before+3 {
						t.Fatalf("status %q, stars %d -> %d; want completed and +3", rows[0].Status, before, stars(t, h))
					}
					if got := locationOf(t, rec).Query().Get("celebrate_date"); got != "2026-10-10" {
						t.Fatalf("celebrate_date = %q; want 2026-10-10", got)
					}
				} else if rows[0].Status != "skipped" || stars(t, h) != before {
					t.Fatalf("status %q, stars %d -> %d; want skipped and no XP", rows[0].Status, before, stars(t, h))
				}
			})
		}
	}
}

func TestEmptyOccurrenceDateOnRepeatingTaskBooksCurrentOccurrence(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, occurrenceClock)
	weekly := makeSeries(t, h, "Weekly review", "2026-09-28", "weekly")
	rec := postQuestForm(t, h, fmt.Sprintf("/tasks/%d/complete", weekly.ID), url.Values{"occurrence_date": {""}})
	rows := completionRows(t, h)
	if len(rows) != 1 || rows[0].OccurrenceDate != "2026-10-05" {
		t.Fatalf("rows = %#v; want one for 2026-10-05", rows)
	}
	if got := locationOf(t, rec).Query().Get("celebrate_date"); got != "2026-10-05" {
		t.Fatalf("celebrate_date = %q", got)
	}
}

func TestNonOccurrenceDateIsRejectedWithoutWriting(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, occurrenceClock)
	weekly := makeSeries(t, h, "Weekly review", "2026-09-28", "weekly")
	before := stars(t, h)
	for _, date := range []string{"2026-10-06", "2026-09-21", "2026-09-27"} { // Tuesday, before start, Sunday
		for _, verb := range []string{"complete", "skip"} {
			rec := postForm(t, h, fmt.Sprintf("/tasks/%d/%s", weekly.ID, verb), url.Values{"occurrence_date": {date}})
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s %s status %d; want 400", verb, date, rec.Code)
			}
		}
	}
	for _, date := range []string{"2026-2-3", "2026-02-30", "garbage"} {
		if rec := postForm(t, h, fmt.Sprintf("/tasks/%d/complete", weekly.ID), url.Values{"occurrence_date": {date}}); rec.Code != http.StatusBadRequest {
			t.Errorf("malformed %q status %d; want 400", date, rec.Code)
		}
	}
	if rows := completionRows(t, h); len(rows) != 0 || stars(t, h) != before {
		t.Fatalf("rejected posts wrote rows %#v / stars %d -> %d", rows, before, stars(t, h))
	}
	// A valid past occurrence is still saved as posted.
	postQuestForm(t, h, fmt.Sprintf("/tasks/%d/complete", weekly.ID), url.Values{"occurrence_date": {"2026-09-28"}})
	if rows := completionRows(t, h); len(rows) != 1 || rows[0].OccurrenceDate != "2026-09-28" {
		t.Fatalf("back-dated rows = %#v", rows)
	}
}

func TestUndoRemovesTheBookedOccurrenceAndItsXP(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, occurrenceClock)
	task := makeSeries(t, h, "Daily stretch", "2026-10-01", "daily")
	before := stars(t, h)
	rec := postQuestForm(t, h, fmt.Sprintf("/tasks/%d/complete", task.ID), url.Values{"occurrence_date": {"2026-10-10"}})
	q := locationOf(t, rec).Query()
	if stars(t, h) != before+3 {
		t.Fatalf("stars after complete = %d", stars(t, h))
	}
	undo := url.Values{"occurrence_date": {q.Get("celebrate_date")}}
	postQuestForm(t, h, q.Get("celebrate_undo"), undo)
	if rows := completionRows(t, h); len(rows) != 0 || stars(t, h) != before {
		t.Fatalf("after undo rows %#v, stars %d; want none and %d", rows, stars(t, h), before)
	}
	postQuestForm(t, h, q.Get("celebrate_undo"), undo) // second undo changes nothing
	if rows := completionRows(t, h); len(rows) != 0 || stars(t, h) != before {
		t.Fatalf("after second undo rows %#v, stars %d", rows, stars(t, h))
	}
}

func TestOverdueFilterListsOnlyOneTimeTasks(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, occurrenceClock)
	makeSeries(t, h, "Daily stretch", "2026-10-01", "daily")
	makeSeries(t, h, "Late errand", "2026-10-03", "none")
	makeSeries(t, h, "Future errand", "2026-10-20", "none")

	overdue := getPage(t, h, "/tasks?status=overdue")
	if !strings.Contains(overdue, "Late errand") || strings.Contains(overdue, "Daily stretch") || strings.Contains(overdue, "Future errand") {
		t.Fatalf("overdue view lists the wrong tasks: %s", overdue)
	}
	all := getPage(t, h, "/tasks")
	if got := strings.Count(all, `class="task-late">overdue`); got != 1 {
		t.Fatalf("overdue labels = %d; want 1 (the one-time task)", got)
	}
}
