# GYST — Claude / Codex Shared Handoff

**Last updated:** 2026-09-11
**Updated by:** Claude
**Repository:** `JayDalool/gyst-lifeops`
**Current implementation branch:** `feature/health-hardening`
**Current implementation HEAD:** `a1dc3da9600c9c36789db5b10226880eca9cff62`
**This document's branch:** `docs/ui-v3-standards` (documentation only, based on `a1dc3da`)

This file is the single shared coordination point between Claude, Codex and Jay. Read it at the start of every session. Git is the source of truth; if this file disagrees with the repository, **the repository wins** — correct this file, never rewrite history to match it.

---

## 1. Project goal

GYST is a self-hosted LifeOps system built primarily with **Go + SQLite**, server-rendered Go templates, plain CSS and small vanilla JavaScript modules. It is deployed with Docker behind Caddy. **Mobile-first PWA (iPhone, Add to Home Screen) is a primary target.**

Main product areas:

- Today
- Projects
- Galaxy
- Health
- Courses
- Reminders / supporting tools

The long-term Health architecture combines:

- GYST (owns the normalized domain model and all deterministic calculations)
- Apple Health
- Apple Watch
- MyFitnessPal nutrition, routed through Apple Health
- Apple Shortcuts ingestion
- Hevy Pro / Hevy API for detailed strength-training data

> **Important:** Hevy integration is **PLANNED but NOT implemented**. No Hevy code, tables, or provider exists in the repository today.

Guiding principle: external apps are **providers**, not the source of truth for application architecture.

---

## 2. Branch map

All SHAs below were verified with `git ls-remote` on 2026-09-11. Do not invent SHAs; re-verify before relying on them.

| Branch | HEAD | Meaning |
|---|---|---|
| `master` | `ae5e67403fac04978bc6ac7b8363955dc5d4cd58` | Mainline. Contains **no** Health work. |
| `feature/health-dashboard-foundation` | `055f098eb047947b0ff588397f6cba43d5359523` | M1 — UI shell + Health dashboard foundation. |
| `feature/health-core` | `3083fb91a9a03a684d59f0a1e370fdea8a906324` | M2 — Health Core. **Approved B1 baseline.** |
| `feature/health-hardening` | `a1dc3da9600c9c36789db5b10226880eca9cff62` | M2.5 hardening. Current implementation branch. |
| `docs/ui-v3-standards` | `da1c1e583618fe3671bbf7c36f24585fbd84a6cb` | Documentation only. Based on `a1dc3da`, so it contains all of B1. |

### B1 history on `feature/health-hardening`

```
3083fb9  Health baseline (feature/health-core)
fb4f1eb  B1 implementation — fix SQLite PRAGMAs on pooled connections
a1dc3da  B1 review fix — seed the integrity audit, finish file: URI support
```

**Do not rewrite, squash, force-push or rebase the B1 commits.**

### Branch relationship note

`docs/ui-v3-standards` branches from `a1dc3da`, so documentation and B1 code currently coexist there. Keep production UI work and hardening work on separate branches, per `docs/design/v3/GYST_UI_V3_STANDARDS.md` §1 of the implementation guide.

---

## 3. Completed work

Only claim what is verified in Git. Later M2.5/M3 work does **not** exist unless listed here.

### Health Core — IMPLEMENTED (on `feature/health-core`, inherited by later branches)

Models in `internal/models/health.go` (verified):

- `HealthCycle`
- `HealthMetrics`
- `DailyHealthSnapshot`
- `HealthSyncRequest`
- `HealthWeightEntry`
- `HealthSyncState`
- `ObservedHealthEnergy`
- `HealthTodayResponse`

Also present: repository layer, service layer, Health API handlers, the 90-day Health Plan form, bearer-token configuration (`GYST_HEALTH_SYNC_TOKEN`), an additive migration (`018_health_core.sql`), partial/idempotent sync, deterministic observed-energy calculations, and tests.

