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

// WebBudget is how long one Catalogue (one turn) waits for syncs in total, over
// every EnsureFresh call it makes (GYST-164): the time actually spent blocked,
// not the time the model spends thinking between tool calls.
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
	slots map[string]Syncer
	now   func() time.Time

	mu sync.Mutex
	// left is the turn's unspent wait budget. waiting counts the EnsureFresh
	// calls blocked right now and since is when the first of them began, so
	// time that several calls spend blocked together is charged once, and a
	// call that begins while another waits inherits the same deadline.
	left    time.Duration
	waiting int
	// afterEnter is nil outside tests (see export_test.go).
	afterEnter func()
	since      time.Time
	done       map[string]Freshness
	flight     map[string]*attempt
	wg         sync.WaitGroup
}

// NewFreshener builds one. budget is the whole turn's wait; a zero budget means
// WebBudget.
func NewFreshener(slots map[string]Syncer, budget time.Duration, now func() time.Time) *Freshener {
	if budget <= 0 {
		budget = WebBudget
	}
	return &Freshener{slots: slots, left: budget, now: now, done: map[string]Freshness{}, flight: map[string]*attempt{}}
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
	if !ok {
		return st
	}
	st.Connected = slot.Connected == nil || slot.Connected()
	if slot.LastSync == nil {
		return st
	}
	// The stored copy has an age whether or not the source is connected now:
	// rows from an earlier sync are still what a tool will read.
	at, err := slot.LastSync()
	if err != nil || at.IsZero() {
		return st
	}
	st.AsOf = at
	st.Stale = f.now().Sub(at) >= freshLimit[source]
	return st
}

// attempt is one refresh of one source. It is reserved under the mutex before
// the goroutine starts, so a repeated source in one call, or an overlapping
// call, waits on it instead of starting a second refresh whose stale result
// could overwrite a good one.
type attempt struct {
	initial Freshness
	start   time.Time
	done    chan struct{} // closed when Sync returns
	err     error         // valid after done is closed
}

// EnsureFresh checks each source once for this Freshener and returns the state
// once the turn's wait budget is spent at the latest. A source that is still
// syncing when the budget runs out keeps syncing in the background and is
// reported stale (outcome "timeout"). It logs only the source, the outcome and the
// duration (09 B.8): never a title, a name or an error text.
func (f *Freshener) EnsureFresh(ctx context.Context, sources ...string) map[string]Freshness {
	out := map[string]Freshness{}
	waits := map[string]*attempt{}
	var order []string
	for _, source := range sources {
		f.mu.Lock()
		if prev, seen := f.done[source]; seen {
			f.mu.Unlock()
			out[source] = prev
			continue
		}
		if at, running := f.flight[source]; running {
			f.mu.Unlock()
			if _, queued := waits[source]; !queued {
				waits[source], order = at, append(order, source)
			}
			out[source] = at.initial
			continue
		}
		f.mu.Unlock()
		st := f.inspect(source)
		slot := f.slots[source]
		// Only a connected source whose stored copy has aged out is refreshed. One
		// that is not connected, or has never produced a copy, is left alone: its
		// own loop (the 37-minute Hevy loop) owns the first fill.
		if !(st.Connected && !st.AsOf.IsZero() && st.Stale && slot.Sync != nil) {
			out[source] = f.record(source, st, outcomeOf(st), time.Time{})
			continue
		}
		f.mu.Lock()
		if prev, seen := f.done[source]; seen { // decided while inspecting
			f.mu.Unlock()
			out[source] = prev
			continue
		}
		at, running := f.flight[source]
		if !running {
			at = &attempt{initial: st, start: time.Now(), done: make(chan struct{})} //gyst:instant duration for the log line
			f.flight[source] = at
			f.wg.Add(1)
			go func() {
				defer f.wg.Done()
				at.err = slot.Sync(context.WithoutCancel(ctx))
				close(at.done)
			}()
		}
		f.mu.Unlock()
		if _, queued := waits[source]; !queued {
			waits[source], order = at, append(order, source)
		}
		out[source] = at.initial
	}
	if len(order) == 0 {
		return out
	}
	// Only the time spent blocked on unfinished attempts is charged to the
	// budget, not the result handling after it (the post-sync inspect and the
	// log line), which can be slow on its own (GYST-164). So the waiting is one
	// phase and the results a second one, after the waiter has left.
	// Wall time on purpose (not the application clock, which tests pin).
	outcomes := map[string]string{}
	pending := false
	for _, source := range order {
		select {
		case <-waits[source].done:
		default:
			pending = true
		}
	}
	if pending {
		func() {
			deadline := f.enterWait()
			if f.afterEnter != nil {
				f.afterEnter() // test hook: pauses a caller between registering and waiting
			}
			// A defer so a cancelled or panicking caller cannot leave the count
			// up and charge the model's think time to the budget.
			defer f.leaveWait()
			wctx, cancel := context.WithDeadline(ctx, deadline)
			defer cancel()
			for _, source := range order {
				at := waits[source]
				select {
				case <-at.done: // already finished, even if the budget is spent
				default:
					select {
					case <-at.done:
					case <-wctx.Done():
						// The sync goes on in its goroutine; the next turn sees its rows.
						outcomes[source] = "timeout"
						if ctx.Err() != nil {
							outcomes[source] = "cancelled"
						}
					}
				}
			}
		}()
	}
	for _, source := range order {
		at := waits[source]
		st, outcome := at.initial, outcomes[source]
		if outcome == "" {
			if at.err != nil {
				outcome = "failed"
			} else {
				outcome, st = "synced", f.inspect(source)
			}
		}
		out[source] = f.record(source, st, outcome, at.start)
	}
	return out
}

// enterWait registers a blocked caller and returns the absolute time its wait
// must end: the turn's remaining budget counted from when the first of the
// currently blocked callers began, so every overlapping caller shares one
// deadline. It is absolute, not a duration, because a caller can be descheduled
// between here and starting its timer; a relative timer started later would
// outlive the shared deadline and let the turn exceed its budget (GYST-164).
// mu is never held while waiting.
func (f *Freshener) enterWait() time.Time {
	now := time.Now() //gyst:instant wall-clock wait budget, not a calendar day
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.waiting == 0 {
		f.since = now
	}
	f.waiting++
	return f.since.Add(f.left)
}

// leaveWait unregisters a caller. The last one out charges the time the group
// was blocked, once, however many callers overlapped.
func (f *Freshener) leaveWait() {
	now := time.Now() //gyst:instant wall-clock wait budget, not a calendar day
	f.mu.Lock()
	defer f.mu.Unlock()
	f.waiting--
	if f.waiting == 0 {
		f.left = max(f.left-now.Sub(f.since), 0)
	}
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

// record stores the first decision for a source and returns the stored one, so
// two callers waiting on the same attempt agree.
func (f *Freshener) record(source string, st Freshness, outcome string, start time.Time) Freshness {
	f.mu.Lock()
	if prev, seen := f.done[source]; seen {
		f.mu.Unlock()
		return prev
	}
	f.done[source] = st
	delete(f.flight, source)
	f.mu.Unlock()
	d := time.Duration(0)
	if !start.IsZero() {
		d = time.Since(start) //gyst:instant duration for the log line
	}
	slog.Info("secretary freshness", "event", "secretary.fresh", "source", source, "outcome", outcome, "ms", d.Milliseconds())
	return st
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
