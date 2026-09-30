# GYST security rules

Project rules for the security-guidance plugin. Shared rules live in
~/.claude/claude-security-guidance.md. Source: docs/RULES.md and the code.

- GYST is single-user. Every route needs the session cookie except the list
  in `authExemptPath` (internal/handlers/middleware.go): `/login`, `/logout`,
  the bearer-token Health API and tokenized `/calendar/feed/*`. A new exempt
  path needs a reason and its own test entry. The session token is stored
  only as a SHA-256 hash; the raw token lives only in the
  `Secure`/`HttpOnly`/`SameSite=Lax` cookie.
- Calendar feed tokens are credentials: never log them or put them in errors.
- `/api/health` and `/api/health/*` authenticate only with their own bearer
  token and are write-only. Never add a read endpoint there, never accept the
  session cookie there, never log request bodies or the token.
- Health data is personal health information: no values in logs, errors,
  fixtures (use `seeds/example.sql` style made-up data), screenshots or
  outside services. Missing stays NULL; never write 0 for missing.
- `GYST_PASSWORD_HASH` and every other secret come from the environment only;
  never in any file in the repository, documentation included.
- State-changing requests keep the same-origin check in the middleware
  (cross-site submissions are rejected); a new state-changing route needs a
  negative test for it (docs/RULES.md §1.6).
- SQLite access only with parameters; never build SQL with string formatting.
