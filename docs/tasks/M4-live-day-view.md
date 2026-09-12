# M4 — Live Health Day view

- **ID:** M4
- **Title:** Replace Health Day-view demo values with real observations
- **Status:** `IN PROGRESS` — **implemented and locally green, but NOT COMMITTED**
- **Owner:** Claude
- **Implementation commit(s):** **None.** The work exists only in a working-tree
  stash on the session machine. Git does not prove any of it, and it will be
  lost if that machine is reclaimed.
- **Review status:** `UNKNOWN` — no commit, therefore no CI run and nothing to review.

## Scope as built (uncommitted)

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
- [ ] **Committed — NO**
- [ ] **CI verified — NO**

## Blockers

None technical. It is uncommitted because the current instruction is a
documentation-only commit.

## Next action

Decide whether to commit it. Until then this task is not DONE and must not be
described as shipped. Recovery: `git stash list` on the session machine —
`M4 day view: complete and green, awaiting decision`.
