package models

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestProjectStateOf(t *testing.T) {
	for _, tt := range []struct {
		status   string
		state    ProjectState
		finished bool
		route    [3]string // progress 0, 99, 100
	}{
		{"active", ProjectStateActive, false, [3]string{"active", "active", "done"}},
		{"paused", ProjectStatePaused, false, [3]string{"locked", "locked", "locked"}},
		{"upcoming", ProjectStatePlanned, false, [3]string{"locked", "locked", "locked"}},
		{"locked", ProjectStateBlocked, false, [3]string{"locked", "locked", "locked"}},
		{"completed", ProjectStateDone, true, [3]string{"done", "done", "done"}},
		{"done", ProjectStateDone, true, [3]string{"done", "done", "done"}},
		{"archived", ProjectStateArchived, true, [3]string{"done", "done", "done"}},
		{" Done ", ProjectStateDone, true, [3]string{"done", "done", "done"}},
		{"", ProjectStateActive, false, [3]string{"active", "active", "done"}},
		{"blocked", ProjectStateActive, false, [3]string{"active", "active", "done"}},
		{"bogus", ProjectStateActive, false, [3]string{"active", "active", "done"}},
	} {
		if got := ProjectStateOf(tt.status); got != tt.state {
			t.Errorf("ProjectStateOf(%q) = %q, want %q", tt.status, got, tt.state)
		}
		if got := ProjectStateOf(tt.status).Finished(); got != tt.finished {
			t.Errorf("ProjectStateOf(%q).Finished() = %v, want %v", tt.status, got, tt.finished)
		}
		for i, progress := range []int{0, 99, 100} {
			if got := RouteState(tt.status, progress); got != tt.route[i] {
				t.Errorf("RouteState(%q, %d) = %q, want %q", tt.status, progress, got, tt.route[i])
			}
		}
	}
}

// oldRouteState is the services.RouteState that GYST-VN-037 replaced, kept
// verbatim as the parity oracle for every stored status.
func oldRouteState(status string, progress int) string {
	switch status {
	case "completed", "done", "archived":
		return "done"
	case "locked", "upcoming", "paused":
		return "locked"
	case "blocked":
		return "blocked"
	default:
		if progress >= 100 {
			return "done"
		}
		return "active"
	}
}

// oldGalaxyComplete is the replaced services.galaxyProjectComplete.
func oldGalaxyComplete(status string) bool {
	return status == "done" || status == "completed" || status == "archived"
}

func TestRouteStateParityWithEveryStoredStatus(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/005_fix_project_status_check.sql")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`status TEXT NOT NULL DEFAULT 'active' CHECK \(status IN \(([^)]*)\)\)`).FindSubmatch(sql)
	if m == nil {
		t.Fatal("status CHECK not found in migration 005")
	}
	var stored []string
	for _, v := range strings.Split(string(m[1]), ",") {
		stored = append(stored, strings.Trim(strings.TrimSpace(v), "'"))
	}
	if len(stored) != 7 {
		t.Fatalf("stored statuses changed (%v): extend the parity table and the mapping", stored)
	}
	for _, status := range stored {
		for _, progress := range []int{0, 99, 100} {
			if got, want := RouteState(status, progress), oldRouteState(status, progress); got != want {
				t.Errorf("RouteState(%q, %d) = %q, old = %q", status, progress, got, want)
			}
		}
		if got, want := ProjectStateOf(status).Finished(), oldGalaxyComplete(status); got != want {
			t.Errorf("Finished(%q) = %v, old galaxyProjectComplete = %v", status, got, want)
		}
	}
}
