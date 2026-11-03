package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// Structured logging and sanitised client errors (GYST-VN-015). Every value
// below is fictional. Requests go through RequestLog and Authenticated exactly
// as the app wires them.

type logRig struct {
	h      *Handler
	db     *sql.DB
	buf    *bytes.Buffer
	server http.Handler
}

func newLogRig(t *testing.T) *logRig {
	t.Helper()
	database := migratedTestDB(t)
	h := New(services.New(repository.New(database)), "../../web/templates")
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	h.Auth = services.AuthConfig{PasswordHash: string(hash)}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))
	h.Logger = logger
	return &logRig{h: h, db: database, buf: buf, server: RequestLog(logger, h.Authenticated())}
}

func (rig *logRig) do(r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	rig.server.ServeHTTP(w, r)
	return w
}

func (rig *logRig) form(method, path string, values url.Values, cookie *http.Cookie) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	return r
}

func (rig *logRig) signIn(t *testing.T) *http.Cookie {
	t.Helper()
	w := rig.do(rig.form(http.MethodPost, "/login", url.Values{"password": {testPassword}}, nil))
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			return c
		}
	}
	t.Fatalf("sign in failed: %d", w.Code)
	return nil
}

// breakTable makes every query on the table fail with a database error.
func (rig *logRig) breakTable(t *testing.T, name string) {
	t.Helper()
	if _, err := rig.db.Exec("ALTER TABLE " + name + " RENAME TO " + name + "_gone"); err != nil {
		t.Fatalf("rename %s: %v", name, err)
	}
}

func (rig *logRig) lines(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(rig.buf.String()), "\n") {
		if raw == "" {
			continue
		}
		var line map[string]any
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			t.Fatalf("log line is not JSON: %q: %v", raw, err)
		}
		out = append(out, line)
	}
	return out
}

func (rig *logRig) linesWith(t *testing.T, key, value string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range rig.lines(t) {
		if line[key] == value {
			out = append(out, line)
		}
	}
	return out
}

var requestIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

