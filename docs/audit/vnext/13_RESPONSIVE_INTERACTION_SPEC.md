# Stage 7C — Responsive & Interaction Spec

- **Stage:** 7 (with `11`, `12`, `prototypes/`)
- **Principle:** one conceptual hierarchy on every device. Presentation changes,
  concepts don't.

---

## 1. Breakpoints

Four layout breakpoints, mobile-first. No device-specific layouts.

| Range | Name | Layout |
|---|---|---|
| < 600 | Phone (390–430 targets) | Bottom tabs · single column · 16 gutter · sheets from bottom |
| 600–1023 | Tablet / small laptop (768 target) | Bottom tabs · single column (max 760) · 24 gutter · 3-col metrics · centred floating sheets |
| 1024–1279 | Transitional / laptop (1024 target) | Sidebar 232 · content 760 (wide pages 1120, two columns) · 32 gutter · 4-col metrics · no inspector |
| ≥ 1280 | Desktop (1440 target) | + contextual **inspector** 380 on list screens that declare one |
| ≥ 1600 | Wide (1920 target) | Sidebar 256 · inspector 440 · content caps (820 / 1120); extra width becomes margin, not more content |

Additional micro-rules:
- < 420: segmented padding 9 px so 5 segments fit.
- < 480: decorative tiles in the Now sub-strip hide so values never wrap.

## 2. Phone navigation

- **Bottom tab bar** (Today · Projects · Health · Learn · More): fixed, blurred,
  56 + safe-area-bottom, 44+ targets, labels always visible.
- **Badge:** Today only; counts Attention `needs_action` + `critical`. The Today badge is the Attention entry.
- **Tab re-tap:** scrolls to top; a second re-tap pops to the tab root.
- **Top bar:** back (if deeper) · title (fades in once the large title scrolls away) · ≤2 icon actions (search, add).
- **Depth:** tab root → detail page → (sheet). No more than 3 levels before a sheet or an external app.
- **Back:** top-bar back button and the system back gesture. State (filters, scroll) is restored on return.

## 3. Tablet navigation

- 600–1023 keeps the bottom tabs (thumb reach on iPad portrait; consistent with phone).
- Lists stay single-column. Metrics widen to 3 columns. Sheets float centred (max 560) with rounded corners on all sides.
- **Landscape iPad ≥ 1024** switches to the sidebar automatically (desktop rules).

## 4. Desktop navigation

- **Sidebar:**
  - GYST brand
  - "Search or run… ⌘K" (opens the launcher)
  - Today (badge) · Projects (expands Development / Galaxy / Roadmaps while inside Projects) · Health · Learn
  - **Operate** group: System (status dot: worst of crit/warn; green when healthy, quiet) · Apps (D6)
  - More at the bottom
- **No Attention item** (C1): reached via the Today badge, Today section, launcher and deep links.
- Current item: accent-soft fill + accent text.
- **Keyboard:** ⌘/Ctrl-K launcher; `g t` / `g p` / `g h` / `g l` go-to shortcuts (optional; must not conflict with inputs).

## 5. Secondary navigation

| Level | Pattern | Example |
|---|---|---|
| Area within a tab | Sticky **segmented control** under the page title | Projects · Development · Galaxy · Roadmaps; Summary · Nutrition · Training · Progress · Plan |
| View within an area | **Filter pills** (a third level; never a second segmented control) | Development: Tasks · Reviews · Agents · Repos |
| Sections within a detail | Segmented control switching sections in place | Project: Tasks · Roadmap · Activity · Resources |
| Filters | Pills with counts | Active · 4 · Planned · 2 |

Sticky offset: under the top bar (52 + safe-top). The segmented control scrolls
horizontally only as a fallback.

## 6. Sheets

- **Use for:** capture, re-authentication, previews (Galaxy world, review summary on phone), item options (snooze/dismiss), short edits.
- **Phone:** bottom sheet, grab handle, 24 radius top, max 88 vh, content scrolls inside, close via ✕ (44 target), scrim tap, swipe down, or Escape.
- **≥ 600:** centred floating sheet, max width 560, bottom 24.
- One sheet at a time; a sheet may open a dialog (confirm), never another sheet.
- Focus moves into the sheet and returns to the invoker. The background is inert (`aria-modal`).