API endpoints: `POST /api/health/sync`, `GET /api/health/today`.
Web routes: `GET /health/plan`, `POST /health/plan`.

Migration count: **18** (`001`–`018`). `017_activity_events.sql`, `018_health_core.sql`. The historical 017 numbering conflict is **already resolved** in `3083fb9`.

### B1 — SQLite PRAGMA correctness — IMPLEMENTED

Commits: `fb4f1eb`, `a1dc3da`.

What was fixed:

- `foreign_keys` is now applied per SQLite **pooled connection**, not once against `*sql.DB`.
- `busy_timeout=5000` is applied per pooled connection.
- WAL remains handled as **persistent database state** (set once; it lives in the database file).
- Safe DSN construction added — filesystem paths become a percent-encoded `file:` URI, because the driver splits a DSN at the first `?` and would otherwise truncate paths containing `?`.
- Filesystem paths with spaces, `?`, `#`, `%` and Unicode tested.
- `:memory:` behaviour tested and preserved.
- `file:` URI support fixed (directory creation is skipped for DSN forms) and covered end to end through `Open`.
- Behavioural foreign-key enforcement tested across four simultaneous connections.
- Seeded `PRAGMA foreign_key_check` audit added, loading `seeds/example.sql` through the production `SeedIfEmpty` path with guards so the check cannot pass vacuously.

Measured before the fix: connection 0 had `foreign_keys=1`/`busy_timeout=5000`; connections 1–3 had `0`/`0` and **accepted orphan inserts**.

#### Important behavioural change

**Foreign keys are now actually enforced**, so the schema's existing relationships now execute as designed:

- **10 `ON DELETE CASCADE`** — deleting a parent now deletes children. `tasks` → `task_completions`, `task_checklist_items`; `projects` → `project_items`, `roadmap_project_meta`, `roadmap_milestones`, `roadmap_resources`; `goals` → `goal_progress_entries`; `tracker_items` → `tracker_logs`; `courses` → `learning_blocks`; `push_subscriptions` → `push_deliveries`.
- **13 `ON DELETE SET NULL`** — including `achievements.source_task_id` / `source_project_id` / `source_project_item_id`, and `projects.parent_project_id` / `depends_on_project_id` / `roadmap_id`.

XP note: migration 009's duplicate-reward guard is a **partial** unique index (`WHERE source_task_id IS NOT NULL`), so a NULLed row leaves that guard. This was checked and is **benign**: `tasks`, `projects` and `project_items` all use `INTEGER PRIMARY KEY AUTOINCREMENT`, so SQLite never reuses ids and no duplicate reward can be granted. XP already earned is retained; only the source pointer clears.

### UI v3 design documentation — DOCUMENTATION ONLY

On `docs/ui-v3-standards`. **Design reference, not implemented.** See §7.

---

## 4. Currently under review

**B1: code complete, independently reviewed, approved by Jay.**

An independent review of the range `3083fb9..a1dc3da` was completed. Verdict: **PASS WITH NOTES** — no blocking findings. Three non-blocking findings were raised and are **approved to be fixed as a follow-up commit** (see §10):

- **H1** — a caller-supplied `_pragma` in a `file:` URI can silently defeat enforcement. Verified: `_pragma=foreign_keys(off)`, `(false)` and `(OFF)` all beat the appended `foreign_keys(1)`, because the driver sorts pragma strings lexicographically and the last one executed wins. `foreign_keys(0)` is safe. `busy_timeout` has an inconsistent sort comparator when duplicated.
- **M1** — `:memory:` gives **each pooled connection its own private empty database**. Verified: a table created on connection 1 is invisible to connection 2. The existing test passes only because it reads pragmas through a single implicit connection, so it gives false assurance.
- **M2** — `TestForeignKeysAreEnforcedOnEveryConnection` holds 4 connections and then issues a 5th query. Safe only while `MaxOpenConns` is unlimited; it would deadlock if pool tuning is ever added.

