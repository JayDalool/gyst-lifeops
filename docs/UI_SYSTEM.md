# GYST UI system — UI-5 implemented

## Scope and visual authority

Planning baseline: `feature/health-hardening` at `5c50011`. Jay approved UI-0 in the follow-up to the planning review. UI-0 now supplies
the opt-in foundation; later phases remain proposals. No business logic, schema, deployment or production data changes belong to UI-0. Jay approves UI-0 before it starts;
review the result on iPhone before approving UI-1 and subsequent phases.

Primary references, in complementary roles:

- [Design1.png](Design1.png): component board; colors, typography, spacing,
  controls, rings, cards, states and five-tab navigation.
- [Design 2.png](<Design 2.png>): screen board; composition, density, Galaxy,
  page hierarchy and placement of compact metrics.

Use the supplied written requirements where the boards conflict: five permanent
tabs, existing Go architecture, real values only, and no projected energy until
supported. The component board mentions other frontend tools; those suggestions
are superseded by the explicit Go/templates/plain CSS/vanilla JS constraint.
The boards are references, never backgrounds pasted into production pages.

## Visual translation

The screen board uses white grouped cards on almost-white cool backgrounds,
compact rows, blue actions, softly tinted square icon containers, thin progress
tracks, and restrained dividers. Page titles anchor the top; secondary text stays
small but readable. The component board specifies a clearer type/spacing scale
than the tiny screen examples. Use that scale rather than shrinking text to fit
everything above the fold. Preserve page density through spacing and grouping.

Do not render simulated iPhone status bars, hardware frames, sample names,
workout photography, or example numbers. Galaxy alone retains a deep navy
visualization surface with luminous nodes and readable overlays. Its surrounding
navigation remains consistent with the app.

## Tokens proposed for UI-0

These values transcribe the component board where legible. Supplemental values
are implementation proposals requiring visual review, not sampled pixel claims.

| Token | Value | Use |
|---|---|---|
| `--gyst-bg` | `#f8fafc` | App background |
| `--gyst-surface` | `#ffffff` | Cards and grouped lists |
| `--gyst-text` | `#0f172a` | Navy headings and primary text |
| `--gyst-text-secondary` | `#6b7280` | Supporting text |
| `--gyst-border` | `#e5e7eb` | Decorative separators |
| `--gyst-control-text` | `#626b7a` | Muted control labels on inset backgrounds (contrast refinement) |
| `--gyst-control-border` | `#64748b` | Identifiable input boundaries where needed |
| `--gyst-blue` | `#2563eb` | Primary action and selected navigation |
| `--gyst-green` | `#10b981` | Positive graphics and progress |
| `--gyst-orange` | `#f59e0b` | Warning graphics |
| `--gyst-purple` | `#8b5cf6` | Secondary accent |
| `--gyst-red` | `#ef4444` | Destructive graphics |
| `--gyst-positive-text` | `#047857` | Readable positive labels |
| `--gyst-warning-text` | `#92400e` | Readable warning labels |
| `--gyst-danger-text` | `#b91c1c` | Destructive text/button background |
| `--gyst-blue-soft` | `#eff6ff` | Focus card and icon tile |
| `--gyst-control-bg` | `#f1f5f9` | Segmented track and inset surfaces |
| `--gyst-galaxy-bg` | `#03050b` | Existing Three.js scene background |

Bright green/orange/red are not automatically suitable text backgrounds. Measure
contrast in the actual pairing: 4.5:1 normal text, 3:1 large text and essential
control graphics. Status always includes a word or icon as well as color.

Font: `-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text",
"Segoe UI", sans-serif`. Use installed fonts only. No proprietary font files.

| Role | Size / line height | Weight |
|---|---|---|
| Display / prominent metric | 34 / 40 px | 700 |
| Page title | 28 / 34 px | 600–700 |
| Section title | 20 / 28 px | 600 |
| Body and form inputs | 16 / 24 px | 400 |
| Caption | 14 / 20 px | 400 |
| Micro label / tab caption | 12 / 16 px | 500 |

Use rem equivalents; numeric metrics use tabular numerals. Do not apply mono
fonts to all metrics. Long titles wrap; rows grow with enlarged text.

Spacing tokens: 4, 8, 12, 16, 24, 32 px (8 px rhythm with half steps).
Mobile horizontal gutter: 16 px; card padding: 16 px; card gaps: 12–16 px;
section separation: 24 px. Radii: 12 px small controls, 16 px cards, 20 px
prominent cards, 24 px sheets, 999 px pills. This follows the board's 12/16/20/24
scale rather than the illustrative values in the written brief.

