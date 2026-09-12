# B1 — SQLite PRAGMA correctness

- **ID:** B1
- **Title:** Apply connection-scoped PRAGMAs to every pooled connection
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):** `fb4f1ebd7ce44eb784e2c0a9f1898880db3e3cd0`
- **Review status:** Reviewed; findings were raised and fixed in a follow-up — see
  [B1-FIX](B1-FIX-review-fixes.md), whose existence is the Git evidence that review
  happened. **No CI run:** this commit predates the CI gate (`f663f3f`), so no
  automated verification of it exists or can be produced retroactively.

## Problem

`db.Open` ran `PRAGMA foreign_keys` and `PRAGMA busy_timeout` once against the
`*sql.DB`. That is a connection pool: the statement configured whichever
connection served it and left every later one on SQLite's defaults.

Measured on a fresh pool of four connections before the fix:

```
connection 0: foreign_keys=1 busy_timeout=5000
connection 1: foreign_keys=0 busy_timeout=0
connection 2: foreign_keys=0 busy_timeout=0
connection 3: foreign_keys=0 busy_timeout=0
```

Behaviourally, three of four connections accepted an insert referencing a
non-existent parent row. Foreign keys were advisory on most requests.

## Acceptance criteria

- [x] Every pooled connection reports `foreign_keys=1` and `busy_timeout=5000`
- [x] Every pooled connection behaviourally rejects an orphan insert
- [x] `journal_mode=wal` still holds (persistent file property, set once)
- [x] Filesystem paths containing spaces, `?`, `#`, `%` and Unicode open correctly
- [x] Test fails against the old implementation and passes against the fix
- [x] gofmt, go vet, go build, go test, go test -race pass locally

## Blockers

None.

## Next action

None — superseded by [B1-FIX](B1-FIX-review-fixes.md). Note the deployment gate
in [OPS1](OPS1-production-fk-audit.md): enforcement is now real, so pre-existing
orphaned rows in the production database become visible.
