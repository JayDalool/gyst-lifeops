# Stage 7A — GYST vNext Design System

- **Stage:** 7, design source of truth (with `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `prototypes/`)
- **Date:** 2026-09-27
- **Binding inputs:** `05A`, `08A`, `10A` (decisions), `03` (live UI evidence), `04`/`05` (vision, IA)
- **Evolves, doesn't replace:** the production `ui-*` layer (`web/static/css/ui-tokens.css`,
  `ui-components.css`), which Stage 3 rated the strongest part of today's UI. Token
  values below are that layer's values tightened into a smaller, complete scale.
- **Reference implementation:** `docs/audit/vnext/prototypes/assets/proto.css` and `proto.js`
  (non-production, isolated, fictional data). Where this document and the prototype
  disagree, **this document wins**.
- **Private preview:** https://claude.ai/artifact/RkJTB44V82FpVekoWfWdA8 (owner-only)

---

## A. Visual identity

GYST is a calm, native-feeling personal command center: cool neutral surfaces,
grouped lists instead of card walls, one confident blue for action, semantic
colour only for state, and a single immersive exception, **Galaxy**, where deep
space, glowing worlds and the pixel **ship** carry the product's personality. Small
identity accents repeat quietly elsewhere: the violet orbit dot beside the GYST
wordmark, the violet XP chip, world-orbs as project icons. The result should feel
at home on an iPhone and on a large desktop without looking like an admin console.

## B. Design principles

1. **Glance, then open.** The default view is icon · name · state/value · one
   signal. Everything else is behind tap, sheet, inspector, disclosure or launcher.
2. **One fact once.** No surface repeats a task, status or progress value (Stage 3 finding).
3. **Lists before cards.** Grouped inset lists are the default container. A card
   (a surface with its own padding) is reserved for the single most important
   block on a screen (Now, System summary).
4. **Exceptions are loud, normal is quiet.** Healthy states are muted text with a
   small green dot. Warnings and failures get tinted chips and, if actionable, Attention.
5. **Honest data.** Missing = "—" plus a reason on request. Recorded zero = "0" plus
   "Recorded zero" where it matters. Stale = amber chip. Never fake values.
6. **Action vocabulary is consistent.** A circle check completes, a chevron opens,
   a filled button commits, a tinted button suggests, destructive lives behind a
   menu or confirmation.
7. **Native rhythm.** 44 px targets, bottom tabs on phone, sidebar on desktop,
   sheets for short tasks, full pages for depth, inspector on wide screens.
8. **Personality in one place.** Galaxy and the ship are expressive; the rest is
   restrained. Motion is brief and purposeful.
9. **Same concepts everywhere.** Phone and desktop present the same hierarchy;
   desktop shows it side by side, never with more concepts.
10. **Build on what exists.** Tokens and components extend the production
    `ui-*` layer. Legacy `styles.css` patterns are retired as screens migrate.

## C. Typography

System stack only (no web fonts): `-apple-system, BlinkMacSystemFont, "SF Pro
Text"/"SF Pro Display", "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`.
Numbers that align use `font-variant-numeric: tabular-nums`.

| Token | Size / line | Weight | Use |
|---|---|---|---|
| `--t-display` | 34 / 40 | 700 | Rare: one hero metric (Calories on Nutrition) |
| `--t-title` | 28 / 32 (30 on desktop) | 700, −0.025em | Page title (one per screen) |
| `--t-section` | 20 / 25 | 650, −0.015em | Now "what" line, sheet titles, big section heads |
| `--t-body` | 16 / 23 | 400; row titles 550 | Row titles, body, inputs (≥16 prevents iOS zoom) |
| `--t-sub` | 14 / 19 | 400–550 | Row meta, state text, secondary values |
| `--t-foot` | 13 / 18 | 400–600 | Footnotes, grouped-list section labels (uppercase +0.06em) |
| `--t-caption` | 12 / 16 (tab labels 11) | 500–650 | Tags, badges, tab labels, launcher group labels |

Rules:
- Seven sizes only.
- Sentence case everywhere except grouped-list section labels (small caps style).
- No ALL-CAPS metric labels (retires the Stage 3 Health caps).
- Titles `text-wrap: balance`.
- Maximum two lines for row titles; meta wraps, never truncates meaning.

## D. Spacing

4-point base, seven steps: **4 · 8 · 12 · 16 · 24 · 32 · 48** (`--s1 … --s12`).

| Context | Value |
|---|---|
| Page side gutter | 16 phone · 24 tablet · 32 desktop |
| Between sections | 24 |
| Section label → group | 8 |
| Row padding | 12 vertical × 16 horizontal; min height 56 (compact 48) |
| Icon tile → text | 12 |
| Inline meta separators | 8 with a "·" glyph |
| Sheet padding | 16 phone, 24 tablet/desktop |

## E. Radii

| Token | Value | Use |
|---|---|---|
| `--r-sm` | 8 | Icon tiles (32), small buttons, tags (6 for tags) |
| `--r-md` | 12 | Buttons, inputs, segmented track, search |
| `--r-lg` | 16 | Grouped lists, Now block, metric grid, banners |
| `--r-xl` | 24 | Sheets, dialogs, launcher, Galaxy space |
| `--r-pill` | 999 | Chips, filters, badges, status strip |

This replaces the roughly 12 radius values found in legacy CSS (Stage 3).

## F. Surfaces

| Token | Light | Dark | Role |
|---|---|---|---|
| `--canvas` | #f4f6fa | #0b111c | App background |
| `--surface` | #ffffff | #141c2b | Grouped lists, Now, sidebar |
| `--surface-2` | #eef1f6 | #1b2537 | Inset controls, segmented track, search, idle tiles |
| `--surface-3` | #e4e8ef | #243047 | Progress tracks, skeletons |
| `--elevated` | #ffffff | #1a2334 | Sheets, dialogs, popovers, launcher |
| `--space` / `--space-2` | #070b18 / #0f1630 | same | Galaxy (always dark) |

Lists sit on the canvas **without borders**. Contrast between canvas and surface
separates them. Separators are used *inside* groups only.

## G. Elevation

| Level | Token | Use |
|---|---|---|
| 0 | none | Rows, inline content |
| 1 | `--e1` (1 px hairline shadow light; none dark) | Grouped lists, Now, chips in the status strip |
| 2 | `--e2` | Popovers, context menus, toasts |
| 3 | `--e3` | Sheets, dialogs, launcher |
| Blur | `backdrop-filter` saturate 1.6, blur 14–16 | Sticky top bar, sub-nav, tab bar only |

## H. Colours

| Token | Light | Dark | Use |
|---|---|---|---|
| `--ink` | #0e1726 | #eef2f8 | Primary text |
| `--ink-2` | #566174 | #a9b3c4 | Secondary text, meta, healthy state text |
| `--ink-3` | #8790a0 | #7b8699 | Tertiary text, chevrons, placeholders |
| `--separator` | #e2e6ed | #243049 | Inset separators |
| `--accent` | #2563eb | #6ea8ff | Primary actions, selection, links |
| `--accent-soft` | #e8f0ff | #15284a | Secondary buttons, selected rows, default tiles |
| `--xp` / `--xp-soft` | #6d5dfc / #eeecff | #a99bff / #231f45 | XP, Galaxy entry points, identity accents only |

The blue matches production `--gyst-blue` (#2563eb) for continuity.

## I. Semantic / status colours

| State family | Graphic | Text | Soft bg | Words used |
|---|---|---|---|---|
| ok | #16a34a / #4ade80 | #15803d / #6ee7a0 | #e7f6ec / #10301f | Healthy, Done, Up to date, Current, Valid |
| info / active | accent | #1d4ed8 / #93bdff | accent-soft | Working, Running, In review, Queued, Syncing |
| warn | #d97706 / #fbbf24 | #b45309 / #fcd062 | #fdf1e1 / #33260c | Stale, Degraded, Needs input, Overdue, Blocked, Elevated |
| crit | #dc2626 / #f87171 | #b91c1c / #fca5a5 | #fdeaea / #3a1618 | Failed, Down, Critical, Privileged |
| idle | #8790a0 / #7b8699 | #566174 / #a9b3c4 | surface-2 | Planned, Paused, Disabled, Disconnected, Unknown, Ready |

Rules:
- **State is never colour-only.** Always a word, and a dot or icon.
- **Healthy is quiet** (`.state.ok` uses `--ink-2` text with a green dot).
- Warn and crit use tinted chips. Critical may also use a tinted banner.
- The vocabulary follows `05` §Y. Agent states per this stage: Idle, Queued,
  Working, Reviewing, Needs input, Failed, Done.

## J. Dark mode

- Default: **follow the system**, with a manual override stored per device. The
  current production toggle keeps working (see decision DD-1).
- Every token has a dark value. Components never hard-code colours.
- Dark tones are designed, not inverted: surfaces lift in steps (#0b111c → #141c2b
  → #1b2537), accent lightens to #6ea8ff for contrast, shadows become none/deep.
- Galaxy is **always** dark and immersive in both modes.
- Implementation: tokens under `@media (prefers-color-scheme: dark)` guarded by the
  override attribute, plus the explicit override selector (the production
  `data-gyst-theme` convention maps onto this).

## K. Icons

- Line icons, 1.8 stroke, round caps (production Lucide subset, extended as needed).
- Sizes: **16** inline/meta/chips · **20** rows, buttons, top bar · **24** tab bar, sidebar primary.
- **Icon tile:** 32 (rows) or 40 (heroes), `--r-sm`/`--r-md`, tinted by meaning
  (accent default, idle for neutral lists, semantic only when the tile *is* the state).
- **World orb:** a project's identity in lists (radial sphere in the project
  colour; dimmed and desaturated when done). It links lists to Galaxy.
- **App tile:** brand-colour tile with 2–3 letter monogram (no third-party logos bundled).
- Every icon-only control has an accessible name.

## L. Motion

| Token | Duration | Easing | Use |
|---|---|---|---|
| `--fast` | 150 ms | `cubic-bezier(.2,.8,.2,1)` | Press scale (0.97), hover, check fill, chevron rotate |
| `--base` | 240 ms | same | Sheets, dialogs, inspector swap, toast |
| Galaxy flight | 900 ms | same | Ship moves to the selected world |
| Pulse | 1.6 s loop | ease | "Working/Running" dot only |

Reduced motion: all transitions and animations collapse to about 1 ms, pulses stop,
the ship jumps, sheets appear without sliding. No motion carries meaning on its own.

## M. Touch targets

- **Minimum 44 × 44 px hit area** for every interactive element on touch devices.
- Smaller visuals (34 px small buttons, filter pills, segments, the 26 px check
  circle, 24 px planets) extend their hit area with padding or a pseudo-element.
- Galaxy worlds: minimum 56 px hit area regardless of drawn size (Stage 3 found about 10 px).
- Adjacent targets keep ≥ 8 px between hit areas.
- Retires the Stage 3 legacy offenders (checkboxes 24×24, text Done/Skip/Edit, per-row Delete).

## N. Component library

Each entry: **Purpose · Anatomy · Variants · States · Phone · Desktop · A11y · Don't.**
Prototype class names are given for reference.

### Shell & navigation

**App Shell** (`.app/.main/.content`)
- Purpose: page frame.
- Anatomy: sidebar or tab bar + top bar + content column (+ inspector).
- Variants: standard; with inspector (`.has-inspector`); immersive (Galaxy).
- States: n/a.
- Phone: content full width, 16 px gutter, bottom padding clears the tab bar.
- Desktop: sidebar 232 (256 ≥1600), content max 760 (wide pages 1120), inspector 380/440 ≥1280.
- A11y: landmarks `nav`/`main`/`aside`; skip link.
- Don't: nest scroll containers; add chrome bars.

**Mobile Tab Bar** (`.tabbar`)
- Purpose: 5 primary destinations (Today · Projects · Health · Learn · More).
- Anatomy: icon 24 + label 11; badge on Today only.
- States: current (accent), default (ink-3), badge (count of needs_action + critical).
- Phone/tablet <1024: fixed bottom, blur, safe-area padding. Desktop: hidden.
- A11y: `nav aria-label="Primary"`, `aria-current="page"`, the badge has a text label.
- Don't: add tabs, put badges on other tabs, hide labels.

**Desktop Sidebar** (`.sidebar`)
- Purpose: same hierarchy on wide screens.
- Anatomy:
  - brand
  - "Search or run… ⌘K" field
  - the 5 primary items (Projects expands its 3 areas when active)
  - an "Operate" group (System with a status dot, Apps; D6)
  - More at the bottom
- States: current (accent-soft fill), hover, status dot (crit/warn/ok).
- Desktop ≥1024 only.
- A11y: one `nav`; group labels are text, not headings.
- Don't: add an Attention item (C1). Attention is reached from Today, the badge and the launcher.

**Page Header** (`.ph`)
- Purpose: orient.
- Anatomy: optional eyebrow (date/context) + title + ≤1 trailing element (XP chip or one action).
- Variants: default; with eyebrow.
- Phone: large title under the top bar. Desktop: same.
- A11y: the single `h1`.
- Don't: add subtitles/descriptions (Stage 3 text density), or more than one action.

**Compact Header** (`.topbar`)
- Purpose: persistent controls.
- Anatomy: leading (back) · centre title (fades in after scrolling past the header) · trailing (≤2 icon buttons).
- States: resting (transparent), scrolled (blur + hairline).
- Phone: sticky. Desktop: sticky, title left-aligned in ink-2.
- A11y: icon buttons named.
- Don't: show the title while the large title is visible.

**Secondary navigation** (`.subnav` + `.seg`)
- Purpose: sibling areas within a tab (Projects · Development · Galaxy · Roadmaps; Summary · Nutrition · Training · Progress · Plan).
- Anatomy: sticky segmented control under the header.
- States: selected (surface fill + shadow), default.
- Phone: horizontally scrollable if needed; ≤5 items fit at 390 with 9 px padding.
- Desktop: full width of the content column. Projects areas are also sub-items in the sidebar.
- A11y: `nav` + links with `aria-current`.
- Don't: stack two segmented controls (use filter pills for a third level, as in Development's Tasks/Reviews/Agents/Repos).

### Lists & rows

**Grouped List** (`.group`)
- Purpose: default container.
- Anatomy: surface, `--r-lg`, `--e1`, inset separators aligned to the text start.
- Variants: default; flush separators; `plain` (no surface).
- A11y: `ul/li`, or `section` + heading.
- Don't: put one row per card; give lists borders.

**List Row** (`.row`)
- Anatomy: [leading: tile/orb/check] · copy (title + meta) · trailing (value/chip/chevron).
- Variants: default 56 / compact 48; link, button or static.
- States: hover (3% ink), pressed (6%), selected (accent-soft), disabled (45% opacity + reason text).
- A11y: the whole row is one target. Nested actions are separate buttons with their own names.
- Don't: put more than one primary trailing action or a destructive action on the row.

**Disclosure Row** (`details > summary.row`)
- Purpose: reveal secondary content (Done · 13, Explanation, History).
- States: closed / open (chevron rotates 90°).
- A11y: native `details/summary`.
- Don't: put links inside the summary; nest disclosures more than one level deep.

**Status Row**: row + trailing state (dot + word). Used in System services and
Agents available. Healthy is muted; problems use a chip.

**Metric Row**: row + trailing `.value` (tabular). Used for Energy, Weight.
Missing = "—" in ink-3.

**Attention Row** (`.attn .row`)
- Anatomy: severity tile (icon in semantic tint) · title · meta (source · age · context) · one action or overflow (⋯).
- Variants: critical · needs_action · warning (info appears only on the page).
- Actions: primary action (Review/Answer/Do today) or ⋯ → snooze 1 h / tomorrow, dismiss (where allowed).
- A11y: the overflow is named "Options for {title}".
- Don't: show descriptions or more than one action; treat it as a task (no check circle).

**Project Row**
- Anatomy: world orb (project colour) · name · one signal (next task, reviews waiting, blocked reason) · progress ring (28).
- Variants: active; blocked (warn state in meta); done (compact, dimmed orb, in a collapsed group).
- Don't: show the triple progress encoding (bar + % + "x of y") (Stage 3).

**Task Row**
- Anatomy: check circle (44 hit) · title · meta (project/course · time · type icon).
- States: to do / done (strike + ink-3, moved into "Done · n").
- Don't: use text "Done/Skip/Edit" buttons. Skip and edit live in a swipe or long-press menu.

**Development Task Row**
- Anatomy: status tile · title · meta (ID tag · repo · agent/review signal · priority tag if P0/P1 · risk tag if Elevated/Privileged) · state chip.
- Chip states: Ready / Queued / Working / Review / Approve / Blocked.
- Don't: show acceptance criteria, branch or PR on the list.

**Agent Row**: tile · agent name · task tag + short title (or "Reviewing PR #n")
· elapsed · state chip (pulsing dot for Working/Reviewing).

**Service Row**: tile · service name · (tailnet only) version/restarts meta · state.

**Automation Row**: tile · name · next run / last run · state. Failed uses a
chip and opens the detail sheet.

**App Row**: app monogram tile · name · optional "Tailnet" meta · health dot ·
external-link glyph. Opens in a new tab.

**Course/Learning Row**: tile · course/book · kind + progress ("4 of 20") · ring.
Session rows reuse the Task Row.

### Data display

**Health Metric** (`.metric` in `.metrics`)
- Anatomy: label with icon · value + unit · one sub-line (provenance or composition, e.g. "1,610 resting + 230 active").
- Variants: with bar; missing ("—" + "Not received"); recorded zero ("0" + "Recorded zero").
- Phone: 2 columns. Tablet: 3. Desktop: 4.
- A11y: a list of items; the value is read with its unit.
- Don't: use more than 4 per grid or ALL-CAPS labels.

**Progress Indicator**
- Bar (6 px, pill), ring (28/34/64), and a **missing** variant (dashed track).
- Colour: accent by default, ok when on track, warn when behind.
- A11y: `aria-label` "62% complete" or a visible number nearby.
- Don't: combine a bar, % and "x of y" for the same value.

**Status Chip** (`.chip`): dot + word, 24 high, pill. Variants ok/warn/crit/info/xp/idle.
`.working` pulses. Don't use chips for healthy states in dense lists (use muted `.state`).

**Badge**: count pill (red for attention, soft grey for neutral counts). Only on the
Today tab/sidebar item and the Attention entry points.

**Icon Container**: see §K (tile, orb, app tile).

### Controls

**Segmented Control**: 2–5 options, equal importance, mutually exclusive views.

**Filter Control** (`.filters .filter`)
- Purpose: pills that narrow a list or map, or act as a third-level view switch.
- States: pressed (ink fill) / default (surface + separator border).
- Behaviour: horizontal scroll.
- Don't: use for navigation between top-level areas.

**Icon Button**: 44 round, accent glyph. The filled variant is for floating controls (Galaxy, Learn play).

**Primary Button** (`.btn`): accent fill, 44 high, one per view.

**Secondary Button** (`.btn.secondary`): accent-soft. `.small` is 34 high with a 44 hit area.

**Destructive Button** (`.btn.destructive`): crit-soft background, crit text. Only
inside confirmations, dialogs, sheets or detail footers, **never per list row**.

**Text Input** (`.input`): 44 high, 16 px text, 12 radius. Focus is a 2 px accent
outline. Error: crit border + message below.

**Search** (`.search`): inset field. On phone it opens the Launcher sheet rather than filtering in place.

### Overlays

**Command Palette / Launcher**: see `13` §Launcher. Three modes (Go/Do/Capture),
grouped results, keyboard hints row.

**Quick Capture**: sheet with one text field plus parsed chips (date, project,
priority) and an "Add task" button. Opened from + on Today, Capture mode, or the tab
re-tap long-press (later).

**Sheet**
- Purpose: short, focused tasks and previews (capture, re-auth, review summary, world preview, item options).
- Phone: bottom sheet, grab handle, max 88 vh, swipe or ✕ to close.
- Tablet/desktop: centred floating sheet (max 560).
- A11y: `role="dialog"` + `aria-modal`, focus trapped and returned, Escape closes.
- Don't: stack sheets; put multi-step flows in one.

**Popover**: anchored mini-surface for ≤6 options on desktop (context menus, filters). Becomes a sheet on phone.

**Dialog** (`role="alertdialog"`): confirmation of a consequential action (Stop run,
Run backup). Title states the action, one sentence of consequence, primary + cancel.

**Context Menu**: long-press (touch) / right-click or ⋯ (desktop) → popover or
sheet with actions ordered safe → destructive, destructive last and separated.

**Confirmation**: pattern combining a Dialog or Sheet with, where required,
**re-authentication** (password field, "valid for 15 minutes"; `10A` D-S5) and a
server-side action nonce.

**Toast**: bottom (above the tab bar), 2.6 s, one line, optional "Undo". Confirms
completed actions ("Queued for Codex"). Don't use for errors that need action (use a Banner).

### States

**Banner**: inline tinted strip with icon + one line + optional trailing chip or
action. Variants info/warn/crit/idle. Used for stale data, tailnet notices and a
critical condition on a detail screen.

**Empty State**: tile + one short title + one sentence + one action
("Nothing to study yet · Add course").

**Error State**: banner (crit) at the top of the affected section: what failed,
last good data kept, Retry. The page stays usable.

**Offline State**: idle banner "Offline · showing saved data" plus the last-updated
time. Actions that need the server are disabled with the reason.

**Stale State**: amber "Stale" chip next to the data's provenance line ("Apple Health · last sync 6:40").

**Loading / Skeleton**: skeleton rows (shimmer, reduced-motion static) for first load
beyond 300 ms. No spinners for page loads.

## O. Component states

| State | Treatment |
|---|---|
| Hover (pointer) | 3% ink overlay on rows; accent-soft on icon buttons |
| Pressed | 6% overlay; buttons scale 0.97 |
| Focus-visible | 2 px accent outline, 2 px offset, on every interactive element |
| Selected | accent-soft row fill; segmented surface fill |
| Current (nav) | accent colour + `aria-current` |
| Disabled | 45% opacity **plus a visible reason** (e.g. "Tailnet only"); never a silent grey button |
| Loading | skeleton or inline pulse dot; buttons keep their width |
| Done | strike + ink-3, moved into a collapsed group |
| Missing | "—" in ink-3 + reason on request |
| Stale | amber chip |
| Error | crit tint + message + recovery |

## P. Accessibility

- **Contrast:** body text ≥ 4.5:1, large text and UI graphics ≥ 3:1 in both themes.
  Token pairs were chosen for this (e.g. warn text #b45309 on #fdf1e1).
  Galaxy labels sit on a 55% navy pill.
- **Names:** every icon-only control has `aria-label`, and state words accompany dots.
- **Keyboard:**
  - logical tab order
  - ⌘/Ctrl-K opens the launcher
  - arrows move selection in the launcher and lists with an inspector
  - Escape closes the top layer
  - focus returns to the invoker
- **Screen readers:** landmarks; one `h1`; section labels as headings; live toasts
  (`role="status"`); dialogs labelled.
- **Targets:** §M.
- **Reduced motion:** §L.
- **Disabled:** always explained in text.
- **Text size:** layouts survive 200% text. Rows grow and nothing clips.

## Q. Anti-patterns (don't build)

- Dashboard walls of equal cards; big-number tiles on screens where numbers aren't the point
- Paragraph descriptions under titles; formula or explanation text on primary surfaces (use info sheets)
- The same fact repeated on a screen (Focus = Next up = first task)
- Per-row Delete or "Attach external work" buttons
- Disabled controls for unbuilt features in primary space (D8)
- ALL-CAPS metric labels; more than 7 type sizes; ad-hoc radii
- Themed functional vocabulary (Mission, Quest, Trip, Destination) (D3)
- Healthy infrastructure shown loudly (D9)
- Hidden-button security (D-J7: server-side only)
- A terminal, n8n editor or diff viewer inside GYST
- Inferred agent reasoning or chain-of-thought text; "Pause" for agents; any agent merge affordance

---

## R. Final design review

### R.A GYST visual identity in one paragraph
See §A. Quiet native surfaces and grouped lists carry daily use; one blue means
"act"; semantic colour only means "state"; Galaxy is the one immersive, dark,
playful place, and its world-orbs and violet accents echo lightly across the app.

### R.B Ten key design rules
1. Glance row = icon · name · state · one signal.
2. One fact, once per screen.
3. Grouped lists by default; one card per screen at most.
4. Normal is quiet; problems are tinted and actionable.
5. 44 px targets everywhere; 56 px for Galaxy worlds.
6. Seven type sizes, seven spacing steps, five radii.
7. Details go to sheets on phone and to the inspector on desktop ≥1280.
8. Destructive actions live in menus, sheets or dialogs, never in list rows.
9. Unbuilt features don't appear; disabled controls always say why.
10. Missing ≠ zero; Watch ≠ phone steps; burned = resting + active.

### R.C Screens most transformed
Today (dedupe + Now + Rhythm up top + Attention), Projects and project detail
(orbs, collapsed Done, no per-row actions), Health Summary (placeholders removed,
metric grid), More (sitemap → 4 groups), Galaxy (completed worlds, big targets,
sheet preview), and the new Development / Reviews / Agents / System / Automations / Apps / Attention.

### R.D Production patterns to preserve
`ui-*` tokens and 44 px components · five-tab bar and desktop sidebar ·
Health missing-data presentation and separate step sources · Training's
"Not connected" pattern · prayer and athkar one-tap check-ins · Galaxy scene,
ship and preview · grouped Menu rows · the ⌘K palette (evolved) · dark mode ·
server-rendered speed.

### R.E Patterns to retire
Legacy `styles.css` cards and radii · Done/Skip/Edit text buttons · per-row Delete
and Attach · Focus + Next up duplication · motivational banner · Health
Week/Month/Year, Plan meal, Start workout and trend placeholders · static
"Hevy not connected" card · ALL-CAPS metric labels · palette search+capture
fusion · 19-row Menu · kickers on every page · Mission/Quest/Trip/Destination copy.

### R.F Open design decisions for Jay
See `12` §"Design decisions for Jay" (DD-1 … DD-4).

### R.G Implementation risks
See `13` §"Implementation risks".

### R.H What must be browser-validated during implementation
See `13` §"Validation checklist".

### R.I Prototype index
See `12` §"Prototype index".
