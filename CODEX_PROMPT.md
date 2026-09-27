# CODEX HANDOFF PROMPT

> **Superseded (2026-09-27).** This 2026-09-12 Codex handoff is historical. Current governance: [`AGENTS.md`](AGENTS.md) · workflow: [`docs/agents/`](docs/agents/README.md) · tasks: GitHub Issues. Don't follow the task order or commit trailer below.

## GYST LifeOps — Continue from A2 Implementation

**Current Date:** 2026-09-12  
**Current State:** OPS1 ✅ | Tailscale HTTPS ✅ | Deployment live on port 3005  
**Your Task:** Implement A2 + A1, then M3. A3/A4 (auth) deferred to later phase.

---

## CRITICAL CONTEXT FIRST

**Read these in order BEFORE any work:**
1. `docs/AI_HANDOFF.md` — live project map and state
2. `docs/RULES.md` — 8 rules that govern every change
3. `docs/tasks/README.md` — task ledger with acceptance criteria
4. `AGENTS.md` — complete project architecture and how-to
5. Your assigned task file (e.g., `docs/tasks/A2-trusted-tls.md`)

**Git is the source of truth.** If docs disagree with the code, code wins. If code disagrees with tests, tests win.

---

## WHAT YOU'RE WALKING INTO

### Project: GYST LifeOps Tracker
- **Type:** Single-user, self-hosted PWA (Progressive Web App)
- **Stack:** Go 1.25 + SQLite (WAL mode, enforced foreign keys)
- **Frontend:** Server-rendered Go templates, vanilla JS, plain CSS
- **Security:** TLS via Tailscale MagicDNS, session-based auth (to be added), CSRF guards
- **Mobile:** iPhone primary target

### Current Deployment Status
- **Live:** http://localhost:3005 (Caddy reverse proxy)
- **Database:** SQLite at `data/lifeops.db` in WAL mode
- **Containers:** gyst-lifeops (Go app) + gyst-caddy (reverse proxy)
- **Config:** docker-compose.yml, Dockerfile, Caddyfile

### What's DONE
| Task | Commit | Status |
|------|--------|--------|
| B1 SQLite PRAGMA correctness | fb4f1eb | ✅ |
| B2 Backup tooling (VACUUM INTO) | 820c704 | ✅ |
| B6 CI safety gate | f663f3f | ✅ |
| A3 Session store | 7835b7d | ✅ |
| A4 Application auth | fb05cc3 | ✅ |
| A5 Health API write-only | 18196e9 | ✅ |
| M4 Day view (reads real data) | 60be715 | ✅ |
| Documentation | d4007a3, 12fabb6 | ✅ |
| OPS1 FK audit | (manual) | ✅ |
| Tailscale HTTPS | (manual) | ✅ |

### What's NEXT (Your Work)
1. **A2 — Trusted TLS** (unblock Apple Shortcuts integration)
   - Update Caddyfile to use MagicDNS certificate
   - Set up 90-day cert renewal automation
   - Test from iPhone Shortcut
2. **A1 — Remove plaintext port** (must follow A2)
   - Remove port 3005 from docker-compose.yml
   - Verify app only accessible via Caddy/HTTPS
3. **M3 — Apple Health ingestion** (Shortcut sync handler)
   - `/api/health/sync` endpoint reads Shortcut POST data
   - Stores observations (steps, calories, etc.) in database
   - Handles NULL vs 0 semantics correctly

### What's DEFERRED (After A2/A1/M3)
- **A3/A4 (Authentication)** — Currently documented as done but not implemented. Will add login + password hashing in next phase after current work ships.

---

## FILE GUIDE: WHAT EACH FILE DOES

### Root Config Files
| File | Purpose |
|------|---------|
| `go.mod`, `go.sum` | Go dependencies (golang.org/x/crypto, modernc.org/sqlite, etc.) |
| `Dockerfile` | Builds Go binary with CGO_ENABLED=0 (static), copies web assets |
| `docker-compose.yml` | Defines gyst-lifeops (app) and gyst-caddy (reverse proxy) services |
| `Caddyfile` | Reverse proxy config; currently uses `tls internal` (self-signed) |

