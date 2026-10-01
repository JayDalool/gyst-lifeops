# VN-009 — Host file permissions (Issue #5)

**Operator: Jay only** (`risk:not-agent-eligible`). The agent prepared this runbook
and `scripts/ops/vn009-permission-check.sh`. The agent cannot run them: the
secret guard blocks agents from listing env and backup files.

**Binding (10A O-1):** no service or agent account (for example `gyst-agent`) may be
created until the verification record at the bottom of this file is filled in
and committed.

Target state:

| What | Mode |
|---|---|
| `.env*` files | `600` |
| `data/`, `backups/` and `letsencrypt/` directories | `700` |
| database, backup and key files, and anything inside `backups/` | `600` |
| `/srv/server/private` | `700` |

The check script and the apply commands use the same tests (any group or other
bit, `-perm /077`). That way the before-list records every mode the apply step
changes, and the rollback can undo all of it.

## What the agent already checked (2026-09-30, metadata only)
Only **host bind mounts** are affected by these chmods. Named Docker volumes (all
the Postgres DBs, n8n, Uptime Kuma, Portainer, SamVision workspaces, Caddy) live
under `/var/lib/docker/volumes` and are not touched.

| Bind-mounted path | Container | Process user (verified) | Owner of path | Safe at 700/600? |
|---|---|---|---|---|
| `repos/gyst-lifeops/data` | gyst-lifeops | uid 0 (root) | root | yes |
| `apps/nginx-proxy-manager/data`, `…/letsencrypt` | nginx-proxy-manager | uid 0 (all processes, including the nginx workers) | root | yes |
| `apps/minecraft/data` | minecraft | **uid 1000 (jay)**, not root | jay | yes (owner is the same uid) |
| `repos/gyst-lifeops/Caddyfile` | gyst-caddy | read-only file, not matched | — | not touched |

Other paths:

| Path | Now | Note |
|---|---|---|
| `/srv/server/private` | `755 jay:jay` | Readable by every account. Make it `700`; jay's cron that reads from it keeps working. |
| `/srv/server/backups` | `750 jay:jay` | jay's cron backups write here; they keep working at `700`/`600`. |
| env files | read by `docker compose` **on the host**, as the user who runs it | Must stay readable by that user (`jay`). Keep owner `jay`, mode `600`. |

**Broad matches (harmless).** `-name data` also hits source and world folders owned by
jay: for example `repos/*/src/data`, minecraft `world/data`, and smoke-test `data` folders
under `artifacts`. Setting those to `700` only removes other accounts' access. `.env.*`
also matches files like `.env.test`, which is intended.

**Side effect to accept:** with `gyst-lifeops/data` at `700` root, reading the production
DB from the host needs `sudo`. That includes the OPS1 runbook and agents'
`sqlite3 -readonly` checks. Backups still work, because
`docker compose exec -T gyst-lifeops /app/lifeops backup` runs inside the root container.

## 0. Get the script (≈1 min)
The script lives on `master` once PR #36 is merged. Until then, use the PR worktree:
```sh
S=/srv/server/repos/gyst-lifeops-task-5/scripts/ops/vn009-permission-check.sh   # or, after merge: /srv/server/repos/gyst-lifeops/scripts/ops/…
A=/srv/server/artifacts/gyst/vn009
sudo mkdir -p $A && sudo chmod 700 $A
```

## 1. Before (≈2 min)
```sh
sudo sh -c "bash $S > $A/before.txt; echo exit=\$? >> $A/before.txt"
sudo cat $A/before.txt
```
The list shows mode, owner and path only, never contents. The last line is `exit=1` when
something needs fixing.

**Check the owners.** Every env file should be owned by `jay`. If one is owned by
`root` and you run `docker compose` as `jay` for that project, fix the owner first
**and record it**:
```sh
echo "chown root:root→jay:jay <path>" | sudo tee -a $A/chowns.txt
sudo chown jay:jay <path>
```
If any listed `data` directory is bind-mounted by a container whose process isn't
root and isn't the directory's owner, skip it in step 2. Today there are none (see the
table above), but check in case something changed.