Shadows: card `0 1px 2px rgb(15 23 42 / 5%)`, raised
`0 4px 12px rgb(15 23 42 / 8%)`, floating
`0 8px 24px rgb(15 23 42 / 12%)`. Default cards need only a thin border;
no large shadow on the tab bar. Motion: 150–250 ms for small state transitions;
remove nonessential transitions and animation under reduced motion.

## CSS migration and theme compatibility

Current `styles.css` has several generations of tokens and high-specificity
`:root` overrides, plus Galaxy and completion effects. Do not append another
unbounded global reskin or delete that stylesheet in UI-0.

Create `ui-tokens.css` and `ui-components.css`. Component classes use a `gyst-`
prefix. UI-0 loads the foundation but adopts it only in an opt-in development
component gallery. UI-1 adopts it for the shell and More. Page bodies migrate in
their assigned phases. Keep old CSS selectors and JS data attributes until their
consumers have migrated. Audit selectors before each removal.

Light is the default for the new system. Preserve explicit existing theme choice
and reduced-motion preference. The current theme toggle stores `light` and removes
the key for dark; therefore absence cannot identify an intentional dark choice.
UI-1 introduces explicit `light`/`dark` persistence, retaining `light` and treating
an absent key as the new light default. Keep the appearance control operational.
Provide semantic dark token overrides for opted-in components; keep the Galaxy
canvas dark independently. Avoid a first-paint theme flash with a small guarded
head initializer. No new JS framework or build tooling.

## Template and component contracts

`Handler.render` currently parses only `layout.html`, `rocket_route.html`, and the
selected page. It does not discover partial directories. Extend its explicit
shared file list; existing render test helpers already call that real path. Use named Go template
definitions, escaped text, and typed presentation structs. Do not introduce raw
HTML strings or a universal component that accepts arbitrary markup.

Container-only patterns can be shared semantic CSS (`section.gyst-card`,
`ul.gyst-list`); repeated structured content gets a template partial. This avoids
pretending Go templates provide JSX-style arbitrary child slots.

| Component family | Contract / ownership | First adoption |
|---|---|---|
| AppShell, TopAppBar, BottomTabBar | Page identity, title, optional back link/actions; semantic header/nav/main | UI-1 |
| PageHeader, SectionHeader | Heading level, text, optional real action | UI-0 foundation |
| Card, StatCard, MetricCard, MetricRingCard, ProgressCard, TrendCard | Label, formatted value/unit, optional target, state, summary | UI-0 foundation; data wiring later |
| ListSection, ListRow, NavigationRow | Heading; icon, text, optional metadata, real URL or form action | UI-0 / UI-1 |
| SegmentedControl, SubNavigationTabs | Links for navigation; buttons for local state; selected value | UI-0 |
| PrimaryButton, SecondaryButton, IconButton, FloatingActionButton | Native button or link as appropriate; accessible name; disabled reason | UI-0 |
| SearchField, TextField, SelectField, Toggle, Checkbox, Radio | Label, stable ID/name, help, error, value; preserve submission names | UI-0 |
| Pill, Badge | Text plus semantic tone; never color alone | UI-0 |
| EmptyState, LoadingState, ErrorState | Heading, concise message, optional working action | UI-0 |
| Modal, BottomSheet, ConfirmDialog, Toast | Dialog semantics, focus return, Escape, status announcements | UI-0 styles; UI-1 existing palette adoption |
| MiniLineChart, MiniBarChart, ProgressBar, CalorieRing, MacroProgressBar | Values, bounds, missing flags, units and textual summary | UI-0 contract; UI-5 onward data wiring |
| ProjectCard, ProjectProgressRow, GalaxyDomainNode | Existing project progress/category; domain grouping is derived presentation | UI-3 / UI-4 |
| HealthSummaryCard, NutritionCard, MacroCard, MealCard | Nullable observations and targets; meals require future data | UI-5 / UI-6 |
| WorkoutCard, ActivityOptionRow, TrainingProgressCard, HealthPlanCard | Provider-neutral workout data or empty; real cycle targets | UI-7 / UI-9; activity future |
| CourseCard, LessonRow | Existing course and learning-block state, next block and URLs | UI-10 |
| ReminderRow, NoteCard, CalendarEventRow, AnalyticsMetricCard | Existing data where available; see screen map for gaps | UI-11 / UI-12 |
| IntegrationStatusRow, SettingsRow, ProfileHeader | Actual connection state and sync age; available XP; no invented profile | UI-13 |

