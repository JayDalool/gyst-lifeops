# Google Calendar Integration — Plan

Status: **ICS export shipped; OAuth not built.** This documents what works today
and the staged path to deeper Google integration. No Google OAuth, secrets, or
third-party SDK is added in this pass.

## What works now (shipped)

GYST exports every dated item as a standard iCalendar feed:

| Route | Contents |
|-------|----------|
| `/calendar.ics` | Unified: everything below |
| `/courses/calendar.ics` | All course learning blocks |
| `/courses/{id}/calendar.ics` | One course |
| `/projects/calendar.ics` | Project target dates, task due dates, mission-item due dates, roadmap milestone due dates |

**Deadline sources in the unified feed** (every model that carries a date):
- Course learning blocks / deadlines — `block-<uuid>@gyst`
- Project target dates — `project-<uuid>@gyst`
- One-time task due dates — `task-<uuid>@gyst`
- Project mission-item due dates — `item-<uuid>@gyst`
- Roadmap milestone due dates — `milestone-<uuid>@gyst`
- Standalone reminders with a date — `reminder-<uuid>@gyst`

**Reminders — included, but only standalone ones.** The `reminders` table holds
two kinds (verified against `syncTaskReminder`): *task-derived* rows, which the
app auto-creates with `reminder_date = the task's due_date`, and *standalone*
rows (`source_type != 'task'`) a user sets independently. Task-derived reminders
are **not** emitted because their date is identical to the task's due-date event
already in the feed — emitting both would show the item twice. Standalone
reminders **are** emitted (with their own date/time). Goals have no due-date
field (only start/end ranges) and are intentionally excluded. No date field is
invented for any model.

Each `VEVENT` has a stable `UID` (from the row's `uuid`), `DTSTAMP`, `SUMMARY`,
a `DESCRIPTION` that links back to GYST, and either a timed `DTSTART/DTEND`
(blocks with a start time, floating local wall-clock) or an all-day
`DTSTART;VALUE=DATE` (due-date-only items). Output is CRLF-terminated
`text/calendar`.

### Two ways to use it
1. **Download / import** — open the `.ics` once; the events are copied into your
   calendar (a snapshot, not live).
2. **Subscribe to the URL** — Apple Calendar ("New Calendar Subscription") or
   Google Calendar ("Other calendars → From URL") poll the feed and stay in
   sync as you add items. This is the recommended path.

## Private tokenized feed (shipped)

If GYST is put behind **Cloudflare Access** (or any auth gateway), Google's and
Apple's calendar fetchers **cannot** load a gated `.ics` URL — they aren't logged
in, so the gateway returns the login page instead of calendar data and the
subscription silently shows nothing.

The fix is a **tokenized feed**: an unguessable per-install secret embedded in
the URL so the feed route can be allowed through Access while the rest of the app
stays gated.

| Tokenized route | Contents |
|-----------------|----------|
| `/calendar/feed/{token}.ics` | Unified |
| `/calendar/feed/{token}/courses.ics` | Courses |
| `/calendar/feed/{token}/projects.ics` | Projects + tasks + items + milestones |

- Generate / rotate / revoke the token under **Settings → Private Feed URL**.
- The token is 32 random bytes (64 hex chars) from `crypto/rand`, stored in
  `app_settings`. It is shown in Settings so you can copy the URL, but is **never
  logged**. Validation is constant-time; a wrong token returns `404`.
- **Anyone with the URL can read the feed** (that's how unauthenticated calendar
  fetchers work) — the Settings page warns about this. Regenerate to invalidate a
  leaked link.
- To use behind Cloudflare Access: add a **Bypass / Service Auth** policy for the
  path `"/calendar/feed/*"` so only that route skips the login wall.
- The plain `/calendar.ics`, `/courses/calendar.ics`, `/projects/calendar.ics`
  remain for local/dev use; **do not expose them publicly** — prefer the
  tokenized routes for any external subscription.

A future alternative is **Google OAuth** (below) so GYST pushes events out
instead of a calendar pulling them in — no public feed needed.

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
2. Tokenized feed URL (unblocks Cloudflare Access subscribers) — **done.**
3. Google OAuth push — larger; only when one-way export isn't enough.
4. CalDAV — last, demand-driven.
