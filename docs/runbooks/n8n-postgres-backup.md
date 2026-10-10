# n8n Postgres backup (GYST-VN-095, Issue #108)

**Operator: Jay only** (`risk:not-agent-eligible`). The agent prepared
`scripts/ops/n8n-postgres-backup.sh` and this runbook. The agent never ran
docker, systemctl, pg_dump or touched n8n, its containers, env files or any
backup. Everything below uses placeholders: `<n8n pg container>`,
`<backup dir>`, `<push URL file>`, `<repo path>`, `<same major>`. Never paste a
real URL, token or key into the repo, a PR or an Issue comment.

**Sensitive:** a dump holds n8n credentials (encrypted with n8n's encryption
key), user records and execution data. Treat each dump like a secret.

## What it does

`n8n-postgres-backup.sh <container> <backup dir> [<push URL file>]`

- Runs `pg_dump -Fc` inside the container with the container's own
  `POSTGRES_USER` and `POSTGRES_DB`. No credential is read on the host.
- Writes `n8n-<UTC>.dump.partial`, checks it with `pg_restore --list`, then
  renames it to `n8n-<UTC>.dump`. Files `0600`, directory `0700`.
- Keeps the newest 7 `n8n-*.dump` in that directory; nothing else is deleted.
- Writes `last_run.json` (0600): UTC time, file, size, result, step.
- On any failure: non-zero exit, `.partial` removed, `status=down` pushed.
  On success: `status=up`. The push URL goes to curl on stdin (`-K -`), never
  argv.
- Takes an exclusive lock (`<backup dir>/.lock`); a second concurrent run
  exits 1 without touching the first run's files.
- Refuses a relative directory or one inside a git repository (exit 2).

Off-host copy is not part of this task: it stays with VN-012.

## Steps for Jay

1. Find the container name (you run it): `docker ps` and pick the n8n Postgres
   container. Call it `<n8n pg container>`.
2. Choose `<backup dir>`: an absolute path outside every git repo, on a disk
   with room for 7 dumps. The script creates it as `0700`.
3. Optional push monitor: create a Push monitor in Kuma (B2 in
   `docs/runbooks/uptime-kuma-monitors.md`, heartbeat 26 h). Store its URL as
   one line in a `0600` file outside the repo:
   ```
   install -m 600 /dev/null <push URL file>
   # paste the URL with your editor; confirm: stat -c %a <push URL file>  -> 600
   ```
4. Manual first run:
   ```
   bash <repo path>/scripts/ops/n8n-postgres-backup.sh <n8n pg container> <backup dir> <push URL file>
   ```
   Expect `dump ok: n8n-<UTC>.dump (<bytes> bytes)` and exit 0.
   Check: `stat -c '%a %n' <backup dir> <backup dir>/*` -> `700` for the
   directory and `600` for every file; `cat <backup dir>/last_run.json` shows
   `"result":"ok"`.
5. Install a reviewed copy of the script, from a reviewed master checkout
   (the timer runs this copy, not the working tree):
   ```
   sudo install -d -o root -g root -m 0755 /usr/local/lib/gyst
   sudo install -o root -g root -m 0755 scripts/ops/n8n-postgres-backup.sh /usr/local/lib/gyst/n8n-postgres-backup.sh
   ```
   Re-run the `install` after every reviewed change to the script.
   Then install the timer. Offset the time from the GYST backup (pick a different
   minute/hour than that job):
   ```
   # /etc/systemd/system/n8n-pg-backup.service
   [Unit]
   Description=n8n Postgres backup (GYST-VN-095)
   [Service]
   Type=oneshot
   User=jay
   ExecStart=/usr/bin/bash /usr/local/lib/gyst/n8n-postgres-backup.sh <n8n pg container> <backup dir> <push URL file>

   # /etc/systemd/system/n8n-pg-backup.timer
   [Unit]
   Description=Daily n8n Postgres backup
   [Timer]
   OnCalendar=*-*-* 03:40:00
   RandomizedDelaySec=5m
   Persistent=true
   [Install]
   WantedBy=timers.target
   ```
   Then:
   ```
   sudo systemctl daemon-reload
   sudo systemctl enable --now n8n-pg-backup.timer
   systemctl list-timers n8n-pg-backup.timer
   ```
   `User=jay` must be able to run docker. Drop `<push URL file>` from
   `ExecStart` if you skip step 3.
