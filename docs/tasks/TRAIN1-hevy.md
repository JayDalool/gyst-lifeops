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
