package secretary

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"lifeops-tracker/internal/services"
)

// Freshness (Issue #236, 19A S2). Before a tool reads a synced copy, the
// Catalogue asks whether the copy is recent enough and, if not, tries one sync
// within a time budget. A slow or failing source never blocks the answer: the
// tool answers from the stored rows and says so (stale: true).
//
// Siri was dropped (19A S2, round 7 D1, D4, D5), so there is one budget only.

// WebBudget is how long one EnsureFresh call waits for syncs in total.
const WebBudget = 10 * time.Second

// Source names. Mail and calendar have no provider yet (chunks C-1 and C-4
// fill their slots); Hevy is wired through services.SyncTraining.
const (
	SourceHevy     = "hevy"
	SourceCalendar = "calendar"
	SourceMail     = "mail"
)

// freshLimit is how old a copy may be before a tool triggers a sync.
var freshLimit = map[string]time.Duration{
	SourceMail:     10 * time.Minute,
	SourceCalendar: 15 * time.Minute,
	SourceHevy:     60 * time.Minute,
}

func validSource(s string) bool { _, ok := freshLimit[s]; return ok }

// Syncer is the slot a source fills. All three functions are optional: a nil
// Connected means connected, a nil Sync means nothing can be refreshed.
type Syncer struct {
	// Connected says whether the source is set up at all.
	Connected func() bool
	// LastSync is when the stored copy was last refreshed; the zero time means
	// never. An error is treated as "unknown": no sync is started.
	LastSync func() (time.Time, error)
	// Sync refreshes the copy. It must bound itself (its own timeout and lock)
	// and must not depend on the caller's context being alive: it keeps running
	// after the budget ends, so a timeout here never turns into a false failure
	// in the provider's record. Its error text is never shown or logged.
	Sync func(ctx context.Context) error
}

// Freshness is the state of one source for one turn.
type Freshness struct {
	Source    string
	Connected bool
	// AsOf is when the stored copy was last refreshed; zero means never.
	AsOf time.Time
	// Stale is true when the copy is older than its limit, was never
	// refreshed, or the refresh did not finish or failed.
	Stale bool
}

// Freshener runs the freshness checks of one Catalogue (one per turn). A source
// is checked at most once per Freshener.
type Freshener struct {
	slots  map[string]Syncer
	budget time.Duration
	now    func() time.Time

	mu   sync.Mutex
	done map[string]Freshness
	wg   sync.WaitGroup
}

// NewFreshener builds one. A zero budget means WebBudget.
func NewFreshener(slots map[string]Syncer, budget time.Duration, now func() time.Time) *Freshener {
	if budget <= 0 {
		budget = WebBudget
	}
	return &Freshener{slots: slots, budget: budget, now: now, done: map[string]Freshness{}}
}

// Wait blocks until every sync started by EnsureFresh has finished. Tests call
// it in cleanup so no goroutine outlives its database.
func (f *Freshener) Wait() { f.wg.Wait() }

// Peek reads the state of a source without syncing anything. BuildDayBrief uses
// it, so a GET can build a brief and never write (Issue #188).
func (f *Freshener) Peek(source string) Freshness {
	f.mu.Lock()
	if st, ok := f.done[source]; ok {
		f.mu.Unlock()
		return st
	}
	f.mu.Unlock()
	return f.inspect(source)
}

// inspect reads connection and last-sync time and applies the age limit.
func (f *Freshener) inspect(source string) Freshness {
	st := Freshness{Source: source, Stale: true}
	slot, ok := f.slots[source]
	if !ok || (slot.Connected != nil && !slot.Connected()) {
		return st
	}
	st.Connected = true
	if slot.LastSync == nil {
		return st
	}
	at, err := slot.LastSync()
	if err != nil || at.IsZero() {
		return st
	}
	st.AsOf = at
	st.Stale = f.now().Sub(at) >= freshLimit[source]
	return st
}

// EnsureFresh checks each source once for this Freshener and returns the state
// after at most one budget. It logs only the source, the outcome and the
// duration (09 B.8): never a title, a name or an error text.
func (f *Freshener) EnsureFresh(ctx context.Context, sources ...string) map[string]Freshness {
	deadline := time.Now().Add(f.budget) //gyst:instant wall-clock budget for a wait, not a calendar day
	type pending struct {
		source string
		start  time.Time
		done   chan error
	}
	out := map[string]Freshness{}
	var waits []pending
	for _, source := range sources {
		f.mu.Lock()
		prev, seen := f.done[source]
		f.mu.Unlock()
		if seen {
			out[source] = prev
			continue
		}
		st := f.inspect(source)
		slot := f.slots[source]
		// Only a stored copy that has aged out is refreshed. A source that is not
		// connected, or has never produced a copy, is left alone: its own loop
		// (the 37-minute Hevy loop) owns the first fill.
		if st.Connected && !st.AsOf.IsZero() && st.Stale && slot.Sync != nil {
			p := pending{source: source, start: time.Now(), done: make(chan error, 1)} //gyst:instant duration for the log line
			f.wg.Add(1)
			go func() {
				defer f.wg.Done()
				p.done <- slot.Sync(context.WithoutCancel(ctx))
			}()
			waits = append(waits, p)
			out[source] = st
			continue
		}
		f.record(source, st, outcomeOf(st), time.Time{})
		out[source] = st
	}
	wctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	for _, p := range waits {
		st := out[p.source]
		var outcome string
		var err error
		select {
		case err = <-p.done: // already finished, even if the budget is spent
			outcome = "synced"
		default:
			select {
			case err = <-p.done:
				outcome = "synced"
			case <-wctx.Done():
				// The sync goes on in its goroutine; the next turn sees its rows.
				outcome = "timeout"
				if ctx.Err() != nil {
					outcome = "cancelled"
				}
			}
		}
		switch {
		case outcome == "synced" && err != nil:
			outcome = "failed"
		case outcome == "synced":
			st = f.inspect(p.source)
		}
		f.record(p.source, st, outcome, p.start)
		out[p.source] = st
	}
	return out
}

func outcomeOf(st Freshness) string {
	switch {
	case !st.Connected:
		return "unconnected"
	case st.AsOf.IsZero():
		return "never_synced"
	case st.Stale:
		return "stale_no_sync"
	}
	return "fresh"
}

func (f *Freshener) record(source string, st Freshness, outcome string, start time.Time) {
	f.mu.Lock()
	f.done[source] = st
	f.mu.Unlock()
	d := time.Duration(0)
	if !start.IsZero() {
		d = time.Since(start) //gyst:instant duration for the log line
	}
	slog.Info("secretary freshness", "event", "secretary.fresh", "source", source, "outcome", outcome, "ms", d.Milliseconds())
}

// HevySyncer fills the Hevy slot with the existing training sync. SyncTraining
// has its own TryLock, one-minute throttle and three-minute cap, and records a
// failure row when the provider errors. The budget therefore must not cancel
// it: Freshener runs it on a context without cancellation, because the Hevy
// client turns a cancelled request into a stored "could not be reached" error
// that the Training page would then show (hevy.go, GYST-163 plan review).
func HevySyncer(svc *services.Service) Syncer {
	return Syncer{
		Connected: func() bool { return svc.Training != nil && svc.Training.Provider != nil },
		LastSync: func() (time.Time, error) {
			cache, err := svc.Repo.TrainingCache("hevy")
			return cache.SyncedAt, err
		},
		Sync: svc.SyncTraining,
	}
}
