package handlers

import (
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func renderMonthly(t *testing.T, items []models.TaskOccurrence, progress int) string {
	t.Helper()
	h := testHandler(t)
	rec := httptest.NewRecorder()
	h.render(rec, "monthly.html", MonthlyData{
		Page:      "monthly",
		MonthName: "October 2026",
		Progress:  progress,
		Chart:     models.ChartData{Labels: []string{"1", "2", "3"}, Values: []int{0, 2, 1}},
		Items:     items,
	})
	if rec.Code != 200 {
		t.Fatalf("render status %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func occ(id int64, title, status string) models.TaskOccurrence {
	return models.TaskOccurrence{
		Task:   models.Task{ID: id, Title: title, RepeatType: "daily", Priority: "medium"},
		Date:   "2026-10-05",
		Status: status,
	}
}

func TestMonthlyShellCommon(t *testing.T) {
	body := renderMonthly(t, nil, 0)
	if n := strings.Count(body, "<h1"); n != 1 {
		t.Fatalf("want one h1, got %d", n)
	}
	main := body[strings.Index(body, "<main"):]
	if !strings.Contains(main, `<a class="gyst-button gyst-button--secondary" href="/tasks/new">New task</a>`) {
		t.Fatal("New task action missing from main")
	}
	if !strings.Contains(body, `<progress class="gyst-progress" value="0" max="100"`) {
		t.Fatal("native progress missing")
	}
	if !strings.Contains(body, "October 2026 &middot; 0% complete") && !strings.Contains(body, "October 2026 · 0% complete") {
		t.Fatal("progress caption missing")
	}
	for _, want := range []string{`class="chart"`, `data-chart="line"`, `data-labels="1|2|3"`, `data-values="0|2|1"`, `data-empty="Complete tasks to build the month's picture."`} {
		if !strings.Contains(body, want) {
			t.Fatalf("canvas attribute %q missing", want)
		}
	}
	for _, gone := range []string{"progress-row", "--xp", "btn ghost", `class="status `} {
		if strings.Contains(main, gone) {
			t.Fatalf("legacy markup %q still present", gone)
		}
	}
}

func TestMonthlyEmptyQueue(t *testing.T) {
	body := renderMonthly(t, nil, 0)
	if strings.Count(body, `class="gyst-empty"`) != 1 || !strings.Contains(body, "No tasks due this month.") {
		t.Fatal("want exactly one empty line")
	}
	if strings.Contains(body, `class="gyst-row"`) {
		t.Fatal("no rows expected")
	}
}

func TestMonthlyPendingRowHasNoChip(t *testing.T) {
	body := renderMonthly(t, []models.TaskOccurrence{occ(7, "Alpha chore", "pending")}, 0)
	main := body[strings.Index(body, "<main"):]
	if strings.Contains(main, "gyst-chip") {
		t.Fatal("pending row must not have a chip")
	}
	if !strings.Contains(main, `<a class="gyst-row" href="/tasks/7">`) {
		t.Fatal("row link missing")
	}
	if strings.Contains(main, `class="gyst-empty"`) {
		t.Fatal("empty line must not show with items")
	}
}

func TestMonthlyStateChipsWithWords(t *testing.T) {
	body := renderMonthly(t, []models.TaskOccurrence{
		occ(1, "Alpha chore", "pending"),
		occ(2, "Beta chore", "completed"),
		occ(3, "Gamma chore", "skipped"),
	}, 33)
	main := body[strings.Index(body, "<main"):]
	if n := strings.Count(main, `class="gyst-chip" data-state="ok"`); n != 1 {
		t.Fatalf("want one ok chip, got %d", n)
	}
	if n := strings.Count(main, `class="gyst-chip" data-state="idle"`); n != 1 {
		t.Fatalf("want one idle chip, got %d", n)
	}
	if !regexp.MustCompile(`data-state="ok"><span class="gyst-dot" aria-hidden="true"></span>Done</span>`).MatchString(main) {
		t.Fatal("Done chip text missing")
	}
	if !regexp.MustCompile(`data-state="idle"><span class="gyst-dot" aria-hidden="true"></span>Skipped</span>`).MatchString(main) {
		t.Fatal("Skipped chip text missing")
	}
	rows := regexp.MustCompile(`<a class="gyst-row" href="/tasks/(\d+)">`).FindAllStringSubmatch(main, -1)
	if len(rows) != 3 || rows[0][1] != "1" || rows[1][1] != "2" || rows[2][1] != "3" {
		t.Fatalf("row links wrong: %v", rows)
	}
	if !strings.Contains(main, `value="33" max="100"`) {
		t.Fatal("progress value wrong")
	}
}
