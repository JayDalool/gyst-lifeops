# AGENTS.md — Codex Handoff: Complete GYST LifeOps Context

**For:** Next agent (Codex) taking over GYST LifeOps development  
**Current branch:** `feature/health-hardening`  
**Last update:** 2026-09-12  
**Status:** M2.5 implementation complete; operator tasks pending  

---

## READ FIRST

Before any task:
1. Read [`docs/AI_HANDOFF.md`](docs/AI_HANDOFF.md) — the live map of all work
2. Read [`docs/RULES.md`](docs/RULES.md) — the rules governing every change
3. Read [`docs/tasks/README.md`](docs/tasks/) — the task ledger with acceptance criteria
4. Run `git status` and `git log --oneline -10`
5. Verify you are on `feature/health-hardening`

**Git is the source of truth.** If this file disagrees with the repository, the repository wins.

---

## 1. THE PROJECT

**GYST LifeOps Tracker** is a **single-user, self-hosted PWA** built in Go + SQLite.

- **Platform:** Mobile-first (iPhone primary target), server-rendered Go templates, vanilla JS modules, plain CSS, Docker + Caddy
- **Database:** SQLite in WAL mode, foreign keys enforced per-connection, additive migrations only
- **Cryptography:** bcrypt (cost 12) for passwords, SHA-256 hex for session tokens, Secure/HttpOnly/SameSite=Lax cookies
- **Security:** CSRF guard (Sec-Fetch-Site + Origin validation), no secrets in code or docs, structured read-only for backups
- **Health architecture:** Apple Health → Shortcut → GYST API, separate Watch/phone step tracking, deterministic calculations only

### Core Concepts You Must Understand

