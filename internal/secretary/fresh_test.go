package secretary_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"lifeops-tracker/internal/secretary"
	"lifeops-tracker/internal/secretary/secretarytest"
)

func freshener(e *secretarytest.Env, budget time.Duration) *secretary.Freshener {
	f := secretary.NewFreshener(map[string]secretary.Syncer{secretary.SourceHevy: secretary.HevySyncer(e.Svc)}, budget, e.Svc.Now)
	e.T.Cleanup(f.Wait)
	return f
}

func hevyErr(t *testing.T, e *secretarytest.Env) string {
	t.Helper()
	c, err := e.Svc.Repo.TrainingCache("hevy")
	if err != nil {
		t.Fatal(err)
	}
	return c.Error
}

func TestWebBudget(t *testing.T) {
	if secretary.WebBudget != 10*time.Second {
		t.Fatalf("WebBudget %v; the web budget is 10 s and Siri has no budget (19A S2)", secretary.WebBudget)
	}
	e := secretarytest.New(t)
	// A zero budget in the constructor means the web budget.
	f := secretary.NewFreshener(nil, 0, e.Svc.Now)
	if got := f.EnsureFresh(context.Background(), secretary.SourceMail); got[secretary.SourceMail].Connected {
		t.Fatal("empty slot reported connected")
	}
}

// The provider is slower than the budget: EnsureFresh gives up waiting, answers
// from the stored copy and marks it stale. The sync is not cancelled, so the
// Hevy record never shows a failure the user did not cause.
func TestEnsureFreshSlowProvider(t *testing.T) {
	for _, honour := range []bool{true, false} {
		name := map[bool]string{true: "provider honours ctx", false: "provider ignores ctx"}[honour]
		t.Run(name, func(t *testing.T) {
			e := secretarytest.New(t)
			stored := e.Svc.Now().Add(-2 * time.Hour)
			e.SeedHevy(stored)
			slow := secretarytest.SlowHevy(honour)
			e.ConnectHevy(slow)
			f := freshener(e, 100*time.Millisecond)
			t.Cleanup(slow.Release)

			start := time.Now()
			got := f.EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
			if took := time.Since(start); took > 3*time.Second { // generous: -race on a busy CI box
				t.Fatalf("EnsureFresh took %v for a 100 ms budget", took)
			}
			if !got.Stale || !got.Connected || !got.AsOf.Equal(stored) {
				t.Fatalf("got %+v, want stale, connected, as_of %v", got, stored)
			}
			if slow.Fetches() != 1 {
				t.Fatalf("Fetch called %d times", slow.Fetches())
			}
			cache, _ := e.Svc.Repo.TrainingCache("hevy")
			if len(cache.Entries) != 2 || cache.Error != "" {
				t.Fatalf("stored rows %d, error %q: the slow sync must not touch the copy", len(cache.Entries), cache.Error)
			}
			slow.Release()
			f.Wait()
			if hevyErr(t, e) != "" {
				t.Fatal("a timeout recorded a Hevy failure")
			}
		})
	}
}

func TestEnsureFreshFastSync(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-2 * time.Hour))
	fake := &secretarytest.FakeHevy{}
	e.ConnectHevy(fake)
	got := freshener(e, time.Second).EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
	if got.Stale || !got.AsOf.Equal(e.Svc.Now()) || fake.Fetches() != 1 {
		t.Fatalf("%+v after %d fetches", got, fake.Fetches())
	}
}

func TestEnsureFreshSkipsFreshSource(t *testing.T) {
	e := secretarytest.New(t)
	stored := e.Svc.Now().Add(-10 * time.Minute)
	e.SeedHevy(stored)
	fake := &secretarytest.FakeHevy{}
	e.ConnectHevy(fake)
	got := freshener(e, time.Second).EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
	if got.Stale || !got.AsOf.Equal(stored) || fake.Fetches() != 0 {
		t.Fatalf("%+v after %d fetches", got, fake.Fetches())
	}
	// Just past the 60 minute limit it syncs.
	e2 := secretarytest.New(t)
	e2.SeedHevy(e2.Svc.Now().Add(-61 * time.Minute))
	fake2 := &secretarytest.FakeHevy{}
	e2.ConnectHevy(fake2)
	freshener(e2, time.Second).EnsureFresh(context.Background(), secretary.SourceHevy)
	if fake2.Fetches() != 1 {
		t.Fatalf("61 minutes old: %d fetches", fake2.Fetches())
	}
}

