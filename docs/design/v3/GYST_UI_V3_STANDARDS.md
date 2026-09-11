# GYST UI/UX Design System v3

**Status:** Approved visual direction, implementation reference only.  
**Target:** GYST LifeOps — Go templates + CSS + small vanilla JS modules.  
**Primary device:** iPhone PWA.  
**Secondary:** tablet and desktop browser.  
**Design direction:** Apple-like clarity and spacing, openGym-like workout usability, and GYST's own Galaxy identity.

This folder is the canonical visual implementation reference for the next GYST UI pass. It intentionally does **not** change backend behaviour, routes, data contracts, migrations, XP integrity, Health ingestion, or external integrations. Codex/Claude should implement screen-by-screen without deleting or weakening existing functionality.

## Reference images

- `assets/gyst_app_experience_showcase.webp` — approved screen-family direction, including the mobile Galaxy.
- `assets/gyst_ui_system_style_guide.webp` — approved component/tokens direction.

The images are inspiration and visual targets, not pixel-perfect contracts. Where an image conflicts with this written specification, **the written specification wins**.

## Canonical documents

1. `UI_STANDARDS.md` — tokens, components, responsive behaviour, accessibility, libraries, interaction rules.
2. `SCREEN_SPECS.md` — page-by-page specification for every existing or planned user-facing screen.
3. `IMPLEMENTATION_GUIDE.md` — how to port the system into the current Go/PWA codebase without breaking behaviour.

## Current product navigation

The five primary destinations are:

1. **Today**
2. **Projects**
3. **Health**
4. **Courses**
5. **More**

On iPhone these are a fixed bottom tab bar. On wider screens the same information architecture may render as a sidebar or top navigation, but the labels and route meaning must stay consistent.

## Core product principle

GYST should feel calm, native, legible, and information-dense without looking crowded. One element per screen may be visually dominant. Everything else should support it.

- Today: today's actions are dominant.
- Projects: active work or Galaxy is dominant.
- Galaxy: the interactive project universe is dominant.
- Health: today's health summary is dominant.
- Courses: active learning is dominant.
- Analytics/Progress: the trend is dominant.

Do not turn every block into a glowing dashboard tile. Avoid decorative UI that does not communicate state or afford an action.


---

# GYST UI Standards v3

## 1. Purpose and authority

This document defines the production UI contract for GYST. It applies to all server-rendered Go templates, CSS, JavaScript-enhanced controls, PWA states, and future Health/Hevy views.

The current app already uses Go templates, one shared stylesheet, lightweight JavaScript, and a dynamically loaded Three.js Galaxy. Preserve that architecture unless a later architecture decision explicitly changes it.

### Non-goals

- Do not introduce React, Vue, Svelte, Tailwind, Bootstrap, or a front-end build pipeline merely to achieve this design.
- Do not add a runtime CDN dependency.
- Do not copy Apple's proprietary assets, SF Symbols files, or San Francisco font files.
- Do not clone openGym markup or styling. It is only interaction/UX inspiration.

---

## 2. Design principles

### 2.1 Apple-like, not Apple-copied

Use the qualities users expect from iOS:

- large, clear page titles;
- generous whitespace;
- grouped cards/lists;
- subtle separators;
- strong hierarchy;
- compact segmented controls;
- 44×44 pt minimum touch targets;
- sheets and confirmation dialogs for short decisions;
- restrained animation;
- content-first layouts.

Use our own markup, icons, colours, and components.

### 2.2 One strong focus per page

Each screen gets one dominant object. Do not make five equally loud cards compete for attention.

### 2.3 Data should answer a question

Every metric must help answer one of:

- What should I do now?
- What is left today?
- Am I on track?
- What changed?
- What is next?

If a metric cannot answer one of those, it belongs in a drill-down page rather than the summary.

### 2.4 Server first

The page must remain understandable with JavaScript unavailable. JavaScript enhances filtering, charts, Galaxy navigation, sheets, and quick actions, but should not own core data or navigation.

---

## 3. Supported visual modes

### 3.1 Theme

Default to `color-scheme: light dark` and follow the OS preference. The user may later override in Settings.

**Exception:** the Galaxy stage is intentionally dark/space-themed in both themes.

### 3.2 Light tokens

```css
:root {
  --bg: #f2f2f7;
  --surface: #ffffff;
  --surface-raised: #ffffff;
  --surface-subtle: #f7f7fa;
  --label: #111114;
  --label-secondary: #636366;
  --label-tertiary: #8e8e93;
  --separator: rgba(60, 60, 67, .18);
  --separator-strong: rgba(60, 60, 67, .30);
  --tint: #0a84ff;
  --green: #30b76f;
  --orange: #ff9f0a;
  --red: #ff453a;
  --yellow: #ffd60a;
  --purple: #8e5af7;
  --indigo: #5e5ce6;
  --cyan: #32ade6;
  --pink: #ff375f;
}
```

### 3.3 Dark tokens

```css
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #000000;
    --surface: #1c1c1e;
    --surface-raised: #2c2c2e;
    --surface-subtle: #151517;
    --label: #f5f5f7;
    --label-secondary: #aeaeb2;
    --label-tertiary: #8e8e93;
    --separator: rgba(84, 84, 88, .65);
    --separator-strong: rgba(99, 99, 102, .85);
    --tint: #0a84ff;
  }
}
```

Semantic colours may be used only when they carry meaning. Do not colour every category arbitrarily.

---

## 4. Typography

### 4.1 Font stack

Use the system stack only:

```css
font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
```

On Apple devices this naturally uses Apple's system font without bundling proprietary font files.