func TestServerErrorIsGenericWithRequestIDAndOneLogLine(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.breakTable(t, "tasks")
	rig.buf.Reset()

	r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	r.AddCookie(cookie)
	w := rig.do(r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	id := w.Header().Get("X-Request-Id")
	if !requestIDPattern.MatchString(id) {
		t.Fatalf("X-Request-Id = %q, want 16 hex characters", id)
	}
	if got, want := strings.TrimSpace(w.Body.String()), "Something went wrong. Reference: "+id+"."; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if strings.Contains(strings.ToLower(w.Body.String()), "no such table") {
		t.Fatal("database error text reached the browser")
	}

	if n := strings.Count(rig.buf.String(), id); n != 1 {
		t.Fatalf("request id appears in %d places, want exactly one log line: %s", n, rig.buf.String())
	}
	lines := rig.lines(t)
	if len(lines) != 1 {
		t.Fatalf("%d log lines, want 1: %s", len(lines), rig.buf.String())
	}
	line := lines[0]
	for _, key := range []string{"time", "level", "msg", "request_id", "method", "route", "status", "error"} {
		if _, ok := line[key]; !ok {
			t.Errorf("log line has no %q: %v", key, line)
		}
	}
	if line["level"] != "ERROR" || line["request_id"] != id || line["method"] != "GET" || line["route"] != "/tasks" || line["status"] != float64(500) {
		t.Errorf("unexpected log line: %v", line)
	}
}

func TestLogLineNeverHoldsRequestSecrets(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.breakTable(t, "tasks")
	rig.buf.Reset()

	form := url.Values{"entity_type": {"task"}, "entity_id": {"1"}, "url": {"https://example.test/x"}, "password": {"wrong-horse-fictional"}}
	r := rig.form(http.MethodPost, "/external-links?token=query-secret-fictional", form, cookie)
	r.Header.Set("Authorization", "Bearer test-token-fictional")
	w := rig.do(r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want the forced 500: %s", w.Code, w.Body.String())
	}
	logged := rig.buf.String()
	if strings.Count(strings.TrimSpace(logged), "\n") != 0 || logged == "" {
		t.Fatalf("want exactly one log line: %s", logged)
	}
	for name, secret := range map[string]string{
		"password":      "wrong-horse-fictional",
		"authorization": "test-token-fictional",
		"session":       cookie.Value,
		"query":         "query-secret-fictional",
		"cookie name":   sessionCookieName,
	} {
		if strings.Contains(logged, secret) {
			t.Errorf("log line contains the %s value", name)
		}
		if strings.Contains(w.Body.String(), secret) {
			t.Errorf("response contains the %s value", name)
		}
	}
}

func TestRouteLabel(t *testing.T) {
	token := strings.Repeat("a1b2c3d4", 4)
	for path, want := range map[string]string{
		"/":                        "/",
		"/today":                   "/today",
		"/tasks/42/edit":           "/tasks/:id/edit",
		"/calendar/feed/" + token:  "/calendar/feed/:token",
		"/calendar/feed/short":     "/calendar/feed/:token",
		"/calendar.ics":            "/calendar.ics",
		"/health/coach":            "/health/coach",
		"/Tasks/Upper":             "/:param/:param",
		"/tasks/" + token + "/x":   "/tasks/:param/x",
		"/x/9223372036854775807":   "/x/:id",
		"/a%2Fb/ünï":               "/:param/:param",
		"/login/":                  "/login",
		"//tasks//7":               "/tasks/:id",
		"/api/health/" + token:     "/api/health/:param",
		"/settings/calendar-token": "/settings/calendar-token",
	} {
		if got := routeLabel(path); got != want {
			t.Errorf("routeLabel(%q) = %q, want %q", path, got, want)
		}
	}
	// The query string is not part of a path, and r.URL.Path never has one.
	r := httptest.NewRequest(http.MethodGet, "/tasks/5?token=query-secret-fictional", nil)
	if got := routeLabel(r.URL.Path); strings.Contains(got, "query-secret-fictional") {
		t.Errorf("route label %q holds the query string", got)
	}
}

func TestRequestIDIsServerGeneratedAndUnique(t *testing.T) {
	rig := newLogRig(t)
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		r := httptest.NewRequest(http.MethodGet, "/login", nil)
		r.Header.Set("X-Request-Id", "client-chosen-id")
		id := rig.do(r).Header().Get("X-Request-Id")
		if !requestIDPattern.MatchString(id) {
			t.Fatalf("id %q is not 16 hex characters", id)
		}
		if seen[id] {
			t.Fatalf("id %q repeated", id)
		}
		seen[id] = true
	}
}

