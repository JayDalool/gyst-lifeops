# REV1 — Independent review of the B1 range

- **ID:** REV1
- **Title:** Code review of `3083fb9..a1dc3da`
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):** None — a review produces findings, not code. The
  findings were implemented in `29adc43`
  ([B1-FIX](B1-FIX-review-fixes.md)), which is the Git evidence that this review
  happened and was acted on.
- **Review status:** N/A — this *is* the review. It was performed by the same
  agent that wrote the code, which is a real limitation: **an independent review
  by a different agent or by a human remains UNKNOWN.**

## Verdict

**PASS WITH NOTES** — no blocking findings. Three non-blocking findings, all
since fixed:

| ID | Finding | Fixed in |
|---|---|---|
| H1 | A caller-supplied `_pragma` in a `file:` URI silently defeats enforcement | `29adc43` |
| M1 | `:memory:` gives each pooled connection its own empty database | `29adc43` |
| M2 | Behavioural FK test would deadlock if a pool maximum were set | `29adc43` |
| L1 | Test asserts on a path relative to the package working directory | **open** |

## What the review verified rather than assumed

Each finding was reproduced against the driver before being reported, not
inferred from reading. H1 was confirmed by measuring which spellings win
(`off`, `false`, `OFF` beat the appended `foreign_keys(1)`; `0` loses on
collation). M1 was confirmed by creating a table on one connection and failing
to read it from another.

The review also **corrected an error in the earlier architecture review**: it
claimed `/health/plan` had no CSRF protection. It already did, at
`internal/handlers/health_plan.go:136-149`, with a test. The claim came from
reading part of the file rather than all of it.

## Acceptance criteria

- [x] Every finding reproduced before being reported
- [x] Schema-wide effect of newly-real foreign keys analysed (10 CASCADE, 13 SET NULL)
- [x] XP reward integrity specifically checked — benign, because ids are AUTOINCREMENT
- [x] Findings triaged as blocking or not, with reasons
- [ ] **Reviewed by a second party — NOT DONE**

## Blockers

None.

## Next action

None for H1/M1/M2. **L1 is open and unscheduled**: a failing run of
`TestOpenAcceptsCallerSuppliedFileURI` can leave a stray `internal/db/file:`
entry that makes later runs fail until it is removed by hand.