func TestEnsureFreshOncePerCatalogue(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-2 * time.Hour))
	fake := &secretarytest.FakeHevy{Err: context.DeadlineExceeded}
	e.ConnectHevy(fake)
	f := freshener(e, time.Second)
	for i := 0; i < 3; i++ {
		f.EnsureFresh(context.Background(), secretary.SourceHevy, secretary.SourceHevy)
	}
	if fake.Fetches() != 1 {
		t.Fatalf("%d fetches in one turn", fake.Fetches())
	}
}

func TestEnsureFreshEmptySlots(t *testing.T) {
	e := secretarytest.New(t)
	f := freshener(e, time.Second)
	got := f.EnsureFresh(context.Background(), secretary.SourceCalendar, secretary.SourceMail)
	for _, src := range []string{secretary.SourceCalendar, secretary.SourceMail} {
		if st := got[src]; st.Connected || !st.AsOf.IsZero() || !st.Stale {
			t.Errorf("%s: %+v", src, st)
		}
	}
}

// Busy and throttled: SyncTraining refuses at once, so there is no wait, no
// second Fetch and no failure row.
func TestEnsureFreshBusyAndThrottled(t *testing.T) {
	t.Run("busy", func(t *testing.T) {
		e := secretarytest.New(t)
		stored := e.Svc.Now().Add(-2 * time.Hour)
		e.SeedHevy(stored)
		slow := secretarytest.SlowHevy(false)
		e.ConnectHevy(slow)
		t.Cleanup(slow.Release)
		manual := make(chan error, 1)
		go func() { manual <- e.Svc.SyncTraining(context.Background()) }() // e.g. the Sync now button
		for i := 0; slow.Fetches() == 0 && i < 400; i++ {
			time.Sleep(5 * time.Millisecond)
		}
		if slow.Fetches() != 1 {
			t.Fatal("the manual sync never started")
		}
		f := freshener(e, 5*time.Second)
		start := time.Now()
		got := f.EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
		if time.Since(start) > 2*time.Second {
			t.Fatal("waited for a sync that was already running")
		}
		if !got.Stale || !got.AsOf.Equal(stored) || slow.Fetches() != 1 {
			t.Fatalf("%+v, %d fetches", got, slow.Fetches())
		}
		slow.Release()
		<-manual
		if hevyErr(t, e) != "" {
			t.Fatal("failure row after a busy refusal")
		}
	})
	t.Run("throttled", func(t *testing.T) {
		e := secretarytest.New(t)
		stored := e.Svc.Now().Add(-2 * time.Hour)
		e.SeedHevy(stored)
		fake := &secretarytest.FakeHevy{Err: context.DeadlineExceeded}
		e.ConnectHevy(fake)
		_ = e.Svc.SyncTraining(context.Background()) // a real failure 0 s ago: the throttle starts
		before := hevyErr(t, e)
		f := freshener(e, 5*time.Second)
		start := time.Now()
		got := f.EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
		if time.Since(start) > 2*time.Second || !got.Stale || !got.AsOf.Equal(stored) {
			t.Fatalf("%+v", got)
		}
		if fake.Fetches() != 1 || hevyErr(t, e) != before {
			t.Fatal("the throttled call fetched again or changed the failure row")
		}
	})
}

