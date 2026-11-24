package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// GYST-188 (spec a4): styled Not found and Server error pages for a signed-in browser page view.
//
// Binding limits, from the technical review of GYST-173 and GYST-188:
//   - The helpers are DB-free. This file imports the standard library only, parses one template at
//     start-up with no Funcs, and takes no Handler, Service or query, like refused.go. A test pins that.
//   - Nothing wraps or rewrites a response. A helper writes the page only when the middleware flagged the
//     request, the header is not yet written and no download header is set; in every other case it writes
//     today's exact text. So exempt routes, /api/, non-HTML callers and handlers called without the
//     recorder keep their bytes.
//   - The flag exists only after Authenticated() has checked the session and the same-origin guard, so a
//     visitor with no session never gets a page and one 404 page serves every missing path (no oracle).

// pageView is what markPageView keeps on the response recorder for the helpers below.
type pageView struct {
	post  bool
	retry string // the request's own path and query, through safeNextPath
	back  string // the same-host Referer path, through safeNextPath
}

// markPageView flags the request on the recorder as a browser page view. It is called only by
// Authenticated(), after CheckSession and the same-origin guard, and never for an exempt path. Fetch and
// XHR callers (Accept */* or application/json, or any Sec-Fetch-Mode other than navigate, or a
// Sec-Fetch-Dest of "empty") are never flagged: galaxy.js, launcher.js, app.js, charts.js and
// food-camera.js parse the plain text they get today.
func markPageView(w http.ResponseWriter, r *http.Request, path string) {
	rec, ok := w.(*respRecorder)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost:
	default:
		return
	}
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") {
		return
	}
	if mode := r.Header.Get("Sec-Fetch-Mode"); mode != "" && mode != "navigate" {
		return
	}
	if dest := r.Header.Get("Sec-Fetch-Dest"); dest != "" && dest != "document" {
		return
	}
	rec.page = &pageView{
		post:  r.Method == http.MethodPost,
		retry: safeNextPath(r.URL.RequestURI()),
		back:  safeNextPath(sameHostReferer(r, "")),
	}
}

type statePageData struct {
	NotFound  bool
	Post      bool
	Reference string
	Retry     string
	Back      string
}

