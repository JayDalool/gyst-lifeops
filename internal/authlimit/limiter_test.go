package authlimit

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// fakeClock is safe for concurrent use so it can drive -race tests.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
}
func (c *fakeClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func newLimiter(c *fakeClock) *Limiter { return New(c.Now, DefaultPolicy) }

func mustAdmit(t *testing.T, l *Limiter, bucket, key string) *Attempt {
	t.Helper()
	a, d := l.Admit(bucket, key, "test")
	if d != Allow {
		t.Fatalf("Admit(%s,%s) = %v, want allow", bucket, key, d)
	}
	return a
}

func mustRefuse(t *testing.T, l *Limiter, bucket, key string, want Decision) {
	t.Helper()
	a, d := l.Admit(bucket, key, "test")
	if d != want || a != nil {
		t.Fatalf("Admit(%s,%s) = %v (attempt %v), want %v", bucket, key, d, a, want)
	}
}

func TestDefaultPolicyIsTheDecidedOne(t *testing.T) {
	p := DefaultPolicy
	if p.KeyMax != 5 || p.KeyWindow != 15*time.Minute || p.KeyBlock != 15*time.Minute {
		t.Errorf("key policy = %+v", p)
	}
	want := [4]time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}
	if p.Delays != want {
		t.Errorf("delays = %v, want 1/2/4/8s (D7)", p.Delays)
	}
	if p.GlobalMax != 50 || p.GlobalWindow != time.Hour || p.GlobalGap < 75*time.Second {
		t.Errorf("global policy = %+v, want 50 per hour and a gap of at least 75s (D2)", p)
	}
}

func TestDelaySequenceUsesNotBeforeTimes(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	for _, d := range DefaultPolicy.Delays {
		mustAdmit(t, l, "public", "k").Fail()
		c.Advance(d - time.Nanosecond)
		mustRefuse(t, l, "public", "k", KeyDelay) // refused, and not counted
		c.Advance(time.Nanosecond)                // exactly lastFailure + delay
	}
	mustAdmit(t, l, "public", "k").Fail() // the fifth failure
	mustRefuse(t, l, "public", "k", KeyBlocked)
}

func TestFifthFailureBlocksUntilExactExpiry(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	for _, d := range DefaultPolicy.Delays {
		mustAdmit(t, l, "public", "k").Fail()
		c.Advance(d)
	}
	mustAdmit(t, l, "public", "k").Fail() // 5th, at t5
	c.Advance(15*time.Minute - time.Nanosecond)
	mustRefuse(t, l, "public", "k", KeyBlocked)
	c.Advance(time.Nanosecond) // exactly t5 + 15m: half-open, so admitted
	mustAdmit(t, l, "public", "k").Succeed()
}

func TestRefusalsDoNotExtendTheBlock(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	for _, d := range DefaultPolicy.Delays {
		mustAdmit(t, l, "public", "k").Fail()
		c.Advance(d)
	}
	mustAdmit(t, l, "public", "k").Fail()
	for i := 0; i < 14; i++ {
		c.Advance(time.Minute)
		mustRefuse(t, l, "public", "k", KeyBlocked)
	}
	c.Advance(time.Minute) // 15 minutes after the 5th failure
	mustAdmit(t, l, "public", "k").Succeed()
}

func TestSuccessClearsOnlyItsKeyAndNotTheBucket(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	mustAdmit(t, l, "public", "a").Fail()
	mustAdmit(t, l, "public", "b").Fail()
	c.Advance(time.Second)
	mustAdmit(t, l, "public", "a").Succeed()
	if l.Keys() != 1 {
		t.Fatalf("keys = %d, want only b left", l.Keys())
	}
	l.mu.Lock()
	n := len(l.buckets["public"].fails)
	l.mu.Unlock()
	if n != 2 {
		t.Fatalf("bucket failures = %d, want 2: success must not reset the bucket", n)
	}
}