### 4.2 Scale

| Token | Mobile | Desktop | Weight | Use |
|---|---:|---:|---:|---|
| `display` | 34/40 | 40/46 | 700 | rare hero metric/title |
| `page-title` | 28/34 | 32/38 | 700 | page heading |
| `section-title` | 20/26 | 22/28 | 700 | section heading |
| `card-title` | 17/22 | 18/24 | 600 | card heading |
| `body` | 16/22 | 16/24 | 400 | normal text |
| `callout` | 15/20 | 15/21 | 500 | list/support text |
| `caption` | 13/18 | 13/18 | 400 | secondary info |
| `micro` | 11/14 | 11/14 | 600 | labels, compact metadata |
| `metric-xl` | 32/36 | 36/40 | 700 | calorie/weight primary metric |
| `metric-lg` | 24/30 | 28/34 | 700 | stat tile |

Use tabular numerals for changing metrics:

```css
.metric, .num { font-variant-numeric: tabular-nums; }
```

Avoid uppercase tracked eyebrows as a default. Use sentence case.

---

## 5. Spacing and sizing

Base grid: **4 px**. Preferred spacing values: `4, 8, 12, 16, 20, 24, 32, 40`.

- Page horizontal padding mobile: 16 px.
- Page horizontal padding tablet: 24 px.
- Desktop content max-width: 1040 px for normal pages; Galaxy may use full viewport.
- Card internal padding: 16 px standard, 20 px roomy.
- Section gap: 24–32 px.
- Row minimum height: 52 px.
- Button minimum height: 44 px; primary buttons normally 48 px.
- Touch target minimum: 44×44 px even if visible icon is smaller.

Safe area:

```css
padding-bottom: calc(var(--bottom-nav-h) + env(safe-area-inset-bottom));
```

---

## 6. Radius, borders, elevation

### Radius

- Input/button: 10–12 px.
- Standard card: 16 px.
- Hero card: 20–22 px.
- Sheet top corners: 24–28 px.
- Pills/chips: `999px`.

### Borders

Use one-pixel separators. Prefer borders and background differences to heavy shadows.

### Shadows

Use sparingly. Default card shadow in light mode:

```css
box-shadow: 0 1px 2px rgba(0,0,0,.04);
```

Raised sheet/modal may use:

```css
box-shadow: 0 18px 50px rgba(0,0,0,.18);
```

No neon glow outside Galaxy.

---

## 7. Navigation architecture

### 7.1 Mobile bottom tabs

Fixed five-item bar:

1. Today — `/today`
2. Projects — `/galaxy` or projects landing according to final route mapping
3. Health — `/health`
4. Courses — `/courses`
5. More — menu/sheet or `/more`

Rules:

- Bottom bar height: 56 px + safe area.
- Selected item uses tint colour and 600 weight.
- Unselected item uses secondary label colour.
- Icon 22–24 px; label 10–11 px.
- Never hide the tab bar while scrolling ordinary pages.
- A temporary full-screen mode may hide it only for immersive Galaxy interaction or modal workout handoff, with an obvious exit affordance.

### 7.2 Top app bar

Mobile page header:

- optional back button left;
- large title or compact title according to context;
- at most two actions right;
- search and add are the usual actions for list pages.

### 7.3 Secondary navigation

Use segmented controls for 2–5 peer views.

Health: `Summary | Nutrition | Training | Progress | Plan`.

Time range: `Day | Week | Month | Year`.

Projects: `Galaxy | List | Roadmap | Completed`.

Courses: `In Progress | Library | Completed`.

On narrow widths, health tabs may horizontally scroll only if all five cannot fit at 320 px. Prefer fit-to-width first.

---

## 8. Approved libraries and dependencies

### 8.1 Front-end framework

**None.** Keep Go templates + CSS + vanilla JS.

### 8.2 Icons

Use **Lucide** icon geometry, vendored as individual SVGs or an SVG sprite under `web/static/icons/`.

- No CDN.
- Do not ship the Lucide JS runtime if icons can be server-rendered.
- Stroke width target: 1.75–2 px.
- Standard icon sizes: 16, 20, 24 px.

Do not use SF Symbols assets in the web app.

### 8.3 Charts

Prefer extending the existing `web/static/js/charts.js` rather than adding Chart.js.

Allowed chart primitives:

- line chart;
- area/line trend;
- vertical bar chart;
- donut/ring via SVG/CSS;
- progress bars;
- mini sparkline.

No 3D charts. No pie charts with many categories.

### 8.4 Galaxy

Keep the already-vendored **Three.js** and dynamically import it only on the Galaxy route. Do not load Three.js on ordinary pages.

### 8.5 Motion

CSS transitions and small vanilla JS only. No GSAP or animation framework.

---

## 9. Core component catalogue

Every component should have a predictable server-rendered base and optional JS enhancement.

### `AppShell`

Owns top bar, page content, bottom tabs, safe area, desktop adaptation.

### `TopBar`

Back/title/actions. Avoid putting XP, search, add, profile, notifications, and settings all in one bar.

### `BottomTabBar`

Five fixed primary destinations.

### `PageTitle`

Title + optional date/subtitle + maximum two actions.

### `SectionHeader`

Title left, optional `See all`/action right.

### `Card`

Standard grouped surface. Variants: `standard`, `hero`, `inset`, `danger`, `galaxy`.

### `ListGroup` / `ListRow`

Grouped rows with optional leading icon/status, primary label, secondary label, trailing value/chevron.

### `PrimaryButton`

Filled tint, 48 px high, 12 px radius.

### `SecondaryButton`