// Error is announced by role="alert" on the content block, not on <main>: spec a4 section 10 puts it on <main>, but axe
// refuses that (aria-allowed-role, and <main> stops being a landmark), and an alert is still announced from the block.
//
// The tab bar below is a literal copy of ui_shell.html's nav.gyst-tabbar without aria-current: these
// pages cannot render the shell (no layout, no data). TestStatePageTabBarMatchesShell fails when they drift.
var statePage = template.Must(template.New("state").Parse(`<!doctype html>
<html lang="en" data-theme="light" data-gyst-theme="light">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<title>{{if .NotFound}}Not found{{else}}Error{{end}} · GYST</title>
<link rel="icon" href="/favicon.ico">
<link rel="icon" type="image/svg+xml" href="/static/favicon.svg">
<meta name="theme-color" content="#F1EFF9" data-light="#F1EFF9" data-dark="#09080F">
<meta name="robots" content="noindex, nofollow">
<script src="/static/js/theme.js?v=appearance-2"></script>
<link rel="preload" href="/static/fonts/Geist-Variable.woff2?v=1.3.0" as="font" type="font/woff2" crossorigin>
<link rel="stylesheet" href="/static/css/ui-tokens.css?v=aurora-1">
<link rel="stylesheet" href="/static/css/ui-accents.css?v=aurora-1">
<link rel="stylesheet" href="/static/css/ui-components.css?v=icons-1.aurora-1">
<link rel="stylesheet" href="/static/css/ui-nav.css?v=nav-8">
<link rel="stylesheet" href="/static/css/ui-signin.css?v=signin-1">
</head>
<body class="gyst-statepage">
<header class="gyst-state-top"><a class="gyst-brand" href="/today">GYST</a></header>
<main class="gyst-state-main">
<p class="gyst-state-kicker">{{if .NotFound}}404 · Not found{{else}}Error{{end}}</p>
<div class="gyst-state-card"><div class="gyst-state8"{{if not .NotFound}} role="alert"{{end}}>
<span class="gyst-state8-icon"{{if not .NotFound}} data-state="crit"{{end}}><svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=r7-6#i-{{if .NotFound}}search{{else}}alert{{end}}"/></svg></span>
{{if .NotFound}}<h1 class="gyst-state8-title">Page not found</h1>
<p class="gyst-state8-text">This link points to something that is not here. It may have been deleted or moved to the Archive.</p>
<div class="gyst-state8-acts"><a class="gyst-state-btn" href="/today">Go to Today</a><a class="gyst-state-btn is-quiet" href="/more">All pages</a><a class="gyst-state-btn is-quiet" href="/archive">Archive</a></div>
{{else}}<h1 class="gyst-state8-title">Something went wrong</h1>
{{if .Post}}<p class="gyst-state8-text">GYST could not finish that. Your last change may not be saved. Go back and check before you try again.</p>
<div class="gyst-state8-acts"><a class="gyst-state-btn" href="{{.Back}}">Go back</a><a class="gyst-state-btn is-quiet" href="/today">Go to Today</a></div>
{{else}}<p class="gyst-state8-text">GYST could not load this page. Nothing you saved was lost.</p>
<div class="gyst-state8-acts"><a class="gyst-state-btn" href="{{.Retry}}">Try again</a><a class="gyst-state-btn is-quiet" href="/today">Go to Today</a></div>
{{end}}{{if .Reference}}<p class="gyst-state8-ref">If it keeps happening, quote reference <span>{{.Reference}}</span>.</p>{{end}}{{end}}
</div></div>
</main>
<div class="gyst-scope gyst-tabs">
  <nav aria-label="Primary" class="gyst-tabbar">
    <a href="/today" data-sec="plan"><svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=r7-6#i-home"/></svg><span>Today</span></a>
    <a href="/tasks" data-sec="plan"><svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=r7-6#i-tasks"/></svg><span>Tasks</span></a>
    <a href="/health" data-sec="health"><svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=r7-6#i-heart"/></svg><span>Health</span></a>
    <a href="/more"><svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=r7-6#i-all"/></svg><span>All</span></a>
  </nav>
</div>
</body>
</html>
`))

// writeStatePage answers with the styled page and reports true, or reports false having written nothing,
// so the caller falls back to today's exact text. The page is rendered into a buffer first, so a template
// failure can never leave a half-written body.
func writeStatePage(w http.ResponseWriter, status int, notFound bool) bool {
	rec, ok := w.(*respRecorder)
	if !ok || rec.page == nil || rec.wroteHeader {
		return false
	}
	// A handler that began a download (courses.go, handlers.go export) and then failed keeps today's text:
	// an HTML body must never arrive under an attachment header.
	if w.Header().Get("Content-Disposition") != "" {
		return false
	}
	data := statePageData{NotFound: notFound, Post: rec.page.post, Reference: w.Header().Get(requestIDHeader), Retry: rec.page.retry, Back: rec.page.back}
	if data.Retry == "" {
		data.Retry = "/today"
	}
	if data.Back == "" {
		data.Back = "/today"
	}
	var buf bytes.Buffer
	if err := statePage.Execute(&buf, data); err != nil {
		log.Printf("render state page: %v", err)
		return false
	}
	// Only what http.Error also changes: the length no longer matches, and the type is HTML.
	w.Header().Del("Content-Length")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
	return true
}

// notFound replaces http.NotFound: the page for a signed-in page view, "404 page not found" otherwise.
func notFound(w http.ResponseWriter, r *http.Request) {
	if writeStatePage(w, http.StatusNotFound, true) {
		return
	}
	http.NotFound(w, r)
}

// recordNotFound is notFound for a missing record, whose plain text has always been "not found".
func recordNotFound(w http.ResponseWriter) {
	if writeStatePage(w, http.StatusNotFound, true) {
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}