### Database & Migrations
| File | Purpose |
|------|---------|
| `migrations/*.sql` | All database schema (timestamp-prefixed, additive only, never edited) |
| `internal/db/db.go` | Connection pooling, DSN building, PRAGMA enforcement (`foreign_keys=1`, `busy_timeout=5000`) |
| `internal/db/backup.go` | B2: VACUUM INTO backups with integrity verification |
| `internal/db/pragma_scope_test.go` | Proves per-connection PRAGMA enforcement works |

### Application Core
| File | Purpose |
|------|---------|
| `internal/app/app.go` | Main App struct, initialization, route setup |
| `internal/app/cli.go` | CLI entry point (dispatches backup subcommand) |
| `internal/app/backup_cmd.go` | Backup CLI implementation |
| `cmd/lifeops/main.go` | Binary entry point (just calls app.Run()) |

### HTTP Handlers & Middleware
| File | Purpose |
|------|---------|
| `internal/handlers/` | All HTTP handlers (health, calendar, auth, etc.) |
| `internal/handlers/middleware.go` | Session validation, CSRF guard (Sec-Fetch-Site + Origin), auth exemptions |
| `internal/handlers/auth.go` | Login/logout pages (currently auth is documented but not deployed) |
| `internal/handlers/health_view.go` | Presentation layer: HealthMetricView, buildHealthTodayView(), formatThousands() |
| `internal/handlers/health_api.go` | `/api/health/sync` (write-only, A5) — WHERE M3 WILL ADD READ |
| `internal/handlers/health_plan.go` | `/health/plan` view (Day view, M4) |

### Services (Business Logic)
| File | Purpose |
|------|---------|
| `internal/services/health.go` | Health calculations (observed burn, energy balance, never double-count energy) |
| `internal/services/auth.go` | VerifyPassword (bcrypt), hashSessionToken (SHA-256), session lifecycle |
| `internal/repository/` | Database queries (sessions, users, health snapshots, etc.) |

### Web Templates & Assets
| File | Purpose |
|------|---------|
| `web/templates/health.html` | Day view rendering (real data or empty state, never 0 for missing) |
| `web/templates/login.html` | Login form |
| `web/templates/base.html` | HTML skeleton (base layout for all pages) |
| `web/static/` | CSS, JS modules (no build system, vanilla only) |

### Seeds & Examples
| File | Purpose |
|------|---------|
| `seeds/example.sql` | Example data loaded on first run (empty database only) |

### Documentation (READ FIRST)
| File | Purpose |
|------|---------|
| `docs/AI_HANDOFF.md` | Live state map, branch overview, next action |
| `docs/RULES.md` | 8 rules + failure sources; read in full before committing |
| `docs/tasks/` | Task ledger (one .md per work item with acceptance criteria) |
| `AGENTS.md` | Complete architecture guide for new agents |
| `README.md` | Quickstart and deployment notes |

### CI/CD
| File | Purpose |
|------|---------|
| `.github/workflows/check.yml` | Runs: gofmt, go vet, go build, go test, go test -race, migration uniqueness check |

---

## HOW TO WORK ON THIS REPO

### Before Starting Any Task
```bash
# 1. Ensure you're on the right branch
git checkout feature/health-hardening
git pull origin feature/health-hardening

# 2. Verify current state
git log --oneline -5
git status

# 3. Read the docs (in order)
# - docs/AI_HANDOFF.md
# - docs/RULES.md
# - docs/tasks/README.md
# - Your task file

# 4. Run local tests to establish baseline
go test ./...
go test -race ./...
```

### While Working on a Task
```bash
# Create a feature branch if needed (usually work on feature/health-hardening directly)
git checkout -b feature/a2-trusted-tls  # if you want isolation

# Make changes, test locally
go fmt ./...
go vet ./...
go build -o /tmp/test-build .
go test ./...
go test -race ./...

# Commit ONE logical change at a time
git add <specific-files>
git commit -m "commit message here"

# Push to feature branch
git push -u origin feature/health-hardening
```

### After Completing a Task
1. Verify CI is green (GitHub Actions runs on every push)
2. Update task file: `docs/tasks/A2-trusted-tls.md`
   - Set Status to DONE
   - Record commit SHA(s)
   - Record CI run link
