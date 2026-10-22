package services

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

type calPriv struct {
	svc *Service
	db  *sql.DB
	t   *testing.T
}

func newCalPriv(t *testing.T) *calPriv {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	// Migration 006 makes the default roadmap private; this test covers
	// project-level flags, so make that roadmap public first.
	if _, err := database.Exec(`UPDATE roadmaps SET privacy = 'public'`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	svc := New(repository.New(database))
	svc.Now = func() time.Time { return time.Date(2026, time.June, 24, 12, 0, 0, 0, time.UTC) }
	return &calPriv{svc: svc, db: database, t: t}
}

// project creates a project with a target date, one dated item and one dated
// milestone, all named after the project.
func (c *calPriv) project(name string, parent int64) int64 {
	c.t.Helper()
	p := models.Project{Name: name, Status: "active", TargetDate: "2026-08-01", ParentProjectID: parent}
	if err := c.svc.Repo.CreateProject(&p); err != nil {
		c.t.Fatal(err)
	}
	if err := c.svc.Repo.CreateProjectItem(&models.ProjectItem{ProjectID: p.ID, Title: name + " item", DueDate: "2026-07-12"}); err != nil {
		c.t.Fatal(err)
	}
	if _, err := c.db.Exec(`INSERT INTO roadmap_milestones (project_id, title, due_date) VALUES (?, ?, '2026-07-20')`, p.ID, name+" milestone"); err != nil {
		c.t.Fatal(err)
	}
	return p.ID
}

func (c *calPriv) flags(id int64, private, local bool) {
	c.t.Helper()
	if _, err := c.db.Exec(`INSERT OR IGNORE INTO roadmap_project_meta (project_id) VALUES (?)`, id); err != nil {
		c.t.Fatal(err)
	}
	if _, err := c.db.Exec(`UPDATE roadmap_project_meta SET is_private = ?, local_only = ? WHERE project_id = ?`, private, local, id); err != nil {
		c.t.Fatal(err)
	}
}

func (c *calPriv) feeds() map[string]string {
	c.t.Helper()
	all, err := c.svc.AllICS("https://gyst.example")
	if err != nil {
		c.t.Fatal(err)
	}
	proj, err := c.svc.ProjectsICS("https://gyst.example")
	if err != nil {
		c.t.Fatal(err)
	}
	parseICS(c.t, all)
	parseICS(c.t, proj)
	return map[string]string{"all": all, "projects": proj}
}

func (c *calPriv) assertHidden(name string) {
	c.t.Helper()
	for feed, ics := range c.feeds() {
		if strings.Contains(ics, name) {
			c.t.Errorf("%s feed leaks %q", feed, name)
		}
	}
}

func (c *calPriv) assertShown(name string) {
	c.t.Helper()
	for feed, ics := range c.feeds() {
		for _, want := range []string{"Target: " + name, name + " item", name + " milestone"} {
			if !strings.Contains(ics, want) {
				c.t.Errorf("%s feed missing %q", feed, want)
			}
		}
	}
}

func TestFeedsOmitPrivateAndLocalOnlyProjects(t *testing.T) {
	for _, mode := range []string{"private", "local-only"} {
		t.Run(mode, func(t *testing.T) {
			c := newCalPriv(t)
			c.project("Fictional Bakery Site", 0)
			hid := c.project("Fictional Garden Planner", 0)
			c.flags(hid, mode == "private", mode == "local-only")

			hiddenFeeds := c.feeds()
			c.assertHidden("Fictional Garden Planner")
			c.assertShown("Fictional Bakery Site")

			// Making it public restores its events, and the public sibling's
			// UID and text lines are unchanged by the filtering.
			c.flags(hid, false, false)
			for feed, ics := range c.feeds() {
				if !strings.Contains(ics, "Fictional Garden Planner") {
					t.Errorf("%s feed: making project public must restore events", feed)
				}
				for _, line := range strings.Split(hiddenFeeds[feed], "\r\n") {
					if strings.HasPrefix(line, "UID:") || strings.HasPrefix(line, "SUMMARY:") || strings.HasPrefix(line, "DESCRIPTION:") {
						if !strings.Contains(ics, line) {
							t.Errorf("%s feed: line %q changed", feed, line)
						}
					}
				}
			}
		})
	}
}

func TestFeedsOmitDescendantsOfPrivateParent(t *testing.T) {
	c := newCalPriv(t)
	parent := c.project("Fictional Atlas Main", 0)
	child := c.project("Fictional Atlas Phase", parent)
	c.project("Fictional Atlas Subphase", child)
	c.project("Fictional Open Book", 0)
	c.flags(parent, true, false)
	c.assertHidden("Fictional Atlas")
	c.assertShown("Fictional Open Book")
	c.flags(parent, false, false)
	c.assertShown("Fictional Atlas Main")
	c.assertShown("Fictional Atlas Phase")
	c.assertShown("Fictional Atlas Subphase")
}

func TestFeedsOmitChildOfArchivedPrivateParent(t *testing.T) {
	c := newCalPriv(t)
	parent := c.project("Fictional Vault Main", 0)
	c.project("Fictional Vault Phase", parent)
	c.flags(parent, false, true)
	if _, err := c.db.Exec(`UPDATE projects SET archived_at = CURRENT_TIMESTAMP WHERE id = ?`, parent); err != nil {
		t.Fatal(err)
	}
	c.assertHidden("Fictional Vault")
}

func TestPrivacyDoesNotAffectTasksRemindersOrCourseBlocks(t *testing.T) {
	c := newCalPriv(t)
	hid := c.project("Fictional Garden Planner", 0)
	c.flags(hid, true, true)
	course := models.Course{Name: "Fictional Course", Status: "active"}
	if err := c.svc.Repo.CreateCourse(&course); err != nil {
		t.Fatal(err)
	}
	if err := c.svc.Repo.CreateLearningBlock(&models.LearningBlock{CourseID: course.ID, Title: "Fictional Block", BlockType: "assignment", DueDate: "2026-07-05"}); err != nil {
		t.Fatal(err)
	}
	if err := c.svc.Repo.CreateTask(&models.Task{Title: "Fictional Task", Status: "pending", DueDate: "2026-07-10", RepeatType: "none", StarsReward: 1}); err != nil {
		t.Fatal(err)
	}
	all := c.feeds()["all"]
	for _, want := range []string{"Fictional Block", "Fictional Task"} {
		if !strings.Contains(all, want) {
			t.Errorf("unified feed missing %q", want)
		}
	}
	c.assertHidden("Fictional Garden Planner")
}

func TestFeedFailsClosedOnPrivacyError(t *testing.T) {
	c := newCalPriv(t)
	c.project("Fictional Bakery Site", 0)
	if _, err := c.db.Exec(`DROP TABLE roadmap_project_meta`); err != nil {
		t.Fatal(err)
	}
	if out, err := c.svc.AllICS("https://gyst.example"); err == nil {
		t.Fatalf("AllICS: expected error, got feed:\n%s", out)
	}
	if out, err := c.svc.ProjectsICS("https://gyst.example"); err == nil {
		t.Fatalf("ProjectsICS: expected error, got feed:\n%s", out)
	}
}

// A project flipped private, or imported private, after the data reads must
// not leak: privacy is read last.
func TestFeedDoesNotLeakProjectMadePrivateMidBuild(t *testing.T) {
	c := newCalPriv(t)
	target := c.project("Fictional Garden Planner", 0)
	c.project("Fictional Bakery Site", 0)
	c.svc.calendarAfterReads = func() {
		c.svc.calendarAfterReads = nil
		c.flags(target, true, false)
		if _, err := c.svc.ImportRoadmapMarkdown("# Fictional Campaign\n\n## Project: Fictional Hidden Atlas\nPrivate: yes\n- [ ] Map it due 2026-07-01\n"); err != nil {
			t.Errorf("import: %v", err)
		}
	}
	for feed, ics := range c.feeds() {
		for _, leak := range []string{"Fictional Garden Planner", "Fictional Hidden Atlas", "Map it"} {
			if strings.Contains(ics, leak) {
				t.Errorf("%s feed leaks %q", feed, leak)
			}
		}
	}
}

// A private roadmap hides its projects' target, item and milestone events;
// tasks and course blocks stay in the feeds.
func TestFeedsOmitProjectsOfPrivateRoadmap(t *testing.T) {
	c := newCalPriv(t)
	c.project("Fictional Bakery Site", 0)
	task := models.Task{Title: "Fictional Task Visible", Status: "pending", DueDate: "2026-07-10", RepeatType: "none", StarsReward: 1}
	if err := c.svc.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	course := models.Course{Name: "Fictional Course", Status: "active"}
	if err := c.svc.Repo.CreateCourse(&course); err != nil {
		t.Fatal(err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Fictional Block Visible", BlockType: "assignment", DueDate: "2026-07-05"}
	if err := c.svc.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatal(err)
	}
	c.assertShown("Fictional Bakery Site")

	if _, err := c.db.Exec(`UPDATE roadmaps SET privacy = 'private'`); err != nil {
		t.Fatal(err)
	}
	c.assertHidden("Fictional Bakery Site")
	feeds := c.feeds()
	for _, want := range []string{"Fictional Task Visible", "Fictional Block Visible"} {
		if !strings.Contains(feeds["all"], want) {
			t.Errorf("all feed should keep %q", want)
		}
	}
}