func TestEnsureFreshUnconnectedAndNeverSynced(t *testing.T) {
	t.Run("not connected, nothing stored", func(t *testing.T) {
		e := secretarytest.New(t)
		got := freshener(e, time.Second).EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
		if got.Connected || !got.Stale || !got.AsOf.IsZero() {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("not connected, rows stored earlier", func(t *testing.T) {
		// The stored rows keep their real age: nothing is synced, and the answer
		// says so rather than pretending the copy is new or absent.
		e := secretarytest.New(t)
		stored := e.Svc.Now().Add(-2 * time.Hour)
		e.SeedHevy(stored)
		got := freshener(e, time.Second).EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
		if got.Connected || !got.Stale || !got.AsOf.Equal(stored) {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("connected but never synced", func(t *testing.T) {
		e := secretarytest.New(t)
		fake := &secretarytest.FakeHevy{}
		e.ConnectHevy(fake)
		got := freshener(e, time.Second).EnsureFresh(context.Background(), secretary.SourceHevy)[secretary.SourceHevy]
		if !got.Connected || !got.Stale || !got.AsOf.IsZero() || fake.Fetches() != 0 {
			t.Fatalf("%+v, %d fetches: the first fill belongs to the 37-minute loop", got, fake.Fetches())
		}
	})
}

// Peek never syncs, so a brief built from a GET cannot write (Issue #188).
func TestPeekNeverSyncs(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-5 * time.Hour))
	fake := &secretarytest.FakeHevy{}
	e.ConnectHevy(fake)
	before := secretarytest.DumpTables(t, e.DB)
	st := freshener(e, time.Second).Peek(secretary.SourceHevy)
	if !st.Stale || fake.Fetches() != 0 || len(secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB))) != 0 {
		t.Fatalf("%+v, %d fetches", st, fake.Fetches())
	}
}

func TestFreshLogsNoContent(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-2 * time.Hour)) // holds the title "Push day"
	fake := &secretarytest.FakeHevy{Err: context.DeadlineExceeded}
	e.ConnectHevy(fake)
	f := freshener(e, time.Second)
	f.EnsureFresh(context.Background(), secretary.SourceHevy, secretary.SourceMail)
	log := buf.String()
	for _, want := range []string{"secretary.fresh", "source=hevy", "outcome=failed", "source=mail", "outcome=unconnected", "ms="} {
		if !strings.Contains(log, want) {
			t.Errorf("log lacks %q:\n%s", want, log)
		}
	}
	for _, banned := range []string{"Push day", "Bench", "Routine A"} {
		if strings.Contains(log, banned) {
			t.Errorf("log holds %q", banned)
		}
	}
}

// The caller's context ends in the middle of a refresh (the request went away).
// The sync runs on a context without cancellation, so the provider is not told
// to stop, no false "could not be reached" is stored, and the sync finishes.
func TestEnsureFreshCallerCancelsMidRefresh(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-2 * time.Hour))
	slow := secretarytest.SlowHevy(true) // returns ctx.Err() if its context ends
	slow.Entries = nil
	e.ConnectHevy(slow)
	f := freshener(e, 5*time.Second)
	t.Cleanup(slow.Release)
	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan secretary.Freshness, 1)
	go func() { got <- f.EnsureFresh(ctx, secretary.SourceHevy)[secretary.SourceHevy] }()
	for i := 0; slow.Fetches() == 0 && i < 400; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	if slow.Fetches() != 1 {
		t.Fatal("the refresh never started")
	}
	cancel()
	select {
	case st := <-got:
		if !st.Stale {
			t.Fatalf("%+v", st)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("EnsureFresh did not return after the caller's cancel")
	}
	slow.Release()
	f.Wait()
	cache, _ := e.Svc.Repo.TrainingCache("hevy")
	if cache.Error != "" {
		t.Fatalf("a cancelled caller stored a Hevy failure: %q", cache.Error)
	}
	if !cache.SyncedAt.Equal(e.Svc.Now()) {
		t.Fatalf("the sync did not complete: synced at %v", cache.SyncedAt)
	}
}

func countingSlot(gate chan struct{}, calls *atomic.Int64) map[string]secretary.Syncer {
	return map[string]secretary.Syncer{secretary.SourceCalendar: {
		LastSync: func() (time.Time, error) { return time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC), nil },
		Sync:     func(context.Context) error { calls.Add(1); <-gate; return nil },
	}}
}

