# Stage 8 — Implementation Roadmap (vNext)

- **Stage:** 8 of the vNext reassessment (final planning stage)
- **Recorded:** 2026-09-27 on `audit/vnext`
- **Status:** plan only. No code, configuration, database, GitHub, Cloudflare, n8n or agent setting was changed. No GitHub Issues were created.
- **Binding inputs:** `05A_PRODUCT_DECISIONS.md`, `08A_INTEGRATION_DECISIONS.md`, `10A_SECURITY_DECISIONS.md`, `13A_DESIGN_DECISIONS.md`. Specifications: `09`–`13`.
- **Galaxy:** `17A_GALAXY_PRESERVATION.md` (binding product constraint added after Stage 8)
- **Companion files:** `15_TASK_BACKLOG.md` (145 tasks, `GYST-VN-001…`), `16_DEPENDENCY_MAP.md`, `17_MIGRATION_STRATEGY.md`.

Task IDs are provisional. They become GitHub Issue numbers only when Jay approves creating the issues.

---

## A. Strategy

**Principle: safe foundation first, then visible product progress, then integrations in the approved order, then write and control.**

1. **Fix what is wrong or unsafe today before adding capability.**
   - The Step 0 package from `10A` O-4 comes first.
   - So does the local-day correctness bug, which is a separate high-priority product fix.
   - Nothing that talks to GitHub, n8n, the server or agents starts before M0 is done.
2. **Give Jay visible progress early.**
   - The UI foundation and Today work (M1) can run alongside the Jay-only parts of M0.
   - Some small, independent UI truth fixes are pulled forward: hide the Health placeholders, fix the Hevy state, and enlarge the Galaxy targets.
3. **Keep the existing architecture.**
   - The stack stays as it is: a single Go binary, `net/http`, handler → service → repository → SQLite, server-rendered templates, vanilla JS and plain CSS, and Three.js for Galaxy.
   - No framework, no build system and no new component library.
   - The vNext design system extends the existing `ui-*` CSS layer.
4. **Integrations follow the approved order** (`08A` SEQ), refined here:
   - Apps + Launcher → Attention → machine auth and hardening → Uptime/System (read) → GitHub (read) → n8n events → agent visibility → GitHub (write) → agent dispatch → system actions / Run Now → logs → console link.
   - The order is refined, not reversed.
5. **Read before write, public-sanitized before tailnet detail, tailnet before public control.**
   - Every write or control capability is behind its `09` §O gates.
6. **Humans own operations.**
   - Production changes, GitHub/Cloudflare/n8n settings, credentials and account creation are NOT AGENT-ELIGIBLE tasks for Jay.
   - Agents prepare runbooks and scripts for them.
7. **Small, reviewable changes.**
   - One task maps to one focused PR (or a short series).
   - Backend correctness changes never ride inside visual PRs.
   - Nothing is batched across hotspot files (`16` §D).

## B. Milestones

| Milestone | Phases | Tasks | P0/P1/P2/P3 | User-visible outcome |
|---|---|---|---|---|
| M0 Safe foundation | 0 | 17 | 5 / 11 / 1 / 0 | GYST data is backed up off-host and restorable, the public login resists guessing, "today" means Jay's local day everywhere, and failures are visible in logs and /healthz. |
| M1 vNext core experience | 1, 2 | 20 | 0 / 17 / 3 / 0 | The app looks and navigates like vNext: Today · Projects · Health · Learn · More, a calm Today with Now, Rhythm and no duplicate blocks, a simpler More, and System/Light/Dark appearance. |
| M2 vNext Life areas | 3 | 27 | 0 / 10 / 13 / 4 | Projects, Galaxy, Learn, Health and Goals use the vNext patterns and plain terminology; Health shows only what really works; Galaxy is usable on a phone. |
| M3 Command centre | 4, 5 | 9 | 0 / 5 / 3 / 1 | One launcher (Go · Do · Capture) reaches everything; Apps opens external tools; one Attention model tells Jay what needs him, from GYST's own data. |
| M4 Read-only integrations | 6, 7, 8, 9 | 41 | 0 / 9 / 30 / 2 | GYST shows sanitized server health, GitHub issues/PRs/reviews in Development, and n8n automation status, all read-only, on a hardened two-listener foundation. |
| M5 Agent-assisted development | 10, 11, 12 | 25 | 0 / 0 / 23 / 2 | Jay can write GitHub issues from GYST, mark a NORMAL issue ready for an agent after re-auth, watch the run, review the PR in GitHub and verify-and-close from GYST. |
| M6 Controlled operations | 13, 14, 15 | 6 | 0 / 0 / 0 / 6 | From the tailnet, with re-auth and confirmation, Jay can run a backup, restart an allow-listed service, run an approved n8n workflow and read bounded redacted logs. |

