# UI-1 — GYST app shell and five destinations

- **Status:** `IN PROGRESS` — implemented; final verification/commit pending.
- **Approval:** Jay reviewed UI-0 and said “i like it, go ahead.”
- **Implementation commit(s):** Pending.
- **CI:** Pending push.
- **Review:** Chromium/WebKit browser checks passed, including final responsive refinements. Physical iPhone review remains pending.

## Changes

- GYST top bar, real XP, SVG Search/Add controls, skip link and five navigation
  destinations: Today, Projects, Health, Courses, More.
- GET-only More page groups existing secondary tools; direct Galaxy link on
  Projects and preserved Galaxy/search/form actions.
- Central page-to-tab ownership includes nested detail/form pages.
- Light default, explicit persisted dark preference, guarded storage, one theme
  owner and chart redraw event. Existing appearance/motion controls remain.
- Mobile safe areas, content clearance, desktop top navigation, Galaxy stage
  compatibility and command palette focus/keyboard handling.
- Static-only service-worker cache; neutral offline navigation; no private-page,
  auth, HTML or API fallback. Old shell caches retired. No eager Galaxy download.
- GYST manifest identity; `/today` start URL and push handlers unchanged.

## Verification

Go format/vet/build/unit/race suite passed. `node scripts/check-ui-shell.cjs`
passed private-route exclusion, offline responses, auth-header and POST bypass,
static fallback, no-store/HTML/redirect exclusion, old cache retirement and theme
persistence/storage denial. This test is also added to the existing CI workflow.
Route tests verify five links/one selected state, retained actions, More method
restrictions and unauthenticated denial. Existing feature assertions are retained,
with secondary links verified through More instead of every page's old popup.

Browser verification uses only `/tmp/gyst-ui1-review/preview.db` with fixture
credentials. During verification, corrected Galaxy's two-pixel border overlap
and made palette Escape/focus return explicit. Shell clearance follows measured header/nav heights for enlarged text and landscape; labels wrap within each tab. Galaxy controls keep dark-surface colors even with the light shell. Chromium and WebKit passed at 375, 390, 430, 768 and 1280px across Today, Projects, Health, Courses, More, Settings and Galaxy, with five navigation destinations, correct selected state, 44px targets, no horizontal overflow and no JavaScript page errors. Palette keyboard/focus behavior and persisted themes passed. Chromium verified sign-out followed by neutral offline navigation and online authentication redirect. Enlarged text (200%), landscape Galaxy sizing and a simulated 34px bottom safe area passed. These are browser emulations, not a physical iPhone/Home Screen review.
No production database access, schema/business-logic changes or deployment.

## Next action

Jay: review UI-1 on iPhone before approving UI-2 Today adoption.
