package handlers

import (
	"net/http"
	"net/url"
	"strings"
)

// sessionCookieName is the browser's half of a session. Only its SHA-256 hash
// is ever stored server-side.
const sessionCookieName = "gyst_session"

// unsafeMethod reports whether a method can change state, and therefore needs
// the cross-site guard. GET and HEAD are excluded because they must not change
// anything; if one of them does, the guard is not the bug to fix.
func unsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// requireSameOrigin rejects cross-site state-changing requests, and reports
// whether the caller should stop. Two independent signals, because each covers
// a gap in the other: Sec-Fetch-Site catches the ordinary cross-site form post,
// and the Origin host comparison catches a same-site request from a sibling
// host, which Sec-Fetch-Site reports as "same-site" rather than "cross-site".
//
// A request carrying neither header is allowed: that is a non-browser client
// such as the Apple Health Shortcut, which cannot be driven by a hostile page
// and authenticates with a bearer token rather than a cookie.
func requireSameOrigin(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "cross-site submission rejected", http.StatusForbidden)
		return true
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host != r.Host || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			http.Error(w, "cross-site submission rejected", http.StatusForbidden)
			return true
		}
	}
	return false
}

// authExemptPath reports whether a path is reachable without a session.
//
// Every entry here is either a credential-free page needed to obtain a session,
// or a route that carries its own credential. Nothing is exempt merely for
// convenience, and the list is asserted entry by entry in the tests so it
// cannot grow silently.
func authExemptPath(path string) bool {
	switch path {
	case "/login", "/logout":
		// /login is how a session is obtained. /logout clears the cookie and
		// must work even when the session it refers to has already lapsed.
		return true
	}
	// The Health API authenticates with its own bearer token and is called by
	// the Apple Health Shortcut, which has no cookie jar.
	if path == "/api/health" || strings.HasPrefix(path, "/api/health/") {
		return true
	}
	// n8n reports workflow events to exactly this path with its own bearer
	// token (GYST_N8N_EVENT_TOKEN); n8n has no cookie jar. An exact match, not
	// a prefix, so nothing under /api/automation inherits the exemption.
	if path == "/api/automation/events" {
		return true
	}
	// Tokenized calendar feeds carry their own token and are fetched by
	// external calendar clients - Apple Calendar, Google Calendar - which never
	// send a session cookie. Putting these behind the session would silently
	// break every existing subscription.
	if strings.HasPrefix(path, "/calendar/feed/") {
		return true
	}
	return false
}

// normalizePath gives a stable leading-slash form for matching, so "/login/"
// and "//login" cannot slip past the exempt list as unmatched paths.
func normalizePath(raw string) string {
	trimmed := strings.Trim(raw, "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed
}

// sessionToken reads the raw session token from the request cookie.
func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// Authenticated wraps the web handler so that every route requires a session
// except the ones authExemptPath allows, and every state-changing request from
// a browser must be same-origin.
func (h *Handler) Authenticated() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := normalizePath(r.URL.Path)

		if h.AuthDisabled {
			h.log().Warn("authentication disabled (GYST_AUTH_DISABLED); serving unauthenticated", "event", "auth.disabled", "method", r.Method, "route", routeLabel(path), "request_id", requestIDFrom(r.Context()))
			h.ServeHTTP(w, r)
			return
		}

		if authExemptPath(path) {
			h.ServeHTTP(w, r)
			return
		}

		token := sessionToken(r)
		renewedUntil, err := h.Service.CheckSession(token)
		if err != nil {
			h.denyUnauthenticated(w, r, path)
			return
		}
		if !renewedUntil.IsZero() {
			// Keep the cookie's expiry in step with the session it names.
			h.setSessionCookie(w, r, token, renewedUntil)
		}

		// Only cookie-authenticated routes need the cross-site guard; the
		// exempt routes above use bearer or path tokens, which a hostile page
		// cannot make the browser attach.
		if unsafeMethod(r.Method) && requireSameOrigin(w, r) {
			return
		}

		h.ServeHTTP(w, r)
	})
}

// denyUnauthenticated answers a request that has no valid session. A browser
// navigation is sent to the login page; anything programmatic gets a 401 it can
// act on rather than an HTML page it would have to parse.
func (h *Handler) denyUnauthenticated(w http.ResponseWriter, r *http.Request, path string) {
	// Never let an authentication decision be reused from a cache.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Cookie")

	if strings.HasPrefix(path, "/api/") || r.Header.Get("Sec-Fetch-Mode") == "cors" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		// A state-changing request cannot be meaningfully replayed after
		// login, so say so rather than redirecting to a page that would drop it.
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}

	target := "/login"
	if next := safeNextPath(r.URL.RequestURI()); next != "" && next != "/login" {
		target += "?next=" + url.QueryEscape(next)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// safeNextPath keeps a post-login redirect on this site. It accepts only a
// rooted path: anything scheme-relative ("//evil.example") or absolute would
// turn the login page into an open redirect.
func safeNextPath(candidate string) string {
	if !isLocalPath(candidate) {
		return ""
	}
	return candidate
}

// contentSecurityPolicy is the policy sent on every response. script-src is
// 'self' only: no inline script, no eval, no nonce. style-src keeps
// 'unsafe-inline' because templates use style attributes (progress widths,
// SVG stop-color, route positions); CSSOM writes from JS are not affected.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; connect-src 'self'; font-src 'self'; worker-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// cspHeader is Report-Only for the first deploy. Enforcing is a one-line
// change to "Content-Security-Policy". Report-Only never enforces
// frame-ancestors, so X-Frame-Options: DENY carries clickjacking protection
// until then.
const cspHeader = "Content-Security-Policy-Report-Only"

// SecurityHeaders sets the browser security headers before the wrapped handler
// runs, so errors, redirects and http.Error responses carry them too.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set(cspHeader, contentSecurityPolicy)
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		next.ServeHTTP(w, r)
	})
}
