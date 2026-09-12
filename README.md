# LifeOps Tracker

LifeOps Tracker is a local-first Go web app for daily checklists, recurring tasks, project checklists, reminders, goals, and progress graphs. Data lives in SQLite under `data/lifeops.db`.

## Run

```powershell
go run .
```

Then open [http://localhost:8080](http://localhost:8080).

The `cmd/lifeops` entrypoint is also available:

```powershell
go run ./cmd/lifeops
```

Optional flags:

```powershell
go run . -addr :9090 -db data/lifeops-dev.db
go run . -seed=false
```

The app applies SQL migrations on startup and loads `seeds/example.sql` only when the database has no tasks, projects, or goals.

## Back up the database

```powershell
go run . backup
go run . backup -db data/lifeops.db -out backups/before-migration.db
```

In Docker, where the same binary is already present:

```sh
docker compose exec gyst-lifeops /app/lifeops backup
```

The backup is written with `VACUUM INTO`, so the server can keep running. The
source is opened read-only and is never modified, and the copy is verified with
`PRAGMA integrity_check` before the command reports success. The default
destination is `backups/` beside the database — inside the mounted volume — named
with a sortable UTC timestamp. An existing destination is never overwritten.

**Do not back up with `cp data/lifeops.db`.** The database runs in WAL mode, so
recently committed data lives in `data/lifeops.db-wal` until a checkpoint. The
main file on its own can be almost empty, and the copy gives no sign of it.

## Features

- Restrained, responsive command-center UI with five primary destinations: Today, Projects (Galaxy), Health, Courses, and More.
- `/` and `/today` share one prayer-first daily surface: next prayer, live countdown, five-prayer tracker, Athkar, missions, study blocks, and deadlines.
- Prayer times are calculated locally with configurable location, method, Asr method, and signed minute adjustments.
- Explicitly enabled open-page prayer alerts support browser notifications, an app-owned chime, or a user-supplied Athan file.
- Installable PWA support and VAPID Web Push deliver prayer notifications when the browser is closed; mobile operating systems control the notification sound and cannot play a full Athan in the background.
- The interactive 3D Galaxy uses stable project positions, project state/progress, and chapter moons. It includes touch controls, reduced motion, a keyboard list, and a conventional `/projects` fallback.
- Compact manual references connect tasks, projects, and missions to Jira, GitHub, Slack, Linear, Asana, Notion, or other work systems without affecting XP.
- More holds Monthly, Analytics, Archive, All Tasks, Goals, Reminders, Settings, and project navigation shortcuts. Add and Search remain in the header.
- Task CRUD with priority, status, due date, recurrence, reminder time, complete, skip, and archive actions.
- AI plan import turns a strict Markdown plan into new projects, project missions, standalone tasks, or checklist items on an existing task, with a preview before writing.
- Standalone tasks support persistent checklist items that can be completed independently from the task itself.
- Today is generated from task recurrence rules plus per-date completion records, with reminders and overdue items surfaced beside the checklist.
- `/projects` as card-based project progress, with detail pages for checklist execution.
- `/goals` as card-based yearly/monthly/weekly progress, with habit-based health goals supported.
- `/health` includes a clearly labeled, static Health Summary preview with secondary section links and Day/Week/Month/Year presentation controls. Existing gym consistency, movement trends, health goals, and tasks remain live below it.
- `/monthly`, `/analytics`, and `/reminders` as clean secondary views with dark-theme charts.
- Native selects/options are explicitly themed for readable dark-mode dropdowns, with light-mode overrides.
- Analytics charts use a lightweight custom canvas renderer, compact sparklines, and batched redraws instead of heavy chart re-creation.
- Optional light mode toggle in Settings, stored locally in the browser.
- The Health Summary remains a presentation preview. Health Core stores configured cycles and received daily aggregates separately; the dashboard does not display those aggregates yet.

## UI Structure

- `/` and `/today` are the same daily command center.
- `/galaxy` is the spatial project overview; `/projects` is its conventional list fallback and project detail remains checklist-based.
- Add creates a task, project, or course from the header.
- Secondary tools live under More to keep daily navigation calm.
- `/dashboard` remains a compatibility redirect to Today.

## Data Model

SQLite tables are created from `migrations/001_init.sql`:

- `tasks`
- `task_completions`
- `projects`
- `project_items`
- `goals`
- `goal_progress_entries`
- `categories`
- `reminders`

Additional lightweight tracking tables are created from `migrations/002_trackers.sql`:

- `tracker_items`
- `tracker_logs`

Default tracker items seed the Daily Mission surface:

- `prayer_fajr`, `prayer_dhuhr`, `prayer_asr`, `prayer_maghrib`, `prayer_isha`
- `athkar_sabah`, `athkar_masaa`
- `gym`

Recurring tasks are not duplicated. The app generates due occurrences from each task's `repeat_type` and `due_date`, then joins them with `task_completions` for the specific occurrence date.

Most tables include `uuid`, `sync_version`, timestamps, and `last_modified_device_id` fields so future clients can sync against stable records. The tracker tables follow the same sync-friendly pattern.

## Project Structure

```text
cmd/lifeops/            Go entrypoint
internal/app/           App bootstrap
internal/db/            SQLite open, migrations, seed loading
internal/handlers/      HTTP handlers and template rendering
internal/models/        Shared model structs
internal/repository/    SQLite persistence layer
internal/services/      Recurrence, dashboard, reminders, stats
migrations/             SQL migrations
seeds/                  Example data
web/templates/          Server-rendered HTML
web/static/             CSS and chart JavaScript
```

## Developer Documentation

- `docs/ARCHITECTURE.md` describes the current application layers and data flow.
- `docs/QUEST_SYSTEM.md` documents quest statuses, dependency unlocking,
  rewards, proof validation, Markdown import, and remaining product gaps.

Prayer times are calculated locally with Winnipeg, ISNA, and Shafi defaults.
The engine supports MWL, Umm al-Qura, Egyptian, and Karachi methods plus Hanafi
Asr and signed per-prayer minute adjustments. Today provides a live countdown
and explicitly enabled open-page notification/audio alerts.

Database migrations are applied in filename order and currently run through
`018_health_core.sql`.

## HTTPS and Web Push

Generate VAPID keys once and place them in an untracked `.env` file:

```sh
docker run --rm -v "$PWD":/app -w /app golang:1.25-bookworm go run ./cmd/vapid
```

Replace the generated subscriber placeholder with your `mailto:` address, then
start the app and Caddy reverse proxy with `docker compose up -d --build`.
HTTPS is available at `https://localhost:3443`. Caddy uses its internal CA for
local deployment; trust the CA certificate exported from the `caddy_data`
volume before enabling push in a browser. Public deployment should use a real
hostname so Caddy obtains a publicly trusted certificate automatically.

## Reminders TODO

Version 1 stores reminder dates and times in SQLite and shows due or overdue reminders on Today.

Future iOS reminders should use native Swift `UserNotifications` so reminders work on-device even when the Go web app is not open.

## Future iOS App Plan

- Build a native iOS app using SwiftUI.
- Use HealthKit to read workouts, steps, and sleep data with user permission.
- Keep health tracking habit-based and opt-in.
- Use Swift Charts for iOS graphs.
- Use UserNotifications for local reminders.
- Sync with this Go backend through REST API endpoints.
- Reuse server UUIDs, `sync_version`, and timestamp fields for conflict handling.

## Health Core (Milestone 2)

`/health/plan` creates and edits goal periods with user-chosen dates and targets.
Daily snapshots retain missing metrics as NULL and keep Watch and phone steps
separate. Health Summary cards remain demo values until the Live Dashboard
milestone.

`POST /api/health/sync` is the entire Health API. It is deliberately
write-only: there is no read endpoint, so `GYST_HEALTH_SYNC_TOKEN` is a write
credential by construction rather than by convention, and the Apple Health
Shortcut that holds it cannot be used to read a health history back out.

It requires `Authorization: Bearer <token>`. Set `GYST_HEALTH_SYNC_TOKEN`
privately in the server environment (or the untracked Compose `.env`); use a
long randomly generated secret. Compose passes it to the app. Do not put it in
URLs, browser JavaScript, source control, or logs. The endpoint fails closed
with HTTP 503 when the secret is unset, and rejects missing or incorrect
credentials with 401. Use the existing HTTPS deployment for API access.

See [Health Core](docs/HEALTH_CORE.md) for the payload contract, observed-energy
calculations, timezone rules, and deferred milestones.
