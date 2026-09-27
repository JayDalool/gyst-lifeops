# Stage 3 — Live UI / UX Audit

- **Stage:** 3, live browser audit of production (read-only)
- **Date:** 2026-09-27, about 00:45–01:30 America/Winnipeg (early morning, before the day's first Health sync)
- **Baseline:** production `dab7995`. Prior stages: `9be649a` (1), `05f2f96` (2A), `80b2613` (2B).
- **Authority:** the Stage 2B feature-status table, unless the browser showed otherwise.

## How this audit was run

| Item | Detail |
|---|---|
| Browser | Claude in Chrome, driving the owner's already-authenticated Chrome on Windows (no credentials entered or seen) |
| Target | `https://gyst.zaylolabs.com` (public ingress) |
| Viewports | The browser window stayed at 1920×911 and `resize_window` had no effect, so mobile, tablet and desktop widths were produced with **same-origin iframes of exact CSS width** (390, 430, 768, 1024, 1440). Frame `innerWidth` was verified (e.g. `390x880`), so media queries evaluate exactly as on a device of that width. 1920 was the native window. |
| Not emulated | Touch events, iOS Safari/PWA standalone chrome, safe-area insets, device pixel ratio 3, real iPhone performance. These are **UNKNOWN** until checked on a device. |
| Measurements | A script run inside each frame recorded page height, word count, top-level cards, interactive elements, and targets under 44×44 px. |
| Login screen | Fetched **without cookies** and rendered in a sandboxed frame (no scripts, no submission) |
| Screenshots | **None committed.** Every production screenshot shows real tasks, projects, prayer tracking or health targets. Findings cite the observations instead. |

### Production safety

- Only GET navigation, scrolling, disclosure toggles, Galaxy camera drag and
  selection, the command palette (opened, typed into, closed with Escape and
  **never submitted**), and temporary client-side theme changes inside a frame
  (not persisted).
- **Not clicked:** checkboxes, Skip, Done, Complete/Move project, Delete,
  Enable prayer alerts, Test Athan, any form submit, Hevy connect, sync.
- The two GET routes that can write (Stage 2B C-3) were checked in production
  before visiting. `GET /projects` would update 0 rows (no roadmap-less projects)
  and `GET /galaxy/data` 0 rows (all eligible projects already have positions).
  **Production data writes: none.** The only incidental write is the owner's
  session `last_seen_at` refresh (at most once per 24 h), which any browsing
  causes.
- State-changing interactions are marked **NOT VERIFIED**. A scratch instance
  wasn't used because the browser runs on the owner's desktop, and reaching a
  server-side scratch instance would have needed a new exposed port, which
  Stage 0 ruled out.

### Measured page profile at 390 px (mobile small)

| Screen | Height (px) | Screens tall | Words | Top-level cards | Interactive | Targets < 44 px | Disabled controls |
|---|---|---|---|---|---|---|---|
| Today `/today` | 2,095 | 2.5 | 175 | 10 | 28 | 2 | 0 |
| Checklist `/checklist` | 844 | 1.0 | 34 | 1 | 16 | 2 | 0 |
| All tasks `/tasks` | 1,194 | 1.4 | 111 | 1 | 36 | **25** | 0 |
| Task detail | 1,002 | 1.2 | 73 | 3 | 18 | 10 | 0 |
| Projects `/projects` | 2,799 | 3.3 | 245 | 13 | 35 | 5 | 0 |
| Project detail (active, 18 missions) | **5,939** | **7.0** | 600 | 7 | 139 | **56** | 0 |
| Roadmap `/roadmap` | **13,918** | **16.5** | **1,114** | 15 | 156 | **120** | 0 |
| Galaxy `/galaxy` | 1,291 | 1.5 | 84 | 0 | 19 | 0 (+ planets, see §I) | 0 |
| Health summary | 4,609 | **5.5** | 357 | 10 | 36 | 2 | **5** |
| Nutrition | 1,933 | 2.3 | 152 | 7 | 16 | 2 | 0 |
| Training | 844 | 1.0 | 40 | 2 | 25 | 2 | 0 |
| Health progress | 1,038 | 1.2 | 103 | 3 | 21 | 3 | 0 |
| Health plan | 2,957 | 3.5 | 187 | 7 | 33 | 2 | 0 |
| Courses (empty) | 844 | 1.0 | 39 | 1 | 11 | 2 | 0 |
| Goals | 958 | 1.1 | 81 | 0 | 13 | 1 | 0 |
| Goal detail | 1,455 | 1.7 | 51 | 3 | 14 | 4 | 0 |
| Analytics | 1,532 | 1.8 | 30 | 3 | 8 | 0 | 0 |
| Monthly | 4,974 | 5.9 | 575 | 3 | 74 | **66** | 0 |
| Reminders | 844 | 1.0 | 27 | 2 | 8 | 0 | 0 |
| Archive | 844 | 1.0 | 29 | 1 | 9 | 1 | 0 |
| Menu `/more` | 1,902 | 2.3 | 70 | 0 | 28 | 0 | 0 |
| Settings | 2,257 | 2.7 | 153 | 5 | 24 | 16 | 0 |
| Plan import | 1,535 | 1.8 | 127 | 2 | 15 | 5 | 0 |
| New task form | 1,606 | 1.9 | 53 | 1 | 24 | 13 | 0 |
| New project form | 1,542 | 1.8 | 126 | 1 | 25 | 2 | 0 |

No page had horizontal scrolling at 390 px. No console errors on Today, Galaxy,
Health or project detail. TTFB was 111 ms and DOMContentLoaded 390 ms (desktop,
public ingress).

The pattern is clear: **screens rebuilt on the newer `ui-*` component layer
(Today, Health subpages, Menu, Galaxy, Courses) meet 44 px targets. Screens
still on legacy `styles.css` (All tasks, project detail, Roadmap, Monthly,
Settings, forms) account for almost all small targets.** Legacy `styles.css`
alone has 153 `border-radius` declarations using about 12 distinct values
(6–20 px plus pills).

---

## A. Executive UX assessment

GYST today feels like **a well-built personal web app with a strong new visual
layer applied unevenly over an older, denser one.** The new layer (shell, tab
bar, Today, Health subpages, Menu, Galaxy) is calm, readable and close to the
Apple-inspired direction: 44 px targets, sentence case, restrained colour, good
dark mode, honest empty states. The older layer (Roadmap, project detail, task
lists, Monthly, Settings, forms) is long, text-heavy, full of inline actions and
duplicated summaries. It reads as an admin tool.

**Three themes dominate:**

1. **Everything is shown at once.** Pages scroll 2.5–16.5 screens on a phone.
   Completed items, destructive actions, explanations and forms are all expanded
   by default. Progressive disclosure exists (`<details>`), but it's used for
   minor things ("Planned tools", "Attach external work") rather than for the
   big ones (finished projects, completed missions, the roadmap's second listing).
2. **The same fact is repeated.** The first task appears 4× on Today. A project's
   progress appears 3× on a card and 3× on its detail page. Roadmap lists every
   project twice. Nutrition appears on the Health summary twice and again on its
   own page.
3. **Placeholders and stale copy blur what's real.** Health has 5 disabled
   controls, two "not built yet" notices, and one false statement (the Shortcut
   "is not built yet"; it is, and has synced). The Workout card always says
   Hevy isn't connected.

**Not broken:** navigation is coherent on phone (five tabs) and desktop (sidebar).
Pages load fast. Nothing errors. Missing ≠ zero is presented honestly everywhere
in Health. Galaxy works, is smooth on desktop, and is visually distinctive.

**No P0 issues.** 7 P1, 14 P2 and several P3 (§V).

---

## B. Visual system assessment

| Aspect | Observation | Assessment |
|---|---|---|
| Typography | System font stack. On Today: 12–28 px, mostly 12/13/14/18 px. Sentence-case headings in the new layer. **ALL-CAPS tracked labels** in Health metrics ("OBSERVED BURN SO FAR", "WATCH STEPS"). | Good base. The caps labels are the one departure from the calm sentence-case voice. |
| Colour | Near-white background, white surfaces, one blue accent, green progress, a navy Galaxy surface. Dark mode is coherent (verified on Today, Health, project detail). | Strong. Close to the intended direction. |
| Radius | New layer uses consistent tokens. Legacy CSS uses about 12 values (6, 7, 8, 9, 10, 12, 14, 16, 18, 20, pill). | Visible inconsistency between old and new screens (P2) |
| Cards | Today has 10 top-level cards at 390 px; Projects 13; Roadmap 15. Cards nest cards (Health "Today's energy" has two inner tiles). | Card overload on Today, Projects and Health summary (P2) |
| Icons | Lucide subset. Recognisable tab icons. **Every project uses the same folder-with-heart icon**; Menu reuses one folder icon for Projects, Galaxy and Roadmaps, and one book for Courses and Calendar. | Icons don't carry meaning beyond the tab bar (P2) |
| Buttons | Mix of filled primary (Add task), outline secondary, text links ("Open checklist", "View health ›"), inline text actions (Done/Skip/Edit). Completion is a checkbox on Today but a "Done" text button on All tasks. | Inconsistent action vocabulary (P2) |
| Kickers | Small grey kickers above titles ("GYST · Health", "Learning deck", "Signals", "Long-term", "Galaxy map" on Roadmaps) | Adds a line of text per page. "Galaxy map" on Roadmaps is wrong. |
| Login | Dark, cyan button, standalone styling | Visually disconnected from the light, blue app (P3) |
| Vocabulary | Project / quest / mission / trip / stop / destination / world / planet. "No external work attached to this **trip**." "4 of 6 **destinations** complete." "Trips & stops." | Conceptual noise from earlier "space" versions (P2) |

---

## C. Navigation assessment

**Structure (verified):** phone and tablet below 1024 px use a bottom tab bar
(Today · Projects · Health · Courses · Menu), 44 px targets, labelled icons.
At 1024 px and above there's a left sidebar with the same five items, plus a top
bar with the GYST wordmark and a search icon. Menu (`/more`) has 19 rows in 5 groups.

| Question | Finding |
|---|---|
| Do I know where I am? | Mostly. The active tab is highlighted, and sub-pages under Health keep Health active. Goals, Analytics, Monthly, Reminders, Archive, Tasks and Settings all highlight **Menu**, so "where am I" becomes "somewhere in Menu". |
| Can I reach important features quickly? | Today, Projects, Health and Courses are one tap. **Galaxy is 2 taps** (Menu → Galaxy, or scroll to the bottom of Projects). **Goals, Reminders and All tasks are 2 taps** via Menu. |
| Are destinations duplicated? | Yes. Menu repeats Today, Projects, Courses and all 5 Health sections, which are already in the tab bar or Health's own tabs. "Daily checklist", "All tasks" and Today overlap. |
| Is Menu overloaded? | 19 rows, 2.3 screens on a phone. It's a sitemap, not a menu. |
| Unnecessary backtracking? | Health subpages have their own tab row, which is good. Project detail → roadmap → project list is a loop with three map metaphors. |
| Desktop vs mobile IA | Coherent: same five destinations. Desktop adds a top bar whose only content is the brand and search. The sidebar is narrow and could hold more (P3). |
| Does Galaxy belong where it is? | It's a peer view of Projects, but it's reached from a dark card at the **bottom** of the Projects list or from Menu. It isn't a mode of Projects in the UI. |
| Secondary tabs | Health's tabs are clear, but **their layout changes per page** (one row on Training; wraps with "Plan" alone on others). Project detail has anchor tabs (Overview/Tasks/Roadmap/Notes). The Health period control (Day/Week/Month/Year) sits right under the section tabs. |
| Icons recognisable? | Tab bar yes. Content icons no (all projects identical). |
| Hidden features | Galaxy (bottom of list), Roadmaps (Menu only), weekly review (about 10,000 px down Roadmap), Import plan (Menu or palette), Ctrl/Cmd+K (no visible hint on desktop apart from the search icon) |

**Cmd/Ctrl+K:** it **already exists** (`app.js:376`). It opens a dialog that
filters **9 static navigation commands** and contains a quick-capture task form.
Typing filters the commands **and** fills the task title at the same time (verified:
typing "gal" showed "Open Galaxy" and put "gal" in the Quick task field). Search
and capture share one input, so it's ambiguous what Enter will do. It's a strong
foundation for a universal launcher; the current behaviour is a hybrid (§U).

---

## D. Mobile experience (390 / 430 px)

- **Feels like:** a responsive website with an app-like shell. The tab bar,
  44 px targets and 16 px inputs (no iOS zoom on focus) are app-like. Long
  scrolling pages of stacked cards, inline forms at the bottom of detail pages,
  and full page reloads on every tap are web-like.
- **Tab bar:** fixed, labelled, 5 items, clear active state. Safe-area CSS
  variables exist (`--gyst-safe-*`); actual iPhone inset behaviour is UNKNOWN.
- **Top bar:** GYST wordmark plus search. It takes about 50 px on every screen
  without carrying page context.
- **390 vs 430:** layouts are identical; 430 just gives slightly less wrapping.
  No breakage.
- **Wrapping hotspots at 390:** Health section tabs (orphan "Plan"); All tasks
  (titles wrap to 3–4 lines because of three text actions per row); Goals (title
  and meta squeezed beside a fixed progress bar); Analytics x-axis labels overlap.
- **Galaxy on mobile:** header and instructions take about 200 px; the scene
  sits in the lower half; the pixel ship overlaps the hint banner; planets are
  about 10 px targets (§I).
- **Page length:** Today 2.5 screens, Health 5.5, project detail 7, Roadmap 16.5.

## E. Tablet experience (768 / 1024 px)

- **768:** bottom tab bar persists; Projects becomes a 2-column card grid with
  **uneven card heights**; Today stays single-column in places and wastes width.
- **1024:** switches to the sidebar **plus** the top bar. Today goes to two columns;
  the right column (prayer card) gets cramped ("Enable prayer alerts and Athan" /
  "Test Athan" wrap to two lines).
- Tablet is acceptable but not designed for: neither a phone layout stretched
  nor a real split view.

## F. Desktop experience (1440 / 1920 px)

- The sidebar plus content column is **well proportioned at 1440**. Health uses a
  sensible 2-column grid. At 1920 the content is centred with large empty
  margins; nothing uses the extra width (no persistent side panel or detail
  pane), which is **fine for calm** but underuses the space for a command center.
- The top bar duplicates branding already in the sidebar. The search icon is
  the only command-center affordance.
- Keyboard: a skip link exists; a focus ring is visible on the brand link;
  Ctrl/Cmd+K works; palette arrow-key navigation exists in code.
- Desktop pages still inherit mobile-first stacks (full-width segmented controls
  on Health, full-width "No data" banner).

---

## G. Today

**Viewports inspected:** 390, 430, 768, 1024, 1920 (and dark at 390).
**Feature status:** IMPLEMENTED (tasks, trackers, prayer, reminders, XP). Day
boundary PARTIAL (UTC, Stage 2A S2-02).

**Order on phone (top → bottom):** greeting and date · Add task · "A focused
today" banner · Today's checklist (2 tasks) · Health snapshot · Focus + Next up
(side-by-side) · Reminders & deadlines · Done/skipped/streak/XP strip · Next
prayer card (countdown, alert buttons) · Prayer tracker (5 tiles) · Athkar (2 tiles).

| Question | Does Today answer it? |
|---|---|
| What matters today? | Partly. The checklist is near the top, but the **same first task is repeated 4×** (checklist, Focus, Next up, Reminders), so the screen has less information than it looks. |
| What should I do next? | "Focus" and "Next up" try to answer this, but both just repeat the first pending task (Focus = first pending task; Stage 2B). |
| What has changed? | No. There's no activity or change feed. |
| Is anything waiting for me? | Partly. Reminders are listed, but the overdue reminder on `/reminders` ("1 overdue") **isn't on Today's reminder card**: the handler discards the overdue list (`handlers.go:456`, `remindersToday, _, err`). |
| How am I doing? | A single-line text strip (done/skipped, streak, level, XP) and "0% complete". XP and level get less visual weight than the motivational banner. |

**What works:** the greeting and date anchor; Add task is a clear primary;
checklist rows are clean; the health snapshot is honest ("No data").
The next-prayer countdown is visually strong. Prayer and Athkar tiles are large,
tappable and clear. Dark mode is excellent.

**Problems:**

| # | Problem | Severity | Product or UI? |
|---|---|---|---|
| T1 | First task repeated 4×; Focus and Next up add no information | P1 | UI (product: no real prioritisation, 2B) |
| T2 | Prayer tracker, the most frequent daily interaction, is at the **bottom** (about 2,000 px down on a phone), below the countdown card and a text strip | P1 | UI |
| T3 | Decorative "A focused today. Small steps build the life you want." banner takes the second-most-prominent slot | P2 | UI |
| T4 | 10 cards for 2 tasks: card overload for a light day; the layout doesn't scale down | P2 | UI |
| T5 | Prayer card mixes a status (countdown) with setup actions (Enable alerts / Test Athan) that are needed once | P2 | UI |
| T6 | Checkbox targets 24×24 px | P3 | UI |
| T7 | Evening check-ins are dated to the next day (S2-02). Invisible in the UI and silently wrong. | P1 | **Product** |

**Direction (not a design):** one "Now" area (next action, next prayer, next
reminder) as compact rows with status; the checklist as a list, not a card;
trackers as a compact always-visible strip; setup actions behind a settings
disclosure; progress and XP as a small status element. **Capacity:** a command
center could add about 2–3 compact status rows (e.g. "2 PRs waiting", "server
OK") without clutter **only if** T1–T5 are resolved first. As it stands, any
addition means another card at the bottom of a 2.5-screen page.

---

## H. Projects

### Projects landing (`/projects`)
**Viewports:** 390, 768, 1440. **Status:** IMPLEMENTED.

- **Current:** title, round "+" FAB-style button, 4-way segmented filter
  (All/Active/Completed/Upcoming), roadmap chips, 13 project cards, "Explore your
  Galaxy" dark card, collapsed "Roadmap route overview".
- **Works:** segmented filter; clear "Active/Done/Upcoming" pills; progress bars; honest 0% states.
- **Problems:**
  - Default filter **All** shows 12 Done projects around 1 active one (P1, UI)
  - Each card encodes progress 3× (bar, %, "x of y missions complete") (P2)
  - Identical folder icons for every project (P2)
  - "coding · main" meta: "main" is internal jargon (P3)
  - A **Done** project shows "Next: …" and 25% with 2 of 8 missions (P2, **product**: status and progress disagree; 2B C-2)
  - Galaxy entry at the very bottom (P2)
- **Direction:** default to Active; compact list rows (icon · name · state · one
  progress indicator); completed collapsed; Galaxy as a view toggle.

### Project detail (`/projects/1`, 18 missions)
**Viewports:** 390, 1440. **Status:** IMPLEMENTED.

- **Current:** breadcrumb, title, summary sentence (repeats status and progress),
  Edit, a progress card (repeats), anchor tabs, an Overview list (repeats a third
  time) with Move upcoming / Complete project, then **all 18 missions expanded**,
  each with checkbox, title, priority · XP · description, **Delete** button and
  an "Attach external work" disclosure; then Roadmap link, Notes, External work,
  and an inline Add mission form.
- **Works:** anchor tabs; mission content is readable; notes and external work are well separated.
- **Problems:**
  - 7 screens tall, 600 words, 56 targets < 44 px (P1)
  - **Delete on every mission, as visible as completion** and in inconsistent
    positions (P1, destructive action prominence)
  - "Attach external work" repeated 18× (P2)
  - Completed missions not collapsed (P2)
  - Status/progress shown 3× (P2)
  - Vocabulary "trip" / "mission" / "Tasks" tab (P3)
- **State-changing actions (complete, delete, move, proof):** NOT VERIFIED.

### Roadmap (`/roadmap`)
**Viewports:** 390, 1440. **Status:** IMPLEMENTED (weekly review and milestones with 0 rows).

- **Current:** kicker "Galaxy map", roadmap selector, rocket-route SVG, "4 of 6
  destinations complete", "100% overall · 0% this week · 0 missed · 0 in focus",
  "Trips & stops" (all 12 projects), Today's execution, Current focus, Missed
  deadlines, collapsed Markdown import, **"Main Goal" repeating all projects
  expanded (about 950 px each, about 7,500 px)**, weekly review at about 10,000 px.
- **Problems:**
  - 16.5 screens, 1,114 words, 120 small targets (P1)
  - Every project listed twice, and again on /projects (P1)
  - "4 of 6 destinations complete" beside "100% overall" (P2, contradictory)
  - The "Galaxy map" kicker conflates two features (P2)
  - Weekly review is effectively unreachable at the bottom (P2)
  - Visible text "Default roadmap created during migration." (P3)
- **Direction:** decide whether Roadmap is a planning view or a timeline; show the
  route plus a concise stop list; move per-project detail to project pages.

### Imports / exports
Plan import page: a 3-step instructional page with a prompt template and a paste
box (IMPLEMENTED; **no AI call**, 2B). The copy explains the workflow honestly.
Export: text links on Roadmap. Import submit: **NOT VERIFIED** (state-changing).

---

## I. Galaxy

**Viewports:** 390, 768, 1920 (native). **Status:** IMPLEMENTED.

| Aspect | Observation |
|---|---|
| Visual quality | Striking and distinctive: deep navy space, starfield and nebula glows, a central star with rings, planets, and the pixel-art ship mascot. It's the most "GYST" screen in the product. |
| Data shown | Only **non-complete top-level projects**: 2 planets in production (Sim Rig Build, LifeOps Tracker V1). With 11 of 13 projects done, **the universe looks almost empty.** Completed work, the bulk of the owner's history, is invisible. |
| Selection | Clicking a planet flies the ship to it and opens a preview panel (category, name, description, progress, missions, target, next mission, Open project). Smooth and delightful on desktop. |
| **Close button** | The panel's "×" is `rgb(231,242,251)` on `rgb(240,240,240)`, about 1.05:1 contrast. It renders as an **empty white square** (P2, UI bug). |
| Drag / orbit | Drag rotates smoothly on desktop. Dragging deselects and closes the preview. |
| Zoom | +/− and Reset view buttons (44 px). Pinch: NOT VERIFIED (no touch emulation). |
| Labels | About 9–10 px grey labels on planets; legend (Active/Upcoming/Locked) about 10 px, low contrast, bottom-right |
| Mobile | Header and instructions about 200 px; the scene fills the rest; **planets are about 10 px tap targets**; the ship overlaps the hint banner at 390; "Browse worlds / All projects / New project" as three text buttons in the header |
| Discoverability | Reached via Menu or the bottom of Projects only. The instructions ("Drag to explore. Pinch or use the zoom controls. Tap a world to preview it.") appear twice (header and bottom banner). |
| Relation to Projects | One-way read-only projection (2B). The preview's "Open project" returns to the standard detail page, a visual jump from space to a white admin-like page. |
| Performance | Desktop: DOMContentLoaded 671 ms, loading about 2 MB of Three.js plus a 1.1 MB PNG (served by the service worker in this session). Smooth at 1920. Phone: NOT VERIFIED. |

**What would make it a signature experience rather than a novelty** (direction, not design):
- It should **mean something at a glance**: show all work, including completed
  worlds (dimmed or archived orbit), so the universe grows with the owner.
- It should be the Projects **mode switch** (list ⇄ galaxy), not a separate page
  found at the bottom.
- Selection → a native-feeling bottom sheet on phones, with the project's next
  action available **in place**.
- Larger hit areas for planets than their visuals; readable labels.
- Keep the ship: it's charming and it's GYST.

---

## J. Health

**Viewports:** 390 (all five pages), 1440 (summary), dark at 390.
**Invariants (verified visually):** missing is shown as "No data", "—" or "Not
logged yet", **never 0**. Watch steps and Phone steps are separate tiles with an
explicit note. Observed burn is only shown with both energies present (shown as
"No data" here). No step-to-calorie values anywhere. **Preserve all of this.**

| Screen | Status | Current | Problems | Sev |
|---|---|---|---|---|
| Summary | IMPLEMENTED (today) + PLACEHOLDER (periods, meals, workouts, trends) | Header · 5 section tabs (wraps) · "Today · date / Week, month and year summaries are not built yet" · Day/Week/Month/Year (**3 disabled**) · "No data for date" banner · Today's energy (2 inner tiles) · Observed energy (3 rows + formula paragraph) · Nutrition (5 rows) · Workout history ("Hevy not connected" static) · Planned tools (disabled Plan meal / Start workout) · Activity (5 tiles + steps paragraph) · Your longer view (Weight / Strength trend text) · Plan card · Habits & consistency (gym week, goals, tasks) | 5.5 screens; about 15 "No data"/"—" markers before the first sync; **placeholder controls occupy prime space**; two "not built yet" notices; **false copy**: "arrives through the GYST Health Sync Shortcut, **which is not built yet**" (`health.html:37`; the Shortcut exists and synced 5 days); Nutrition duplicated (summary card and energy tiles); engineering explanations as body text | **P1** |
| Nutrition | IMPLEMENTED (totals) · PLACEHOLDER (meals) | "Today · date" card with a **7-line explanatory paragraph**, calorie ring ("No data"), macro rows, Meals & planning note | Wall of text before the first number; "Edit targets" links to Plan | P2 |
| Training | PARTIAL (dormant, Hevy not configured) | "Hevy · Not connected / Workout history unavailable" · Connect Hevy (disclosure) · gym manual check-ins (disclosure) | Honest and compact: **the best empty state in Health**. But it contradicts nothing only because Hevy is off; the Summary card is static (2B C34). | P3 |
| Progress | PARTIAL (weight view without a writer) · PLACEHOLDER (trends) | Weight history (30/90/All), "No weight readings in this range", "Weight entry … not available", Nutrition & activity trends (not available), Strength trend | Honest, but it's a page that can never show data today (no writer); header action is "Manage health plan" | P2 (product) |
| Plan | IMPLEMENTED | Opens directly into an **edit form** of the current cycle ("Edit cycle", Goal & dates …), with "Both dates are included. Choose any duration" | Form-first page; the claim "any duration" is wrong (1-day cycles rejected, 2A S2-20); saved cycles list comes after the form | P2 |

**Coherence:** the five pages share a header and tab row, but the **tab row
geometry changes per page**, the header action differs on every page (New health
goal / Edit targets / none / Manage health plan / New cycle), and the ALL-CAPS
metric labels are used only on the summary. **Provenance** is shown only as
update time on Nutrition; the summary doesn't say where each metric comes from
(Watch, phone, MFP via Apple Health).

**Direction:** a Health summary made of **metric rows (icon · name · value ·
source/staleness)** with details on tap; placeholders removed from the primary
surface or reduced to one "Coming later" disclosure; formula and explanation text
behind an info affordance; one consistent section navigation.

---

## K. Courses

**Viewports:** 390, 1440. **Status:** IMPLEMENTED but **dormant (0 courses)**.

- **Current:** kicker "Learning deck", title, "Self-paced courses, bootcamps,
  and modules — tracked your way.", Add course, "Calendar export: get a private
  feed URL in Settings.", empty card "No courses yet. Add your first course —
  Udemy, a bootcamp, a YouTube series, anything self-paced."
- **Works:** honest and short.
- **Problems:** occupies **one of the five primary tabs while empty** (P2,
  product/IA); calendar-export hint shown before any course exists (P3); the
  learning-block repeat option (PLACEHOLDER, 2B) couldn't be seen without data;
  course detail NOT VERIFIED (no data, and creating one is a write).
- **Direction:** decide whether Courses earns a primary tab (see §C, §U).

## L. Goals / Analytics

| Screen | Status | Observations | Sev |
|---|---|---|---|
| Goals | IMPLEMENTED (edit missing: PARTIAL) | 4 goal cards: title, meta (type · category · x of y unit · by date), progress bar. At 390 the title and meta wrap into a narrow column beside a fixed-width bar. Goal progress is manual (e.g. "18 of 120 sessions") while workout data would live elsewhere. **No edit** affordance on detail (2B). The Menu tab is highlighted. | P2 |
| Goal detail | IMPLEMENTED | History chart, entries, add-progress form (not submitted) | P3 |
| Analytics | IMPLEMENTED | "Completion trends … **All charts are static.**" (the page describes its own limitation). 30-day line with one spike to 100 then zeros; x-axis labels overlap at 390; project progress bars | P2 |
| Monthly | IMPLEMENTED | 5.9 screens, 66 small targets; an all-zero chart; long month queue | P2 |
| Reminders | IMPLEMENTED (read-only) | Clean: Due today / Overdue lists. Good compact pattern. | — |
| Archive | IMPLEMENTED | Short list | — |

## M. More / Settings

- **Menu:** 19 rows in 5 groups (Daily planning, Projects & learning, Health,
  Review & tools, Preferences), each an icon plus label (plus some subtitles).
  Clean row pattern (**preserve**), but it duplicates tab destinations and Health
  sections, reuses icons, and is 2.3 screens long (P2).
- **Settings:** opens with **Prayer times**: raw latitude, longitude and timezone
  text fields (grey, prefilled with defaults), calculation method, Asr method,
  per-prayer minute adjustments; then calendar feed, appearance, motion.
  16 targets < 44 px. Technical inputs lead the page (P2). No backup, status,
  integration-connection or account section exists (backup is CLI-only, 2A).
  Changing any setting: NOT VERIFIED.

---

## N. Empty / error / placeholder states

| State | Where | Honest? | Notes |
|---|---|---|---|
| No Health data yet today | Summary, Nutrition, Today snapshot | Mostly | "No data", "—", "Not logged yet". **False copy** about the Shortcut not being built (Summary). |
| Hevy not configured | Training | **Yes** | "Hevy · Not connected" plus how to connect. Good pattern. |
| Hevy card on Summary | Summary | **No** (static) | Always "not connected" regardless of state (2B C34) |
| Disabled periods | Summary | Partly | Disabled buttons look like real segmented controls (grey text only), and a separate sentence repeats that they aren't built |
| Plan meal / Start workout | Summary "Planned tools" | Partly | Hidden in a disclosure, good. They're still disabled buttons that look actionable when opened. |
| Weight | Progress | Yes | But the page can't ever fill (no writer) |
| Courses | Courses | Yes | Short and clear |
| Galaxy with few active projects | Galaxy | Misleading by omission | Two planets; no hint that completed work is hidden |
| Analytics with no data | Analytics, Monthly | Weak | All-zero charts rendered in full; "All charts are static" |
| Loading | Galaxy | Not observed | Loads fast on desktop; phone NOT VERIFIED |
| Errors | — | NOT VERIFIED | Stage 2A: raw `err.Error()` 500s. Reproducing one would need a write (e.g. proof-required completion), so not done. |
| Offline | Service worker | NOT VERIFIED | The SW provides an offline page (code) |

---

## O. Accessibility

| Check | Finding | Sev |
|---|---|---|
| Contrast | Galaxy close "×" about 1.05:1 (invisible). Galaxy labels and legend low-contrast grey on navy. Grey meta text (12–13 px) is light but readable. | P2 |
| Text sizing | 12 px used widely for meta; ALL-CAPS 11–12 px labels on Health | P3 |
| Icon-only buttons | Tab bar icons are labelled. Search button is labelled "Search and quick capture" with `aria-keyshortcuts="Control+K Meta+K"` (`ui_shell.html:6`); the shortcut has no visible hint. Checkboxes labelled "Complete: {task}" (seen in measurements). | Good |
| Focus | A visible focus ring on links (verified on the brand link); skip link present | Good |
| Keyboard | Ctrl/Cmd+K, arrow navigation, Escape in the palette. Galaxy has a keyboard/list fallback (code). | Good |
| Disabled controls | 5 disabled controls on Health summary; the visual difference is grey text only | P2 |
| Target sizes | New layer ≥ 44 px. Legacy screens have many sub-44 px targets (table above). Galaxy planets about 10 px. | P1 (project detail, roadmap), P2 elsewhere |
| Motion | Reduced motion honoured (Galaxy, settings toggle). The ship animation is decorative. | Good |
| Semantics | Headings are structured (H1 per page, H2 per section); section tabs have `aria-label`s | Good |

Not a WCAG certification. Screen-reader output wasn't tested.

---

## P. Performance perception

- **Fast** on desktop via the public ingress: TTFB about 110 ms, DOMContentLoaded
  about 390–700 ms, no console errors, no visible layout shift during inspection.
- **Every navigation is a full page load** (server-rendered). It feels instant
  on desktop; on a phone over mobile data it will feel like a website rather than
  an app (UNKNOWN; needs a device).
- **Galaxy** loads about 3.1 MB of Galaxy-only assets. Served from the service
  worker in this session; on first load over cellular it will be noticeable
  (2B §F). Rendering smooth at 1920; phone NOT VERIFIED.
- **Long pages** (Roadmap 13,918 px, project detail 5,939 px) are the main
  perceived-performance cost: scrolling, not loading.

---

## Q. Progressive disclosure opportunities

**Too much shown immediately:**

| Where | What could be disclosed |
|---|---|
| Today | Focus / Next up duplicates; alert setup buttons; motivational banner |
| Projects | Completed projects (default filter); the triple progress encoding |
| Project detail | Completed missions; per-mission Delete; per-mission "Attach external work"; the Add mission form (as a sheet) |
| Roadmap | The second full project listing; import Markdown (already collapsed); weekly review |
| Health summary | Placeholder period controls; formula and explanation paragraphs; the steps explanation; the "Planned tools" block |
| Nutrition | The 7-line data-timing paragraph (an info affordance) |
| Settings | Coordinates and minute adjustments (advanced) |
| Galaxy | Duplicated instructions (a first-run hint only) |

**Too much hidden:**

| Where | What is hard to find |
|---|---|
| Galaxy | Only reachable from Menu or the bottom of Projects |
| Weekly review | About 10,000 px down Roadmap |
| Overdue reminders | Not on Today |
| Cmd/Ctrl+K | No visible hint |
| Data provenance | The Health summary doesn't show source or last sync per metric |

---

## R. Things already strong — preserve these

1. **The new component layer** (`ui-*`): 44 px targets, tokens, sentence-case
   headings, calm colour. It's the right foundation.
2. **Dark mode:** coherent and good-looking across screens.
3. **Five-tab phone navigation and the matching desktop sidebar:** simple and consistent.
4. **Honest missing-data presentation** in Health ("No data", "—", "Not logged
   yet"; never 0) and **separate Watch/Phone step tiles**.
5. **The Training empty state** ("Hevy · Not connected" plus how to connect): the model for disconnected integrations.
6. **Galaxy's visual identity and selection flight** (ship → planet → preview):
   the signature GYST moment.
7. **Prayer and Athkar tiles and the next-prayer countdown:** large, clear, one-tap.
8. **Menu's row pattern** (icon · label · chevron) and Reminders' compact lists:
   the right density for secondary content.
9. **Cmd/Ctrl+K with quick capture:** an existing launcher seed.
10. **Speed:** fast server-rendered pages with no client framework and no console errors.

---

## S. Product problems exposed by the UI

These look like UI issues but come from product or backend state (Stage 2B):

| UI symptom | Product cause |
|---|---|
| Focus = Next up = first checklist item | No prioritisation model (Focus = first pending task) |
| Disabled Week/Month/Year, Plan meal, Start workout | PLACEHOLDER features without a backend |
| Progress page can never show weight | No weight writer |
| "Hevy not connected" always on Summary | Static template text vs. runtime state (C34) |
| "Shortcut not built yet" | Stale copy after M3 shipped |
| A Done project at 25% with a Next item | Status is set independently of mission completion; three state mappings (C-2) |
| Galaxy looks empty | Galaxy excludes completed projects by design (`galaxyProjectComplete`) |
| Courses tab empty | Dormant feature occupying primary navigation |
| Goals progress manual beside real data elsewhere | No link between goals and tasks, health or training |
| Evening check-ins attributed to tomorrow | UTC day boundary (S2-02); invisible in the UI |
| Analytics "All charts are static" | No range or aggregation backend |

## T. UI problems independent of product functionality

1. Repetition of the same fact (Today 4×, project progress 3×, Roadmap 2×).
2. Card overload and nested cards.
3. Destructive actions (Delete) with the same prominence as primary actions, per item.
4. Legacy screens below 44 px targets; radius and button inconsistency across layers.
5. Health tab-row geometry changing per page; varying header actions.
6. ALL-CAPS metric labels; engineering explanations as body copy.
7. Galaxy close button invisible; tiny planet targets; duplicated instructions.
8. Vocabulary drift (trip / mission / destination / world / quest).
9. "Galaxy map" kicker on Roadmaps; three map metaphors.
10. Login page's separate visual system.
11. Palette search and quick capture sharing one input.

---

## U. Command-center readiness

| Question | Assessment |
|---|---|
| Can Today absorb command-center status? | **Not yet.** It's already 2.5 screens of cards for two tasks. With T1–T5 resolved (dedupe, compact rows, a tracker strip), about 2–3 compact status rows could fit above the fold without clutter. |
| Is there a place for Development / GitHub / Server / Agents? | Not in the current IA: five tabs are taken, one of them by an empty Courses. Menu is already a 19-item sitemap. **The IA question comes before any widget.** |
| Is there a pattern for integration state? | **Yes.** The Training "Hevy · Not connected" card is the right pattern for connected / disconnected / stale / error, and the Health "last synced / stale" treatment is the right staleness pattern. |
| Launcher | **Yes, the seed exists** (Cmd/Ctrl+K, keyboard navigation, quick capture). It needs separate search vs. capture modes and real destinations (apps, repos, services). |
| Desktop space | 1440–1920 widths leave room for a persistent secondary pane (status or detail) without becoming an admin console, if density stays controlled. |
| Visual system | The new `ui-*` layer is suitable; migrating the legacy screens onto it is the precondition for consistency. |

---

## V. Top UX findings

### Top 10 UX issues

| # | Issue | Severity | Affected screens | Why it matters | General direction |
|---|---|---|---|---|---|
| 1 | **Pages show everything at once**; extreme length on the densest screens | P1 | Roadmap (16.5 screens), project detail (7), Health (5.5), Monthly (5.9) | Phone use becomes scrolling; key actions are buried | Collapse completed and secondary content; move detail behind taps and sheets |
| 2 | **Repeated information** (first task 4× on Today; progress 3× per project; projects 2× on Roadmap) | P1 | Today, Projects, project detail, Roadmap, Health | Lower information per pixel; looks busier than it is | One representation per fact per screen |
| 3 | **Health summary is dominated by placeholders, disabled controls and explanatory text, plus false copy** | P1 | Health summary, Nutrition | Undermines trust in the most sensitive area; "Shortcut not built yet" is wrong | Real metrics first as rows; placeholders out of primary space; explanations behind info |
| 4 | **Destructive and secondary actions on every item** (Delete, Attach external work) | P1 | Project detail, All tasks | Accidental-deletion risk; visual noise | Item actions behind swipe, menu or edit mode |
| 5 | **Today doesn't prioritise**: Focus and Next up repeat the checklist; trackers at the bottom | P1 | Today | The future command center's home screen doesn't yet answer "what next?" | Now/Next area; tracker strip; compact status |
| 6 | **Legacy screens fail touch-target and consistency standards** | P1 | All tasks, project detail, Roadmap, Monthly, Settings, forms | Mixed feel (app vs admin); mis-taps on phone | Move legacy screens onto the `ui-*` component layer |
| 7 | **Wrong-day tracker and task attribution** after local evening | P1 (product) | Today, trackers, Analytics | Silently wrong personal records | Single app clock (2B D1). Surfaced here because the UI can't reveal it. |
| 8 | **Galaxy under-represents the owner's work and is hard to find**; tiny planet targets; invisible close button | P2 | Galaxy, Projects, Menu | The signature feature feels empty and hidden | Include completed worlds; a list⇄galaxy mode on Projects; larger hit areas; fix close contrast |
| 9 | **IA: Menu is a 19-item sitemap; Courses (empty) holds a primary tab; Galaxy, Goals and Roadmaps are 2+ taps away** | P2 | Navigation, Menu | Blocks adding command-center areas | Revisit which areas earn primary slots before adding new ones |
| 10 | **Inconsistent components and vocabulary** (radius set, completion affordances, Health tab geometry, header actions, trip/mission/quest terms, ALL-CAPS labels) | P2 | Across the app | Erodes the calm, native feel | Consolidate on tokens and one vocabulary |

### Top 10 things to preserve

| # | Preserve | Why |
|---|---|---|
| 1 | The `ui-*` component layer and tokens | Already meets the design direction; the base for everything |
| 2 | Five-tab phone navigation / sidebar on desktop | Simple and coherent |
| 3 | Dark mode | Polished across screens |
| 4 | Missing ≠ zero presentation and separate Watch/Phone steps | Correctness made visible |
| 5 | Training's "Not connected" empty-state pattern | Template for all integrations |
| 6 | Galaxy's visuals, ship flight and preview panel | Signature identity |
| 7 | Prayer/Athkar tiles and the next-prayer countdown | Best one-tap interaction in the app |
| 8 | Menu and Reminders row patterns | The right density for secondary content |
| 9 | Cmd/Ctrl+K palette with quick capture | Seed of the universal launcher |
| 10 | Speed and simplicity of the server-rendered stack | Fast, reliable, no console errors |

---

### Interactions NOT VERIFIED (state-changing or device-only)

Completing, skipping or reopening tasks and missions · project status changes ·
delete actions · proof submission · Add task / Add mission / New project / New
goal submits · goal progress entry · course and block creation · plan and
roadmap import · settings changes (prayer, calendar token, theme persistence) ·
prayer alerts, Athan and Web Push · Hevy connect and sync · Health sync ·
real-iPhone PWA (safe areas, standalone mode, pinch, performance) · error
responses · offline page.
