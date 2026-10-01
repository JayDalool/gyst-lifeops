package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func locationOf(t *testing.T, rec *httptest.ResponseRecorder) *url.URL {
	t.Helper()
	u, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func makeTask(t *testing.T, h *Handler, title string, reward int) models.Task {
	t.Helper()
	task := models.Task{Title: title, DueDate: "2026-06-24", StarsReward: reward}
	if err := h.Service.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	return task
}

func stars(t *testing.T, h *Handler) int {
	t.Helper()
	stats, err := h.Service.Repo.PlayerStats()
	if err != nil {
		t.Fatal(err)
	}
	return stats.TotalStars
}

func TestTaskCompleteRedirectCarriesToastParams(t *testing.T) {
	h := testHandler(t)
	task := makeTask(t, h, "Water the plants", 4)
	rec := postQuestForm(t, h, fmt.Sprintf("/tasks/%d/complete", task.ID), url.Values{
		"occurrence_date": {"2026-06-24"},
		"return_to":       {fmt.Sprintf("/today?view=a#task-%d", task.ID)},
	})
	loc := locationOf(t, rec)
	q := loc.Query()
	if loc.Path != "/today" || loc.Fragment != fmt.Sprintf("task-%d", task.ID) || q.Get("view") != "a" {
		t.Fatalf("return_to not preserved: %s", loc)
	}
	want := map[string]string{
		"celebrate": "task", "celebrate_title": "Water the plants", "celebrate_xp": "4",
		"celebrate_undo": fmt.Sprintf("/tasks/%d/undo", task.ID), "celebrate_date": "2026-06-24",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q (%s)", k, q.Get(k), v, loc)
		}
	}
	if q.Has("celebrate_level") {
		t.Errorf("small reward crossed no level but got celebrate_level: %s", loc)
	}
	if !strings.Contains(rec.Header().Get("Location"), "celebrate_undo=%2Ftasks%2F") {
		t.Errorf("undo not encoded in %s", rec.Header().Get("Location"))
	}
}

func TestRecompletionAfterSkipShowsZeroXP(t *testing.T) {
	h := testHandler(t)
	task := makeTask(t, h, "Flip flop", 4)
	base := fmt.Sprintf("/tasks/%d", task.ID)
	form := url.Values{"occurrence_date": {"2026-06-24"}}
	postQuestForm(t, h, base+"/complete", form)
	postQuestForm(t, h, base+"/skip", form)
	rec := postQuestForm(t, h, base+"/complete", form)
	if got := locationOf(t, rec).Query().Get("celebrate_xp"); got != "0" {
		t.Fatalf("celebrate_xp = %q, want 0 (stars were already awarded)", got)
	}
}

func TestLevelUpParamOnlyWhenLevelCrossed(t *testing.T) {
	h := testHandler(t)
	big := makeTask(t, h, "Big one", 100)
	rec := postQuestForm(t, h, fmt.Sprintf("/tasks/%d/complete", big.ID), url.Values{"occurrence_date": {"2026-06-24"}})
	if got := locationOf(t, rec).Query().Get("celebrate_level"); got != "2" {
		t.Fatalf("celebrate_level = %q, want 2", got)
	}
	small := makeTask(t, h, "Small one", 1)
	rec = postQuestForm(t, h, fmt.Sprintf("/tasks/%d/complete", small.ID), url.Values{"occurrence_date": {"2026-06-24"}})
	if locationOf(t, rec).Query().Has("celebrate_level") {
		t.Fatal("celebrate_level set although the level did not change")
	}
}

func TestLongTitleIsTruncatedInRedirect(t *testing.T) {
	h := testHandler(t)
	task := makeTask(t, h, strings.Repeat("é", 120), 4)
	rec := postQuestForm(t, h, fmt.Sprintf("/tasks/%d/complete", task.ID), url.Values{"occurrence_date": {"2026-06-24"}})
	if got := []rune(locationOf(t, rec).Query().Get("celebrate_title")); len(got) != 80 {
		t.Fatalf("title length = %d runes, want 80", len(got))
	}
}

