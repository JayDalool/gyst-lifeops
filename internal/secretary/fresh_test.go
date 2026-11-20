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