## 2. Apply (≈2 min)
Run each block and read its output. Each changed path is printed with its old mode.
```sh
R=/srv/server
P='( -name .git -o -name node_modules -o -name vendor ) -prune -o'
# env files → 600
sudo find $R $P -type f \( -name '.env' -o -name '.env.*' -o -name '*.env' \) ! -name '*.example' ! -name '*.sample' -perm /077 -exec chmod 600 {} + -printf '%m→600 %p\n'
# database / dump files → 600
sudo find $R $P -type f \( -name '*.db' -o -name '*.db-wal' -o -name '*.db-shm' -o -name '*.sqlite' -o -name '*.sqlite3' -o -name '*.sql.gz' -o -name '*.dump' \) -perm /077 -exec chmod 600 {} + -printf '%m→600 %p\n'
# anything inside a backups directory → 600
sudo find $R $P -type f -path '*/backups/*' -perm /077 -exec chmod 600 {} + -printf '%m→600 %p\n'
# private keys → 600
sudo find $R $P -type f \( -name '*.key' -o -name 'privkey*.pem' -o -name '*_rsa' -o -name '*_ed25519' \) -perm /077 -exec chmod 600 {} + -printf '%m→600 %p\n'
# data / backups / letsencrypt directories → 700
sudo find $R $P -type d \( -name data -o -name backups -o -name 'pre-rollback-*' -o -name letsencrypt \) -perm /077 -exec chmod 700 {} + -printf '%m→700 %p\n'
# the private directory → 700
sudo chmod 700 /srv/server/private
```
`$P` is unquoted on purpose so its words split into separate find arguments.

## 3. Verify (≈8 min, includes restarts)
Running containers keep files they already have open, so a permission problem only
shows up when a process starts again. **Restart every container whose bind mount was
touched**, then check:
```sh
sudo sh -c "bash $S > $A/after.txt; echo exit=\$? >> $A/after.txt"; sudo tail -2 $A/after.txt   # expect the "OK …" line and exit=0

# minecraft: players are disconnected for about a minute
docker restart gyst-lifeops nginx-proxy-manager minecraft
sleep 20; docker ps --format '{{.Names}}\t{{.Status}}'       # all Up; none "Restarting"
docker logs --since 2m gyst-lifeops 2>&1 | head -5            # timezone + "running", no "permission denied"
docker logs --since 2m nginx-proxy-manager 2>&1 | grep -i -E 'denied|error' | head
docker logs --since 2m minecraft 2>&1 | grep -i -E 'denied|exception' | head
docker compose -f /srv/server/repos/gyst-lifeops/docker-compose.yml exec -T gyst-lifeops /app/lifeops backup   # "verified: integrity_check ok"

# compose can still read every project's env file, using the exact files each project was started with (prints nothing on success)
docker ps -q | xargs docker inspect -f '{{index .Config.Labels "com.docker.compose.project.working_dir"}}|{{index .Config.Labels "com.docker.compose.project.config_files"}}' | sort -u | grep -v '^|' |
  while IFS='|' read -r dir files; do args=""; for f in $(echo "$files" | tr ',' ' '); do args="$args -f $f"; done
    (cd "$dir" && docker compose $args config -q && echo "compose ok: $dir") || echo "COMPOSE FAILED: $dir"; done
```
Then in a browser:
- GYST login works and Today loads.
- The Nginx Proxy Manager admin UI loads, and a proxied site still has a valid certificate.
- n8n, SamVision, MyFamilyExpenses, FMS and Uptime Kuma each load.
- The Minecraft server accepts a connection, if you use it.

## 4. Rollback (only if something broke)
```sh
# restore every mode recorded in before.txt (lines are "mode owner:group path"; safe for spaces in paths)
sudo awk '/^[0-7]+ [^ ]+ /{m=$1; p=$0; sub(/^[0-7]+ [^ ]+ /,"",p); printf "%s\t%s\n", m, p}' $A/before.txt \
  | while IFS=$'\t' read -r m p; do sudo chmod "$m" "$p"; done
# undo any owner changes from step 1
sudo cat $A/chowns.txt 2>/dev/null     # then: sudo chown root:root <path> for each line
```
Restore `/srv/server/private` too, if needed: `sudo chmod 755 /srv/server/private`.
Then restart the three containers again: `docker restart gyst-lifeops nginx-proxy-manager minecraft`.

## 5. Verification record (fill in, then commit; paths and modes only, no secrets)
- Date and time:
- `before.txt`: N lines in N categories. Keep it in `/srv/server/artifacts/gyst/vn009/`; don't commit it.
- Owner changes (`chowns.txt`): none / N
- After check: `OK …` / exit=0
- Restarted gyst-lifeops, nginx-proxy-manager and minecraft: all Up, no permission errors in their logs
- `docker compose … config -q`: all projects ok
- GYST backup after the change: verified / failed
- Browser checks: GYST / NPM / n8n / SamVision / MyFamilyExpenses / FMS / Uptime Kuma: ok
- Anything skipped, and why:
