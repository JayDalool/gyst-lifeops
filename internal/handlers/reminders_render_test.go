package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func getReminders(t *testing.T, h *Handler) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/reminders", nil))
	if rec.Code != 200 {
		t.Fatalf("GET /reminders: %d", rec.Code)
	}
	return rec.Body.String()
}

func TestRemindersEmptyStates(t *testing.T) {
	body := getReminders(t, testHandler(t))
	for _, want := range []string{`<div class="gyst-scope gyst-more">`, `<div class="gyst-empty"><p>No reminders due today.</p></div>`, `<div class="gyst-empty"><p>No overdue reminders.</p></div>`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if n := strings.Count(body, "<h1"); n != 1 {
		t.Fatalf("want one h1, got %d", n)
	}
	if strings.Contains(body, `data-state="warn"`) {
		t.Fatal("empty page shows an overdue chip")
	}
}

func TestRemindersRowsAndOverdueChip(t *testing.T) {
	h := testHandler(t)
	today := h.Service.TodayString()
	past := h.Service.Today().AddDate(0, 0, -3).Format("2006-01-02")
	older := h.Service.Today().AddDate(0, 0, -9).Format("2006-01-02")
	for _, task := range []models.Task{
		{Title: "Fictional due task A", DueDate: today, ReminderTime: "09:30"},
		{Title: "Fictional due task B", DueDate: today, ReminderTime: "11:00"},
		{Title: "Fictional late task A", DueDate: past, ReminderTime: "08:15"},
		{Title: "Fictional late task B", DueDate: older, ReminderTime: "07:45"},
	} {
		task.Status, task.RepeatType = "pending", "none"
		if err := h.Service.Repo.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
	}
	body := getReminders(t, h)
	for _, gone := range []string{"page-head", `class="kicker"`, "page-sub", "status late", "status soon", `class="cols`} {
		if strings.Contains(body, gone) {
			t.Fatalf("legacy markup %q still rendered", gone)
		}
	}
	dueAt, overdueAt := strings.Index(body, "Due today</h2>"), strings.Index(body, "Overdue</h2>")
	if dueAt < 0 || overdueAt < 0 || dueAt > overdueAt {
		t.Fatalf("sections out of order: due %d, overdue %d", dueAt, overdueAt)
	}
	dueHTML, overdueHTML := body[dueAt:overdueAt], body[overdueAt:]
	if n := strings.Count(body, `data-state="warn"`); n != 2 {
		t.Fatalf("want 2 warn chips, got %d", n)
	}
	for _, want := range []string{"Fictional due task A", "Fictional due task B", `<span class="gyst-row-meta">09:30</span>`, `<span class="gyst-row-meta">11:00</span>`} {
		if !strings.Contains(dueHTML, want) {
			t.Fatalf("due section missing %q", want)
		}
	}
	if strings.Contains(dueHTML, "gyst-chip") || strings.Contains(body, "No reminders due today.") || strings.Contains(body, "No overdue reminders.") {
		t.Fatal("due rows have a chip or empty copy shown beside rows")
	}
	for _, tc := range []struct{ title, date, time string }{
		{"Fictional late task A", past, "08:15"},
		{"Fictional late task B", older, "07:45"},
	} {
		start := strings.Index(overdueHTML, tc.title)
		if start < 0 {
			t.Fatalf("missing overdue row %q", tc.title)
		}
		row := overdueHTML[start:]
		row = row[:strings.Index(row, "</li>")]
		for _, want := range []string{`class="gyst-caption">` + niceDate(tc.date) + " &middot; " + tc.time + "</span>", `data-state="warn"`, `class="gyst-dot"`, "Overdue</span>"} {
			if !strings.Contains(row, want) {
				t.Fatalf("overdue row %q missing %q: %s", tc.title, want, row)
			}
		}
	}
}
