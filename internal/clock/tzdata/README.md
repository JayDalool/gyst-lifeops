# Pinned America/Winnipeg zone (GYST #193)

`America_Winnipeg` is the compiled (TZif) America/Winnipeg zone that
`internal/clock` embeds as the default application timezone. `VERSION` is the
IANA tzdata release it comes from.

Why it is pinned: Manitoba stays on UTC-05 all year from 2026-11-01 (IANA
tzdata 2026e and later). Go 1.25.14 embeds tzdata 2025b and Go 1.27.3 embeds
2026c, and the production image used to ship 2026c, so `time.LoadLocation`
alone would put GYST an hour behind from 2026-11-01. The pinned file makes the
default zone the same in every build, image and test run.

## Provenance (2026f)

- Source: `https://data.iana.org/time-zones/releases/tzdata2026f.tar.gz`
  (sha256 `01b4a91cec5167ed48c11e8ab78f2048cead78ce94b0cafcad0ae8c5a3044d3d`),
  with a good signature (`tzdata2026f.tar.gz.asc`) from Paul Eggert, key
  `7E37 92A9 D8AC F7D6 33BC 1588 ED97 E90E 62AA 7E34`.
- Built with `zic -b fat -d out northamerica` (glibc 2.43 zic).
- Output sha256 `4cbeda42e9c21c0e3b8017248071bbd194136350581e3c71ee3312c0f6ddb383`.
  It is byte-identical to `/usr/share/zoneinfo/America/Winnipeg` in Debian
  `tzdata 2026f-0+deb12u1`, and its `zdump -v` transitions match Ubuntu
  `tzdata 2026e-0ubuntu0.26.04.1`.
- From 2026-11-01 07:00 UTC the zone is UTC-05 with the abbreviation `EST`
  (IANA's compatibility choice). Tests check offsets, not abbreviations.

## Refreshing

`TestPinnedWinnipegMatchesSystemTzdata` fails when the machine running the
tests has a newer tzdata release whose America/Winnipeg differs from the pin.
Then repeat the steps above with the new release, replace `America_Winnipeg`
and `VERSION`, and update this file.

Production checks, the data a stale build wrote, and rollback:
`docs/runbooks/timezone-check.md`.
