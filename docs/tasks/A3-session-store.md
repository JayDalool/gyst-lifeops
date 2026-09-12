# A3 — Session store and auth service

- **ID:** A3
- **Title:** Session storage and password verification (no HTTP surface)
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):** `7835b7d4a4777b2de47d4c63fe89e2b9188901c0`
- **Review status:** CI green —
  [run 34643476690](https://github.com/JayDalool/gyst-lifeops/actions/runs/34643476690).
  Human or independent-agent review: **UNKNOWN**, no artifact records one.

## Scope

Storage and service layer only. No HTTP surface, no middleware, no protected
route — [A4](A4-application-auth.md) wires it in. On its own this commit changes
no application behaviour.

## Design decisions worth re-reading before changing

- **Sessions are stored by SHA-256 of the token, never the token.** The raw
  token is returned once, goes in the cookie, and is not recoverable from the
  database. A backup or stolen disk cannot be turned into a working cookie.
  SHA-256 rather than bcrypt because the input is 32 bytes of CSPRNG output:
  a slow hash defends against nothing here and taxes every request.
- **Rolling 30-day expiry is throttled to one write per day.** Extending on
  every request would take a SQLite write lock on the read path for a value
  whose precision does not matter.
- **bcrypt over Argon2id** for the password: self-describing hash, constant-time
  compare in the library, no hand-rolled PHC parsing to get subtly wrong.
- Fails closed when `GYST_PASSWORD_HASH` is unset, matching the Health token.
- First migration to use a timestamp prefix (`20260911203000_sessions.sql`).

## Acceptance criteria

- [x] Raw token never stored — negative-tested
- [x] Rolling expiry throttled — negative-tested (removing it fails the test)
- [x] Unset password hash fails closed
- [x] Expired sessions rejected and removed; sweep leaves live ones
- [x] `cmd/passwordhash` output round-trips through `VerifyPassword`
- [x] CI green

## Blockers

None.

## Next action

Operator step before this can be used in production: generate a hash with
`go run ./cmd/passwordhash` and set `GYST_PASSWORD_HASH`. Tracked in
[OPS1](OPS1-production-fk-audit.md) alongside the other server tasks.