// One refresh per source, however the calls arrive: a source repeated inside
// one call, or two overlapping calls, share the first attempt.
func TestEnsureFreshOneRefreshPerSource(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }
	t.Run("repeated in one call", func(t *testing.T) {
		var calls atomic.Int64
		gate := make(chan struct{})
		f := secretary.NewFreshener(countingSlot(gate, &calls), 100*time.Millisecond, now)
		t.Cleanup(f.Wait)
		f.EnsureFresh(context.Background(), secretary.SourceCalendar, secretary.SourceCalendar, secretary.SourceCalendar)
		close(gate)
		f.Wait()
		if calls.Load() != 1 {
			t.Fatalf("%d refreshes", calls.Load())
		}
	})
	t.Run("overlapping calls", func(t *testing.T) {
		var calls atomic.Int64
		gate := make(chan struct{})
		f := secretary.NewFreshener(countingSlot(gate, &calls), 5*time.Second, now)
		t.Cleanup(f.Wait)
		var wg sync.WaitGroup
		results := make([]secretary.Freshness, 4)
		for i := range results {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = f.EnsureFresh(context.Background(), secretary.SourceCalendar)[secretary.SourceCalendar]
			}()
		}
		for i := 0; calls.Load() == 0 && i < 400; i++ {
			time.Sleep(5 * time.Millisecond)
		}
		time.Sleep(50 * time.Millisecond) // let every caller reach the wait
		close(gate)
		wg.Wait()
		f.Wait()
		if calls.Load() != 1 {
			t.Fatalf("%d refreshes for 4 overlapping calls", calls.Load())
		}
		for i := 1; i < len(results); i++ {
			if results[i] != results[0] {
				t.Fatalf("callers disagree: %+v vs %+v", results[0], results[i])
			}
		}
	})
}

// GYST-164 c3: one wait budget per Catalogue turn. Helpers for slow sources
// whose Sync ignores its context, as the real Hevy client's sync must.
type slowSource struct {
	gate  chan struct{}
	calls atomic.Int64
}

func newSlow() *slowSource { return &slowSource{gate: make(chan struct{})} }

func (s *slowSource) release() {
	select {
	case <-s.gate:
	default:
		close(s.gate)
	}
}

func budgetFreshener(t *testing.T, budget time.Duration, slow map[string]*slowSource) *secretary.Freshener {
	t.Helper()
	old := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	slots := map[string]secretary.Syncer{}
	for name, s := range slow {
		slots[name] = secretary.Syncer{
			LastSync: func() (time.Time, error) { return old, nil },
			Sync:     func(context.Context) error { s.calls.Add(1); <-s.gate; return nil },
		}
	}
	f := secretary.NewFreshener(slots, budget, func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) })
	t.Cleanup(func() {
		for _, s := range slow {
			s.release()
		}
		f.Wait()
	})
	return f
}

