package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"lifeops-tracker/internal/authlimit"
)

// Login throttle tests (GYST-VN-006). Every request goes through
// h.Authenticated() exactly as a browser's does. The clock is fake, so nothing
// sleeps. Addresses are documentation ranges (RFC 5737 / RFC 3849).

const (
	throttleText = "Too many sign-in attempts. Try again later."
	wrongText    = "Incorrect password."
	wrongPW      = "definitely not the password"
)

type throttleClock struct {
	mu sync.Mutex
	t  time.Time
}

func newThrottleClock() *throttleClock {
	return &throttleClock{t: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)}
}
func (c *throttleClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *throttleClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func throttledHandler(t *testing.T) (*Handler, *throttleClock) {
	t.Helper()
	h := authHandler(t)
	clk := newThrottleClock()
	h.loginClockForTest(clk.Now)
	return h, clk
}

// loginClockForTest swaps in a limiter on a fake clock with the production policy.
func (h *Handler) loginClockForTest(now func() time.Time) {
	h.loginLimiter = authlimit.New(now, authlimit.DefaultPolicy)
}

func post(h *Handler, host, peer, cf, password string) *httptest.ResponseRecorder {
	form := url.Values{"password": {password}}
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	r.Host = host
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-Forwarded-For", peer)
	if cf != "" {
		r.Header.Set("CF-Connecting-IP", cf)
	}
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, r)
	return rec
}

func pub(h *Handler, password, cf string) *httptest.ResponseRecorder {
	return post(h, "gyst.zaylolabs.com", "192.0.2.7", cf, password)
}

func tail(h *Handler, password, peer, cf string) *httptest.ResponseRecorder {
	return post(h, "gyst-test.example.ts.net:3443", peer, cf, password)
}

func hasSession(rec *httptest.ResponseRecorder) bool {
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			return true
		}
	}
	return false
}

func want(t *testing.T, label string, rec *httptest.ResponseRecorder, code int) {
	t.Helper()
	if rec.Code != code {
		t.Fatalf("%s: status %d, want %d: %s", label, rec.Code, code, rec.Body.String())
	}
	if code != http.StatusSeeOther && hasSession(rec) {
		t.Fatalf("%s: a refused or failed login set a session cookie", label)
	}
}

// failFive makes five wrong attempts from one public client, honouring the
// progressive delays, and returns the time of the fifth failure.
func failFive(t *testing.T, h *Handler, clk *throttleClock, cf string) time.Time {
	t.Helper()
	for i, d := range authlimit.DefaultPolicy.Delays {
		want(t, "failure "+string(rune('1'+i)), pub(h, wrongPW, cf), http.StatusUnauthorized)
		clk.Advance(d)
	}
	want(t, "failure 5", pub(h, wrongPW, cf), http.StatusUnauthorized)
	return clk.Now()
}

func TestThrottleFailuresOneToFive(t *testing.T) {
	h, clk := throttledHandler(t)
	for i, d := range authlimit.DefaultPolicy.Delays {
		rec := pub(h, wrongPW, "203.0.113.10")
		want(t, "failure", rec, http.StatusUnauthorized)
		if !strings.Contains(rec.Body.String(), wrongText) {
			t.Fatalf("failure %d body lacks %q", i+1, wrongText)
		}
		clk.Advance(d)
	}
	rec := pub(h, wrongPW, "203.0.113.10")
	want(t, "fifth failure", rec, http.StatusUnauthorized)
	if !strings.Contains(rec.Body.String(), wrongText) {
		t.Fatalf("fifth failure body lacks %q", wrongText)
	}
}

func TestThrottleSixthAttemptBlocked(t *testing.T) {
	h, clk := throttledHandler(t)
	failFive(t, h, clk, "203.0.113.10")
	clk.Advance(time.Minute)
	rec := pub(h, wrongPW, "203.0.113.10")
	want(t, "sixth", rec, http.StatusTooManyRequests)
	body := rec.Body.String()
	if !strings.Contains(body, throttleText) || strings.Contains(body, wrongText) {
		t.Fatalf("sixth body should carry the throttle text only: %s", body)
	}
}

