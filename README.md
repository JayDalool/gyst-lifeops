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

## Features

- Dark, minimal command-center UI with glass panels, subtle cyan/blue/purple accents, and responsive layouts.
- Homepage with only four primary icon cards: Today, Projects, Goals, and More, plus a small floating Add menu.
- Daily Mission dashboard section with date-based toggles for five prayers, Sabah/Masaa Athkar, a weekly gym strip, and quick project momentum cards.
- Signal Board under the homepage command cards with compact clickable mini graphs for Deen consistency, body rhythm, task flow, project momentum, goal progress, and reminder load.
- More drawer with icons for Monthly View, All Tasks, Reminders, Analytics, Health, Archive, and Settings.
- Task CRUD with priority, status, due date, recurrence, reminder time, complete, skip, and archive actions.
- `/today` generated from task recurrence rules plus per-date completion records, with reminders and overdue items surfaced beside the checklist.
- `/projects` as card-based project progress, with detail pages for checklist execution.
- `/goals` as card-based yearly/monthly/weekly progress, with habit-based health goals supported.
- `/health` includes the weekly gym strip and a compact body-rhythm sparkline. The logic stays habit-based and consistency-focused.
- `/monthly`, `/analytics`, and `/reminders` as clean secondary views with dark-theme charts.
- Native selects/options are explicitly themed for readable dark-mode dropdowns, with light-mode overrides.
- Dashboard charts use a lightweight custom canvas renderer, compact sparklines, and batched redraws instead of heavy chart re-creation.
- Optional light mode toggle in Settings, stored locally in the browser.
- Habit-based health and wellness goals only. There are no weight-loss, calorie-counting, fat-loss, or body-comparison features.

## UI Structure

- `/` is the personal command center and intentionally stays simple.
- The command center uses four large icon cards and a compact Signal Board rather than a dense dashboard.
- `/today`, `/projects`, and `/goals` are the three primary work surfaces.
- `/today` includes the Deen tracker; `/health` includes the gym rhythm tracker.
- Secondary tools live behind the More drawer to keep the front page calm.
- The floating `+ Add` menu creates tasks, projects, or goals from anywhere.

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

Database migrations are applied in filename order and currently run through
`009_quest_reward_integrity.sql`.

## Reminders TODO

Version 1 stores reminder dates and times in SQLite and shows due or overdue reminders in the dashboard.

Future iOS reminders should use native Swift `UserNotifications` so reminders work on-device even when the Go web app is not open.

## Future iOS App Plan

- Build a native iOS app using SwiftUI.
- Use HealthKit to read workouts, steps, and sleep data with user permission.
- Keep health tracking habit-based and opt-in.
- Use Swift Charts for iOS graphs.
- Use UserNotifications for local reminders.
- Sync with this Go backend through REST API endpoints.
- Reuse server UUIDs, `sync_version`, and timestamp fields for conflict handling.
