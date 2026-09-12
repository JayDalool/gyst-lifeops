# UI-0 — Design tokens and reusable foundation

- **Status:** `DONE` — implementation committed and CI green; iPhone visual review remains pending.
- **Scope:** Approved by Jay in response to UI-PLAN. UI-0 only.
- **Implementation commit(s):** `99f5968`.
- **Review:** Chromium and WebKit; no real iPhone/Home Screen verification.
- **CI:** [Green for 99f5968](https://github.com/JayDalool/gyst-lifeops/actions/runs/34669485151). Local gofmt, vet, build, unit and race checks passed after final refinements.

## Implemented

- Central tokens, opt-in typography, cards/lists/buttons/forms, segmented controls,
  progress/rings, states, dialog/sheet/toast styles, safe-area values and dark variants.
- Typed presentation inputs and shared escaped Go partials; local Lucide SVG subset/license.
- Standalone labelled fixture gallery exported by an opt-in test; no production route.
- Production template loading and synchronized service-worker asset versions only.
- Tests for missing/zero/partial/stale/loading/error, bounds, escaping, accessible
  labels and production gallery isolation. Existing feature tests retained.

## Verification

Browser checks: 375/390/430/768/1280 widths, no horizontal overflow; native dialogs,
Escape/focus return, sheet dismissal, period selection, dark theme, 200% text,
reduced motion; no page errors in Chromium or WebKit. Screenshots inspected.
Contrast pairing refined for muted text on inset controls. No production data,
deployment, schema, business logic or shell adoption changes.

The browser connector failed because Chrome was unavailable; verification used
an isolated Playwright 1.58 container instead. An initial browser-script export
failed due to root-owned artifact directory; the script was mounted read-only
from /tmp and the browser checks passed.

## Review artifact

Export: `GYST_UI_GALLERY_OUT=/tmp/gyst-ui0-review go test ./internal/handlers -run TestExportUIGallery -count=1`.
Output: standalone `index.html` with embedded assets. Generated HTML/screenshots
are review artifacts, not production pages or committed mock data.

## Next action

Jay: inspect the UI-0 gallery on iPhone before approving UI-1 shell adoption.
