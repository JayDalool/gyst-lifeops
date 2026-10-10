# Restore runbook (GYST-VN-013, Issue #112)

**Operator: Jay only** (`risk:not-agent-eligible`). An agent prepared this
document and rehearsed the commands on a fictional scratch database built from
`seeds/example.sql`. No agent has run any of it against `data/` or production.

Two procedures:

- **Part A, the drill.** Restores a backup into a scratch directory only and
  checks it. It never targets `data/lifeops.db`. Run it once now, then monthly.
- **Part B, a real restore.** Replaces the live database. Use it only when the
  live database is lost or damaged, or Jay decides to go back in time.

**Rules that apply to both** (`docs/RULES.md` §3.1):

- Never back up or restore with `cp` of the *live* `data/lifeops.db`. In WAL
  mode the real data can sit in the `-wal` file, and the copy looks fine.
- A file made by `lifeops backup` is different: it is one self-contained
  `journal_mode=delete` file with no `-wal` companion (`internal/db/backup.go`,
  `VACUUM INTO`). Copying *that* file is safe.
- Nothing is deleted in a real restore. Files are moved aside.
- Do not auto-repair foreign-key problems. Record them and stop.
- Put no real values (titles, health numbers, hostnames, keys) in a PR, an
  Issue or the record below. Counts and pass/fail only.

## Before you start

Run everything from the repo root on the host (`/srv/server/repos/gyst-lifeops`).
Once VN-009 makes `data/` mode `700` and root-owned, commands that read it need
`sudo`; they are written with `sudo`. Check the one tool this needs:

```bash
command -v sqlite3 && docker compose ps gyst-lifeops
```

Both must succeed. (`sqlite3` is already relied on by OPS1; nothing new to install.)

Two helpers used by both parts. Paste them into your shell first:

```bash
# counts FILE: one "table|rows" line for every table in FILE (counts only, no values).
# $SQ is the sqlite3 command to use; each part sets it.
counts() { $SQ "$1" "SELECT 'SELECT ''' || name || ''', count(*) FROM \"' || name || '\";' FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite\_%' ESCAPE '\' ORDER BY name;" | $SQ "$1"; }

# The image compose builds for the app. The drill and the migration check use it.
IMG=gyst-lifeops-gyst-lifeops:latest
```

## Part A. The drill (scratch path only)

Everything here happens in `/srv/server/artifacts/gyst/restore-drill`. **No
command in Part A names `data/lifeops.db`.** The only thing read from `data/` is
a finished backup file, copied with `sudo`.

### A1. Start the clock and make the scratch directory

```bash
S=/srv/server/artifacts/gyst/restore-drill
SQ="sqlite3 -readonly"
T0=$(date -u +%s); echo "start $(date -u -d @$T0 +%FT%TZ)"
sudo install -d -m 700 -o "$(id -u)" -g "$(id -g)" "$S" "$S/data"
```

### A2. Take a fresh backup and copy it into the scratch directory

```bash
docker compose exec -T gyst-lifeops /app/lifeops backup
BK=$(sudo sh -c 'ls -1t data/backups/lifeops-*.db | head -1'); echo "$BK"
sudo install -m 600 -o "$(id -u)" -g "$(id -g)" "$BK" "$S/source.db"
cp "$S/source.db" "$S/data/lifeops.db"
```

`lifeops backup` prints `verified: integrity_check ok`. To drill an older backup
instead, set `BK` to that file by hand and note its age in the record.

### A3. Check the copy

```bash
$SQ "$S/data/lifeops.db" 'PRAGMA integrity_check;'
$SQ "$S/data/lifeops.db" 'PRAGMA foreign_key_check;' | wc -l
$SQ "$S/data/lifeops.db" 'SELECT count(*) FROM schema_migrations;'
counts "$S/source.db" > "$S/counts-source.txt"
counts "$S/data/lifeops.db" > "$S/counts-copy.txt"
diff "$S/counts-source.txt" "$S/counts-copy.txt" && echo "counts identical"
wc -l < "$S/counts-copy.txt"
```

| Check | Pass |
|---|---|
| `integrity_check` | prints exactly `ok` |
| `foreign_key_check`, line count | `0` |
| `diff` | no output, then `counts identical` |
| last line | number of tables compared (write it in the record) |

If foreign-key rows appear, do not repair anything. Record "fail" and the number
of rows only, and tell Claude.

### A4. Boot check (no network, no keys, no ports)

```bash
docker run --rm -d --name gyst-restore-drill --network none -v "$S/data:/app/data" "$IMG"
sleep 5
docker logs gyst-restore-drill 2>&1 | grep -E 'Application timezone|running at'
docker stop gyst-restore-drill
```

Pass: both lines print (an `Application timezone: ...` line and a
`LifeOps Tracker running at ...` line). A `SECURITY: GYST_PASSWORD_HASH is not
set` line is expected: this container has no environment on purpose, so it cannot
send pushes, call Hevy or an AI provider, or be reached from outside. `--rm`
removes the container when it stops. Do not add `-p`, `--env-file` or `-e`.

### A5. Stop the clock, record, clean up

```bash
T1=$(date -u +%s); echo "end $(date -u -d @$T1 +%FT%TZ), minutes taken: $(( (T1 - T0 + 59) / 60 ))"
```

Fill in the record (Part D). Then remove the scratch directory, and only that
directory:

```bash
sudo rm -r "$S"
```

The boot check wrote files into `$S/data` as root, which is why this needs `sudo`.

## Part B. A real restore (only when needed)

This replaces the live database. The app is down during it. Do not skip steps.

### B1. Choose and check the backup