UI-0 implements the reusable primitives, contracts and gallery examples. Domain
components are specified now and implemented with their first real consumer;
no empty template stubs for all future screens.

Icons: one locally hosted Lucide SVG subset with its license included, no CDN or
runtime icon library. Use a 24×24 viewBox, consistent rounded 2 px strokes,
`currentColor`; tab glyphs approximately 22 px. Decorative SVGs are hidden from
assistive technology; icon-only buttons have accessible labels. GYST branding
and existing Galaxy artwork are not navigation icon families.

## Navigation and safe areas

Exactly five primary links: Today `/today`, Projects `/projects`, Health
`/health`, Courses `/courses`, More `/more`. Use one nav DOM structure adapted
across breakpoints, labels below icons on mobile, `aria-current="page"` on the
selected destination. Projects owns Galaxy, project details and roadmap routes;
Health owns all Health routes; Courses owns course detail/forms; More owns
secondary tools. `/` is Today; `/dashboard` keeps its redirect.

More is a GET-only grouped destination, not a sixth menu or creation tab.
Its initial rows link to existing Daily checklist (temporarily `/today`), tasks,
reminders, goals, monthly view, analytics, archive, roadmaps, settings and plan
import. Projects provides Galaxy access; command search keeps it discoverable.
New backend-dependent destinations are added when their real empty-state routes
land. Keep Search, quick capture, Add actions and XP available in the top shell.
Do not remove working actions to match screenshot density.

Mobile (<768 px): 56 px top bar, 56 px tab content plus bottom safe inset;
minimum 44×44 px link/button hit area. Use a solid white fallback with optional
subtle translucent surface. Keep active blue and inactive muted gray.
Main bottom padding = tab content + safe inset + 24 px. FABs and toast stacks
sit above that boundary; never overlap the final list row.

Use `viewport-fit=cover`, `env(safe-area-inset-top/bottom/left/right)` and `100dvh`
with `100vh` fallback. Safe insets are applied once per edge. Dialogs must remain
scrollable with the software keyboard open. Do not disable browser page zoom.
Tablet (768–1023 px): top navigation, two columns where useful. Desktop
(≥1024 px): max-width 1200 px shell, 2–3 content columns, narrower forms
(approximately 640 px). Breakpoint values live in one documented responsive
section; native CSS media queries use literals, not unsupported var() conditions.
Legacy breakpoints remain only on unmigrated page bodies until UI-14.

Galaxy UI-1 receives only shell-clearance compatibility adjustments. UI-4 will
frame the existing scene below the header and above the tabs, put selection
information in a safe-area-aware sheet, keep desktop roam and list fallback,
and limit touch gestures to the visualization. Suppress accidental navigation
after drag; do not intercept edge gestures across the whole page. Category/domain
nodes must derive from real project categories; never seed example life domains.
Preserve project positions, moons, selection, progress, URLs and completion rules.

## Data, charts and states

Every data component accepts normal, empty, partial, stale, loading and error
presentation. Server-rendered pages do not show synthetic loading spinners;
loading appears only during a real pending operation. Reserve its dimensions.

| State | Required behavior |
|---|---|
| Empty | “No data synced yet” or specific empty copy; no fake zero, percentage or chart line |
| Partial | Render available metrics; mark missing ones individually; do not hide the whole card |
| Stale | Retain last known values with visible age and stale label |
| Current | Available values with source/time where meaningful |
| Loading | `aria-busy`, stable skeleton area, reduced-motion-safe; no invented value |
| Error | Clear failure message and working retry where supported; never masquerade as empty |

Health keeps `HealthMetricView.Present` and the existing 8-hour snapshot stale
threshold. That timestamp is snapshot-wide, not proof each metric was refreshed;
do not label individual metrics freshly synced without per-metric evidence.
Observed zero displays 0. Unset targets remain “Not set.” Calorie ring fill needs
both consumption and a positive target; clamp only graphic fill, retain actual
over-target text. Neutral ring for missing inputs. Food allowance and observed
energy balance are separate cards/regions. Resting + active is observed burn;
never add workouts or steps. Phone fallback must be labelled and must not merge
with Watch steps. No projected numbers, fake workouts, meal entries, or adherence.

