package services

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/repository"
)

var autoPinned = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

var autoWF = []AutomationWorkflow{{ID: "wf_backup", Name: "Nightly backup"}, {ID: "wf_digest", Name: "Daily digest"}}

func autoSvc(t *testing.T) (*Service, *time.Time) {
	t.Helper()
	svc := testService(t)
	now := autoPinned
	svc.Now = func() time.Time { return now }
	return svc, &now
}

// record stores an event that occurred at the given offset from the pinned
// time, moving the clock to that moment first so the +/-5 minute window holds.
func record(t *testing.T, svc *Service, now *time.Time, id, wf, status string, at time.Time, summary string) {
	t.Helper()
	*now = at
	res, ignored, err := svc.RecordAutomationEvent(AutomationEventInput{EventID: id, WorkflowID: wf, Status: status, OccurredAt: at.Format(time.RFC3339Nano), ErrorSummary: summary}, autoWF)
	if err != nil || ignored || res != AutomationApplied {
		t.Fatalf("record %s: %q ignored=%v err=%v", id, res, ignored, err)
	}
}

func statusOf(t *testing.T, svc *Service, id string) AutomationStatus {
	t.Helper()
	all, err := svc.AutomationStatuses(autoWF)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range all {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no status for %s", id)
	return AutomationStatus{}
}

func TestParseWorkflows(t *testing.T) {
	got, err := ParseWorkflows(" wf_a:Alpha job , wf-b:Beta ")
	if err != nil || len(got) != 2 || got[0] != (AutomationWorkflow{ID: "wf_a", Name: "Alpha job"}) || got[1].ID != "wf-b" {
		t.Fatalf("got %+v err %v", got, err)
	}
	for _, bad := range []string{"", "  ", "wf_a", ":Name", "wf_a:", "wf a:Name", "wf_a:A,wf_a:B", "wf_a:A,,wf_b:B", "wf_a:A,", "wf_a:Na:me", "wf_a:bad\x00name", "wf_a:" + strings.Repeat("n", 81), strings.Repeat("i", 65) + ":Name", "wf/a:Name"} {
		if _, err := ParseWorkflows(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestRecordAutomationEventWindowBothDirections(t *testing.T) {
	svc, _ := autoSvc(t)
	for _, test := range []struct {
		name   string
		offset time.Duration
		ok     bool
	}{
		{"now", 0, true}, {"past edge", -5 * time.Minute, true}, {"past over", -5*time.Minute - time.Second, false},
		{"future edge", 5 * time.Minute, true}, {"future over", 5*time.Minute + time.Second, false},
	} {
		in := AutomationEventInput{EventID: "e-" + strings.ReplaceAll(test.name, " ", "-"), WorkflowID: "wf_backup", Status: "started", OccurredAt: autoPinned.Add(test.offset).Format(time.RFC3339)}
		_, _, err := svc.RecordAutomationEvent(in, autoWF)
		if (err == nil) != test.ok {
			t.Errorf("%s: err=%v want ok=%v", test.name, err, test.ok)
		}
		if err != nil {
			if _, is := err.(*AutomationValidationError); !is {
				t.Errorf("%s: want validation error, got %T", test.name, err)
			}
		}
	}
}

func TestRecordAutomationEventNormalisesAndUsesServerReceiveTime(t *testing.T) {
	svc, _ := autoSvc(t)
	offset := time.FixedZone("x", -5*3600)
	at := autoPinned.Add(-time.Minute).In(offset).Add(250 * time.Millisecond)
	if _, _, err := svc.RecordAutomationEvent(AutomationEventInput{EventID: "e1", WorkflowID: "wf_backup", Status: "failed", OccurredAt: at.Format(time.RFC3339Nano), ErrorSummary: "  a\tb\nc\x00d  "}, autoWF); err != nil {
		t.Fatal(err)
	}
	rows, _ := svc.Repo.AutomationEventsForWorkflows([]string{"wf_backup"})
	if len(rows) != 1 || rows[0].OccurredAt != "2026-10-11T11:59:00.250000000Z" || rows[0].ReceivedAt != "2026-10-11T12:00:00.000000000Z" || rows[0].ErrorSummary != "abcd" {
		t.Fatalf("%+v", rows)
	}
}

func TestRecordAutomationEventDuplicateAndUnregistered(t *testing.T) {
	svc, _ := autoSvc(t)
	in := AutomationEventInput{EventID: "e1", WorkflowID: "wf_backup", Status: "started", OccurredAt: autoPinned.Format(time.RFC3339)}
	if res, _, _ := svc.RecordAutomationEvent(in, autoWF); res != AutomationApplied {
		t.Fatal(res)
	}
	if res, _, _ := svc.RecordAutomationEvent(in, autoWF); res != AutomationDuplicate {
		t.Fatal(res)
	}
	in.EventID, in.WorkflowID = "e2", "wf_unknown"
	if res, ignored, err := svc.RecordAutomationEvent(in, autoWF); !ignored || res != "" || err != nil {
		t.Fatalf("%q %v %v", res, ignored, err)
	}
	rows, _ := svc.Repo.AutomationEventsForWorkflows([]string{"wf_backup", "wf_unknown"})
	if len(rows) != 1 {
		t.Fatalf("rows %d", len(rows))
	}
}

func TestAutomationSummaryIsCappedAt200Runes(t *testing.T) {
	if got := automationSummary(strings.Repeat("é", 250)); len([]rune(got)) != 200 {
		t.Fatalf("%d runes", len([]rune(got)))
	}
	if got := automationSummary("\u0007\u001b[31mred"); strings.ContainsAny(got, "\u0007\u001b") {
		t.Fatalf("control characters kept: %q", got)
	}
}

func TestAutomationStatesAndOutOfOrderArrival(t *testing.T) {
	svc, now := autoSvc(t)
	if got := statusOf(t, svc, "wf_backup"); got.State != AutomationStateNoEvents || got.LastRunAt != "" {
		t.Fatalf("empty: %+v", got)
	}
	t0 := autoPinned
	// Arrival order is the reverse of occurrence order: the failure arrives
	// first but happened last, so it decides.
	record(t, svc, now, "e3", "wf_backup", "failed", t0.Add(2*time.Minute), "boom")
	*now = t0.Add(2 * time.Minute)
	record(t, svc, now, "e1", "wf_backup", "started", t0.Add(0), "")
	*now = t0.Add(2 * time.Minute)
	record(t, svc, now, "e2", "wf_backup", "succeeded", t0.Add(1*time.Minute), "")
	*now = t0.Add(3 * time.Minute)
	got := statusOf(t, svc, "wf_backup")
	if got.State != AutomationStateFailed || got.ConsecutiveFailures != 1 || got.Severity != AutomationSeverityWarning || got.LastError != "boom" {
		t.Fatalf("%+v", got)
	}
	if got.LastRunAt != "2026-10-11T12:02:00Z" || got.LastSuccessAt != "2026-10-11T12:01:00Z" {
		t.Fatalf("times %+v", got)
	}
	// A later success clears the failure; a recent start then shows Running.
	record(t, svc, now, "e4", "wf_backup", "succeeded", t0.Add(4*time.Minute), "")
	if got := statusOf(t, svc, "wf_backup"); got.State != AutomationStateHealthy || got.ConsecutiveFailures != 0 || got.Severity != "" || got.LastError != "" {
		t.Fatalf("healthy %+v", got)
	}
	record(t, svc, now, "e5", "wf_backup", "started", t0.Add(5*time.Minute), "")
	if got := statusOf(t, svc, "wf_backup"); got.State != AutomationStateRunning {
		t.Fatalf("running %+v", got)
	}
	// Exactly 2 h after the start it is still Running; just past it is not.
	*now = t0.Add(5*time.Minute + 2*time.Hour)
	if got := statusOf(t, svc, "wf_backup"); got.State != AutomationStateRunning {
		t.Fatalf("2h edge %+v", got)
	}
	*now = t0.Add(5*time.Minute + 2*time.Hour + time.Second)
	if got := statusOf(t, svc, "wf_backup"); got.State != AutomationStateHealthy {
		t.Fatalf("stale start falls back to the last finish: %+v", got)
	}
}

func TestAutomationSameInstantFinishBeatsStart(t *testing.T) {
	svc, now := autoSvc(t)
	record(t, svc, now, "e1", "wf_backup", "succeeded", autoPinned, "")
	record(t, svc, now, "e2", "wf_backup", "started", autoPinned, "")
	if got := statusOf(t, svc, "wf_backup"); got.State != AutomationStateHealthy {
		t.Fatalf("%+v", got)
	}
}

func TestAutomationStaleStartAloneIsNoEvents(t *testing.T) {
	svc, now := autoSvc(t)
	record(t, svc, now, "e1", "wf_backup", "started", autoPinned, "")
	*now = autoPinned.Add(3 * time.Hour)
	if got := statusOf(t, svc, "wf_backup"); got.State != AutomationStateNoEvents || got.LastRunAt == "" {
		t.Fatalf("%+v", got)
	}
}

func TestAutomationConsecutiveFailureSeverity(t *testing.T) {
	svc, now := autoSvc(t)
	record(t, svc, now, "s0", "wf_backup", "succeeded", autoPinned, "")
	for i := 1; i <= 4; i++ {
		record(t, svc, now, fmt.Sprintf("f%d", i), "wf_backup", "failed", autoPinned.Add(time.Duration(i)*time.Minute), fmt.Sprintf("error %d", i))
		// A start between failures does not break the streak.
		record(t, svc, now, fmt.Sprintf("st%d", i), "wf_backup", "started", autoPinned.Add(time.Duration(i)*time.Minute+30*time.Second), "")
		got := statusOf(t, svc, "wf_backup")
		want := map[int]string{1: AutomationSeverityWarning, 2: AutomationSeverityWarning, 3: AutomationSeverityCritical, 4: AutomationSeverityCritical}[i]
		if got.ConsecutiveFailures != i || got.Severity != want {
			t.Fatalf("after %d failures: %+v want %q", i, got, want)
		}
		if got.State != AutomationStateRunning || got.LastError != fmt.Sprintf("error %d", i) {
			t.Fatalf("after %d failures: %+v", i, got)
		}
	}
}

func TestAutomationFailedWorkflowsSortFirst(t *testing.T) {
	svc, now := autoSvc(t)
	record(t, svc, now, "e1", "wf_backup", "succeeded", autoPinned, "")
	record(t, svc, now, "e2", "wf_digest", "failed", autoPinned, "x")
	all, err := svc.AutomationStatuses(autoWF)
	if err != nil || len(all) != 2 || all[0].ID != "wf_digest" || all[1].ID != "wf_backup" {
		t.Fatalf("%+v %v", all, err)
	}
}

func TestAutomationPruneAt30Days(t *testing.T) {
	svc, now := autoSvc(t)
	old := autoPinned
	record(t, svc, now, "old", "wf_backup", "succeeded", old, "")
	// 30 days later (minus a second): still kept.
	record(t, svc, now, "keep", "wf_backup", "succeeded", old.Add(30*24*time.Hour-time.Second), "")
	rows, _ := svc.Repo.AutomationEventsForWorkflows([]string{"wf_backup"})
	if len(rows) != 2 {
		t.Fatalf("rows %d", len(rows))
	}
	// One second more and the first row is older than 30 days.
	record(t, svc, now, "new", "wf_backup", "succeeded", old.Add(30*24*time.Hour+time.Second), "")
	rows, _ = svc.Repo.AutomationEventsForWorkflows([]string{"wf_backup"})
	if len(rows) != 2 {
		t.Fatalf("rows after prune %d", len(rows))
	}
	for _, r := range rows {
		if r.EventID == "old" {
			t.Fatal("old row survived")
		}
	}
}

// GYST-131 (Codex P2): sub-second order must survive storage, or a finish at
// .100 followed by a start at .900 in the same second would read as Healthy.
func TestAutomationSubSecondOrderKeepsRunning(t *testing.T) {
	svc, now := autoSvc(t)
	record(t, svc, now, "e1", "wf_backup", "succeeded", autoPinned.Add(100*time.Millisecond), "")
	record(t, svc, now, "e2", "wf_backup", "started", autoPinned.Add(900*time.Millisecond), "")
	if got := statusOf(t, svc, "wf_backup"); got.State != AutomationStateRunning {
		t.Fatalf("want Running: %+v", got)
	}
}

// GYST-131 (Codex P2): the insert has committed, so a failing prune must not
// turn the response into an error (n8n would retry a stored event).
func TestAutomationFailingPruneStillReportsApplied(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`CREATE TRIGGER fail_prune BEFORE DELETE ON automation_events
		BEGIN SELECT RAISE(ABORT, 'prune blocked'); END`); err != nil {
		t.Fatal(err)
	}
	svc := New(repository.New(database))
	now := autoPinned
	svc.Now = func() time.Time { return now }
	// An old row makes the prune actually try to delete something.
	if _, err := database.Exec(`INSERT INTO automation_events (event_id, workflow_id, status, occurred_at, received_at)
		VALUES ('old', 'wf_backup', 'succeeded', '2020-01-01T00:00:00.000000000Z', '2020-01-01T00:00:00.000000000Z')`); err != nil {
		t.Fatal(err)
	}
	res, ignored, err := svc.RecordAutomationEvent(AutomationEventInput{EventID: "e1", WorkflowID: "wf_backup", Status: "started", OccurredAt: now.Format(time.RFC3339)}, autoWF)
	if err != nil || ignored || res != AutomationApplied {
		t.Fatalf("%q ignored=%v err=%v", res, ignored, err)
	}
	var n int
	if err := database.QueryRow(`SELECT COUNT(*) FROM automation_events WHERE event_id = 'e1'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("event not stored: %d %v", n, err)
	}
}