// Two slow sources reached by two different tools in one turn wait the budget
// once, not once per call. The sync still runs on, so no failure is recorded.
func TestOneWaitBudgetPerTurn(t *testing.T) {
	e := secretarytest.New(t)
	e.SeedHevy(e.Svc.Now().Add(-2 * time.Hour))
	slowHevy := secretarytest.SlowHevy(false)
	e.ConnectHevy(slowHevy)
	slowCal := newSlow()
	cat, err := secretary.New(secretary.Deps{Service: e.Svc, Health: secretarytest.FakeHealth{}, Budget: 500 * time.Millisecond,
		Syncers: map[string]secretary.Syncer{secretary.SourceCalendar: {
			LastSync: func() (time.Time, error) { return e.Svc.Now().Add(-2 * time.Hour), nil },
			Sync:     func(context.Context) error { slowCal.calls.Add(1); <-slowCal.gate; return nil },
		}}})
	if err != nil {
		t.Fatal(err)
	}
	// Cleanups run last-in first-out: release the gates, then wait, so a failed
	// assertion cannot hang the test on a blocked sync.
	t.Cleanup(cat.Wait)
	t.Cleanup(slowHevy.Release)
	t.Cleanup(slowCal.release)
	before := secretarytest.DumpTables(t, e.DB)
	start := time.Now()
	first := run(t, cat, "get_recent_workouts", `{}`)
	brief := run(t, cat, "get_day_brief", `{}`)
	if took := time.Since(start); took > 800*time.Millisecond || took < 400*time.Millisecond {
		t.Fatalf("two tools took %v; one 500 ms budget for the turn gives about 500 ms (per-call deadlines would give about 1 s)", took)
	}
	if first["stale"] != true || brief["calendar"].(map[string]any)["stale"] != true {
		t.Fatalf("stale not reported: %v / %v", first["stale"], brief["calendar"])
	}
	for i := 0; slowCal.calls.Load() == 0 && i < 200; i++ { // the goroutine may not have run yet
		time.Sleep(5 * time.Millisecond)
	}
	if slowCal.calls.Load() != 1 {
		t.Fatalf("calendar sync started %d times", slowCal.calls.Load())
	}
	slowHevy.Release()
	slowCal.release()
	cat.Wait()
	if hevyErr(t, e) != "" {
		t.Fatalf("a spent budget recorded a Hevy failure: %q", hevyErr(t, e))
	}
	if changed := secretarytest.Changed(before, secretarytest.DumpTables(t, e.DB)); len(changed) != 1 || changed[0] != "training_snapshots" {
		t.Fatalf("changed %v", changed)
	}
}

// Three slow sources in three sequential calls share one budget; once it is
// spent a new stale source still starts its sync but is not waited for.
func TestWaitBudgetSharedAcrossSources(t *testing.T) {
	slow := map[string]*slowSource{secretary.SourceCalendar: newSlow(), secretary.SourceMail: newSlow(), secretary.SourceHevy: newSlow()}
	f := budgetFreshener(t, 300*time.Millisecond, slow)
	start := time.Now()
	var got []secretary.Freshness
	for _, src := range []string{secretary.SourceCalendar, secretary.SourceMail, secretary.SourceHevy} {
		got = append(got, f.EnsureFresh(context.Background(), src)[src])
	}
	took := time.Since(start)
	if took > 600*time.Millisecond || took < 250*time.Millisecond {
		t.Fatalf("three calls took %v; want about the 300 ms budget once", took)
	}
	for i, g := range got {
		if !g.Stale {
			t.Errorf("call %d not stale: %+v", i, g)
		}
	}
	for src, s := range slow {
		for i := 0; s.calls.Load() == 0 && i < 200; i++ { // the goroutine may not have run yet
			time.Sleep(5 * time.Millisecond)
		}
		if s.calls.Load() != 1 {
			t.Errorf("%s: %d syncs, want one even after the budget was spent", src, s.calls.Load())
		}
	}
}

// Overlapping callers share one deadline: A blocks at 0 and B at 300 ms with a
// 600 ms budget, so B is done about 600 ms after A began, not 900 ms.
func TestWaitBudgetOverlappingCallsShareOneDeadline(t *testing.T) {
	slow := map[string]*slowSource{secretary.SourceCalendar: newSlow(), secretary.SourceMail: newSlow()}
	f := budgetFreshener(t, 600*time.Millisecond, slow)
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); f.EnsureFresh(context.Background(), secretary.SourceCalendar) }()
	time.Sleep(300 * time.Millisecond)
	f.EnsureFresh(context.Background(), secretary.SourceMail)
	wg.Wait()
	if took := time.Since(start); took > 750*time.Millisecond {
		t.Fatalf("B finished %v after A began; one shared 600 ms deadline gives about 600 ms (a per-caller deadline about 900 ms)", took)
	}
}

