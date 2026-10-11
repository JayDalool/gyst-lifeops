# Health Core — Milestone 2

This milestone provides storage, deterministic calculations, a secure sync API,
and a server-rendered cycle form. It does not construct an Apple Shortcut or
connect any provider. All Summary cards remain demo values; targets and received
snapshots can be inspected through the plan form and authenticated API.

Migration `018_health_core.sql` follows `017_activity_events.sql` in the combined
migration sequence. Health SQL is unchanged by this numbering correction.

## Configuration and access

Set `GYST_HEALTH_SYNC_TOKEN` to a private, randomly generated secret in the server
environment. Docker Compose passes this variable through; an untracked `.env`
can supply it. Both Health endpoints require `Authorization: Bearer <token>`.
There is no default credential, query-token support, or token in rendered HTML.
Missing server configuration returns 503; missing/wrong credentials return 401.
The comparison uses constant-time comparison of fixed-length SHA-256 digests.
Use HTTPS. No additional public Health read endpoint exists.

`/health/plan` follows the existing private application's web access model, like
other forms, and rejects cross-site browser submissions. API authentication does
not introduce site-wide login. API and plan responses use `Cache-Control:
no-store`; the service worker bypasses them and all authenticated requests.

## Sync contract

`POST /api/health/sync` accepts `Content-Type: application/json`, a single JSON
object, and at most 16 KiB. Unknown fields, malformed JSON, extra documents,
invalid dates, unsupported sources, and negative/non-finite numbers are rejected.
The allowed source in M2 is `apple_shortcuts`. Dates must be valid `YYYY-MM-DD`
calendar dates. At least one numeric metric is required.

Synthetic payload illustrating the contract (never automatically seeded):

```json
{
  "date": "2026-09-10",
  "source": "apple_shortcuts",
  "dietary_calories": 130,
  "protein_g": 8,
  "carbs_g": 13,
  "fat_g": 6,
  "fiber_g": 1,
  "steps_watch": 1195,
  "steps_phone": 1302,
  "active_energy_kcal": 211,
  "resting_energy_kcal": 1318.596,
  "exercise_minutes": 0
}
```

All metric fields are optional, nullable numbers; step fields must be integers.
All numeric inputs and cycle targets have a generous 1 billion upper validation
bound to reject unreasonable magnitudes and keep arithmetic finite. This bound
is an input safeguard, not a health recommendation.

- Omitted fields and explicit JSON `null` mean no new measurement and preserve
  stored values. A new row retains NULL for metrics never received.
- Explicit numeric `0` is stored as zero.
- There is no metric-clearing operation in M2.
- Each received number replaces that date's aggregate for that metric; it is
  **not** added as a delta. Repeating a payload does not duplicate records.
- Success returns HTTP 200 with the merged snapshot. `last_synced_at` records the
  server receipt time in UTC and refreshes on successful retries.
- Snapshot and source sync state commit together or both roll back.
- **Date window.** Only today and the previous 7 local days are accepted (app
  clock, `GYST_TIMEZONE`; constants `HealthSyncPastDays = 7`,
  `HealthSyncFutureDays = 0`). Any other well-formed date, including a future
  date, gets HTTP 422 `date must be between <earliest> and <latest>` and nothing
  is written. Older backfill is refused; a backfill tool is out of scope. A
  malformed date is still 400, and a wrong token is still 401 before the window
  is checked. Each refusal logs one `SECURITY: health sync refused: date outside
  window` line and one `health.sync_refused` audit row, both with the date and
  bounds only.
- Watch and phone steps remain independent; neither is labeled a reconciled
  total and they are never added together.

Errors are JSON objects with an `error` string. Other statuses are 400 for invalid
input, 404 for unknown Health endpoints, 405 for incorrect methods (with `Allow`),
413 for oversized JSON, 415 for incorrect media type, and 500 for storage errors.
Storage errors do not disclose SQL or private data in API responses.

## Today and observed energy

`Service.HealthToday()` returns `date`, nullable `snapshot`, nullable
`health_cycle`, and `observed`. It is a service-layer call with no HTTP
endpoint: the Health API is write-only, and the server-rendered dashboard will
call this directly rather than going through an API. The date uses the existing
configured local timezone
(`prayer.timezone`, default America/Winnipeg). Incoming sync dates are stored
literally, without timezone conversion. Weight timestamps include timezone and
are normalized to UTC; weight is always stored in kilograms.

Observed fields are:

```text
observed_total_burn     = resting_energy_kcal + active_energy_kcal
observed_energy_balance = dietary_calories - observed_total_burn
food_remaining         = daily_calorie_target - dietary_calories
```

Each result is null if a required input is absent. Zero inputs remain valid.
Negative food remaining represents consumption over target and is not clamped.
The API makes no claim that observed afternoon burn is final daily burn. Steps,
exercise minutes, and target deficit never add calories to observed burn.
There is no projected burn, final deficit forecast, or burn recommendation.

## Cycles and weight

`GET /health/plan` edits the current cycle when present, otherwise opens a new
form. `?new=1` starts a new cycle; `?id=<id>` edits a selected cycle. POST to the
same route validates and saves, then redirects to the saved cycle. Failed
validation returns 400 and retains entered values for correction.

Cycles require title, primary goal, and end date after start date. Targets are
non-negative. Dates include both endpoints, so the next cycle must begin after
the previous end date. No status or fixed 90-day duration is imposed: date ranges
define active periods. The default end date is 12 weeks after the start, editable
by the user. Overlap is rejected by the service and guarded atomically by SQL.

Weight entries support multiple measurements, retain their source and timestamp,
and store positive kilograms. Repository and service operations exist; there is
no weight-entry UI or API in this milestone. Weight source can be `manual` or
`apple_shortcuts`; the daily sync API does not accept weight.

## Deferred

- M3: construct and validate the actual Apple Shortcut.
- M4: live Health dashboard and replacement of demonstration cards.
- Later: Hevy, food/nutrition databases, recipes, historical charts, AI, source
  reconciliation for steps, and justified daily energy projections.

Tests use temporary databases only. Deployment and migration of the personal
production database are separate from this milestone's commit/push workflow.