3. Update `docs/AI_HANDOFF.md` → section 3 (State at a glance)
4. Commit the docs update in the same commit or immediately after
5. Push everything to `feature/health-hardening`

### Testing Your Work
```bash
# Unit tests
go test ./...

# Race condition detector (catches concurrency bugs)
go test -race ./...

# Build the binary
go build -o ./lifeops .

# Run the app locally (if needed)
./lifeops -db=/tmp/test.db

# Run in Docker (closer to production)
docker compose up -d
docker compose logs gyst-lifeops
curl http://localhost:3005
```

---

## THE THREE RULES THAT MATTER MOST

1. **Git is proof.** If it's not in Git with a green CI run, it's not done.
2. **NULL ≠ 0.** "Not received" vs "observed zero" are kept distinct all the way down. Never render NULL as 0.
3. **Never weaken a constraint.** Foreign keys are structural. Tests guard against regressions. Trust them.

See `docs/RULES.md` for all 8 rules and the failures that created them.

---

## YOUR FIRST TASK: A2 — Trusted TLS

**What to do:**
1. Read `docs/tasks/A2-trusted-tls.md` (full acceptance criteria)
2. Update `Caddyfile` to use Tailscale MagicDNS certificate
3. Implement 90-day renewal automation (certificates expire)
4. Test from iPhone: Shortcut `Get Contents of URL` must succeed
5. Update task file when complete

**Key files you'll touch:**
- `Caddyfile` (site address, tls directive, renewal)
- `docs/tasks/A2-trusted-tls.md` (update Status to DONE)
- `docs/AI_HANDOFF.md` (update the table in section 3)

**Why this matters:**
- Apple Shortcuts validates TLS; self-signed cert fails silently
- A2 unblocks A1 (remove plaintext port)
- A1 unblocks M3 (Shortcut can only reach HTTPS)

---

## GETTING UNSTUCK

**"Is this task actually done?"**
- Check: Git has a commit? CI is green? Task file says DONE? AI_HANDOFF.md lists it?
- If all four: yes. Otherwise: no.

**"Why did a test fail?"**
- Read the test error (it names the failure)
- Check `docs/RULES.md` — most failures violate a rule there
- Reproduce locally: `go test -run TestName -v`

**"Can I do X?"**
- Check `docs/RULES.md` section "Do not do yet"
- If not forbidden and doesn't weaken a constraint: probably yes

**"What branch do I push to?"**
- Always: `feature/health-hardening`
- Never push anywhere else without explicit permission

---

## ARCHITECTURE YOU NEED TO KNOW

### Database Safety
- Foreign keys enforced per-connection (every query sees FK enforcement)
- Schema has 10 `ON DELETE CASCADE` and 13 `ON DELETE SET NULL`
- Migrations are filename-tracked; never edit an applied migration
- Never reset `data/lifeops.db`; use fresh migrations

### Health Calculations (Critical)
- **Never double-count energy:** Resting + Active only. No step-derived calories added.
- **Watch and phone steps stay separate:** Never sum them (sources overlap).
- **NULL ≠ 0:** "Not synced" ≠ "observed zero". Keep them distinct in code and UI.
- **No projection:** Show only observed data. No end-of-day estimates.

### Deployment Safety
- Production database is live. Always backup before any changes.
- Backup command: `docker compose exec gyst-lifeops /app/lifeops backup`
- Backup uses VACUUM INTO (not cp), verifies with PRAGMA integrity_check
- Never auto-repair production foreign-key violations

---

## COMMUNICATION

- **Questions?** Check `docs/RULES.md` and `AGENTS.md` first
- **Blocked?** Update your task file with the blocker and next action
- **Done?** Update docs and commit with clear message
- **Changes visible to others?** Update `docs/AI_HANDOFF.md` in the same commit

---

## YOU HAVE EVERYTHING YOU NEED

Start with A2. Read `docs/tasks/A2-trusted-tls.md`. Ask questions if the acceptance criteria are unclear. Commit one logical change at a time. Push to `feature/health-hardening`. Update docs. Move to the next task.

**The next agent will read this same handoff. Leave breadcrumbs in the docs so they know what you did and what's next.**

Go.
