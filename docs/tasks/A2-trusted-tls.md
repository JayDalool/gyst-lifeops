# A2 — Trusted TLS for Apple Shortcuts

- **ID:** A2
- **Status:** IN PROGRESS — configuration verified in isolation; deployment and phone verification pending
- **Implementation commit(s):** See the A2 implementation commit in Git history.
- **Review status:** Local format/vet/build/unit/race checks passed; CI pending.

## Implementation

Caddy serves `GYST_TLS_HOSTNAME`, the server's full Tailscale MagicDNS name,
using its built-in `get_certificate tailscale` integration. Compose requires
that variable and mounts `/var/run/tailscale` read-only into Caddy. Mounting the
directory allows the socket to be replaced when tailscaled restarts. A missing
host directory fails rather than silently creating an empty directory.

Caddy obtains certificates from tailscaled during TLS handshakes. Certificate
renewal is automatic through this integration; there are no exported private
keys or scheduled file-copy jobs. The Caddy image runs as root, which permits
certificate requests. The socket grants privileged access to the host daemon;
a read-only bind does not restrict socket API operations. Keep it confined to
the trusted reverse-proxy container, never the application.

The previous IP/short-name self-signed sites are replaced by the MagicDNS
hostname. Port 3443 stays the HTTPS port. HSTS applies to this host, without
claiming a preload registration or imposing a policy on subdomains.

Compose also forwards the existing password/auth-disable variables so a deploy
does not silently discard the operator's current authentication configuration.
Authentication is implemented; historical handoff claims that it is a stub are
incorrect. Do not enable the auth-disable flag for the health rollout.

References:
- https://tailscale.com/docs/integrations/web-servers/caddy/caddy-certificates
- https://tailscale.com/docs/how-to/set-up-https-certificates
- https://caddyserver.com/docs/caddyfile/directives/tls

## Evidence (2026-09-22 session)

- Host tailscaled is running, MagicDNS and HTTPS certificate eligibility enabled.
- No Tailscale Serve route exists; Caddy remains the chosen ingress.
- Caddy 2.10.2 validates the configuration.
- Isolated Caddy obtained a trusted certificate through the mounted host socket.
- A disposable static backend returned HTTP 200 through HTTPS, curl certificate
  verification result 0 (no insecure bypass), with the HSTS header.
- An unconfigured hostname failed its TLS handshake (curl exit 35).
- Host-loopback access to the temporary published port timed out; testing inside
  the probe network passed. Actual host/phone reachability is NOT established.
- No production app/container/database was changed. The live app still exposes
  port 3005; A1 is not operationally complete.

## Acceptance criteria

- [x] HTTPS enabled in Tailscale (verified from host state)
- [x] Caddy configuration uses the full MagicDNS hostname
- [x] Trusted certificate retrieval verified in an isolated container
- [x] Automatic certificate renewal configured through the official integration
- [ ] Production deployment after operator backup and authentication check
- [ ] Real iPhone Shortcut successfully reaches the trusted HTTPS endpoint

## Deployment procedure (operator gate)

1. In the existing deployment directory, take a verified B2 backup before any
   deployment. Do not deploy from a new worktree with an empty `./data` directory.
2. Reconcile the deployment checkout with this feature branch while preserving
   local configuration. Set `GYST_TLS_HOSTNAME` in its environment to the full
   MagicDNS hostname reported by Tailscale, without protocol, port or trailing dot.
3. Ensure `GYST_PASSWORD_HASH` is configured and `GYST_AUTH_DISABLED` is unset;
   preserve the existing health token. Do not print credentials or compose's
   expanded configuration. Use `docker compose config --quiet` for validation.
4. Deploy the reviewed configuration. A1 already removes the application's
   published HTTP port; Caddy must be recreated to receive the new socket mount
   and environment. Do not deploy only one half of this ingress change.
5. Verify `https://<MagicDNS-hostname>:3443/login` without bypassing certificate
   validation; verify login, unauthenticated API denial and absence of port 3005.
6. On iPhone with Tailscale connected, open the HTTPS URL and test it with
   Shortcuts. Record results before marking A1/A2 complete or beginning M3.

## Next action

Operator backup and authentication readiness confirmation, then deploy and
verify the reviewed A1/A2 configuration on the phone.
