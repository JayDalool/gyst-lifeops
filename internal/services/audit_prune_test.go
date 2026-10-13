package services

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/repository"
)

// Retention tests (GYST-VN-084). Rows are fictional; clocks are pinned.

func pruneService(t *testing.T, now time.Time) *Service {
	t.Helper()
	svc := testService(t)
	svc.Now = func() time.Time { return now }
	return svc
}

func withLifecycle(t *testing.T, actions ...string) {
	t.Helper()
	previous := lifecycleActions
	lifecycleActions = map[string]bool{}
	for _, a := range actions {
		lifecycleActions[a] = true
	}
	t.Cleanup(func() { lifecycleActions = previous })
}

// seedAuditAt inserts one row straight through the repository so the test
// controls `at`.
func seedAuditAt(t *testing.T, svc *Service, at time.Time, action, ref string) {
	t.Helper()
	err := svc.Repo.InsertAuditEvent(context.Background(), repository.AuditEvent{
		At: clock.FormatInstant(at), ActorType: AuditActorHuman, ActorRef: ref,
		Ingress: AuditIngressPublic, Action: action, Outcome: AuditOutcomeDenied,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func remainingRefs(t *testing.T, svc *Service) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, r := range auditRows(t, svc) {
		out[r.ActorRef] = true
	}
	return out
}

var prunePinned = time.Date(2026, 10, 11, 4, 0, 0, 0, time.UTC)

func TestPruneSecurityRetentionBoundaries(t *testing.T) {
	svc := pruneService(t, prunePinned)
	cutoff := prunePinned.Add(-securityRetention)
	seedAuditAt(t, svc, cutoff.Add(-24*time.Hour), AuditActionLoginFailed, "old")
	seedAuditAt(t, svc, cutoff.Add(-time.Nanosecond), AuditActionLoginFailed, "justold")
	seedAuditAt(t, svc, cutoff, AuditActionLoginFailed, "exact")
	seedAuditAt(t, svc, cutoff.Add(24*time.Hour), AuditActionLoginFailed, "364d")
	seedAuditAt(t, svc, prunePinned, AuditActionLoginFailed, "now")

	sec, life, err := svc.PruneAuditEvents(context.Background())
	if err != nil || sec != 2 || life != 0 {
		t.Fatalf("got sec=%d life=%d err=%v, want 2 0 nil", sec, life, err)
	}
	got := remainingRefs(t, svc)
	for _, want := range []string{"exact", "364d", "now"} {
		if !got[want] {
			t.Errorf("row %q was deleted", want)
		}
	}
	if got["old"] || got["justold"] {
		t.Errorf("expired rows survived: %v", got)
	}
}

func TestPruneLifecycleUsesShorterWindowSecurityUnaffected(t *testing.T) {
	withLifecycle(t, "agent.test_action")
	svc := pruneService(t, prunePinned)
	lc := prunePinned.Add(-lifecycleRetention)
	seedAuditAt(t, svc, lc.Add(-time.Hour), "agent.test_action", "life-old")
	seedAuditAt(t, svc, lc, "agent.test_action", "life-exact")
	seedAuditAt(t, svc, lc.Add(time.Hour), "agent.test_action", "life-new")
	// A security row at 200 days is past the lifecycle window but must stay.
	seedAuditAt(t, svc, prunePinned.Add(-200*24*time.Hour), AuditActionLoginFailed, "sec-200d")
	seedAuditAt(t, svc, prunePinned.Add(-400*24*time.Hour), AuditActionLoginFailed, "sec-400d")
	// A lifecycle row past 365 days goes through the lifecycle pass.
	seedAuditAt(t, svc, prunePinned.Add(-400*24*time.Hour), "agent.test_action", "life-400d")

	sec, life, err := svc.PruneAuditEvents(context.Background())
	if err != nil || sec != 1 || life != 2 {
		t.Fatalf("got sec=%d life=%d err=%v, want 1 2 nil", sec, life, err)
	}
	got := remainingRefs(t, svc)
	for _, want := range []string{"life-exact", "life-new", "sec-200d"} {
		if !got[want] {
			t.Errorf("row %q was deleted", want)
		}
	}
	if len(got) != 3 {
		t.Errorf("remaining = %v, want 3 rows", got)
	}
}

func TestPruneEmptyLifecycleListSkipsItsDelete(t *testing.T) {
	withLifecycle(t) // empty
	svc := pruneService(t, prunePinned)
	seedAuditAt(t, svc, prunePinned.Add(-200*24*time.Hour), "agent.test_action", "unlisted-200d")
	seedAuditAt(t, svc, prunePinned.Add(-400*24*time.Hour), AuditActionLogout, "sec-400d")
	sec, life, err := svc.PruneAuditEvents(context.Background())
	if err != nil || sec != 1 || life != 0 {
		t.Fatalf("got sec=%d life=%d err=%v, want 1 0 nil", sec, life, err)
	}
	if got := remainingRefs(t, svc); !got["unlisted-200d"] || len(got) != 1 {
		t.Errorf("remaining = %v", got)
	}
}

func TestPruneCutoffsAcrossBoundariesAndDST(t *testing.T) {
	toronto, err := time.LoadLocation("America/Toronto")
	if err != nil {
		t.Skip("no tzdata")
	}
	cases := []struct {
		name string
		now  time.Time
	}{
		{"year boundary", time.Date(2027, 1, 1, 0, 0, 0, 5, time.UTC)},
		{"leap day", time.Date(2028, 2, 29, 23, 59, 59, 999999999, time.UTC)},
		{"month end", time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)},
		{"DST spring-forward local", time.Date(2027, 3, 14, 3, 30, 0, 0, toronto)},
		{"DST fall-back local", time.Date(2026, 11, 1, 1, 30, 0, 0, toronto)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := pruneService(t, c.now)
			cutoff := c.now.UTC().Add(-securityRetention)
			seedAuditAt(t, svc, cutoff.Add(-time.Second), AuditActionLoginFailed, "before")
			seedAuditAt(t, svc, cutoff, AuditActionLoginFailed, "exact")
			seedAuditAt(t, svc, cutoff.Add(time.Second), AuditActionLoginFailed, "after")
			sec, _, err := svc.PruneAuditEvents(context.Background())
			if err != nil || sec != 1 {
				t.Fatalf("sec=%d err=%v, want 1", sec, err)
			}
			got := remainingRefs(t, svc)
			if got["before"] || !got["exact"] || !got["after"] {
				t.Errorf("remaining = %v", got)
			}
		})
	}
}