Milestones are defined by the user-visible outcome, not by dates. A milestone is done when all of its P0/P1 tasks are done and its stop/go checkpoint (§I) passes. P2/P3 tasks may continue into the next milestone if they don't block it.

## C. Phase sequence

| Phase | Name | Milestone | Tasks | IDs |
|---|---|---|---|---|
| 0 | Foundation | M0 | 17 | GYST-VN-001–017 |
| 1 | vNext UI foundation | M1 | 7 | GYST-VN-018–024 |
| 2 | Today + navigation + cleanup | M1 | 13 | GYST-VN-025–036 + 143 |
| 3 | Projects / Galaxy / Learn / Health migrations | M2 | 27 | GYST-VN-037–061 + 144, 145 |
| 4 | Apps + Launcher | M3 | 4 | GYST-VN-062–065 |
| 5 | Attention | M3 | 5 | GYST-VN-066–070 |
| 6 | Machine auth / integration foundation | M4 | 18 | GYST-VN-071–088 |
| 7 | Uptime / System read-only | M4 | 7 | GYST-VN-089–095 |
| 8 | GitHub read-only / Development | M4 | 13 | GYST-VN-096–108 |
| 9 | n8n events | M4 | 3 | GYST-VN-109–111 |
| 10 | Agent visibility | M5 | 6 | GYST-VN-112–117 |
| 11 | GitHub write | M5 | 8 | GYST-VN-118–125 |
| 12 | Agent dispatch | M5 | 11 | GYST-VN-126–136 |
| 13 | Controlled System / n8n actions | M6 | 4 | GYST-VN-137–140 |
| 14 | Logs | M6 | 1 | GYST-VN-141–141 |
| 15 | Console link (much later) | M6 | 1 | GYST-VN-142–142 |

**Refinements to the suggested phase shape** (all driven by dependencies):

| Refinement | Reason |
|---|---|
| Navigation shell and More are in Phase 2, not Phase 1 | They consume the Phase 1 components. Phase 1 is components only, with no screen changes. |
| Terminology strings and removing writes from GET requests are in Phase 2 ("cleanup") | Terminology touches many templates, so it lands before the per-screen migrations. The GET writes fix is a prerequisite for the single status mapping in Phase 3. |
| Must-before-write hardening is in Phase 6 | Covers dual listener, tailnet bind, headers, non-root, session cap, revoke-all, step-up, bcrypt, deploy recording. These are `09` §P "MUST BEFORE WRITE/CONTROL" items. Placing them with machine auth keeps Phases 7–9 read-only work on a hardened base. |
| The packaging decision (bridge/provider/runner repository) is in Phase 6 | The Server Provider in Phase 7 needs a home. |
| The Server Provider is in Phase 7, before GitHub | This follows the suggested shape (Uptime/System read at 7). `08A` placed n8n events before the provider; both are read-only and independent, so the swap carries no security effect. |
| Branch protection is in Phase 8 (it can be done any time after the baseline) | It is useful immediately and is required before agent PRs (G12). |
| Cloudflare Access is validated in Phase 11 | It must precede higher-impact public writes. The first of these is public `agent:ready` (Phase 12). Phase 11's own writes (create/edit/label/comment) are public-authenticated reversible writes under `09` §G.2. |
| Agent limits are enforced from the first dispatch | The dispatch task (9 of 11) hard-enforces 1 job, the 45-minute timeout and 0 retries. The limits task (11 of 11) turns the whole table into configuration and adds cleanup. Production dispatch stays disabled until the go-live task. |

## D. User-visible outcome per phase

| Phase | What Jay notices |
|---|---|
| 0 Foundation | Mostly invisible. Late-evening check-ins land on the right day. Repeated wrong passwords get blocked. Daily backups appear, with a copy off the server. Errors show a clean message with a request id. |
| 1 vNext UI foundation | Appearance follows iOS light/dark automatically (System/Light/Dark). Tap targets in the shell are comfortable. Screens otherwise look the same. |
| 2 Today + navigation + cleanup | New tab bar: Today · Projects · Health · Learn · More. Today opens with **Now**, then the **Rhythm** strip. There are no duplicate blocks, overdue items are clear, and a small Lv chip appears. More is short. Mission/Quest/Trip wording is gone. |
| 3 Life areas | Projects has four peer areas. Projects lists active work first. Roadmaps are scannable. Galaxy is tappable on the phone, completed worlds can be hidden, and the preview closes properly. Learn answers "what next?". Health shows only real data and has no fake buttons. Goals can be edited. |
| 4 Apps + Launcher | ⌘K or the search icon reaches anything. Apps opens external tools. Commands and capture are explicit. |
| 5 Attention | One "needs you" list, reached from Today, a badge and the launcher. Snooze and dismiss are available. Critical push notifications are available. |
| 6 Machine auth / integration foundation | Little is visible. Security settings appear on the tailnet (sessions, credentials, audit). Re-auth appears for sensitive actions. |
| 7 Uptime / System read-only | System page: services, backups, certificates and rounded resource use. Detail is shown on the tailnet only. Problems show on Today; normal operation shows only a quiet chip. |
| 8 GitHub read / Development | Development shows GitHub issues, reviews and repositories inside Projects. Review and CI problems reach Attention. |
| 9 n8n events | Automations page. Failed automations raise Attention. |
| 10 Agent visibility | Agents page: Claude/Codex availability and bridge heartbeat. There are no runs yet. |
| 11 GitHub write | Create and edit issues, change labels and status, assign an agent, comment, and verify & close (with re-auth), all from the phone. |
| 12 Agent dispatch | "Ready for agent" on NORMAL issues (with re-auth). Runs are visible and can be stopped. Agent PRs get an independent agent review. Results reach Reviews. |
| 13 Controlled actions | From the tailnet: run a backup, restart an allow-listed service, run an approved automation. |
| 14 Logs | From the tailnet, after re-auth: bounded, redacted service logs. |
| 15 Console link | From the tailnet: a link to Tailscale SSH. |

## E. Security gates

Gates from `09` §O are satisfied by these tasks:

| Gate | Control | Satisfied by | Must be done before |
|---|---|---|---|
| G1 | Login throttling + failed-login audit | GYST-VN-006, GYST-VN-007 | Any external integration (end of M0) |
| G2 | Audit log | GYST-VN-005, GYST-VN-084 | Machine auth (Phase 6) |
| G3 | Machine credentials | GYST-VN-081 | Kuma provider, n8n events, GitHub webhooks, bridge |
| G4 | File permissions | GYST-VN-009 (**re-verified on the day before** GYST-VN-114) | Any agent or bridge account (O-1, binding) |
| G5 | Backups + tested restore | GYST-VN-010, GYST-VN-012, GYST-VN-013 | Any external integration; system actions |
| G6 | `/healthz`, logging, rotation, monitors | GYST-VN-014, GYST-VN-015, GYST-VN-016, GYST-VN-089 | Uptime/System |
| G7 | Step-up + action nonce + confirmation | GYST-VN-079 | GitHub `agent:ready`, credential admin, system actions, Run Now, logs |
| G8 | Dual listener + tailnet bind | GYST-VN-073, GYST-VN-074 | Any tailnet-only view or action |
| G9 | UTC instants | GYST-VN-004 | Attention, audit, events |
| G10 | Non-root GYST container | GYST-VN-076 | Server Provider deployment (recommended), dispatch |
| G11 | Agent accounts + §I controls | GYST-VN-114, GYST-VN-115, GYST-VN-126, GYST-VN-127 | Agent dispatch |
| G12 | Branch protection | GYST-VN-099 | First agent PR |
| G13 | Fixed-host outbound credentials | GYST-VN-087, GYST-VN-090, GYST-VN-101 | Each outbound integration |
| G14 | Service-layer writes + sanitized errors | GYST-VN-015, GYST-VN-038, GYST-VN-120 | GitHub write |
| G15 | Security headers (`frame-ancestors`) | GYST-VN-075 | GitHub write |

### E.1 Step 0 (the `10A` O-4 package)

