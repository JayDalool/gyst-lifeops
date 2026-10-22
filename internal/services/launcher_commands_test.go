package services

import (
	"regexp"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

var (
	launcherPostAllowed = regexp.MustCompile(`^/tasks/\d+/complete$|^/health/water$|^/health/training/sync$`)
	launcherDate        = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	launcherBanned      = regexp.MustCompile(`(?i)delete|archive|merge|backup|restart|deploy|shell|agent|queue|remove|reset|wipe`)
)

func launcherByID(cmds []LauncherCommand, id string) (LauncherCommand, bool) {
	for _, c := range cmds {
		if c.ID == id {
			return c, true
		}
	}
	return LauncherCommand{}, false
}

func launcherFixture(t *testing.T) *Service {
	t.Helper()
	svc := testService(t)
	pinUTC(t, svc, "2026-10-05T18:00:00Z")
	return svc
}

func TestLauncherCommandsAllowlist(t *testing.T) {
	svc := launcherFixture(t)
	if err := svc.Repo.CreateTask(&models.Task{Title: "Fictional errand", Status: "pending", DueDate: "2026-10-05", RepeatType: "none"}); err != nil {
		t.Fatal(err)
	}
	cmds, err := svc.LauncherCommands(true)
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := map[string]bool{"new-task": true, "new-project": true, "import-plan": true, "log-weight": true, "prayer-times": true, "reminders": true, "continue-learning": true, "water": true, "complete-focus": true, "sync-hevy": true}
	if len(cmds) != len(wantIDs) {
		t.Fatalf("registry has %d commands, want %d", len(cmds), len(wantIDs))
	}
	for _, c := range cmds {
		if !wantIDs[c.ID] {
			t.Errorf("unexpected command %q", c.ID)
		}
		if launcherBanned.MatchString(c.ID) || launcherBanned.MatchString(c.Label) || launcherBanned.MatchString(c.Href) || launcherBanned.MatchString(c.Action) {
			t.Errorf("unsafe command %+v", c)
		}
		switch c.Kind {
		case LauncherKindLink:
			if c.Action != "" || len(c.Fields) != 0 || !strings.HasPrefix(c.Href, "/") || strings.HasPrefix(c.Href, "//") {
				t.Errorf("bad link %+v", c)
			}
		case LauncherKindPost:
			if c.Href != "" || !launcherPostAllowed.MatchString(c.Action) || c.Confirm == "" {
				t.Errorf("bad post %+v", c)
			}
			for k, v := range c.Fields {
				if k != "occurrence_date" || !launcherDate.MatchString(v) {
					t.Errorf("field outside allowlist: %q=%q", k, v)
				}
			}
		default:
			t.Errorf("unknown kind %q", c.Kind)
		}
	}
}

func TestLauncherCommandsFocusTask(t *testing.T) {
	svc := launcherFixture(t)
	cmds, err := svc.LauncherCommands(false)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := launcherByID(cmds, "complete-focus")
	if c.DisabledReason == "" || c.Action != "" {
		t.Fatalf("no pending tasks: want disabled, got %+v", c)
	}
	for _, task := range []models.Task{
		{Title: "Fictional first", Status: "pending", DueDate: "2026-10-05", RepeatType: "none"},
		{Title: "Fictional second", Status: "pending", DueDate: "2026-10-05", RepeatType: "none"},
	} {
		task := task
		if err := svc.Repo.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
	}
	items, err := svc.TaskOccurrencesOn(svc.Today())
	if err != nil {
		t.Fatal(err)
	}
	pending, _, _ := SplitOccurrences(items)
	if len(pending) == 0 {
		t.Fatal("fixture has no pending occurrence")
	}
	cmds, _ = svc.LauncherCommands(false)
	c, _ = launcherByID(cmds, "complete-focus")
	want := pending[0]
	if c.DisabledReason != "" || c.Action != "/tasks/"+itoa(want.Task.ID)+"/complete" || c.Fields["occurrence_date"] != want.Date || c.Label != "Complete: "+want.Task.Title {
		t.Fatalf("focus command %+v does not match first pending %+v", c, want)
	}
}

func TestLauncherCommandsWaterState(t *testing.T) {
	svc := launcherFixture(t)
	cmds, _ := svc.LauncherCommands(false)
	w, _ := launcherByID(cmds, "water")
	if w.DisabledReason != "Set a bottle size in Health" {
		t.Fatalf("no plan: %+v", w)
	}
	c := cycleFixture()
	c.WaterBottleML, c.WaterTargetML = 1000, 2000
	if err := svc.CreateHealthCycle(&c); err != nil {
		t.Fatal(err)
	}
	cmds, _ = svc.LauncherCommands(false)
	w, _ = launcherByID(cmds, "water")
	if w.DisabledReason != "" || w.Action != "/health/water" {
		t.Fatalf("with plan: %+v", w)
	}
	for _, cmd := range cmds {
		text := cmd.Label + " " + cmd.Meta + " " + cmd.Confirm + " " + cmd.DisabledReason
		if cmd.ID != "complete-focus" && regexp.MustCompile(`\d`).MatchString(text) {
			t.Errorf("number in %+v", cmd)
		}
		if strings.Contains(strings.ToLower(text), "litre") {
			t.Errorf("health value in %+v", cmd)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := svc.LogHealthWaterBottle(); err != nil {
			t.Fatal(err)
		}
	}
	cmds, _ = svc.LauncherCommands(false)
	w, _ = launcherByID(cmds, "water")
	if w.DisabledReason == "" {
		t.Fatalf("target reached should disable: %+v", w)
	}
}

func TestLauncherCommandsHevy(t *testing.T) {
	svc := launcherFixture(t)
	cmds, _ := svc.LauncherCommands(false)
	if _, ok := launcherByID(cmds, "sync-hevy"); ok {
		t.Fatal("Hevy command present without Hevy")
	}
	cmds, _ = svc.LauncherCommands(true)
	if c, ok := launcherByID(cmds, "sync-hevy"); !ok || c.Action != "/health/training/sync" {
		t.Fatalf("Hevy command: %+v %v", c, ok)
	}
}

func TestLauncherContinueLearning(t *testing.T) {
	svc := launcherFixture(t)
	cmds, _ := svc.LauncherCommands(false)
	if c, _ := launcherByID(cmds, "continue-learning"); c.Href != "/courses" {
		t.Fatalf("fallback: %+v", c)
	}
	course := models.Course{Name: "Fictional course", Status: "active"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatal(err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Fictional block", BlockType: "lesson", DueDate: "2026-10-05"}
	if err := svc.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatal(err)
	}
	cmds, _ = svc.LauncherCommands(false)
	c, _ := launcherByID(cmds, "continue-learning")
	if want := "/courses/" + itoa(course.ID) + "#block-" + itoa(block.ID); c.Href != want {
		t.Fatalf("href %q want %q", c.Href, want)
	}
}

func TestFilterLauncherCommands(t *testing.T) {
	svc := launcherFixture(t)
	cmds, _ := svc.LauncherCommands(false)
	if got := FilterLauncherCommands(cmds, ""); len(got) != len(cmds) {
		t.Fatalf("empty filter: %d", len(got))
	}
	got := FilterLauncherCommands(cmds, "BOTTLE")
	if len(got) != 1 || got[0].ID != "water" {
		t.Fatalf("label filter: %+v", got)
	}
	if got := FilterLauncherCommands(cmds, "hydration"); len(got) != 1 {
		t.Fatalf("terms filter: %+v", got)
	}
	if got := FilterLauncherCommands(cmds, "zzzz"); len(got) != 0 {
		t.Fatalf("no match: %+v", got)
	}
}
