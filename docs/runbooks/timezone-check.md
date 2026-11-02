# Timezone check runbook (GYST #193)

**Operator: Jay only.** An agent wrote this page and ran the image checks on
throwaway copies of the images, on an empty temporary database. No agent ran
any of it against the live container's data or `data/`.

Manitoba stays on UTC-05 all year from 2026-11-01 (IANA tzdata 2026e). GYST
builds before #193 took America/Winnipeg from the image's zone database. The
live image built on 2026-10-08 ships Debian tzdata 2026c, which still falls
back to UTC-06. From #193 on, the default zone is pinned in the binary
(`internal/clock/tzdata/`) and the image's zone data no longer matters for it.

## 1. Read-only checks on the running container

None of these print environment variables, secrets or database rows.

```bash
# The image's own zone data (stale image: 2026c and CST in winter).
docker exec gyst-lifeops sh -c 'dpkg -l tzdata | tail -1; TZ=America/Winnipeg date'
date    # the host, for comparison (2026e or later: EST, -0500)

# The zone the app resolved at startup.
docker logs gyst-lifeops 2>&1 | grep 'Application timezone'

# Which build is running.
docker inspect -f '{{.Image}} {{.Created}}' gyst-lifeops
```

The `TZ=... date` line only shows the image's zone data. For a build with #193
the app ignores it, so after the deploy also check what the app shows:

- Open **Today** at any time: the date matches your phone.
- **Best check, 00:00-01:00 local:** Today shows the new date, not
  yesterday's.
- Prayer times on Today match a trusted prayer timetable for Winnipeg (a
  stale build shows them 1 h early).

## 2. What a stale build wrote

While a stale build ran (from 2026-11-01 07:00 UTC until the deploy):

- Rows created between 00:00 and 01:00 local (05:00-06:00 UTC), from
  2026-11-02 on, carry the previous day's date: task completions, tracker and
  prayer logs, water, goal progress and quick-capture due dates.
- Manually typed weight times are stored 1 h late (the time was read as
  UTC-06).
- UTC instants (sessions, audit, sync times) and synced Health days from the
  phone are not affected. Prayer push notifications fired on time.

Issue #193 does not rewrite any row. To size the problem, count on a backup copy
only, never on `data/lifeops.db` (make the copy with `lifeops backup`, as in
`restore.md` A2):

```sql
SELECT 'task_completions', COUNT(*) FROM task_completions WHERE created_at >= '2026-11-02' AND strftime('%H', created_at) = '05';
SELECT 'tracker_logs', COUNT(*) FROM tracker_logs WHERE created_at >= '2026-11-02' AND strftime('%H', created_at) = '05';
SELECT 'health_water_log', COUNT(*) FROM health_water_log WHERE created_at >= '2026-11-02' AND strftime('%H', created_at) = '05';
SELECT 'goal_progress_entries', COUNT(*) FROM goal_progress_entries WHERE created_at >= '2026-11-02' AND strftime('%H', created_at) = '05';
SELECT 'tasks (quick capture)', COUNT(*) FROM tasks WHERE created_at >= '2026-11-02' AND strftime('%H', created_at) = '05';
SELECT 'manual weights', COUNT(*) FROM health_weight_entries WHERE source = 'manual' AND measured_at >= '2026-11-01T07:00:00Z';
```

The 05:xx window over-counts a little: a row made at 00:30 local that was
meant for that day is counted too. Any correction is a separate Issue with its
own review.

## 3. Rollback

The pin adds no migration and stores nothing new, so rolling back is the
image alone (deploy runbook §3 R-A):

```bash
# before deploying
docker tag gyst-lifeops-gyst-lifeops:latest gyst-lifeops-gyst-lifeops:rollback-<live sha>
# rollback
docker tag gyst-lifeops-gyst-lifeops:rollback-<live sha> gyst-lifeops-gyst-lifeops:latest
git checkout <live sha> && docker compose up -d --force-recreate gyst-lifeops
```

Rolling back to a build without #193 brings the 1-hour error back.

## 4. When IANA changes Winnipeg again

`TestPinnedWinnipegMatchesSystemTzdata` fails when the machine running the
tests has newer tzdata whose America/Winnipeg differs from the pin. Refresh
`internal/clock/tzdata/` as its README says, then deploy.
