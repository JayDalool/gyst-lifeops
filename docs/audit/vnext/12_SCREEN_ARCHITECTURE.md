# Stage 7B — Screen Architecture

- **Stage:** 7 (with `11_DESIGN_SYSTEM_VNEXT.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `prototypes/`)
- **Binding:** target navigation Today · Projects · Health · Learn · More (`05A` D11);
  Projects contains Projects · Development · Galaxy · Roadmaps (D2); Goals at More → Goals (D7);
  one Attention (D5, C1); System/Apps under More, desktop secondary group (D6);
  unbuilt Health hidden (D8); quiet healthy status (D9); prayer in Now, check-ins near the top (D10);
  public = sanitized System only, control tailnet-only (`08A` D-J7); re-auth ≈ 15 min (`10A` D-S5).
- **Backend dependency note:** screens marked ◇ depend on Stage 5 integrations
  (GitHub, Agent Bridge, Server Provider, n8n events, Attention store). Until
  those exist, they aren't shown at all (no placeholders, D8).

Each screen lists the same fields. "Phone" is < 1024 px; "Desktop" is ≥ 1024 px
(inspector ≥ 1280). Responsive rules are in `13`.

---

## Today — `today.html`

| Field | Definition |
|---|---|
| Purpose | Answer: what now, what needs me, what's today, how am I keeping my rhythm |
| Primary information | Now (next action + next prayer + next timed item), Rhythm, Attention (if any), Today's items |
| Primary action | Complete the Now item (check circle) |
| Secondary actions | Capture (+), open launcher, rhythm check-ins, open an attention item, "All tasks" |
| Visible at a glance | Greeting/date, XP chip, Now block, Rhythm strip (8 check-ins), ≤3 attention rows + "All n", open items, status strip (exceptions only), Health glance (once data exists) |
| Hidden until requested | Done items ("Done · n"), full Attention, item details, prayer schedule (via Now → prayer sheet), analytics |
| Empty state | No open items: one line "Nothing else today" under Now; Now falls back to "Continue learning" or the next routine |
| Error state | A failed section shows an inline banner; other sections render |
| Stale/offline | Status chip "Health not synced · 6:40"; offline banner keeps last data; check-ins queue locally only if the PWA supports it (otherwise disabled with reason) |
| Phone | Order: header → **Now → Rhythm** → Attention → Today → Status → Health glance |
| Desktop | Two columns (max 1120): left = Now, Rhythm, Attention, Today; right = Status strip, Health glance. No extra widgets because space exists. |

**Scenario rules (from the brief):**

| Scenario | Presentation |
|---|---|
| No Attention items | Attention section absent; the Today badge is hidden |
| One overdue task | One warn row "Renew parking permit · Overdue since Friday" with "Do today"; badge unchanged (warnings don't badge) |
| Two reviews waiting | One grouped row "2 reviews waiting · ATL-142, ATL-151" + "Review" (needs_action → badges) |
| Claude finishes a task | Status strip chip disappears when the run ends; the review appears as a needs_action row once the PR is ready; optional toast if Today is open |
| Health not synced | Warn chip in the status strip; Health glance shows last values with a stale marker |
| Server problem | Critical attention row at the top ("Backup job failed · System"); sidebar System dot turns red; push if critical |
| Everything normal | No Attention, no status strip (before System exists: nothing, D9; later: one quiet "All systems normal" chip) |

## Projects — `projects.html`

| Field | Definition |
|---|---|
| Purpose | See active projects and their next signal; reach Development, Galaxy, Roadmaps |
| Primary information | Active projects (orb, name, one signal, progress ring) |
| Primary action | Open a project |
| Secondary actions | New project (+), filter (Active/Planned/Blocked/Paused), switch area (sub-nav) |
| Visible at a glance | Sub-nav, filter pills, active list, collapsed "Done · n", roadmaps summary row |
| Hidden until requested | Completed projects, archived, project details, dependencies, proof |
| Empty state | "No active projects · New project" |
| Error state | Section banner; list keeps last data |
| Stale/offline | Offline banner; creation disabled with reason |
| Phone | Sticky segmented sub-nav under the title; default filter **Active** (Stage 3) |
| Desktop | Same; sidebar shows Development / Galaxy / Roadmaps as sub-items while in Projects |

## Project Detail — `project-detail.html`

| Field | Definition |
|---|---|
| Purpose | Work the project: next task, then tasks, roadmap position, activity, resources |
| Primary information | Hero (orb, state chip, roadmap tag, title, progress bar + "13 of 21 · due") and the Next task |
| Primary action | Complete the next task |
| Secondary actions | Add task, section switch (Tasks/Roadmap/Activity/Resources), ⋯ menu (edit, dependencies & proof, move roadmap, archive, delete…) |
| Visible at a glance | Hero, Next, open tasks |
| Hidden until requested | Completed tasks (Done · n), roadmap chain, activity timeline, notes/links, dependencies/proof (menu), delete (menu → confirm) |
| Empty state | No tasks: "Add the first task" row |
| Error state | Inline banner in the affected section |
| Stale/offline | Offline banner; edits disabled |
| Phone | Single column; sections as segmented control; ⋯ opens a sheet |
| Desktop | Same column; ⋯ opens a popover; optional inspector for a selected task (≥1280) |

## Development ◇ — `development.html`

| Field | Definition |
|---|---|
| Purpose | Native control surface for GitHub-backed development work |
| Primary information | Needs you (reviews, approvals, agent questions) → In progress → Ready → (Inbox link) → Repositories |
| Primary action | Open the top "Needs you" item |
| Secondary actions | New task (creates a GitHub issue), switch view (Tasks · Reviews · Agents · Repos pills) |
| Visible at a glance | Status tile · title · ID · repo · agent/review signal · risk/priority tags (only if Elevated/Privileged or P0/P1) · state chip |
| Hidden until requested | Explanation, acceptance, attachments, dependencies, history, branch/PR details |
| Empty state | "No development tasks · Connect a repository" (after the GitHub integration exists) |
| Error state | Banner "GitHub unreachable · showing data from 14:02" |
| Stale/offline | Footer "Synced from GitHub 2 min ago"; stale chip past threshold; writes disabled |
| Phone | Rows link to the task page |
| Desktop | ≥1280 inspector shows the selected task's summary and next action |

## Development Task ◇ — `dev-task.html`

| Field | Definition |
|---|---|
| Purpose | Understand and advance one issue |
| Primary information | ID, state, repo, priority, title, one-line summary, workflow strip (Spec · Ready · Working · Review · Done), agent + risk |
| Primary action | Contextual: "Ready for agent" (re-auth), "Review PR", "Verify & close" |
| Secondary actions | Open on GitHub, change labels (sheet), assign agent |
| Visible at a glance | Hero, workflow strip, agent row with risk class, acceptance checklist |
| Hidden until requested | Explanation (incl. out-of-scope), dependencies, attachments, history (disclosures); branch/PR/verification when present |
| Empty state | n/a (detail) |
| Error state | Banner; the GitHub link is always present |
| Stale/offline | "Issue and labels live on GitHub · synced n min ago" footer; actions disabled when stale beyond threshold |
| Phone | Full page; re-auth as a sheet |
| Desktop | Page, or as inspector content from Development |

Risk display: **Normal** is a neutral tag; **Elevated** an amber tag with an
approval banner (tailnet only); **Privileged** a red tag with a per-run
confirmation; **Not agent-eligible**: agent row replaced by "Human only".

## Reviews ◇ — `reviews.html`

| Field | Definition |
|---|---|
| Purpose | Fast queue of decisions on agent/human PRs, and merged-not-verified tasks |
| Primary information | Waiting for you: title · ID · "Claude → Codex" · PR # · check/concern chip |
| Primary action | Open review summary → **Open PR** (GitHub) |
| Secondary actions | Request changes (writes a review), re-run with notes, Later; "Verify" for merged items |
| Visible at a glance | Queue rows; "Merged · verify & close" group |
| Hidden until requested | Acceptance, test result, reviewer findings, diff summary (+312 −40, 7 files) |
| Empty state | "No reviews waiting" (one line) |
| Error state | Banner; queue from last snapshot |
| Stale/offline | Footer freshness; actions disabled when offline |
| Phone | Row → bottom sheet summary |
| Desktop | Row → inspector summary (≥1280) |

GYST never merges and never shows full diffs (`08A`).

## Repositories ◇ — Development → Repos (`development.html#repos`)

| Field | Definition |
|---|---|
| Purpose | Which repos are connected and healthy |
| Primary information | Repo name · linked project · open count · CI dot |
| Primary action | Open repo (filtered task list) |
| Secondary actions | Open on GitHub |
| Visible at a glance | Rows + "Synced n min ago" |
| Hidden until requested | Branches, workflow runs, settings (never in GYST) |
| Empty state | "No repositories connected" + a pointer to Settings → Integrations |
| Error / stale / offline | As Development |
| Phone / Desktop | List / list + inspector |

## Agents ◇ — `agents.html`

| Field | Definition |
|---|---|
| Purpose | What agents are doing and what's queued, within limits |
| Primary information | Now (agent · task · elapsed · Working/Reviewing), Queue, Recent (Failed / Done / Needs input), Available (Claude/Codex readiness) |
| Primary action | Open a run |
| Secondary actions | None on the list (Stop lives on the run) |
| Visible at a glance | Rows; "Limits: 1 implementation and 1 review"; bridge freshness footer |
| Hidden until requested | Activity, branch, PR, errors, logs (tailnet) |
| Empty state | "No agent runs yet" |
| Error state | "Agent Bridge not connected since HH:MM" banner; runs show Unknown |
| Stale/offline | Heartbeat age in footer; Unknown state after 3 min without heartbeat |
| Phone / Desktop | List / list (+ inspector optional) |

States: **Idle · Queued · Working · Reviewing · Needs input · Failed · Done** only.

## Agent Detail ◇ — `agent-detail.html`

| Field | Definition |
|---|---|
| Purpose | One run: facts, safe activity, stop |
| Primary information | Runtime, state chip, elapsed, task, project, branch, risk, reviewer, PR |
| Primary action | **Stop run** (dialog confirm), allowed publicly (`10A` D-S4) |
| Secondary actions | Open task, open PR, Logs (tailnet only) |
| Visible at a glance | Facts list + last 5 safe activity summaries |
| Hidden until requested | Logs (tailnet, bounded), error details |
| Empty state | n/a |
| Error state | Failed run: crit banner with a one-line reason + "Retry" (manual, counts toward limits) |
| Stale/offline | Heartbeat age; Stop disabled offline with reason |
| Phone / Desktop | Page / page or inspector |

Never: provider credentials, raw prompts/transcripts, inferred reasoning, Pause, Merge.

## Galaxy — `galaxy.html`

| Field | Definition |
|---|---|
| Purpose | Signature spatial view of projects; explore and jump in |
| Primary information | Active worlds on the inner orbit (size ~ scope, glow ~ activity); completed worlds dimmed on the outer orbit (toggle) |
| Primary action | Select a world → preview sheet → "Open project" |
| Secondary actions | Filters (Active · Done · category), zoom/recenter (44 px), list of worlds under the scene |
| Visible at a glance | Scene with labelled worlds, legend, ship |
| Hidden until requested | Project details (sheet), completed worlds (Done toggle, off by default) |
| Empty state | One sun + "Create a project to launch your first world" |
| Error state | WebGL unavailable → the world list becomes primary (existing fallback) |
| Stale/offline | Renders from cached project data |
| Phone | Scene about 64 vh (min 420), sheet preview, 56 px hit areas, ship flies to the selection |
| Desktop | Larger scene; preview as a side panel inside the scene; keyboard list below |

Sparse data (≤3 worlds): worlds scale up and labels stay on; never a grid of cards.

## Roadmaps — Projects → Roadmaps

| Field | Definition |
|---|---|
| Purpose | Ordered route of projects toward a goal |
| Primary information | Roadmap rows (name · "3 of 5 projects" · next project); detail = ordered stops with state |
| Primary action | Open roadmap |
| Secondary actions | Import/export Markdown (⋯, Level 3), weekly review (future) |
| Visible at a glance | One row per roadmap |
| Hidden until requested | Full route, import/export, historical detail |
| Empty state | "No roadmaps" + New roadmap |
| Error / stale / offline | Banner; read-only offline |
| Phone / Desktop | List → page (route as a vertical stop list; the rocket route visual optional) |

Replaces the Stage 3 16-screen Roadmap page: projects are no longer repeated in full.

## Health Summary — `health.html`

| Field | Definition |
|---|---|
| Purpose | Today's body at a glance from real data only |
| Primary information | 4-metric grid: Intake (vs target), Burned (resting + active), Steps (Watch, phone separately), Exercise |
| Primary action | Open Nutrition (tap Intake) |
| Secondary actions | Energy info sheet; open Training, Weight, Plan rows |
| Visible at a glance | Metrics, Energy (Food left, Balance so far: observed), Training last workout (if connected), Weight latest (if any), Plan day X of Y, sync banner only when stale |
| Hidden until requested | Formulas/provenance (info sheet), history (Progress), targets (Plan) |
| Empty state | Before first sync today: metrics show "—" with "No data yet today" in one line; no explanatory paragraphs |
| Error state | Sync failed: warn banner "Apple Health sync failed · last 6:40" |
| Stale/offline | Amber "Stale" chip on the sync line |
| Phone / Desktop | 2-column metrics / 4-column metrics |

**Never shown (D8):** Week/Month/Year, Plan meal, Start workout, trend placeholders,
weight entry without a writer. **Invariants:** missing = "—"; Watch ≠ phone; burned = resting + active; no step→calorie.

## Nutrition — `nutrition.html`

| Field | Definition |
|---|---|
| Purpose | Daily calories and macros vs targets |
| Primary information | Calories ring (value of target, left); Protein, Carbs, Fat, Fiber rows with bars |
| Primary action | None required (read); open Targets |
| Secondary actions | Targets row → Plan |
| Visible at a glance | Ring, 4 macro rows, provenance line with stale chip |
| Hidden until requested | Data-timing explanation (info), history (future Progress) |
| Empty state | "No food logged yet today" + "—" values |
| Error state | Sync failed banner |
| Stale/offline | Stale chip on the provenance line |
| Phone / Desktop | Stack / stack (max 760) |

Missing macro: "— not received" + a dashed bar. Target not configured: value only, no bar.

## Training — Health → Training

| Field | Definition |
|---|---|
| Purpose | Hevy as a provider: recent training and simple progress |
| Primary information | **Connected:** last workout, this week count, next routine (if any), muscle focus summary. **Disconnected:** "Hevy · Not connected" + how to connect (existing pattern) |
| Primary action | Open latest workout (connected) / Connect Hevy (disconnected; settings) |
| Secondary actions | Manual gym check-ins (disclosure), Sync now (rate-limited) |
| Visible at a glance | 3–4 rows |
| Hidden until requested | Sets/exercises, routine details, muscle map |
| Empty / error / stale | Not connected (pattern) / "Hevy unreachable · cached" banner / "Updated n h ago" |
| Phone / Desktop | List / list + inspector for a workout |

Not a Hevy clone: no workout logging, no Start workout.

## Progress — Health → Progress

| Field | Definition |
|---|---|
| Purpose | Trends that have real pipelines |
| Primary information | Weight history (30/90/all) when readings exist |
| Primary action | Change range |
| Secondary actions | None until more pipelines exist |
| Visible at a glance | Chart + latest + table disclosure |
| Hidden until requested | Table of readings |
| Empty state | "No weight readings yet" (no disabled entry button). Future trend cards appear only when their data exists (D8). |
| Error / stale | Banner / provenance line |
| Phone / Desktop | Stack / stack |

## Plan — Health → Plan

| Field | Definition |
|---|---|
| Purpose | Current health plan (period, targets) |
| Primary information | **Read view first:** title, day X of Y, targets list |
| Primary action | Edit plan (sheet/page) |
| Secondary actions | Past plans |
| Visible at a glance | Summary + targets |
| Hidden until requested | Edit form, past plans |
| Empty state | "No plan yet · Create plan" |
| Error / stale | Inline form errors on save |
| Phone / Desktop | Page / page |

**Future plan capabilities** (adherence, meal plans) appear as new rows only
when their backend exists. There are no placeholder buttons today.

## Learn — `learn.html`

| Field | Definition |
|---|---|
| Purpose | Answer "what should I study next?" |
| Primary information | Continue block (next session: course, duration, due) with Play |
| Primary action | Start/continue session |
| Secondary actions | Add course (+), open course, complete a session |
| Visible at a glance | Continue, This week (sessions due, catch-ups flagged), Courses (kind + progress ring) |
| Hidden until requested | Course detail (sessions, notes, links), paused/completed courses (collapsed), calendar feed (course ⋯) |
| Empty state | "Nothing to study yet · Add course" |
| One course | Continue + progress + next deadline only |
| Many courses | Continue picks the most urgent due/catch-up, then the most recent |
| Phone / Desktop | Stack / stack (+ inspector for a course ≥1280) |

Designed for later: Programs (degree → term → course) nest above courses without changing this screen's shape.

## Goals — More → Goals

| Field | Definition |
|---|---|
| Purpose | Long-horizon goals as their own destination (D7) |
| Primary information | Active goals: icon · name · period · progress (ring or bar) · pace signal ("on track", "behind") |
| Primary action | Open goal |
| Secondary actions | New goal, add progress, **edit** (missing today; required) |
| Visible at a glance | Active goals; completed collapsed |
| Hidden until requested | History chart, entries, archive |
| Empty state | "No goals · New goal" |
| Phone / Desktop | List / list + inspector |

Today shows a goal only when relevant (e.g. a weekly check-in due).

## Attention — `attention.html`

| Field | Definition |
|---|---|
| Purpose | Everything that needs Jay, from one model (D5) |
| Primary information | Grouped by severity: Critical · Needs you · Warnings; row = severity tile · title · source · age · one action or ⋯ |
| Primary action | The row's action (Review / Answer / Do today) or open the source |
| Secondary actions | Snooze 1 h / tomorrow, dismiss (returns if the condition changes), filter (All / Decisions / Problems) |
| Visible at a glance | Open items |
| Hidden until requested | Info items, resolved today (collapsed) |
| Empty state | "Nothing needs you" |
| Error state | "Some sources unavailable" banner listing the sources |
| Stale/offline | Freshness per source in meta |
| Phone / Desktop | Page (reached from Today, badge, launcher, push) / page (no sidebar item, C1) |

Not a task manager: no check circles, no editing; auto-resolves.

## System — `system.html`

| Field | Definition |
|---|---|
| Purpose | Is it healthy? Does anything need me? What next? |
| Primary information | Summary (n problems / All healthy), CPU/Memory/Disk (rounded, public-safe), Services with state, Backup, Certificates |
| Primary action | Open the problem (links to its Attention item / detail) |
| Secondary actions | **Tailnet only:** run backup, restart allow-listed service (re-auth + confirm), bounded logs, open Portainer |
| Visible at a glance (public) | Service name + state, backup current/stale/failed + age, rounded metrics, certificate validity |
| Hidden / tailnet-only | Versions, restart counts, deploy times, logs, actions (server-side enforced; the public view shows "Details and actions open on your tailnet") |
| Never | Env vars, mounts, container IDs, host paths, secrets, topology beyond names |
| Empty state | Before the Server Provider exists: System shows Kuma-based availability only, or isn't shown |
| Error state | "Status unavailable" (never "healthy") |
| Stale/offline | "Checked n min ago"; stale after 10 min |
| Phone / Desktop | Page / page (sidebar dot reflects the worst state) |

## Automations ◇ — `automations.html`

| Field | Definition |
|---|---|
| Purpose | GYST's n8n automations at a glance |
| Primary information | Name · next/last run · state (Failed / Running / Healthy / Disabled) |
| Primary action | Open automation (failed ones open detail) |
| Secondary actions | **Run now** (tailnet, confirm), Open n8n |
| Visible at a glance | Rows; failed first |
| Hidden until requested | Last success, error summary, schedule |
| Empty state | "No GYST automations" |
| Error state | "n8n unreachable" banner |
| Phone / Desktop | List + sheet / list + inspector |

Never the n8n editor; SamVision automations aren't listed.

## Apps — `apps.html`

| Field | Definition |
|---|---|
| Purpose | Launch external tools |
| Primary information | Grouped rows (Work · Operate · Learning): monogram · name · health dot · tailnet marker · external glyph |
| Primary action | Open in a new tab |
| Secondary actions | None (registry edits in Settings) |
| Visible at a glance | All registry apps with visibility "more" |
| Empty state | n/a (registry ships defaults) |
| Phone / Desktop | List / list |

## More — `more.html`

| Field | Definition |
|---|---|
| Purpose | Everything that isn't a primary tab, without being a sitemap |
| Primary information | Search field; group 1: **Attention**, **Goals**; group "Operate": **System**, **Automations**, **Apps**; group "Tools": All tasks & reminders, Calendar, Review & analytics, Import & export, Archive; **Settings** |
| Primary action | Open a destination |
| Visible at a glance | 12 rows in 4 groups (from 19 in 5) |
| Must not contain | Today, Projects, Health sub-pages, Learn, Galaxy, Development (all reachable from tabs) |
| Phone | Tab destination |
| Desktop | Reached from the sidebar's More; Operate items also appear in the sidebar (D6) |

## Settings — More → Settings

| Field | Definition |
|---|---|
| Purpose | Preferences and integrations |
| Primary information | Grouped rows: Prayer times · Appearance & motion · Notifications · Integrations (Apple Health, Hevy, GitHub, Agent Bridge, n8n: each Connected/Not connected) · Security (sessions, sign out everywhere, credentials: tailnet only) · Apps registry · About |
| Primary action | Open a group |
| Visible at a glance | Group rows with current value ("Winnipeg · ISNA") |
| Hidden until requested | Coordinates, per-prayer adjustments (advanced disclosure), credential admin (tailnet + re-auth) |
| Phone / Desktop | List → sub-pages / list + inspector |

## Launcher — `launcher.html`

| Field | Definition |
|---|---|
| Purpose | Go anywhere, do an allowed command, capture |
| Primary information | Input; modes Go · Do (>) · Capture (+); grouped results |
| Primary action | Enter on the selected result |
| Visible at a glance | Top hit, then groups (Development, Projects, Learn, Health, Apps, Settings) |
| Hidden until requested | Commands (Do mode), capture form (Capture mode) |
| Empty state | Recents + suggestions (Now item, top Attention) |
| Phone | Full-height sheet from the search icon (top bar or More) |
| Desktop | ⌘/Ctrl-K centred panel; sidebar "Search or run…" field |

Details in `13` §Launcher.

---

## Design decisions for Jay

Only choices that change identity or workflow. Everything else has a stated default.

### DD-1 — Default appearance
- **Question:** Should GYST follow the system light/dark setting by default?
- **Options:** A: follow system + manual override (prototype behaviour). B: always light unless toggled (current production). C: always dark.
- **Recommendation:** **A.** An always-open app should match the device at night.
- **Consequence:** The theme setting gains "Automatic".

### DD-2 — Rhythm strip contents
- **Question:** Keep Gym in the same strip as the prayers and athkar?
- **Options:** A: one strip (5 prayers · 2 athkar · gym), as prototyped. B: prayers + athkar only; gym moves to Health/Training. C: user-configurable strip.
- **Recommendation:** **A now, C later**. One tap row for all daily check-ins is fastest, while the spiritual items stay first and visually grouped.
- **Consequence:** A mixes spiritual and fitness check-ins in one row. B separates them at the cost of one more place to tap.

### DD-3 — XP presence on Today
- **Options:** A: small violet "Lv 16" chip in the header (tap → progress sheet). B: hidden from Today (Review only). C: larger XP progress bar.
- **Recommendation:** **A.** Keeps GYST's playful identity without competing with Now.
- **Consequence:** XP stays visible daily but quiet.

### DD-4 — Brand accent
- **Question:** Adopt the violet "orbit dot" next to the GYST wordmark and violet for XP/Galaxy entry points as the identity accent (blue stays the action colour)?
- **Options:** A: yes (as prototyped). B: blue only.
- **Recommendation:** **A.** One restrained identity colour links Galaxy, XP and the brand.
- **Consequence:** Violet is reserved for identity; it must never mean state.

---

## Prototype index

Non-production, fictional data, isolated from `web/`. Private preview:
https://claude.ai/artifact/RkJTB44V82FpVekoWfWdA8

| # | Required prototype | Path |
|---|---|---|
| — | Index | `docs/audit/vnext/prototypes/index.html` |
| 1–2 | Today (phone + desktop; scenarios Busy / One overdue / Everything normal) | `prototypes/today.html` |
| 3 | Projects (phone) | `prototypes/projects.html` |
| 4 | Project detail | `prototypes/project-detail.html` |
| 5 | Development (desktop, inspector ≥1280) | `prototypes/development.html` |
| 6 | Development task | `prototypes/dev-task.html` |
| 7 | Reviews | `prototypes/reviews.html` |
| 8 | Agents | `prototypes/agents.html` |
| 9 | Agent detail (run) | `prototypes/agent-detail.html` |
| 10 | Galaxy (phone concept) | `prototypes/galaxy.html` |
| 11 | Health summary | `prototypes/health.html` |
| 12 | Nutrition | `prototypes/nutrition.html` |
| 13 | Learn (Many / One / Empty) | `prototypes/learn.html` |
| 14 | More | `prototypes/more.html` |
| 15 | Attention | `prototypes/attention.html` |
| 16 | System (Public domain / Tailnet views) | `prototypes/system.html` |
| 17 | Automations | `prototypes/automations.html` |
| 18 | Apps | `prototypes/apps.html` |
| 19 | Universal launcher | `prototypes/launcher.html` |
| — | Shared system | `prototypes/assets/proto.css`, `prototypes/assets/proto.js` |
