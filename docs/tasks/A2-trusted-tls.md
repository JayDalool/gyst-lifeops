# A2 — Trusted TLS for Apple Shortcuts

- **ID:** A2
- **Title:** Tailscale MagicDNS certificate with a renewal strategy
- **Status:** `BLOCKED`
- **Owner:** Jay (operator action required before any code can be written)
- **Implementation commit(s):** None. `Caddyfile` is unchanged on
  `feature/health-hardening`.
- **Review status:** N/A — not started.

## Problem

The `Caddyfile` uses `tls internal` — a self-signed CA. Apple Shortcuts
validates TLS and offers no bypass, so `Get Contents of URL` fails with an
opaque error. This blocks M3 (the Health Sync Shortcut) entirely.

The current site addresses are bare IPs and short hostnames
(`jay-server`, `10.0.0.183`, `100.117.218.14`). None of those can receive a
trusted certificate; the MagicDNS FQDN can.

## Why it needs an operator action first

`tailscale cert` only issues for the MagicDNS name **after HTTPS is enabled in
the tailnet admin panel**. Writing the Caddyfile before that would produce a
configuration that breaks the deploy.

## Acceptance criteria

- [ ] HTTPS enabled in the Tailscale admin panel *(operator)*
- [ ] Caddyfile site address is the MagicDNS FQDN
- [ ] `tls internal` replaced with the issued certificate
- [ ] Renewal scheduled — certificates are 90-day Let's Encrypt and Caddy cannot
      renew them through its normal ACME path, because the challenge is brokered
      by Tailscale. An unrenewed certificate breaks Shortcut sync ~90 days after
      M3 ships, which is the worst possible failure timing.
- [ ] Verified from the iPhone: a Shortcut `Get Contents of URL` succeeds

## Blockers

Operator: enable HTTPS in the Tailscale admin panel.

## Next action

Jay: enable Tailscale HTTPS. Then A2 and [A1](A1-remove-plaintext-port.md) land
together.
