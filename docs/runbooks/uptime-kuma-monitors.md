# Uptime Kuma monitors (GYST-VN-089, Issue #107)

**Operator: Jay only** (`risk:not-agent-eligible`). The agent prepared this
runbook and `scripts/ops/disk-push.sh`. The agent never touched Kuma, the host,
cron or any token. Everything below uses placeholders: `<public GYST URL>`,
`<tailnet GYST URL>`, `<n8n URL>`, `<SamVision URL>`, `<push URL file>`. Never
paste a real URL or push token into the repo, a PR or an Issue comment.

Alerts go straight from Kuma to Jay through Kuma notifications. GYST does not
read Kuma (VN-090 and VN-094 are dropped).

## Why /healthz is public

VN-073 (internal listener) is dropped, so `/healthz` stays on the single
`:8080` listener. Caddy proxies every path on the public site, so `/healthz` is
reachable from the internet. That is acceptable: the body is just `ok`, and the
probe is not logged (`internal/app/healthz.go`). VN-014's note that Caddy "does
not route it publicly" is out of date; this runbook is the record.

The public `/healthz` also shows whether the database answers (200 vs 503) and runs one `SELECT 1` per hit. Accepted.

## Part A: do now

Common settings for HTTP monitors: method GET, accepted status `200-299`,
max redirects 0, timeout 10 s. Retries 3 means three failed checks in a row
before Kuma alerts (keeps alert fatigue down).

| # | Name | Type | Target | Interval | Retries | Expected | Enabled by |
|---|---|---|---|---|---|---|---|
| A1 | GYST public | HTTP(s) - Keyword | `<public GYST URL>/healthz`, keyword `ok` | 60 s | 3 | status 200, body contains `ok` | VN-014 (done) |
| A2 | GYST tailnet | HTTP(s) - Keyword | `<tailnet GYST URL>/healthz`, keyword `ok` | 60 s | 3 | status 200, body contains `ok` | VN-014 (done) |
| A3 | n8n | HTTP(s) | `<n8n URL>/healthz` (current n8n; verify it answers 200 first) | 60 s | 3 | status 200 | now |
| A4 | SamVision | HTTP(s) | `<SamVision URL>` (its health or home URL) | 60 s | 3 | status 200 | now |
| A5 | Disk warn | Push | token in `<warn push URL file>` | heartbeat 30 min | 1 | `up` pushed every 15 min while usage is under 85% | this Issue |
| A6 | Disk crit | Push | token in `<crit push URL file>` | heartbeat 30 min | 1 | `up` pushed every 15 min while usage is under 95% | this Issue |

Notes:
- A1: if your Kuma version offers certificate-expiry notification, turn it on.
- A2: first check from inside the Kuma container that the tailnet site is
  reachable. If it is not, do not create A2. Record `skipped: unreachable` in
  the verification record and say why (container cannot reach the tailnet).
- A5/A6: a 30 min heartbeat with a 15 min script run means a dead script or
  dead cron also turns the monitor red.
- Add your notification channel to every monitor (Kuma: Settings, Notifications,
  tick "apply to all existing monitors" only if you want it on all).

### Steps for Jay

1. In Kuma, create A1 to A4 from the table. Save each. Wait for green.
2. Create A5 and A6 as Push monitors. Kuma shows a push URL for each. Copy it
   once, store it in a file outside the repo, one line, no other content:
   ```
   install -m 600 /dev/null <warn push URL file>
   install -m 600 /dev/null <crit push URL file>
   # then paste the URL into each file with your editor; confirm: stat -c %a <file>  -> 600
   ```
   Do not leave the URL in shell history or a terminal log.
3. Test the script by hand once (replace placeholders):
   ```
   bash scripts/ops/disk-push.sh <warn push URL file> <crit push URL file> / <data mount>
   ```
   Expected output: `mount=... used=NN%` lines, then
   `warn=up crit=up worst=NN%` and exit 0. A5 and A6 turn green.