| # | Item | Task |
|---|---|---|
| 1 | Login throttling | GYST-VN-006 |
| 2 | Failed-login auditing | GYST-VN-007 (store: GYST-VN-005) |
| 3 | Expired-session cleanup | GYST-VN-008 |
| 4 | File-permission hardening | GYST-VN-009 |
| 5 | Scheduled GYST backup | GYST-VN-010 |
| 6 | Off-host backup copy | GYST-VN-012 |
| 7 | One tested restore | GYST-VN-013 |
| 8 | `/healthz` | GYST-VN-014 |
| 9 | Basic application logging | GYST-VN-015 |
| 10 | Log rotation | GYST-VN-016 |
| 11 | UTC instants for new audit/event records | GYST-VN-004 |
| — | **Separate:** local-day correctness | GYST-VN-003 (P0), with its historical-data **decision** in GYST-VN-011. No automatic rewrite of history. |

### E.2 Session model (`10A` D-S5)

| Rule | Value | Task |
|---|---|---|
| Rolling lifetime | 30 days (exists; verified by tests) | GYST-VN-077 |
| Absolute lifetime | 90 days from creation | GYST-VN-077 |
| Re-auth window | about 15 minutes per session | GYST-VN-079 |
| Revoke all / on password change | yes | GYST-VN-078 |
| Expired sweep | hourly | GYST-VN-008 |

### E.3 "Meaningful" GitHub writes — resolved classification

This resolves carry-forward item 3 of `10A`. Jay confirms it in GYST-VN-118, and it is encoded as a policy map with one test per action.

Legend (`09` §G.2):
- **PA:** public, authenticated
- **PR:** public, plus re-authentication within the ~15-minute window
- **TO:** tailnet only, plus step-up

| Write | Class | Reason |
|---|---|---|
| Create issue | **PA** | Reversible (close it), attributed to the GYST App, audited, doesn't start work |
| Edit title/body | **PA** | GitHub keeps edit history; reversible |
| Add/remove a normal label (`type:*`, `priority:*`, others) | **PA** | Reversible workflow metadata |
| Set workflow status (`status:inbox`/`needs-spec`/`ready`/`in-progress`/`blocked`/`review`) | **PA** | Reversible. `status:ready` means "specified", not "dispatch". `status:done` is only set by the normalizer on close. |
| Comment | **PA** | Reversible (editable/deletable), audited |
| Assign an agent (`agent:claude`/`agent:codex`) | **PA** | Intent only; doesn't dispatch (`09` §G.2) |
| Set `agent:ready` — NORMAL issue | **PR** | Authorises spending and code changes (D-S4) |
| Set `agent:ready` — ELEVATED/PRIVILEGED | **TO** (+ per-run confirm for PRIVILEGED) | Security or infrastructure code (D-S4) |
| Request agent review | **PR** | Starts a run and spends money |
| Close / verify & close issue | **PR** | Closing attests that Jay verified the work (`Refs #n`, D-S3), and closes the Attention loop. A stolen session shouldn't be able to mark work verified. |
| Any bulk write (more than one issue in one action) | **PR** | Limits the blast radius of a stolen session |
| `risk:*` labels | **TO** | They change which approvals are required |
| Merge PR, repository settings, branch protection | **NEVER** through GYST | `08A` D-J2/D-J4; the App has no such permission |

### E.4 Public System data (`09` §K.3, binding)

| | Fields |
|---|---|
| **Allowed publicly (authenticated)** | Service display name and health state; backup current/stale/failed with age; CPU/memory/disk as single numbers rounded to 5 %; certificate valid/expiring (days); host uptime rounded to days; last check time |
| **Never exposed publicly** | Environment/config, mounts and host paths, container IDs, network layout/internal hostnames/ports, raw logs, deploy internals (SHA, image, restart counts) |
| **Tailnet only** | Image tag, restart count, exact per-mount metrics, deploy SHA/time, bounded redacted logs (step-up), actions (step-up + confirm), console link |

Enforced by separate public and tailnet view types in GYST-VN-093, and by the dual-listener route sets in GYST-VN-073.

### E.5 Agent limits (`09` §I.3 as explicit configuration in GYST-VN-133)

| Limit | Value |
|---|---|
| Repositories | `gyst-lifeops` only |
| Implementation jobs | 1 |
| Review jobs | 1 |
| Implementation timeout | 45 min |
| Review timeout | 15 min |
| Worktrees | 5 max |
| Automatic retries | 0 |
| Runs per day | 6 |
| Spend | Per-run cap (e.g. Claude `--max-budget-usd`) + provider budget alerts |

SamVision is added only through GYST-VN-136, after evidence on all 8 criteria: worktree isolation, repository isolation, no credential exposure, branch behaviour, independent review, cleanup, useful logs, no host-privilege violations.

