#!/usr/bin/env bash
# GYST-VN-089 (#107): push disk-usage state to two Uptime Kuma Push monitors.
#
#   disk-push.sh <warn-url-file> <crit-url-file> <mount> [<mount> ...]
#
# Each URL file holds one Kuma push URL (a single line, mode 0600, kept outside
# the repo). The worst usage across the mounts decides the push:
#   >= 95%  -> crit monitor "down" (and warn monitor "down")
#   >= 85%  -> warn monitor "down", crit monitor "up"
#   else    -> both "up"
# The push URL is handed to curl as a config on stdin (-K -; -q first so ~/.curlrc is never read), so it never
# appears in argv, ps or logs. Output carries only the percent and a mount
# label, never a URL. Exit status: 0 = pushed, 1 = a push or df failed,
# 2 = usage or file-permission error. An unreadable mount pushes down and exits 1.
# Push URLs must be https:// (http only for localhost).
set -uo pipefail
PATH=/usr/sbin:/usr/bin:/sbin:/bin; export PATH

WARN_PCT=85
CRIT_PCT=95

if [ "$#" -lt 3 ]; then
  echo "usage: ${0##*/} <warn-url-file> <crit-url-file> <mount> [<mount> ...]" >&2
  exit 2
fi

warn_file="$1"; crit_file="$2"; shift 2

for f in "$warn_file" "$crit_file"; do
  if [ ! -f "$f" ] || [ ! -r "$f" ]; then
    echo "error: URL file missing or unreadable: $f" >&2
    exit 2
  fi
  mode=$(stat -c %a -- "$f" 2>/dev/null || echo "")
  if [ "$mode" != "600" ] && [ "$mode" != "400" ]; then
    echo "error: URL file must be mode 600 (is ${mode:-unknown}): $f" >&2
    exit 2
  fi
done

worst=-1
worst_label=""
unreadable=""
rc=0
for m in "$@"; do
  # --output=pcent is one column, so filesystem names with spaces cannot shift it.
  pct=""
  if mountpoint -q -- "$m" 2>/dev/null; then
    pct=$(df --output=pcent -- "$m" 2>/dev/null | tail -n 1 | tr -d '[:space:]%')
  fi
  case "$pct" in
    ''|*[!0-9]*)
      label=$(printf '%s' "$m" | tr -c 'A-Za-z0-9' '_')
      [ "$label" = "_" ] && label="root"
      echo "error: cannot read usage for mount $label" >&2
      rc=1; unreadable="$label"; continue ;;
  esac
  label=$(printf '%s' "$m" | tr -c 'A-Za-z0-9' '_')
  [ "$label" = "_" ] && label="root"
  echo "mount=$label used=${pct}%"
  if [ "$pct" -gt "$worst" ]; then worst="$pct"; worst_label="$label"; fi
done


push() { # url-file status msg
  local file="$1" status="$2" msg="$3" base url scheme rest auth
  base=$(head -n 1 -- "$file" | tr -d '[:space:]')
  base="${base%%\?*}"
  scheme="${base%%://*}"
  rest="${base#*://}"
  auth="${rest%%/*}"
  # Refuse userinfo first: it can disguise the real host (http://localhost:@evil/).
  case "$auth" in
    *@*) echo "error: push URL must not contain userinfo (@)" >&2; return 1 ;;
  esac
  case "$scheme" in
    https) ;;
    http)
      if ! [[ "$auth" =~ ^(localhost|127\.0\.0\.1|\[::1\])(:[0-9]+)?$ ]]; then
        echo "error: push URL must be https:// (http only for localhost)" >&2; return 1
      fi ;;
    *) echo "error: push URL must be https:// (http only for localhost)" >&2; return 1 ;;
  esac
  case "$base" in
    *[\"\\]*) echo "error: URL file holds unsupported characters" >&2; return 1 ;;
  esac
  url="${base}?status=${status}&msg=${msg}&ping="
  # The URL goes in on stdin as a curl config. curl's own output and errors
  # are discarded so nothing can echo the URL back.
  if printf 'url = "%s"\n' "$url" | curl -q -fsS --max-time 20 -K - >/dev/null 2>&1; then
    return 0
  fi
  echo "error: push failed ($status)" >&2
  return 1
}

warn_status=up; crit_status=up
if [ -n "$unreadable" ]; then
  # Any unreadable mount means we cannot vouch for disk space: push down.
  warn_status=down; crit_status=down
  msg="mount_unreadable_${unreadable}"
  worst_label="${unreadable}"
else
  msg="${worst_label}_${worst}pct"
  [ "$worst" -ge "$WARN_PCT" ] && warn_status=down
  [ "$worst" -ge "$CRIT_PCT" ] && crit_status=down
fi

push "$warn_file" "$warn_status" "$msg" || rc=1
push "$crit_file" "$crit_status" "$msg" || rc=1
echo "warn=$warn_status crit=$crit_status worst=${worst}%"
exit "$rc"
