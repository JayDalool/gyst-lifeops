#!/usr/bin/env bash
# Seed a DISPOSABLE local GYST instance with fictional projects for the Galaxy
# baseline (GYST-VN-143). It uses the app's own "New project" form, so it
# exercises the same code path as a user.
#
# Safety: refuses any base URL that isn't 127.0.0.1 / localhost. Never point it
# at production. Start a throwaway instance first, for example:
#
#   docker run -d --rm --name gyst-galaxy-baseline -p 127.0.0.1:18090:8080 \
#     -e GYST_AUTH_DISABLED=1 -v "$(mktemp -d)":/app/data \
#     gyst-lifeops-gyst-lifeops:latest /app/lifeops -seed=false
#
# Usage: seed-fictional-galaxy.sh <base-url> <active-count> [--with-states]
#   --with-states also adds one paused, one upcoming, one locked and four
#   completed ("done") projects, to exercise state rendering and the
#   completed-project exclusion.
set -euo pipefail

base="${1:?usage: $0 <base-url> <active-count> [--with-states]}"
count="${2:?usage: $0 <base-url> <active-count> [--with-states]}"
states="${3:-}"

case "$base" in
  http://127.0.0.1:*|http://localhost:*) ;;
  *) echo "refusing non-local base URL: $base" >&2; exit 2 ;;
esac

# Values allowed by the projects table CHECK constraints.
categories=(coding "sim rig" car health business personal other)
colors=(cyan violet emerald amber rose blue orange magenta)

create() { # name category color status
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$base/projects/new" \
    --data-urlencode "name=$1" \
    --data-urlencode "description=Fictional Galaxy baseline project" \
    --data-urlencode "category=$2" \
    --data-urlencode "color=$3" \
    --data-urlencode "status=$4" \
    --data-urlencode "project_kind=main")
  [ "$code" = "303" ] || { echo "create '$1' failed: HTTP $code" >&2; exit 1; }
}

for i in $(seq 1 "$count"); do
  create "Demo World $i" "${categories[$((i % 7))]}" "${colors[$((i % 8))]}" active
done

if [ "$states" = "--with-states" ]; then
  create "Demo Paused World" personal amber paused
  create "Demo Upcoming World" business blue upcoming
  create "Demo Locked World" other rose locked
  for i in 1 2 3 4; do create "Demo Completed World $i" coding emerald done; done
fi

echo "seeded $count active fictional projects${states:+ (+ state fixtures)} at $base"
