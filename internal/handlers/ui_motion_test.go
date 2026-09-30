package handlers

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func readCSS(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../../web/static/css/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMotionViewTransitionsAndReducedMotion(t *testing.T) {
	c := readCSS(t, "ui-components.css")
	for _, want := range []string{"@view-transition { navigation: auto; }", "@media (prefers-reduced-motion: reduce) { @view-transition { navigation: none; } }"} {
		if !strings.Contains(c, want) {
			t.Errorf("ui-components.css missing %q", want)
		}
	}
	sh := readCSS(t, "ui-shell.css")
	for _, want := range []string{".gyst-topbar { view-transition-name: gyst-topbar; }", ".gyst-primary { view-transition-name: gyst-nav; }", "::view-transition-group(gyst-topbar), ::view-transition-group(gyst-nav) { animation: none; }"} {
		if !strings.Contains(sh, want) {
			t.Errorf("ui-shell.css missing %q", want)
		}
	}
	st := readCSS(t, "styles.css")
	for _, want := range []string{`:root[data-motion="reduced"]::view-transition-group(*)`, `:root[data-motion="reduced"] ::backdrop { animation: none !important; }`, "  ::backdrop { animation: none !important; }"} {
		if !strings.Contains(st, want) {
			t.Errorf("styles.css missing %q", want)
		}
	}
}

func TestMotionCheckboxTickUsesTokensAndClipOnly(t *testing.T) {
	st := readCSS(t, "styles.css")
	base := regexp.MustCompile(`(?s):root \.task input\[type="checkbox"\]::before \{(.*?)\}`).FindStringSubmatch(st)
	if base == nil || !strings.Contains(base[1], "transition: clip-path var(--vn-base) var(--vn-ease)") || !strings.Contains(base[1], "clip-path: inset(0 100% 0 0)") {
		t.Fatalf("checkbox ::before must draw via clip-path with --vn-base: %v", base)
	}
	checked := regexp.MustCompile(`:root \.task input\[type="checkbox"\]:checked::before \{(.*?)\}`).FindStringSubmatch(st)
	if checked == nil || strings.TrimSpace(checked[1]) != "clip-path: inset(0);" {
		t.Fatalf(":checked::before must only set clip-path: %v", checked)
	}
}

func TestTodayChecklistCarriesProgressCounts(t *testing.T) {
	occ := func(id int64, title string) models.TaskOccurrence {
		return models.TaskOccurrence{Task: models.Task{ID: id, Title: title}, Date: "2026-09-11"}
	}
	body := renderBody(t, "today.html", TodayData{Page: "today", Today: "2026-09-11",
		Pending:   []models.TaskOccurrence{occ(901, "Fictional pending")},
		Completed: []models.TaskOccurrence{occ(902, "Fictional done")},
		Skipped:   []models.TaskOccurrence{occ(903, "Fictional skipped")}})
	for _, want := range []string{`data-done="1" data-pending="1" data-skipped="1"`, `method="post" action="/tasks/901/complete"`, `name="occurrence_date" value="2026-09-11"`, `name="return_to"`} {
		if !strings.Contains(body, want) {
			t.Errorf("today.html missing %q", want)
		}
	}
}
