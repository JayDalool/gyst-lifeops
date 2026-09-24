# TRAIN1 — Hevy training and mobile Galaxy

Status: IN PROGRESS. Authorized by Jay on 2026-09-24, including UI changes and deployment.

## Plan and acceptance

1. Read-only Hevy import: exercise catalog, all paginated routines and workouts;
   normalized provider-neutral training cache. Atomic replacement, preserve last
   good data on errors, server-only API key, bounded requests, no redirected keys.
2. Simple Training page: routines, recent sessions, exercise/set details and a
   front/back muscle map. Count primary work sets; exclude warmups, distinguish
   secondary involvement and unknown mappings. Plans are one pass through routines,
   not a guessed weekly schedule. No calorie or recovery estimates.
3. Preserve the existing Galaxy; fit its camera to portrait screens and simplify
   mobile overlays. Keep project navigation, preview, gestures and fallback list.
4. Run Go format/vet/build/unit/race and browser checks using isolated data.
   Commit/push each logical change, verified production backup before deployment.

Live Hevy validation is pending a server-side HEVY_API_KEY. No key is currently
configured. API reference: https://api.hevyapp.com/docs/ . Missing data stays missing.

Next action: implement and test the provider/cache.

Backend implemented: read-only paginated catalog/routines/workouts, nullable set
normalization, atomic generic snapshot, refresh scheduler and server environment
wiring. Local gofmt/vet/build/unit/race and Node checks passed. Provider tests
cover pagination, rep ranges, absent versus zero, API failures and redirect/key
protection. Service test proves failed refresh retains the successful snapshot.
No production changes yet. Self-reviewed; no independent review.

Backend commit `c5e32bc`; [CI green](https://github.com/JayDalool/gyst-lifeops/actions/runs/36070399889).
Training UI implemented: routine selection, rolling 7/30-day logged set counts,
front/back primary-muscle map, secondary count table, expandable routines and
30 most recent sessions, authenticated same-origin POST refresh, cached/empty/
stale states. Manual check-ins preserved. Docker now includes trusted CA roots
for outbound HTTPS. Browser validation: Chromium and WebKit at 320, 390, 844,
1440px; filters/details and 200% text passed with isolated synthetic fixtures.
No JavaScript errors. Physical iPhone and live Hevy account remain unverified.
Next action: finish mobile Galaxy framing and overlays.

Training UI commit `39ec35d`; [CI green](https://github.com/JayDalool/gyst-lifeops/actions/runs/36071378784).
Galaxy measured before changes: at 390×844 the hub overlay covered 164.875px of
the 730px canvas. Header now outside canvas; camera fits all project bounds to
the narrower viewport dimension, responsive zoom limits and fog, labels avoid
overlaps/edges, phone detail panel flows below the canvas. Scene retained.
Chromium/WebKit checks passed at 320/390/844/1440px: no horizontal overflow,
project list, preview/open/close and reset. Both engines rendered the 3D scene.
Node gesture and camera math checks passed; physical iPhone review remains open.
Next action: deploy the tested changes after a verified production backup.
