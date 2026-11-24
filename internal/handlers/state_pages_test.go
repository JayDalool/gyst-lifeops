package handlers

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// GYST-188 (spec a4): the styled Not found and Server error pages and the guards around them. Every
// value is fictional. Requests go through RequestLog and Authenticated as the app wires them.

const pageAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"

func (rig *logRig) page(method, path string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(""))
	r.Header.Set("Accept", pageAccept)
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	for k, v := range headers {
		if v == "" {
			r.Header.Del(k)
		} else {
			r.Header.Set(k, v)
		}
	}
	return rig.do(r)
}

func isStatePage(body string) bool { return strings.Contains(body, `class="gyst-statepage"`) }

func TestNotFoundPageForASignedInPageView(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.buf.Reset()
	for _, path := range []string{"/no-such-page", "/projects/999999", "/tasks/999999"} {
		rec := rig.page(http.MethodGet, path, cookie, nil)
		body := rec.Body.String()
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
		if !isStatePage(body) || !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("%s: not the styled page: %q", path, body)
		}
		for _, want := range []string{`<h1 class="gyst-state8-title">Page not found</h1>`, `<title>Not found · GYST</title>`, `href="/today">Go to Today`, `href="/more">All pages`, `href="/archive">Archive`, `<nav aria-label="Primary" class="gyst-tabbar">`} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: lacks %s", path, want)
			}
		}
		if strings.Contains(body, `role="alert"`) || strings.Contains(body, "<main role") {
			t.Errorf("%s: a not-found page is not an alert", path)
		}
		// One page for every missing thing: nothing in it says which path or record was asked for.
		if strings.Contains(body, "999999") || strings.Contains(body, "no-such-page") {
			t.Errorf("%s: the page names what was asked for", path)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: no nosniff", path)
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store on a signed-in page", path, rec.Header().Get("Cache-Control"))
		}
	}
	if len(rig.lines(t)) != 0 {
		t.Errorf("a 404 logged: %s", rig.buf.String())
	}
}

