# Stage 8A — Galaxy Preservation (binding)

- **Decided by:** Jay (product owner)
- **Recorded:** 2026-09-27 on `audit/vnext`
- **Status:** **Binding** product constraint for all vNext work. It sits alongside `05A`, `08A`, `10A` and `13A`, and overrides any Stage 7 or Stage 8 wording that could be read as permitting Galaxy removal or replacement.
- **Implementation:** none. Documentation only; no production code changed.
- **Evidence base:** read-only inspection of the production code at `dab7995`:
  - `web/templates/galaxy.html`
  - `web/static/js/galaxy.js`, `galaxy-gestures.js`, `galaxy-camera.js`
  - `web/static/css/ui-galaxy.css`
  - `internal/services/galaxy.go`
  - `internal/handlers/handlers.go` (`galaxy`, `galaxyData`)
  - `scripts/check-galaxy.cjs`

  §B describes what the code does. It isn't a substitute for the runtime baseline required in §F.

---

## A. Why Galaxy is protected

Galaxy is GYST's **signature feature**. It is the spatial, playful expression of Jay's projects and the clearest part of GYST's identity. Earlier decisions already point this way:
- `AGENTS.md`: "Do not replace the existing Galaxy without explicit approval".
- `05A` D3: Galaxy, worlds, the ship and XP carry GYST's personality.
- `05A` D4: completed projects belong in Galaxy.

The audits found usability problems: small targets, hard to find, preview close control, sparse scenes. Those are **reasons to improve Galaxy, not to replace it**. A conventional list already exists (Projects), so a Galaxy reduced to a list would add nothing.

**Philosophy:**

- **Required:** PRESERVE → STABILIZE → IMPROVE → AMPLIFY
- **Not allowed:** DELETE → REBUILD FROM SCRATCH

## B. Existing production Galaxy definition

This is the Galaxy that must be preserved and evolved (as of `dab7995`):

> **Verified and corrected at runtime by GYST-VN-143:** see [`docs/design/galaxy-baseline/README.md`](../../design/galaxy-baseline/README.md) §Q for the claim-by-claim result. Where they differ, the baseline's LIVE findings describe the current Galaxy.

| Area | Current behaviour |
|---|---|
| **Entry points** | Projects list "Galaxy" entry tile (`projects_list.html`); command menu "Open Galaxy" (`layout.html`); More destination "Galaxy · Explore your projects as worlds" (`ui_shell.go`); direct `/galaxy`; quick-task `return_to=/galaxy`. Pinned by `navigation_test.go` and `ui_shell_test.go`. |
| **Page structure** | Hub header (kicker, title, hint, buttons: Browse worlds · All projects · New project) → 3D stage (canvas, zoom +/−, Reset view, status line, legend Active/Upcoming/Locked) → preview panel (`aside`, close ×, category, name, summary, progress bar, meta, next item, "Open project") → **Accessible navigator** (category filter, list of worlds with links, "Preview" buttons, moons as nested links) → WebGL/data failure fallback text. |
| **Three.js architecture** | Vendored `three.module.js`, loaded by dynamic `import()`; on failure the fallback is shown. `WebGLRenderer` (antialias, alpha, high-performance), pixel ratio ≤ 2, `ResizeObserver` resizing, one `requestAnimationFrame` loop. Scene: dark background `#03050b`, exponential fog, `PerspectiveCamera` 55°, ambient light + central point light. |
| **World generation** | `GET /galaxy/data` returns **active top-level projects** as planets. Their non-complete child projects are moons. Projects with status `done`/`completed`/`archived` are **excluded** ("Completed worlds have left the Galaxy"). Each planet has id, name, category, colour, summary, target, next due item, item count, URL, route state (`RouteState`), size (from item count, capped at 3), progress, and x/y/z. **Positions are deterministic** (SHA-256 of the project UUID → spherical coordinates, radius 18–90). They are persisted on first read (the GET write in S2-15). The client flattens y (×0.16). Locked worlds sit 1.35× further out at half opacity. Upcoming/locked worlds get a ring. |
| **Visual identity** | Central glowing core with three slowly rotating hub rings and a glow sprite; two nebula clouds (blue, violet); 850-star spiral-arm starfield; per-project colour palette (cyan, violet, emerald, amber, rose, blue, orange, magenta); atmosphere shell per world; faint orbit ring per world; moons; DOM labels with collision avoidance; selection halo that pulses; selected world scales ×1.14 with raised emissive. The scene stays dark regardless of app theme. |
| **Ship** | "Oliver", a pixel-art sprite (`/static/images/oliver-galaxy-pixel.png`) attached to the camera. It rides along on every camera flight, follows the pointer within the view, and has a subtle flap/tilt animation (disabled under reduced motion). |
| **Camera and movement** | Orbit camera (yaw/pitch/radius) fitted to all worlds by `galaxyCameraFit` (portrait/landscape aware). Drag orbits; pinch, wheel and +/− zoom (radius 58 … 1.8× default). On desktop (fine pointer) hover near the edges slowly roams the orbit (dead-zone). **Selecting a world flies the camera to it**; Reset/close/Escape flies back. Motion is eased; reduced motion snaps instantly. |
| **Selection** | Tap/click: raycast hit, else nearest world whose screen edge is within **28 px** (44 px for the already selected world); minimum screen radius 5 px. Gesture model (`galaxy-gestures.js`): movement > 8 px or a second pointer is never a tap; cancellation is safe. Keyboard: the stage is focusable; Escape recentres; arrow keys move to the nearest world in that direction once one is selected. The navigator's "Preview" buttons select a world and focus "Open project". |
| **Preview and opening** | The preview panel shows the selected world's details. "Open project" links to `/projects/{id}`. Mouse: clicking the already selected world, double-clicking a world, or right-clicking a world opens the project. Touch: tap previews, then "Open project". On phones the panel scrolls into view. Close (×) recentres. |
| **Filters / rules** | Category filter in the navigator (filters the list only). Only unfinished projects appear (see World generation). |
| **Mobile** | Stage and page tuned for phones (`ui-galaxy.css` mobile rules); app header hides while scrolling down on the Galaxy page and returns on scroll up. |
| **Tests** | `scripts/check-galaxy.cjs`: tap vs drag vs pinch vs cancel, and camera-fit bounds for 320×568, 390×600, 844×280, 1440×800. |
| **Performance (known from code only)** | One draw loop at display rate, no frame throttling when idle; about 5–7 meshes per world; the ship texture PNG is about 1.17 MB. **No measured frame-time or load figures exist yet.** The baseline task measures them. |

