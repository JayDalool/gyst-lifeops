#!/usr/bin/env bash
# GYST-VN-095 (#108): dump the n8n Postgres database to a local directory.
#
#   n8n-postgres-backup.sh <container> <backup-dir> [<push-url-file>]
#
#   container       name of the n8n Postgres container (placeholder in the repo)
#   backup-dir      ABSOLUTE path, outside any git repository
#   push-url-file   optional: one Uptime Kuma push URL, mode 0600, outside the repo
#
# Run by Jay (or his systemd timer, User=jay). Agents never run it.
# The dump runs INSIDE the container with the container's own POSTGRES_USER and
# POSTGRES_DB, so no credential is read, passed or printed on the host.
# Steps: pg_dump -Fc -> n8n-<UTC>.dump.partial -> pg_restore --list check ->
# rename to n8n-<UTC>.dump. Files are 0600 in a 0700 directory. Only the newest
# 7 n8n-<UTC>.dump files in that directory are kept. last_run.json (0600) records
# the last run. The dump holds n8n credentials (encrypted with n8n's own
# encryption key, which is NOT in the dump), so treat it as sensitive.
#
# Output: step names and the dump file name only; never a URL or env value.
# Exit status: 0 = ok, 2 = bad usage (nothing run), any other non-zero = a step
# failed (status=down pushed, last_run.json says which step).
set -euo pipefail
umask 077

KEEP=7

usage() { echo "usage: ${0##*/} <container> <absolute-backup-dir> [<push-url-file>]" >&2; exit 2; }
[ "$#" -ge 2 ] && [ "$#" -le 3 ] || usage

container="$1"; dir="$2"; pushfile="${3:-}"

case "$container" in
  ''|-*|*[!A-Za-z0-9_.-]*) echo "error: container name must be letters, digits, _ . - and not start with -" >&2; exit 2 ;;
esac

dir="${dir%/}"   # strip one trailing slash
case "$dir" in
  /*) ;;
  *) echo "error: backup dir must be an absolute path" >&2; exit 2 ;;
esac
[ -n "$dir" ] || { echo "error: backup dir must not be /" >&2; exit 2; }
case "$dir/" in
  */../*) echo "error: backup dir must not contain a .. segment" >&2; exit 2 ;;
  */./*|*//*) echo "error: backup dir must not contain a . segment or a double slash" >&2; exit 2 ;;
esac

# Refuse a dir inside a git work tree. Check from the nearest existing ancestor
# (the dir may not exist yet). The check is a read-only git call.
probe="$dir"
while [ ! -d "$probe" ]; do probe="$(dirname -- "$probe")"; done
if git -C "$probe" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: backup dir is inside a git repository; refusing" >&2
  exit 2
fi

if [ -n "$pushfile" ]; then
  if [ ! -f "$pushfile" ] || [ ! -r "$pushfile" ]; then
    echo "error: push URL file missing or unreadable" >&2; exit 2
  fi
  mode=$(stat -c %a -- "$pushfile" 2>/dev/null || echo "")
  if [ "$mode" != "600" ] && [ "$mode" != "400" ]; then
    echo "error: push URL file must be mode 600 (is ${mode:-unknown})" >&2; exit 2
  fi
fi

ts=$(date -u +%Y%m%dT%H%M%SZ)
fname="n8n-${ts}.dump"
final="$dir/$fname"
partial="$final.partial"
step="start"
result="failed"
size=0
locked=0

write_status() {
  local tmp now file=""
  now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  [ "$result" = ok ] && file="$fname"
  tmp=$(mktemp -- "$dir/.last_run.json.XXXXXX") || return 1
  printf '{"time_utc":"%s","file":"%s","size_bytes":%s,"result":"%s","step":"%s"}\n' \
    "$now" "$file" "$size" "$result" "$step" > "$tmp" || return 1
  chmod 600 -- "$tmp" && mv -f -- "$tmp" "$dir/last_run.json"
}

push() { # status msg ; the URL goes to curl on stdin only, never argv
  local status="$1" msg="$2" base
  [ -n "$pushfile" ] || return 0
  base=$(head -n 1 -- "$pushfile" | tr -d '[:space:]')
  base="${base%%\?*}"
  case "$base" in
    https://*|http://*) ;;
    *) echo "error: push URL file does not hold an http(s) URL" >&2; return 1 ;;
  esac
  case "$base" in
    *[\"\\]*) echo "error: push URL file holds unsupported characters" >&2; return 1 ;;
  esac
  if printf 'url = "%s?status=%s&msg=%s&ping="\n' "$base" "$status" "$msg" \
      | curl -fsS --max-time 20 -K - >/dev/null 2>&1; then
    return 0
  fi
  echo "error: push failed ($status)" >&2
  return 1
}

on_exit() {
  local rc=$?
  trap - EXIT
  set +e
  [ "$rc" -eq 0 ] || result="failed"
  if [ "$locked" -eq 1 ]; then rm -f -- "$partial"; fi
  [ "$locked" -eq 0 ] || write_status || { echo "error: could not write last_run.json" >&2; [ "$rc" -ne 0 ] || rc=1; }
  if [ "$rc" -eq 0 ]; then
    push up "ok" || rc=1
  else
    push down "failed_at_${step}" || true
    echo "FAILED at step: $step" >&2
  fi
  exit "$rc"
}
trap on_exit EXIT
trap 'exit 143' TERM
trap 'exit 130' INT

step="mkdir"
mkdir -p -- "$dir"
chmod 700 -- "$dir"

# One run at a time. A second run exits 1 and leaves the first run's files alone.
step="lock"
exec 9>>"$dir/.lock"
if ! flock -n 9; then
  echo "error: another backup run holds the lock on $dir; exiting" >&2
  exit 1
fi
locked=1

step="dump"
if [ -e "$final" ]; then echo "error: $fname already exists" >&2; exit 1; fi
# Single quotes: $POSTGRES_USER and $POSTGRES_DB expand inside the container.
docker exec "$container" sh -c 'pg_dump -Fc -U "$POSTGRES_USER" -d "$POSTGRES_DB"' > "$partial"
chmod 600 -- "$partial"

step="verify"
[ -s "$partial" ] || { echo "error: dump is empty" >&2; exit 1; }
docker exec -i "$container" pg_restore --list < "$partial" > /dev/null

step="rename"
mv -- "$partial" "$final"
size=$(stat -c %s -- "$final")
echo "dump ok: $fname ($size bytes)"

step="retention"
# Only regular, non-symlink files named exactly n8n-YYYYMMDDTHHMMSSZ.dump in
# this directory count. Glob order is name order, so oldest first.
shopt -s nullglob
dumps=()
for f in "$dir"/n8n-[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]T[0-9][0-9][0-9][0-9][0-9][0-9]Z.dump; do
  if [ -f "$f" ] && [ ! -L "$f" ]; then dumps+=("$f"); fi
done
shopt -u nullglob
if [ "${#dumps[@]}" -gt "$KEEP" ]; then
  for f in "${dumps[@]:0:${#dumps[@]}-KEEP}"; do rm -f -- "$f"; done
fi

step="done"
result="ok"
