# A4 — Application authentication

- **ID:** A4
- **Title:** Require a session for the web UI
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):** `fb05cc3dcc30618e0527950df7c5fc2b4413c71d`
- **Review status:** CI green —
  [run 34644320678](https://github.com/JayDalool/gyst-lifeops/actions/runs/34644320678).
  Human or independent-agent review: **UNKNOWN**. **Not verified against a real
  browser or the PWA** — no artifact records a manual sign-in.

## Problem

Before this commit the only authenticated surface was `/api/health/*`.
`/health/plan` and every task, project and course route accepted requests from
anyone who could reach the port.

## Exempt routes, and why each one is exempt

Four, each either carrying its own credential or needed to obtain one. The list
is asserted entry by entry in tests so it cannot grow quietly.

| Path | Why |
|---|---|
| `/login` | Issues the session. |
| `/logout` | Must work when the session it refers to has already lapsed. |
| `/api/health/*` | Bearer token; the Apple Health Shortcut has no cookie jar. |
| `/calendar/feed/*` | Path token; Apple and Google Calendar never send a cookie. |

The calendar feed is the one that would have broken silently: subscriptions
would have started failing with no visible error. Adjacent paths
(`/calendar.ics`, `/calendar/feedback`) are asserted to *require* a session so
nothing inherits an exemption by prefix.

## Other decisions

- **`Secure` is conditional on TLS.** Setting it unconditionally makes login
  fail silently over plain HTTP: the browser drops the cookie and the next
  request bounces back to `/login` with nothing to explain why.
- **Unauthenticated mutations get 401, not a redirect** — a POST cannot be
  replayed after signing in, and redirecting would drop it while appearing to work.
- The cross-site guard extracted from `health_plan.go` now covers POST, PUT,
  PATCH and DELETE.
- `GYST_AUTH_DISABLED=1` serves everything unauthenticated and logs loudly. It
  exists so a bad deploy is recoverable without shell access to the database.

## Acceptance criteria

- [x] No route outside the exempt list serves data without a session
- [x] Every exemption asserted individually, with its reason
- [x] Cross-site mutations rejected for all four unsafe methods
- [x] Shell assets and `/login` reachable while signed out (tested through the real mux)
- [x] Post-login redirect cannot become an open redirect
- [x] CI green
- [ ] **Verified in a real browser / iOS PWA — NOT DONE**

## Blockers

None for the code. Deployment needs `GYST_PASSWORD_HASH` set first, or nobody
can sign in.

## Next action

Operator: set `GYST_PASSWORD_HASH` and **verify login works before closing the
session you deploy from**. `GYST_AUTH_DISABLED=1` is the escape hatch.