func TestUserErrorHelperShowsOnlySafeText(t *testing.T) {
	rig := newLogRig(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/safe", func(w http.ResponseWriter, r *http.Request) {
		userError(w, &services.UserError{Message: "fixed text"}, http.StatusBadRequest, "Generic.")
	})
	mux.HandleFunc("/internal", func(w http.ResponseWriter, r *http.Request) {
		userError(w, errors.New("pq: secret-detail-fictional"), http.StatusBadRequest, "Generic.")
	})
	server := RequestLog(rig.h.log(), mux)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/safe", nil))
	if w.Code != http.StatusBadRequest || strings.TrimSpace(w.Body.String()) != "fixed text" {
		t.Fatalf("safe: %d %q", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/internal", nil))
	id := w.Header().Get("X-Request-Id")
	if w.Code != http.StatusBadRequest || strings.TrimSpace(w.Body.String()) != "Generic. Reference: "+id+"." {
		t.Fatalf("internal: %d %q", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "secret-detail-fictional") {
		t.Fatal("internal error text reached the browser")
	}
	lines := rig.linesWith(t, "request_id", id)
	if len(lines) != 1 || !strings.Contains(lines[0]["error"].(string), "secret-detail-fictional") {
		t.Fatalf("want one log line with the detail, got %v", lines)
	}
	if len(rig.lines(t)) != 1 {
		t.Fatalf("the safe error must not log: %s", rig.buf.String())
	}
}

func TestErrorTextInLogIsCapped(t *testing.T) {
	if got := cappedError(errors.New(strings.Repeat("é", 400))); len(got) > maxLoggedError || !strings.HasPrefix(got, "é") {
		t.Fatalf("capped error length %d", len(got))
	}
}

func TestPrayerSettingsKeepValidationText(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.buf.Reset()
	w := rig.do(rig.form(http.MethodPost, "/settings/prayer", url.Values{"latitude": {"95"}}, cookie))
	if w.Code != http.StatusBadRequest || strings.TrimSpace(w.Body.String()) != "latitude must be between -90 and 90" {
		t.Fatalf("%d %q", w.Code, w.Body.String())
	}
}

func TestExternalLinkSitesShowSafeTextOrGenericWithReference(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)

	w := rig.do(rig.form(http.MethodPost, "/external-links", url.Values{"entity_type": {"task"}, "entity_id": {"1"}, "url": {"not a url"}}, cookie))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad link: %d %q", w.Code, w.Body.String())
	}
	if text := strings.TrimSpace(w.Body.String()); text != "link target not found" && text != "external link must be an absolute http or https URL" {
		t.Fatalf("unexpected text %q", text)
	}

	rig.breakTable(t, "tasks")
	rig.buf.Reset()
	w = rig.do(rig.form(http.MethodPost, "/external-links", url.Values{"entity_type": {"task"}, "entity_id": {"1"}, "url": {"https://example.test/x"}}, cookie))
	id := w.Header().Get("X-Request-Id")
	if w.Code != http.StatusInternalServerError || strings.TrimSpace(w.Body.String()) != "Something went wrong. Reference: "+id+"." {
		t.Fatalf("internal: %d %q", w.Code, w.Body.String())
	}
	if len(rig.linesWith(t, "request_id", id)) != 1 {
		t.Fatalf("want one line: %s", rig.buf.String())
	}
}

func TestPlanImportMissingTargetIsFixedText(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.buf.Reset()
	form := url.Values{"markdown": {"# Plan\n- Task one"}, "action": {"import"}, "destination": {"project"}, "target_id": {"987654"}}
	w := rig.do(rig.form(http.MethodPost, "/plan-import", form, cookie))
	if w.Code != http.StatusBadRequest || strings.TrimSpace(w.Body.String()) != "That project or task no longer exists." {
		t.Fatalf("%d %q", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "sql:") {
		t.Fatal("SQL error text reached the browser")
	}
}

func TestTrainingSyncShowsCuratedTextOnly(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	w := rig.do(rig.form(http.MethodPost, "/health/training/sync", nil, cookie))
	if w.Code != http.StatusServiceUnavailable || strings.TrimSpace(w.Body.String()) != "Hevy is not connected" {
		t.Fatalf("%d %q", w.Code, w.Body.String())
	}
}

func TestAuthEventsAreOneLineEachWithoutSecrets(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	wrong := rig.do(rig.form(http.MethodPost, "/login", url.Values{"password": {"wrong-horse-fictional"}}, nil))
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d", wrong.Code)
	}
	out := rig.do(rig.form(http.MethodPost, "/logout", nil, cookie))
	if out.Code != http.StatusSeeOther || out.Header().Get("Location") != "/login" {
		t.Fatalf("logout: %d %s", out.Code, out.Header().Get("Location"))
	}

	events := []string{}
	for _, line := range rig.lines(t) {
		if line["msg"] != "auth" {
			t.Fatalf("unexpected line: %v", line)
		}
		events = append(events, line["event"].(string))
		for _, key := range []string{"time", "level", "request_id", "ingress", "client"} {
			if _, ok := line[key]; !ok {
				t.Errorf("auth line has no %q: %v", key, line)
			}
		}
	}
	if got := strings.Join(events, ","); got != "login.succeeded,login.failed,logout" {
		t.Fatalf("events = %s", got)
	}
	for _, secret := range []string{"wrong-horse-fictional", testPassword, cookie.Value, sessionCookieName} {
		if strings.Contains(rig.buf.String(), secret) {
			t.Errorf("auth lines contain %q", secret)
		}
	}
}