func TestServerErrorPageForASignedInPageView(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.breakTable(t, "tasks")
	rig.buf.Reset()

	rec := rig.page(http.MethodGet, "/tasks?view=open", cookie, nil)
	body := rec.Body.String()
	id := rec.Header().Get("X-Request-Id")
	if rec.Code != http.StatusInternalServerError || !isStatePage(body) {
		t.Fatalf("status %d, styled %v: %s", rec.Code, isStatePage(body), body)
	}
	for _, want := range []string{
		`<div class="gyst-state8" role="alert">`, `<header class="gyst-state-top">`, `<h1 class="gyst-state8-title">Something went wrong</h1>`, `<title>Error · GYST</title>`,
		`GYST could not load this page. Nothing you saved was lost.`, `href="/tasks?view=open">Try again`, `href="/today">Go to Today`,
		`quote reference <span>` + id + `</span>.`, `<nav aria-label="Primary" class="gyst-tabbar">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("lacks %s", want)
		}
	}
	if !requestIDPattern.MatchString(id) {
		t.Errorf("X-Request-Id = %q", id)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store on a signed-in page", rec.Header().Get("Cache-Control"))
	}
	lower := strings.ToLower(body)
	for _, leak := range []string{"no such table", "sql", "tasks_gone"} {
		if strings.Contains(lower, leak) {
			t.Errorf("the page shows %q", leak)
		}
	}
	lines := rig.lines(t)
	if len(lines) != 1 || lines[0]["level"] != "ERROR" || lines[0]["request_id"] != id || lines[0]["status"] != float64(500) {
		t.Errorf("want exactly one ERROR line for %s: %s", id, rig.buf.String())
	}
}

func TestServerErrorPageForAFormPostOffersGoBack(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.breakTable(t, "tasks")
	rig.buf.Reset()
	form := url.Values{"title": {"Fictional task"}}
	post := func(referer string) *httptest.ResponseRecorder {
		r := rig.form(http.MethodPost, "/tasks/new", form, cookie)
		r.Header.Set("Accept", pageAccept)
		r.Header.Set("Sec-Fetch-Mode", "navigate")
		r.Header.Set("Sec-Fetch-Dest", "document")
		if referer != "" {
			r.Header.Set("Referer", referer)
		}
		return rig.do(r)
	}
	rec := post("http://example.com/tasks/new?x=1")
	body := rec.Body.String()
	if rec.Code != http.StatusInternalServerError || !isStatePage(body) {
		t.Fatalf("status %d: %s", rec.Code, body)
	}
	if !strings.Contains(body, `href="/tasks/new?x=1">Go back`) || strings.Contains(body, "Try again") {
		t.Errorf("a failed post must offer Go back to the Referer's path and never Try again: %s", body)
	}
	if !strings.Contains(body, "Your last change may not be saved.") {
		t.Error("the post wording is missing")
	}
	// A hostile or missing Referer sends the person to /today.
	for _, ref := range []string{"", "//evil.example/x", "/\\evil.example", "javascript:alert(1)", "http://evil.example/tasks", "https://example.com.evil.example/x"} {
		got := post(ref).Body.String()
		if !strings.Contains(got, `href="/today">Go back`) {
			t.Errorf("Referer %q: Go back is not /today: %s", ref, got)
		}
	}
}

// markPageView flags only a real page view; every fetch caller, the API and non-HTML requests keep text.
func TestPageViewPredicate(t *testing.T) {
	const html = "text/html,application/xhtml+xml"
	cases := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
		want    bool
	}{
		{"browser navigation", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}, true},
		{"no Sec-Fetch headers, Accept text/html", "GET", "/today", map[string]string{"Accept": html}, true},
		{"HEAD", "HEAD", "/today", map[string]string{"Accept": html}, true},
		{"document form post", "POST", "/tasks/new", map[string]string{"Accept": html, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}, true},
		{"Accept html with q=0.5", "GET", "/today", map[string]string{"Accept": "application/json, text/html;q=0.5"}, true},
		{"Accept html refused with q=0", "GET", "/today", map[string]string{"Accept": "application/json, text/html;q=0"}, false},
		{"Accept html q=0.0 mid-list", "GET", "/today", map[string]string{"Accept": "text/html;q=0.0, */*"}, false},
		{"Accept near-miss text/htmlx", "GET", "/today", map[string]string{"Accept": "text/htmlx"}, false},
		{"Accept uppercase with spaces", "GET", "/today", map[string]string{"Accept": "application/json ,  Text/HTML ; q=1"}, true},
		// GYST-188: the service worker's network-only re-fetch of a navigation, fetch(request, {cache: "no-store"}),
		// as Chrome sent it with the worker active: same-origin, empty, Cache-Control and Pragma no-cache.
		{"worker re-fetch (same-origin, empty, no-cache)", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "same-origin", "Sec-Fetch-Dest": "empty", "Sec-Fetch-Site": "same-origin", "Cache-Control": "no-cache", "Pragma": "no-cache"}, true},
		{"same-origin, empty, without no-cache (a script's own)", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "same-origin", "Sec-Fetch-Dest": "empty", "Sec-Fetch-Site": "same-origin"}, false},
		{"default script fetch (cors, empty, no-cache)", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "cors", "Sec-Fetch-Dest": "empty", "Cache-Control": "no-cache"}, false},
		{"worker-like but dest document", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "same-origin", "Sec-Fetch-Dest": "document", "Cache-Control": "no-cache"}, false},
		{"worker-like but Accept */*", "GET", "/today", map[string]string{"Accept": "*/*", "Sec-Fetch-Mode": "same-origin", "Sec-Fetch-Dest": "empty", "Cache-Control": "no-cache"}, false},
		{"navigate with dest empty (not what any client sends now)", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "empty"}, false},
		{"navigate with dest iframe", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "iframe"}, false},
		{"navigate with dest script", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "script"}, false},
		{"cors", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "cors"}, false},
		{"no-cors", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "no-cors"}, false},
		{"same-origin", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Mode": "same-origin"}, false},
		{"dest empty (fetch)", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Dest": "empty"}, false},
		{"dest iframe", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Dest": "iframe"}, false},
		{"dest script", "GET", "/today", map[string]string{"Accept": html, "Sec-Fetch-Dest": "script"}, false},
		{"Accept */*", "GET", "/today", map[string]string{"Accept": "*/*"}, false},
		{"Accept json", "GET", "/today", map[string]string{"Accept": "application/json"}, false},
		{"no Accept", "GET", "/today", nil, false},
		{"api path", "GET", "/api/foods", map[string]string{"Accept": html}, false},
		{"api root", "GET", "/api", map[string]string{"Accept": html}, false},
		{"PUT", "PUT", "/today", map[string]string{"Accept": html}, false},
		{"DELETE", "DELETE", "/today", map[string]string{"Accept": html}, false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, nil)
		for k, v := range c.headers {
			r.Header.Set(k, v)
		}
		rec := &respRecorder{ResponseWriter: httptest.NewRecorder()}
		markPageView(rec, r, normalizePath(r.URL.Path))
		if got := rec.page != nil; got != c.want {
			t.Errorf("%s: flagged = %v, want %v", c.name, got, c.want)
		}
	}
	// Without the recorder there is nothing to flag, and nothing may panic.
	markPageView(httptest.NewRecorder(), httptest.NewRequest("GET", "/today", nil), "/today")
}

// No request can set the flag: only Authenticated() does, after the session check. A visitor with no
// session gets today's redirect or text for every header combination.
func TestNoSessionNeverGetsAStatePage(t *testing.T) {
	rig := newLogRig(t)
	for _, path := range []string{"/today", "/no-such-page", "/projects/999999", "/tasks", "/galaxy"} {
		rec := rig.page(http.MethodGet, path, nil, map[string]string{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"})
		if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/login") {
			t.Errorf("GET %s with no session: %d %s", path, rec.Code, rec.Header().Get("Location"))
		}
		body := rec.Body.String()
		for _, shell := range []string{"gyst-tabbar", "gyst-primary", "gyst-command", "gyst-statepage"} {
			if strings.Contains(body, shell) {
				t.Errorf("GET %s with no session leaks %s", path, shell)
			}
		}
	}
	rec := rig.page(http.MethodPost, "/tasks/new", nil, map[string]string{"Sec-Fetch-Mode": "navigate"})
	if rec.Code != http.StatusUnauthorized || rec.Body.String() != "session expired\n" {
		t.Errorf("POST with no session: %d %q", rec.Code, rec.Body.String())
	}
	// A forged cookie is no session either.
	rec = rig.page(http.MethodGet, "/no-such-page", &http.Cookie{Name: sessionCookieName, Value: "forged-fictional"}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("forged cookie: %d", rec.Code)
	}
}

// Everything that is not a signed-in browser page view keeps today's exact bytes.
func TestNonPageViewsKeepTheTextAnswers(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.breakTable(t, "tasks")
	rig.buf.Reset()
	cases := []struct {
		name    string
		path    string
		headers map[string]string
		code    int
		body    func(id string) string
	}{
		{"404, no Accept", "/no-such-page", map[string]string{"Accept": ""}, 404, func(string) string { return "404 page not found\n" }},
		{"404, Accept */*", "/no-such-page", map[string]string{"Accept": "*/*"}, 404, func(string) string { return "404 page not found\n" }},
		{"404, application/json", "/no-such-page", map[string]string{"Accept": "application/json"}, 404, func(string) string { return "404 page not found\n" }},
		{"404, Sec-Fetch-Mode cors", "/no-such-page", map[string]string{"Sec-Fetch-Mode": "cors"}, 404, func(string) string { return "404 page not found\n" }},
		{"404, /api/", "/api/no-such", nil, 404, func(string) string { return "404 page not found\n" }},
		{"404 record, Accept */*", "/projects/999999", map[string]string{"Accept": "*/*"}, 404, func(string) string { return "not found\n" }},
		{"500, no Accept", "/tasks", map[string]string{"Accept": ""}, 500, func(id string) string { return "Something went wrong. Reference: " + id + ".\n" }},
		{"500, Accept */*", "/tasks", map[string]string{"Accept": "*/*"}, 500, func(id string) string { return "Something went wrong. Reference: " + id + ".\n" }},
		{"500, Sec-Fetch-Dest empty", "/tasks", map[string]string{"Sec-Fetch-Dest": "empty"}, 500, func(id string) string { return "Something went wrong. Reference: " + id + ".\n" }},
		{"500, /api/", "/api/tasks", nil, -1, nil},
	}
	for _, c := range cases {
		rec := rig.page(http.MethodGet, c.path, cookie, c.headers)
		if c.code == -1 {
			if isStatePage(rec.Body.String()) {
				t.Errorf("%s: a state page for /api/", c.name)
			}
			continue
		}
		if rec.Code != c.code || rec.Body.String() != c.body(rec.Header().Get("X-Request-Id")) {
			t.Errorf("%s: %d %q", c.name, rec.Code, rec.Body.String())
		}
		if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
			t.Errorf("%s: Content-Type %q", c.name, rec.Header().Get("Content-Type"))
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "" {
			t.Errorf("%s: the plain-text answer gained Cache-Control %q", c.name, cc)
		}
	}
}

// The exempt routes keep their text answers whatever the request says, with a session and with auth off.
func TestExemptRoutesKeepTheirTextAnswers(t *testing.T) {
	for _, authOff := range []bool{false, true} {
		rig := newLogRig(t)
		rig.h.AuthDisabled = authOff
		cookie := rig.signIn(t)
		rig.h.TemplateDir = "/nonexistent-fictional-dir" // /login fails to render: serverError
		nav := map[string]string{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
		cases := []struct {
			method, path, body string
			code               int
		}{
			{"GET", "/login", "", 500},
			{"GET", "/logout", "method not allowed\n", 405},
			{"GET", "/calendar/feed/bad-token-fictional", "404 page not found\n", 404},
		}
		for _, c := range cases {
			for _, ck := range []*http.Cookie{nil, cookie} {
				// A valid session on /login is redirected away before any page is drawn (auth.go), unchanged.
				if c.path == "/login" && ck != nil && !authOff {
					continue
				}
				rec := rig.page(c.method, c.path, ck, nav)
				if rec.Code != c.code {
					t.Errorf("authOff=%v %s %s: status %d, want %d", authOff, c.method, c.path, rec.Code, c.code)
				}
				if isStatePage(rec.Body.String()) {
					t.Errorf("authOff=%v %s %s: a state page on an exempt route", authOff, c.method, c.path)
				}
				if c.body != "" && rec.Body.String() != c.body {
					t.Errorf("authOff=%v %s %s: body %q, want %q", authOff, c.method, c.path, rec.Body.String(), c.body)
				}
				if c.code == 500 && !strings.HasPrefix(rec.Body.String(), "Something went wrong. Reference: ") {
					t.Errorf("authOff=%v %s: body %q", authOff, c.path, rec.Body.String())
				}
			}
		}
		for _, path := range []string{"/api/health/sync", "/api/health", "/api/automation/events", "/api/no-such"} {
			rec := rig.page("GET", path, nil, nav)
			if isStatePage(rec.Body.String()) {
				t.Errorf("authOff=%v %s: a state page on an API route", authOff, path)
			}
		}
	}
}

// A handler called without the recorder, after a partial write, or after it set a download header
// answers with today's exact text.
func TestHelpersFallBackToTodaysText(t *testing.T) {
	err := errors.New("boom fictional")

	rec := httptest.NewRecorder()
	serverError(rec, err)
	if rec.Code != 500 || rec.Body.String() != "Something went wrong.\n" || rec.Header().Get("Cache-Control") != "" {
		t.Errorf("no recorder: %d %q %v", rec.Code, rec.Body.String(), rec.Header())
	}
	rec = httptest.NewRecorder()
	notFound(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != 404 || rec.Body.String() != "404 page not found\n" {
		t.Errorf("notFound, no recorder: %d %q", rec.Code, rec.Body.String())
	}

	flagged := func() (*respRecorder, *httptest.ResponseRecorder) {
		under := httptest.NewRecorder()
		return &respRecorder{ResponseWriter: under, status: 200, page: &pageView{retry: "/x"}}, under
	}
	rw, under := flagged()
	_, _ = rw.Write([]byte("partial"))
	serverError(rw, err)
	if under.Body.String() != "partial"+"Something went wrong.\n" || under.Code != 200 {
		t.Errorf("after a partial write: %d %q", under.Code, under.Body.String())
	}
	rw, under = flagged()
	rw.Header().Set("Content-Disposition", `attachment; filename="export.csv"`)
	serverError(rw, err)
	if isStatePage(under.Body.String()) || under.Body.String() != "Something went wrong.\n" || under.Header().Get("Content-Disposition") == "" {
		t.Errorf("with a download header: %q", under.Body.String())
	}
	rw, under = flagged()
	rw.Header().Set("Content-Disposition", "attachment")
	notFound(rw, httptest.NewRequest("GET", "/x", nil))
	if under.Body.String() != "404 page not found\n" {
		t.Errorf("notFound with a download header: %q", under.Body.String())
	}
	// With the flag the same calls answer with the page.
	rw, under = flagged()
	serverError(rw, err)
	if under.Code != 500 || !isStatePage(under.Body.String()) {
		t.Errorf("flagged serverError: %d", under.Code)
	}
	if rw.err == nil {
		t.Error("serverError must still hand the error to the log line")
	}
	rw, under = flagged()
	notFoundOrError(rw, errors.New("other"))
	if under.Code != 500 {
		t.Errorf("notFoundOrError(other): %d", under.Code)
	}
}

// The page carries no text from the error and escapes what comes from the request.
func TestStatePageEscapesRequestData(t *testing.T) {
	r := httptest.NewRequest("GET", `/tasks?x="><script>alert(1)</script>&y='`, nil)
	r.Header.Set("Accept", "text/html")
	r.Header.Set("Referer", `http://example.com/a"><img src=x onerror=alert(1)>`)
	under := httptest.NewRecorder()
	under.Header().Set(requestIDHeader, `<b>id</b>`)
	rw := &respRecorder{ResponseWriter: under, status: 200}
	markPageView(rw, r, "/tasks")
	serverError(rw, errors.New("secret detail fictional"))
	body := under.Body.String()
	for _, bad := range []string{"<script>alert", "<img src=x", "<b>id</b>", "secret detail"} {
		if strings.Contains(body, bad) {
			t.Errorf("the page contains %q", bad)
		}
	}
	if !strings.Contains(body, "&lt;b&gt;id&lt;/b&gt;") {
		t.Error("the reference is not escaped")
	}
}

// The session renewal cookie and the security headers survive on the page: the helpers add no header
// and drop none but Content-Length.
func TestStatePageKeepsCookieAndHeaders(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	rig.h.Service.Clock.Now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	r := httptest.NewRequest(http.MethodGet, "/no-such-page", nil)
	r.Header.Set("Accept", pageAccept)
	r.AddCookie(cookie)
	rec := httptest.NewRecorder()
	SecurityHeaders(rig.server).ServeHTTP(rec, r)
	if rec.Code != 404 || !isStatePage(rec.Body.String()) {
		t.Fatalf("status %d", rec.Code)
	}
	renewed := false
	for _, c := range rec.Result().Cookies() {
		renewed = renewed || (c.Name == sessionCookieName && c.Value == cookie.Value)
	}
	if !renewed {
		t.Error("the renewed session cookie is gone")
	}
	for _, h := range []string{"Content-Security-Policy-Report-Only", "X-Frame-Options", "X-Content-Type-Options", "Referrer-Policy", "Permissions-Policy", "X-Request-Id"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("header %s is gone", h)
		}
	}
	if rec.Header().Get("Content-Length") != "" && rec.Header().Get("Content-Length") != itoa(rec.Body.Len()) {
		t.Error("a stale Content-Length")
	}
}

