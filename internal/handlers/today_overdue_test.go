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

func makeOverdue(t *testing.T, h *Handler, title, due string) models.Task {
	t.Helper()
	task := models.Task{Title: title, DueDate: due, RepeatType: "none", ReminderTime: "09:30"}
	if err := h.Service.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	return task
}

func dueOf(t *testing.T, h *Handler, id int64) string {
	t.Helper()
	got, err := h.Service.Repo.TaskByID(id)
	if err != nil {
		t.Fatal(err)
	}
	return got.DueDate
}

// For every local-day case: a task due that day is not overdue, one due the
// day before is, and Do today stores the case's local day (never UTC's).
func TestDoTodayUsesAppLocalDay(t *testing.T) {
	for _, tc := range localDayCases {
		t.Run(tc.name, func(t *testing.T) {
			h := testHandler(t)
			pinClock(t, h, tc.instant)
			same := makeOverdue(t, h, "Fictional errand", tc.day)
			late := makeOverdue(t, h, "Fictional chore", prevDay(t, tc.day))
			items, err := h.Service.OverdueTasks()
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 1 || items[0].Task.ID != late.ID || items[0].Since != "since yesterday" {
				t.Fatalf("overdue = %#v; want only the task due the day before", items)
			}
			// A hostile form date is ignored.
			rec := postForm(t, h, fmt.Sprintf("/tasks/%d/do-today", late.ID), url.Values{"due_date": {"2030-01-01"}, "return_to": {"/today#checklist"}})
			if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/today#checklist" {
				t.Fatalf("do-today = %d %q", rec.Code, rec.Header().Get("Location"))
			}
			if got := dueOf(t, h, late.ID); got != tc.day {
				t.Fatalf("due_date = %s; want %s", got, tc.day)
			}
			if got := dueOf(t, h, same.ID); got != tc.day {
				t.Fatalf("not-overdue task changed to %s", got)
			}
			// The moved task is now in today's checklist.
			if page := getProjects(t, h, "/today"); !strings.Contains(page, "Fictional chore") {
				t.Fatal("moved task missing from /today")
			}
		})
	}
}

func prevDay(t *testing.T, day string) string {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.AddDate(0, 0, -1).Format("2006-01-02")
}

func TestDoTodayRepeatIsNoOp(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-01-16T06:30:00Z")
	task := makeOverdue(t, h, "Fictional once", "2026-01-10")
	for i := 0; i < 2; i++ {
		if rec := postForm(t, h, fmt.Sprintf("/tasks/%d/do-today", task.ID), url.Values{}); rec.Code != http.StatusSeeOther {
			t.Fatalf("call %d = %d", i+1, rec.Code)
		}
		if got := dueOf(t, h, task.ID); got != "2026-01-16" {
			t.Fatalf("call %d due = %s", i+1, got)
		}
	}
	// Time moves on: the moved task is overdue again only on a later day.
	pinClock(t, h, "2026-01-16T05:30:00Z")
	postForm(t, h, fmt.Sprintf("/tasks/%d/do-today", task.ID), url.Values{})
	if got := dueOf(t, h, task.ID); got != "2026-01-16" {
		t.Fatalf("a not-overdue task was moved backwards to %s", got)
	}
}

func TestDoTodayGuards(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)
	pinClock(t, h, "2026-01-16T06:30:00Z")
	task := makeOverdue(t, h, "Fictional guarded", "2026-01-10")
	path := fmt.Sprintf("/tasks/%d/do-today", task.ID)

	do := func(method, path string, headers map[string]string, withCookie bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if withCookie {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		h.Authenticated().ServeHTTP(rec, req)
		return rec
	}
	same := map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://example.com"}

	for name, headers := range map[string]map[string]string{
		"cross-site":     {"Sec-Fetch-Site": "cross-site"},
		"foreign origin": {"Origin": "https://untrusted.example"},
	} {
		if rec := do("POST", path, headers, true); rec.Code != http.StatusForbidden {
			t.Errorf("%s = %d, want 403", name, rec.Code)
		}
		if got := dueOf(t, h, task.ID); got != "2026-01-10" {
			t.Fatalf("%s wrote due_date %s", name, got)
		}
	}
	if rec := do("POST", path, same, false); rec.Code != http.StatusUnauthorized {
		t.Errorf("no session = %d, want 401", rec.Code)
	}
	if rec := do("GET", path, nil, true); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET = %d, want 405", rec.Code)
	}
	if rec := do("POST", "/tasks/9999/do-today", same, true); rec.Code != http.StatusNotFound {
		t.Errorf("unknown id = %d, want 404", rec.Code)
	}
	if got := dueOf(t, h, task.ID); got != "2026-01-10" {
		t.Fatalf("rejected requests wrote due_date %s", got)
	}
	rec := do("POST", path, same, true)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/today#checklist" {
		t.Fatalf("same-origin = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if got := dueOf(t, h, task.ID); got != "2026-01-16" {
		t.Fatalf("due_date = %s, want 2026-01-16", got)
	}
}

func renderToday(t *testing.T, n int) string {
	t.Helper()
	data := TodayData{Page: "today", Today: "2026-06-25"}
	for i := 0; i < n; i++ {
		data.OverdueTasks = append(data.OverdueTasks, models.OverdueTask{
			Task:  models.Task{ID: int64(10 + i), Title: fmt.Sprintf("Fictional late %d", i+1), DueDate: "2026-06-20"},
			Since: "since Saturday",
		})
	}
	h := &Handler{TemplateDir: "../../web/templates"}
	rec := httptest.NewRecorder()
	h.render(rec, "today.html", data)
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "executing \"") {
		t.Fatalf("render %d: %d %s", n, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func TestTodayOverdueRendering(t *testing.T) {
	if body := renderToday(t, 0); !strings.Contains(body, "Nothing due, nothing overdue.") || strings.Contains(body, "Do today") {
		t.Error("0 overdue: want the empty line and no action")
	}
	for _, n := range []int{1, 2} {
		body := renderToday(t, n)
		for _, want := range []string{"Overdue &middot; since Saturday", `aria-label="Do today: Fictional late 1"`, `action="/tasks/10/do-today"`, `data-state="warn"`} {
			if !strings.Contains(body, want) {
				t.Errorf("%d overdue: missing %q", n, want)
			}
		}
		if strings.Contains(body, "today-overdue-group") || strings.Contains(body, "<details class=\"today-overdue") {
			t.Errorf("%d overdue: must not group", n)
		}
	}
	body := renderToday(t, 3)
	if strings.Count(body, `<details class="today-overdue-group"`) != 1 || strings.Contains(body, `<details class="today-overdue-group" open`) {
		t.Error("3 overdue: want one closed details")
	}
	for _, want := range []string{"overdue<span", "Oldest since Saturday", `href="/tasks?status=overdue"`, "All overdue"} {
		if !strings.Contains(body, want) {
			t.Errorf("3 overdue: missing %q", want)
		}
	}
	// Scope the colour check to the overdue block.
	start := strings.Index(body, "today-overdue-wrap")
	block := body[start:]
	if end := strings.Index(block, "</ul>"); end > 0 {
		block = block[:end]
	}
	for _, bad := range []string{"gyst-badge", "danger", "crit", "violet"} {
		if strings.Contains(block, bad) {
			t.Errorf("overdue block contains %q", bad)
		}
	}
}
