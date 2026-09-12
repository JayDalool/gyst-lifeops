# B2 — Backup tooling

- **ID:** B2
- **Title:** `lifeops backup` — a consistent on-line database copy
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):** `820c704`
- **Review status:** CI — see the run for `820c704`. Human or independent-agent
  review: **UNKNOWN**. Exercised end-to-end against a live server in a scratch
  directory; **never run against the production database** by an agent.

## Why it was needed

[OPS1](OPS1-production-fk-audit.md) tells the operator to take a backup before
running `foreign_key_check` against production, and the only available method
was copying `data/lifeops.db` by hand.

**That method is unsound here, and silently so.** The database runs in WAL
mode, so committed transactions live in the `-wal` file until a checkpoint
moves them into the main file. Measured against a freshly started server:

```
lifeops.db        4,096 bytes
lifeops.db-wal    1,285,472 bytes
```

A `cp data/lifeops.db` at that moment produces a 4 KB near-empty database and
gives no sign that anything is wrong. The `lifeops backup` output for the same
database was 487 KB, 32 tables, 19 applied migrations.

## What was built

```
lifeops backup
lifeops backup -db data/lifeops.db -out backups/before-migration.db
```

- `internal/db/backup.go` — `Backup`, `DefaultBackupPath`, `BackupResult`.
- `internal/app/backup_cmd.go` — the subcommand and its flag set.
- `internal/app/cli.go` — dispatches `backup` before any server flag is defined.

Decisions, each with the reason it was made:

| Decision | Why |
|---|---|
| `VACUUM INTO`, not a file copy | Takes its own read transaction; writes one complete compacted file; writers continue throughout |
| Source opened `mode=ro` | Structural, not a promise: SQLite refuses writes, and a mistyped path cannot create an empty database |
| Destination as a bound parameter | The `INTO` clause takes an expression, so the path is never hand-quoted |
| Existing destination refused | The likeliest thing at a backup path is an earlier backup |
| Copy verified before success | `integrity_check`, table count, migration count. A copy that fails is deleted — an untrustworthy backup is worse than none |
| Default is `backups/` **beside the database** | Only `./data` is a mounted volume; a backup relative to the working directory would vanish with the container |
| A subcommand, not a second binary | The image builds only `./cmd/lifeops`, so a separate tool would not exist inside the container holding the database |

Scheduling, rotation, off-site copying and encryption are deliberately absent.
They are a cron line around this command.

## Tests

`internal/db/backup_test.go`, `internal/app/backup_cmd_test.go`. The source
databases are built from the **real** `migrations/` directory, so the copy is
verified against the production schema shape rather than a toy one.

- every row copied; table and migration counts correct
- the copy is self-contained — no `-wal`/`-shm` left beside it
- **the source is unchanged**: size, mtime and `journal_mode` identical after
- **backup succeeds under continuous concurrent writes**, and blocks no writer
- an existing destination is refused and left byte-identical
- the source cannot be its own destination
- a missing source fails and is **not created**
- a DSN source is rejected
- an unwritable destination directory leaves no partial file (skipped when the
  test runs as root, where permission bits are not enforced)
- a file that is not a database fails verification
- `DefaultBackupPath` is UTC, sortable, and beside the database

## Acceptance criteria

- [x] `VACUUM INTO`, not a file copy
- [x] Source database never written to
- [x] Existing destination refused
- [x] Copy verified with `integrity_check` before success is reported
- [x] Tests cover the concurrent-write case
- [x] gofmt, go vet, go build, go test, go test -race pass
- [ ] CI green — check the run for `820c704`
- [x] Documented so Jay can run it before OPS1

## Observed while building this, not fixed

`docker-compose.yml` sets `DB_PATH=/app/data/lifeops.db`, but nothing reads
`DB_PATH` — the path comes from the `-db` flag, which defaults to
`data/lifeops.db`. The container's working directory is `/app`, so both resolve
to the same file and the discrepancy is harmless today. It is a trap for anyone
who changes `DB_PATH` and expects it to take effect. **Not in scope for B2**;
recorded rather than silently fixed.

## Blockers

None.

## Next action

None for B2. The operator can now run
`docker compose exec gyst-lifeops /app/lifeops backup` before
[OPS1](OPS1-production-fk-audit.md).