Retain `charts.js` canvas infrastructure for existing analytics; compact charts
use thin strokes, sparse gridlines and 120–180 px plotting height. Every chart
has adjacent textual summary and accessible values/table. Missing samples create
gaps, not zeros. No-data charts have no illustrative trend. SVG/CSS rings and
native progress are preferable for single-value graphics.

Forms keep labels, submitted field names, errors and values after validation.
Controls have visible focus (2 px blue outline with offset), sufficient contrast,
and keyboard support. Local tab widgets need arrow-key handling and associated
panels if using ARIA tab roles; plain navigation links need no faux tab semantics.
Dialogs use native `dialog`, focus containment and restoration. Confirmations
appear for existing consequential actions, not every checkbox. Toast uses a polite
status region; errors stay visible beside the affected field/action.

## Exact proposed UI-0 changes — foundation only

| File | Change |
|---|---|
| `web/static/css/ui-tokens.css` (new) | Tokens, type/spacing scale, theme variants, safe-area dimensions |
| `web/static/css/ui-components.css` (new) | Namespaced primitives, documented responsive section and states |
| `web/templates/ui_components.html` (new) | Named primitives for repeated rows, metrics, navigation controls and states |
| `web/templates/ui_icons.html` (new) | Local Lucide subset as named SVG definitions |
| `web/static/licenses/lucide.txt` (new) | License matching the icon source |
| `internal/handlers/ui_view.go` (new) | Typed presentation-only component inputs; no service arithmetic |
| `internal/handlers/handlers.go` | Add shared templates to render file list only |
| `internal/handlers/render_test.go` | Exercise shared templates through existing real-render helper; preserve cases |
| `internal/handlers/ui_components_test.go` (new) | Meaningful absent/zero, escaping, accessible label and progress-bound cases |
| `web/templates/layout.html` | Load new styles with matching versions; no shell redesign yet |
| `web/static/service-worker.js` | Align foundation asset versions; no policy redesign in this phase |
| `internal/handlers/ui_gallery_test.go` (new) | Opt-in test fixture rendering gallery to a chosen temporary output directory |
| `web/templates/ui_gallery.html` (new) | Clearly labelled development examples, no public route or production inclusion |
| `docs/UI_SYSTEM.md`, `docs/UI_SCREEN_MAP.md`, task ledger and handoff | Record implementation evidence and review outcome |

UI-0 gallery covers buttons/forms, list/card variants, segmentation, rings/bars,
all data states and dialog/sheet visual variants. Sample values are confined to
this explicit development fixture. Compare the rendered gallery with Design1 on
phone width before expanding adoption. No public `/ui` demo route.

## Exact proposed UI-1 changes — shell only

| File | Change |
|---|---|
| `web/templates/layout.html` | Adopt AppShell, GYST title/brand, top bar, skip link, shared nav; retain XP and palette |
| `web/templates/ui_shell.html` (new) | TopAppBar/BottomTabBar definitions with five links |
| `web/templates/more.html` (new) | Grouped real secondary links using shared NavigationRow |
| `internal/handlers/ui_shell.go` (new) | GET-only More view and deterministic primary-section mapping |
| `internal/handlers/handlers.go` | Explicit `/more` dispatch; include shell partial; no business service changes |
| `web/static/css/ui-shell.css` (new) | Shell layout, safe areas, responsive navigation, Galaxy clearance and palette geometry |
| `web/static/css/styles.css` | Retire only shell selectors whose consumers have migrated; retain page/Galaxy effects |
| `web/static/js/theme.js` (new) | Guarded early theme initialization and explicit persisted preference |
| `web/static/js/charts.js` | Delegate existing theme toggle to one owner; keep charts/trackers/toasts |
| `web/static/js/app.js` | Only shell hook integration as needed; preserve quick capture/prayer/progression logic |
| `web/static/manifest.webmanifest` | GYST name, approved light background/theme; retain `/today` start URL |
| `web/static/service-worker.js` | Synchronize versions; static shell cache; avoid eager Three.js downloads and private HTML caching |
| `internal/handlers/navigation_test.go` | Assert exact destinations, selected state and reachability through More |
| `internal/handlers/render_test.go` | Representative page rendering through shared shell partials |
| `internal/handlers/ui_shell_test.go` (new) | More GET/method handling, nested active state, retained action links |
| `internal/handlers/pwa_assets_test.go` (new) | Asset URL consistency and private-route cache policy regression coverage |
| UI docs, task ledger and handoff | Commit, CI and device review evidence |

