# Calendar & Course Integration — Design Notes (Planning Only)

Status: **planning**. Nothing here is built yet. No Google/Apple OAuth in this
pass. This captures a staged path so we can ship value early (manual, local-first)
and only take on heavy sync later if it earns its keep.

GYST is local-first (SQLite, no external accounts). The plan keeps that property:
everything works offline first; calendar *sync* is an optional later layer.

## Concepts to model

These map cleanly onto the existing "trip → mission" vocabulary:

| New concept      | GYST analogue                  | Notes |
|------------------|--------------------------------|-------|
| Course           | Project / "Trip"               | A course is a long-running project. |
| Lecture block    | Recurring task occurrence      | Reuses the repeat engine already powering daily missions. |
| Study block      | Task with a time window        | Self-scheduled focus session. |
| Assignment       | Project mission with due date  | Already supported (mission + due_date + priority). |
| Postpone         | New "catch-up" mission         | Reschedule = create a follow-up occurrence, keep history. |

The only genuinely new data is **scheduling metadata** (day-of-week, start/end
time, location) and a **term/semester** range. Most else is reuse.

## Staged MVP (recommended order)

1. **Manual course schedule inside GYST.**
   - New `courses` table: `id, name, term_start, term_end, color, notes`.
   - A course is surfaced as a Project so the roadmap/route visuals work for free.
2. **Recurring lecture blocks.**
   - `course_blocks` table: `course_id, weekday (0–6), start_min, end_min, location, kind (lecture|lab|study)`.
   - Generate occurrences from blocks using the existing repeat/occurrence
     machinery (don't invent a second scheduler).
3. **"Postpone" → catch-up mission.**
   - Postponing a lecture/assignment doesn't delete it; it spawns a catch-up
     mission (due = next free study block, or user-picked date) and marks the
     original `postponed`. Preserves streak/history honesty.
4. **ICS export (read-only, no OAuth).**
   - Serve `/courses.ics` (and per-course `/courses/{id}.ics`) as a standard
     VEVENT feed with RRULE for recurring blocks + VEVENT for assignment due
     dates. Apple Calendar and Google Calendar can *subscribe* to a URL with
     zero auth. This is the cheapest real interop win and should ship before any
     OAuth work.
   - Implementation: pure Go text generation; no library needed. Use VTIMEZONE
     or floating local times to start.
5. **Later — Google Calendar OAuth import/sync.**
   - Only if subscribe-via-ICS proves insufficient. Two-way sync is the
     expensive part (conflict resolution, token refresh, webhook channels).
   - Keep it opt-in and isolated behind a `calendar_sync` settings flag so the
     local-first core never depends on it.
6. **Later — Apple Calendar / CalDAV.**
   - Only if there's demand beyond ICS subscribe. CalDAV is heavier than it
     looks; defer until 1–4 are validated.

## Why this order

- 1–3 are **pure local** features that deliver immediate value and reuse the
  occurrence engine — low risk, no third-party surface.
- 4 (ICS export) gives real Apple/Google interop **without** OAuth, tokens, or a
  background sync loop. Highest value-to-effort ratio.
- 5–6 are the only parts needing accounts/secrets; gate them behind a flag and
  don't start until the local model is proven.

## Lightweight UI concept (not yet built)

- A **"Course" trip type** on the existing project form (adds term dates + a
  weekly block editor: pick weekday + start/end + kind).
- The roadmap/route view already visualizes a course's assignments as a flight
  path — no new visualization needed.
- Today page gains a **"Today's Schedule"** strip: lecture/study blocks for the
  current weekday, each with a one-tap **Postpone** that creates the catch-up
  mission.
- A small **"Subscribe in Calendar"** button exposing the ICS URL once (4) lands.

## Open questions (resolve before building)

- Term/timezone handling: store local wall-clock times vs UTC. (Lean local for a
  single-user local-first app; revisit if sync arrives.)
- Do study blocks auto-generate from free time, or stay fully manual? (MVP: manual.)
- Streak semantics for postponed vs skipped lectures. (Proposal: postpone is
  neutral, skip breaks streak — mirrors current mission rules.)