func TestDeleteAuditEventsBeforeRefusesEmptyCutoff(t *testing.T) {
	svc := pruneService(t, prunePinned)
	seedAuditAt(t, svc, prunePinned.Add(-time.Hour), AuditActionLoginFailed, "keep")
	for _, include := range []bool{true, false} {
		if _, err := svc.Repo.DeleteAuditEventsBefore(context.Background(), "", []string{"x"}, include); err == nil {
			t.Errorf("include=%v: empty cutoff accepted", include)
		}
	}
	if _, err := svc.Repo.DeleteAuditEventsBefore(context.Background(), "", nil, false); err == nil {
		t.Error("empty cutoff with no actions accepted")
	}
	if got := remainingRefs(t, svc); !got["keep"] {
		t.Error("row deleted by a refused call")
	}
}

func TestDeleteAuditEventsBeforeInjectionShapedActionIsData(t *testing.T) {
	svc := pruneService(t, prunePinned)
	seedAuditAt(t, svc, prunePinned.Add(-time.Hour), AuditActionLoginFailed, "keep")
	n, err := svc.Repo.DeleteAuditEventsBefore(context.Background(), clock.FormatInstant(prunePinned),
		[]string{"x') OR 1=1 --"}, true)
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if got := remainingRefs(t, svc); !got["keep"] {
		t.Error("action list was not bound as data")
	}
}

