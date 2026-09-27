# GYST Galaxy — production baseline (GYST-VN-143)

- **Issue:** #9 (GYST-VN-143) · **Recorded:** 2026-09-27 · **Type:** observation / documentation only
- **Binding context:** [`17A_GALAXY_PRESERVATION.md`](../../audit/vnext/17A_GALAXY_PRESERVATION.md) (this file verifies and corrects its §B)
- **Galaxy code verified:** `master` @ `ca48004` (application tree identical to production `dab7995`); the Galaxy files were last changed in `0b2ea78`.
- **Nothing in Galaxy was changed.** No JS, CSS, template, Three.js, ship, data or position change; no migration.

## How to read the evidence labels

| Label | Meaning |
|---|---|
| **LIVE** | Observed in a real browser running the **production Docker image** (`gyst-lifeops-gyst-lifeops:latest`, whose Galaxy JS, template and CSS are byte-identical to `master`), with **fictional data** on a disposable local instance |
| **LIVE-PROD (Stage 3)** | Read-only observation of the real production site recorded in `03_UI_UX_AUDIT.md` (2026-09-27, early morning), cited, not repeated |
| **CODE** | Derived from reading the code only |
| **NOT VERIFIED** | Not established by this task; the reason is given |

Production itself was **not opened** in this task: loading production Galaxy calls `GET /galaxy/data`, which writes positions for any project without one (§G), and `docs/RULES.md` §3.1 reserves anything touching production data to Jay. The local instances run the same image, so behaviour is identical; only the data differs.

---

## A. Environment

| Item | Value |
|---|---|
| Application | Production image `sha256:af525f02f5f7…` (built 2026-09-24 from `0b2ea78`; only a test file differs from `dab7995`), run as throwaway containers with `GYST_AUTH_DISABLED=1`, bound to `127.0.0.1`, `-seed=false`, empty data directories |
| Data | Fictional only, created by [`seed-fictional-galaxy.sh`](seed-fictional-galaxy.sh) (refuses non-local URLs): datasets with **0, 1, 3, 10 and 30** active worlds; the 30-world set adds paused/upcoming/locked and 4 completed projects (37 projects → 33 worlds) |
| Browsers | (1) **Playwright 1.58** in `mcr.microsoft.com/playwright/python:v1.58.0-noble`, host networking: headless **Chromium** (WebGL through SwiftShader = software rendering) and **WebKit** with the Playwright *iPhone 13* profile (emulated, not a device). Harness: [`measure_galaxy.py`](measure_galaxy.py); raw output: [`results.json`](results.json). (2) Jay's **Chrome on Windows** via the Claude extension (real GPU, 120 Hz) for a first idle measurement; later its page was hidden (display asleep), which throttles rendering, so it wasn't used for timing afterwards. |
| Viewports | 390×844, 430×932, 768×1024, 1280×800, 1440×900, 1920×1080 |
| Real iPhone | **NOT VERIFIED** — no device available to the agent; steps for Jay in §J.7 |
| Reproduce | Start instances (see the script header), then `docker run --rm --network host -v "$PWD/docs/design/galaxy-baseline":/work -w /work mcr.microsoft.com/playwright/python:v1.58.0-noble sh -c 'pip install -q --break-system-packages playwright==1.58.0 && python3 measure_galaxy.py'` |

## B. Entry points (LIVE)

| Entry point | Where | Observed |
|---|---|---|
| Projects list tile | `projects_list.html` | "Explore your Galaxy · See how your projects connect" → `/galaxy` (below the list) |
| Menu row | `/more` (`ui_shell.go`) | "Galaxy · Explore your projects as worlds" |
| Command menu | `layout.html` (every page) | "Open Galaxy" (`data-command-terms="galaxy projects planets"`) |
| Direct URL | `/galaxy` | 200 |
| Quick-task return | `return_to=/galaxy` | **CODE** + existing test (`quick_task_test.go`) |

Discoverability (LIVE-PROD, Stage 3): Galaxy is **2 taps** away (Menu → Galaxy, or scroll to the bottom of Projects). There is no Projects → Galaxy segment yet (GYST-VN-039/043).