The service worker currently caches navigations broadly, excludes `/health/plan`
but not `/health`, and preloads Galaxy vendor files even from Today. UI-1 should
cache static assets only, use a neutral offline response for uncached navigation,
and never fall back to another private page. This is a deliberate change to the
offline behavior, preserving confidentiality and avoiding unrelated stale content;
include online/offline and sign-out verification in the shell review.

## Review gates

For each implementation phase run format, vet, build, unit and race checks;
update relevant route/render regression cases without weakening feature assertions.
For UI-1 an old test requiring every secondary URL on every Health page should
be changed to prove reachability through More, not simply deleted.

Visual review: 375/390/430 px portrait, phone landscape, 768 px tablet and
1280 px desktop; 200% text zoom; keyboard focus; reduced motion; long titles;
empty/partial/stale Health; software keyboard; PWA safe areas and offline reload.
Galaxy checks use a temporary database: `GalaxyData()` can persist generated
positions, so a GET against production is not a read-only visual inspection.

Planning verification is source inspection plus both boards. No browser/device
verification is claimed for this document. Approval request: approve these two
phase boundaries and files, starting with UI-0 only.

## UI-0 implementation and review

UI-0 now provides namespaced CSS primitives, typed Go presentation inputs,
shared escaped partials, and a pinned local Lucide 0.468.0 subset with license.
Production loads the CSS and shared definitions but does not opt into
`gyst-scope`. Existing shell, page markup and business behavior remain intact.
Dark overrides use `data-gyst-theme="dark"` inside the new scope; the legacy
appearance setting is unchanged until UI-1.

The standalone development gallery is `web/templates/ui_gallery.html`. It is
not included by the production renderer and has no public app route. Fixture
values live only in `ui_gallery_test.go` and clearly marked gallery examples.
Chart previews illustrate the contract; actual chart data wiring and reusable
trend rendering remain with their planned screen phases. Containers, badges,
buttons, dialogs, sheets and toast use semantic HTML plus reusable CSS. Dialog
preview interaction is confined to the gallery; production palette adoption is UI-1.

Export a self-contained HTML review artifact without starting the app or opening
a database:

```sh
GYST_UI_GALLERY_OUT=/tmp/gyst-ui0-review go test ./internal/handlers -run TestExportUIGallery -count=1
```

The resulting `index.html` embeds CSS and SVGs and can be opened independently.
No CDN, frontend build or runtime gallery route is required. The existing
`render_test.go` already exercises the real shared renderer, so it needed no
helper change. Additional isolation/escaping/presence tests live in
`ui_components_test.go`.

Browser review: Chromium and WebKit at 375, 390, 430, 768 and 1280 px passed
horizontal overflow checks; dialog Escape/focus return, bottom-sheet dismissal,
period selection, dark theme, 200% text and reduced motion passed. Screenshots
were visually inspected. No real iPhone/Home Screen or software-keyboard check
has been performed. Secondary labels on the inset control background use
`--gyst-control-text` because the board's muted text pairing measured 4.41:1.

UI-1 still requires Jay's review of this gallery. Its routes and shell are not
implemented by UI-0. Service-worker changes here only align existing asset
versions and add the foundation CSS; the cache policy change stays in UI-1.

## UI-1 implementation

Jay reviewed the UI-0 gallery and approved proceeding with “i like it, go ahead.”
UI-1 adopts the five-link shell and adds GET-only `/more`. Projects links directly
to Galaxy; More and command search also retain it. The shared shell owns no
business calculations. Existing page bodies keep their own templates/CSS.

Light is the default with a no-JS HTML fallback. `theme.js` runs before styles,
reads `lifeops-theme` defensively and saves explicit light/dark choices. It sets
both the old theme selector and the new component selector; charts listen for a
theme-change event. Storage denial does not stop rendering or interactions.
The legacy theme toggle remains functional. UI-1 does not restyle the separate
login screen or change authentication rules.

The service worker caches only public `/static/` assets. Navigations always use
the network without the HTTP cache; offline navigation returns a neutral 503 page.
Private pages, auth decisions, APIs, HTML responses, no-store responses and
redirected assets are excluded. Activation removes old `lifeops-shell-*` caches,
including previously cached personal pages, while leaving unrelated cache names
alone. Three.js/Galaxy files load only when requested, not during shell install.
Push handlers and `/today` PWA start URL are preserved.

