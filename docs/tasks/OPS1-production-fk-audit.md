# OPS1 — Production foreign-key audit

- **ID:** OPS1
- **Title:** `PRAGMA foreign_key_check` against the real database
- **Status:** `PENDING` (operator)
- **Owner:** Jay
- **Implementation commit(s):** N/A — an operational task, not a code change.
- **Review status:** N/A. **Result UNKNOWN** — the check has not been reported
  as run, and nothing in this repository can establish the state of the
  production database.

## Why this gates deployment

Before [B1](B1-sqlite-pragma-correctness.md), foreign keys were unenforced on
most pooled connections, so the live database may already contain orphaned rows
that were accepted at the time. B1 does not create that problem and does not
clean it up — it makes such rows visible.

The database must be certified clean before the B1 code is deployed against it.

## Procedure

```sh
# 1. Back up first, while the container is still running.
docker compose exec gyst-lifeops /app/lifeops backup

# 2. Then stop the app, so the check cannot miss or misreport rows.
docker compose stop gyst-lifeops

# 3. Run the audit on the host - the image has no sqlite3 binary.
sqlite3 ./data/lifeops.db "PRAGMA foreign_key_check;"
```

Step 1 uses [B2](B2-backup-tooling.md) and writes to `data/backups/`, inside
the mounted volume. It reports the size, table count and applied-migration
count, and verifies the copy with `integrity_check` before claiming success.

**Do not back up with `cp data/lifeops.db`.** The database runs in WAL mode, so
committed data sits in `data/lifeops.db-wal` until a checkpoint moves it. A
measured example: the main file was 4,096 bytes while its `-wal` held
1,285,472. Copying the main file alone would have produced a 4 KB near-empty
database, with nothing to indicate the backup was worthless.

- **Zero rows** → certified; B1 is clear to deploy. Record the date here.
- **Any rows** → **STOP.** Each row is `table | rowid | parent_table | fk_index`.
  Record them here and report. **Do not delete, repair or migrate anything** —
  deciding what an orphaned row should become is a product decision, not a
  cleanup script.

## Related operator steps

Also required before the auth commits can be deployed:

1. `go run ./cmd/passwordhash` → set `GYST_PASSWORD_HASH` in `.env`.
   **Verify login works before closing the session you deploy from.**
   `GYST_AUTH_DISABLED=1` is the escape hatch.
2. Enable Tailscale HTTPS — unblocks [A2](A2-trusted-tls.md) and therefore M3.

## What is now live that was not before

Enforcement is real, so the schema's referential actions now execute: 10
`ON DELETE CASCADE` and 13 `ON DELETE SET NULL`. Deleting a project now also
removes its `project_items`, `roadmap_project_meta`, `roadmap_milestones` and
`roadmap_resources` rows.

`achievements.source_*_id` are `SET NULL`, which was checked specifically
because XP integrity is a stated guardrail: it is **benign**, because `tasks`,
`projects` and `project_items` all use `INTEGER PRIMARY KEY AUTOINCREMENT`, so
SQLite never reuses ids and no duplicate reward can be granted.

## Acceptance criteria

- [ ] App stopped, backup taken
- [ ] `PRAGMA foreign_key_check` run on the host
- [ ] Result recorded here with the date
- [ ] If non-empty: rows recorded and reported, nothing repaired

## Blockers

None — waiting on the operator.

## Next action

Jay: run the audit and report the output.
