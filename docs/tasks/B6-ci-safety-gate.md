# B6 — CI safety gate

- **ID:** B6
- **Title:** GitHub Actions check workflow
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):** `f663f3f79d18477f3cccc3fcab1c4946af30c2e1`
- **Review status:** Verified by its own first run —
  [run 34642109757](https://github.com/JayDalool/gyst-lifeops/actions/runs/34642109757),
  all 10 steps green in 4m12s. This is the first commit with CI evidence; every
  commit after it has a run of its own.

## What it runs

`gofmt` · `go vet` · `go build` · `go test ./...` · `go test -race ./...` ·
migration-prefix uniqueness · no database file committed.

The two test passes are separate on purpose: race instrumentation changes
scheduling, so each mode can surface failures the other misses.

Two guards are repository-specific:

- **Migration prefixes must be unique.** Migrations are tracked by filename, so
  two branches claiming the same prefix leave real ambiguity about what ran.
  This is the check that would have caught the historical 017 collision before
  merge. Covers both `NNN_` and timestamped `YYYYMMDDHHMMSS_` names.
- **No database file may be tracked.** The production database holds personal
  health data. `.gitignore` covers `data/*.db`; this fails the build if one is
  committed from a path the ignore rules miss.

Go version comes from `go-version-file: go.mod`, so it cannot drift.

## Acceptance criteria

- [x] Runs on every push and pull request
- [x] Both guards negative-tested locally (they fail when they should)
- [x] First run green on GitHub, not only locally
- [x] Baseline timings recorded for judging future cold-cache runs:
      vet 62s, build 2s, test 19s, test -race 2m26s; job cap 15 min

## Blockers

None.

## Next action

None. Completed out of sequence — it is independent of
[OPS1](OPS1-production-fk-audit.md).