var tabbarRE = regexp.MustCompile(`(?s)<nav aria-label="Primary" class="gyst-tabbar">.*?</nav>`)

// The page cannot render the shell, so its tab bar is a literal; this fails if the shell's four tabs move.
func TestStatePageTabBarMatchesShell(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	shell := rig.page("GET", "/today", cookie, nil)
	want := tabbarRE.FindString(shell.Body.String())
	want = strings.ReplaceAll(want, ` aria-current="page"`, "")
	if want == "" {
		t.Fatal("no tab bar in the shell's page")
	}
	got := tabbarRE.FindString(rig.page("GET", "/no-such-page", cookie, nil).Body.String())
	if got != want {
		t.Errorf("tab bar drifted from ui_shell.html:\ngot:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "aria-current") || strings.Contains(got, "gyst-plus") {
		t.Error("the state page's tab bar has no current tab and no +")
	}
}

// Binding limit: the helpers cannot reach the database. state_pages.go imports the standard library only,
// uses no template function, and renders with no Handler.
func TestStatePageHelpersAreDBFree(t *testing.T) {
	src, err := os.ReadFile("state_pages.go")
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "state_pages.go", src, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(strings.Split(path, "/")[0], ".") || strings.HasPrefix(path, "lifeops-tracker") {
			t.Errorf("state_pages.go imports %s: standard library only", path)
		}
	}
	if strings.Contains(string(src), ".Funcs(") {
		t.Error("the state page template must have no template functions")
	}
	full, err := parser.ParseFile(token.NewFileSet(), "state_pages.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(full, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			switch id.Name {
			case "Handler", "Service", "Repo", "Repository", "DB", "sql":
				t.Errorf("state_pages.go names %s: it must not reach a Handler, Service or database", id.Name)
			}
		}
		return true
	})
	// No database: the page renders from the recorder alone.
	under := httptest.NewRecorder()
	rw := &respRecorder{ResponseWriter: under, page: &pageView{}}
	if !writeStatePage(rw, 404, true) || !isStatePage(under.Body.String()) {
		t.Error("the page did not render with no Handler")
	}
}

