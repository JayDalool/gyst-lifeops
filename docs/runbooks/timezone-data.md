# Time zone data runbook (Issue #193)

**Operator: Jay only.** Every command here is read-only: it opens no
database, changes nothing and prints no environment values. An agent wrote
this and ran the `tzinfo` commands against local builds only. Nobody has run
any of it against production.

## What this is about

Manitoba moved to permanent UTC-05 on 2026-10-31. It did not fall back on
2026-11-01 and has no clock changes in 2027. IANA tzdb 2026e models this
(abbreviation `EST`, applied from 2026-11-01 02:00, a temporary detail).

Before #193, GYST took zones from Go's `time.LoadLocation`. That reads the
container's `/usr/share/zoneinfo` first. If the container has none, it falls
back to the copy built into Go, which is tzdb 2025b for every Go 1.25 build.
Any data older than 2026e still falls back to UTC-06 on 2026-11-01. With
stale data, since 2026-11-01 the app would be **one hour behind**:

- prayer times shown an hour early;
- reminders an hour late;
- entries made between 00:00 and 00:59 filed under the previous day;
- the Health day ending at 01:00.

Stored instants (UTC) are not affected. **Production has not been checked.**

From #193 the binary carries its own pinned database (`internal/tzdb`,
release and hashes in `internal/tzdb/PROVENANCE`). It never reads the image's
zone files, `$ZONEINFO` or Go's copy. The Docker build runs
`lifeops tzinfo -check` and stops if the Winnipeg offsets are wrong.

Run everything from the repo root on the host (`/srv/server/repos/gyst-lifeops`).

## 1. Before deploying #193: is production affected?

These commands look at the image that is running now.

```bash
# Zone files in the running image. The old binary prefers these.
docker compose exec -T gyst-lifeops sh -c \
  'if [ -e /usr/share/zoneinfo/America/Winnipeg ]; then
     head -n 1 /usr/share/zoneinfo/tzdata.zi 2>/dev/null || echo "zone files present, release unknown"
   else
     echo "no zone files: the binary uses the copy built into Go (2025b for Go 1.25)"
   fi'
```

- `# version 2026e` or later: the running app has the right offsets.
- Anything older, or no zone files: the running app is one hour behind
  since 2026-11-01.

Cross-check what the app actually shows. Open Today and read the Dhuhr
time in the prayer row. With the default prayer settings, under permanent
UTC-05 it is **1:12 PM** on 2026-11-02, **1:13 PM** on 11-15, **1:18 PM** on
12-01 and **1:38 PM** on 2027-01-15. Stale data shows the same prayer one
hour earlier (12:12 PM, 12:13 PM, …). Changed prayer adjustments shift these
by their minutes.

Optional, on any machine with an up-to-date OS (`dpkg-query -W tzdata`
should show 2026e or later):

```bash
TZ=America/Winnipeg date   # the correct local time, to compare with GYST's
```

Record only the outcome (the release seen, and whether Dhuhr was right or an
hour early). Do not record any real data.

## 2. After deploying #193: verify

```bash
docker compose exec -T gyst-lifeops /app/lifeops tzinfo -check; echo "exit=$?"
docker compose logs --tail 50 gyst-lifeops | grep 'Application timezone'
```

Pass:

- `exit=0` and a line `probe: America/Winnipeg, 14 known UTC offsets
  (history and permanent UTC-05): OK`.
- `tzdb: 2026e, pinned in the binary (zoneinfo.zip sha256 f2d09714…)`. The
  hash matches `zoneinfo_zip_sha256` in `internal/tzdb/PROVENANCE` for the
  deployed commit.
- `now: … UTC offset -05:00, local day …` shows today's Winnipeg date.
- The log line reads `Application timezone: America/Winnipeg (tzdb 2026e)`.
- The Today page's Dhuhr matches the values in step 1.

`tzinfo` prints the zone name from `GYST_TIMEZONE` (a zone name, not a
secret) and nothing else from the environment. A `host /usr/share/zoneinfo:`
line says what the image holds; GYST does not use it.

## 3. Rollback

#193 has no migration and writes nothing, so rollback is only a redeploy of
the previous commit, done the same way as any deploy. Rolling back brings the
stale-data risk back: run step 1 again afterwards.

## 4. Data written while the app was an hour behind

No rows are rewritten by #193. If step 1 showed stale data, entries made
between 00:00 and 00:59 local from 2026-11-01 until the deploy may be filed
under the previous day. Counting or repairing them is a separate decision
for a new Issue. Use a backup copy only, never the live database
(`docs/RULES.md` §3.1).

## 5. Refreshing the pinned database (a future rule change)

On a Linux or WSL development machine with `curl`, `make`, a C compiler,
`gpg` and `go`:

1. In `scripts/tzdb/update.sh`, set `VERSION` and the two `SHA512` values
   from the IANA tz-announce message for the new release.
2. Run `scripts/tzdb/update.sh`. It downloads the release, checks the
   SHA-512 pins and the signature (key `7E3792A9D8ACF7D633BC1588ED97E90E62AA7E34`),
   builds it with Go's `lib/time/update.bash` recipe and rewrites
   `internal/tzdb/zoneinfo.zip` and `PROVENANCE`.
3. Read the release's `NEWS`. Change `clock.DefaultZoneProbes` only if
   Manitoba's rules changed again.
4. Run `scripts/tzdb/update.sh --check` (a second clean build must be
   byte-identical) and the full checks, including `go test -race ./...`.
   `TestPinnedAgreesWithSystemBuildOfSameRelease` compares every zone with
   the OS build of the same release, where one is installed.
5. Open a PR for the Issue. Deploy only after Jay merges it.

Anyone can confirm the committed zip without trusting this repository:
`scripts/tzdb/update.sh --check` rebuilds it from IANA's signed release.