func TestPruneLogsCountsOnlyNoRowContent(t *testing.T) {
	buf := captureLog(t)
	svc := pruneService(t, prunePinned)
	seedAuditAt(t, svc, prunePinned.Add(-400*24*time.Hour), AuditActionLoginFailed, "client:203.0.113")
	seedAuditAt(t, svc, prunePinned, AuditActionLoginFailed, "keeper")
	svc.pruneAuditAndLog(context.Background())
	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, `"level":"INFO"`) ||
		!strings.Contains(out, `"deleted_security":1`) || !strings.Contains(out, `"deleted_lifecycle":0`) {
		t.Fatalf("log = %q", out)
	}
	if strings.Contains(out, "203.0.113") || strings.Contains(out, AuditActionLoginFailed) {
		t.Errorf("row content reached the log: %q", out)
	}
}

func TestPruneFailureLogsOneErrorLine(t *testing.T) {
	buf := captureLog(t)
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	svc := New(repository.New(database))
	svc.Now = func() time.Time { return prunePinned }
	database.Close() // the DELETE now fails while ctx is still live
	svc.pruneAuditAndLog(context.Background())
	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, `"level":"ERROR"`) || strings.Contains(out, "deleted_security") {
		t.Fatalf("log = %q", out)
	}
}

func TestRunAuditPruneRunsAtStartThenOnTickAndStopsOnCancel(t *testing.T) {
	previous := auditPruneInterval
	auditPruneInterval = 20 * time.Millisecond
	t.Cleanup(func() { auditPruneInterval = previous })

	svc := pruneService(t, prunePinned)
	seedAuditAt(t, svc, prunePinned, AuditActionLoginFailed, "keeper")
	seedAuditAt(t, svc, prunePinned.Add(-400*24*time.Hour), AuditActionLoginFailed, "first")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	svc.PruneAuditAtStart(ctx)
	go func() { svc.RunAuditPruneLoop(ctx); close(done) }()

	waitFor(t, func() bool { return !remainingRefs(t, svc)["first"] }, "startup prune")
	seedAuditAt(t, svc, prunePinned.Add(-401*24*time.Hour), AuditActionLoginFailed, "second")
	waitFor(t, func() bool { return !remainingRefs(t, svc)["second"] }, "tick prune")

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunAuditPruneLoop did not stop on cancel")
	}
}

func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestPruneClockJumpGuardSkipsAndLogsError(t *testing.T) {
	buf := captureLog(t)
	svc := pruneService(t, prunePinned.Add(2*365*24*time.Hour))
	seedAuditAt(t, svc, prunePinned.Add(-time.Hour), AuditActionLoginFailed, "a")
	seedAuditAt(t, svc, prunePinned, AuditActionLoginFailed, "b")
	sec, life, err := svc.PruneAuditEvents(context.Background())
	if !errors.Is(err, ErrAuditPruneSkipped) || sec != 0 || life != 0 {
		t.Fatalf("sec=%d life=%d err=%v", sec, life, err)
	}
	if got := remainingRefs(t, svc); len(got) != 2 {
		t.Fatalf("rows deleted by a skipped run: %v", got)
	}
	svc.pruneAuditAndLog(context.Background())
	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, `"level":"ERROR"`) {
		t.Fatalf("log = %q", out)
	}
}

func TestPruneEmptyTableIsNoOp(t *testing.T) {
	svc := pruneService(t, prunePinned)
	sec, life, err := svc.PruneAuditEvents(context.Background())
	if err != nil || sec != 0 || life != 0 {
		t.Fatalf("sec=%d life=%d err=%v", sec, life, err)
	}
}

