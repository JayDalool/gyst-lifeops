package handlers

import (
	"net/http"
	"net/http/httptest"
	"regexp"
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
	for _, task := range []models.Task{
		{Title: "Fictional due task", DueDate: today, ReminderTime: "09:30"},
		{Title: "Fictional late task", DueDate: past, ReminderTime: "08:15"},
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
	if !strings.Contains(body, `<span class="gyst-row-meta">09:30</span>`) {
		t.Fatal("due-today row lost its time")
	}
	if strings.Contains(body, "No reminders due today.") || strings.Contains(body, "No overdue reminders.") {
		t.Fatal("empty copy shown beside rows")
	}
	late := regexp.MustCompile(`(?s)<li><div class="gyst-row">(?:[^<]|<[^l]|<l[^i])*?Fictional late task.*?</li>`).FindString(body)
	for _, want := range []string{`class="gyst-caption"`, "08:15", `data-state="warn"`, `class="gyst-dot"`, "Overdue</span>"} {
		if !strings.Contains(late, want) {
			t.Fatalf("overdue row missing %q: %s", want, late)
		}
	}
	due := regexp.MustCompile(`(?s)<li><div class="gyst-row">(?:[^<]|<[^l]|<l[^i])*?Fictional due task.*?</li>`).FindString(body)
	if due == "" || strings.Contains(due, "gyst-chip") {
		t.Fatalf("due-today row missing or has a chip: %s", due)
	}
}
