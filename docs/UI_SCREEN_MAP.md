# GYST screen map — approval draft

Baseline: `feature/health-hardening` at `5c50011`. Visual references:
[component board](Design1.png), [screen board](<Design 2.png>).
All redesign phases below are **not started**; this task supplies specifications.
Existing means the current source supports the route/data, not that its new UI
is implemented or that production contains observations.

Status vocabulary: **Existing** = current route/feature; **Needs UI** = proposed
presentation/route using existing domain data; **Needs backend** = missing data
capability; **Future** = intentionally deferred. These may coexist in one row.
Shared shell, headers, cards and states apply throughout; the component column
lists additional reusable patterns. Paths marked proposed do not exist yet.

## Screen inventory

| Screen | Current / proposed route | Status and real data dependencies | Components | Phase |
|---|---|---|---|---|
| Today dashboard | `/`, `/today`; `/dashboard` redirects | Existing; Needs UI. Task occurrences, reminders, overdue tasks, prayer schedule/trackers, study blocks, ship stats. Health snapshot requires presentation wiring to existing HealthToday service; no focus scheduler exists in this view | StatCard, ProgressCard, ListRow, HealthSummaryCard, CalendarEventRow | UI-2 |
| Daily checklist | Today currently; proposed `/checklist` | Needs UI. Reuse occurrence/checklist and tracker data/actions. Group only by real task categories; Morning/Work/Evening are not new stored categories | ListSection, Checkbox, ProgressBar | UI-2 |
| Projects home | `/projects`, `?roadmap={id}` | Existing; Needs UI. Roadmap groups, main/child projects, progress, next due item, upcoming/locked states. Use Active/Completed/Upcoming with explicit paused/blocked handling; no separate Ideas model | ProjectCard, ProjectProgressRow, SegmentedControl | UI-3 |
| Galaxy | `/galaxy`, data at `/galaxy/data` | Existing; Needs UI. Projects/child moons with categories, positions, progress and links. Life-domain grouping is proposed presentation derived from categories, not a stored domain graph | GalaxyDomainNode, BottomSheet, NavigationRow, ErrorState | UI-4 |
| Project detail | `/projects/{id}` | Existing; Needs UI. Items/proof, children/stops, notes, roadmap metadata, external work, dates/status. Milestones belong to roadmap features; expose only actual associations | ProjectProgressRow, SubNavigationTabs, ListRow | UI-3 |
| Health summary | `/health` | Existing; Needs UI. HealthToday, nullable daily snapshot, cycle and observed energy; existing gym consistency/goals/tasks retained | HealthSummaryCard, MetricRingCard, MacroProgressBar, StatCard | UI-5 |
| Nutrition | `/health#nutrition`; proposed `/health/nutrition` | Needs UI for totals/macros from snapshot/cycle. Needs backend for individual meals, next-meal plan, recipes, portion engine and nutrition insights | NutritionCard, CalorieRing, MacroCard, MealCard | UI-6 |
| Training | `/health#training`; proposed `/health/training` | Empty placeholder exists; Needs UI and backend for provider-neutral Hevy sessions, history, sets/reps/weights/PRs and sync status. Gym tracker is not a Hevy workout | WorkoutCard, TrainingProgressCard, IntegrationStatusRow | UI-7 |
| Workout detail | proposed `/health/training/workouts/{id}` | Future; Needs backend. No normalized workout model or handler; do not show a functioning Start Workout action | WorkoutCard, ListRow, MetricCard | Future after Hevy |
| Health progress | `/health#progress`; proposed `/health/progress` | Needs UI; weight entry model/repository/service exist but have no UI. Needs backend queries/wiring for historical nutrition/activity ranges and adherence; strength requires Hevy | TrendCard, MiniLineChart, MiniBarChart, SegmentedControl | UI-8 |
| Current health plan / edit | `/health/plan`, `?id={id}`, `?new=1` | Existing; Needs UI. HealthCycle title/dates/goal, nutrition/energy/training/steps targets and notes. Adherence requires backend. Cycles allow any duration; never force real records to 90 days | HealthPlanCard, MetricCard, ProgressBar, labelled fields | UI-9 |
| Activity options | proposed `/health/activity-options` | Future; Needs UI for option list, backend/specification for duration/energy recommendations. No manufactured calorie estimates | ActivityOptionRow, EmptyState | Future |
| Courses list | `/courses` | Existing; Needs UI. Course status/provider/instructor/progress, next learning block/deadline, site URL. No cover asset pipeline; use SVG fallback | CourseCard, ProgressBar, SegmentedControl | UI-10 |
| Course detail | `/courses/{id}` | Existing; Needs UI. Learning blocks, catch-ups, course/block notes and URLs, dates/repetition, progress. Preserve completion/reopen/postpone/delete/archive/status/forms | LessonRow, SubNavigationTabs, ProgressCard | UI-10 |
| Lesson / block detail | currently inline `/courses/{id}#block-{id}` | Existing inline; standalone screen Future. Blocks contain notes/URLs and status; no new locking engine, lesson player or resources database | LessonRow, ListSection | UI-10 inline; standalone future |
| Reminders | `/reminders` | Existing today/overdue read view; Needs UI. ReminderViews and reminder repository. All/Scheduled/Completed queries/actions are not exposed by this handler; require separate backend scope | ReminderRow, ListSection, SegmentedControl | UI-11 |
| Notes | proposed `/notes` | Needs UI empty state; Needs backend for standalone searchable/favorited notes. Existing project/course/block notes remain in their owners | NoteCard, SearchField, EmptyState | UI-11 |
| Calendar / schedule | `/monthly` current; proposed `/schedule` | Needs UI. Reuse TaskOccurrencesBetween and existing calendar/course scheduling services. `/calendar` is the private-feed dispatcher, not a page. Unified daily timeline needs handler wiring; workouts unavailable | CalendarEventRow, SegmentedControl, ListSection | UI-11 |
| Global analytics | `/analytics` | Existing; Needs UI. Dashboard aggregates daily/monthly task completion, streak, project and yearly-goal charts. Study/Health adherence/life-domain comparisons and arbitrary ranges need backend | AnalyticsMetricCard, MiniLineChart, MiniBarChart | UI-12 |
| More | proposed `/more` | Needs UI; existing destinations require no new business data. Initial route in UI-1, richer sections in UI-13 | NavigationRow, ListSection | UI-1 / UI-13 |
| Settings | `/settings` | Existing; Needs UI. Prayer/location/timezone, calendar token feed management, appearance/motion and data paths currently shown. Reorganize without losing controls; remove private technical presentation from general profile | SettingsRow, Toggle, TextField, SelectField | UI-13 |
| Integrations | proposed `/integrations` | Needs UI and some backend wiring. HealthSyncState service exists, no status handler; Web Push browser status exists. Apple Shortcut onboarding absent; MFP is indirect via Apple Health; Hevy absent. Existing external-link providers stay available | IntegrationStatusRow, ErrorState | UI-13 |
| Profile | proposed `/profile` | Needs UI. PlayerStats/ship progression available; name/avatar account profile fields not exposed. Use GYST identity/real XP until profile storage exists | ProfileHeader, StatCard, NavigationRow | UI-13 |
| Habits | Today/Health trackers now; proposed `/habits` only if useful | Existing trackers; Needs UI for consolidated view, not a new habit engine | Checkbox, ListRow, ProgressCard | UI-13 |
| Help / About | proposed `/help`, `/about` | Needs UI; static truthful app guidance, no fabricated version/integration status | ListSection, NavigationRow | UI-13 |