**Nutrition semantics:**
- `NULL` = "not received yet" (e.g., MyFitnessPal hasn't synced)
- `0` = "an observed zero" (e.g., you ate nothing, or took zero steps)
- **Never render NULL as 0.** These are kept separate all the way down the stack.

**Energy (the single most important rule):**
- **Food budget** = target − consumed (answering: "how much can I still eat?")
- **Observed energy balance** = consumed − observed burn (answering: "did I hit my goal?")
- **Never sum them.** Never project. Never add Active Energy + step-derived calories + workout calories.
- Observed burn = resting + active only. Steps are a separate activity metric.

**Steps:**
- Store Watch and phone steps **separately.**
- **Never add them together.** Sources overlap; summing = double-counting.
- Watch is primary; phone is context.

---

## 2. CURRENT BRANCH STATE

**Baseline:** `feature/health-core` (3083fb9) — Health Core service layer exists but handlers incomplete  
**Current HEAD:** `feature/health-hardening` (12fabb6)  
**Commits since baseline:** 14  
**Status:** M2.5 hardening phase **complete**; M3 (ingestion) and operator tasks **pending**

### Completed Work (All Committed, All CI Green)

| Task | Commit | Status | CI |
|------|--------|--------|-----|
| B1 SQLite PRAGMA correctness | `fb4f1eb` | DONE | predates CI |
| B1-FIX review fixes | `a1dc3da`, `29adc43` | DONE | predates CI |
| B6 CI safety gate | `f663f3f` | DONE | green |
| A3 session store | `7835b7d` | DONE | green |
| A4 application auth | `fb05cc3` | DONE | green |
| A5 Health API write-only | `18196e9` | DONE | green |
| M4 Day view (reads real data) | `60be715` | DONE | green |
| B2 backup tooling | `820c704` | DONE | green |
| Documentation & ledger | `d4007a3`, `5341e46`, `ccec84a`, `12fabb6` | DONE | green |

### Blocked Work (Waiting on Operator)

| Task | Blocker | Owner |
|------|---------|-------|
| OPS1 production FK audit | Run `PRAGMA foreign_key_check` on live database | Jay |
| A2 trusted TLS | Enable Tailscale HTTPS | Jay |
| GYST_PASSWORD_HASH | Set environment variable after A4 deployment | Jay |
| A1 remove plaintext port | Unblocked once A2 is ready | Agent |
| M3 Apple Health ingestion | Unblocked once OPS1, GYST_PASSWORD_HASH, Tailscale HTTPS are done | Agent |

---

## 3. KEY FILES AND MODULES

### Database (`internal/db/`)

**db.go** — Connection pooling and PRAGMA enforcement
- `Open(dbPath string)` — creates connection pool, applies `_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)` to **every connection**
- `sqliteDSN(path string)` — converts filesystem paths to file: URIs safely
- `escapeSQLiteURIPath()` — escapes special characters in filesystem paths
- `validateCallerDSN()` — rejects :MEMORY: and in-memory forms; allows only file paths or sqlite: URIs

**pragma_scope_test.go** — Proves per-connection PRAGMA enforcement
- FK violations caught on 3 of 4 pooled connections when foreign_keys unenforced
- Validates driver behavior empirically

**backup.go** — B2 implementation (uses VACUUM INTO)
- `Backup(sourcePath, destinationPath)` — atomic backup with integrity verification
- Source opened in read-only mode (mode=ro); copy verified with `PRAGMA integrity_check`
- Deletes failed backups; default destination `data/backups/` beside source
- **Do not use `cp`.** Main file can be near-empty while -wal holds most data (measured: 4 KB vs 1.3 MB).

### Authentication (`internal/services/auth.go`, `internal/handlers/auth.go`, `internal/handlers/middleware.go`)

**auth.go**
- `VerifyPassword(hashedPassword, password string)` — bcrypt verify (cost 12 for single-user)
- `hashSessionToken(token string)` — SHA-256 hex of raw token
- `CreateSession(userID string, lifespan time.Duration)` — 32-byte random token, lifetime = 30 days
- `ValidateSession(tokenHash string)` — retrieves from db, checks expiry
- `SweepExpiredSessions()` — runs hourly, deletes expired rows

**middleware.go**
- `sessionCookieName` = "gyst_session" (Secure/HttpOnly/SameSite=Lax)
- `requireSameOrigin()` — validates Sec-Fetch-Site and Origin headers (CSRF guard)
- `authExemptPath()` — returns false for /login, /logout, /api/health*, /calendar/feed/* (token-based feeds must work unauthenticated)
- `Authenticated()` — checks cookie, validates hash, refreshes if stale (24h window)

**sessions.go** (`internal/repository/`)
- `Session{TokenHash TEXT PRIMARY KEY, ExpiresAt TEXT}`
- Methods: Create, Get, Touch, Delete, DeleteExpired, Count

### Health (`internal/handlers/health_view.go`, `internal/handlers/health_api.go`)

**health_view.go** — Presentation layer (rules enforced here, not in template)
- `HealthMetricView{Value string, Unit string, Present bool}` — three-part representation
  - Value: preformatted, empty when absent
  - Unit: "kcal", "g", "min", "" (for steps)
  - Present: explicitly tracks absent vs observed zero
- `formatThousands(float64)` — adds commas, uses U+2212 (minus sign, not hyphen) for negatives
- `buildHealthTodayView()` — turns service response into presentation values
  - **All absent-vs-zero decisions happen here.** Template holds no arithmetic.
  - Calories is an explicit field (not indexed), prevents template index-out-of-range aborts
- `humaniseSince()` — coarse age ("just now", "15 min ago", "2 hours ago", "1 day ago")
- `staleAfter` = 8 hours — data beyond this is flagged as stale

**health_api.go** — A5: read-only API (no new handlers for write)
- `/api/health/sync` — accepts POST from Shortcut
- Removed deprecated "today" case; use `/health` view instead

### CLI (`internal/app/`)

**cli.go** — Dispatches backup subcommand before flag parsing
- `backupRequested(os.Args)` — peeks at args for "backup" keyword
- Backup has its own flag set (-db, -out); server flags don't apply

**backup_cmd.go** — Backup subcommand implementation
- `RunBackup(args []string)` — parses -db and -out flags
- `humanBytes()` — formats file sizes (e.g., "1.2 MB")
- Handles `flag.ErrHelp` cleanly (no exit code 2 panic)

### Web (`web/templates/health.html`)

- Renders real data or empty state
- `{{if .Present}}...{{else}}<span class="is-absent">No data</span>{{end}}`
- Uses `{{with .Today.Calories}}` (explicit field, not index), prevents template abort
- Never shows 0 for missing data; never sums Watch+phone steps; never projects

### CI (`.github/workflows/check.yml`)

Runs on every push:
- `gofmt` — code format
- `go vet` — static analysis
- `go build` — builds binary
- `go test` — unit tests
- `go test -race` — race detector
- Migration prefix uniqueness — all files match `YYYYMMDDHHMMSS_*.sql`
- No database file committed — rejects `data/lifeops.db` push

---

## 4. RULES THAT GOVERN EVERY CHANGE

**Read [`docs/RULES.md`](docs/RULES.md) in full.** These are the failure sources:

### Evidence
- **Git is the source of truth.** If this file disagrees, the repo wins.
- **Read whole files.** Partial reads miss context (learned: CSRF guard on /health/plan wasn't visible until full file read).
- **Measure before fixing.** Backups lost data silently (main 4 KB, -wal 1.3 MB); only measured against live server.
- **Probe dependencies.** PRAGMA scope proved empirically with 3-of-4 connections test.
- **Negative-test security.** Never trust a security feature without a test that proves it fails when disabled.

### Changing Code
- **One commit at a time.** One feature per commit, one reviewable change. No batching unrelated work.
- **Never weaken a test.** If a test breaks, fix the code, not the test.
- **Never weaken a foreign-key constraint.** The schema exists to catch bugs; trust it.
- **Never weaken the Health API.** No projection, no double-counting, no invented values.

### Data
- **NULL ≠ 0.** Keep them distinct all the way down the stack. `NULL` means "not received"; `0` means "observed zero."
- **Never sum Watch + phone steps.** Sources overlap; summing double-counts.
- **Never double-count energy.** Do not add Active Energy + step-derived calories + workout calories.
- **Never render missing data as zero.** Use explicit `.Present` checks.

### Secrets
- **Never put secrets, tokens, API keys, passwords, bearer tokens, or personal credentials in code or docs.**
- Docs are on the implementation branch and visible to the agent; treat them as public-facing.

### Branches
- **Do not rewrite, squash, force-push, or rebase completed commits.**
- Completed work (B1, B2, A3, A4, A5, M4, docs) is immutable.
- Only the next unstarted task gets a fresh commit.

### Ledger
- **Update task files and AI_HANDOFF.md in the same commit as the work or the one immediately after.**
- Record commit SHAs, CI results, blockers, and next action.
- If Git does not prove completion, mark it PENDING or UNKNOWN.

### Honesty
- **Do not claim work is complete unless Git proves it.** CI green means the suite passes, not that the app was used.
- **Nothing above has been verified in a real browser on a real device.**

### Scope Guards
- Do not implement Hevy (not decided yet; no tables exist).
- Do not auto-repair production foreign-key violations.
- Do not replace the existing Galaxy without explicit approval.
- Do not hard-code mockup sample values as production data.
- Do not edit applied migrations.
- Do not reset `data/lifeops.db`.

---

## 5. HOW TO CONTINUE

### Next Action: Operator Steps (Jay)

These are **not agent tasks.** They belong to Jay and must be done before the next agent work can unblock.

**OPS1 — Production Foreign-Key Audit** (`docs/tasks/OPS1-production-fk-audit.md`)
```sh
# 1. Back up first (using B2 backup tooling)
docker compose exec gyst-lifeops /app/lifeops backup

# 2. Stop the app so the check cannot miss rows
docker compose stop gyst-lifeops

# 3. Run the audit on the host
sqlite3 ./data/lifeops.db "PRAGMA foreign_key_check;"
```
- Zero rows → certified; B1 is production-safe
- Any rows → **STOP.** Report them; do not repair.

**Set GYST_PASSWORD_HASH** (before A4 deployment)
```sh
go run ./cmd/passwordhash
# Set the output in .env as GYST_PASSWORD_HASH
# Verify login works before closing session
```

**Enable Tailscale HTTPS** (unblocks A2, A1, M3)
- See `docs/tasks/A2-trusted-tls.md` for details

### After Operator Steps: Unblocked Agent Work

Once OPS1, GYST_PASSWORD_HASH, and Tailscale HTTPS are done:

**A1** — Remove plaintext HTTP port (blocked on A2)  
**A2** — Trusted TLS setup (blocked on Tailscale HTTPS)  
**M3** — Apple Health Shortcut ingestion (blocked on operator steps)

No other agent work remains in M2.5.

---

## 6. SECURITY AND SAFETY GUARDRAILS

### Database Safety
- **Foreign keys are enforced.** The schema has 10 `ON DELETE CASCADE` and 13 `ON DELETE SET NULL` rules. Trust them.
- **Never touch production data without a backup.** Use B2 backup tooling: `docker compose exec gyst-lifeops /app/lifeops backup`
- **Never reset `data/lifeops.db`.** Migrations are filename-tracked; a reset causes them to re-run and fail.
- **Migrations are additive only.** No `DROP TABLE`, no `DROP COLUMN`, no destructive `UPDATE`. Use timestamps: `YYYYMMDDHHMMSS_name.sql`
- **Never edit an applied migration.** The runner tracks by filename; an edit is silently ignored.

### Authentication Safety
- **Sessions live 30 days.** Refresh window is 24 hours (automatic on every request).
- **Password hash cost is 12.** Single-user deployment; cost 11 is acceptable, 13 is slow.
- **Session tokens are 32 bytes random.** SHA-256 hash is stored in db; raw token only in cookie.
- **CSRF guard is active.** Sec-Fetch-Site + Origin validation on every unsafe method (POST/PUT/PATCH/DELETE).
- **Calendar feeds are token-based.** /calendar/feed/{token} routes are auth-exempt (no session required); tokens are separate.

### Health Calculations
- **Resting + active only.** Never project. Never add step-derived calories.
- **Watch and phone steps stay separate.** Never sum them in view logic.
- **Observed zero is distinct from absent.** `NULL` in database, `Present: false` in view, empty in template.
- **Food budget and observed balance are separate.** They answer different questions and happen to have similar magnitudes—exactly why conflating them is dangerous.

### Backup Safety
- **Use VACUUM INTO, not `cp`.** The -wal file can hold most data; cp copies only main file (measured: 4 KB vs 1.3 MB loss).
- **Backups are verified with `PRAGMA integrity_check`.** Failed backups are deleted.
- **Source is opened read-only (mode=ro).** Concurrent writes don't corrupt it.

---

## 7. DEVELOPMENT WORKFLOW

### Before Every Task

```bash
git fetch origin feature/health-hardening
git checkout feature/health-hardening
git log --oneline -5
git status
```

Read in order:
1. `docs/AI_HANDOFF.md` — current state map
2. `docs/RULES.md` — rules
3. `docs/tasks/` — all task files
4. The task file for your assigned work

### During a Task

- One commit per logical change
- Commit message: clear, concise, no model identifiers
- End with: `Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>`
- Run: `gofmt`, `go vet`, `go build`, `go test`, `go test -race`
- All must pass locally before push

### After a Task

- Push to `feature/health-hardening` with `-u` flag
- Update task file and `AI_HANDOFF.md` in the commit or immediately after
- Record: commit SHA, CI result (run URL), blockers, next action
- Claim completion only when Git proves it (CI green, acceptance criteria met)
- Never claim "tested in a browser" unless you actually did

---

## 8. THE MANDATE

From `BUILD_PLAN.md` (the original authority):

> You have **full authority** to make decisions, prioritize work, and change course if evidence suggests the plan is wrong. Every architectural decision is open for re-evaluation if new data arrives.
>
> **Four hard guardrails:**
> 1. The production database is a live service. Never touch it without a backup. Never auto-repair.
> 2. Foreign-key enforcement is structural. Never weaken a constraint to make something else easier.
> 3. The schema exists to catch bugs. Trust it.
> 4. Health calculations are deterministic. No projection, no invention, no AI guessing.

---

## 9. QUICK REFERENCE: FILES TO KNOW

### Docs (Read First, Every Time)
- `docs/AI_HANDOFF.md` — live state map, branch overview, next action
- `docs/RULES.md` — rules and failure sources
- `docs/tasks/README.md` — task ledger index
- `docs/tasks/B1-*.md`, `A3-*.md`, `A4-*.md`, `A5-*.md`, `M4-*.md`, `B2-*.md`, etc. — one per work item

### Code (Organized by Responsibility)
- `internal/db/db.go` — connection pooling, PRAGMA enforcement
- `internal/db/backup.go` — VACUUM INTO backup implementation
- `internal/services/auth.go` — authentication (bcrypt, SHA-256)
- `internal/handlers/middleware.go` — session validation, CSRF guard
- `internal/handlers/health_view.go` — presentation layer (absent-vs-zero enforcement)
- `internal/handlers/health_api.go` — A5 read-only API
- `internal/app/cli.go`, `internal/app/backup_cmd.go` — CLI dispatch

### Web
- `web/templates/health.html` — Day view rendering
- `web/templates/login.html` — authentication form

### Config and Deploy
- `.env.example` — template (never commit real `.env`)
- `docker-compose.yml` — container definition
- `.github/workflows/check.yml` — CI pipeline
- `Dockerfile` — Go 1.25, CGO_ENABLED=0, modernc.org/sqlite v1.52.0

### Migrations
- `migrations/` — all numbered `YYYYMMDDHHMMSS_*.sql` files, additive only
- Never edit an applied migration
- Never reset the database

---

## 10. GETTING UNSTUCK

### "Is this complete?"
Check Git:
- Does the feature branch have a commit for this task?
- Does that commit pass CI?
- Does the task file in `docs/tasks/` record the commit SHA?
- Does `docs/AI_HANDOFF.md` list it as DONE?

If all four are true, it's complete. Otherwise, it's not.

### "What's the current state?"
- Run: `git log --oneline -20 origin/feature/health-hardening`
- Read: `docs/AI_HANDOFF.md` section 3 (State at a Glance)
- The table lists every task, its status, commits, and CI result.

### "Why did something break?"
- Read: `docs/RULES.md` — most failures come from violating these
- Check: the task file for the work (e.g., `docs/tasks/M4-live-day-view.md`)
- Measure: don't assume; reproduce the failure locally

### "Can I do X?"
- Check: `docs/RULES.md` section "Do not do yet" and "Scope Guards"
- If it's not explicitly forbidden and doesn't violate the four hard guardrails, ask yourself: does Git and CI prove it's safe?

### "Where do I push?"
- **Always:** `feature/health-hardening`
- **Never:** another branch without explicit permission
- Command: `git push -u origin feature/health-hardening`

---

## 11. CONTACT AND HANDOFF

- **User (Jay):** jay16ca@gmail.com
- **Primary reference:** `docs/AI_HANDOFF.md` (live map)
- **Rules reference:** `docs/RULES.md`
- **Task tracker:** `docs/tasks/` (one file per work item)

If your work is blocked or you discover a rule violation in existing code, update the relevant task file and `docs/AI_HANDOFF.md` immediately. The next agent will read them.

---

**You have everything you need. Start with the next unblocked task. Git and the docs are your source of truth.**