func TestThrottleCorrectPasswordWhileBlocked(t *testing.T) {
	h, clk := throttledHandler(t)
	t5 := failFive(t, h, clk, "203.0.113.10")
	clk.Advance(time.Minute)
	wrongBody := pub(h, wrongPW, "203.0.113.10").Body.String()
	early := pub(h, testPassword, "203.0.113.10")
	want(t, "correct at +1m", early, http.StatusTooManyRequests)

	clk.Advance(t5.Add(15*time.Minute - time.Second).Sub(clk.Now()))
	late := pub(h, testPassword, "203.0.113.10")
	want(t, "correct at +14m59s", late, http.StatusTooManyRequests)

	if early.Body.String() != wrongBody || late.Body.String() != wrongBody {
		t.Fatal("a refusal must be byte-identical whether the password was right or wrong")
	}
}

func TestThrottleBlockExpires(t *testing.T) {
	h, clk := throttledHandler(t)
	t5 := failFive(t, h, clk, "203.0.113.10")
	clk.Advance(t5.Add(15*time.Minute - time.Nanosecond).Sub(clk.Now()))
	want(t, "1ns before expiry", pub(h, testPassword, "203.0.113.10"), http.StatusTooManyRequests)
	clk.Advance(time.Nanosecond)
	rec := pub(h, testPassword, "203.0.113.10")
	want(t, "at expiry", rec, http.StatusSeeOther)
	if !hasSession(rec) {
		t.Fatal("expected a session cookie once the block expired")
	}
}

func TestThrottleSuccessResetsKey(t *testing.T) {
	h, clk := throttledHandler(t)
	for _, d := range authlimit.DefaultPolicy.Delays[:3] {
		want(t, "pre-success failure", pub(h, wrongPW, "203.0.113.10"), http.StatusUnauthorized)
		clk.Advance(d)
	}
	want(t, "fourth failure", pub(h, wrongPW, "203.0.113.10"), http.StatusUnauthorized)
	clk.Advance(authlimit.DefaultPolicy.Delays[3])
	want(t, "success", pub(h, testPassword, "203.0.113.10"), http.StatusSeeOther)

	for _, d := range authlimit.DefaultPolicy.Delays {
		want(t, "post-success failure", pub(h, wrongPW, "203.0.113.10"), http.StatusUnauthorized)
		clk.Advance(d)
	}
	want(t, "fifth post-success failure", pub(h, wrongPW, "203.0.113.10"), http.StatusUnauthorized)
	clk.Advance(time.Minute)
	want(t, "sixth", pub(h, wrongPW, "203.0.113.10"), http.StatusTooManyRequests)
}

func TestThrottleCFHeaderUsedOnPublicSite(t *testing.T) {
	h, clk := throttledHandler(t)
	failFive(t, h, clk, "203.0.113.10")
	want(t, "A blocked", pub(h, wrongPW, "203.0.113.10"), http.StatusTooManyRequests)
	// Same peer, different Cloudflare client: a separate key.
	want(t, "B wrong", pub(h, wrongPW, "203.0.113.11"), http.StatusUnauthorized)
	clk.Advance(time.Second)
	want(t, "B correct", pub(h, testPassword, "203.0.113.11"), http.StatusSeeOther)
}

func TestThrottleSpoofedCFIgnoredOffPublicSite(t *testing.T) {
	for _, host := range []string{"gyst-test.example.ts.net:3443", "example.com"} {
		t.Run(host, func(t *testing.T) {
			h, clk := throttledHandler(t)
			spoof := func(i int) string { return "203.0.113." + string(rune('1'+i)) }
			for i, d := range authlimit.DefaultPolicy.Delays {
				want(t, "failure", post(h, host, "100.64.1.2", spoof(i), wrongPW), http.StatusUnauthorized)
				clk.Advance(d)
			}
			want(t, "fifth", post(h, host, "100.64.1.2", spoof(4), wrongPW), http.StatusUnauthorized)
			clk.Advance(time.Minute)
			want(t, "sixth, fresh spoofed CF", post(h, host, "100.64.1.2", spoof(5), wrongPW), http.StatusTooManyRequests)
			want(t, "correct password, fresh spoofed CF", post(h, host, "100.64.1.2", spoof(6), testPassword), http.StatusTooManyRequests)
		})
	}
}

// engageGlobal makes one failure from each of 50 Cloudflare clients, a minute
// apart (t0 .. t0+49m), which engages the public bucket.
func engageGlobal(t *testing.T, h *Handler, clk *throttleClock) {
	t.Helper()
	for i := 1; i <= 50; i++ {
		want(t, "global fill", pub(h, wrongPW, "198.51.100."+itoa(i)), http.StatusUnauthorized)
		if i < 50 {
			clk.Advance(time.Minute)
		}
	}
}

func itoa(i int) string { return strconv.Itoa(i) }

func TestThrottleGlobalCeilingEngages(t *testing.T) {
	h, clk := throttledHandler(t)
	engageGlobal(t, h, clk)
	want(t, "correct, fresh client, gate engaged", pub(h, testPassword, "203.0.113.200"), http.StatusTooManyRequests)
	clk.Advance(75*time.Second - time.Nanosecond)
	want(t, "just before the 75s gap", pub(h, testPassword, "203.0.113.200"), http.StatusTooManyRequests)
	clk.Advance(time.Nanosecond)
	want(t, "after the 75s gap", pub(h, testPassword, "203.0.113.200"), http.StatusSeeOther)
	want(t, "immediately after", pub(h, wrongPW, "203.0.113.201"), http.StatusTooManyRequests)
}

func TestThrottleGlobalGapIsAtLeast75Seconds(t *testing.T) {
	h, clk := throttledHandler(t)
	engageGlobal(t, h, clk)
	clk.Advance(30 * time.Second) // the Issue's old value must no longer admit
	want(t, "30s is not enough", pub(h, wrongPW, "203.0.113.200"), http.StatusTooManyRequests)
}

func TestThrottleGlobalClearsAfterWindow(t *testing.T) {
	h, clk := throttledHandler(t)
	engageGlobal(t, h, clk)
	clk.Advance(time.Hour)
	want(t, "first", pub(h, wrongPW, "203.0.113.200"), http.StatusUnauthorized)
	want(t, "second, back to back", pub(h, wrongPW, "203.0.113.201"), http.StatusUnauthorized)
}

func TestThrottleGlobalWindowBoundary(t *testing.T) {
	h, clk := throttledHandler(t)
	engageGlobal(t, h, clk)
	clk.Advance(11 * time.Minute) // exactly t0+1h: the first failure drops out
	want(t, "49 counted, not engaged", pub(h, wrongPW, "203.0.113.200"), http.StatusUnauthorized)
	want(t, "engaged again", pub(h, wrongPW, "203.0.113.201"), http.StatusTooManyRequests)
}

func TestThrottleIndependentKeys(t *testing.T) {
	h, clk := throttledHandler(t)
	failFive(t, h, clk, "203.0.113.10")
	want(t, "B wrong", pub(h, wrongPW, "203.0.113.11"), http.StatusUnauthorized)
	clk.Advance(time.Second)
	want(t, "B correct", pub(h, testPassword, "203.0.113.11"), http.StatusSeeOther)
	want(t, "tailnet correct", tail(h, testPassword, "100.64.1.2", ""), http.StatusSeeOther)
	want(t, "A still blocked", pub(h, testPassword, "203.0.113.10"), http.StatusTooManyRequests)
}

func TestThrottleGlobalBucketsSeparate(t *testing.T) {
	h, clk := throttledHandler(t)
	engageGlobal(t, h, clk)
	want(t, "public gated", pub(h, testPassword, "203.0.113.200"), http.StatusTooManyRequests)
	want(t, "tailnet unaffected", tail(h, testPassword, "100.64.1.2", ""), http.StatusSeeOther)
}

func TestThrottleConcurrentBurst(t *testing.T) {
	h, _ := throttledHandler(t) // clock frozen
	const n = 50
	codes := make([]int, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			codes[i] = pub(h, wrongPW, "203.0.113.10").Code
		}(i)
	}
	close(start)
	wg.Wait()
	var ok401, ok429 int
	for _, c := range codes {
		switch c {
		case http.StatusUnauthorized:
			ok401++
		case http.StatusTooManyRequests:
			ok429++
		}
	}
	if ok401 != 1 || ok429 != n-1 {
		t.Fatalf("401=%d 429=%d, want exactly 1 password check and %d refusals", ok401, ok429, n-1)
	}
	if h.loginLimiter.Inflight() != 0 {
		t.Fatalf("inflight = %d after the burst", h.loginLimiter.Inflight())
	}
}

func TestThrottleDelaySequence(t *testing.T) {
	h, clk := throttledHandler(t)
	for _, d := range authlimit.DefaultPolicy.Delays {
		want(t, "failure", pub(h, wrongPW, "203.0.113.10"), http.StatusUnauthorized)
		clk.Advance(d - time.Nanosecond)
		want(t, "one ns early", pub(h, wrongPW, "203.0.113.10"), http.StatusTooManyRequests)
		clk.Advance(time.Nanosecond)
	}
	// The early attempts were refused and not counted: the fifth failure is
	// still the one that blocks.
	want(t, "fifth failure", pub(h, wrongPW, "203.0.113.10"), http.StatusUnauthorized)
	want(t, "blocked", pub(h, wrongPW, "203.0.113.10"), http.StatusTooManyRequests)
}

func TestLoginErrorTextUnchanged(t *testing.T) {
	h, clk := throttledHandler(t)
	fresh := pub(h, wrongPW, "203.0.113.10")
	want(t, "fresh wrong", fresh, http.StatusUnauthorized)
	if !strings.Contains(fresh.Body.String(), wrongText) {
		t.Fatalf("401 body lost %q", wrongText)
	}
	failFive(t, h, clk, "203.0.113.20")
	refused := pub(h, wrongPW, "203.0.113.20")
	want(t, "refused", refused, http.StatusTooManyRequests)

	if got := refused.Header().Get("Retry-After"); got != "" {
		t.Errorf("Retry-After = %q, want none", got)
	}
	// The refusal page is the 401 page with only the message swapped: no
	// counts, times or hints can hide anywhere else in it.
	got := strings.Replace(refused.Body.String(), throttleText, "MSG", 1)
	base := strings.Replace(fresh.Body.String(), wrongText, "MSG", 1)
	if got != base {
		t.Errorf("refused page differs from the 401 page beyond the message:\n%s", got)
	}
	for _, banned := range []string{"remaining", "minute", "second", "retry", "locked"} {
		if strings.Contains(strings.ToLower(refused.Body.String()), banned) {
			t.Errorf("refused body contains %q", banned)
		}
	}
}

func TestLoginFailsClosedWithoutLimiter(t *testing.T) {
	h, _ := throttledHandler(t)
	h.loginLimiter = nil
	rec := pub(h, testPassword, "203.0.113.10")
	want(t, "no limiter", rec, http.StatusTooManyRequests)
	if !strings.Contains(rec.Body.String(), throttleText) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

// A refusal must never reach the password check. If it did, the 429 for a
// correct password would be a 303.
func TestThrottleRefusalNeverCreatesASession(t *testing.T) {
	h, clk := throttledHandler(t)
	failFive(t, h, clk, "203.0.113.10")
	for i := 0; i < 5; i++ {
		rec := pub(h, testPassword, "203.0.113.10")
		want(t, "blocked correct password", rec, http.StatusTooManyRequests)
		if rec.Header().Get("Location") != "" {
			t.Fatal("a refusal must not redirect")
		}
	}
}

// Failures that are not password attempts must not be counted.
func TestThrottleIgnoresNonAttempts(t *testing.T) {
	h, _ := throttledHandler(t)
	for i := 0; i < 8; i++ {
		r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password=x"))
		r.Host = "gyst.zaylolabs.com"
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Sec-Fetch-Site", "cross-site")
		r.Header.Set("X-Forwarded-For", "192.0.2.7")
		r.Header.Set("CF-Connecting-IP", "203.0.113.10")
		rec := httptest.NewRecorder()
		h.Authenticated().ServeHTTP(rec, r)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("cross-site POST = %d, want 403", rec.Code)
		}
	}
	want(t, "still allowed", pub(h, wrongPW, "203.0.113.10"), http.StatusUnauthorized)
	if h.loginLimiter.Keys() != 1 {
		t.Fatalf("keys = %d, want only the one real attempt", h.loginLimiter.Keys())
	}
}