func TestMissionToastHasUndoUnlessProjectFinished(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Two step quest", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	first := models.ProjectItem{ProjectID: project.ID, Title: "Step one"}
	second := models.ProjectItem{ProjectID: project.ID, Title: "Step two"}
	for _, item := range []*models.ProjectItem{&first, &second} {
		if err := h.Service.Repo.CreateProjectItem(item); err != nil {
			t.Fatal(err)
		}
	}
	rec := postQuestForm(t, h, fmt.Sprintf("/projects/%d/items/%d/complete", project.ID, first.ID), nil)
	q := locationOf(t, rec).Query()
	wantUndo := fmt.Sprintf("/projects/%d/items/%d/undo", project.ID, first.ID)
	if q.Get("celebrate") != "task" || q.Get("celebrate_undo") != wantUndo || q.Get("celebrate_xp") != "20" {
		t.Fatalf("mid-project toast params: %v", q)
	}

	// Undo via the real route puts it back and removes the stars.
	postQuestForm(t, h, wantUndo, nil)
	if got := stars(t, h); got != 0 {
		t.Fatalf("stars after undo = %d, want 0", got)
	}
	postQuestForm(t, h, wantUndo, nil) // idempotent

	// Finish both: the last completion finishes the project and carries no Undo.
	postQuestForm(t, h, fmt.Sprintf("/projects/%d/items/%d/complete", project.ID, first.ID), nil)
	rec = postQuestForm(t, h, fmt.Sprintf("/projects/%d/items/%d/complete", project.ID, second.ID), nil)
	q = locationOf(t, rec).Query()
	if q.Get("celebrate") != "project" || q.Has("celebrate_undo") {
		t.Fatalf("project-finishing toast params: %v", q)
	}
	// The refused undo leaves everything alone.
	before := stars(t, h)
	postQuestForm(t, h, fmt.Sprintf("/projects/%d/items/%d/undo", project.ID, second.ID), nil)
	if after := stars(t, h); after != before {
		t.Fatalf("undo on a finished project changed stars %d -> %d", before, after)
	}
}

func TestRoadmapMissionToastUsesProjectsUndoPath(t *testing.T) {
	h := testHandler(t)
	project := models.Project{Name: "Roadmap quest", Status: "active", ProjectKind: "main"}
	if err := h.Service.Repo.CreateProject(&project); err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"A", "B"} {
		item := models.ProjectItem{ProjectID: project.ID, Title: title}
		if err := h.Service.Repo.CreateProjectItem(&item); err != nil {
			t.Fatal(err)
		}
	}
	items, _ := h.Service.Repo.ProjectItems(project.ID)
	rec := postQuestForm(t, h, fmt.Sprintf("/roadmap/projects/%d/items/%d/complete", project.ID, items[0].ID), nil)
	loc := locationOf(t, rec)
	if loc.Path != "/roadmap" || loc.Query().Get("celebrate_undo") != fmt.Sprintf("/projects/%d/items/%d/undo", project.ID, items[0].ID) {
		t.Fatalf("roadmap toast: %s", loc)
	}
}

func TestStudyBlockToastAndUndo(t *testing.T) {
	h := testHandler(t)
	course := models.Course{Name: "Sample course", Status: "active"}
	if err := h.Service.Repo.CreateCourse(&course); err != nil {
		t.Fatal(err)
	}
	block := models.LearningBlock{CourseID: course.ID, Title: "Lesson one", DueDate: "2026-06-24"}
	if err := h.Service.Repo.CreateLearningBlock(&block); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/courses/%d/blocks/%d", course.ID, block.ID)
	rec := postQuestForm(t, h, base+"/complete", url.Values{"return_to": {"/today"}})
	loc := locationOf(t, rec)
	if loc.Path != "/today" || loc.Query().Get("celebrate_undo") != base+"/undo" || loc.Query().Get("celebrate_xp") != "0" {
		t.Fatalf("study block toast: %s", loc)
	}
	postQuestForm(t, h, base+"/undo", nil)
	if b, _ := h.Service.Repo.LearningBlockByID(block.ID); b.Status != "pending" {
		t.Fatalf("status after undo = %q, want pending", b.Status)
	}
	postQuestForm(t, h, base+"/undo", nil) // idempotent
}

