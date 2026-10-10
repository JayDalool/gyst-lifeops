// Package authlimit throttles failed sign-in attempts (GYST-VN-006).
//
// It knows nothing about HTTP. A caller asks Admit for permission to check a
// password for a (bucket, key) pair, checks it, then reports the outcome with
// Fail, Succeed or Release. The decision is made before the password is
// compared, so a refusal can never reveal whether a password was right.
//
// State is in memory and is lost on restart. That is accepted for a
// single-process deployment and is documented in docs/ARCHITECTURE.md.
//
// Two layers apply to every attempt:
//
//   - Per key: at most one attempt in flight, a "not before" delay after each
//     failure (never a sleep), and a time-bound block after KeyMax failures
//     inside KeyWindow.
//   - Per bucket (one bucket per ingress): once GlobalMax failures are inside
//     GlobalWindow, admissions must be at least GlobalGap apart, for everyone,
//     including the correct password.
//
// All windows are half-open: a failure at time t counts while
// now-window < t <= now. A block holds while now < blockedUntil.
package authlimit

import (
	"log/slog"
	"sync"
	"time"
)

// Decision is the outcome of Admit.
type Decision int

const (
	// Allow means the caller reserved an attempt and may check the password.
	Allow Decision = iota
	// KeyBlocked means the key is inside a block (or has used its failures).
	KeyBlocked
	// KeyDelay means the key's progressive "not before" time has not passed.
	KeyDelay
	// KeyBusy means another attempt for the key is already in flight.
	KeyBusy
	// GlobalGate means the bucket is engaged and its gap has not passed.
	GlobalGate
	// Capacity means the key table is full of entries that cannot be evicted.
	Capacity
)

func (d Decision) String() string {
	switch d {
	case Allow:
		return "allow"
	case KeyBlocked:
		return "key-blocked"
	case KeyDelay:
		return "key-delay"
	case KeyBusy:
		return "key-busy"
	case GlobalGate:
		return "global-gate"
	case Capacity:
		return "capacity"
	}
	return "unknown"
}

// Policy holds every tunable. Production uses DefaultPolicy; nothing reads the
// environment, so a limit cannot be loosened silently.
type Policy struct {
	KeyMax    int           // failures inside KeyWindow that trigger a block
	KeyWindow time.Duration // window for counting a key's failures
	KeyBlock  time.Duration // how long a key stays blocked
	Delays    [4]time.Duration

	GlobalMax    int           // failures inside GlobalWindow that engage a bucket
	GlobalWindow time.Duration // window for counting a bucket's failures
	GlobalGap    time.Duration // minimum spacing between admissions while engaged

	MaxKeys    int           // hard cap on tracked keys
	PruneEvery time.Duration // how often Admit sweeps expired keys
}

// DefaultPolicy is the production policy (Issue #4, decisions D2, D6, D7).
var DefaultPolicy = Policy{
	KeyMax:       5,
	KeyWindow:    15 * time.Minute,
	KeyBlock:     15 * time.Minute,
	Delays:       [4]time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second},
	GlobalMax:    50,
	GlobalWindow: time.Hour,
	GlobalGap:    75 * time.Second,
	MaxKeys:      10000,
	PruneEvery:   time.Minute,
}

type keyState struct {
	fails        []time.Time // most recent failures, at most KeyMax
	inflight     int         // 0 or 1
	lastFailure  time.Time
	blockedUntil time.Time
}

type bucketState struct {
	fails     []time.Time // most recent failures, at most GlobalMax
	inflight  int
	lastAdmit time.Time
	engaged   bool // for transition logging only
	refused   int  // gate refusals since engaging, for the log line
}

// Limiter is safe for concurrent use. One mutex guards all state; critical
// sections are O(1) apart from the periodic prune and the at-capacity eviction.
type Limiter struct {
	mu        sync.Mutex
	now       func() time.Time
	p         Policy
	keys      map[string]*keyState
	buckets   map[string]*bucketState
	lastPrune time.Time
}

