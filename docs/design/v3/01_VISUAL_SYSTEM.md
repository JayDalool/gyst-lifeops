# 01 — Visual System

## Direction

GYST v3 uses Apple-like clarity rather than an Apple clone: large readable titles, grouped surfaces, restrained separators, strong numeric hierarchy, predictable controls, generous spacing, and 44px minimum touch targets. The primary target is the iPhone PWA, with deliberate tablet and desktop adaptations. The Galaxy remains the one cinematic, space-themed area.

## Theme tokens

Light mode uses a grouped-system background and white surfaces: `--bg #f2f2f7`, `--surface #ffffff`, `--label #111114`, `--label-secondary #636366`, `--label-tertiary #8e8e93`, and `--separator rgba(60,60,67,.18)`. Primary tint is `#0a84ff`. Semantic accents are green `#30b76f`, orange `#ff9f0a`, red `#ff453a`, yellow `#ffd60a`, purple `#8e5af7`, indigo `#5e5ce6`, cyan `#32ade6`, and pink `#ff375f`.

Dark mode uses `--bg #000000`, `--surface #1c1c1e`, `--surface-raised #2c2c2e`, `--label #f5f5f7`, `--label-secondary #aeaeb2`, and stronger dark separators. The default follows `prefers-color-scheme`; the Galaxy stays dark in either theme.

## Typography

Use only the system stack: `-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif`. No proprietary font files are bundled.

Scale: display 34/40 700; page title 28/34 700; section title 20/26 700; card title 17/22 600; body 16/22 400; callout 15/20 500; caption 13/18 400; micro 11/14 600; primary metric 32/36 700; secondary metric 24/30 700. Metrics use tabular numerals. Sentence case is standard.

## Spacing and shape

The base grid is 4px with preferred values 4, 8, 12, 16, 20, 24, 32, and 40. Mobile page padding is 16px; tablet 24px; normal desktop max width 1040px. Standard cards use 16px padding, hero cards 20px. Section gaps are 24–32px. Rows are at least 52px high. Buttons are at least 44px high, normally 48px for the primary action.

Radii: inputs/buttons 10–12px, normal cards 16px, hero cards 20–22px, sheets 24–28px, pills fully rounded. Cards rely on hairline borders and surface contrast; shadows remain very light outside sheets/dialogs. Neon/glow styling belongs only to Galaxy.

## Primary navigation

Mobile uses a fixed safe-area-aware five-tab bar: Today, Projects, Health, Courses, More. The active tab uses the primary tint and stronger label weight. Normal scrolling never hides this bar. Desktop may translate the same information architecture into a top row or left rail without changing labels or route meaning.

Peer views use segmented controls. Health uses `Summary | Nutrition | Training | Progress | Plan`, with `Day | Week | Month | Year` where period switching is meaningful. Projects uses `Galaxy | List | Roadmap | Completed`. Courses uses `In Progress | Library | Completed`.

## Libraries

The production front end remains Go templates + CSS + small vanilla JavaScript modules. No React/Vue/Svelte/Tailwind/Bootstrap build is required for this design.

Icons use vendored Lucide-style SVGs or an SVG sprite, not a CDN and not distributed SF Symbols assets. Charts should extend the existing `web/static/js/charts.js` with line, bar, sparkline, area-line, progress bars and SVG/CSS rings. The current vendored Three.js remains exclusive to Galaxy and loads dynamically there. Motion uses CSS and small vanilla-JS transitions.

## Responsive behavior

Phone 0–599px: single column, bottom tabs, full-width cards, two-up compact stat tiles, bottom sheets for secondary choices, and bottom-anchored Galaxy context. Tablet 600–899px: selected two-column regions may appear. Desktop 900px+: normal content uses roughly 1040px max width; Galaxy may fill the viewport.

Normal pages never require horizontal scrolling. Only deliberate segment/chip scrollers may overflow.

## Accessibility and motion

Target WCAG 2.2 AA. Controls remain keyboard reachable, focus is visible, headings are semantic, form labels are explicit, icon-only actions have accessible labels, colour is not the sole state indicator, charts include text summaries, and Galaxy always keeps a conventional list fallback. Minimum target size is 44×44px.

Timing guidance: press feedback 80–120ms; normal control changes 120–180ms; sheets 220–300ms; Galaxy camera changes 300–600ms. Reduced-motion mode is supported. Continuous ambient animation is limited to Galaxy.

## Health visual semantics

Calories and active energy use orange; protein blue/cyan; carbs orange/yellow; fat purple; fiber green; resting energy indigo/blue; steps cyan/blue; weight trends blue. Food budget, observed energy balance and future projected balance remain separate concepts.

Health values render in three freshness states: current, stale, and missing. Missing data is shown as `No data` or an em dash rather than an invented zero. Stale values remain visible with a last-sync warning.

## Copy style

Labels are short and plain: `460 kcal left`, `Last synced 18 min ago`, `3 of 4 workouts`, `No workout scheduled`, `Plan dinner`. Sci-fi language is concentrated in Galaxy rather than spread through ordinary productivity and health screens.