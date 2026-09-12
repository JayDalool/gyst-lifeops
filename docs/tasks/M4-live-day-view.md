# M4 — Live Health Day view

- **ID:** M4
- **Title:** Replace Health Day-view demo values with real observations
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):** `60be715ebc01234080adafdbdee00dbc003d8e32`
- **Review status:** CI result **PENDING** at the time of writing — the run was
  triggered by the push but had not completed. Human or independent-agent
  review: **UNKNOWN**. **Not verified in a real browser** — no artifact records
  the page being loaded.

## Scope as built

The Day view reads `Service.HealthToday()` directly rather than through an HTTP
round-trip, and renders real observations through a view model that does all
formatting and absent-versus-zero handling in Go.

Three rules the view model enforces:

1. **A metric that was not observed is absent, not zero.** NULL and 0 mean
   different things — "MyFitnessPal has not synced" versus "you ate nothing" —
   and collapsing them at the last step throws away the distinction the schema
   exists to keep.
2. **Nothing is projected.** No projection is implemented, so the demo markup's
   "projected total burn" and the "estimated energy balance" derived from it are
   removed rather than reproduced with real-looking arithmetic.
3. **Food budget and observed energy balance stay separate.**

Watch and phone steps render separately and are never summed.

## Acceptance criteria

- [x] Day view reads real data via the service layer
- [x] Absent metrics render as "No data"; an observed zero renders as 0
- [x] Food remaining is not synthesised when nutrition has not been observed
- [x] No projection rendered; a test bans the old claim labels
- [x] Steps never summed
- [x] Empty state, missing-cycle state and stale-data state all render
- [x] Existing Health goals/tasks content preserved
- [x] gofmt, go vet, go build, go test, go test -race pass locally
- [x] Committed
- [ ] **CI verified — PENDING**
- [ ] **Loaded in a real browser — NOT DONE**

## Bug found while testing

Reaching into the macro slice by index broke the page outright on a zero-valued
view: a template `index out of range` aborts execution and silently drops
everything after it, including the existing health goals and tasks. Calories is
now an explicit field. Worth remembering — the failure mode is a partial page,
not an error.

## Blockers

None.

## Next action

Confirm the CI run for `60be715` is green. The page renders empty states until
M3 ingestion exists, so there is nothing to see on it yet beyond those states.