## 7. Inspectors

- **≥ 1280 only**, on list screens that declare one (Development, Reviews, Goals, Settings, Learn optional).
- Selecting a row fills the inspector **without navigation**. Below 1280 the same row navigates to the detail page (or opens a sheet for Reviews).
- The inspector shows the detail's **Level 1** only (hero, key facts, next action). "Open" goes to the full page.
- The selected row keeps the accent-soft highlight. Arrow keys move the selection; Enter opens the full page.
- The inspector never contains forms longer than 3 fields (use a sheet or page).

## 8. Dialogs

- Only for **confirmation of consequential actions**: Stop run, Run backup, Restart service, Delete.
- Title = the action ("Stop this run?"), one sentence of consequence, primary action (semantic: destructive for stop/delete) + Cancel.
- Combine with re-authentication where required (`10A` D-S5: ≈15 min window). The server also checks an action nonce.
- `role="alertdialog"`, initial focus on the safest button (Cancel for destructive).

## 9. Context menus

- **Touch:** long-press a row → sheet of actions. **Pointer:** right-click or the ⋯ button → popover.
- Order: common → rare → destructive (separated, red, last).
- The same actions appear in the detail page's ⋯ menu. Context menus are shortcuts, never the only path.

## 10. Keyboard

| Key | Action |
|---|---|
| ⌘/Ctrl-K | Open/close launcher |
| Esc | Close top layer (launcher, sheet, dialog, inspector selection) |
| ↑ / ↓ | Move selection in the launcher; in lists with an inspector |
| ↵ | Open selection / confirm primary |
| Tab (in launcher input) | Cycle Go → Do → Capture |
| `>` / `+` as first character | Switch to Do / Capture |
| Space on a focused check | Toggle complete |

A focus ring is visible on every control. The skip link goes to main content.

## 11. Launcher

