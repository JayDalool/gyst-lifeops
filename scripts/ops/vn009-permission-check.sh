#!/usr/bin/env bash
# GYST-VN-009 (#5): list secret, database and backup files on this host that
# carry any group/other permission bits. The tests are exactly the ones the
# runbook's apply step uses, so the before-list records every mode it changes. Prints ONLY "mode owner:group path", never file
# contents, so the output is safe to keep as evidence.
#
# Run as root so nothing is skipped:   sudo bash scripts/ops/vn009-permission-check.sh
# Exit status: 0 = nothing exposed, 1 = something listed below needs fixing.
set -uo pipefail

ROOT="${1:-/srv/server}"
exposed=0

report() { # header, then find arguments
  local header="$1"; shift
  local out
  out=$(find "$@" -printf '%m %u:%g %p\n' 2>/dev/null | sort)
  if [ -n "$out" ]; then
    echo "## $header"
    echo "$out"
    echo
    exposed=1
  fi
}

prune=( \( -name .git -o -name node_modules -o -name vendor \) -prune -o )

# 1. Env files (any .env, .env.local, prod.env …) readable by group or others.
report "env files with any group/other bits (want 600)" \
  "$ROOT" "${prune[@]}" -type f \( -name '.env' -o -name '.env.*' -o -name '*.env' \) \
  ! -name '*.example' ! -name '*.sample' -perm /077

# 2. SQLite databases, WAL/SHM side files and backups readable by others.
report "database / backup files with any group/other bits (want 600)" \
  "$ROOT" "${prune[@]}" -type f \( -name '*.db' -o -name '*.db-wal' -o -name '*.db-shm' \
  -o -name '*.sqlite' -o -name '*.sqlite3' -o -name '*.sql.gz' -o -name '*.dump' \) -perm /077

# 2b. Anything inside a backups directory (tarballs, .sql, .bak …) (want 600).
report "files inside backups directories with any group/other bits (want 600)" \
  "$ROOT" "${prune[@]}" -type f -path '*/backups/*' -perm /077

# 3. Data and backup directories other accounts can enter (want 700).
report "data / backup directories with any group/other bits (want 700)" \
  "$ROOT" "${prune[@]}" -type d \( -name data -o -name backups -o -name 'pre-rollback-*' -o -name letsencrypt \) \
  -perm /077

# 3b. Private keys and certificates' keys (want 600).
report "private key files with any group/other bits (want 600)" \
  "$ROOT" "${prune[@]}" -type f \( -name '*.key' -o -name 'privkey*.pem' -o -name '*_rsa' -o -name '*_ed25519' \) -perm /077

# 4. The private directory itself (want 700).
if [ -d "$ROOT/private" ]; then
  report "private directory (want 700)" "$ROOT/private" -maxdepth 0 -perm /077
fi

# 5. Docker volume root (should already be 700/710 root; listed only if wider).
report "docker volumes root readable or writable by group/others" /var/lib/docker/volumes -maxdepth 0 -perm /066

if [ "$exposed" -eq 0 ]; then
  echo "OK: no world/group-readable secret, database or backup files under $ROOT"
fi
exit "$exposed"