## F. Product gates

| Gate | Rule | Applies to |
|---|---|---|
| PG-1 No empty destinations | A navigation entry, More row, sub-nav segment or sidebar item appears only when its destination works | GYST-VN-025, GYST-VN-026, GYST-VN-039, GYST-VN-062, GYST-VN-093, GYST-VN-105, GYST-VN-111 |
| PG-2 No UI for missing pipelines | No Health, trend or integration UI for data that isn't produced, unless the task builds the pipeline | GYST-VN-052, GYST-VN-058, GYST-VN-057 |
| PG-3 Health invariants | Missing ≠ zero; Watch and phone steps separate; burn = resting + active; no step→calorie conversion; food budget ≠ observed balance | Every Health task |
| PG-4 One Attention model | No separate inbox/notifications/review queue; items point at sources; no editable work in Attention | Phase 5 onward |
| PG-5 Terminology | Project/Task/Roadmap in functional UI; Galaxy/world/ship/XP only as personality; no persisted-data renames | GYST-VN-027 and every later UI task |
| PG-6 GitHub is canonical | GitHub Issues hold development work; GYST caches only what the UI needs | Phase 8 onward |
| PG-7 Human merges | GYST never merges, never shows full diffs | Phases 8–12 |
| PG-8 Learn scope | Courses → Learn is IA/presentation only; no Master's/program features | GYST-VN-050, GYST-VN-051 |
| PG-9 Quiet healthy | Healthy infrastructure never dominates Today; problems are prominent | GYST-VN-035, GYST-VN-094 |
| PG-10 Galaxy preservation (binding, `17A`) | The real Three.js Galaxy is preserved and evolved, never deleted, replaced, downgraded or substituted by a list/grid or the Stage 7 prototype; baseline GYST-VN-143 precedes material Galaxy changes; rewrites need Jay's explicit approval | GYST-VN-143, GYST-VN-028, GYST-VN-037, GYST-VN-043, GYST-VN-044, GYST-VN-045, GYST-VN-046, GYST-VN-047, GYST-VN-048, GYST-VN-049, GYST-VN-144, GYST-VN-145 and every shared-surface task |

## G. Design gates

Every UI task (and every UI-changing bug/feature) is checked against:

| Gate | Requirement |
|---|---|
| DG-1 Source of truth | `13A` > `11`/`12`/`13` > prototypes. Old design docs are context only. |
| DG-2 No invented patterns | No new spacing, radii, navigation, state colours, hierarchy or component patterns without a design task. Conflicts are reported, not solved ad hoc. |
| DG-3 Browser verification | Phone (390×844) and desktop (1280×800) in Chrome, light and dark, keyboard pass, no console errors or failed requests |
| DG-4 Real device | Major mobile screens (Today, Health, Galaxy, Projects, Learn, launcher, re-auth sheet) are verified on a real iPhone in Safari and in the installed PWA where practical |
| DG-5 Targets | ≥ 44 px; Galaxy 56 px |
| DG-6 Overflow | No horizontal page overflow at 320–1280 px |
| DG-7 Disclosure | Hidden-until-requested content stays hidden by default (`12`) |
| DG-8 Accessibility | Labels on icon-only controls, visible focus, heading order, state conveyed by icon + word, not colour alone |
| DG-9 Violet | `--xp`/`--xp-soft` are identity only, never state (DD-4) |
| DG-10 Appearance | One System/Light/Dark setting; System follows OS changes live (DD-1) |
| DG-11 No new UI dependency | No framework, build system or component library (`13A`, working agreement §9) |
| DG-12 Galaxy regression | Galaxy tasks pass the `17A` §L checklist; shared CSS/JS, shell, CSP, terminology and project-data tasks pass the Galaxy quick check |

## H. Parallelization

Summary (details and file groups in `16` §C–§D):

- **Lanes that can run at the same time:**
  - Jay's operations lane (permissions, backups, restore, OPS1, Kuma, GitHub/Cloudflare/n8n settings)
  - a backend-correctness lane (time, auth)
  - a UI-foundation lane
  - a small independent-screen lane (Health truth fixes, Galaxy targets)
- **Must stay sequential:**
  - all Today tasks (`today.html`, `today_view.go`, `ui-today.css`)
  - all Galaxy tasks (`galaxy.js`)
  - the shared component tasks (`ui-components.css`, `ui_gallery.html`)
  - the shell tasks (`layout.html`, `ui_shell.html`, `ui_shell.go`)
  - tasks editing `handlers.go` or `middleware.go`
- **Tasks safe to start immediately after the branch baseline:** GYST-VN-003, GYST-VN-006, GYST-VN-004, GYST-VN-018, GYST-VN-052, GYST-VN-143, plus Jay's GYST-VN-009. No shared files between these tasks (checked by the generator against each task's RELEVANT FILES).
- **Concurrency limit for agents:** the same rule as for dispatch. Keep at most **two** agent implementation sessions in flight: one backend and one UI, on disjoint files. Jay reviews every PR.

## I. Stop/go checkpoints

| Checkpoint | When | Go criteria | If no-go |
|---|---|---|---|
| CP-0 Start | Before any vNext PR | Base branch decided (GYST-VN-001); audit docs on it; CI green | Stop; resolve governance |
| CP-1 Foundation | End of M0 | All Step 0 items done with evidence; local-day fix deployed; 2 scheduled backups verified; 1 restore recorded; OPS1 result recorded (or explicitly deferred by Jay) | No external integration, no machine credential, no agent account |
| CP-2 UI foundation | End of Phase 1 | Components in the gallery pass DG-3 to DG-9; no visual regression on existing screens | Fix before migrating screens |
| CP-3 Today | End of M1 | Real iPhone PWA check of Today and navigation; Jay uses it for several days without regressions | Adjust before Phase 3 |
| CP-4 Life areas | End of M2 | Health invariant review passed; Galaxy on device; no empty destinations | — |
| CP-5 Hardened base | End of Phase 6 | Dual listener verified from both networks; step-up and nonce negative tests; headers enforced; non-root running; audit viewer shows real events | No tailnet-only views, no GitHub or machine traffic |
| CP-6 Read-only integrations | End of M4 | Public System page contains no forbidden field (checked against §E.4); GitHub client has no write methods; webhook signature/replay negative tests; PAT (if used) has an expiry plan | No writes |
| CP-7 Before public higher-impact writes | Before GYST-VN-130 is exposed publicly | Cloudflare Access validated on all 6 flows (GYST-VN-125); bypass list exact; meaningful-writes policy confirmed (GYST-VN-118) | Keep `agent:ready` tailnet-only |
| CP-8 Agent go-live | GYST-VN-135 | Provider auth verified within the last few days (GYST-VN-112, GYST-VN-113); permissions re-verified; branch protection active; limits configured; 3 supervised runs reviewed | Dispatch stays disabled |
| CP-9 Controlled operations | Before GYST-VN-139 | Action runner allow-list reviewed; every action tested once under supervision; backups current | Actions stay disabled |

## J. Deployment approach

The approach stays the same: Docker Compose on the host, deployed by Jay. vNext adds discipline, not a pipeline.

1. **Branch → PR → CI → review → merge** to the base branch chosen in GYST-VN-001. Branch protection (GYST-VN-099) enforces it once it is in place.
2. **Deploy is a Jay action.** It runs from a clean checkout of the merged commit:
   1. Backup first (`lifeops backup`; after GYST-VN-088, the deploy script does it and refuses to continue if the backup fails).
   2. `docker compose build` and `up -d`.
   3. `/healthz` healthy.
   4. Record the SHA (GYST-VN-088).
3. **One risky change per deploy.** PRIVILEGED changes (Compose, Caddy, Dockerfile, listeners, non-root) deploy alone, with the rollback written down before the deploy.
4. **Migrations are additive only.** Each is applied first to a copy of the latest backup, with `integrity_check` and `foreign_key_check`. Applied migrations are never edited. The database is never reset.
5. **Feature flags** for writes and integrations are plain booleans in the config struct (GYST-VN-072); there is no flag framework. New write paths ship disabled and are enabled after the checkpoint.
6. **Verification after each deploy:**
   - public host from the phone (login, Today)
   - tailnet host
   - `/healthz`
   - one screen touched by the change
   - the result is recorded in the task ledger
7. **Rollback:**
   - Code: redeploy the previous SHA.
   - Additive migrations stay; old code ignores the new columns and tables.
   - Data: restore only by following the tested runbook (GYST-VN-013), never ad hoc.

## K. Final review

### K.1 Counts

**Total tasks: 145**

| Priority | Tasks |
|---|---|
| P0 | 5 |
| P1 | 52 |
| P2 | 73 |
| P3 | 15 |
| **Total** | **145** |

