# Google Calendar Integration — Plan

Status: **ICS export shipped; OAuth not built.** This documents what works today
and the staged path to deeper Google integration. No Google OAuth, secrets, or
third-party SDK is added in this pass.

## What works now (shipped)

GYST exports any dated item as a standard iCalendar feed:

| Route | Contents |
|-------|----------|
| `/calendar.ics` | Unified: course blocks + project target dates + task due dates |
| `/courses/calendar.ics` | All course learning blocks |
| `/courses/{id}/calendar.ics` | One course |
| `/projects/calendar.ics` | Project target dates + task due dates |

Each `VEVENT` has a stable `UID` (`block-<uuid>@gyst`, `project-<uuid>@gyst`,
`task-<uuid>@gyst`), `DTSTAMP`, `SUMMARY`, a `DESCRIPTION` that links back to
GYST, and either a timed `DTSTART/DTEND` (blocks with a start time, floating
local wall-clock) or an all-day `DTSTART;VALUE=DATE` (due-date-only items).
Output is CRLF-terminated `text/calendar`.

### Two ways to use it
1. **Download / import** — open the `.ics` once; the events are copied into your
   calendar (a snapshot, not live).
2. **Subscribe to the URL** — Apple Calendar ("New Calendar Subscription") or
   Google Calendar ("Other calendars → From URL") poll the feed and stay in
   sync as you add items. This is the recommended path.

## ⚠️ Cloudflare Access caveat

If GYST is put behind **Cloudflare Access** (or any auth gateway), Google's and
Apple's calendar fetchers **cannot** load the private `.ics` URL — they aren't
logged in, so the gateway returns the login page instead of calendar data and
the subscription silently shows nothing.

Options when that happens:
- **Tokenized feed URL (recommended next step):** a per-user unguessable token
  (`/calendar.ics?token=…` or `/feed/<token>.ics`) that bypasses Access for that
  one read-only route via a Cloudflare Access *service token* / bypass policy.
  Keeps the rest of the app gated; the feed is unguessable but unauthenticated.
- **Google OAuth** (below) so GYST pushes events out instead of the calendar
  pulling them in — no public feed needed.

## Future: Google Calendar OAuth (not now)

When two-way sync is worth the cost:
- Connect a Google account via OAuth 2.0 (offline access for a refresh token).
- Create/update events from GYST (course deadlines, project targets) keyed by the
  same stable UIDs so re-syncs upsert instead of duplicating.
- Read Google events/reminders back and surface them in Today.
- Sync Google reminders into GYST tasks.
- Store the refresh token encrypted; **all secrets (client ID/secret, tokens) in
  environment variables, never in the repo**.
- Handle token refresh + revocation; degrade gracefully to ICS export if a token
  is missing or revoked.

### Future: Apple Calendar / CalDAV
Only if subscribe-via-ICS proves insufficient. CalDAV is heavier (app-specific
passwords, PROPFIND/REPORT); defer until there is real demand.

## Sequencing
1. ICS export + subscribe — **done.**
2. Tokenized feed URL — small, unblocks Cloudflare Access subscribers.
3. Google OAuth push — larger; only when one-way export isn't enough.
4. CalDAV — last, demand-driven.
