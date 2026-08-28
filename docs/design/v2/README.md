# LifeOps — Design Prototype v2

Calm, premium, daily-use first. The space identity survives as a *tone* (Mission
Control subtitle, destinations, missions, one quiet map) instead of a costume.

Open `docs/design/v2/index.html`. These files are the original prototype and
design rationale; production lives under `web/`.

**Total weight:** one stylesheet (~15 KB), one script (~8 KB), no fonts fetched,
no libraries, no build step.

---

## What changed from v1

| | v1 | v2 |
|---|---|---|
| Dashboard panels | 10 panels, 2 charts, 6-stat command bar | 6 sections, no charts, one quiet stats line |
| Background | animated 2-layer starfield + 2 blurred ambient orbs + fixed gradients | solid color |
| Panels | glassmorphism, `backdrop-filter: blur(16px)`, luminous gradient borders | flat surface, 1px hairline |
| Always-running animations | 8+ (starfield ×2, ship drift, flame, trail shimmer, node pulses, chart pulses, orbit spins) | 1 (current-node pulse, paused offscreen) |
| Map | 68px glowing planets, reward chips, labels on every node, dotted+solid double trail, legend | 12px dots, thin single line, one label, tooltip for detail |
| Navigation labels | Galaxy Map / Solar System / Mission Control | Dashboard / Today / Projects / Roadmaps / Courses / Settings |
| Copy | "Current Destination · est. arrival · fuel banked" everywhere | plain language first; themed words only where they help |
| Color | 5 accent hues + per-galaxy themes + gold + glows | 1 accent + 3 semantic colors, no glow |
| Page width | 1220px, dense 2-col grids | 960px, mostly single column, real whitespace |
| Type | uppercase tracked eyebrows on every panel | sentence-case kickers, used sparingly |
| Files | 4 CSS + 1 JS (~2,300 lines) | 1 CSS + 1 JS (~950 lines) |

Kept from v1: the interactive route map (signature feature), the Catmull-Rom curved
path, hover/focus tooltips, keyboard-navigable nodes, the XP/streak/medal system
(presented smaller), the calendar-feed security flow.

## Why this reads less AI-generated

The tells in v1 were: every panel decorated the same way (icon-left, eyebrow,
badge, glow), every metric displayed twice, uppercase labels everywhere, five
colors competing, and every feature given equal visual weight. v2 removes those:

- **One thing is loud per page** (the map on Dashboard/Roadmaps, the list on Today);
  everything else recedes. Uniform loudness is the strongest generated-UI tell.
- **Sentence case, plain words, contractions.** "2 of 5 destinations complete",
  "done 8:12", "anytime" — not "MISSION ACHIEVED · +150 XP".
- **Asymmetric rhythm**: sections vary — a borderless stats line, a full-bleed map
  card, a plain list, a two-column pair. Not a grid of identical glass tiles.
- **Almost no icons.** Status is a 7px dot + a word. Icons appear only where they
  carry meaning (checkboxes, the map marker).
- **Restraint as polish**: hairline borders, one shadow (tooltip only), 120ms
  color-only hovers — the Linear/Stripe move of letting typography and spacing
  do the work.

## Performance budget

Enforced in code, not aspiration:

- **Max one ambient animation per page** — the current map node's 3s pulse ring.
  An `IntersectionObserver` pauses it when the map is offscreen
  (`.map.paused` → `animation-play-state: paused`). Every other page is fully static.
- **No continuous rAF loops.** The only script animation is the marker flight on
  node click: one 450–900ms rAF run that terminates. Idle CPU cost of the page: none.
- **No blur anywhere.** Zero `backdrop-filter`, zero `filter` on the render path.
- **Only cheap properties change**: `transform`, `opacity`, `color`,
  `background-color`, `border-color`. Nothing animates layout, shadows, or filters.
- **SVG per map**: two paths + ≤6 dot nodes + an 18px single-path marker
  (v1: gradients, glows, per-node SVG icons, reward chips).
- **No horizontal overflow at 390px** — verified headlessly (below).
- **Scroll cost**: no fixed-attachment backgrounds, no parallax layers, sticky
  header removed (static header, cheaper on mobile).

## Implementation plan (port to production, screen by screen)

Each step ships independently; the app never breaks in between.

1. **Foundation** — add the v2 tokens to `web/static/css/styles.css` behind a
   temporary design gate; delete the starfield/ambient divs and
   `backdrop-filter` when it is set. This alone fixes most of the perceived lag.
2. **Layout shell** (`layout.html`) — static top nav with the plain labels, 960px
   container, footer. Retire the FAB and the "More" drawer (all pages fit the nav).
3. **Today** — smallest template, pure lists (`today.html` pattern: `.task`,
   `.day-group`, deadlines rail). No JS beyond the checkbox POST that exists today.
4. **Dashboard** — replace the bridge grid with the 6-section column. Port the
   map: server renders `.node` buttons with `data-x/data-y` + tooltip data-attrs;
   `app.js#initMap` replaces `rocketroute.js` (~150 lines instead of 330+).
5. **Projects** — `.project-row` list grouped by roadmap; nested `.subprojects`.
6. **Roadmaps** — sidebar + tall map. Server-render the selected roadmap's nodes
   (the JSON-switching in the prototype is a stand-in for a normal page load with
   `?roadmap=` param; no client dataset needed in production).
7. **Project detail** — phase `<details>` groups map 1:1 to existing checklist
   markup; add the phase strip. Drop the orbit view.
8. **Courses / Settings** — card and form styles port directly; keep existing
   handlers for token regenerate/revoke.
9. **Cleanup** — remove `starmap.js`, unused chart types, and the temporary
   design gate once every page is migrated. Completed in production; the chart
   module remains for pages that still render static charts and tracker controls.

## Verification results

Rendered headlessly (Chromium) at **1440×1000 and 390×844**, all 8 pages:

- 0 console errors / page errors
- 0 px horizontal overflow at 390px
- interaction smoke-checked: tooltip on hover/focus, marker flight on click,
  roadmap sidebar switching, copy button, reduced-motion toggle

## Intentional simplifications

- **Orbit/solar-system view removed.** The phase strip + grouped checklist answers
  "where am I in this project?" faster than orbiting planets. The route map remains
  the one spatial metaphor.
- **Charts removed.** The numbers that mattered (streak, XP, % complete) are now
  text — the trajectory/sparkline charts were decoration. They can return later
  as one static SVG on an Analytics page.
- **Per-roadmap accent colors dropped** — one accent everywhere. Roadmap identity
  comes from names and content, not hue.
- **Medals reduced to a sentence** ("9 medals · latest: Two-Week Burn"). A full
  achievements sheet belongs on its own page, not the dashboard.
- **Galaxy switcher (arrows/dots/swipe) replaced** by a plain sidebar list — more
  scannable, better for keyboards, one less interaction pattern to learn.
- **Reminders/overdue/skipped counters** folded into the deadlines list and one
  summary line, per the "no giant noisy blocks" requirement.