| Risk class | Tasks |
|---|---|
| NORMAL | 68 |
| ELEVATED | 33 |
| PRIVILEGED | 18 |
| NOT AGENT-ELIGIBLE | 26 |
| **Total** | **145** |

| Type | Tasks |
|---|---|
| Bug | 4 |
| Feature | 33 |
| UI | 39 |
| UX | 4 |
| Refactor | 3 |
| Security | 18 |
| Infrastructure | 9 |
| Integration | 6 |
| Operations | 19 |
| Documentation | 3 |
| Research | 7 |
| **Total** | **145** |

| Agent eligibility | Tasks |
|---|---|
| Claude/Codex implementable | 116 |
| Human-assisted (agent prepares; Jay decides or executes) | 11 |
| Human-only (Jay) | 5 |
| NOT AGENT-ELIGIBLE operational work (Jay) | 13 |
| **Total** | **145** |

| Complexity | Tasks |
|---|---|
| S | 89 |
| M | 52 |
| L | 4 |
| **Total** | **145** |

| Milestone | Phases | Tasks | P0/P1/P2/P3 | User-visible outcome |
|---|---|---|---|---|
| M0 Safe foundation | 0 | 17 | 5 / 11 / 1 / 0 | GYST data is backed up off-host and restorable, the public login resists guessing, "today" means Jay's local day everywhere, and failures are visible in logs and /healthz. |
| M1 vNext core experience | 1, 2 | 20 | 0 / 17 / 3 / 0 | The app looks and navigates like vNext: Today · Projects · Health · Learn · More, a calm Today with Now, Rhythm and no duplicate blocks, a simpler More, and System/Light/Dark appearance. |
| M2 vNext Life areas | 3 | 27 | 0 / 10 / 13 / 4 | Projects, Galaxy, Learn, Health and Goals use the vNext patterns and plain terminology; Health shows only what really works; Galaxy is usable on a phone. |
| M3 Command centre | 4, 5 | 9 | 0 / 5 / 3 / 1 | One launcher (Go · Do · Capture) reaches everything; Apps opens external tools; one Attention model tells Jay what needs him, from GYST's own data. |
| M4 Read-only integrations | 6, 7, 8, 9 | 41 | 0 / 9 / 30 / 2 | GYST shows sanitized server health, GitHub issues/PRs/reviews in Development, and n8n automation status, all read-only, on a hardened two-listener foundation. |
| M5 Agent-assisted development | 10, 11, 12 | 25 | 0 / 0 / 23 / 2 | Jay can write GitHub issues from GYST, mark a NORMAL issue ready for an agent after re-auth, watch the run, review the PR in GitHub and verify-and-close from GYST. |
| M6 Controlled operations | 13, 14, 15 | 6 | 0 / 0 / 0 / 6 | From the tailnet, with re-auth and confirmation, Jay can run a backup, restart an allow-listed service, run an approved n8n workflow and read bounded redacted logs. |

### K.2 First implementation tasks

The first tasks mix foundation work with visible product progress. The numbering is the recommended start order; tasks on different lanes run in parallel (`16` §C).

| # | Task | Lane | Why it belongs early |
|---|---|---|---|
| 1 | GYST-VN-001 Branch baseline (Jay) | Governance | P0: no PR has a defined base until it's decided |
| 2 | GYST-VN-009 File permissions (Jay) | Ops | P0: secrets and health data are world-readable; binding gate for agent accounts; needs nothing else |
| 3 | GYST-VN-010 Scheduled backup | Ops | P0: a live single-user database with no scheduled backup |
| 4 | GYST-VN-006 Login throttling | Auth | P0: the public login can be brute-forced |
| 5 | GYST-VN-003 Local-day correctness | Time | P0: verified wrong-day data; Now, Rhythm, overdue and Attention all build on it |
| 6 | GYST-VN-052 Hide Health placeholders | Screens | **Visible:** Health stops promising features that don't exist; isolated templates; S |
| 7 | GYST-VN-143 Galaxy baseline → GYST-VN-044 Galaxy 56 px targets | Galaxy | The baseline protects the signature feature (`17A`); then a **visible** fix: Galaxy becomes comfortable on the iPhone; isolated files |
| 8 | GYST-VN-018 Design tokens | UI | Unlocks every vNext UI task; no visible change, so low risk |
| 9 | GYST-VN-004 → GYST-VN-005 → GYST-VN-007 | Auth | Step 0 items 11 and 2; the audit store is reused by every later security feature |
| 10 | GYST-VN-053 Real Hevy state | Screens | **Visible:** removes a UI contradiction; small |
| 11 | GYST-VN-019 System/Light/Dark | UI | **Visible:** GYST follows iOS appearance (DD-1) |
| 12 | GYST-VN-014 → GYST-VN-015 | Observability | Step 0 items 8 and 9; after throttling (shared middleware) |
| 13 | GYST-VN-012, GYST-VN-013 (Jay) | Ops | Step 0 items 6 and 7: a backup is only useful off-host and proven restorable |
| 14 | GYST-VN-008, GYST-VN-016 | Auth / Ops | Step 0 items 3 and 10; small |
| 15 | GYST-VN-020 → GYST-VN-021 → GYST-VN-022 | UI | Components for the Today and navigation work in Phase 2 |

### K.3 Estimated shape

**Overall:** large. Most tasks are S or M. The L items are the dual listener, the Server Provider, agent dispatch and the legacy-screen sweep.

| Milestone | Shape | Driver |
|---|---|---|
| M0 | Small–medium | Many S tasks; the Jay-only operations set the pace |
| M1 | Medium | Serialized Today and shell work |
| M2 | Medium–large | Many screens, but they parallelize by area |
| M3 | Small–medium | Launcher + Attention |
| M4 | Large | Hardening plus three read-only integrations |
| M5 | Large | Bridge, GitHub writes, human gates |
| M6 | Small–medium | Mostly small tasks behind gates |

## Unresolved decisions (carried into implementation)

| # | Decision | Task | Recommendation |
|---|---|---|---|
| U-1 | Base branch for vNext; fate of unpushed `master` commits and `stash@{2}` | GYST-VN-001 (Issue #2, open; see `18` §L) | Jay decides; agents don't touch them |
| U-2 | Historical wrong-day rows | GYST-VN-011 | Measure first; correct only rows with a provable instant, or leave them |
| U-3 | Off-host backup destination | GYST-VN-012 | A second machine on the tailnet; if cloud, client-side encrypted |
| U-4 | Repository and packaging for Bridge/Provider/Runner | GYST-VN-071 | Separate private `gyst-ops` repo, not on the agent allowlist |
| U-5 | GitHub App-first or bootstrap PAT | GYST-VN-096 | App-first |
| U-6 | Claude Code / Codex unattended auth | GYST-VN-112, GYST-VN-113 | Decide from current official docs right before the bridge |
| U-7 | Meaningful GitHub writes | GYST-VN-118 | Confirm §E.3 |
| U-8 | Cloudflare Access | GYST-VN-125 | Validate the 6 flows before public `agent:ready` |
| U-9 | Replacing the NAV1 assertions in `scripts/check-ui-shell.cjs` | GYST-VN-025 | Treat as an explicit spec change that Jay acknowledges in the PR, not as weakening a test (`docs/RULES.md`) |
| U-10 | Training storage shape | GYST-VN-059 | Decide before any training-history feature |
| U-11 | Reconciling the existing `docs/tasks/` ledger (A1, A2, M3 Apple Health ingestion, `GYST_PASSWORD_HASH`) with this backlog | GYST-VN-001 | Record each as done, superseded or still open during the baseline task |
| U-13 | Permanent Galaxy rule in `AGENTS.md`/`CLAUDE.md` | Stage 9 | **Resolved in Stage 9** (`18` §K): the rule is in root `AGENTS.md` |
| U-12 | Root containers other than GYST (n8n, SamVision, Kuma) | outside this backlog | Separate server-hardening review |

## Explicitly deferred

These are not in the backlog:
- passkeys/WebAuthn
- agent egress restrictions and rootless test sandboxes
- signed agent commits
- audit hash-chaining/export
- Tailscale identity headers
- read-only root filesystem
- deploying from GYST
- a web terminal
- merging PRs or showing full diffs in GYST
- editing n8n workflows in GYST
- SamVision agent runs (until GYST-VN-136)
- a herdr dependency
- Learn programs/Master's features
- Health Week/Month/Year, meal planning, workout logging, Hevy writes
- configurable Rhythm or a widget framework
- the weekly/monthly Review feature
- Attention digests
- multi-timezone support
- automatic rewriting of historical rows
- any JS framework, build system or new component library