### Still required before production deployment

Read-only audit against the **real** production database:

```sh
sqlite3 /path/to/data/lifeops.db "PRAGMA foreign_key_check;"
```

- Zero rows returned = safe to proceed.
- **If any rows are returned: STOP deployment.**
- **Do not auto-delete or auto-repair records.** Report them.
- Run with the application **stopped** so the WAL is checkpointed, and take a backup first.

---

## 5. Current blockers / gates

- The real deployed database has **not** been certified by `PRAGMA foreign_key_check`. Until it is, **B1 is not production-safe**, even though the code is approved.
- No backup tooling exists in the repository. B3 (pre-migration snapshot) does not exist. Take a manual backup before any deployment.
- **No CI exists.** There is no `.github/` directory. The safety gate is still manual.
- M3 must not begin before the M2.5 correctness/security gates are complete.
- Hevy is not integrated.
- UI v3 is **design-only** and must not be treated as implemented.
- There is still **no application authentication**. Only `/api/health/*` is credential-protected. `/health/plan` has a same-origin/`Sec-Fetch-Site` guard, but the rest of the app has neither auth nor that guard.

---

## 6. Approved architectural decisions

Recorded so agents stop reconsidering them every session.

### Database

- **SQLite stays.** Do not propose Postgres.
- Additive migrations only. No destructive `DROP`, no broad destructive `UPDATE`.
- **Never reset or reseed the real `data/lifeops.db`.**
- Tests use temporary databases only.
- Foreign keys must actually be enforced (delivered by B1).
- Migration correctness and working backups are required **before** storing irreplaceable Health data.
- Do not edit an already-applied migration. The runner tracks by filename with no checksum, so an edit is silently ignored and a rename re-runs the migration and fails.

### Health calculations

Never double-count energy. **Do not** add together:

- Active Energy
- steps converted to calories
- workout calories

Observed energy uses **resting energy + active energy** only. Steps are a separate activity metric and are never converted to calories. Hevy's own calorie estimates must never enter the energy engine.

Keep these three concepts visually and structurally distinct:

1. **Food budget** — target calories minus consumed calories.
2. **Observed energy balance** — consumed minus observed burn.
3. **Projected balance** — future feature, always labelled as projected.

### Nutrition

MyFitnessPal currently remains the food logger. Apple Health → Shortcut → GYST supplies dietary calories, protein, carbs, fat and fiber.

- Do not invent missing nutrition values.
- **Missing / stale / current must remain distinguishable.** `NULL` means "not received"; `0` means "an observed zero". Never render missing data as zero.

### Steps

Store Apple Watch and iPhone steps **separately** (`steps_watch`, `steps_phone`). **Never add them together** — the sources overlap and summing double-counts. Watch is primary where appropriate; phone is fallback/context.

### Training

Hevy Pro is the selected training system, but **the Hevy API integration does not exist yet**.

Future architecture:

```
Hevy API → HevyProvider → normalized GYST training domain → SQLite → Health Training UI
```

The UI must never depend on raw Hevy JSON field names anywhere in the application. Store generic `training_*` tables with `provider` / `provider_id` columns, not `hevy_*` tables.

### AI

AI comes **after** deterministic services. The assistant calls GYST tools/functions for nutrition, portions, energy balance, training and goals. It must **not** invent calorie or macronutrient calculations.

---

## 7. UI v3 direction

**Canonical consolidated design reference:** `docs/design/v3/GYST_UI_V3_STANDARDS.md`

Supporting split detail lives in the same folder: `README.md`, `01_VISUAL_SYSTEM.md`, `02_COMPONENT_LIBRARY.md`, `03_SCREEN_SPECS.md`, `04_HEALTH_SPECS.md`, `05_GALAXY_SPEC.md`, `06_IMPLEMENTATION_NOTES.md`. Where the consolidated standard and a split file disagree, **the consolidated standard wins**.

