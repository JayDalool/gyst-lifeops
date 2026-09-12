# B2 — Backup tooling

- **ID:** B2
- **Title:** `lifeops backup` — a consistent on-line database copy
- **Status:** `NOT STARTED`
- **Owner:** Claude (next agent task)
- **Implementation commit(s):** none
- **Review status:** n/a

## Why this is next

It is the only unblocked implementable item, and it gates the operator work.
[OPS1](OPS1-production-fk-audit.md) says "take a backup first", and today that
means copying a file by hand while hoping the app is not mid-write. Every other
open item — A1, A2, M3 — waits on Jay.

Copying `data/lifeops.db` with `cp` while the server runs can produce a torn
copy: WAL mode means committed data lives in `-wal` until a checkpoint, so the
main file alone may be stale, inconsistent, or both.

## Scope

A `backup` subcommand alongside the existing `cmd/` binaries, using SQLite's
`VACUUM INTO`. `VACUUM INTO` takes its own read transaction, so it produces a
single consistent file without stopping the server and without touching the
source database.

- New `cmd/lifeops-backup/` (or a `backup` argument to `cmd/lifeops`, chosen at
  implementation time and stated in the commit message).
- Destination path from an argument; default to a timestamped file under a
  `backups/` directory.
- **Read-only with respect to the source.** No checkpoint forced, no PRAGMA
  changed, nothing written to `data/lifeops.db`. `VACUUM INTO` does not require
  it and the rule in [`../RULES.md` §3.1](../RULES.md) forbids it.
- Refuse to overwrite an existing destination rather than clobbering a backup.
- Verify the result before reporting success: open the copy and run
  `PRAGMA integrity_check`, and report the row count of one known table.
- Print the destination path and byte size on success; exit non-zero on failure
  with the SQLite error intact.

## Out of scope

Scheduling, rotation, retention, off-site copying, encryption. Those are
operator concerns and can be a cron line around this command. Do not build a
daemon.

## Tests

`internal/db` or a `cmd` test, against a temporary database from `t.TempDir()`:

- a populated source produces a copy whose rows match;
- the copy passes `PRAGMA integrity_check`;
- concurrent writes to the source during the backup do not corrupt the copy and
  do not fail the backup;
- an existing destination is refused, and the existing file is left untouched;
- an unwritable destination directory returns an error, not a partial file.

**No test may point at `data/lifeops.db`.**

## Acceptance criteria

- [ ] `VACUUM INTO`, not a file copy
- [ ] Source database never written to
- [ ] Existing destination refused
- [ ] Copy verified with `integrity_check` before success is reported
- [ ] Tests cover the concurrent-write case
- [ ] gofmt, go vet, go build, go test, go test -race pass
- [ ] CI green
- [ ] Documented in the handoff so Jay can run it before OPS1

## Blockers

None.

## Next action

Implement it. One commit.