The command palette keeps quick capture, keyboard search and submission fields;
Escape closes it and returns focus to its trigger. Add menu outside-click/Escape
handling uses the existing menu behavior. Safe-area padding separates the tab bar,
content, toasts and Galaxy stage; header and tab borders are included in clearance. A ResizeObserver measures the
actual header/navigation sizes for enlarged text and landscape; CSS provides
fallback dimensions when JavaScript is unavailable.

Small additions to the planned file list: `projects_list.html` adds the direct
Galaxy link; `scripts/check-ui-shell.cjs` behavior-tests cache privacy and theme
persistence using Node built-ins, and the existing CI workflow runs that check.
No package manager, frontend dependencies or build system was added.

Verification evidence is recorded in [UI-1](tasks/UI-1-shell.md). Real-device
Home Screen review remains Jay's next step before UI-2.

## UI-2 implementation

Jay approved UI-1 and requested UI-2. Today now composes UI-0 cards, headers,
progress, metric and status components in a mobile column and bounded desktop
main/sidebar layout. `ui-today.css` is loaded only for Today/checklist bodies;
existing shell tokens, safe areas and 768/1024 breakpoints apply. At widths below
375px compact paired cards stack. No new framework or page JavaScript bundle.

Today previews up to four pending, two completed and two skipped tasks; the dedicated checklist renders all occurrences with identical actions.
`today_components.html` owns the shared checklist markup for Today and
`checklist.html`; completion/skip forms retain their actual occurrence dates
and originating page. `today_view.go` adapts existing Health data, selects an
existing upcoming timed reminder and supplies the checklist return path.
`handlers.go` wires those reads/routes without modifying service calculations.
`charts.js` keeps visible tracker status text synchronized with existing AJAX
state and rollback. The worker cache/version list follows changed assets.

No fabricated progress rings are rendered without a supported denominator.
An entirely absent Health snapshot uses one concise empty state; present
snapshots use compact shared metric cards with explicit missing/freshness states.
The pending focus card does not imply a scheduled deep-work session. Existing
prayer and Athkar controls remain below the mobile overview; on desktop they
share the right column with deadlines and XP.

See [UI-2](tasks/UI-2-today.md) for verification evidence. UI-3 awaits review.

## UI-3 implementation

Jay approved continuation after UI-2. `project_components.html` supplies one
project card/progress pattern for top-level projects, nested subprojects and
detail stops. `projects_view.go` prepares view-only filters and typed progress
inputs; actual project status controls filtering, never an inferred 100% bar.
Both query filters persist in ordinary server-rendered links. All remains the
default; a context parent keeps its true badge if a child matches another state.

`ui-projects.css` applies only to Projects page bodies, including create/edit
forms. It uses shared tokens and 768/1024 breakpoints, bounded desktop columns,
44px controls and wrapping cards/forms. Detail sections use native anchor links
so content and return fragments work without a tab script. Notes and roadmap
empty states lead to existing edit functionality; no milestone database is added.
The existing route overview remains in a disclosure and Galaxy remains a direct
navigation card. No heavy visual assets or new JavaScript dependencies are added.

See [UI-3](tasks/UI-3-projects.md) for verification. UI-4 awaits review.

## UI-4 implementation

Jay authorized Mobile Galaxy while testing UI-3. The existing Three.js scene,
positions, project planets and child moons remain. `galaxy-gestures.js` isolates
canvas pointer capture, movement thresholds, pinch zoom and cancelled gestures.
Desktop mouse roaming/right-click remain; touch uses preview followed by an
explicit Open project link. Camera buttons and Escape provide alternatives.

`ui-galaxy.css` styles the dark scene controls, opaque scrollable mobile selection
panel and shared-token navigator. The panel stays within the scene above the
primary navigation; underlying mobile controls hide while selected. Categories
come from the API and filter the navigator only. Child project links remain
available there. No domain graph, position changes or service changes are added.

Node gesture regression checks run in CI. Browser evidence and limitations live
in [UI-4](tasks/UI-4-galaxy.md). Next phase is UI-5 Health summary after review.

## UI-5 implementation

Jay requested continuation and specifically Health sizing fixes. `ui-health.css`
uses the shared palette/type scale with wrapping navigation and responsive
energy/activity cards. The static ring is removed; native calorie progress
requires the existing HasBar contract. Food allowance and observed energy are
separate cards. The existing model retains all null/zero/source distinctions.
History controls are visibly unavailable; the old period script is no longer
loaded. Gym controls retain their handlers and show text statuses. Supported
plan/goals/tasks remain reachable; planned tools are a compact disclosure.
No service/schema changes. Evidence: [UI-5](tasks/UI-5-health.md).