Tint text or outlined surface.

### `DestructiveButton`

Red, only in destructive confirmation context.

### `IconButton`

44×44 target with 20–24 px icon.

### `SegmentedControl`

Peer view selector. The selected segment gets white/raised fill on light grouped background, or tint where needed.

### `FilterChip`

Small optional filter; not a substitute for segmented navigation.

### `TaskRow`

Checkbox/status, title, optional time/project/priority, optional chevron.

### `ProgressBar`

4–8 px thick. Default tint blue; semantic colours only when meaningful.

### `MetricRing`

Used for calories/goal completion. SVG ring, numeric centre, short label. Do not put more than one primary ring per card.

### `MetricTile`

One primary number, short label, optional trend. 2-up or 3-up grid on mobile.

### `MacroBar`

Label, consumed/target, horizontal progress. Protein blue/cyan, carbs orange, fat purple, fiber green. Keep colour mapping stable across all screens.

### `MiniTrendChart`

Small line/sparkline with no dense axes. Full charts belong in drill-down screens.

### `TrendChart`

Day/week/month/year line or bar chart with accessible summary text.

### `ProjectCard`

Project icon, title, category/roadmap, completion %, progress bar, next milestone where useful.

### `GalaxyWorld`

Interactive Three.js planet mapped to a project. Visual size must not encode importance unless documented; progress/state is communicated by halo/ring/label, not raw planet size alone.

### `CourseCard`

Optional cover, course title, category, progress, next lesson.

### `WorkoutHeroCard`

Today's Hevy workout summary, exercise count, expected duration, primary `Start workout`/`Open in Hevy` action.

### `IntegrationStatusRow`

Provider icon/name, state, last successful sync, warning if stale, action/chevron.

### `ActivityOptionRow`

Walking/swimming/cycling/etc., estimated intensity, optional estimated target duration, never presented as guaranteed calorie burn.

### `NutrientSummaryCard`

Calories ring + macro bars + food budget; no energy-balance claims unless Health observations are fresh.

### `NextMealCard`

Meal name, suggested budget, target protein/fiber need, primary `Plan meal` action.

### `90DayPlanCard`

Cycle dates, goal, targets, adherence summary, edit action.

### `SyncHealthCard`

Last success, last attempt, failure count, data freshness. Used in Health settings/integrations and optionally as a warning card on Summary.

### `EmptyState`

Short title, one sentence, one primary action. No mascot required.

### `ErrorState`

Plain explanation, retry if possible, support/log identifier only if useful.

### `Skeleton`

Use only for asynchronous enhanced regions; server-rendered pages normally do not need skeletons.

### `Toast`

Short transient success/failure message. No critical decisions in toasts.

### `BottomSheet`

Mobile actions, filters, quick create, activity substitution. Must trap focus and be dismissible.

### `ConfirmationDialog`

Only for destructive/irreversible actions.

---

## 10. Forms

- Label always visible; placeholder is not a label.
- Field height ≥ 44 px.
- Group related fields in inset sections.
- Inline validation appears below the field.
- Errors must not rely on red alone.
- Save action should be visible without excessive scrolling where practical.
- Numeric health targets display units next to/in the field.
- Destructive actions belong at the bottom in their own section.

---

## 11. Health-specific visual semantics

Stable colour mapping:

- Calories / active energy: orange.
- Food remaining / on-target state: green when appropriate.
- Protein: blue/cyan.
- Carbs: orange/yellow.
- Fat: purple.
- Fiber: green.
- Weight: blue trend line.
- Strength: green if improving, neutral if unchanged; never red merely because a value dropped without context.
- Resting energy: indigo or neutral blue.
- Steps: cyan/blue.

Health cards must distinguish:

1. **Food budget** — target calories minus consumed calories.
2. **Observed energy balance** — consumed minus observed burn.
3. **Projected balance** — future feature, clearly labeled projected.

Never visually collapse these into one number.

Freshness states:

- Current: normal display.
- Stale: value remains visible with `Last synced …` label and muted warning.
- Missing: show em dash / `No data` rather than zero.

---

## 12. Galaxy standards

The Galaxy is the one deliberately cinematic area.

### Mobile interaction

- Full-width dark stage near the top of Projects/Galaxy.
- Minimum stage height: 58vh; immersive mode may use full viewport.
- One-finger drag: orbit/pan.
- Pinch: zoom if performance remains smooth.
- Tap planet once: select and reveal context sheet/card.
- Tap primary action / second explicit tap: open project. Do not use ambiguous double-tap as the only way to navigate.
- A visible `List` control must always provide a conventional accessible alternative.

### Context card

When a world is selected show:

- project name;
- category/domain;
- progress;
- next mission/milestone;
- target date if present;
- `Open project` action.

On phone this becomes a bottom sheet or bottom anchored card, not a narrow desktop aside.

### Performance

- Three.js loads only on Galaxy.
- Respect `prefers-reduced-motion`.
- Cap pixel ratio to avoid unnecessary GPU load (for example `Math.min(devicePixelRatio, 2)`).
- Suspend animation when page/stage is hidden.
- Preserve the existing conventional list fallback if WebGL fails.

---

## 13. Responsive rules

### Phone: 0–599 px

- single column;
- bottom navigation;
- full-width cards;
- 2-up small metric tiles;
- sheets for secondary actions;
- Galaxy context as bottom card/sheet.

### Tablet: 600–899 px

- bottom tabs may remain;
- 2-column grids for metric/detail cards;
- larger Galaxy stage.

### Desktop: ≥900 px