6. After two scheduled runs: `journalctl -u n8n-pg-backup.service` (record
   names and times only) and `ls -l <backup dir>`.

## Scratch restore test (never into the live database)

The scratch container has no network, so it cannot reach n8n or anything else.
Use the same Postgres major version as the live container
(`<same major>`, check with `docker exec <n8n pg container> postgres --version`).

```
D=<backup dir>
F=$(ls -1 "$D"/n8n-*.dump | sort | tail -n 1)
time ( \
docker run --rm -d --network none -e POSTGRES_HOST_AUTH_METHOD=trust \
  --name n8n-restore-test postgres:<same major> && \
until docker exec n8n-restore-test pg_isready -q -h 127.0.0.1; do sleep 1; done && \
docker exec n8n-restore-test createdb -U postgres n8n_restore && \
docker exec -i n8n-restore-test pg_restore -U postgres -d n8n_restore --no-owner -x < "$F" )
echo "restore exit=$?"
```

Compare row counts. Confirm the table names first with `\dt` in each database;
n8n normally has these:

```
for t in workflow_entity credentials_entity execution_entity; do
  echo -n "$t restored: "
  docker exec n8n-restore-test psql -U postgres -d n8n_restore -Atc "select count(*) from $t"
  echo -n "$t live (read-only): "
  docker exec <n8n pg container> sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "select count(*) from '"$t"'"'
done
docker rm -f n8n-restore-test
```

Counts can differ slightly if n8n ran between the dump and the live count
(`execution_entity` grows). Workflows and credentials should match unless you
edited them since the dump. Run the live counts right after a fresh manual
backup to compare cleanly.

## Verification record (Jay fills in; names and numbers only)

Paste into Issue #108. No URLs, tokens or keys.

```
Scheduled run 1:  <date/time UTC>, file n8n-<UTC>.dump, <bytes>, ok
Scheduled run 2:  <date/time UTC>, file n8n-<UTC>.dump, <bytes>, ok
Modes:            dir 700, files 600 (yes/no)
Scratch restore:  exit 0 (yes/no), took <seconds>
Counts:           workflow_entity <restored>/<live>, credentials_entity <restored>/<live>, execution_entity <restored>/<live>
Kuma push B2:     green (yes/no/not used)
Encryption key:   stored apart from dumps (yes/no)
```

## Encryption key (do this once)

The dump keeps n8n credentials encrypted with n8n's encryption key
(`N8N_ENCRYPTION_KEY`, or the `encryptionKey` in n8n's config). The key is
**not** in the dump. Without it, restored credentials are unusable.

Store the key in your password manager. Never next to the dumps, in this repo,
or in any off-host copy of the dumps. Do not paste it anywhere an agent can
read.

## Failure and recovery

- Exit non-zero, Kuma goes red after the heartbeat, or `last_run.json` says
  `"result":"failed"`: its `step` field names where it stopped
  (`dump`, `verify`, `rename`, `retention`). Run `journalctl -u
  n8n-pg-backup.service` for the same step name. Earlier good dumps are kept.
- A kill -9 or power loss can leave an `n8n-*.dump.partial`; it is not counted
  as a backup and not deleted by retention. Delete it by hand.

## Rollback

```
sudo systemctl disable --now n8n-pg-backup.timer
sudo rm /etc/systemd/system/n8n-pg-backup.service /etc/systemd/system/n8n-pg-backup.timer
sudo systemctl daemon-reload
```

Delete the Kuma push monitor and `<push URL file>` if unused. Existing dumps
stay until you delete them (they are sensitive; delete when no longer needed).
Nothing in n8n, its database or the GYST app is changed by this task.
