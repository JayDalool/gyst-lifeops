package handlers

import (
	"errors"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	"lifeops-tracker/internal/authlimit"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// throttledMessage is the one fixed text for every throttle refusal (decision
// D4). It carries no counts, times, Retry-After or hint about the password, so
// a refusal looks identical whether the submitted password was right or wrong.
const throttledMessage = "Too many sign-in attempts. Try again later."

var limiterMissingLogged atomic.Bool

// admitLogin reserves a password check for this request's client. A nil
// attempt means the request was refused and the caller must not check the
// password. It fails closed when no limiter is configured.
func (h *Handler) admitLogin(bucket, key, label string) *authlimit.Attempt {
	if h.loginLimiter == nil {
		if limiterMissingLogged.CompareAndSwap(false, true) {
			h.log().Warn("login throttle", "event", "limiter_missing", "detail", "login refused")
		}
		return nil
	}
	attempt, decision := h.loginLimiter.Admit(bucket, key, label)
	if decision != authlimit.Allow {
		return nil
	}
	return attempt
}

type loginPageData struct {
	Error string
	Next  string
}

// renderLogin draws the login page from its own template rather than the app
// layout. The layout carries navigation, the XP display and a database read for
// player stats - none of which a logged-out visitor should see or trigger.
func (h *Handler) renderLogin(w http.ResponseWriter, status int, data loginPageData) {
	tmpl, err := template.ParseFiles(filepath.Join(h.TemplateDir, "login.html"))
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := tmpl.Execute(w, data); err != nil {
		// The status line is already written, so this can only be logged.
		log.Printf("render login.html: %v", err)
	}
}

// secureCookie reports whether the session cookie may carry the Secure
// attribute. Setting it unconditionally would make login fail silently over
// plain HTTP - the browser accepts the response, drops the cookie, and the next
// request bounces back to the login page with no visible reason. Caddy sets
// X-Forwarded-Proto; a direct TLS connection sets r.TLS.
func secureCookie(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   secureCookie(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookie(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		methodNotAllowed(w)
		return
	}

	next := safeNextPath(r.URL.Query().Get("next"))

	if r.Method == http.MethodGet {
		// Someone already signed in has no business on the login page.
		if !h.AuthDisabled && h.Service.ValidateSession(sessionToken(r)) == nil {
			http.Redirect(w, r, defaultIfEmpty(next, "/"), http.StatusSeeOther)
			return
		}
		h.renderLogin(w, http.StatusOK, loginPageData{Next: next})
		return
	}

	// The login form is a cookie-setting endpoint, so it needs the same
	// cross-site guard as the rest of the mutating surface.
	if requireSameOrigin(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		h.renderLogin(w, http.StatusBadRequest, loginPageData{Error: "That form could not be read. Try again.", Next: next})
		return
	}
	if formNext := safeNextPath(r.PostForm.Get("next")); formNext != "" {
		next = formNext
	}

	if !h.Auth.Configured() {
		// Fail closed and say so plainly: this is a server misconfiguration,
		// not a wrong password, and the operator is the only person who sees it.
		h.renderLogin(w, http.StatusServiceUnavailable, loginPageData{
			Error: "No password is configured. Set GYST_PASSWORD_HASH on the server.",
			Next:  next,
		})
		return
	}

	// Throttle before the password is compared, so a refusal can never depend
	// on whether the password was right. The 403, 400 and 503 paths above are
	// not attempts and are never counted.
	ingress, key, client := loginClient(r)
	attempt := h.admitLogin(ingress, key, client)
	if attempt == nil {
		// At most one row per key per KeyBlock. A refusal writes no auth log
		// line, so an attacker cannot grow the log (TestThrottleLogsOnlyTransitions);
		// the limiter's lockout line records the attack. The response is unchanged.
		if h.throttleAudit.allow(ingress+"|"+key, h.Service.Now()) {
			h.audit(r, repository.AuditEvent{
				ActorRef: clientRef(client), Action: services.AuditActionLoginFailed,
				Outcome: services.AuditOutcomeDenied, Detail: auditDetailThrottle,
			}, ingress)
		}
		h.renderLogin(w, http.StatusTooManyRequests, loginPageData{Error: throttledMessage, Next: next})
		return
	}
	defer attempt.Release() // a server error or panic must not leak the slot

	if err := h.Auth.VerifyPassword(r.PostForm.Get("password")); err != nil {
		if !errors.Is(err, services.ErrInvalidCredentials) {
			serverError(w, err)
			return
		}
		attempt.Fail()
		h.audit(r, repository.AuditEvent{
			ActorRef: clientRef(client), Action: services.AuditActionLoginFailed,
			Outcome: services.AuditOutcomeFailed, Detail: auditDetailWrongPW,
		}, ingress)
		h.authEvent(r, "login.failed", ingress, client, slog.LevelWarn)
		h.renderLogin(w, http.StatusUnauthorized, loginPageData{Error: "Incorrect password.", Next: next})
		return
	}

	token, expires, err := h.Service.CreateSession()
	if err != nil {
		serverError(w, err)
		return
	}
	attempt.Succeed()
	h.audit(r, repository.AuditEvent{
		ActorRef: services.SessionActorRef(token), Action: services.AuditActionLoginSucceeded,
		Outcome: services.AuditOutcomeSuccess,
	}, ingress)
	h.authEvent(r, "login.succeeded", ingress, client, slog.LevelInfo)
	h.setSessionCookie(w, r, token, expires)
	http.Redirect(w, r, defaultIfEmpty(next, "/"), http.StatusSeeOther)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		methodNotAllowed(w)
		return
	}
	if requireSameOrigin(w, r) {
		return
	}
	// Revoking server-side matters more than clearing the cookie: a token that
	// has already been copied elsewhere must stop working too.
	token := sessionToken(r)
	ref := services.SessionActorRef(token) // before revocation
	revoked, err := h.Service.RevokeSession(token)
	if err != nil {
		serverError(w, err)
		return
	}
	if revoked {
		// Logged only when a live session ended, so a stray or forged logout
		// leaves no line; the response below is the same either way.
		ingress, _, client := loginClient(r)
		h.audit(r, repository.AuditEvent{
			ActorRef: ref, Action: services.AuditActionLogout, Outcome: services.AuditOutcomeSuccess,
		}, ingress)
		h.authEvent(r, "logout", ingress, client, slog.LevelInfo)
	}
	h.clearSessionCookie(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// revokeAllSessions is POST /settings/sessions/revoke-all. It deletes every
// session row (public and tailnet hosts share the table), clears this
// browser's cookie and sends it to /login. The log line and the audit row
// carry the action and a count only, never a token or hash.
func (h *Handler) revokeAllSessions(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "POST") {
		return
	}
	ref := services.SessionActorRef(sessionToken(r)) // before revocation
	count, err := h.Service.RevokeAllSessions()
	if err != nil {
		serverError(w, err)
		return
	}
	ingress, _, client := loginClient(r)
	h.audit(r, repository.AuditEvent{
		ActorRef: ref, Action: services.AuditActionSessionRevokeAll,
		Outcome: services.AuditOutcomeSuccess, Detail: "user",
	}, ingress)
	h.log().LogAttrs(r.Context(), slog.LevelInfo, "auth",
		slog.String("event", "session.revoked_all"),
		slog.String("reason", "user"),
		slog.Int64("count", count),
		slog.String("ingress", ingress),
		slog.String("client", client),
		slog.String("request_id", requestIDFrom(r.Context())),
	)
	h.clearSessionCookie(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