// Rows written by RecordAudit under a non-UTC app clock with a sub-second
// fraction compare correctly against the cutoff.
func TestPruneEndToEndWithRecordAuditNonUTCClock(t *testing.T) {
	toronto, err := time.LoadLocation("America/Toronto")
	if err != nil {
		t.Skip("no tzdata")
	}
	svc := testService(t)
	pruneNow := time.Date(2026, 10, 11, 0, 30, 0, 500_000_000, toronto)
	cutoff := pruneNow.Add(-securityRetention)
	write := func(at time.Time, ref string) {
		svc.Now = func() time.Time { return at.In(toronto) }
		e := goodAudit()
		e.ActorRef = ref
		if err := svc.RecordAudit(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	write(cutoff.Add(-time.Nanosecond), "before")
	write(cutoff, "exact")
	write(cutoff.Add(time.Nanosecond), "after")
	write(pruneNow, "now")
	svc.Now = func() time.Time { return pruneNow }
	sec, _, err := svc.PruneAuditEvents(context.Background())
	if err != nil || sec != 1 {
		t.Fatalf("sec=%d err=%v", sec, err)
	}
	got := remainingRefs(t, svc)
	if got["before"] || !got["exact"] || !got["after"] || !got["now"] {
		t.Errorf("remaining = %v", got)
	}
}

// GYST-146: with lifecycle actions in use, the guard checks the shorter
// (lifecycle) cutoff and a jump skips the whole run.
func TestPruneLifecycleClockJumpSkipsWholeRun(t *testing.T) {
	withLifecycle(t, "agent.test_action")
	buf := captureLog(t)
	svc := pruneService(t, prunePinned.Add(200*24*time.Hour))
	seedAuditAt(t, svc, prunePinned.Add(-400*24*time.Hour), AuditActionLoginFailed, "sec-400d")
	seedAuditAt(t, svc, prunePinned.Add(-time.Hour), "agent.test_action", "life-a")
	seedAuditAt(t, svc, prunePinned, AuditActionLoginFailed, "sec-b")
	sec, life, err := svc.PruneAuditEvents(context.Background())
	if !errors.Is(err, ErrAuditPruneSkipped) || sec != 0 || life != 0 {
		t.Fatalf("sec=%d life=%d err=%v", sec, life, err)
	}
	if got := remainingRefs(t, svc); len(got) != 3 {
		t.Fatalf("rows deleted by a skipped run: %v", got)
	}
	svc.pruneAuditAndLog(context.Background())
	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, `"level":"ERROR"`) || strings.Contains(out, "life-a") {
		t.Fatalf("log = %q", out)
	}
}

func TestPruneLifecycleGuardBoundary(t *testing.T) {
	withLifecycle(t, "agent.test_action")
	latest := prunePinned
	for _, tc := range []struct {
		name string
		now  time.Time
		skip bool
	}{
		{"exactly 180d after newest runs", latest.Add(lifecycleRetention), false},
		{"180d plus 1ns skips", latest.Add(lifecycleRetention + time.Nanosecond), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := pruneService(t, tc.now)
			seedAuditAt(t, svc, latest, AuditActionLoginFailed, "newest")
			_, _, err := svc.PruneAuditEvents(context.Background())
			if got := errors.Is(err, ErrAuditPruneSkipped); got != tc.skip || (!tc.skip && err != nil) {
				t.Fatalf("err=%v, want skipped=%v", err, tc.skip)
			}
		})
	}
}

func TestPruneEmptyLifecycleListJumpStillRunsSecurityPass(t *testing.T) {
	withLifecycle(t) // empty
	svc := pruneService(t, prunePinned.Add(200*24*time.Hour))
	seedAuditAt(t, svc, prunePinned.Add(-400*24*time.Hour), AuditActionLoginFailed, "sec-400d")
	seedAuditAt(t, svc, prunePinned, AuditActionLoginFailed, "newest")
	sec, life, err := svc.PruneAuditEvents(context.Background())
	if err != nil || sec != 1 || life != 0 {
		t.Fatalf("got sec=%d life=%d err=%v, want 1 0 nil", sec, life, err)
	}
}
