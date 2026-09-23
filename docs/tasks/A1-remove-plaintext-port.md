# A1 — Remove the plaintext port

- **ID:** A1
- **Title:** Make Caddy the only ingress
- **Status:** `IN PROGRESS` — source committed, production rollout pending
- **Owner:** Claude
- **Implementation commit(s):** `6700cec`
- **Review status:** Implementation exists in `6700cec`; production acceptance not met.

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
- [ ] Port 3005 not listening on any host interface
- [ ] Application still reachable through Caddy
- [x] HSTS header added

## Blockers

[A2](A2-trusted-tls.md) — a trusted certificate must exist first.

## Next action

Deploy together with A2 after the operator backup and auth readiness check.
The 2026-09-22 inspection found production still publishing port 3005; the
earlier DONE label did not establish deployment. See A2 for verification.