`docs/design/v2/` is retained as historical reference and must not be deleted. v3 supersedes it for new decisions.

Primary bottom navigation: **Today · Projects · Health · Courses · More**

Projects secondary: **Galaxy · List · Roadmap · Completed**

Health secondary: **Summary · Nutrition · Training · Progress · Plan**

Health time range: **Day · Week · Month · Year**

Style direction: Apple Health clarity + openGym workout usability + GYST identity.
**Do not clone Apple. Do not clone openGym.** No React/Vue/Tailwind, no front-end build pipeline, no runtime CDN.

Galaxy remains GYST's signature immersive component. The current production Galaxy must be **evolved, not deleted**, and its accessible list fallback must be preserved.

> Known gap: the standard references `docs/design/v3/assets/gyst_app_experience_showcase.webp` and `gyst_ui_system_style_guide.webp`. That `assets/` directory does **not** exist in the repository yet. The written specification takes priority over any image regardless.

---

## 8. Planned screen inventory

Status legend: **EXISTING** (implemented today) · **DESIGNED** (spec written, not built) · **PLANNED** (spec written, backend missing) · **NOT IMPLEMENTED**.

### Global

| Screen | Status |
|---|---|
| Login | PLANNED (M2.5A; no auth exists) |
| Today | EXISTING — redesign designed |
| Daily Checklist / All Tasks | EXISTING — redesign designed |
| Projects | EXISTING — redesign designed |
| Galaxy | EXISTING — redesign designed |
| Project Detail | EXISTING — redesign designed |
| Roadmaps | EXISTING — redesign designed |
| Project Create/Edit | EXISTING — redesign designed |
| Courses | EXISTING — redesign designed |
| Course Detail | EXISTING — redesign designed |
| Course Create/Edit | EXISTING — redesign designed |
| More | EXISTING — redesign designed |
| Reminders | EXISTING — redesign designed |
| Goals / Goal Detail | EXISTING — redesign designed |
| Calendar / Monthly | EXISTING — redesign designed |
| Notes | NOT IMPLEMENTED (no Notes domain exists; do not invent one) |
| Analytics | EXISTING — redesign designed |
| Archive | EXISTING — redesign designed |
| Settings | EXISTING — redesign designed |
| Integrations | PLANNED |
| Search / Command palette | EXISTING — redesign designed |
| Empty / loading / error states | EXISTING behaviour — standardization designed |

### Health

| Screen | Status |
|---|---|
| Health Summary | EXISTING (placeholder values) — major redesign designed |
| Nutrition | PLANNED (UI designed, backend later) |
| Meal Planner | PLANNED |
| Food / Recipe Search | PLANNED |
| Recipe Detail | PLANNED |
| Training | PLANNED (requires Hevy) |
| Workout History | PLANNED (requires Hevy) |
| Workout Detail | PLANNED (requires Hevy) |
| Exercise Progress | PLANNED (requires Hevy) |
| Progress | PLANNED |
| Weight Detail | PLANNED (backing data exists: `HealthWeightEntry`) |
| Nutrition Trends | PLANNED |
| Activity Trends | PLANNED |
| 90-Day Plan | EXISTING core form — redesign designed |
| Plan Create/Edit | EXISTING core form — redesign designed |
| Activity Alternatives | PLANNED |
| Health Sync Status | PLANNED |
| Hevy Integration settings | PLANNED (requires Hevy) |

> The Health Summary currently renders **demonstration values**, not live data. Do not describe it as live until M4.

---

## 9. Current implementation sequence

If actual repository state contradicts this, **report it rather than silently changing it**.

1. ~~Complete independent review of B1.~~ **Done** — PASS WITH NOTES, approved.
2. Fix B1 review findings H1, M1, M2 as one follow-up commit. ← *approved, next*
3. Run the real production `foreign_key_check` before deployment.
4. Add a CI safety gate (still absent).
5. Continue M2.5 security/network/backup work **one commit at a time**.
6. Complete Health provenance / timezone correctness.
7. M3 Shortcut ingestion.
8. Only after ingestion is reliable, begin UI v3 production migration / M4.
9. Hevy integration, after its provider/storage contract is designed.
10. Nutrition / recipe system.
11. Analytics.
12. AI assistant last.