## C. Non-negotiable preservation rules

1. The real, production **Three.js Galaxy stays**. It is evolved in place.
2. **The spatial world concept stays:**
   - space scene
   - worlds per project
   - moons for sub-projects
   - camera flight to the selected world
   - the ship
   - the preview → open project flow
   - the project data relationship
3. **The ship stays** unless an explicitly approved task changes its behaviour.
4. Existing working interactions stay unless an **approved replacement** exists in the same task:
   - drag
   - pinch/wheel/buttons zoom
   - tap/click select
   - keyboard navigation
   - open project
   - reset
   - reduced motion
   - WebGL fallback
5. The **accessible navigator** (world list) is a supplement and the WebGL-failure fallback. It is **never** promoted to replace a working scene.
6. The **Projects list stays a separate view**. Galaxy stays a peer, alternate and signature project view (Projects · Development · **Galaxy** · Roadmaps, `05A` D2).
7. Entry points never drop below the current set until their approved successors exist (for example, the More row stays until Projects → Galaxy ships).
8. The Stage 7 prototype (`prototypes/galaxy.html`) is a **design reference only**. It is not a substitute and not permission to replace the production implementation.

## D. Allowed enhancements

Agents may, and are encouraged to:
- improve, refactor safely and clean up the existing code
- improve performance, mobile controls, touch targets, camera behaviour, world spacing, labels, world visuals, previews, close/back controls, loading, empty and sparse states, and responsive layout
- add filters, completed projects, active/completed treatment, outer-orbit concepts, and hide/show completed
- add better animations, subtle effects, richer project information and deeper usefulness
- improve accessibility, keyboard support, reduced-motion behaviour and discoverability
- connect Galaxy more clearly to Projects

## E. Prohibited replacements and regressions

Agents must not:
- delete Galaxy
- replace it with cards, a table or a normal project list
- replace Three.js merely because plain HTML is easier
- remove the ship
- remove the spatial/world concept
- reduce Galaxy to a decorative background or a static screenshot/image
- route users away from the real Galaxy and leave it abandoned
- silently rebuild it as an unrelated implementation
- remove working interactions without an approved replacement
- use the Stage 7 static prototype as a substitute

**A larger rewrite requires all six of the following:**
1. A concrete technical reason.
2. A full understanding of existing behaviour (§B plus the baseline).
3. An explicit feature-parity document.
4. Preservation of the signature experience.
5. Regression tests and browser evidence.
6. **Jay's explicit approval of the rewrite task before implementation.**

## F. Baseline capture requirements

Task GYST-VN-143 must be done **before the first task that materially changes Galaxy**. That includes GYST-VN-028, GYST-VN-037, GYST-VN-043, GYST-VN-044, GYST-VN-045, GYST-VN-046, GYST-VN-047, GYST-VN-048, GYST-VN-049. The baseline records:

- entry points
- Three.js architecture
- project-data flow
- world generation and position rules
- ship behaviour
- selection
- preview
- camera controls
- desktop and mobile interactions
- filters and rules
- how unfinished projects are selected
- important visual details (all of §B, verified at runtime)
- **Performance:** load time to first frame, steady-state frame time and dropped frames while idle, dragging and flying to a world. Measured on desktop Chrome and on a real iPhone (Safari and installed PWA), with 3, 10 and 30 seeded worlds.
- **Media:**
  - screenshots or short recordings at 390×844 and 1280×800 (idle, selected with preview open, zoomed, empty, sparse, WebGL-fallback)
  - **fictional seeded data only**, never Jay's private projects
  - stored under `docs/design/galaxy-baseline/` or referenced from the task record
- **Parity checklist:** the list of behaviours every later Galaxy task re-checks (§L).

## G. Mobile goals

- **Touch targets:** every world offers about a **56 px** usable target, or an equivalent comfortable area. Build on the existing screen-space edge padding rather than replacing the hit logic.
- **Controls:** zoom, reset and close are at least 44 px.
- **Gestures:** no conflict between page scroll and scene gestures; pinch stays reliable.
- **Preview:** a readable bottom sheet (or equivalent) with a visible close control, clear hierarchy (name, state, progress, next item) and a primary "Open project".
- **Layout:** the scene is about 64 vh (minimum 420 px) and sits comfortably within safe areas.
- **Verification:** each of these is checked on a real iPhone in Safari and in the installed PWA.

## H. Desktop goals

- Keep the mouse behaviours: hover roam, wheel zoom, click to select, click-again / double-click / context-click to open.
- Preview as a side panel inside the scene.
- Full keyboard path: focus the stage, select via the navigator or arrows, Escape to recentre, Enter/"Open project".
- Larger scene; labels readable without overlap.
- **Discoverability:** Galaxy is reachable from the Projects sub-navigation and the sidebar, and is a launcher Go result.

## I. Completed-project behaviour

- Completed projects may appear dimmed, distant, on an outer orbit, or in an archived visual state (`05A` D4).
- They **must be hideable**. They are hidden by default and shown via a Done toggle, remembered per device.
- They must not clutter or out-shine active worlds. Active worlds stay dominant in size, brightness and position.
- They reuse the deterministic positions so a world doesn't jump when its project completes, or they move to an outer orbit by a documented rule.
- They are never shown by colour alone. The navigator marks them in words too.

## J. Performance requirements

- **No regression** against the baseline (frame time, first-frame time) **without documented justification** in the task record.
- Target: smooth interaction on the iPhone used by Jay. Any richer rendering (atmospheres, particles, rings, lighting) is **measured before and after** on that device.
- Respect `prefers-reduced-motion` and the app's reduced-motion setting: no continuous non-essential animation, and instant camera moves.
- Consider pausing or throttling the render loop when the page is hidden or idle.
- Keep pixel ratio capped (currently ≤ 2).
- Keep asset sizes in check (the ship PNG, about 1.17 MB, is a candidate for optimisation without visual change).

## K. Accessibility requirements