```bash
SQ="sudo sqlite3 -readonly"
sudo sh -c 'ls -1t data/backups/lifeops-*.db | head -5'
BK=data/backups/lifeops-<UTC-STAMP>.db     # the one you chose, from the list above
$SQ "$BK" 'PRAGMA integrity_check;'
$SQ "$BK" 'PRAGMA foreign_key_check;' | wc -l
```

Expect `ok` and `0`.

**An older backup on newer code is fine; a newer backup on older code is not.**
On start the app runs any migrations the backup is missing
(`internal/app/app.go`). It cannot cope with a backup that has migrations the
deployed code lacks. This must print nothing:

```bash
comm -23 <($SQ "$BK" 'SELECT filename FROM schema_migrations;' | sort) \
         <(docker run --rm --network none "$IMG" ls /app/migrations | sort)
```

If it prints filenames, do not restore this backup onto this code.

### B2. Take a fresh backup of what is there now (if the database still opens)

```bash
docker compose exec -T gyst-lifeops /app/lifeops backup
```

If this fails because the database is damaged, carry on: the files are moved
aside in B4, not deleted.

### B3. Stop only the app

```bash
docker compose stop gyst-lifeops
```

Not `down`, and not Caddy or any other service. `docker compose ps gyst-lifeops`
should show it exited.

### B4. Move the live files aside (never delete)

```bash
T=$(date -u +%Y%m%dT%H%M%SZ)
OLD=data/backups/pre-restore-$T
OWN=$(sudo stat -c '%u:%g' data/lifeops.db 2>/dev/null || echo 0:0)
MODE=$(sudo stat -c '%a' data/lifeops.db 2>/dev/null || echo 600)
echo "$OLD owner $OWN mode $MODE"
sudo install -d -m 700 "$OLD"
for f in lifeops.db lifeops.db-wal lifeops.db-shm; do
  if sudo test -e "data/$f"; then sudo mv "data/$f" "$OLD/"; fi
done
sudo ls "$OLD"
sudo ls data
```

All three files must leave `data/`: `sudo ls data` must show no `lifeops.db*`.
A stale `-wal` or `-shm` left beside a restored file can be replayed onto it and
corrupt it.

### B5. Install the backup with the original owner and mode

```bash
sudo install -m "$MODE" -o "${OWN%:*}" -g "${OWN#*:}" "$BK" data/lifeops.db
sudo stat -c '%u:%g %a' data/lifeops.db
```

The output must match the owner and mode printed in B4. (If there was no live
file, the defaults used are `0:0` and `600`.)

### B6. Check before you start

```bash
$SQ data/lifeops.db 'PRAGMA integrity_check;'
$SQ data/lifeops.db 'PRAGMA foreign_key_check;' | wc -l
counts "$BK" > /tmp/counts-bk.txt
counts data/lifeops.db > /tmp/counts-live.txt
diff /tmp/counts-bk.txt /tmp/counts-live.txt && echo "counts identical"
rm /tmp/counts-bk.txt /tmp/counts-live.txt
```

Expect `ok`, `0`, `counts identical`. The count files hold table names and
numbers only. If anything differs, go to Part C.

### B7. Start and verify

```bash
docker compose start gyst-lifeops
sleep 5
docker compose ps gyst-lifeops
docker compose logs --tail 20 gyst-lifeops | grep -E 'Application timezone|running at'
curl -ksS -o /dev/null -w '%{http_code}\n' https://<GYST_TLS_HOSTNAME>:3443/healthz
```

Pass: the container is `running`, both log lines print, and `/healthz` prints
`200`. Then sign in and open Today in the browser. Finally take one more backup,
which also proves the restored database is healthy:

```bash
docker compose exec -T gyst-lifeops /app/lifeops backup
```

It must print `verified: integrity_check ok`. The files in `$OLD` stay where they
are until Jay removes them by hand, later.

## Part C. Rollback of a real restore

Use this if B6 or B7 fails.

```bash
docker compose stop gyst-lifeops
sudo install -d -m 700 "data/backups/failed-restore-$T"
for f in lifeops.db lifeops.db-wal lifeops.db-shm; do
  if sudo test -e "data/$f"; then sudo mv "data/$f" "data/backups/failed-restore-$T/"; fi
done
for f in lifeops.db lifeops.db-wal lifeops.db-shm; do
  if sudo test -e "$OLD/$f"; then sudo mv "$OLD/$f" "data/$f"; fi
done
docker compose start gyst-lifeops
```

`mv` keeps the original owner and mode. Verify as in B7. If the shell was
closed, first set `T` to the stamp in the `pre-restore-<UTC>` directory name and
`OLD=data/backups/pre-restore-$T`.

## Part D. Verification record

Copy this block into a comment on Issue #112 (or the Records section below, in a
small PR). Pass/fail, times and counts only. Per-table counts stay in the
scratch directory and are deleted with it. Nothing else goes in.

```text
Restore drill record
Date (UTC):
Backup age at drill (hours):
integrity_check = ok:            pass / fail
foreign_key_check rows = 0:      pass / fail
schema_migrations rows:
Tables compared / identical:     N / yes / no
Boot check (timezone + running): pass / fail
Start / end (UTC):
Minutes taken:
Scratch directory removed:       yes / no
Notes (no values):
```

Repeat Part A **monthly**. It is a manual step; the MY_ACTIONS line tracks it.

## Later changes to this runbook

- After VN-076 (non-root container), the scratch `$S/data` must be owned by UID
  10001 (`sudo chown -R 10001 "$S/data"` before A4).
- After GYST-071 (healthcheck command), the A4 boot check can call
  `/app/lifeops healthcheck` instead of reading log lines.
- After GYST-069 (off-host backups), an off-host copy is decrypted on a trusted
  device first, then copied in as `$S/source.db`.

## Records

(None yet. The first drill is Issue #112.)