4. Install the script outside the repo checkout (from a reviewed master
   checkout), so the timer never runs code that a branch switch could change:
   ```
   sudo install -d -o root -g root -m 0755 /usr/local/lib/gyst
   sudo install -o root -g root -m 0755 scripts/ops/disk-push.sh /usr/local/lib/gyst/disk-push.sh
   ```
   Push URLs must be `https://`; the script refuses `http://` unless the host is
   localhost, 127.0.0.1 or [::1].
5. Schedule it every 15 minutes as user `jay`. Keep the unit files out of the repo.
   ```
   # /etc/systemd/system/gyst-disk-push.service
   [Service]
   Type=oneshot
   User=jay
   ExecStart=/usr/bin/bash /usr/local/lib/gyst/disk-push.sh <warn push URL file> <crit push URL file> / <data mount>

   # /etc/systemd/system/gyst-disk-push.timer
   [Timer]
   OnCalendar=*:0/15
   Persistent=true
   [Install]
   WantedBy=timers.target
   ```
   Then:
   ```
   sudo systemctl daemon-reload && sudo systemctl enable --now gyst-disk-push.timer
   systemctl list-timers gyst-disk-push.timer
   journalctl -u gyst-disk-push.service -n 20
   ```
   If any listed mount cannot be read or is not a mount point, the script pushes `down` to both disk
   monitors and exits 1.
6. Deliberate down test (required): push `down` by hand to A5 using the same
   stdin technique, so the token stays out of argv:
   ```
   printf 'url = "%s?status=down&msg=test&ping="\n' "$(head -n1 <warn push URL file> | tr -d '[:space:]' | cut -d'?' -f1)" | curl -q -fsS --max-time 20 -K - >/dev/null
   ```
   Confirm A5 turns red and an alert
   arrives. Then wait for the next scheduled run (or run step 3) and confirm A5
   returns to green.

## Part B: later, gated

Push monitors with a 26 h heartbeat (daily job plus 2 h slack), retries 1.
Create each only when its enabling task is live; until then do not create it
(it would just alert). Each backup job pushes `status=up` on success and
`status=down` on failure, using the same 0600-file and stdin technique.

| # | Name | Type | Target | Heartbeat | Retries | Expected | Enable when |
|---|---|---|---|---|---|---|---|
| B1 | GYST backup | Push | token in `<GYST backup push URL file>` | 26 h | 1 | `up` after each daily backup (push hook in `scripts/backup-daily.sh`) | VN-010 (#6) is live |
| B2 | n8n Postgres backup | Push | token in `<n8n backup push URL file>` | 26 h | 1 | `up` after each n8n Postgres dump | VN-095 / GYST-129 is live |
| B3 | Off-host copy | Push | token in `<offhost push URL file>` | 26 h | 1 | `up` after each off-host copy | VN-012 is live |

## Verification record (Jay fills in; names and states only)

Paste into Issue #107. No URLs, no tokens.

```
A1 GYST public:   green / red
A2 GYST tailnet:  green / skipped: unreachable (reason)
A3 n8n:           green / red
A4 SamVision:     green / red
A5 Disk warn:     green
A6 Disk crit:     green
Down test on A5:  went red (alert received yes/no), recovered to green yes/no
Part B:           not created yet (waiting on VN-010 / VN-095 / VN-012)
```

## Rollback

Delete the monitors in Kuma, then:
```
sudo systemctl disable --now gyst-disk-push.timer
sudo rm /etc/systemd/system/gyst-disk-push.service /etc/systemd/system/gyst-disk-push.timer
sudo rm /usr/local/lib/gyst/disk-push.sh
sudo rmdir /usr/local/lib/gyst   # ignore if not empty
sudo systemctl daemon-reload
```
Delete the push URL files. Nothing else is changed: no GYST, Caddy or compose edits.