// New builds a Limiter. now is time.Now in production and a fake in tests.
func New(now func() time.Time, p Policy) *Limiter {
	return &Limiter{
		now:     now,
		p:       p,
		keys:    make(map[string]*keyState),
		buckets: make(map[string]*bucketState),
	}
}

// Attempt is a reservation returned by Admit. Exactly one of Fail or Succeed
// settles it; Release is a safe no-op after either, so it can be deferred.
type Attempt struct {
	l       *Limiter
	bucket  string
	mapKey  string
	label   string
	settled bool // guarded by l.mu
}

func mapKey(bucket, key string) string { return bucket + "\x00" + key }

func (l *Limiter) bucket(name string) *bucketState {
	b := l.buckets[name]
	if b == nil {
		b = &bucketState{}
		l.buckets[name] = b
	}
	return b
}

// countIn returns how many of fails fall inside the half-open window.
func countIn(fails []time.Time, now time.Time, window time.Duration) int {
	cutoff := now.Add(-window)
	n := 0
	for _, t := range fails {
		if t.After(cutoff) && !t.After(now) {
			n++
		}
	}
	return n
}

// push appends t, keeping only the most recent max entries.
func push(fails []time.Time, t time.Time, max int) []time.Time {
	if max < 1 {
		max = 1
	}
	if len(fails) >= max {
		copy(fails, fails[len(fails)-max+1:])
		fails = fails[:max-1]
	}
	return append(fails, t)
}

// globalLoad is the bucket's failures inside its window. The caller adds
// in-flight attempts where it needs them.
func (l *Limiter) globalLoad(b *bucketState, now time.Time) int {
	return countIn(b.fails, now, l.p.GlobalWindow)
}

// updateEngaged records the engaged/disengaged transition for logging. It is
// detected lazily, on the next access, not by a timer.
func (l *Limiter) updateEngaged(name string, b *bucketState, now time.Time) {
	engaged := l.globalLoad(b, now) >= l.p.GlobalMax
	switch {
	case engaged && !b.engaged:
		b.engaged = true
		b.refused = 0
		slog.Warn("login throttle", "event", "gate_engaged", "bucket", name)
	case !engaged && b.engaged:
		b.engaged = false
		slog.Warn("login throttle", "event", "gate_disengaged", "bucket", name, "refused", b.refused)
		b.refused = 0
	}
}

// prune removes keys that can no longer affect a decision. Caller holds mu.
func (l *Limiter) prune(now time.Time) {
	for k, s := range l.keys {
		if l.expired(s, now) {
			delete(l.keys, k)
		}
	}
	l.lastPrune = now
}

func (l *Limiter) expired(s *keyState, now time.Time) bool {
	return s.inflight == 0 && !now.Before(s.blockedUntil) && countIn(s.fails, now, l.p.KeyWindow) == 0
}

// evictOne removes the unblocked, idle key with the oldest last failure and
// reports whether it removed anything. Blocked and in-flight keys are never
// evicted. Caller holds mu.
func (l *Limiter) evictOne(now time.Time) bool {
	var victim string
	var oldest time.Time
	found := false
	for k, s := range l.keys {
		if s.inflight != 0 || now.Before(s.blockedUntil) {
			continue
		}
		if !found || s.lastFailure.Before(oldest) {
			victim, oldest, found = k, s.lastFailure, true
		}
	}
	if found {
		delete(l.keys, victim)
	}
	return found
}

