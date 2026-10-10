# A2-DOMAIN — gyst.zaylolabs.com

- **Status:** DONE — origin deployed, route saved by Jay, public HTTPS/auth checks passed
- **Implementation:** `04cf28e`; [CI pending](https://github.com/JayDalool/gyst-lifeops/actions/runs/35821849995).
- **Scope:** Jay explicitly requested the public domain instead of the Tailscale
  hostname, plus step-by-step nutrition Shortcut setup.

## Finding

The existing remotely managed Cloudflare Tunnel route already names
`gyst.zaylolabs.com`, but its origin is `http://192.0.2.10:3005`. A1 removed that
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
- Go formatting/vet/build/unit/race checks passed locally.
- Public DNS resolves, but external HTTPS request timed out during inspection.
  Public success must be verified after the dashboard route update.

## Cloudflare configuration (saved by Jay)

In the existing tunnel's published application route for `gyst.zaylolabs.com`:

- Type: HTTP
- URL: `gyst-caddy:8081`
- HTTP Host Header: `gyst.zaylolabs.com`

No authenticated Cloudflare management API/dashboard is available in this
session. Jay saved this change after the server side was prepared and tested. Keep any existing access controls in place; do not disable
security settings as a shortcut. If a browser challenge interferes with Apple
Shortcuts, diagnose the actual response before changing policy.

## Next action

Follow the nutrition Shortcut guide using the public URL; compare all four
nutrients with MyFitnessPal.

## Public verification

After Jay saved the route, HTTPS `/login` returned 200 with certificate verify
result 0 from both host and proxy network. Responses came through Cloudflare
and retained no-store. `/health` returned 303 to login; POST `/api/health/sync`
without credentials returned 401. Public password login and an authenticated
phone sync on this hostname remain user checks; the previous Tailscale-hostname
phone login/sync were already confirmed.

Phone guide: [HEALTH_SHORTCUT_SETUP.md](../HEALTH_SHORTCUT_SETUP.md).