- **Modes** (never simultaneous; fixes Stage 3's fused search and capture):
  - **Go** (default): navigation targets.
  - **Do**: allowed commands only; each shows its requirement ("Needs your password", "Tailnet only" and disabled with the reason).
  - **Capture**: a task form with parsed chips; nothing is created until "Add task".
- **Search scope (Go):** projects · development tasks · repositories · reviews · agents/runs · courses & sessions · health pages · goals · system services · apps · settings pages.
- **Ranking:**
  1. exact ID match (ATL-160)
  2. items needing Jay (Attention)
  3. recently opened
  4. active over done
  5. title prefix over substring

  Max 5 per group; groups ordered by the best hit.
- **Empty query:** recents (5) + suggestions (the Now item, the top Attention item).
- **Phone:** the search icon opens a full-height sheet with the same modes as a segmented row. The + on Today opens Capture directly.
- **Security:** Do-mode commands obey the same public/tailnet and re-auth rules as their screens (server-side). The launcher never offers destructive commands (Delete, Merge, arbitrary shell).

## 12. Quick capture

- Entry: + on Today, Capture mode, launcher `+` prefix.
- One field ("Book dentist tomorrow 9am"). Parsed into chips (date/time, project, priority) that the user can tap to change. Parsing is deterministic and local.
- "Add task" commits; a toast confirms with Undo. The sheet closes and focus returns.
- Capture creates **GYST tasks** only. Development tasks are created with Do → "New development task" (GitHub issue), which is a separate command.

## 13. Safe areas

- Top bar padding-top `env(safe-area-inset-top)`; sticky elements offset by the same.
- Tab bar height `56 + env(safe-area-inset-bottom)`; side padding respects left/right insets in landscape.
- Content bottom padding = tab bar + safe-bottom + 32, so the last row is never hidden.
- Sheets add safe-bottom to their bottom padding. Toasts sit above the tab bar.
- Galaxy scene respects the insets (controls never under the notch or home indicator).

## 14. Scroll behaviour

- One scroll container per screen (the document). No nested scrolling except inside sheets, the launcher results and horizontal pill rows.
- Large title scrolls away; the top-bar title fades in (48 px threshold).
- Scroll position is restored on back navigation and tab switches.
- Galaxy: the canvas handles drag/pinch; the page scrolls only outside the scene.

## 15. Sticky elements

Top bar (always), sub-nav segmented control (on area screens), tab bar (phone).
Nothing else is sticky. Section headers don't stick (they would crowd a phone).

## 16. Loading

- Server-rendered pages arrive complete. Skeletons appear only for client-fetched regions (Galaxy data, inspector content) after 300 ms.
- Actions give immediate feedback: pressed state, then toast or inline state change. No full-screen spinners.
- Background refresh never shifts layout. Freshness text updates in place.

## 17. Stale / offline handling

| Condition | Presentation | Actions |
|---|---|---|
| Source stale (past its freshness target) | Amber "Stale" chip beside the provenance/freshness line | Allowed unless the action depends on fresh state (then disabled with reason) |
| Source failed | Crit/warn banner at the top of the section, last good data kept | Retry where the source supports it |
| App offline | Idle banner "Offline · showing saved data" | Server actions disabled with "Offline" |
| Unknown (no data ever) | "—" + "No data yet" | — |

Healthy is never assumed: missing status renders "Unknown", not green.

## 18. Motion / reduced motion

- Press 150 ms, sheets and dialogs 240 ms, ship flight 900 ms, working pulse 1.6 s.
- Reduced motion: no pulses, no ship flight (it jumps), sheets appear without translation, the chevron doesn't rotate.
- No autoplay loops except the working pulse and Galaxy's very slow ambient drift (disabled under reduced motion).

## 19. Touch behaviour

- 44 px targets (56 for Galaxy worlds). Rows are whole-row targets.
- Check circles complete **immediately** with a toast + Undo, not a confirmation dialog.
- Swipe on task rows (later): leading = complete, trailing = snooze/skip. Always also available via long-press.
- Galaxy: one-finger drag orbits, pinch zooms, tap selects, double-tap recenters. Controls (+ − ⌖) mirror gestures for accessibility.
- No hover-only information: anything shown on hover is also reachable by tap.

---

## Prototype browser review log (Stage 7)

- **Tooling:** Claude in Chrome on the owner's desktop Chrome.
- **How the prototypes were served:** as a **private artifact**
  (https://claude.ai/artifact/RkJTB44V82FpVekoWfWdA8), not from the server, so no
  port was exposed.
- **How widths were set:** the browser window can't be resized in this setup, so
  the artifact's content iframe was resized **from the parent page**. Its viewport,
  and therefore its media queries, then equals the set width.
- **How light mode was checked:** by setting the iframe element's `color-scheme`,
  which Chrome passes to `prefers-color-scheme`.
- **Tooling quirks:** wheel scrolling doesn't reach the cross-origin frame (lower
  content was inspected by panning a tall frame), and the first click after a
  frame change is sometimes consumed.

| Viewport | Screens inspected |
|---|---|
| 390 | Index, Today (all sections), Development, Galaxy (select world, preview sheet, close, Done toggle), Health, More, System |
| 430 | Today |
| 768 | Today |
| 1440 | Today, Projects, Development (inspector selection), Health, System, Launcher (Go → ↓↓ → Tab → Do) |
| ~1900 | Today |
| Light + dark | Today (390/430/1440/1900), Index (dark at all; light at 1440) |

**Issues found and fixed during review (5 republishes):**

| # | Issue | Fix |
|---|---|---|
| 1 | Rhythm check-ins about 1,200 px down on phone (violates D10) | Moved directly under Now |
| 2 | Meta "·" separator missing outside rows | Separator style made global |
| 3 | Now sub-values wrapped at 390 | Countdown moved into the label; decorative tiles hidden < 480 |
| 4 | Desktop left column indented (nested max-width + auto margin) | Columns fill their grid tracks |
| 5 | Duplicate "Today" title in the desktop top bar | Title fades in only after scrolling, as on phone |
| 6 | **Horizontal overflow at 390** on screens with segmented controls (grid `min-width:auto`) | `min-width: 0` on grid children |
| 7 | Project orbs overlapped row text (class collision with Galaxy's `.world`) | Renamed to `.tile.orb` |
| 8 | Icon-only "Open PR" button stretched to half width | Fixed 44 px square |
| 9 | Badge count differed between pages | One default count |
| 10 | Health's fifth segment ("Plan") off-screen at 390 | Tighter segment padding < 420 |
| 11 | Healthy states in loud green | Healthy = muted text + green dot |
| 12 | More tiles contradicted status (green tile, "1 issue") | Neutral tiles; status in trailing text |
| 13 | Sidebar System dot green during a problem | Dot reflects the worst state |
| 14 | Script-driven launcher navigation blocked in the sandbox | Launcher entry points are real links |
| 15 | Galaxy zoom-out showed "^" | Minus icon |
| 16 | Rhythm narrower than its desktop column | Cap removed |

**Not individually browser-reviewed** (they share the components validated above):
Project detail, Development task, Reviews, Agents, Agent detail, Nutrition, Learn,
Attention, Automations, Apps; the Today "One overdue" and "Everything normal"
scenarios; the System tailnet view. Their HTML and CSS use the same classes. They
should get the same pass in implementation review.

## Implementation risks

| Risk | Mitigation |
|---|---|
| Legacy screens (Roadmap, Monthly, forms) resist the new components | Migrate screen by screen onto the `ui-*` layer; retire legacy CSS per screen; the Stage 3 targets are the bar |
| Server-rendered app + no build step: sheets, inspector and launcher need JS | Progressive enhancement: every row is a real link; JS upgrades to sheet or inspector; nothing requires JS to reach data |
| Token rename (`--gyst-*` → vNext names) breaks existing pages | Map vNext tokens onto the existing `--gyst-*` variables first; rename later in one change |
| Galaxy prototype is DOM/CSS; production is Three.js | Port the rules, not the markup: 56 px picking radius, dimmed outer orbit, sheet preview, ship flight |
| Screens ◇ depend on Stage 5 backends | Don't ship screens before their data exists (D8); ship Attention with GYST sources first |
| Backdrop blur performance on older iPhones | Solid fallback via `@supports`; blur only on 3 sticky bars |
| iOS PWA quirks (safe areas, sheet swipe vs page scroll) | Device review on iPhone for every phase (Stage 3 couldn't emulate this) |
| Accessibility regressions in custom controls (segmented, filters, sheets) | Use native elements (links, buttons, `details`, `dialog`) and keep the prototype's ARIA patterns |
| Public/tailnet UI divergence misread as security | Server-side enforcement per `09`/`10A`; UI hiding is presentation only |
| Scope creep from prototypes | Prototypes are fictional; each production screen is its own approved task |

## Validation checklist (must be browser-validated during implementation)

1. 390, 430, 768, 1024, 1440, 1920 on every migrated screen; no horizontal scroll; 44 px targets.
2. Real iPhone PWA: safe areas, tab bar, sheet swipe/scroll interplay, keyboard with inputs, pull-to-refresh behaviour.
3. Light, dark and system switching; contrast spot checks on chips, tags and tinted banners.
4. 200% text size; reduced motion; VoiceOver pass on Today, a list with a sheet, and the launcher.
5. Keyboard: ⌘K, arrows, Esc, focus return from sheets and dialogs.
6. Today scenarios with real data: none, overdue only, reviews, stale Health, system problem.
7. Health invariants on screen: missing "—", recorded zero, separate Watch/phone steps, burned composition.
8. Public vs tailnet: privileged routes absent (404) on the public listener; UI shows the tailnet notice.
9. Re-auth flows (15 min window) and action nonces on every consequential action.
10. Galaxy on iPhone: 56 px picking, pinch/drag, completed-worlds toggle, performance with 15+ worlds.
11. Offline/stale banners with the network disabled.
12. No console errors; no layout shift on background refresh.