// Admit asks to check one password for (bucket, key). label is a short,
// already-truncated description of the client used only in log lines; it is
// never part of a decision. On Allow the returned Attempt must be settled.
func (l *Limiter) Admit(bucket, key, label string) (*Attempt, Decision) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()

	if now.Sub(l.lastPrune) >= l.p.PruneEvery {
		l.prune(now)
	}

	mk := mapKey(bucket, key)
	s := l.keys[mk]

	// Per-key checks. Refusals are read-only and create nothing.
	if s != nil {
		if now.Before(s.blockedUntil) {
			return nil, KeyBlocked
		}
		if s.inflight > 0 {
			return nil, KeyBusy
		}
		n := countIn(s.fails, now, l.p.KeyWindow)
		if n >= l.p.KeyMax {
			return nil, KeyBlocked
		}
		if n >= 1 && n <= len(l.p.Delays) && now.Before(s.lastFailure.Add(l.p.Delays[n-1])) {
			return nil, KeyDelay
		}
	}

	// Global checks. In-flight attempts count, so a burst cannot overshoot.
	b := l.bucket(bucket)
	l.updateEngaged(bucket, b, now)
	if l.globalLoad(b, now)+b.inflight >= l.p.GlobalMax {
		if now.Before(b.lastAdmit.Add(l.p.GlobalGap)) {
			b.refused++
			return nil, GlobalGate
		}
	}

	// Capacity applies only to a key we have not seen.
	if s == nil {
		if len(l.keys) >= l.p.MaxKeys {
			l.prune(now)
		}
		if len(l.keys) >= l.p.MaxKeys && !l.evictOne(now) {
			return nil, Capacity
		}
		s = &keyState{}
		l.keys[mk] = s
	}

	s.inflight = 1
	b.inflight++
	b.lastAdmit = now
	return &Attempt{l: l, bucket: bucket, mapKey: mk, label: label}, Allow
}

// Fail records a failed password. It may block the key or engage the bucket.
func (a *Attempt) Fail() {
	l := a.l
	l.mu.Lock()
	defer l.mu.Unlock()
	if a.settled {
		return
	}
	a.settled = true
	now := l.now()

	b := l.bucket(a.bucket)
	if b.inflight > 0 {
		b.inflight--
	}
	b.fails = push(b.fails, now, l.p.GlobalMax)
	l.updateEngaged(a.bucket, b, now)

	s := l.keys[a.mapKey]
	if s == nil {
		// Evicted is impossible for an in-flight key; recreate defensively.
		s = &keyState{}
		l.keys[a.mapKey] = s
	}
	s.inflight = 0
	s.fails = push(s.fails, now, l.p.KeyMax)
	s.lastFailure = now
	if countIn(s.fails, now, l.p.KeyWindow) >= l.p.KeyMax {
		s.blockedUntil = now.Add(l.p.KeyBlock)
		slog.Warn("login throttle", "event", "lockout", "bucket", a.bucket, "client", a.label, "block_for", l.p.KeyBlock.String())
	}
}

// Succeed records a correct password: the key's history is cleared. The
// bucket's failure history is deliberately not reset.
func (a *Attempt) Succeed() {
	l := a.l
	l.mu.Lock()
	defer l.mu.Unlock()
	if a.settled {
		return
	}
	a.settled = true
	if b := l.bucket(a.bucket); b.inflight > 0 {
		b.inflight--
	}
	delete(l.keys, a.mapKey)
}

// Release undoes a reservation that ended without a verdict (a server error or
// a panic). It is a no-op once Fail or Succeed has run, and is safe on nil.
func (a *Attempt) Release() {
	if a == nil {
		return
	}
	l := a.l
	l.mu.Lock()
	defer l.mu.Unlock()
	if a.settled {
		return
	}
	a.settled = true
	now := l.now()
	if b := l.bucket(a.bucket); b.inflight > 0 {
		b.inflight--
	}
	if s := l.keys[a.mapKey]; s != nil {
		s.inflight = 0
		if l.expired(s, now) {
			delete(l.keys, a.mapKey)
		}
	}
}

// Inflight returns the number of reserved, unsettled attempts across buckets.
// It exists so tests can prove no reservation leaks.
func (l *Limiter) Inflight() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, b := range l.buckets {
		n += b.inflight
	}
	return n
}

// Keys returns the number of tracked keys, for tests of the memory bound.
func (l *Limiter) Keys() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.keys)
}