func TestBusyKeyAllowsExactlyOneInFlight(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	a := mustAdmit(t, l, "public", "k")
	mustRefuse(t, l, "public", "k", KeyBusy)
	a.Release()
	if l.Inflight() != 0 {
		t.Fatalf("inflight = %d after Release", l.Inflight())
	}
	if l.Keys() != 0 {
		t.Fatalf("a released fresh key must not linger, keys = %d", l.Keys())
	}
	mustAdmit(t, l, "public", "k").Release()
}

func TestSettleExactlyOnce(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	a := mustAdmit(t, l, "public", "k")
	a.Fail()
	a.Fail()
	a.Succeed()
	a.Release()
	l.mu.Lock()
	n := len(l.keys["public\x00k"].fails)
	l.mu.Unlock()
	if n != 1 || l.Inflight() != 0 {
		t.Fatalf("failures = %d inflight = %d, want 1 and 0", n, l.Inflight())
	}
	var nilAttempt *Attempt
	nilAttempt.Release() // must not panic
}

// engage fills the public bucket with 50 failures from 50 keys, one a minute.
func engage(t *testing.T, c *fakeClock, l *Limiter) time.Time {
	t.Helper()
	for i := 0; i < 50; i++ {
		mustAdmit(t, l, "public", fmt.Sprintf("k%d", i)).Fail()
		if i < 49 {
			c.Advance(time.Minute)
		}
	}
	return c.Now()
}

func TestGlobalGateSpacingAndRelease(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	engage(t, c, l)
	mustRefuse(t, l, "public", "fresh", GlobalGate)
	c.Advance(DefaultPolicy.GlobalGap - time.Nanosecond)
	mustRefuse(t, l, "public", "fresh", GlobalGate)
	c.Advance(time.Nanosecond)
	mustAdmit(t, l, "public", "fresh").Succeed() // the gate also covers a correct password
	mustRefuse(t, l, "public", "fresh2", GlobalGate)
	// A different bucket is untouched.
	mustAdmit(t, l, "tailnet", "x").Succeed()
}

func TestGlobalWindowBoundary(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	engage(t, c, l)                           // first failure at t0, the 50th at t0+49m
	c.Advance(11 * time.Minute)               // now is exactly t0+1h: the first failure drops out
	mustAdmit(t, l, "public", "fresh").Fail() // 49 in window: not yet engaged
	mustRefuse(t, l, "public", "fresh2", GlobalGate)
}

func TestGlobalClearsAfterWindow(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	engage(t, c, l)
	c.Advance(time.Hour) // the 50th failure is exactly one hour old: out of the window
	mustAdmit(t, l, "public", "f1").Fail()
	mustAdmit(t, l, "public", "f2").Fail()
}

func TestInflightCountsTowardTheGlobalCeiling(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	var held []*Attempt
	admitted := 0
	for i := 0; i < 100; i++ {
		a, d := l.Admit("public", fmt.Sprintf("k%d", i), "test")
		if d == Allow {
			admitted++
			held = append(held, a)
		}
	}
	// 50 in flight engage the gate; with the clock frozen only the 50 reserved
	// before engagement (plus none after) get in.
	if admitted != 50 {
		t.Fatalf("admitted %d concurrent attempts, want exactly the ceiling (50)", admitted)
	}
	for _, a := range held {
		a.Release()
	}
	if l.Inflight() != 0 {
		t.Fatalf("inflight = %d", l.Inflight())
	}
}

func TestPruneRemovesOnlyExpiredKeys(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	mustAdmit(t, l, "public", "old").Fail()
	c.Advance(time.Minute)
	mustAdmit(t, l, "public", "recent").Fail()
	c.Advance(DefaultPolicy.KeyWindow - time.Minute) // "old" is now exactly window old
	mustAdmit(t, l, "public", "trigger").Release()
	l.mu.Lock()
	_, oldThere := l.keys["public\x00old"]
	_, recentThere := l.keys["public\x00recent"]
	l.mu.Unlock()
	if oldThere || !recentThere {
		t.Fatalf("old present=%v recent present=%v, want false and true", oldThere, recentThere)
	}
}

