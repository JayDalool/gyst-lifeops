package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

func attentionService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return New(repository.New(database)), database
}

func produce(t *testing.T, svc *Service, p AttentionProducer) []AttentionItem {
	t.Helper()
	items, err := p.Produce(context.Background(), svc.Now())
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func TestAttentionOverdueTasks(t *testing.T) {
	svc, _ := attentionService(t)
	pinUTC(t, svc, "2026-11-05T18:00:00Z") // Thu 2026-11-05 12:00 CST
	for _, task := range []models.Task{
		{Title: "Paint fence", DueDate: "2026-11-04"},
		{Title: "Today task", DueDate: "2026-11-05"},
		{Title: "Tomorrow task", DueDate: "2026-11-06"},
		{Title: "Weekly chore", DueDate: "2026-11-01", RepeatType: "weekly"},
		{Title: "Before DST end", DueDate: "2026-10-31"},
		{Title: "On DST end", DueDate: "2026-11-01"},
	} {
		if err := svc.Repo.CreateTask(&task); err != nil {
			t.Fatal(err)
		}
	}
	archived := models.Task{Title: "Archived old", DueDate: "2026-10-01"}
	if err := svc.Repo.CreateTask(&archived); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.ArchiveTask(archived.ID); err != nil {
		t.Fatal(err)
	}

	items := produce(t, svc, taskAttention{svc})
	byTitle := map[string]AttentionItem{}
	for _, i := range items {
		byTitle[i.Title] = i
	}
	if len(items) != 3 {
		t.Fatalf("items = %#v", items)
	}
	fence := byTitle["Paint fence"]
	if fence.Severity != AttentionWarning || fence.Kind != AttentionProblem ||
		fence.Context != "Overdue since 2026-11-04" || !strings.HasPrefix(fence.Target, "/tasks/") {
		t.Fatalf("fence = %#v", fence)
	}
	// Local midnight after the due date: CST (UTC-6) after the DST end.
	if want := time.Date(2026, 11, 5, 6, 0, 0, 0, time.UTC); !fence.FirstSeen.Equal(want) {
		t.Fatalf("FirstSeen = %v, want %v", fence.FirstSeen, want)
	}
	// 2026-10-31 -> 2026-11-01 00:00 CDT (UTC-5); 2026-11-01 -> 2026-11-02 00:00 CST (UTC-6).
	if want := time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC); !byTitle["Before DST end"].FirstSeen.Equal(want) {
		t.Fatalf("before DST = %v", byTitle["Before DST end"].FirstSeen)
	}
	if want := time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC); !byTitle["On DST end"].FirstSeen.Equal(want) {
		t.Fatalf("on DST = %v", byTitle["On DST end"].FirstSeen)
	}
}

func TestAttentionTaskFingerprintIgnoresTitle(t *testing.T) {
	svc, database := attentionService(t)
	pinUTC(t, svc, "2026-11-05T18:00:00Z")
	task := models.Task{Title: "Old name", DueDate: "2026-11-01"}
	if err := svc.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	before := produce(t, svc, taskAttention{svc})[0]
	if _, err := database.Exec(`UPDATE tasks SET title = 'New name' WHERE id = ?`, task.ID); err != nil {
		t.Fatal(err)
	}
	renamed := produce(t, svc, taskAttention{svc})[0]
	if renamed.Title != "New name" || renamed.Fingerprint != before.Fingerprint {
		t.Fatalf("rename changed fingerprint: %q vs %q", renamed.Fingerprint, before.Fingerprint)
	}
	if _, err := database.Exec(`UPDATE tasks SET due_date = '2026-11-02' WHERE id = ?`, task.ID); err != nil {
		t.Fatal(err)
	}
	if moved := produce(t, svc, taskAttention{svc})[0]; moved.Fingerprint == before.Fingerprint {
		t.Fatal("moving the due date must change the fingerprint")
	}
}