// 63 calls were replaced, none left.
func TestNoPlainNotFoundCallsLeft(t *testing.T) {
	files, _ := os.ReadDir(".")
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") || f.Name() == "state_pages.go" {
			continue
		}
		b, _ := os.ReadFile(f.Name())
		if strings.Contains(string(b), "http.NotFound(") {
			t.Errorf("%s still calls http.NotFound", f.Name())
		}
	}
}

// The service worker's offline page is one constant with no request data and nothing that runs or loads.
func TestOfflinePageIsInert(t *testing.T) {
	worker := readStatic(t, "service-worker.js")
	m := regexp.MustCompile(`(?m)^const OFFLINE = '(.*)';$`).FindStringSubmatch(worker)
	if m == nil {
		t.Fatal("OFFLINE is not one single-quoted constant")
	}
	page := m[1]
	for _, bad := range []string{"<script", "src=", "<use", "<link", "<img", "<iframe", "@import", "url(", "http:", "https:", "on", "${", "`"} {
		if bad == "on" {
			if regexp.MustCompile(`(?i)\son[a-z]+=`).MatchString(page) {
				t.Error("OFFLINE has an on*= attribute")
			}
			continue
		}
		if strings.Contains(page, bad) {
			t.Errorf("OFFLINE contains %q", bad)
		}
	}
	for _, href := range regexp.MustCompile(`href="([^"]*)"`).FindAllStringSubmatch(page, -1) {
		if href[1] != "" {
			t.Errorf("OFFLINE has href=%q: only the empty one (Try again) is allowed", href[1])
		}
	}
	for _, want := range []string{`<h1>You are offline</h1>`, `<a href="">Try again</a>`, `prefers-color-scheme:dark`, `<svg `} {
		if !strings.Contains(page, want) {
			t.Errorf("OFFLINE lacks %s", want)
		}
	}
}

