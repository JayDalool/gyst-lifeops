# UI-PLAN — Reference-based design system and screen inventory

- **Status:** `PENDING` — specification prepared; awaiting Jay's approval.
- **Implementation commit(s):** `0145c9c` (specifications and reference boards).
- **Review status:** Source and both supplied boards inspected. No browser/iPhone review.
- **CI:** [Green for 0145c9c](https://github.com/JayDalool/gyst-lifeops/actions/runs/34668589837). Local gofmt, vet, build, unit and race checks passed in Go 1.25 Docker; specification links and reference image integrity checked.

## Scope

Create `docs/UI_SYSTEM.md` and `docs/UI_SCREEN_MAP.md`, preserve both source
boards, map real routes/data, and propose exact UI-0/UI-1 changes. No runtime,
business logic, schema, deployment or production data changes. Final user planning
instruction takes precedence over the earlier A2-start instruction for this task.

## Acceptance

- [x] Inspect references, templates, CSS, JS, routes, Galaxy, Health, Projects,
  Courses, Today and PWA navigation.
- [x] Document visual tokens, reusable contracts, states, accessibility and safe areas.
- [x] Distinguish existing data from missing backend capabilities.
- [x] Specify concrete files and approval boundaries for UI-0 and UI-1.
- [ ] Jay approves the plan.

## Blockers

UI implementation awaits the explicit design approval requested by Jay. A2 is
separate; contradictory handoff/operator status is recorded in UI_SCREEN_MAP.md.

## Next action

Jay: review the two specifications and approve UI-0's foundation scope.