func TestUndoRouteIsPostOnly(t *testing.T) {
	h := testHandler(t)
	task := makeTask(t, h, "Get me", 4)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%d/undo", task.ID), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET undo = %d, want 405", rec.Code)
	}
}

func postUndo(t *testing.T, h *Handler, cookie *http.Cookie, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(url.Values{"occurrence_date": {"2026-06-24"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "http://"+req.Host)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, req)
	return rec
}

func TestSignedInSameOriginUndoWorksAndRepeats(t *testing.T) {
	h := authHandler(t)
	cookie := signIn(t, h)
	task := makeTask(t, h, "Signed in task", 4)
	if err := h.Service.Repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/tasks/%d/undo", task.ID)
	for i := 0; i < 2; i++ {
		if rec := postUndo(t, h, cookie, path, nil); rec.Code != http.StatusSeeOther {
			t.Fatalf("undo #%d = %d, want 303", i+1, rec.Code)
		}
	}
	if got, _ := h.Service.Repo.TaskByID(task.ID); got.Status != "pending" {
		t.Fatalf("status = %q, want pending", got.Status)
	}
	if got := stars(t, h); got != 0 {
		t.Fatalf("stars = %d, want 0", got)
	}
}

// The undo route sits behind the same-origin guard: a cross-site or foreign-origin
// POST is refused and nothing is reverted.
func TestCrossSiteUndoIsRejected(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"sec-fetch-site": {"Sec-Fetch-Site": "cross-site"},
		"foreign origin": {"Origin": "https://untrusted.example"},
	} {
		t.Run(name, func(t *testing.T) {
			h := authHandler(t)
			cookie := signIn(t, h)
			task := makeTask(t, h, "Guarded task", 4)
			if err := h.Service.Repo.SetTaskOccurrenceStatus(task.ID, "2026-06-24", "completed"); err != nil {
				t.Fatal(err)
			}
			rec := postUndo(t, h, cookie, fmt.Sprintf("/tasks/%d/undo", task.ID), headers)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("cross-site undo = %d, want 403", rec.Code)
			}
			if got, _ := h.Service.Repo.TaskByID(task.ID); got.Status != "completed" {
				t.Fatalf("task status = %q, want completed", got.Status)
			}
			if got := stars(t, h); got != 4 {
				t.Fatalf("stars = %d, want 4 unchanged", got)
			}
		})
	}
}

func TestOldCelebrationOverlayIsGone(t *testing.T) {
	read := func(path string) string {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	css := read("../../web/static/css/styles.css")
	for _, gone := range []string{".completion-event", "completion-burst", "completion-suck", "planet-consumed", "black-hole-pulse"} {
		if strings.Contains(css, gone) {
			t.Errorf("styles.css still contains %q", gone)
		}
	}
	for _, want := range []string{".completion-toast", "var(--vn-base) var(--vn-ease)", `:root[data-motion="reduced"] .lifeops-toast.completion-toast { transform: translate(-50%, 0); }`, "@media (prefers-reduced-motion: reduce) {\n  .lifeops-toast.completion-toast { transform: translate(-50%, 0); }"} {
		if !strings.Contains(css, want) {
			t.Errorf("styles.css missing %q", want)
		}
	}
	app := read("../../web/static/js/app.js")
	for _, gone := range []string{"initCelebration", "completion-particles"} {
		if strings.Contains(app, gone) {
			t.Errorf("app.js still contains %q", gone)
		}
	}
	if !strings.Contains(app, "initCheckOff") {
		t.Error("app.js lost initCheckOff")
	}
	if strings.Contains(read("../../web/static/js/completion-toast.js"), "innerHTML") {
		t.Error("completion-toast.js must not use innerHTML")
	}
}
