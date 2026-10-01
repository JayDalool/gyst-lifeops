# VN-009 — Host file permissions (Issue #5)

**Operator: Jay only** (`risk:not-agent-eligible`). The agent prepared this runbook
and `scripts/ops/vn009-permission-check.sh`. The agent cannot run them: the
secret guard blocks agents from listing env and backup files.
**Binding (10A O-1):** no service or agent account (for example `gyst-agent`) may be
created until the verification record at the bottom of this file is filled in
and committed.

Target state:
- `.env*` files: `600`
- `data/` and backup directories: `700`
- database and backup files: `600`
- private keys: `600`
- `/srv/server/private`: `700`

## What the agent already checked (2026-09-30, metadata only)
| Item | Now | Note |
|---|---|---|
| `/srv/server/private` | `755 jay:jay` | readable by every account. Make it 700. |
| `gyst-lifeops/data` | `755 root:root` | Make it 700. The GYST container runs as **root**, so it keeps access. |
| `nginx-proxy-manager/data` | `755 root:root` | NPM runs as root. Its `letsencrypt/` holds private keys. |
| Containers running as root | gyst-lifeops, gyst-caddy, samvision-web, samvision-postgres, minecraft, n8n-postgres, myfamilyexpenses-db/app, fms-web/api/postgres, uptime-kuma, nginx-proxy-manager, portainer | Tightening host modes cannot lock these out. |
| Containers running as non-root | n8n (`node`), myfamilyexpenses-ocr (`ocr`), cloudflared (`65532`) | They use Docker **volumes** only (no host bind mounts), so host chmods don't affect them. |
| Env files | read by `docker compose` **on the host**, as the user who runs compose | They must stay readable by that user (`jay`). Keep the owner `jay` and use mode `600`. |

**Side effect to accept:** with `gyst-lifeops/data` at 700 root, you and agents can no
longer read the production DB from the host without `sudo`. This includes the
agents' read-only `sqlite3 -readonly` checks. That is the point of the change.
Backups still work, because `docker compose exec -T gyst-lifeops /app/lifeops backup`
runs inside the root container.

## 1. Before (≈2 min)
```sh
cd /srv/server/repos/gyst-lifeops
sudo mkdir -p /srv/server/artifacts/gyst/vn009 && sudo chmod 700 /srv/server/artifacts/gyst/vn009
sudo bash scripts/ops/vn009-permission-check.sh | sudo tee /srv/server/artifacts/gyst/vn009/before.txt
```
Read the list. It shows mode, owner and path only, never file contents.

**Check:** every env file line should show the owner `jay`. If one is owned by `root`
and you normally run `docker compose` as `jay` for that project, run
`sudo chown jay:jay <that file>` first. Otherwise compose won't be able to read it
after the chmod.

## 2. Apply (≈2 min)
Run each block, then look at its output:
```sh
R=/srv/server
P='( -name .git -o -name node_modules -o -name vendor ) -prune -o'
# env files → 600
sudo find $R $P -type f \( -name '.env' -o -name '.env.*' -o -name '*.env' \) ! -name '*.example' ! -name '*.sample' -perm /077 -exec chmod 600 {} + -printf '%m→600 %p\n'
# database / backup files → 600
sudo find $R $P -type f \( -name '*.db' -o -name '*.db-wal' -o -name '*.db-shm' -o -name '*.sqlite' -o -name '*.sqlite3' -o -name '*.sql.gz' -o -name '*.dump' \) -perm /077 -exec chmod 600 {} + -printf '%m→600 %p\n'
# private keys → 600
sudo find $R $P -type f \( -name '*.key' -o -name 'privkey*.pem' -o -name '*_rsa' -o -name '*_ed25519' \) -perm /077 -exec chmod 600 {} + -printf '%m→600 %p\n'
# data / backup / letsencrypt dirs → 700 (review the before-list first: skip any you don't want)
sudo find $R $P -type d \( -name data -o -name backups -o -name 'pre-rollback-*' -o -name letsencrypt \) -perm /077 -exec chmod 700 {} + -printf '%m→700 %p\n'
# the private directory → 700
sudo chmod 700 /srv/server/private
```
`$P` is deliberately unquoted so its words split into find arguments.
**Before running the directory command:** if the before-list shows a `data` directory
that a **non-root** container bind-mounts, skip it. From the table above there are
none, but check whether anything has changed since.

## 3. Verify (≈5 min)
```sh
sudo bash scripts/ops/vn009-permission-check.sh | sudo tee /srv/server/artifacts/gyst/vn009/after.txt   # expect: OK …, exit 0
# compose can still read every project's env file (prints nothing on success)
for d in /srv/server/repos/gyst-lifeops /srv/server/apps/n8n /srv/server/apps/samvisionai /srv/server/repos/MyFamilyExpenses /srv/server/repos/FMS /srv/server/apps/nginx-proxy-manager /srv/server/apps/uptime-kuma /srv/server/apps/cloudflared; do (cd "$d" && docker compose config -q && echo "compose ok: $d"); done
docker ps --format '{{.Names}}\t{{.Status}}'                     # every container Up, none restarting
docker compose -f /srv/server/repos/gyst-lifeops/docker-compose.yml exec -T gyst-lifeops /app/lifeops backup   # "verified: integrity_check ok"
```
Then in a browser:
- GYST login works and Today loads.
- n8n, SamVision, MyFamilyExpenses, FMS, Uptime Kuma and Nginx Proxy Manager each load.

Optional: at a quiet time, `docker compose up -d --force-recreate gyst-lifeops`, then log in
again. This proves a fresh start reads the env file and the DB.

## 4. Rollback (only if something broke)
```sh
# restore every mode recorded in before.txt (lines are "mode owner:group path")
sudo awk '/^[0-7]+ /{m=$1; $1=""; $2=""; sub(/^  /,""); printf "%s\t%s\n", m, $0}' /srv/server/artifacts/gyst/vn009/before.txt \
  | while IFS=$'\t' read -r m p; do sudo chmod "$m" "$p"; done
```
Then restore `/srv/server/private` with `sudo chmod 755 /srv/server/private` if it's needed.

## 5. Verification record (fill in, then commit; paths and modes only)
- Date and time:
- Before-list: N lines in N categories (keep the file in `/srv/server/artifacts/gyst/vn009/`; don't commit it)
- After check: `OK …` / exit 0
- `docker compose config -q`: all projects ok
- Containers healthy after the change: yes / no (list any that weren't)
- GYST backup after the change: verified / failed
- Anything skipped, and why:
