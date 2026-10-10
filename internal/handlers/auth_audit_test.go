package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"lifeops-tracker/internal/authlimit"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// Login and logout audit rows (GYST-VN-007). Every value is fictional;
// addresses are RFC 5737 / RFC 3849 documentation ranges.

const auditWrongPW = "wrong-horse-fictional"

// auditRig is a throttled handler whose limiter and service share one pinned
// clock, with a captured log.
type auditRig struct {
	h   *Handler
	clk *throttleClock
	buf *bytes.Buffer
}

func newAuditRig(t *testing.T) *auditRig {
	t.Helper()
	h, clk := throttledHandler(t)
	h.Service.Clock.Now = clk.Now
	buf := &bytes.Buffer{}
	h.Logger = slog.New(slog.NewJSONHandler(buf, nil))
	return &auditRig{h: h, clk: clk, buf: buf}
}

func (a *auditRig) rows(t *testing.T) []repository.AuditEvent {
	t.Helper()
	rows, err := a.h.Service.Repo.RecentAuditEvents(context.Background(), 1000)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func (a *auditRig) lines(t *testing.T, msg string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(a.buf.String()), "\n") {
		if raw == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatal(err)
		}
		if m["msg"] == msg {
			out = append(out, m)
		}
	}
	return out
}

func cookieValue(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			return c.Value
		}
	}
	return ""
}

func rowText(e repository.AuditEvent) string {
	return strings.Join([]string{e.At, e.ActorType, e.ActorRef, e.Ingress, e.Action, e.TargetRef, e.Outcome, e.Detail, e.RequestID}, "|")
}

func TestAuditWrongPasswordRow(t *testing.T) {
	a := newAuditRig(t)
	want(t, "wrong", pub(a.h, auditWrongPW, "203.0.113.7"), http.StatusUnauthorized)
	rows := a.rows(t)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.Action != "login.failed" || r.ActorType != "human" || r.ActorRef != "client:203.0.113.0/24" ||
		r.Ingress != "public" || r.TargetRef != "account:owner" || r.Outcome != "failed" || r.Detail != "wrong-credentials" {
		t.Fatalf("row = %+v", r)
	}
	if strings.Contains(rowText(r), auditWrongPW) || strings.Contains(rowText(r), "203.0.113.7") {
		t.Fatalf("row leaks the password or full address: %+v", r)
	}
}

func TestAuditIPv6Label(t *testing.T) {
	a := newAuditRig(t)
	want(t, "wrong", pub(a.h, auditWrongPW, "2001:db8:1:2::5"), http.StatusUnauthorized)
	rows := a.rows(t)
	if len(rows) != 1 || rows[0].ActorRef != "client:2001:db8:1::/48" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestAuditTailnetSpoofedCFUsesPeerLabel(t *testing.T) {
	a := newAuditRig(t)
	want(t, "wrong", tail(a.h, auditWrongPW, "100.64.1.2", "203.0.113.9"), http.StatusUnauthorized)
	rows := a.rows(t)
	if len(rows) != 1 || rows[0].ActorRef != "client:100.64.1.0/24" || rows[0].Ingress != "tailnet" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestAuditSuccessRowAndRequestID(t *testing.T) {
	a := newAuditRig(t)
	server := RequestLog(a.h.Logger, a.h.Authenticated())
	form := url.Values{"password": {testPassword}}
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, r)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login = %d", rec.Code)
	}
	token := cookieValue(rec)
	rows := a.rows(t)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	e := rows[0]
	if e.Action != "login.succeeded" || e.Outcome != "success" || e.Detail != "" || e.ActorRef != services.SessionActorRef(token) {
		t.Fatalf("row = %+v", e)
	}
	if e.RequestID == "" || e.RequestID != rec.Header().Get(requestIDHeader) {
		t.Fatalf("request id %q vs header %q", e.RequestID, rec.Header().Get(requestIDHeader))
	}
	if strings.Contains(rowText(e), token) || strings.Contains(rowText(e), testPassword) {
		t.Fatalf("row leaks the token or password: %+v", e)
	}
	auth := a.lines(t, "auth")
	if len(auth) != 1 || auth[0]["event"] != "login.succeeded" || auth[0]["request_id"] != e.RequestID {
		t.Fatalf("auth lines = %v", auth)
	}
}

func TestAuditThrottledRowOncePerKeyPerWindow(t *testing.T) {
	a := newAuditRig(t)
	failFive(t, a.h, a.clk, "203.0.113.10")
	if n := len(a.rows(t)); n != 5 {
		t.Fatalf("rows after five failures = %d", n)
	}
	want(t, "sixth", pub(a.h, auditWrongPW, "203.0.113.10"), http.StatusTooManyRequests)
	for i := 0; i < 10; i++ {
		want(t, "refused", pub(a.h, auditWrongPW, "203.0.113.10"), http.StatusTooManyRequests)
	}
	throttled := func() int {
		n := 0
		for _, r := range a.rows(t) {
			if r.Detail == "throttled" {
				n++
				if r.Outcome != "denied" || r.Action != "login.failed" || r.ActorRef != "client:203.0.113.0/24" {
					t.Fatalf("throttled row = %+v", r)
				}
			}
		}
		return n
	}
	if n := throttled(); n != 1 {
		t.Fatalf("throttled rows = %d, want 1", n)
	}
}

func TestAuditThrottledRowAgainAfterWindow(t *testing.T) {
	a := newAuditRig(t)
	count := func() int {
		n := 0
		for _, r := range a.rows(t) {
			if r.Detail == "throttled" {
				n++
			}
		}
		return n
	}
	failFive(t, a.h, a.clk, "203.0.113.10")
	want(t, "refused", pub(a.h, auditWrongPW, "203.0.113.10"), http.StatusTooManyRequests)
	if count() != 1 {
		t.Fatalf("throttled rows = %d, want 1", count())
	}
	// Let the block lapse and earn a second one; the first row's window has
	// then passed, so the next refusal is recorded again.
	a.clk.Advance(authlimit.DefaultPolicy.KeyBlock)
	failFive(t, a.h, a.clk, "203.0.113.10")
	want(t, "refused again", pub(a.h, auditWrongPW, "203.0.113.10"), http.StatusTooManyRequests)
	if count() != 2 {
		t.Fatalf("throttled rows = %d, want 2", count())
	}
	want(t, "refused a third time", pub(a.h, auditWrongPW, "203.0.113.10"), http.StatusTooManyRequests)
	if count() != 2 {
		t.Fatalf("throttled rows = %d, want 2", count())
	}
}

func TestAuditRowsOnlyForRealAttempts(t *testing.T) {
	a := newAuditRig(t)
	// Cross-site POST: 403.
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password="+auditWrongPW))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://evil.example")
	r.Host = "gyst.zaylolabs.com"
	rec := httptest.NewRecorder()
	a.h.Authenticated().ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site = %d", rec.Code)
	}
	// Unreadable form: 400.
	r = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("a=%zz"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	a.h.Authenticated().ServeHTTP(rec, r)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad form = %d", rec.Code)
	}
	// No password configured: 503.
	a.h.Auth = services.AuthConfig{}
	rec = pub(a.h, auditWrongPW, "203.0.113.7")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("no password = %d", rec.Code)
	}
	if n := len(a.rows(t)); n != 0 {
		t.Fatalf("rows = %d, want 0", n)
	}
}

func TestAuditLogoutRows(t *testing.T) {
	a := newAuditRig(t)
	rec := pub(a.h, testPassword, "203.0.113.7")
	token := cookieValue(rec)
	if token == "" {
		t.Fatal("no session")
	}
	logout := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/logout", nil)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		a.h.Authenticated().ServeHTTP(w, r)
		return w
	}
	for _, c := range []*http.Cookie{nil, {Name: sessionCookieName, Value: "unknown-fictional"}} {
		w := logout(c)
		if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login" {
			t.Fatalf("logout = %d %s", w.Code, w.Header().Get("Location"))
		}
	}
	if n := len(a.rows(t)); n != 1 { // the login.succeeded row only
		t.Fatalf("rows = %d, want 1", n)
	}
	w := logout(&http.Cookie{Name: sessionCookieName, Value: token})
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login" {
		t.Fatalf("logout = %d", w.Code)
	}
	rows := a.rows(t)
	if len(rows) != 2 || rows[0].Action != "logout" {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].ActorRef != services.SessionActorRef(token) || rows[0].Outcome != "success" || strings.Contains(rowText(rows[0]), token) {
		t.Fatalf("logout row = %+v", rows[0])
	}
	// A second logout with the now dead token writes nothing.
	logout(&http.Cookie{Name: sessionCookieName, Value: token})
	if n := len(a.rows(t)); n != 2 {
		t.Fatalf("rows = %d, want 2", n)
	}
}