// Each static file has one ?v= string wherever it is named: layout, the sign-in page, the state page,
// the icon helper and the service worker's list. The two new files carry signin-1 and stay out of the
// worker's list (nothing about them is needed offline, and pwa_assets_test would refuse them).
func TestSignInAndStatePageAssetVersions(t *testing.T) {
	rig := newLogRig(t)
	cookie := rig.signIn(t)
	state := rig.page("GET", "/no-such-page", cookie, nil).Body.String()
	_, assets := shellAssets(t)
	sources := map[string]string{
		"layout.html": readTemplate(t, "layout.html"), "login.html": readTemplate(t, "login.html"),
		"state page": state, "ui_icons.html": readTemplate(t, "ui_icons.html"), "SHELL": strings.Join(assets, " "),
	}
	seen := map[string]string{}
	for name, text := range sources {
		for _, m := range regexp.MustCompile(`/static/[A-Za-z0-9_./-]+\?v=[A-Za-z0-9_.-]+`).FindAllString(text, -1) {
			path, _, _ := strings.Cut(m, "?")
			if prev, ok := seen[path]; ok && prev != m {
				t.Errorf("%s names %s but another file names %s", name, m, prev)
			}
			seen[path] = m
		}
	}
	for path, v := range map[string]string{
		"/static/css/ui-signin.css": "signin-1", "/static/js/signin.js": "signin-1", "/static/icons.svg": "r7-6",
	} {
		if seen[path] != path+"?v="+v {
			t.Errorf("%s is %q, want ?v=%s", path, seen[path], v)
		}
	}
	for _, a := range assets {
		if strings.Contains(a, "ui-signin.css") || strings.Contains(a, "signin.js") {
			t.Errorf("SHELL lists %s", a)
		}
	}
	for _, want := range []string{"/static/css/ui-signin.css?v=signin-1", "/static/css/ui-nav.css?v=nav-8", "/static/js/theme.js?v=appearance-2"} {
		if !strings.Contains(state, want) {
			t.Errorf("the state page does not load %s", want)
		}
	}
}

// Every symbol the state page names exists in the sprite, and the sign-in page's too (check-icons.cjs checks
// the templates; this covers the page that lives in Go).
func TestStatePageIconsExistInTheSprite(t *testing.T) {
	have := map[string]bool{}
	for _, id := range spriteIDs(t) {
		have[id] = true
	}
	for _, kind := range []bool{true, false} {
		under := httptest.NewRecorder()
		rw := &respRecorder{ResponseWriter: under, page: &pageView{}}
		writeStatePage(rw, 404, kind)
		found := regexp.MustCompile(`icons\.svg\?v=r7-6#i-([a-z0-9-]+)`).FindAllStringSubmatch(under.Body.String(), -1)
		if len(found) < 5 {
			t.Fatalf("only %d sprite references", len(found))
		}
		for _, m := range found {
			if !have[m[1]] {
				t.Errorf("the state page names i-%s, which the sprite lacks", m[1])
			}
		}
	}
}