func TestCapacityFailsClosedForNewKeysOnly(t *testing.T) {
	c := newClock()
	p := DefaultPolicy
	p.MaxKeys = 3
	l := New(c.Now, p)
	// Three blocked keys fill the table; blocked entries can never be evicted.
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("b%d", i)
		for j := range p.Delays {
			mustAdmit(t, l, "public", key).Fail()
			c.Advance(p.Delays[j])
		}
		mustAdmit(t, l, "public", key).Fail()
	}
	mustRefuse(t, l, "public", "new", Capacity)
	mustRefuse(t, l, "public", "b0", KeyBlocked) // existing keys keep their decisions
	if l.Keys() != 3 || l.Inflight() != 0 {
		t.Fatalf("keys=%d inflight=%d after a capacity refusal", l.Keys(), l.Inflight())
	}
}

func TestCapacityEvictsOldestIdleUnblockedKey(t *testing.T) {
	c := newClock()
	p := DefaultPolicy
	p.MaxKeys = 3
	l := New(c.Now, p)
	mustAdmit(t, l, "public", "a").Fail()
	c.Advance(time.Second)
	mustAdmit(t, l, "public", "b").Fail()
	c.Advance(time.Second)
	held := mustAdmit(t, l, "public", "c") // in flight: not evictable
	mustAdmit(t, l, "public", "d").Release()
	l.mu.Lock()
	_, a := l.keys["public\x00a"]
	_, b := l.keys["public\x00b"]
	_, cc := l.keys["public\x00c"]
	l.mu.Unlock()
	if a || !b || !cc {
		t.Fatalf("a=%v b=%v c=%v, want a evicted, b and the in-flight c kept", a, b, cc)
	}
	held.Release()
}

func TestPruneIsLazyAndBounded(t *testing.T) {
	c := newClock()
	p := DefaultPolicy
	p.MaxKeys = 50
	l := New(c.Now, p)
	for i := 0; i < 50; i++ {
		mustAdmit(t, l, "public", fmt.Sprintf("k%d", i)).Fail()
	}
	c.Advance(DefaultPolicy.KeyWindow + time.Minute)
	mustAdmit(t, l, "public", "fresh").Release()
	if l.Keys() > 1 {
		t.Fatalf("keys = %d, want expired entries pruned", l.Keys())
	}
}

func TestLimiterConcurrentManyKeys(t *testing.T) {
	c := newClock()
	p := DefaultPolicy
	p.MaxKeys = 100
	l := New(c.Now, p)
	var wg sync.WaitGroup
	for g := 0; g < 200; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(g)))
			for i := 0; i < 50; i++ {
				key := fmt.Sprintf("k%d", r.Intn(150))
				bucket := "public"
				if r.Intn(4) == 0 {
					bucket = "tailnet"
				}
				if a, d := l.Admit(bucket, key, "t"); d == Allow {
					switch r.Intn(3) {
					case 0:
						a.Fail()
					case 1:
						a.Succeed()
					default:
						a.Release()
					}
				}
				if r.Intn(5) == 0 {
					c.Advance(time.Duration(r.Intn(30)) * time.Second)
				}
			}
		}(g)
	}
	wg.Wait()
	if l.Inflight() != 0 {
		t.Fatalf("inflight = %d, want 0", l.Inflight())
	}
	if l.Keys() > p.MaxKeys {
		t.Fatalf("keys = %d exceeds cap %d", l.Keys(), p.MaxKeys)
	}
}

func TestConcurrentSameKeyAdmitsOne(t *testing.T) {
	c := newClock()
	l := newLimiter(c)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	start := make(chan struct{})
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, d := l.Admit("public", "k", "t"); d == Allow {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()
	if allowed != 1 {
		t.Fatalf("allowed = %d, want exactly 1", allowed)
	}
}