func TestAttentionOverdueRemindersExcludeTaskLinked(t *testing.T) {
	svc, database := attentionService(t)
	pinUTC(t, svc, "2026-11-05T18:00:00Z")
	for _, r := range []struct{ title, date string }{
		{"Water plants", "2026-11-04"},
		{"Today reminder", "2026-11-05"},
		{"Tomorrow reminder", "2026-11-06"},
	} {
		if _, err := database.Exec(`INSERT INTO reminders (source_type, title, reminder_date, reminder_time) VALUES ('reminder', ?, ?, '09:00')`, r.title, r.date); err != nil {
			t.Fatal(err)
		}
	}
	// Task-linked reminder for an overdue task: only the task appears.
	task := models.Task{Title: "Overdue linked", DueDate: "2026-11-03", ReminderTime: "09:00"}
	if err := svc.Repo.CreateTask(&task); err != nil {
		t.Fatal(err)
	}
	if err := svc.Repo.UpdateTask(&task); err != nil {
		t.Fatal(err)
	}
	var linked int
	if err := database.QueryRow(`SELECT COUNT(*) FROM reminders WHERE source_type = 'task'`).Scan(&linked); err != nil || linked != 1 {
		t.Fatalf("linked reminders = %d, %v", linked, err)
	}
	items := produce(t, svc, reminderAttention{svc})
	if len(items) != 1 || items[0].Title != "Water plants" || items[0].Target != "/reminders" ||
		items[0].Severity != AttentionWarning {
		t.Fatalf("items = %#v", items)
	}
	list := ListAttention(context.Background(), svc.Now(), svc.AttentionProducers...)
	count := 0
	for _, i := range list.Items {
		if i.Title == "Overdue linked" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("overdue task appears %d times in %v", count, keysOf(list))
	}
}

func TestAttentionLearning(t *testing.T) {
	svc, _ := attentionService(t)
	pinUTC(t, svc, "2026-11-05T18:00:00Z")
	course := models.Course{Name: "Fictional Course"}
	if err := svc.Repo.CreateCourse(&course); err != nil {
		t.Fatal(err)
	}
	for _, b := range []models.LearningBlock{
		{Title: "Due today", DueDate: "2026-11-05"},
		{Title: "Due yesterday", DueDate: "2026-11-04"},
		{Title: "Catch up lesson", DueDate: "2026-11-03", IsCatchup: true},
		{Title: "Tomorrow", DueDate: "2026-11-06"},
	} {
		b.CourseID = course.ID
		if err := svc.Repo.CreateLearningBlock(&b); err != nil {
			t.Fatal(err)
		}
	}
	items := produce(t, svc, learningAttention{svc})
	got := map[string]AttentionItem{}
	for _, i := range items {
		got[i.Title] = i
	}
	if len(items) != 3 {
		t.Fatalf("items = %#v", items)
	}
	if got["Due today"].Severity != AttentionInfo || got["Due yesterday"].Severity != AttentionWarning {
		t.Fatalf("severities = %#v", got)
	}
	if !strings.Contains(got["Due yesterday"].Target, "#block-") || !strings.HasPrefix(got["Due yesterday"].Target, "/courses/") {
		t.Fatalf("target = %q", got["Due yesterday"].Target)
	}
	if !strings.Contains(got["Catch up lesson"].Target, "#catchup-") {
		t.Fatalf("target = %q", got["Catch up lesson"].Target)
	}
}