---

## 10. Next task

**Exactly one task. Do not start another implementation commit until this is resolved.**

> **Fix B1 review findings H1, M1 and M2 as a single follow-up commit on `feature/health-hardening`.**
>
> Approved by Jay on 2026-09-11. Scope:
>
> - **H1** — ensure GYST's own `foreign_keys`/`busy_timeout` cannot be overridden by a caller-supplied `_pragma` in a `file:` URI.
> - **M1** — either make `:memory:` genuinely usable across pooled connections or reject/guard it, and fix the test that currently gives false assurance.
> - **M2** — remove the latent connection-exhaustion deadlock in `TestForeignKeysAreEnforcedOnEveryConnection`.
>
> Constraints: one commit; no pool tuning unless required for correctness and justified; no schema changes; no migrations; `gofmt`, `go vet`, `go test ./...`, `go test -race ./...`, `go build ./...` must all pass. Do **not** start B2/A1/auth.

---

## 11. Do not do yet

- Do **not** implement Hevy prematurely.
- Do **not** redesign the production UI while M2.5 hardening is mid-review, unless Jay explicitly requests that work.
- Do **not** implement all plan commits in one batch — one small reviewable commit at a time.
- Do **not** edit applied migrations.
- Do **not** reset `data/lifeops.db`.
- Do **not** auto-repair production foreign-key violations.
- Do **not** replace the existing Galaxy without explicit approval.
- Do **not** remove existing functionality just to match a mockup.
- Do **not** hard-code mockup sample values as production data.
- Do **not** weaken a foreign-key constraint or the Health API to make a test or a UI easier.

---

## 12. Rules for Claude and Codex

### Before every task

1. Read `docs/AI_HANDOFF.md`.
2. Verify the branch and HEAD (`git branch --show-current`, `git rev-parse HEAD`).
3. Verify the task has not already been completed.
4. Read the relevant architecture/design docs.
5. Check `git status`.
6. Do **not** rely solely on chat history.

### After every task

1. Update `docs/AI_HANDOFF.md`.
2. Record what was completed.
3. Record the commit SHA if applicable.
4. Record tests and results.
5. Record blockers.
6. Change **Next task** to exactly one concrete next action.
7. Do **not** claim work is complete unless it is verified.

### When this file is stale

If this document disagrees with Git history, **the repository wins**. Correct the document. **Never rewrite history merely to make the document match.**

### Secrets

**Never** put secrets, tokens, API keys, passwords, personal credentials or bearer tokens in this file.

---

## 13. Handoff log

Append-only. Newest last.

```
2026-09-11 — Claude — Architecture review of the Health expansion plan against the real
                      code on feature/health-core. Result: documentation/review only.

2026-09-11 — Claude — B1 implemented: SQLite per-connection PRAGMA correctness.
                      Commit: fb4f1eb
                      Tests: go test ./... and go test -race ./... green; vet/build clean.

2026-09-11 — Claude — B1 review corrections: seeded the foreign-key integrity audit and
                      completed file: URI support in Open.
                      Commit: a1dc3da
                      Tests: go test ./... and go test -race ./... green; vet/build clean.

2026-09-11 — Claude — Independent review of 3083fb9..a1dc3da.
                      Result: PASS WITH NOTES. No blocking findings.
                      Raised H1 (pragma override), M1 (:memory: per-connection isolation),
                      M2 (latent test deadlock). Approved by Jay for a follow-up commit.

2026-09-11 — Claude — Added the approved consolidated UI v3 standard and created this
                      shared handoff. Documentation only; no application behaviour changed.
                      Branch: docs/ui-v3-standards
```
