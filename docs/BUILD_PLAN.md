# GYST Build Plan — Daily Command Center

Owner: Jay Dalool
Status: Phases 0-7 and safe Phase 10 cleanup complete; Phase 8 optional; Phase 9 blocked on HTTPS
Date: 2026-08-27

This document answers every open question from the design review and defines the
build order. It supersedes the proposal it responds to where the two disagree.

---

## 0. Mandate for the implementing agent

You have full authority in this repository. Specifically, you may:

- add, rename, move, and delete files;
- add migrations, tables, columns, and indexes;
- add routes, handlers, services, repositories, models, and templates;
- restructure navigation and templates;
- add a vendored frontend dependency (see Phase 6);
- refactor `internal/handlers/handlers.go` (1,943 lines) into smaller files;
- update `README.md` and `docs/ARCHITECTURE.md` to match what you build.

Do not stop to ask permission for ordinary implementation decisions. Make the
call, note it in the commit message, and keep going. Ask only when a decision
would change the product shape in a way this document does not cover.

Four hard guardrails, because they protect data you cannot reconstruct:

1. **Migrations are additive.** No `DROP TABLE`, no `DROP COLUMN`, no destructive
   `UPDATE` without a `WHERE`. The pattern in `migrations/010_project_item_rewards.sql`
   is the model: `ALTER TABLE ... ADD COLUMN` with defaults that keep every
   existing row valid.
2. **Do not touch quest reward integrity.** `migrations/009_quest_reward_integrity.sql`
   installs unique partial indexes that stop duplicate XP on retry/reopen. Nothing
   in this plan may award XP, and nothing may weaken those indexes.
3. **Hide routes, don't delete them.** Phase 1 removes links from navigation. The
   handlers stay registered and reachable by URL until Phase 10 confirms the
   replacement works.
4. **`data/lifeops.db` is real data.** Never reset, reseed, or delete it. Test
   against a temp database, which is what the existing tests already do.

---

## 1. Decisions — all questions answered

| # | Question | Decision |
|---|---|---|
| 1 | Prayer calculation method | **ISNA** as the default, configurable in Settings |
| 2 | Asr method | **Standard / Shafi** default, **Hanafi** toggle in Settings |
| 3 | Athan audio | **Ship a short chime we own as the default.** Add a Settings field pointing at a user-supplied file in `web/static/audio/`. Do not block on sourcing a full Athan recording. |
| 4 | Galaxy technology | **Three.js**, vendored locally and pinned. No npm, no build step. |

Additional decisions this plan makes, which were not asked but are needed:

| # | Question | Decision |
|---|---|---|
| 5 | Prayer time source | **Compute locally in Go.** AlAdhan is optional cross-check only, never a runtime dependency. |
| 6 | Location | **Winnipeg, Manitoba** — lat `49.8951`, lon `-97.1384`, tz `America/Winnipeg`. Configurable. |
| 7 | External PM links | **Build them.** Manual-link phase first, provider APIs later. See Phase 2. |
| 8 | External links and XP | **Never award XP.** References only, display layer, no effect on project completion. |
| 9 | New Go dependencies | **Zero.** `go.mod` currently has one real dependency (`modernc.org/sqlite`). Keep it that way; write the prayer algorithm in-repo. |
| 10 | Root route | `/` and `/today` render **the same page**. |

### On decision 3, in detail

The blocker on Athan audio was never the file — it is licensing plus browser
autoplay policy. Browsers block audible autoplay until the user has interacted
with the page, so Today needs one deliberate "Enable prayer alerts" button
regardless of which recording is used. Shipping a short owned chime unblocks the
entire notification phase today, and the recording can be swapped at any time
through Settings without a code change.

### On decision 5, in detail

`docs/ARCHITECTURE.md` opens by calling this app local-first. Making the most
prominent element of the new landing page depend on a third-party HTTP call
contradicts that, and caching does not cover first run, a new month, or a
machine that has been offline for weeks.

Local computation is the standard solar-position algorithm — roughly 150 lines,
no dependency, microseconds per call. It also **removes the need for a prayer
time cache table entirely**, which deletes a whole slice of the original plan.
`services.Service` already carries `Now func() time.Time` (`internal/services/lifeops.go:14`),
so prayer time tests are deterministic for free.

---

## 2. What the original proposal got right

Adopt these as written:

- Four primary destinations: **Today, Galaxy, Add, More**.
- Today is the root page and the default landing surface.
- Merge Dashboard into Today; merge Projects and Roadmaps into Galaxy.
- Push Courses, Monthly, Analytics, Health, Archive, All Tasks under More.
- Remove the seven-link footer.
- Today section order: next prayer, five prayers, Athkar, missions, study and deadlines.
- Galaxy needs a keyboard-navigable list, a reduced-motion mode, and a
  conventional list fallback. No information may exist only in 3D.
- Project checklist pages stay conventional. No 3D inside them.

---

## 3. What changes, and why

**The proposal omitted external PM links entirely.** That is the feature that
makes this a daily driver — it is what stops the context switch into Jira and
Slack. It is now Phase 2, immediately after navigation, because it is small and
depends on nothing else.

**The Galaxy blocker is data, not rendering.** `internal/models/models.go`
already defines `SolarSystem`, `RocketRoute` with `galaxy`/`solar`/`compact`
modes, and `RouteNode` carrying `LeftPct`/`TopPct`/`PathPct`. Those are 2D
percentages of a box, recomputed on every render. Ported to 3D as-is, **planets
would move every time a project is added**, destroying the spatial memory that
makes a galaxy worth building. Persistent 3D positions come first (Phase 5),
before any Three.js work (Phase 6).

**Web Push is gated on HTTPS, not on the Athan file.** The app runs at
`localhost:8080` via `go run .`, and `docs/ARCHITECTURE.md` states deployment is
not yet tracked. The untracked `Dockerfile` and `docker-compose.yml` are the
start of that. TLS behind a reverse proxy is the real prerequisite for Phase 9.
Note also: on iOS, web push requires the PWA be installed to the home screen,
and **Athan audio with the browser closed will not work** — a notification sound
is the realistic ceiling. A true Athan on a locked phone is the native SwiftUI
app already described in `README.md`.

---

## 4. Build order

Ship each phase as its own commit. Each phase must leave the app runnable.

### Phase 0 — Clean the tree

The working tree has an uncommitted `design-v2/` to `docs/design/` move plus new
`Dockerfile`, `docker-compose.yml`, and `docs/ARCHITECTURE.md`. Commit that
first. Do not start a refactor on top of an unstaged file move.

### Phase 1 — Navigation reduction

Lowest risk in the plan. All page routing is one switch statement at
`internal/handlers/handlers.go:225`, so this is a template change.

- Rewrite the `<nav>` block in `web/templates/layout.html` (lines 18-25)
  to four items: Today, Galaxy, Add, More.
- Delete the `<footer class="foot">` block (lines 32-40).
- "Add" is the existing add menu: task, project, or course.
- "More" is a drawer or menu holding Courses, Monthly, Analytics, Health,
  Archive, All Tasks, Goals, Reminders, Settings.
- In `ServeHTTP`, route `""` to `h.today` instead of `h.dashboard`. Keep
  `h.dashboard` registered and reachable.
- "Galaxy" points at `/galaxy`, which for now renders the existing projects view.
  Phase 6 replaces its body.

Acceptance: every old URL still resolves; nothing in the primary nav except the
four items; `/` and `/today` render identically.

### Phase 2 — External PM links, manual phase

New migration `migrations/011_external_links.sql`:

```sql
CREATE TABLE IF NOT EXISTS external_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('task', 'project', 'project_item')),
    entity_id INTEGER NOT NULL,
    provider TEXT NOT NULL CHECK (provider IN ('jira', 'slack', 'linear', 'asana', 'github', 'notion', 'other')),
    external_key TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    status_text TEXT NOT NULL DEFAULT '',
    sync_source TEXT NOT NULL DEFAULT 'manual' CHECK (sync_source IN ('manual', 'api')),
    last_synced_at TEXT,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_external_links_entity
ON external_links (entity_type, entity_id);
```

The `uuid` / `sync_version` / timestamp / `last_modified_device_id` columns match
the convention every other table follows, so a future sync client works.

Behavior:

- Paste a URL, and the app infers `provider` and `external_key` from the host and
  path. Jira browse URLs yield `PROJ-123`; GitHub issue URLs yield `owner/repo#12`;
  Slack archive URLs yield the channel and timestamp. Unrecognized hosts fall back
  to `other` with the hostname as the title.
- The user types a one-line summary. That is the whole input.
- Links render as a compact chip on the task, project, or mission detail page:
  provider icon, key, summary, and status text. `target="_blank"` with
  `rel="noopener noreferrer"`.
- Full CRUD: attach, edit summary, detach.

Non-negotiable scoping rules:

- **No XP, ever.** External links do not award stars and do not count toward
  project completion. See guardrail 2.
- **`proof_url` is a different thing.** `Task` and `ProjectItem` already carry
  `ProofURL` (`internal/models/models.go:3`, `:64`), which is proof of completion.
  Do not overload it.

New files: `internal/repository/external_links.go`,
`internal/services/external_links.go`, `models.ExternalLink`, plus handler cases.

Acceptance: a Jira URL pasted onto a mission renders as a clickable chip with its
key parsed correctly, and opens Jira in a new tab. Project XP totals are unchanged.

### Phase 3 — Prayer engine

- `internal/services/prayer.go` — solar position and prayer time calculation.
  Support ISNA, MWL, Umm al-Qura, Egyptian, and Karachi as method constants, with
  ISNA the default. Support Shafi and Hanafi Asr.
- Settings live in the existing `app_settings` key/value table via
  `repository.GetSetting` / `SetSetting` (`internal/repository/settings.go`). Keys:
  `prayer.latitude`, `prayer.longitude`, `prayer.timezone`, `prayer.method`,
  `prayer.asr_method`, and `prayer.adjust.{fajr,dhuhr,asr,maghrib,isha}` holding
  signed minute offsets.
- Defaults seed to Winnipeg per decision 6.
- `NextPrayer(now)` returns the upcoming prayer, its time, and the duration
  remaining, correctly rolling over to tomorrow's Fajr after Isha.
- Tests use the injected `Service.Now` and assert against known-good Winnipeg
  times for a solstice, an equinox, and a DST boundary day.

No cache table. No network call.

### Phase 4 — Prayer UI on Today

Extend `handlers.TodayData` (`internal/handlers/handlers.go:30`) and render, in
this order, above everything else:

1. **Next prayer hero** — name, exact time, live countdown, "Winnipeg", alert state.
2. **Five prayer tracker** — Fajr through Isha, current prayer emphasized. Wire to
   the tracker items that already exist: `prayer_fajr`, `prayer_dhuhr`,
   `prayer_asr`, `prayer_maghrib`, `prayer_isha` (`migrations/002_trackers.sql:40-44`).
   `models.DailyMissionData.PrayerItems` already carries them.
3. **Athkar** — `athkar_sabah`, `athkar_masaa`, same mechanism.
4. **Today's missions** — actionable only. Missions belonging to `locked` projects
   are excluded.
5. **Study and deadlines** — compact secondary sections, from the existing
   `models.TodayLearning` and deadline rows.

Countdown ticks client-side in `web/static/js/app.js`. Prayer times are rendered
server-side into data attributes; the browser does not recompute them.

Add prayer settings controls to `web/templates/settings.html`: method, Asr method,
per-prayer minute adjustments, latitude, longitude.

### Phase 5 — Galaxy data layer

Do this before any 3D code.

Migration `012_galaxy_positions.sql` adds `pos_x`, `pos_y`, `pos_z` REAL columns
to `projects`, nullable. When a project has no stored position, derive one
deterministically by hashing its existing `uuid` into a distribution, then persist
it. Positions never change after first assignment unless the user moves a planet.

Also derive per project:

- **size** from importance — child mission count, or project kind;
- **state** from the canonical statuses in `docs/QUEST_SYSTEM.md`: `active`,
  `upcoming`, `locked`, `done`, `archived`;
- **moons** from chapters, which are child projects;
- **progress** from the existing `ProjectWithProgress`.

Expose it as `GET /galaxy/data` returning JSON. Build and test this endpoint on
its own — the renderer is then a pure consumer.

### Phase 6 — Galaxy renderer

- Vendor a single pinned `three.module.js` into `web/static/js/vendor/`. Record
  the exact version in `docs/ARCHITECTURE.md`. **No npm, no bundler** — this
  preserves the no-build-step guarantee the architecture doc makes.
- `web/static/js/galaxy.js` — full-screen star field, projects as textured
  planets, chapters as moons, size by importance, done planets lit, locked
  planets distant and dimmed.
- Desktop: vehicle follows the cursor with easing, cursor motion pans the camera,
  clicking a planet flies toward it, clicking again or pressing "Open project"
  navigates to the normal checklist page.
- Mobile: drag to pan, tap to select, tap "Open project" to enter. No
  cursor-follow on touch devices.
- Accessibility, all three required:
  - a keyboard-navigable list of planets, tab-reachable, Enter to open;
  - `prefers-reduced-motion` disables the fly-to animation and the drifting camera;
  - a conventional project list fallback if WebGL is unavailable — the existing
    `/projects` view serves this.

### Phase 7 — In-page notifications and Athan

- One deliberate "Enable prayer alerts and Athan" button on Today. It requests
  notification permission and, in the same user gesture, primes the audio element
  so later playback is allowed.
- A "Test sound" button next to it.
- While the page is open: countdown updates, a browser notification fires at each
  prayer time, and the alert sound plays.
- Settings: `prayer.alert_sound` (`chime` or `athan`) and `prayer.athan_file`
  pointing at a file in `web/static/audio/`.

### Phase 8 — External links, provider APIs (optional)

Only if you want live status. Everything works without it.

- `internal/services/providers/` with `type Provider interface { Fetch(ctx, link) (LinkSnapshot, error) }`.
- Jira: `GET /rest/api/3/issue/{key}?fields=summary,status,assignee,updated`.
- GitHub: the issues API. Slack: `conversations.history` plus a permalink.
- The snapshot writes `title`, `status_text`, `last_synced_at` back into the same
  `external_links` row and sets `sync_source = 'api'`. **The UI never changes** —
  it degrades to the Phase 2 manual view whenever a token is missing or a fetch
  fails.
- Read tokens from **environment variables**, not `app_settings`. That table
  stores plaintext, and the `ALLOW_PUBLIC_CALENDAR_FEEDS` env pattern in
  `internal/app/` is the established precedent for this kind of setting.
- Fetch on a timer or on page load with a short TTL. Never block a page render
  on a third-party call — always render the stored snapshot immediately.

### Phase 9 — PWA and Web Push

Blocked on HTTPS. Do Phase 0-8 first.

- Web app manifest, icons, service worker.
- VAPID keys, push subscription storage, a backend scheduler that sends a push at
  each prayer time.
- Requires TLS. Extend the existing `Dockerfile` / `docker-compose.yml` with a
  reverse proxy.
- Set expectations in the UI: closed-browser Athan audio is not achievable on
  mobile. Deliver a notification with a sound, not a full Athan.

### Phase 10 — Cleanup

- Remove the old dashboard and 2D route map code **only after** the replacements
  are verified in a browser.
- Consider splitting `internal/handlers/handlers.go` into per-domain files.
- Update `README.md` and `docs/ARCHITECTURE.md` — the UI Structure and Features
  sections in the README describe the old four-card homepage and the seven-link
  More drawer, and will both be wrong after Phase 1.
- Test matrix: desktop, mobile, keyboard-only, reduced-motion, notifications
  granted and denied, audio enabled and blocked, WebGL unavailable, and offline.

---

## 5. Verification

**There is no Go toolchain installed on this host.** Go exists only inside
Docker — the `Dockerfile` builds from `golang:1.25-bookworm`. Running `go test`
directly will fail with `go: command not found`. Use the wrapper instead:

```sh
./scripts/check.sh          # gofmt + go vet + go test, in a container
./scripts/check.sh test     # tests only, faster
```

It mounts the working tree into a throwaway `golang:1.25-bookworm` container and
caches modules in a named volume. Run it before every commit. Baseline as of
2026-08-28: all packages pass.

The app also already runs in Docker as the `gyst-lifeops` container, published on
**http://localhost:3005**. After changing Go code, rebuild it to see the change:

```sh
docker compose up -d --build
```

Static assets and templates under `web/` are baked into the image, so they need
the same rebuild.

Migration tests must cover both a fresh database and a populated pre-migration
database upgrading through the newest migration. That pattern already exists in
`internal/repository/` — follow it for `011` and `012`.

---

## 6. Summary of the order

| Phase | Work | Depends on |
|---|---|---|
| 0 | Commit the working tree | — |
| 1 | Navigation reduction, Today as root | 0 |
| 2 | External PM links, manual | 1 |
| 3 | Prayer engine, computed locally | 0 |
| 4 | Prayer UI on Today | 1, 3 |
| 5 | Galaxy data and 3D positions | 1 |
| 6 | Galaxy renderer, Three.js | 5 |
| 7 | In-page notifications and Athan | 4 |
| 8 | Provider APIs | 2 |
| 9 | PWA and Web Push | 7, HTTPS deploy |
| 10 | Cleanup and docs | 6, 7 |

Start with Phase 0, then Phase 1. Phases 2 and 3 are independent of each other
and can be built in either order.