func TestAttentionHealthSync(t *testing.T) {
	svc, _ := attentionService(t)
	pinUTC(t, svc, "2026-11-05T18:00:00Z")
	if got := produce(t, svc, healthSyncAttention{svc}); len(got) != 0 {
		t.Fatalf("no row should give none: %#v", got)
	}
	// Fictional values that must never appear in the output.
	calories, steps := 1234.5, int64(7777)
	if err := svc.Repo.UpsertDailyHealthSnapshot(models.DailyHealthSnapshot{
		Date:          "2026-11-05",
		HealthMetrics: models.HealthMetrics{DietaryCalories: &calories, StepsWatch: &steps},
		LastSyncedAt:  "2026-11-05T10:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, synced string
		want         int
	}{
		{"7h59m", "2026-11-05T10:01:00Z", 0},
		{"8h", "2026-11-05T10:00:00Z", 1},
		{"older", "2026-11-04T10:00:00Z", 1},
	} {
		if err := svc.Repo.UpsertHealthSyncState(models.HealthSyncState{Source: "apple_shortcuts", LastSyncedAt: c.synced}); err != nil {
			t.Fatal(err)
		}
		items := produce(t, svc, healthSyncAttention{svc})
		if len(items) != c.want {
			t.Fatalf("%s: items = %#v", c.name, items)
		}
		for _, i := range items {
			all := strings.Join([]string{i.Title, i.Context, i.Target, i.SourceKey}, " ")
			for _, secret := range []string{"1234", "7777"} {
				if strings.Contains(all, secret) {
					t.Fatalf("health value leaked: %q", all)
				}
			}
			if i.Severity != AttentionWarning || i.Target != "/health" || i.Title != "Apple Health not synced" {
				t.Fatalf("item = %#v", i)
			}
		}
	}
}

func TestHealthSyncStaleAfterIsEightHours(t *testing.T) {
	if HealthSyncStaleAfter != 8*time.Hour {
		t.Fatalf("stale rule = %v", HealthSyncStaleAfter)
	}
}

func writeBackup(t *testing.T, svc *Service, body string) {
	t.Helper()
	svc.BackupStatusPath = filepath.Join(t.TempDir(), "last_run.json")
	if err := os.WriteFile(svc.BackupStatusPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func backupJSON(finished, result string, verified bool) string {
	v := "false"
	if verified {
		v = "true"
	}
	return `{"finished_at":"` + finished + `","result":"` + result + `","verified":` + v + `,"location":"/secret/place"}`
}

func TestAttentionBackup(t *testing.T) {
	svc, _ := attentionService(t)
	pinUTC(t, svc, "2026-11-05T18:00:00Z")
	now := svc.Now()
	at := func(age time.Duration) string { return now.Add(-age).UTC().Format(time.RFC3339Nano) }

	cases := []struct {
		name string
		body string
		want AttentionSeverity // "" means no item
	}{
		{"fresh", backupJSON(at(35*time.Hour+59*time.Minute), "ok", true), ""},
		{"36h", backupJSON(at(36*time.Hour), "ok", true), AttentionWarning},
		{"71h59m", backupJSON(at(71*time.Hour+59*time.Minute), "ok", true), AttentionWarning},
		{"72h", backupJSON(at(72*time.Hour), "ok", true), AttentionCritical},
		{"failed", backupJSON(at(time.Hour), "failed", true), AttentionWarning},
		{"unverified", backupJSON(at(time.Hour), "ok", false), AttentionWarning},
	}
	for _, c := range cases {
		writeBackup(t, svc, c.body)
		items, err := backupAttention{svc}.Produce(context.Background(), now)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if c.want == "" {
			if len(items) != 0 {
				t.Fatalf("%s: items = %#v", c.name, items)
			}
			continue
		}
		if len(items) != 1 || items[0].Severity != c.want {
			t.Fatalf("%s: items = %#v", c.name, items)
		}
		if strings.Contains(items[0].Context+items[0].Title+items[0].Target, "secret") {
			t.Fatalf("%s: location leaked", c.name)
		}
	}
}

func TestAttentionBackupUnavailable(t *testing.T) {
	svc, _ := attentionService(t)
	pinUTC(t, svc, "2026-11-05T18:00:00Z")
	future := svc.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	bodies := map[string]string{
		"invalid json": "{nope",
		"too large":    `{"result":"` + strings.Repeat("x", 70<<10) + `"}`,
		"future":       backupJSON(future, "ok", true),
		"bad time":     backupJSON("yesterday", "ok", true),
	}
	for name, body := range bodies {
		writeBackup(t, svc, body)
		list := svc.Attention(context.Background())
		if !contains(list.Unavailable, "GYST backup") {
			t.Fatalf("%s: unavailable = %v", name, list.Unavailable)
		}
	}
	svc.BackupStatusPath = filepath.Join(t.TempDir(), "missing.json")
	list := svc.Attention(context.Background())
	if !contains(list.Unavailable, "GYST backup") || list.BadgeCount() != 0 || len(list.Surface()) != 0 {
		t.Fatalf("missing file: %#v", list)
	}
	svc.BackupStatusPath = ""
	if list := svc.Attention(context.Background()); len(list.Unavailable) != 0 {
		t.Fatalf("unconfigured: %v", list.Unavailable)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