## Existing supporting screens and actions to retain

| Screen / route family | Existing behavior | New navigation / preservation |
|---|---|---|
| `/tasks`, `/tasks/new`, `/tasks/{id}`, `/tasks/{id}/edit` | Filtered task list; create/detail/edit; recurrence, proof and checklist | More → All tasks; page-level create; use shared fields/rows |
| `/tasks/quick`, task complete/skip/archive/checklist actions | Existing form/JS workflows | Keep action URLs, occurrence dates, return anchors and quick-capture parsing |
| `/projects/new`, `/projects/{id}/edit` and status/archive/items actions | Hierarchy, roadmap assignment, dependencies, proof/rewards, external links | Projects actions and detail menus; no change to unlock rules |
| `/roadmap`, `/roadmaps/new`, `/roadmaps/{id}/edit` | Roadmap map/forms, selection, weekly review, import/export, project/milestone completion | Projects and More → Roadmaps; retain route map and all operations |
| `/goals`, `/goals/new`, `/goals/{id}` and progress/archive actions | Real goals and logged progress | More; Health links to health-category goals |
| `/courses/new`, `/courses/{id}/edit` and blocks/status/archive actions | Course management, catch-up and scheduling | Courses; preserve next-block and external lesson links |
| `/monthly` | Current-month task list, completion and chart | More → Monthly; schedule is additional presentation |
| `/archive` | Archived/completed domain records | More → Archive; no lost recovery/navigation paths |
| `/plan-import` | Existing AI plan import/preview workflow | More and creation/search menus |
| `/login`, `/logout` | Authentication code exists on inspected branch | Restyle separately using shared forms; preserve enforcement and CSRF |
| `/settings/prayer`, `/settings/calendar-token` | Prayer settings and feed-token management | Grouped Settings; never expose tokens in Profile or screenshots |
| `/calendar/feed/...`, `/calendar.ics`, course/project ICS routes | Token feeds and gated public export | Existing Settings/course links; never hijack `/calendar` for new page |
| `/trackers/{key}/toggle` | Prayer/Athkar/gym mutations | Keep current forms, dates, keyboard behavior and async feedback |
| `/external-links` and per-link edit/delete/sync | GitHub/Jira/Slack/other links and provider synchronization | Preserve on project/item detail |
| `/api/health/sync`, `/api/push/...` | Ingestion/push APIs | No API redesign for UI; never fake connection state |
| `/manifest.webmanifest`, `/service-worker.js`, `/static/*`, `/favicon.ico` | PWA shell, push, assets | Update names/versions consistently; preserve registration and scope |

Source routing authority: `internal/handlers/handlers.go` (ServeHTTP and screen
handlers), `courses.go` (blocks and calendar feeds), `health_plan.go`,
`health_api.go`, and `internal/app/app.go`. Forms/templates specify action suffixes;
this table groups actions rather than proposing any endpoint removal.

## Current implementation versus the boards

- `layout.html` is the real shared layout; the handoff's `base.html` reference
  is stale. It contains XP HUD, Search/quick task dialog, Add menu and five primary
  items. Projects currently points to Galaxy; More is a `details` popup.
- `styles.css` has dark defaults, an optional light theme, repeated later token
  overrides and multiple breakpoint generations. Several production selectors
  use `:root` specificity. UI-0 must be opt-in, not a global override pile.
- `app.js` owns prayer alerts, push registration, quick capture, copy, motion and
  completion effects. `charts.js` also owns trackers, theme, toast and scroll
  memory. Do not remove it simply because its name sounds chart-only.
- `health.js` exposes Day/Week/Month/Year switches; historical selections show
  explanatory placeholders, not aggregated data. Replace these with honest
  unavailable states until real history views exist.
- The Health ring currently uses a fixed CSS border arc. The new component must
  use actual progress or a neutral no-data ring. Existing Present/stale handling
  stays in `health_view.go`; no template arithmetic.
- `galaxy.js` dynamically imports local Three.js, fetches `/galaxy/data`, builds
  project planets/moons and provides selection, desktop pointer roaming, wheel
  zoom and accessible list navigation. Mobile touch pan/zoom needs dedicated
  work; preserve its existing scene instead of substituting a flat list.
- The service worker precaches mismatched old asset versions and Galaxy vendor
  files, and its fetch path excludes `/health/plan` but not the summary. UI-1
  explicitly addresses asset consistency and private-page offline behavior.
- Health cycles support arbitrary inclusive dates; the current new-plan default
  is 84 days after start. “90-Day Plan” is a visual feature name, not evidence
  that stored duration is 90 days. Display real dates/duration without schema edits.

## Phase delivery and approval boundaries

| Phase | Reviewable outcome | Gate / limitation |
|---|---|---|
| UI-PLAN (this task) | These documents plus reference boards and status ledger | Jay reviews exact UI-0/UI-1 file lists in UI_SYSTEM.md |
| UI-0 | Tokens, primitives, typed inputs and development gallery | Compare with component board; approve on iPhone before shell |
| UI-1 | GYST shell, five destinations, More, safe areas and PWA asset policy | All old destinations/actions reachable; offline/auth checks |
| UI-2 | Today and dedicated checklist | Preserve prayer/Athkar/study/XP; no fabricated categories |
| UI-3 | Projects list/detail | Preserve hierarchy, proof, external links, milestones and unlocks |
| UI-4 | Mobile Galaxy | Same Three.js behavior/data; category presentation, touch and desktop review |
| UI-5 | Health shell/summary | Missing/zero/partial/stale states and separated energy concepts |
| UI-6 | Nutrition | Real aggregate macros first; meal/recipe features remain honest empty states |
| UI-7 | Training | Empty/onboarding UI until separately approved Hevy backend exists |
| UI-8 | Progress | Real accessible charts only for supported data/ranges; record backend prerequisites |
| UI-9 | Plan | Existing cycle editor and real dates/targets; no fictional adherence |
| UI-10 | Courses | Existing learning blocks and catch-ups; no invented locking |
| UI-11 | Reminders/calendar/notes | Existing data first; new capabilities are separate backend tasks |
| UI-12 | Analytics | Global LifeOps metrics, supported ranges only |
| UI-13 | More/settings/integrations/profile | Real status, no private technical information in profile |
| UI-14 | Tablet/desktop consistency | Consolidate remaining legacy breakpoints after migration |
| UI-15 | Accessibility/performance/consistency | Device review, keyboard, charts, stale/offline/error states |

Small commits and approval between phases. UI priority within later refinement
is Health, while UI-0/UI-1 establish shared behavior first. Do not silently build
missing backend work inside a styling phase.

## Handoff discrepancies and operational work

The final user instruction requests planning and approval before implementation;
A2 is not bundled into UI planning. Trusted TLS remains a separate work item.
`HANDOFF_TO_CODEX.md`/`CODEX_PROMPT.md` say HTTPS and OPS1 operator steps are done;
older task/blocker sections still say blocked. Neither is a fresh operational
verification. The source contains session/auth enforcement and the Health sync
POST handler despite prose calling them absent/deferred. Do not reimplement,
disable or deploy these based on the prose. Resolve the actual deployment state
when resuming A2; no TLS or production claims are made by this planning task.

Inspection used source and both supplied boards. No live production database or
browser was opened. Original workspace was `master` with a user Compose edit;
planning work uses an isolated `/tmp/gyst-ui-plan` worktree on the requested
branch. No runtime code or schema changes are included.
