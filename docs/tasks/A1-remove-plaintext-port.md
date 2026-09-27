# A1 — Remove the plaintext port

- **ID:** A1
- **Title:** Make Caddy the only ingress
- **Status:** `DONE` — deployed and server-verified 2026-09-22
- **Owner:** Claude
- **Implementation commit(s):** `6700cec`
- **Review status:** Implementation `6700cec`, trusted TLS `ac9ae82`; production checks passed.

## Problem

`docker-compose.yml` publishes `3005:8080`, a plaintext HTTP path straight to
the application that bypasses Caddy. Every "HTTPS required" guarantee in the
Health docs is defeated by anyone who reaches that port.

## Why it is blocked rather than simply pending

Removing port 3005 while Caddy still serves `tls internal` leaves a self-signed
certificate as the *only* way in — a downgrade to access, not an improvement.
A1 and [A2](A2-trusted-tls.md) must land together, and A2 needs an operator
action first.

## Acceptance criteria

- [x] `ports:` removed from the `gyst-lifeops` service
- [x] Port 3005 not listening on any host interface
- [x] Application still reachable through Caddy
- [x] HSTS header added

## Blockers

[A2](A2-trusted-tls.md) — a trusted certificate must exist first.

## Next action

Phone verification for A2, then M3. The approved rollout removed the live
3005 mapping: `docker ps` shows only container port 8080 and `ss` shows no
3005 listener. Trusted HTTPS login returns 200 with HSTS; unauthenticated
Health redirects to login and Health sync returns 401. See A2 for backup evidence.