func TestLogoutOfUnknownSessionLogsNothingAndAnswersTheSame(t *testing.T) {
	rig := newLogRig(t)
	known := rig.signIn(t)
	rig.buf.Reset()
	unknown := &http.Cookie{Name: sessionCookieName, Value: "unknown-token-fictional"}
	for _, c := range []*http.Cookie{unknown, nil} {
		w := rig.do(rig.form(http.MethodPost, "/logout", nil, c))
		if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login" {
			t.Fatalf("logout: %d %s", w.Code, w.Header().Get("Location"))
		}
	}
	if rig.buf.Len() != 0 {
		t.Fatalf("a logout that revoked nothing logged: %s", rig.buf.String())
	}
	_ = known
}

func TestThrottleLogsOnlyTransitions(t *testing.T) {
	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	h, clk := throttledHandler(t)
	failFive(t, h, clk, "203.0.113.10")
	for i := 0; i < 20; i++ {
		post(h, "gyst.zaylolabs.com", "192.0.2.7", "203.0.113.10", wrongPW)
	}
	var lockouts, failures, other int
	var lockout map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var line map[string]any
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			t.Fatal(err)
		}
		switch line["event"] {
		case "lockout":
			lockouts++
			lockout = line
		case "login.failed":
			failures++
		default:
			other++
		}
	}
	// 25 attempts: five real failures and one lockout line. The 20 refused
	// attempts that follow are silent, so an attacker cannot grow the log.
	if lockouts != 1 || failures != 5 || other != 0 {
		t.Fatalf("lockouts=%d failures=%d other=%d: %s", lockouts, failures, other, buf.String())
	}
	if lockout["level"] != "WARN" || lockout["client"] != "203.0.113.0/24" {
		t.Fatalf("lockout line = %v", lockout)
	}
	if strings.Contains(buf.String(), wrongPW) {
		t.Fatal("password in the log")
	}
}

func TestCoachProviderFailureLogsCategoryOnly(t *testing.T) {
	rig := newLogRig(t)
	rig.h.Coach = &fakeCoach{err: errors.New("vendor: prompt echo 99.9 kg sk-secret")}
	cookie := rig.signIn(t)
	rig.buf.Reset()
	w := rig.do(rig.form(http.MethodPost, "/health/coach", url.Values{"q": {"Hi there"}}, cookie))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("status %d", w.Code)
	}
	logged := rig.buf.String()
	if !strings.Contains(logged, `"provider":"health-coach"`) || !strings.Contains(logged, healthcoach.ErrUnavailable.Error()) {
		t.Fatalf("no category line: %s", logged)
	}
	for _, leak := range []string{"99.9", "sk-secret", "prompt echo", "Hi there"} {
		if strings.Contains(logged, leak) {
			t.Errorf("log contains %q", leak)
		}
	}
}

func TestAuthDisabledLineUsesSanitisedRoute(t *testing.T) {
	rig := newLogRig(t)
	rig.h.AuthDisabled = true
	token := strings.Repeat("f0e1d2c3", 4)
	rig.do(httptest.NewRequest(http.MethodGet, "/calendar/feed/"+token, nil))
	if strings.Contains(rig.buf.String(), token) {
		t.Fatalf("feed token reached the log: %s", rig.buf.String())
	}
	if !strings.Contains(rig.buf.String(), `"route":"/calendar/feed/:token"`) {
		t.Fatalf("no sanitised route: %s", rig.buf.String())
	}
}

func TestHealthServiceErrorKeepsTextAndRecordsCause(t *testing.T) {
	rig := newLogRig(t)
	server := RequestLog(rig.h.log(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rig.h.healthServiceError(w, errors.New("database is locked"))
	}))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/health/x", nil))
	if w.Code != 500 || !strings.Contains(w.Body.String(), "health data unavailable") || strings.Contains(w.Body.String(), "locked") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(rig.buf.String(), "database is locked") {
		t.Fatalf("cause not logged: %s", rig.buf.String())
	}
}