## C. Current architecture (CODE, LIVE where noted)

- `web/templates/galaxy.html`: hub (title, hint, Browse worlds / All projects / New project) → stage (`canvas[aria-hidden]`, zoom +/−, Reset view, `role=status` line, legend) → preview `aside` → **accessible navigator** (category filter, world list, Preview buttons, moons as nested links) → WebGL fallback text.
- `web/static/js/galaxy.js` (module) dynamically imports vendored **Three.js r180** (`three.module.js` 0.60 MB + `three.core.js` 1.40 MB); helpers `galaxy-camera.js` (camera fit) and `galaxy-gestures.js` (tap/drag/pinch/cancel).
- One `WebGLRenderer` (antialias, alpha, high-performance), pixel ratio ≤ 2, `ResizeObserver`, one `requestAnimationFrame` loop that runs continuously (no idle throttling).
- **Draw calls per frame (LIVE, counted by wrapping WebGL draw functions):** 13 / 21 / 49 / 147 for 1 / 3 / 10 / 33 worlds, i.e. about **9 fixed + ~4 per world** (fixed: core, 3 hub rings, core glow, 2 nebulae, starfield, ship; per world: planet, atmosphere, orbit ring, plus a ring for upcoming/locked and moons).

## D. Data flow (LIVE)

`GET /galaxy` (HTML only, **no write**) → `galaxy.js` → `GET /galaxy/data` (10 s timeout) → JSON planets (id, name, category, colour, summary, target, next, missions, url, state, size, progress, x/y/z, moons). Measured payload: 7.5 KB for 28 worlds, ~50 ms locally. Only **non-complete top-level projects** become planets; their non-complete child projects become moons. State comes from `RouteState(status, progress)`.

## E. Three.js scene behaviour (LIVE unless marked)

- Dark scene (`#03050b`) independent of app theme; central core with rotating hub rings and glow; two nebulae; 850-point spiral starfield; per-project colour; atmosphere shell and faint orbit per world; moons; DOM labels with collision hiding.
- **Visible labels:** only a subset at a time — 3–5 of 33 on phones, 9–12 of 33 on desktop (§J, §K); the rest appear when zoomed or selected.
- Legend Active / Upcoming / Locked. **Paused and upcoming fixtures rendered as `locked`**; "Upcoming" was never produced by the fixtures (§Q).
- **Empty (0 worlds): no scene at all** — no core, stars or ship; controls stay disabled; status "No active planets. Completed worlds have left the Galaxy." ([media](media/galaxy-390x844-empty.jpg)).
- **WebGL unavailable (LIVE, Chromium with `--disable-webgl`):** status "3D view unavailable. Use the project list below.", fallback text visible, all worlds listed in the navigator ([media](media/galaxy-1280x800-webgl-fallback.jpg)).

## F. Ship and camera behaviour