// Overlap is charged once: A blocked 0-500 ms and B 100-600 ms leave about
// 400 ms of a 1 s budget. Double counting would leave about 0, a budget reset
// per call 1 s.
func TestWaitBudgetOverlapCountedOnce(t *testing.T) {
	slow := map[string]*slowSource{secretary.SourceCalendar: newSlow(), secretary.SourceMail: newSlow(), secretary.SourceHevy: newSlow()}
	f := budgetFreshener(t, time.Second, slow)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); f.EnsureFresh(context.Background(), secretary.SourceCalendar) }()
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		f.EnsureFresh(context.Background(), secretary.SourceMail)
	}()
	time.Sleep(500 * time.Millisecond)
	slow[secretary.SourceCalendar].release()
	time.Sleep(100 * time.Millisecond)
	slow[secretary.SourceMail].release()
	wg.Wait()
	start := time.Now()
	f.EnsureFresh(context.Background(), secretary.SourceHevy)
	if left := time.Since(start); left < 250*time.Millisecond || left > 550*time.Millisecond {
		t.Fatalf("budget left %v, want about 400 ms", left)
	}
}

// Time the model spends between tool calls is free: a fast sync, a pause, then
// a slow source with a 300 ms budget still waits most of it.
func TestWaitBudgetIgnoresThinkTime(t *testing.T) {
	slow := map[string]*slowSource{secretary.SourceCalendar: newSlow(), secretary.SourceMail: newSlow()}
	slow[secretary.SourceCalendar].release() // finishes at once
	f := budgetFreshener(t, 300*time.Millisecond, slow)
	f.EnsureFresh(context.Background(), secretary.SourceCalendar)
	time.Sleep(300 * time.Millisecond)
	start := time.Now()
	f.EnsureFresh(context.Background(), secretary.SourceMail)
	if waited := time.Since(start); waited < 200*time.Millisecond {
		t.Fatalf("waited %v after a think pause; a deadline fixed at New would give about 0", waited)
	}
}

// A caller cancelled while waiting returns promptly and its leave is counted:
// a pause afterwards is not charged to the budget.
func TestWaitBudgetCancelFreesTheWait(t *testing.T) {
	slow := map[string]*slowSource{secretary.SourceCalendar: newSlow(), secretary.SourceMail: newSlow()}
	f := budgetFreshener(t, 600*time.Millisecond, slow)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	start := time.Now()
	f.EnsureFresh(ctx, secretary.SourceCalendar)
	if took := time.Since(start); took > 300*time.Millisecond {
		t.Fatalf("cancelled caller took %v", took)
	}
	time.Sleep(300 * time.Millisecond)
	start = time.Now()
	f.EnsureFresh(context.Background(), secretary.SourceMail)
	if waited := time.Since(start); waited < 400*time.Millisecond {
		t.Fatalf("waited %v; skipped leave accounting charges the pause and leaves about 200 ms", waited)
	}
}

// Only time spent blocked is charged: a slow result step after the sync (here
// the post-sync LastSync read) is not. The first call finishes its sync at once
// but its inspect takes 400 ms; a later slow source must still get nearly the
// whole 600 ms.
func TestWaitBudgetIgnoresResultHandling(t *testing.T) {
	old := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	var reads atomic.Int64
	mailGate := newSlow()
	slots := map[string]secretary.Syncer{
		secretary.SourceCalendar: {
			LastSync: func() (time.Time, error) {
				if reads.Add(1) >= 2 { // the read after the sync
					time.Sleep(400 * time.Millisecond)
				}
				return old, nil
			},
			Sync: func(context.Context) error { return nil },
		},
		secretary.SourceMail: {
			LastSync: func() (time.Time, error) { return old, nil },
			Sync:     func(context.Context) error { <-mailGate.gate; return nil },
		},
	}
	f := secretary.NewFreshener(slots, 600*time.Millisecond, func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) })
	// LIFO: release the gate, then wait, so a failed assertion cannot hang.
	t.Cleanup(f.Wait)
	t.Cleanup(mailGate.release)
	f.EnsureFresh(context.Background(), secretary.SourceCalendar)
	start := time.Now()
	f.EnsureFresh(context.Background(), secretary.SourceMail)
	if waited := time.Since(start); waited < 450*time.Millisecond {
		t.Fatalf("waited %v; result handling was charged to the budget (about 200 ms left)", waited)
	}
}