- max content width 1040 px normal screens;
- side/top primary nav may replace bottom bar if desired, but routes/labels stay identical;
- Health summary can use 2-column sections;
- Galaxy can fill viewport with context panel on side.

No screen may require horizontal scrolling except intentionally scrollable chip/segment areas.

---

## 14. Accessibility

Target WCAG 2.2 AA.

Required:

- 44×44 minimum touch targets;
- keyboard reachable controls;
- clear focus rings;
- semantic headings in order;
- real `<button>`/`<a>` controls;
- form labels;
- `aria-current` on nav;
- charts accompanied by textual summaries;
- colour never sole indicator;
- reduced motion support;
- Galaxy accessible list fallback always present;
- minimum contrast 4.5:1 for normal text.

---

## 15. Motion

Duration ranges:

- tap/press feedback: 80–120 ms;
- simple panel/segment transition: 120–180 ms;
- sheet present/dismiss: 220–300 ms;
- Galaxy camera transition: 300–600 ms.

Do not animate layout continuously. Do not pulse metrics. Do not use infinite ambient animation outside Galaxy.

---

## 16. Loading, offline and stale data

PWA behaviour must be explicit.

- Static shell may be cached.
- Authenticated/private health HTML must not be stored in CacheStorage once auth hardening lands.
- When offline, show last locally available non-sensitive shell state and a clear offline banner; do not fake fresh Health data.
- A stale health provider displays its last sync time.
- Hevy unavailable: show `Hevy not connected` or stale last-sync state, not an empty workout as if no workout exists.

---

## 17. Copy style

Use plain language.

Good:

- `460 kcal left`
- `Last synced 18 min ago`
- `3 of 4 workouts`
- `No workout scheduled`
- `Plan dinner`

Avoid:

- `NUTRITIONAL OPTIMIZATION MATRIX`
- `MISSION STATUS: NOMINAL`
- excessive emoji or sci-fi language outside the Galaxy.

GYST's identity should come from structure and the Galaxy, not gimmicky copy.


---

# GYST Screen Specifications v3

This file is the page-by-page implementation reference. Screens are marked **Existing**, **Existing—redesign**, or **Planned**. Planned screens must not be treated as already implemented backend functionality.

## Route / screen matrix

| Screen | Status | Primary route / entry | Primary components |
|---|---|---|---|
| Login | Planned by M2.5A | `/login` | auth form, error banner |
| Today dashboard | Existing—redesign | `/today` | checklist, health snapshot, upcoming |
| Daily checklist / all tasks | Existing—redesign | `/tasks` | task rows, filters, quick add |
| Task detail | Existing—redesign | `/tasks/{id}` | detail sections, status, schedule |
| Task create/edit | Existing—redesign | `/tasks/new`, edit route | grouped form |
| Projects home | Existing—redesign | `/projects` | project cards, segmented view |
| Galaxy | Existing—redesign | `/galaxy` | Three.js stage, project context, fallback list |
| Project detail | Existing—redesign | `/projects/{id}` | overview, tasks, milestones, notes/metadata |
| Project create/edit | Existing—redesign | `/projects/new`, edit route | grouped form |
| Roadmaps | Existing—redesign | `/roadmap` | roadmap selector, milestones, progress |
| Courses list | Existing—redesign | `/courses` | course cards, filters |
| Course detail | Existing—redesign | `/courses/{id}` | overview, lessons, notes/resources |
| Course create/edit | Existing—redesign | `/courses/new`, edit route | grouped form |
| Health summary | Existing—major redesign | `/health` | energy, nutrition, training, activity |
| Health Nutrition | Planned UI / backend later | `/health/nutrition` | calories, macros, meals, next meal |
| Meal planner | Planned | `/health/nutrition/plan` | meal budget, recipe/food selection |
| Food search | Planned | `/health/foods` | search, source badges, nutrition rows |
| Recipe detail | Planned | `/health/recipes/{id}` | per-100g nutrition, serving calculator |
| Recipe create/edit | Planned | `/health/recipes/new` | ingredients, weights, cooked weight |
| Health Training | Planned UI / Hevy later | `/health/training` | workout hero, week, history, provider status |
| Workout detail | Planned | `/health/training/workouts/{id}` | exercises, sets, totals, Hevy link |
| Health Progress | Planned | `/health/progress` | weight, strength, activity, nutrition trends |
| Health Plan | Existing core form / redesign | `/health/plan` | 90-day plan targets |
| Activity options | Planned | `/health/activity-options` | walk/swim/bike/etc. options |
| Weight history | Existing data / planned page | `/health/weight` | chart, entries, add weight |
| Health integrations | Planned | `/health/integrations` | Apple Shortcut, Hevy, MFP status |
| Goals list | Existing—redesign | `/goals` | goal rows/cards |
| Goal detail | Existing—redesign | `/goals/{id}` | progress, history, linked tasks |
| Goal create/edit | Existing—redesign | `/goals/new`, edit route | goal form |
| Reminders | Existing—redesign | `/reminders` | reminder rows, schedule |
| Monthly / Calendar | Existing—redesign | `/monthly` | month calendar, agenda |
| Analytics | Existing—redesign | `/analytics` | period selector, trends, life domains |
| Archive | Existing—redesign | `/archive` | archived items by type |
| Plan import | Existing—redesign | `/plan-import` | paste/import preview, confirm |
| Settings / More | Existing—redesign | `/settings`, More entry | grouped settings |
| Command palette / search | Existing—redesign | global | search, quick task, navigation |
| Error / offline | Existing behaviour—standardize | global | error/empty/offline state |

---

## 1. Global app shell

### Mobile

Top: current page header. Bottom: fixed five-tab bar: Today, Projects, Health, Courses, More.

`More` opens a full page or sheet listing: Reminders, Goals, Analytics, Calendar/Monthly, Archive, Settings, Plan Import, and secondary utilities.

Global `+` belongs in context-specific headers rather than as a sixth bottom tab.

### Desktop

Primary navigation may become a left rail/top row. Keep route names and selected state consistent.

---

## 2. Today dashboard — `/today`

**Question answered:** What should I do today?

Order on mobile:

1. Greeting/date header + notification/search action.
2. Optional short focus message (one sentence, dismissible).
3. `Today's Checklist` card — show 4–7 immediate tasks and progress count.
4. `Health Snapshot` — calories food budget, steps, protein; tap opens Health.
5. `Upcoming` — next 2–4 timed items/reminders/course events.
6. `Projects / Focus` — at most two current projects or deep-work block.
7. `Next up` — single strongest recommendation.

Do not put full analytics or long charts here.

Empty state: `Nothing scheduled yet` with `Add task`.

---

## 3. Daily checklist / tasks — `/tasks`

**Question:** What is on my list, and what is done?

Header: `Daily Checklist` or `Tasks`, date/filter action, add button.

Segment/filter: `Today | Upcoming | All | Completed` where route/data supports it.

Group rows by `Morning`, `Work`, `Personal`, or by due date; never group merely for decoration.

Task row:

- completion control;
- title;
- optional time;
- small project/category label;
- priority only when non-default;
- chevron if detail exists.

Long-press is not required; all actions must be accessible through explicit UI.

---

## 4. Task detail

Hero area: title + status. Then grouped sections:

- schedule/due date;
- project/goal links;
- priority/category;
- checklist/subtasks if available;
- repeat configuration;
- proof/reward metadata if existing functionality uses it;
- created/updated metadata in a quiet footer.

Actions: Complete/Reopen, Edit, Delete (destructive section).

---

## 5. Task create/edit

Grouped form sections:

1. Basics — title, category.
2. Timing — due date/time, repeat.
3. Context — project, goal, priority.
4. Checklist/proof/reward only if those fields exist.

Sticky/fixed save button on mobile only if it does not cover content or keyboard.

---

## 6. Projects home — `/projects`

Header: `Projects`, search, add.

Segment: `Galaxy | List | Roadmap | Completed`.

`List` view shows active projects first. Project card:

- icon/domain colour;
- title;
- category/roadmap;
- completion % + progress bar;
- next milestone/task;
- target date if meaningful.

Completed projects appear under Completed, not mixed into active list by default.

---

## 7. Galaxy — `/galaxy`

**Question:** How is my life/project universe connected, and where should I go next?

This is a signature screen and may be visually richer than the rest of the app.

### Mobile layout

1. Compact dark header: `My Galaxy`, optional info/list control.
2. Interactive Three.js stage, nearly full width, 58–70vh.
3. GYST core at centre.
4. Project worlds around core. Optional domain grouping/rings only if readable.
5. Selected project opens a bottom context card/sheet with name, domain, progress, next mission, target, `Open project`.
6. Small page dots/domain filter only if they represent real distinct views. Do not add decorative dots.
7. `List` fallback/control always available.

### Gesture rules

- drag = orbit;
- pinch = zoom if supported smoothly;
- tap = select;
- explicit `Open project` = navigate;
- double tap must not be required.

### Accessibility

Preserve the current conventional project list below/alongside the stage as keyboard/screen-reader fallback.

### Desktop

Galaxy stage fills most viewport; context can become a side panel.

---

## 8. Project detail

Header: back, project icon/title, status badge, edit.

Progress hero: completion %, current phase, target date.

Segment/tabs: `Overview | Tasks | Milestones | Notes` where data exists.

Overview:

- short project brief;
- next milestone;
- roadmap/phase list;
- relevant goal;
- recent activity.

Tasks: project tasks grouped active/completed.

Milestones: vertical timeline with completed/current/upcoming.

Notes: if no real Notes domain exists yet, show existing project notes field only; do not invent a notes subsystem.

---

## 9. Project create/edit

Sections: Identity, Outcome, Dates, Roadmap/Goal, Visual (domain/icon/colour only if currently supported), Status.

Avoid large custom colour pickers unless project colours are already part of Galaxy data.

---

## 10. Roadmaps

Mobile: roadmap selector at top, then current milestone summary and vertical path/timeline.

Desktop: optional sidebar selector + main roadmap.

Each milestone: status, title, target date, completion. Current milestone is visually strongest.

---

## 11. Courses list — `/courses`

Header search + add.

Segment: `In Progress | Library | Completed`.

Course card:

- optional cover/icon;
- title;
- subject/category;
- progress;
- next lesson/module;
- next scheduled study block if available.

No giant thumbnails if they push useful data below the fold.

---

## 12. Course detail

Hero: title, subject, progress.

Tabs/sections: `Overview | Lessons | Notes | Resources` only when backed by real fields/features.

Primary next-action card: next lesson/task and duration.

Lesson rows: completion, title, duration/type.

Schedule section links into calendar/tasks rather than duplicating scheduling logic.

---

## 13. Course create/edit

Basics, dates/schedule, progress structure, notes/resources. Keep form grouped and short; advanced fields collapse under `More options`.

---

# HEALTH FAMILY

## 14. Health summary — `/health`

**Question:** Where do I stand today?

Header: `Health`, settings/integrations action.

Top control: `Day | Week | Month | Year`.

Health secondary nav: `Summary | Nutrition | Training | Progress | Plan`.

