# A2-DOMAIN — gyst.zaylolabs.com

- **Status:** IN PROGRESS — internal origin deployed; dashboard route update pending
- **Scope:** Jay explicitly requested the public domain instead of the Tailscale
  hostname, plus step-by-step nutrition Shortcut setup.

## Finding

The existing remotely managed Cloudflare Tunnel route already names
`gyst.zaylolabs.com`, but its origin is `http://10.0.0.183:3005`. A1 removed that
published port. The prior rollout missed this dependency. Do not restore public
plaintext access to work around it.

## Change

Caddy joins the existing external Docker network `proxy`, also used by
cloudflared, and serves `http://gyst.zaylolabs.com:8081` internally. No host
port 8081 is published. Only Caddy joins that shared proxy network; the
application remains on its existing private network. The origin preserves the
public Host for CSRF checks, forwards HTTPS as the client scheme for Secure
cookies, sets no-store and HSTS, and redirects requests marked as public HTTP
to public HTTPS. The Tailscale HTTPS listener remains available.

## Evidence

- Verified B2 backup `lifeops-20260923T051508Z.db` before Caddy deployment.
- Caddy validation and Compose config validation passed.
- From the proxy network: login 200, protected Health 303 to login,
  unauthenticated sync POST 401, public-HTTP redirect 301 to HTTPS.
- Existing Tailscale HTTPS login 200, certificate verification result 0.
- Neither 3005 nor 8081 has a host listener.
- Go formatting/vet/build/unit/race checks: see implementation commit evidence.
- Public DNS resolves, but external HTTPS request timed out during inspection.
  Public success must be verified after the dashboard route update.

## Remaining Cloudflare step

In the existing tunnel's published application route for `gyst.zaylolabs.com`:

- Type: HTTP
- URL: `gyst-caddy:8081`
- HTTP Host Header: `gyst.zaylolabs.com`

No authenticated Cloudflare management API/dashboard is available in this
session. The operator was given this exact change after the server side was
prepared and tested. Keep any existing access controls in place; do not disable
security settings as a shortcut. If a browser challenge interferes with Apple
Shortcuts, diagnose the actual response before changing policy.

## Next action

Jay saves the existing tunnel route, then verify public login and API denial
through `https://gyst.zaylolabs.com`. Do not claim the public domain works yet.

Phone guide: [HEALTH_SHORTCUT_SETUP.md](../HEALTH_SHORTCUT_SETUP.md).