- The accessible navigator stays and improves: every world is reachable, previewable and openable without the canvas.
- Every control has an accessible name.
- The preview gets focus management: focus moves in on open, Escape and close work, and focus returns to the invoking world or item.
- State is conveyed by words and icons, not colour alone (DD-4 applies to state colours outside the scene's identity palette).
- Reduced motion is honoured (§J).
- Visible focus on the stage and the navigator.

## L. Regression checklist (every Galaxy task)

- [ ] The real Galaxy still exists at `/galaxy`.
- [ ] Three.js/spatial interaction still works.
- [ ] Existing project worlds remain accessible (scene and navigator).
- [ ] The ship remains part of the experience unless an explicitly approved task changes its behaviour.
- [ ] Project selection works (tap, click, keyboard, navigator "Preview").
- [ ] Project preview and detail access work ("Open project").
- [ ] Desktop interaction works (drag, wheel, hover roam, click/double-click open).
- [ ] Mobile interaction works (tap, drag, pinch; real iPhone for material changes).
- [ ] No major existing Galaxy behaviour was silently lost (compare with the baseline parity list).
- [ ] No replacement list or grid was substituted for Galaxy.
- [ ] The normal Projects list remains available separately.
- [ ] Galaxy remains an alternate/signature project view.
- [ ] Performance is no worse than the baseline without documented justification.
- [ ] Reduced-motion and accessibility behaviour was considered and checked.
- [ ] Completed-project enhancements don't clutter active projects.
- [ ] Completed projects can be hidden/filtered (once GYST-VN-047 has shipped).
- [ ] `node scripts/check-galaxy.cjs` passes (extended when behaviour is added).

## M. Projects list, Galaxy and Roadmaps

```text
Projects (tab)
├── Projects   the working list: filter, find, act (default)
├── Development GitHub work (Phase 8+)
├── Galaxy     the signature spatial view of the same projects
└── Roadmaps   ordered routes of projects toward a goal
```

- **One data source.** Galaxy, the list and Roadmaps read the same projects and the single status mapping (GYST-VN-037). Galaxy never keeps its own project state.
- **Peers, not substitutes.** The list is for efficient work; Galaxy is for orientation, identity and delight. Neither replaces the other.
- **Links both ways.** A Galaxy preview opens the project. The Projects area shows Galaxy as a peer segment. Roadmaps may deep-link to worlds later (designed for later, not required).
- **Roadmaps** keep their own route/rocket visual. That isn't a replacement for Galaxy either.

## N. Guidance for Claude/Codex implementation agents

1. Read this file, `13A` and the baseline (GYST-VN-143) before touching `galaxy*.js`, `galaxy.html`, `ui-galaxy.css`, `internal/services/galaxy.go` or `/galaxy/data`.
2. Change Galaxy **incrementally in the existing files**. Extract helpers (as `galaxy-camera.js` and `galaxy-gestures.js` already do) rather than rewriting. Add unit-testable pieces to `scripts/check-galaxy.cjs`.
3. Keep the `/galaxy/data` JSON contract backward compatible. Add fields; don't rename or remove them.
4. Keep world positions stable. Never re-randomise them.
5. **Shared-change awareness:** any change to shared CSS/JS, the shell, CSP headers, terminology, the status mapping or project data must include the §L quick check (Galaxy loads, a world can be selected and opened, the ship is visible).
6. When something can't be done without removing behaviour, **stop and report** to Jay with options. Don't remove the behaviour and don't substitute a simpler UI.
7. Use the Stage 7 prototype for layout, sheet and control intent only.
8. Record browser/device evidence and performance numbers in the task record.

## O. Conditions requiring Jay's explicit approval

- Any rewrite or replacement of the Three.js implementation, or of the rendering library
- Removing or materially changing the ship, camera flight or the world/moon concept
- Removing any working interaction, entry point or the accessible navigator
- Changing how world positions are generated, or moving existing worlds
- Showing completed projects by default, or changing the D4 hide/show rule
- Adding a rendering technique that measurably regresses iPhone performance
- Making Galaxy the default Projects view, or removing it from the Projects area
- Any visual encoding that adds attention or state badges on worlds beyond the approved enhancement tasks (GYST-VN-145)

## Carry-forward to Stage 9 (governance)

Stage 9 must add this short, non-negotiable rule to the permanent `AGENTS.md` / `CLAUDE.md` governance:

> "GYST Galaxy is a protected signature feature. Agents may enhance and refactor it but must not delete, replace, or downgrade the real Three.js Galaxy without explicit product-owner approval."

`AGENTS.md` is **not** modified by this stage.

## Backlog changes made with this decision

Documentation only, in `15`/`14`/`16`/`17`. IDs were kept stable, and new tasks were appended.

| Change | Tasks |
|---|---|
| **Added** the Galaxy baseline task (required before material Galaxy changes) | GYST-VN-143 |
| **Added** amplification tasks for goals 6 and 7 (visual depth; subtle project-state encoding), P3, gated on the baseline and performance measurement | GYST-VN-144, GYST-VN-145 |
| Full §L regression checklist, `17A` reference, and "prototype is reference only" added | GYST-VN-043, GYST-VN-044, GYST-VN-045, GYST-VN-046, GYST-VN-047, GYST-VN-048, GYST-VN-049, GYST-VN-028 and the new tasks |
| Wording corrected to match the real Galaxy (existing navigator, filter, hit padding, ship behaviour, completed-project exclusion) | GYST-VN-043, GYST-VN-044, GYST-VN-045, GYST-VN-046, GYST-VN-047, GYST-VN-048, GYST-VN-049 |
| Position stability and data contract made explicit | GYST-VN-028, GYST-VN-037 |
| Entry points protected until successors exist | GYST-VN-025, GYST-VN-026, GYST-VN-039, GYST-VN-040, GYST-VN-063 |
| Galaxy quick regression check added for shared-surface tasks | GYST-VN-018, GYST-VN-019, GYST-VN-023, GYST-VN-024, GYST-VN-025, GYST-VN-026, GYST-VN-027, GYST-VN-037, GYST-VN-039, GYST-VN-040, GYST-VN-061, GYST-VN-063, GYST-VN-075 |