func TestAuditFailureNeverChangesLogin(t *testing.T) {
	rig := newLogRig(t)
	rig.breakTable(t, "audit_events")
	w := rig.do(rig.form(http.MethodPost, "/login", url.Values{"password": {testPassword}}, nil))
	if w.Code != http.StatusSeeOther || cookieValue(w) == "" {
		t.Fatalf("login = %d, cookie %q", w.Code, cookieValue(w))
	}
	w = rig.do(rig.form(http.MethodPost, "/login", url.Values{"password": {auditWrongPW}}, nil))
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), wrongText) {
		t.Fatalf("wrong password = %d", w.Code)
	}
	var failed int
	for _, l := range rig.lines(t) {
		if l["msg"] == "audit write failed" {
			failed++
			if l["level"] != "ERROR" || l["action"] == nil || l["request_id"] == nil || len(l) != 5 { // time, level, msg, action, request_id
				t.Fatalf("audit failure line = %v", l)
			}
		}
	}
	if failed != 2 {
		t.Fatalf("audit write failed lines = %d, want 2: %s", failed, rig.buf.String())
	}
	if strings.Contains(rig.buf.String(), auditWrongPW) {
		t.Fatal("password in the log")
	}
}

func TestAuditWrongPasswordLogLineMatchesRow(t *testing.T) {
	a := newAuditRig(t)
	server := RequestLog(a.h.Logger, a.h.Authenticated())
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password="+auditWrongPW))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(httptest.NewRecorder(), r)
	rows := a.rows(t)
	auth := a.lines(t, "auth")
	if len(rows) != 1 || len(auth) != 1 || auth[0]["request_id"] != rows[0].RequestID || rows[0].RequestID == "" {
		t.Fatalf("rows %+v, auth lines %v", rows, auth)
	}
}

func TestThrottleAuditGate(t *testing.T) {
	t0 := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	window := authlimit.DefaultPolicy.KeyBlock
	g := newThrottleAuditGate(3)
	if !g.allow("a", t0) || g.allow("a", t0.Add(window-time.Nanosecond)) {
		t.Fatal("window not enforced")
	}
	if !g.allow("a", t0.Add(window)) { // now == until allows
		t.Fatal("now == until must allow")
	}
	if !g.allow("b", t0) || !g.allow("c", t0) {
		t.Fatal("new keys must be allowed under the cap")
	}
	// Full (a, b, c), none expired: a new key is skipped and not stored.
	if g.allow("d", t0.Add(time.Minute)) || len(g.until) != 3 {
		t.Fatalf("full gate: len %d", len(g.until))
	}
	// b and c expired by t0+window; a was re-armed at t0+window. d prunes them.
	if !g.allow("d", t0.Add(window)) {
		t.Fatal("expired entries must be pruned to make room")
	}
	if len(g.until) != 2 { // a, d
		t.Fatalf("len after pruning = %d, want 2", len(g.until))
	}
	if g.allow("a", t0.Add(window+time.Minute)) {
		t.Fatal("a is still inside its renewed window")
	}
	var nilGate *throttleAuditGate
	if nilGate.allow("x", t0) {
		t.Fatal("nil gate must skip")
	}
}

func TestThrottleAuditGateCap(t *testing.T) {
	g := newThrottleAuditGate(throttleAuditMaxKeys)
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	for i := 0; i < throttleAuditMaxKeys+500; i++ {
		g.allow(fmt.Sprintf("k%d", i), now)
	}
	if len(g.until) != 1000 {
		t.Fatalf("len = %d, want 1000", len(g.until))
	}
}

func TestThrottleAuditGateConcurrent(t *testing.T) {
	g := newThrottleAuditGate(50)
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	var wg sync.WaitGroup
	allowed := make(chan struct{}, 1000)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if g.allow(fmt.Sprintf("k%d", j), now) {
					allowed <- struct{}{}
				}
			}
		}()
	}
	wg.Wait()
	if len(allowed) != 20 { // exactly one winner per key
		t.Fatalf("allowed = %d, want 20", len(allowed))
	}
}

func TestAuditConcurrentLogins(t *testing.T) {
	a := newAuditRig(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pub(a.h, auditWrongPW, fmt.Sprintf("203.0.113.%d", 100+i))
		}(i)
	}
	wg.Wait()
	if n := len(a.rows(t)); n != 8 {
		t.Fatalf("rows = %d, want 8", n)
	}
}