| Behaviour | Evidence | Result |
|---|---|---|
| Ship "Oliver" (pixel sprite, texture 1448×1086 PNG, 1.17 MB) drawn with the camera, bottom centre before any pointer input | Screenshots of every non-empty state | **LIVE** |
| Ship follows the pointer (desktop hover) **and jumps to the tap point on touch** (position is copied each frame, not animated) | Phone world-tap screenshot shows the ship beside the tapped world ([media](media/phone-webkit_iphone13_emulated-390-world-tap.jpg)); desktop screenshots | **LIVE** |
| Ship flap/tilt animation, off under reduced motion | Byte-level crop comparison was inconclusive | **NOT VERIFIED** (CODE: `sin` flap, disabled when reduced) |
| Selecting a world flies the camera to it | Label settle time after Preview: **2.3–3.8 s** (1–33 worlds) | **LIVE** |
| Reduced motion (OS `prefers-reduced-motion` **and** the app's *Reduce motion* setting, `localStorage lifeops-motion`) | Flight settles in **0.11 s** (instant) in both cases | **LIVE** |
| Escape / close / Reset return to the overview | Escape and close hide the panel; status "Returning to the Galaxy map" | **LIVE** |
| Desktop hover roam near the stage edges | Labels moved **31 px in 2 s** with the pointer at the right edge; **0.1 px** at the centre (dead zone) | **LIVE** |
| Zoom: +/− buttons, wheel, pinch | Two "+" clicks: label spread 100 → 129 px; pinch-out: 72 → 110 px; wheel **CODE** | **LIVE** (buttons, pinch) |

## G. World positioning (LIVE on a disposable instance)

- Positions are **deterministic** (SHA-256 of the project UUID → spherical radius 18–90) and **persisted on the first `GET /galaxy/data`**.
- Observed: 9 fictional projects all `pos = NULL` → after `GET /galaxy` (HTML) still NULL → after the first `GET /galaxy/data` the 5 non-completed projects had positions; the 4 completed stayed NULL. Later reads return identical positions.
- The write is `UPDATE projects SET pos_x, pos_y, pos_z, updated_at = CURRENT_TIMESTAMP` (`internal/repository/projects.go:116`), so **viewing Galaxy also bumps `updated_at`** of newly positioned projects.
- LIVE-PROD (Stage 3): before viewing production, the write-on-GET would have updated **0 rows** (all eligible projects already had positions). No production write happened in this task.
- The client flattens y (×0.16); locked worlds sit 1.35× further out (CODE).

## H. Selection and hit testing

Rule (CODE): raycast hit, else the world whose projected edge is within **28 px** (44 px for the already-selected world), minimum screen radius 5 px, nearest edge wins.

Measured (LIVE, binary search in 4 directions from each world's centre; 0.65 s between probes to avoid double-click):

| Scene | Uncontested edge distance | Where neighbours compete |
|---|---|---|
| 10 worlds, desktop mouse, 1280 | **33 px** (≈ 5 + 28), 41–44 px for larger worlds | 14–28 px, lost to a nearby world |
| 10 worlds, phone touch, 390 | 33–41 px | 11–23 px |
| 30 worlds, desktop, 1280 | 39–66 px (larger/closer worlds) | **8–28 px** |
| 30 worlds, phone, 390 | 44–77 px | **2–9 px** in crowded directions |

So an **isolated world already offers ≈ 66 px of target diameter**; the real problem is **crowding** — in a 30-world scene some directions give only a few pixels before the tap selects a neighbour. Drag (> 8 px) never selects (LIVE); pinch never selects (CODE + `check-galaxy.cjs`).

- **Selected world, 44 px padding / click-again-to-open:** **NOT VERIFIED.** In the harness, clicking the centre of the already-selected world after its flight did not open the project within 1 s; the cause (preview panel/label overlap or harness timing) wasn't isolated. Open project via the panel button works (LIVE).
- **Double-click / double-tap:** during the desktop sweep, two clicks < ~0.2 s apart on a world **navigated to its project** (LIVE, incidental, matches the `dblclick` handler); a controlled double-tap in emulated Chromium touch **did not** open the project. Real-iPhone double-tap: NOT VERIFIED.

## I. Preview behaviour (LIVE)

- Navigator **Preview** → world selected, panel shown, focus moves to **Open project**; status "<name> · <progress>% · <state> · use Open project to enter".
- Panel content: category, name, summary, progress bar, "<n> missions · <p>% complete · target <date>", "Next mission: …" / "No pending mission queued.", **Open project** (`/projects/<id>`).
- **Close control: renders as an empty white square** — foreground `rgb(231,242,251)` on `rgb(239,239,239)`, **contrast 1.01:1** (Stage 3 production: ~1.05:1). Size **30×43** at ≥ 768 px, 44×44 on phones.
- Closing returns focus to the **stage**, not to the Preview button that opened it (LIVE, earlier session).
- Desktop (≥ 768): panel overlays the stage top-right ([media](media/galaxy-1280x800-selected.jpg)). Phones (< 768): panel sits **below** the stage and the page scrolls to it ([media](media/phone-webkit_iphone13_emulated-390-world-tap.jpg)).

## J. Mobile findings

1. Layout (LIVE, no horizontal overflow at 390/430): stage 390×560 / 430×648; +/− 44×44, Reset 92×44, Preview buttons 83×44, close 44×44, Open project 332×44 / 372×44.
2. Touch: tap selects; one-finger drag orbits (labels moved 36 px) and **does not select**; pinch zooms (LIVE, Chromium touch via CDP).
3. After a **world tap** or **Preview**, Open project is **fully visible above the tab bar** (top/bottom 702/746 of 844 at 390) in both Chromium and WebKit iPhone emulation. *(The earlier overnight preflight note that the tab bar covered it came from a hidden, frame-throttled tab and is retracted.)*
4. **Header hide-on-scroll has no visible effect:** scrolling adds `body.galaxy-nav-hidden`, but the CSS targets a legacy `body > .nav` that the current shell no longer has; the top header and bottom tab bar stay visible (LIVE).
5. Only 3–5 of 33 labels visible on phones.
6. Load to "Drag to explore" in WebKit iPhone emulation: 2.9 s cold (first), 0.85 s warm.
7. **Real iPhone steps for Jay** (Safari and installed PWA, production, read-only — do not create or edit projects): open Galaxy from Projects; note seconds until worlds appear; drag, pinch, tap a world, open and close the preview, tap Open project; enable *Settings → Reduce motion* and repeat one selection; report stutter (yes/no), time to first worlds, and anything that didn't respond.

## K. Desktop findings (LIVE)

- 768 / 1280 / 1440 / 1920: stage 768×800 / 1072×625 / 1232×725 / 1712×850; no overflow; 9–12 visible labels; controls 44 px; **close 30×43 and Open project 107×40 are under 44 px**.
- Mouse: click previews; hover near edges roams; +/−/Reset work; Escape closes; arrows move selection **only when a world lies in that direction** (true in one session, no neighbour in the other).
- Real GPU (Jay's Windows Chrome, 28 worlds, 120 Hz, earlier in the day): idle **116 fps, median 8.3 ms, p95 8.6 ms**, one 115 ms long task at start.

## L. Accessibility

| Check | Result |
|---|---|
| Keyboard | Tab order: Browse worlds → All projects → New project → stage ("Interactive project galaxy") → Zoom in → Zoom out → Reset view → Category → list links / Preview buttons (LIVE). Arrows select neighbours; Escape recentres. |
| Non-visual path | Canvas is `aria-hidden`; the navigator lists every world with a link and a labelled "Preview <name>" button; the status line is `role=status` (LIVE, accessibility snapshot) |
| WebGL fallback | Works; list stays usable (LIVE) |
| Contrast (vs scene `#03050b`) | Labels 13.7:1, status 16.5:1, legend 9.4:1 by colour — but labels/legend are ~10 px text. **Close button 1.01:1** (fails). |
| Targets | Controls 44 px; close/Open project under 44 at ≥ 768; world targets per §H |
| Reduced motion | Honoured for camera flight (OS setting and app setting) (LIVE); ship/ring animation NOT VERIFIED |
| Focus return after close | Goes to the stage, not the invoker |
| VoiceOver | **NOT VERIFIED** |

## M. Performance baseline

**How measured:** `requestAnimationFrame` intervals in the page (idle 3 s; during a 40-step mouse drag); a `PerformanceObserver('longtask')`; WebGL draw calls counted per frame by wrapping `drawElements/drawArrays`; "ready" = navigation start until the status reads "Drag to explore"; "flight" = time until the selected world's label moves < 0.5 px for 0.5 s. **Headless numbers are software-rendered (SwiftShader) — use them only relative to each other and to future runs of the same harness; they are not device FPS.**

| Worlds | Ready (ms) | Long task at start | Draw calls/frame | Idle fps · median / p95 ms | Drag median / p95 ms | Preview → status (ms) | Flight settle (s) | Console errors |
|---|---|---|---|---|---|---|---|---|
| 0 | 242 | none | 0 | — (no scene) | — | — | — | 0 |
| 1 | 227 | 144 ms | 13 | 45 · 16.7 / 33.4 | 16.7 / 33.4 | 91 | 3.8 | 0 |
| 3 | 188 | 101 ms | 21 | 45 · 16.7 / 33.4 | 16.7 / 33.4 | 106 | 2.3 | 0 |
| 10 | 212 | 128 ms | 49 | 36 · 33.3 / 33.4 | 33.3 / 50 | 74 | 3.8 | 0 |
| 33 | 227 | 134 ms | 147 | 23 · 49.9 / 66.7 | 49.9 / 66.6 | 131 | 2.9 | 0 |

- Real GPU reference (Windows Chrome, 28 worlds): idle 116 fps (median 8.3 ms, p95 8.6 ms); DOMContentLoaded 198 ms; one 115 ms long task at start.
- Assets: Three.js 2.0 MB decoded; `galaxy.js` 24 KB; ship PNG 1.17 MB for a small sprite (optimisation candidate).
- Cost grows with world count (~4 draw calls per world); the render loop never idles.
- **Real iPhone: NOT VERIFIED** (§J.7).

## N. Completed-project behaviour (LIVE)

37 fictional projects including 4 `done` → `/projects` lists all 37, `/galaxy/data` returns 33: **completed projects are excluded** and never receive a stored position. LIVE-PROD (Stage 3): production showed 2 planets while 11 of 13 projects were done.

## O. Sparse-state behaviour (LIVE)

- 1 world: a small dot left of the sun, one label ([media](media/galaxy-1280x800-sparse-1.jpg)); 3 worlds similar ([media](media/galaxy-1280x800-sparse-3.jpg)). The scene reads as mostly empty; no hint that completed work is hidden.
- 0 worlds: no scene (§E).

## P. Regression checklist (parity list for every later Galaxy task)

Run `node scripts/check-galaxy.cjs` and the harness; compare against the values above. Every Galaxy PR records the result.

- [ ] Entry points still reach the real `/galaxy` (§B), and none removed before its successor exists
- [ ] Three.js scene renders with core, starfield, nebulae, worlds, moons, labels and the ship (§E, §F)
- [ ] Navigator lists every world with link + "Preview <name>"; category filter still filters it (§C, §L)
- [ ] Preview → panel → focus on Open project → `/projects/<id>` (§I)
- [ ] Tap/click selects; drag never selects; pinch/+/−/wheel zoom; Reset/Escape recentre (§F, §H)
- [ ] Uncontested hit edge ≥ 33 px (isolated world ≥ 66 px diameter) — must not shrink; crowding improvements recorded as numbers (§H)
- [ ] Camera flies to the selected world; reduced motion (OS and app setting) makes it instant (§F)
- [ ] Ship present and following pointer/tap (§F)
- [ ] Positions unchanged for existing worlds (compare `/galaxy/data` x/y/z before/after on a seeded copy) (§G)
- [ ] Completed projects excluded unless a task deliberately changes that (then hidden by default, `17A` §I) (§N)
- [ ] WebGL fallback still shows the list (§E)
- [ ] No horizontal overflow at 390–1920; controls ≥ 44 px (no regression of existing sizes) (§J, §K)
- [ ] Draw calls/frame and frame timings no worse than §M for the same dataset without justification
- [ ] No console errors at load or during interaction
- [ ] Projects list still exists separately; Galaxy still a peer view

## Q. Corrections to `17A` §B (and related specs)

| 17A claim | Live result | Status | Notes |
|---|---|---|---|
| Entry points: Projects tile, command menu, More row, `/galaxy`, quick-task return | Tile, command menu, Menu row, URL confirmed; quick-task via existing test | MATCH | 2 taps from the tab bar (Stage 3) |
| Page structure: hub → stage → preview → accessible navigator → fallback | As described | MATCH | |
| Three.js architecture (vendored, dynamic import, one loop, pixel ratio ≤ 2) | Confirmed; **r180**; ~9 + 4/world draw calls | MATCH | Added numbers |
| World generation: active top-level planets, non-complete children as moons, done/completed/archived excluded | Confirmed (33 of 37) | MATCH | |
| Positions deterministic, persisted on first `/galaxy/data` read | Confirmed; the write also bumps `updated_at` | MATCH + detail | |
| Locked 1.35× out, rings for upcoming/locked; legend Active/Upcoming/Locked | Paused and upcoming fixtures render as **locked**; "Upcoming" never produced; the create form stored "paused" as `upcoming` | **DIFFER** | Legend entry may be unreachable |
| Visual identity (core, rings, nebulae, starfield, colours, atmospheres, labels, halo) | Confirmed in screenshots | MATCH | Only a subset of labels visible |
| Ship rides with the camera, follows the pointer, flap animation | Follows pointer; **also jumps to the tap point on touch**; flap not verified | MATCH / **DIFFER (detail)** / NOT VERIFIED | Movement is instant, not a flight |
| Camera: orbit, drag, pinch/wheel/+/−, hover roam, fly to selection, Reset/close/Escape return, reduced-motion snaps | All confirmed except wheel (CODE) | MATCH | Flight 2.3–3.8 s; reduced 0.11 s |
| Hit test 28 px (44 px selected), 5 px minimum | Uncontested edge 33 px confirmed; **crowding** reduces targets to 2–28 px; selected 44 px NOT VERIFIED | MATCH / NOT VERIFIED | Isolated worlds already ≈ 66 px diameter |
| Keyboard: stage focusable, Escape, arrows | Confirmed; arrows direction-dependent | MATCH | |
| Preview and opening; mouse click-again / double-click / context-click opens | Panel + Open project confirmed; double-click open observed incidentally; click-again NOT VERIFIED; context-click CODE | PARTIAL | |
| Category filter filters the list only | Confirmed | MATCH | |
| **Mobile: app header hides while scrolling** | **No visible effect** — CSS targets legacy `body > .nav` | **DIFFER** | Dead styling path |
| Tests (`check-galaxy.cjs`) | Pass | MATCH | |
| Performance "no measured figures" | Now measured (§M); iPhone still missing | UPDATED | |
| (Not in §B) Empty state | **No scene, no ship, controls disabled** | NEW | `12` §Galaxy wants "one sun + create"; `17A` §G goal stands |
| (Stage 3 / 15 wording) "+/−/Reset small" | Already 44 px; only close (30×43) and Open project (107×40) at ≥ 768 are under 44 | **DIFFER** | GYST-VN-044/045 scope |
| (Preflight, 2026-09-27 overnight) "Open project covered by the tab bar on phones" | Not covered when rendering normally | **RETRACTED** | Artifact of a hidden tab |
| (`13` spec) "double-tap recenters" | Double-click on a world opens it; on empty space while selected it recentres (CODE) | DIFFER (spec vs current) | Design decision for a later task |

## R. Known risks for future Galaxy work

1. **Crowding** is the real hit-target problem; enlarging padding alone increases "wrong world" selections — needs spacing/label/priority rules (GYST-VN-044).
2. **Write-on-GET** persists positions and bumps `updated_at`; GYST-VN-028 must keep positions identical and decide what happens to `updated_at`.
3. **Completed worlds** have no stored position (GYST-VN-046 must define placement without moving active worlds).
4. **Render cost scales with worlds** (~4 draw calls each) and the loop never idles — richer effects (GYST-VN-144) must be measured with this harness and on a real iPhone.
5. **Label visibility**: most worlds are unlabeled at a time; any change to labels affects discoverability and hit expectations.
6. **Close control** is effectively invisible (1.01:1) and under 44 px on desktop (GYST-VN-045).
7. **Dead scroll-hide CSS** and the "Upcoming" legend may mislead implementers; don't "fix" them silently — decide in a task.
8. **Terminology** ("missions") lives in `galaxy.js` strings (GYST-VN-027 must touch strings only).
9. **Evidence gaps:** real iPhone (Safari + PWA), VoiceOver, ship flap, selected-world click-again — close them before or during the first material Galaxy change.
