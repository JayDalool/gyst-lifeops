# A5 — Health API write-only

- **ID:** A5
- **Title:** Remove `GET /api/health/today`
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):** `18196e9b4357034c2a3d3308efc391e0e00ad16f`
- **Review status:** CI green —
  [run 34644653007](https://github.com/JayDalool/gyst-lifeops/actions/runs/34644653007).
  Human or independent-agent review: **UNKNOWN**.

## What changed

`POST /api/health/sync` is now the entire Health API. The read endpoint had zero
non-test consumers — only the README, the Health doc and its own tests
referenced it.

Removing it beats the alternative the plan first proposed (a second token to
narrow read access): `GYST_HEALTH_SYNC_TOKEN` becomes a write credential *by
construction* rather than by convention, with no second credential to manage.
That matters because the Shortcut holding it syncs through iCloud and is
shareable from the share sheet in one tap — a leak now costs junk health data
rather than a copy of a health history. The sync endpoint already returns the
merged snapshot, so the Shortcut keeps its confirmation.

`Service.HealthToday()` is kept deliberately. It has no HTTP caller, but it is
tested and is what the server-rendered dashboard calls in
[M4](M4-live-day-view.md).

## Acceptance criteria

- [x] Consumers re-verified as zero immediately before deletion
- [x] `GET`/`POST /api/health/today` return 404
- [x] `POST /api/health/sync` unaffected
- [x] Partial-merge and NULL-versus-zero coverage preserved by retargeting it at
      the service layer rather than deleting it with the endpoint
- [x] README and `docs/HEALTH_CORE.md` updated
- [x] CI green

## Blockers

None.

## Next action

None.
