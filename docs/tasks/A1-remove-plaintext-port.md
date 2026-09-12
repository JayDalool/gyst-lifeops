# A1 — Remove the plaintext port

- **ID:** A1
- **Title:** Make Caddy the only ingress
- **Status:** `BLOCKED`
- **Owner:** Claude (blocked on operator action for A2)
- **Implementation commit(s):** None. No commit touches `docker-compose.yml`
  for this on `feature/health-hardening`.
- **Review status:** N/A — not started.

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

- [ ] `ports:` removed from the `gyst-lifeops` service
- [ ] Port 3005 not listening on any host interface
- [ ] Application still reachable through Caddy
- [ ] HSTS header added

## Blockers

[A2](A2-trusted-tls.md) — a trusted certificate must exist first.

## Next action

Wait for A2. Nothing to implement until then.
