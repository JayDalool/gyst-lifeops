# B1-FIX — B1 review fixes

- **ID:** B1-FIX
- **Title:** Seeded integrity audit, file: URI support, and review findings H1/M1/M2
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):**
  - `a1dc3da9600c9c36789db5b10226880eca9cff62` — seeded integrity audit, file: URI support
  - `29adc435d63912eeea8ce5aaf3873440f9ffd469` — findings H1, M1, M2
- **Review status:** Two rounds of review feedback were acted on; these commits
  are that record. An independent review by another agent is **UNKNOWN** — no
  Git or CI artifact records one. **No CI run:** both predate the CI gate
  (`f663f3f`).

## What was fixed

**Round one (`a1dc3da`)**

- The foreign-key integrity audit ran against an empty database, so
  `PRAGMA foreign_key_check` had nothing to inspect. It now loads
  `seeds/example.sql` through the production `SeedIfEmpty` path and asserts the
  seed landed before checking, so it cannot pass vacuously.
- `Open` claimed to support caller-supplied `file:` URIs but still ran
  `filepath.Dir`/`MkdirAll` over them, creating a literal `file:` directory.

**Round two (`29adc43`)**

- **H1** — a caller-supplied `_pragma` in a `file:` URI could silently defeat
  enforcement. The driver executes every `_pragma` in lexicographic order and
  the last wins, so `foreign_keys(off)` sorted after the appended
  `foreign_keys(1)`. Such DSNs are now refused.
- **M1** — `:memory:` was accepted but unusable: each pooled connection gets its
  own private empty database. Now refused, with detection mirroring SQLite's own
  case sensitivity (`:MEMORY:` is an ordinary filename and still works).
- **M2** — the behavioural FK test held four connections then issued a fifth
  query, which would deadlock if a pool maximum were ever set.

## Acceptance criteria

- [x] Integrity audit is seeded and asserts the seed loaded
- [x] `file:` URIs bypass filesystem directory creation; no stray `file:` directory
- [x] DSNs setting `foreign_keys` or `busy_timeout` are refused
- [x] In-memory databases refused; `:MEMORY:` still opens as a file
- [x] FK test releases connections before its follow-up query
- [x] H1 and M1 guarantees negative-tested (removing each makes a test fail)
- [x] gofmt, go vet, go build, go test, go test -race pass locally

## Blockers

None.

## Next action

None. One non-blocking finding remains open and unscheduled: **L1** —
`TestOpenAcceptsCallerSuppliedFileURI` asserts on `os.Stat("file:")` relative to
the package working directory, so a failing run can leave a stray entry that
makes later runs fail until removed by hand.
