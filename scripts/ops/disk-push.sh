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
# The push URL is handed to curl as a config on stdin (-K -), so it never
# appears in argv, ps or logs. Output carries only the percent and a mount
# label, never a URL. Exit status: 0 = pushed, 1 = a push or df failed,
# 2 = usage or file-permission error.
set -uo pipefail

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
rc=0
for m in "$@"; do
  pct=$(df -P -- "$m" 2>/dev/null | awk 'NR==2 {gsub("%","",$5); print $5}')
  case "$pct" in
    ''|*[!0-9]*) echo "error: cannot read usage for mount $m" >&2; rc=1; continue ;;
  esac
  label=$(printf '%s' "$m" | tr -c 'A-Za-z0-9' '_')
  [ "$label" = "_" ] && label="root"
  echo "mount=$label used=${pct}%"
  if [ "$pct" -gt "$worst" ]; then worst="$pct"; worst_label="$label"; fi
done

# No readable mount at all: push nothing. The push heartbeat (30 min) then
# turns red on its own, which is the intended dead-script signal.
if [ "$worst" -lt 0 ]; then exit 1; fi

push() { # url-file status msg
  local file="$1" status="$2" msg="$3" base url
  base=$(head -n 1 -- "$file" | tr -d '[:space:]')
  base="${base%%\?*}"
  case "$base" in
    https://*|http://*) ;;
    *) echo "error: URL file does not hold an http(s) URL" >&2; return 1 ;;
  esac
  case "$base" in
    *[\"\\]*) echo "error: URL file holds unsupported characters" >&2; return 1 ;;
  esac
  url="${base}?status=${status}&msg=${msg}&ping="
  # The URL goes in on stdin as a curl config. curl's own output and errors
  # are discarded so nothing can echo the URL back.
  if printf 'url = "%s"\n' "$url" | curl -fsS --max-time 20 -K - >/dev/null 2>&1; then
    return 0
  fi
  echo "error: push failed ($status)" >&2
  return 1
}

msg="${worst_label}_${worst}pct"

warn_status=up; crit_status=up
[ "$worst" -ge "$WARN_PCT" ] && warn_status=down
[ "$worst" -ge "$CRIT_PCT" ] && crit_status=down

push "$warn_file" "$warn_status" "$msg" || rc=1
push "$crit_file" "$crit_status" "$msg" || rc=1
echo "warn=$warn_status crit=$crit_status worst=${worst}%"
exit "$rc"