### Day order

1. `Today's Energy` hero:
   - dietary calories / calorie target;
   - food budget remaining when nutrition data is current;
   - observed active energy;
   - observed resting energy;
   - observed total burn;
   - energy balance only with correct freshness/semantics;
   - status label.
2. `Nutrition` card: protein/carbs/fat/fiber bars.
3. `Today's Training` workout hero from Hevy once connected; otherwise connection/empty state.
4. 2-up metric tiles: steps, active energy, exercise minutes, weight/latest.
5. `Next Meal` if meal planning data exists.
6. Sync freshness warning only when stale/failed.

### Week/month/year

Replace operational cards with trends/adherence. Do not just stretch day cards across longer periods.

---

## 15. Health Nutrition — `/health/nutrition`

Top: today/date selector.

Hero: calories consumed/target + remaining food budget when fresh.

Macros: stable colour bars and consumed/target values.

Meals section: Breakfast, Lunch, Dinner, Snacks. Each row shows known calories and major macro summary if available.

Next Meal card: meal name, suggested budget, protein/fiber remaining, `Plan meal`.

Recent foods/recipes section.

Insights are secondary and deterministic. Do not add AI copy until assistant phase exists.

---

## 16. Meal planner — planned

Inputs:

- meal type;
- remaining daily target;
- recipe/food selection;
- optional desired calorie budget.

Output:

- recommended grams;
- calories/macros for that serving;
- after-meal remaining calories/macros;
- source quality label: Saved Recipe / Database / Estimate.

Primary actions: `Use serving`, `Adjust grams`.

---

## 17. Food search — planned

Search field pinned near top.

Rows show: food name, brand/source, kcal per 100 g, optional protein. Source badge is quiet but visible.

Filters: Saved, Canadian Nutrient File, USDA, packaged/open source if enabled.

No giant nutrition panels in search results; detail opens separately.

---

## 18. Recipe detail — planned

Hero: recipe title + Saved/Estimate badge.

Primary nutrition per 100 g.

Serving calculator with grams field and computed calories/macros.

Ingredients list and total cooked weight.

Actions: Plan meal, Edit, Duplicate.

---

## 19. Recipe create/edit — planned

Step-like grouped form:

1. Name/details.
2. Ingredient rows (food + grams).
3. Finished cooked weight.
4. Review computed per-100g nutrition.
5. Save.

Never require the user to manually calculate totals.

---

## 20. Health Training — planned Hevy UI

Header: Training. Provider status row near top if not healthy.

`Today's Workout` hero:

- routine/workout name;
- expected duration if available;
- exercise count;
- scheduled status;
- `Open in Hevy` or `Start workout` deep-link where supported.

`This Week`: sessions completed / target, compact day dots.

`Recent Workouts`: name, date, duration, volume/PR if available.

`Strength Progress`: key exercises with trend.

GYST is not the canonical set logger while Hevy is chosen; detailed live logging remains in Hevy unless product direction changes.

---

## 21. Workout detail — planned

Imported from Hevy/local normalized cache.

Show:

- workout name/date/duration;
- exercises;
- sets (weight/reps/RPE/RIR only if provided);
- total sets/volume;
- PR markers;
- source = Hevy;
- `Open in Hevy` action if deep link is supported.

Read-only in first integration version.

---

## 22. Health Progress — planned

Top dimension segment: `Weight | Strength | Activity | Nutrition`.

Secondary range: `1M | 3M | 6M | 1Y | All` where enough data exists.

Weight: latest weight, change, line chart, entries.

Strength: selected exercise trend, estimated 1RM only if computed by a documented formula and labeled estimate; otherwise use direct top-set/volume trends.

Activity: steps, active energy, exercise minutes, sessions.

Nutrition: calorie/protein adherence averages and days on target.

Every chart has an accessible text summary.

---

## 23. 90-Day Plan — `/health/plan`

Hero: current cycle title, inclusive date range, week count/progress.

Sections:

- Goal;
- Nutrition Targets;
- Training targets;
- Activity targets;
- Energy target;
- Notes.

Actions: Edit current plan, view previous cycles.

UI accepts human-friendly positive `Target deficit` only while the backend domain represents signed energy balance internally; surplus support must have a separate explicit UI if/when exposed.

---

## 24. Activity Options — planned

Use when user cannot/do not want to perform scheduled strength training or needs an optional activity suggestion.

Header message explains target/gap as an estimate, never a command to punish food intake.

Rows: Walking, Running, Cycling, Swimming, Elliptical, Other Activity.

Each row: intensity label, estimated duration range only if derived from user data/known estimate, optional target active-energy delta.

Primary action should guide starting an Apple Watch workout; GYST reads resulting Active Energy later. Do not claim exact calorie burn in advance.

---

## 25. Weight history

Latest weight hero + trend.

Chart and list of entries. `Add weight` manual action remains even if Apple Health sync is later added.

Show source per entry if multiple sources exist.

---

## 26. Health Integrations

Grouped provider rows:

- Apple Health via GYST Shortcut — connected/not connected, last sync, diagnostics.
- MyFitnessPal — represented as nutrition source through Apple Health, not as direct API unless that changes.
- Hevy — API status once implemented.

Diagnostics section: last successful sync, last attempt, consecutive failures, missing-day grid where backend supports it.

Never display secret tokens after initial creation.

---

# GOALS / REMINDERS / ANALYTICS / MORE

## 27. Goals list

Active first, completed separate. Each goal row: title, type, progress, target/current value, due date if any.

Health cycles remain in Health Plan, not duplicated as generic goals unless linked intentionally.

---

## 28. Goal detail

Progress hero, history entries, linked tasks/projects, target metadata. Edit action. Add progress entry action.

---

## 29. Goal create/edit

Title/type, target/current, unit, dates, project/category links as current data allows.

---

## 30. Reminders

Segment: `Today | Scheduled | All` if data supports it.

Rows: reminder title, next time/date, recurrence, completion/acknowledgment state.

Use notification icon sparingly; the row itself carries meaning.

---

## 31. Monthly / Calendar

Top month strip with selected day. Below, agenda list for selected date.

Events from tasks/courses/reminders/prayer should be visually distinguishable by small semantic markers, not full rainbow cards.

Desktop can use month grid + right agenda.

---

## 32. Analytics

Period selector: Week / Month / Year.

Summary first: completed tasks, project progress, course progress, health adherence when available.

Life-domain section uses a small stable colour key.

Charts should be few and meaningful. Prefer bars/lines over decorative radar charts.

---

## 33. Archive

Filter by type. Rows/cards show archived/completed date and source type. Restore action only if feature supports it.

---

## 34. Plan Import

Large text/paste field or file input (existing behavior). Then preview screen/table of items to create, grouped by type. User must confirm before writes.

Errors identify which input section failed. Never silently drop unrecognized lines.

---

## 35. More / Settings landing

More list groups:

**Plan & organize** — Reminders, Goals, Calendar/Monthly, Analytics, Archive.

**System** — Settings, Integrations, Appearance, Notifications.

**Data & tools** — Plan Import, export/backup only if user-facing later.

**About** — About GYST/version.

Do not show fake profile/social features unless implemented.

---

## 36. Settings

Grouped sections:

- General: timezone, date/time preferences.
- Appearance: system/light/dark, motion preference if exposed.
- Notifications/push.
- Prayer configuration (existing functionality).
- Calendar feed/integration (existing functionality).
- Security/auth once implemented.
- Developer/diagnostics only if deliberately exposed.

Sensitive tokens use masked state and regenerate/revoke flows; never render token values repeatedly.

---

## 37. Command palette / global search

Desktop shortcut and mobile search entry.

Search commands, routes, tasks/projects/courses. Quick task capture remains supported.

On mobile this should present as a full-screen search sheet rather than a tiny modal.

---

## 38. Login — planned M2.5A

Simple page, no application data behind it.

Logo/title, password field, primary Sign In button, clear error state. PWA standalone mode must preserve session after successful login.

No remember-me checkbox if sessions already roll securely.

---

## 39. Offline/error states

### Offline

Banner: `You're offline`. Ordinary cached shell navigation may work, but stale private Health data must not be presented as current.

### 404

Plain title, sentence, `Go to Today`.

### 500

Plain failure, retry/back action, no stack trace.

### Empty list

Contextual action: `Create project`, `Add course`, etc.

---

## 40. Page approval rule

Codex should implement one family at a time and compare against the approved reference images plus this written spec. No backend behavior may be removed just to match the mockup. If a mockup contains a feature not present in the backend, render an honest disabled/empty/planned state or defer the element; do not invent data.


---

# GYST UI v3 — Implementation Guide for Codex

## 1. Safety first

This design work is independent from the M2.5/M3 hardening sequence. Do not mix production UI rewrites into a security/database hardening commit.

Implement UI on a dedicated branch after the relevant hardening baseline is approved/merged or explicitly coordinate the branch base.

Hard constraints:

- no destructive database migration for UI work;
- do not reset `data/lifeops.db`;
- preserve XP/reward integrity;
- preserve existing routes/actions unless a separate route migration is approved;
- preserve CSRF/same-origin/auth behavior once hardening lands;
- no Health API weakening to make UI easier;
- tests use temp DBs.

---

## 2. Current codebase mapping

Production UI lives under:

- `web/templates/`
- `web/static/css/styles.css`
- `web/static/js/app.js`
- `web/static/js/charts.js`
- `web/static/js/galaxy.js`
- vendored Three.js under `web/static/js/vendor/`

The current Galaxy already has a WebGL canvas, a project-data fetch, an accessible conventional list, and a fallback path. Preserve those capabilities while restyling/reworking mobile interaction.

The existing `docs/design/v2/` remains historical reference. v3 supersedes it for new implementation decisions.

---

## 3. Recommended file organization

Do not create a component framework. Use semantic partials/classes.

Suggested production structure where Go template organization allows:

```text
web/
  templates/
    layout.html
    partials/
      bottom_tabs.html
      top_bar.html
      health_tabs.html
      metric_card.html
      list_row.html
      empty_state.html
      integration_row.html
    today.html
    galaxy.html
    projects_list.html
    project_detail.html
    ...
  static/
    css/
      styles.css
    js/
      app.js
      charts.js
      galaxy.js
      health.js
    icons/
      lucide/*.svg or sprite.svg
```

If the current template loader cannot easily support nested partials, keep files flat but retain the same conceptual component boundaries.

---

## 4. CSS architecture

Use CSS custom properties for tokens. Avoid large utility-class proliferation.

Naming convention:

- layout: `.app-shell`, `.page`, `.page-header`, `.section`
- components: `.ui-card`, `.ui-row`, `.ui-button`, `.ui-segment`, `.metric-ring`
- domain: `.health-summary`, `.project-card`, `.course-card`, `.galaxy-stage`
- state: `.is-active`, `.is-stale`, `.is-error`, `[aria-current="page"]`

Do not encode colours directly in dozens of selectors; use variables.

Example:

```css
.ui-card {
  background: var(--surface);
  border: 1px solid var(--separator);
  border-radius: 16px;
  padding: 16px;
}
```

---

## 5. Mobile shell implementation

### Bottom tabs

Add once in `layout.html`, outside page content.

- fixed to bottom on phone/tablet;
- safe-area aware;
- only five destinations;
- selected state comes from `.Page` or equivalent server view model;
- desktop breakpoint may convert it to desktop nav.

Do not render a separate unrelated primary nav on mobile at the same time.

### Main content

Ensure content has bottom padding equal to bottom tabs + safe area.

### Header actions

Context-specific actions belong in page header, not global tab bar.

---

## 6. SVG/icon implementation

Vendor a small fixed set of Lucide SVG icons. Prefer an SVG sprite so markup can render:

```html
<svg class="icon" aria-hidden="true"><use href="/static/icons/sprite.svg#heart-pulse"></use></svg>
```

Provide visible text or `aria-label` for icon-only controls.

Do not fetch icons from a CDN.

---

## 7. Chart implementation

Extend the existing chart module instead of adding a framework.

Required primitives:

- `line`
- `bar`
- `sparkline`
- optional `area-line`

Metric rings should be SVG/CSS, not canvas, because they need crisp text and straightforward accessibility.

Each chart container must have a text summary such as:

`Weight decreased 2.1 kg over the last 30 days.`

---

## 8. Galaxy implementation

The current `galaxy.js` dynamically imports vendored Three.js; preserve that. Do not globally bundle/load Three.js.

Required mobile changes:

1. stage uses full device width and a meaningful viewport height;
2. cap renderer pixel ratio;
3. selected project context moves to bottom sheet/card on narrow widths;
4. gestures must not conflict with page scrolling — consider touch-action rules carefully;
5. selection is one tap; navigation is an explicit action;
6. keep WebGL fallback/list;
7. pause/suspend rendering when document/stage is hidden;
8. reduced-motion mode minimizes camera drift/transitions.

Do not replace the Galaxy with a static illustration; it remains interactive.

---

## 9. Health implementation sequencing

UI must reflect backend availability.

### Stage H0 — visual shell

Can be implemented with current server data only:

- Health navigation shell;
- current cycle card;
- current metrics already in DailyHealthSnapshot;
- honest missing/stale states.

### Stage H1 — Shortcut ingestion complete

Show real Apple Health/MFP metrics and sync state.

### Stage H2 — Nutrition/recipes

Enable meal planner, foods, recipes, next-meal calculation.

### Stage H3 — Hevy provider

Enable Training page, imported workouts/exercise history, provider status.

### Stage H4 — analytics/AI

Enable richer recommendations and assistant actions only after deterministic services exist.

Never populate an unavailable stage with hard-coded fake values in production.

---

## 10. Hevy UI contract

The UI design assumes a future `TrainingProvider` abstraction and normalized local records.

The UI must not bind directly to Hevy JSON field names. Render domain models such as:

```go
type TrainingWorkout struct {
    Provider string
    ProviderID string
    Title string
    StartedAt string
    DurationMinutes int
    ExerciseCount int
    WorkingSetCount int
    VolumeKg *float64
}
```

Exact model belongs to the Hevy integration milestone, not this UI document.

Provider-specific status belongs in Integrations, while user-facing workout history belongs in Training.

---

## 11. Data freshness contract

For Health metrics use three states:

- **current** — normal;
- **stale** — value + last-synced warning;
- **missing** — no-data state, never numeric zero unless zero is an observed value.

Do not calculate `calories left` from missing/stale dietary intake as if it were current.

---

## 12. Accessibility implementation checklist

For every screen:

- heading hierarchy starts at h1;
- all controls keyboard accessible;
- icon buttons have labels;
- lists use list semantics when appropriate;
- selected tabs expose `aria-current` or `aria-selected`;
- sheets/dialogs trap focus and restore focus on close;
- charts include text equivalents;
- form errors are connected with `aria-describedby`;
- minimum 44 px touch targets;
- test at 200% text zoom where possible;
- test reduced motion.

---

## 13. Visual QA matrix

Every implemented screen must be checked at minimum:

- 320×568 narrow phone;
- 390×844 modern iPhone class;
- 430×932 large phone;
- 768×1024 tablet;
- 1440×900 desktop;
- light theme;
- dark theme;
- reduced motion;
- empty state;
- long text/content;
- offline/stale where relevant.

Acceptance:

- no horizontal page overflow;
- no bottom-nav overlap;
- no clipped safe-area content;
- no inaccessible tiny controls;
- no fake data;
- no regression in forms/actions.

---

## 14. Implementation order

Recommended UI-only sequence after hardening allows it:

1. token layer + global typography;
2. mobile/desktop app shell + bottom tabs;
3. Today + task rows;
4. Projects list + project detail;
5. Galaxy mobile adaptation;
6. Courses family;
7. More/Settings/Goals/Reminders/Calendar/Archive/Plan Import;
8. Health shell + current supported summary;
9. Health pages unlocked progressively by backend milestones;
10. final accessibility, dark mode, responsive and PWA polish.

Do not attempt every template in one commit. Each family should be independently reviewable.

---

## 15. Definition of done for a screen

A screen is not done because it looks like the mockup. It is done when:

1. existing functionality remains available;
2. route works without JS for core content;
3. mobile layout matches the written spec;
4. desktop/tablet layout is deliberate;
5. dark/light theme works;
6. keyboard/touch accessibility works;
7. empty/error/loading states exist;
8. tests remain green;
9. no private data caching regression;
10. visual QA matrix passes.


---

