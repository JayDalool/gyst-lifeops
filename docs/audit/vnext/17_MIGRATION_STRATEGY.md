# Stage 8 — Migration Strategy (vNext)

- **Recorded:** 2026-09-27 on `audit/vnext`. Planning only.
- **Purpose:** how the current GYST becomes vNext **incrementally**, without a rewrite, without data loss, and with every step reversible.
- **Companions:** `14` (roadmap), `15` (tasks), `16` (dependencies).

**Rule of the whole migration:**
- The existing application keeps working after every merged PR.
- There is no big-bang switch, no parallel "v2 app", no feature branch that lives for weeks, and no data rename.

---

## A. CSS and component migration

**Current state:**
- Two CSS layers: legacy `web/static/css/styles.css`, and the newer `ui-*` layer (`ui-tokens.css`, `ui-components.css`, `ui-shell.css`, plus per-area `ui-today.css`, `ui-health.css`, `ui-projects.css`, `ui-galaxy.css`, `ui-courses.css`, `ui-training.css`).
- Template component partials: `ui_components.html`, `ui_icons.html`, `today_components.html`, `health_components.html`, `project_components.html`.
- A living gallery in `ui_gallery.html`.

**Target:**
- The `11` design system is implemented **inside the existing `ui-*` layer**.
- There is no new CSS framework, preprocessor or build step (DG-11).

| Step | What happens | Tasks |
|---|---|---|
| A-1 Tokens | vNext tokens are added to `ui-tokens.css` (prefixed if they collide) next to the current ones. No screen changes. | GYST-VN-018 |
| A-2 Appearance | One System/Light/Dark setting. The stored old values map `light`→Light and `dark`→Dark; no stored value → System. | GYST-VN-019 |
| A-3 Components | Grouped lists, icons/buttons, state components, sheets/popover/inspector/confirm, tap targets. Each is added to the gallery first. | GYST-VN-020, GYST-VN-021, GYST-VN-022, GYST-VN-023, GYST-VN-024 |
| A-4 Adoption per screen | A screen switches to vNext components in its own task. Its legacy selectors stay until nothing uses them. | Phases 2–3 and later UI tasks |
| A-5 Legacy removal | A legacy rule is deleted only when a `grep` over templates and JS shows no user, and it is deleted in the same PR as the last migration that used it. | GYST-VN-061 and each screen task |

**Rules:**
- A component exists in the gallery before any screen uses it.
- One screen per PR. A screen PR may add screen-specific CSS in its `ui-<area>.css`, never in `ui-components.css`. A shared need becomes a component task.
- `styles.css` is never mass-edited. It shrinks as screens migrate.
- Violet tokens are used only for identity (DD-4). Review each PR with `grep -n "xp" web/static/css/*.css`.
- The service worker cache version is bumped in every PR that changes shipped CSS or JS, so the PWA picks up the change.

## B. Screen-by-screen migration

| Current screen (template) | vNext target (`12`) | Tasks | Order note |
|---|---|---|---|
| Shell (`layout.html`, `ui_shell.html`) | Tab bar Today·Projects·Health·Learn·More; desktop sidebar | GYST-VN-025 | After components |
| `today.html` | Today: Now → Rhythm → Attention → Today → Status → Health glance | GYST-VN-029, GYST-VN-030, GYST-VN-031, GYST-VN-032, GYST-VN-033, GYST-VN-034, GYST-VN-035, GYST-VN-036, GYST-VN-069 | Sequential (H-5) |
| `more.html` | 4 groups, no empty rows | GYST-VN-026 | After nav |
| `projects_list.html` | Projects list + sub-nav | GYST-VN-039, GYST-VN-040 | After status mapping |
| `project_detail.html`, `project_form.html` | Project detail with ⋯ menu | GYST-VN-041 | After transactions |
| `roadmap.html`, `rocket_route.html`, `roadmap_form.html` | Roadmaps list + route page | GYST-VN-042 | — |
| `galaxy.html` (+ `galaxy*.js`) | Galaxy with 56 px targets, accessible preview, outer-orbit completed worlds, filter, sparse state, phone sheet | GYST-VN-044, GYST-VN-045, GYST-VN-043, GYST-VN-046, GYST-VN-047, GYST-VN-048, GYST-VN-049 | Sequential (H-6); keep Three.js |
| `courses_list.html`, `course_detail.html`, `course_form.html` | Learn | GYST-VN-050, GYST-VN-051 | Routes aliased |
| `health.html`, `health_components.html` | Health Summary | GYST-VN-052, GYST-VN-053, GYST-VN-054, GYST-VN-055 | Placeholders first |
| `nutrition.html` | Nutrition | GYST-VN-056 | — |
| `training.html` | Training (connected/disconnected) | GYST-VN-053, GYST-VN-057 | — |
| `health_progress.html` | Progress (real data only) | GYST-VN-058 | — |
| `health_plan.html` | Plan (read view first) | covered by GYST-VN-055; no redesign task until needed | Keep behaviour |
| `goals_list.html`, `goal_detail.html`, `goal_form.html` | More → Goals, with edit | GYST-VN-060 | — |
| `tasks_list.html`, `task_detail.html`, `task_form.html`, `reminders.html`, `checklist.html`, `monthly.html`, `analytics.html`, `archive.html`, `settings.html`, `plan_import.html` | vNext shell and components, same behaviour | GYST-VN-061 (one PR per screen) | Last |
| `dashboard.html` | Retired as a destination only if nothing links to it (it is superseded by Today) | GYST-VN-061 | Verify with `grep` and route logs |
| — (new) | Apps, Launcher, Attention, System, Automations, Development, Dev task, Reviews, Agents, Agent run | Phases 4–12 | Each appears only when it works (PG-1) |
| `login.html` | Unchanged apart from generic errors and throttle messaging | GYST-VN-006, GYST-VN-015 | — |

**For every screen:**
1. Implement against the prototype and the `12` field table.
2. Run the standard UI acceptance (`15`).
3. Verify in the browser at 390 and 1280 px in both modes.
4. For major mobile screens, also verify on an iPhone in Safari and the installed PWA.
5. Record the result in the task record. Private screenshots are never committed.

## C. Terminology migration

**Scope:**
- Change **user-visible strings only**: templates, flash messages, titles, aria labels, push text, calendar feed titles.
- Keep **internal names**: table names (for example the `004_quest_system` tables), columns, Go types, route paths, JSON keys and CSS hooks. Renaming persisted data would risk data loss and break old links, and it gives Jay no visible benefit.

| Retired in UI | Use | Keep internally |
|---|---|---|
| Quest, Trip | Project | Quest tables and types |
| Mission, Daily mission | Task (Rhythm for daily check-ins) | Mission tables and types |
| Destination, Planet (in copy) | World (Galaxy only) / Project | Galaxy position tables |
| "Galaxy map", Route (as a page name) | Roadmap (the rocket visual may be called the *route*) | Roadmap tables |
| Tracker (in UI) | Rhythm | Tracker tables |
| Cycle (in UI) | Plan / Past plans | Cycle columns |
| Learning deck | Learn (contains Courses) | Course tables, `/courses` routes as aliases |
| Fuel, Medal, Stars (in copy) | XP, Level, Achievement | Reward columns |
| Dev task | Issue | — |
| Job, Session (for agents) | Run | — |
| Command palette | Launcher (label: *Search*) | — |
| Inbox, Alerts | Attention | — |
| Container (in UI) | Service | — |
| Workflow (in GYST UI) | Automation | — |
| Provider (in UI) | Integration | — |
| Herder | **Agent Bridge** (`08A` D-J1 supersedes `05` §X's "Herder") | — |

**Process:**
- GYST-VN-027 does one strings-only sweep in Phase 2, with a list of the allowed personality uses that remain.
- Every later UI task follows the table.
- Reviewers `grep` for the retired terms.
- Tests that assert strings are updated from the table in the same PR. This counts as a specification change, not a weakened test.

## D. Navigation migration

| Step | Change | Compatibility |
|---|---|---|
| D-1 | GYST-VN-025 replaces NAV1 with Today · Projects · Health · Learn · More (phone) and the same five in the desktop sidebar | Every existing URL keeps resolving. The PWA start URL (`/today` or `/`) is unchanged. |
| D-2 | `scripts/check-ui-shell.cjs` NAV1 assertions are replaced by D11 assertions | Explicit spec change acknowledged by Jay in the PR (U-9); the checks stay at least as strict |
| D-3 | Learn tab → `/learn`, an alias of the courses list until GYST-VN-050 makes it canonical; `/courses*` keeps working (redirect or alias) | Calendar feed URLs (`/calendar/feed/{token}`) are untouched |
| D-4 | More loses rows that duplicate tabs; rows appear only for working destinations | Removed rows' URLs still work |
| D-5 | Projects gains the sub-nav; the Development segment appears in Phase 8, Agents in Phase 10 | — |
| D-6 | Desktop Operate group (System, Apps, Automations) appears when each exists | — |
| D-7 | Attention: no tab and no permanent sidebar item (C1); reached from Today, the badge, the launcher, More and push deep links | — |

Navigation changes land in **one PR per step**. They are never mixed with screen content.

## E. Database changes

**Principles** (from `AGENTS.md`, `docs/RULES.md` and the working agreement):
- Migrations are additive only, with a `YYYYMMDDHHMMSS_name.sql` prefix that CI checks for uniqueness.
- An applied migration is never edited. The database is never reset. Foreign keys are never weakened.
- Every migration is applied to a copy of the latest backup before deploy, followed by `integrity_check` and `foreign_key_check`.
- New instants are UTC RFC 3339 with `Z`; local day buckets are stored as explicit dates.
- There are no automatic data repairs. Historical wrong-day rows are a decision (GYST-VN-011), not a migration.

**Planned additive migrations:**

| Migration (provisional name) | Adds | Task | Existing data impact |
|---|---|---|---|
| `audit_events` | Audit table + indexes | GYST-VN-005 | None |
| `attention_conditions` (+ owner state for computed items) | Stored conditions, snooze/dismiss | GYST-VN-067 | None |
| `sessions_created_at` | `created_at` on sessions (existing rows: treated as created at migration time) | GYST-VN-077 | Existing sessions get at most 90 more days |
| `sessions_reauth` / `action_nonces` | Step-up window and single-use nonces | GYST-VN-079 | None |
| `machine_credentials` | Hashed tokens, scopes, namespaces | GYST-VN-081 | None (the Health token keeps working until migrated deliberately) |
| `processed_events` | Idempotency | GYST-VN-083 | None |
| `github_*` cache tables | Repos, issues, PRs, check states (cache/reference only) | GYST-VN-101, GYST-VN-102 | None; can be dropped and rebuilt from GitHub (the drop would be a separate, reviewed migration) |
| `automations` snapshot | n8n workflow status | GYST-VN-110 | None |
| `agent_runs`, `agent_approvals` | Runs, heartbeats, approval records | GYST-VN-116, GYST-VN-129 | None |
| Uptime/System snapshots | Provider snapshots | GYST-VN-090, GYST-VN-093 | None |

**Not planned:**
- renaming Quest/Mission tables
- dropping unused tables (S2-21); later, and only with Jay's approval
- changing existing day columns
- training schema changes (pending GYST-VN-059)

## F. Integration rollout

Each integration follows the same five steps:

1. **Credential and identity:** a least-privilege credential created by Jay and stored in a `0600` secret file. It never goes in the repo, the DB or logs.
2. **Read-only provider:** a fixed host, timeouts and size caps. The snapshot is stored with a freshness time.
3. **Sanitized UI:** public projection types are separate from tailnet detail types. Stale data shows "unavailable", never "healthy".
4. **Attention producers:** conditions resolve automatically.
5. **Writes/actions (only where approved):** behind a config flag that ships **off**, re-auth per `14` §E.3, audit, and CP checkpoints.

| Integration | Read phase | Write/control phase | Identity | Kill switch |
|---|---|---|---|---|
| Uptime Kuma | 7 | — | Read-only Kuma access | Config flag |
| Server Provider | 7 | 13 (via the action runner) | Provider token; runner credential | Stop the service |
| GitHub | 8 | 11 (Issues RW), 12 (agent identity: Contents/PR RW) | GYST App (read → Issues RW); separate agent App/bot | Config flag; uninstall the App |
| n8n | 9 (events) | 13 (Run Now) | GYST `automation:event` credential; per-app Header-Auth; no full n8n key | Revoke the credential |
| Agent Bridge | 10 (heartbeat) | 12 (dispatch) | `gyst-bridge`/`gyst-agent` accounts; `agent:report`/`agent:commands` scopes | Disable dispatch in bridge config; stop the unit |
| Logs | 14 | — | Provider Level 3 | Disable Level 3 |
| Console | 15 | — | Tailscale SSH ACLs | Remove the link |

**Temporary PAT** (GYST-VN-097), only if chosen:
- repository: `gyst-lifeops` only
- permissions: read only
- expiry: ≤ 30 days, secure storage
- replaced by the App before GitHub write and before expiry
- revocation verified by a 401 response and recorded

## G. Rollback philosophy

1. **Every PR is revertible on its own.** Keep PRs small and never couple a migration with a behaviour change that depends on it in the same deploy when it can be avoided. The migration ships first, then the code that uses it.
2. **Additive schema means code rollback is safe.** Old code ignores new tables and columns. The table is never dropped as part of a rollback.
3. **Flags over reverts for integrations.** Writes, dispatch, actions and Run Now ship disabled and are turned on after the checkpoint. Rolling back means turning the flag off.
4. **Data rollback is a restore, and a restore is a runbook.** Data is never hand-edited. Restores follow GYST-VN-013, with a fresh backup taken first.
5. **Operations have written undo steps.** Every NOT AGENT-ELIGIBLE task records "before" state (modes, settings summary, not secrets) so it can be undone.
6. **Credentials roll forward.** A compromised or wrong credential is revoked and replaced, never reused.
7. **Design rollback is per screen.** A screen that regresses reverts to its previous template while the rest of vNext stays.

## H. Deployment checkpoints

| Checkpoint (`14` §I) | Deploy scope | Verify after deploy |
|---|---|---|
| CP-1 Foundation | Step 0 changes, each deployed alone where PRIVILEGED (Compose logging/healthcheck) | Login on the phone, `/healthz`, backup timer run, throttle message, log line format |
| CP-2 / CP-3 UI + Today | UI PRs can be grouped **per deploy** when they touch different screens | Real iPhone PWA: Today, tabs, More; light/dark via iOS setting |
| CP-4 Life areas | Screen PRs | Health invariants on real data; Galaxy on device |
| CP-5 Hardened base | Dual listener, tailnet bind, non-root, headers: **each alone** | Public and tailnet hosts from two devices; LAN access refused; CSP console clean; backups still run |
| CP-6 Read-only integrations | Providers with flags on | Public System page field check; GitHub client read-only; webhook redelivery |
| CP-7 Public higher-impact writes | Cloudflare Access (Jay) | All 6 flows; machine endpoints still reachable |
| CP-8 Agent go-live | Bridge enabled | 3 supervised runs |
| CP-9 Controlled operations | Runner and actions | Each action once, supervised |

**Every deploy:**
1. Backup first.
2. Record the SHA.
3. Check `/healthz`.
4. Do a phone check of the public host.
5. Write a one-line result in the task ledger.

## I. Compatibility considerations

| Area | Consideration | Handling |
|---|---|---|
| URLs and bookmarks | Tabs and areas change; Courses → Learn | No route removed; aliases/redirects; `dashboard` retired only when unused |
| PWA / service worker | Cached old CSS/JS/templates after deploy | Bump the cache version in every asset-changing PR; test the installed PWA after nav and appearance changes |
| Theme preference | Old `data-gyst-theme` values | Mapped to Light/Dark; absent → System |
| Calendar feeds | Token URLs are auth-exempt and used by external calendars | Never moved or renamed; exempt list preserved in the listener split |
| Apple Health Shortcut | Posts to `/api/health/sync` with the Health token | Path and token keep working across the listener split. It moves to a machine credential only by a deliberate, coordinated task; the date bound (GYST-VN-085) is announced to Jay before deploy. |
| Sessions | New absolute cap and re-auth | Existing sessions stay valid (created_at = migration time); no mass logout except a deliberate revoke-all |
| Password hash | Cost change | Jay regenerates; old hash keeps working until then |
| Cloudflare Access | Could block machine endpoints and PWA flows | Exact-path bypasses, validated on device (GYST-VN-125) |
| Tests | UI shell tests pin NAV1; string assertions use retired terms | Updated as explicit spec changes in the same PR, never deleted to pass |
| Old ledger | `docs/tasks/` items (A1, A2, M3, OPS1) predate vNext | Reconciled in GYST-VN-001; OPS1 is carried as GYST-VN-017 |
| Other agents / branches | Claude Code and Codex may work in other worktrees | One task per branch; hotspot groups serialized (`16` §D); rebase on the base branch before review, never force-push shared branches |
| Timezone | Local-day fix changes where late-evening entries land | Only new entries; history unchanged pending GYST-VN-011 |
