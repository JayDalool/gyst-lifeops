# 🤖 HANDOFF TO CODEX — Complete Briefing

**Date:** 2026-09-12  
**Status:** Ready for Codex to take over  
**Current branch:** `feature/health-hardening` (all work here)  
**Deployment:** Live on port 3005 (port 3443 for HTTPS via Caddy)

---

## START HERE — What To Do First

1. **Clone or pull the repo:**
   ```bash
   git clone https://github.com/JayDalool/gyst-lifeops.git
   cd gyst-lifeops
   git checkout feature/health-hardening
   ```

2. **Read these docs in order (10 min total):**
   - `CODEX_PROMPT.md` — detailed file guide + task walkthrough
   - `docs/AI_HANDOFF.md` — live state map
   - `docs/RULES.md` — 8 rules you must follow
   - `docs/tasks/A2-trusted-tls.md` — your first task

3. **Start working:** First task is **A2 — Trusted TLS**

---

## WHAT EACH FILE DOES — Quick Reference

### 📋 Documentation (Read These First)
| File | What It Is | Read When |
|------|-----------|-----------|
| **CODEX_PROMPT.md** | Detailed handoff: files, workflow, architecture | Before any work |
| **docs/AI_HANDOFF.md** | Live state map, branch overview, task table | Before any work |
| **docs/RULES.md** | 8 rules + failures that created them | Before committing |
| **docs/tasks/README.md** | Task ledger index (all task files listed) | To find your task |
| **docs/tasks/A2-trusted-tls.md** | Your first task (acceptance criteria, blockers) | To start A2 |
| **AGENTS.md** | Complete architecture guide for agents | If stuck or context needed |
| **README.md** | Quickstart and deployment notes | For deployment questions |

### 🗄️ Database & Migrations
| File | Purpose | Touch When |
|------|---------|-----------|
| **migrations/*.sql** | Schema (additive only, never edited) | Adding schema changes |
| **internal/db/db.go** | Connection pooling, DSN, PRAGMA enforcement | Database setup issues |
| **internal/db/backup.go** | B2: VACUUM INTO backups | Backup/restore work |
| **internal/db/pragma_scope_test.go** | Tests proving per-connection PRAGMA enforcement | Understanding FK enforcement |

### 🚀 Application Core
| File | Purpose | Touch When |
|------|---------|-----------|
| **cmd/lifeops/main.go** | Binary entry point (just calls app.Run()) | Adding new CLI subcommands |
| **internal/app/app.go** | Main App struct, initialization, routes | Changing app structure |
| **internal/app/cli.go** | CLI dispatcher (checks for backup, passwordhash) | Adding subcommands |
| **internal/app/backup_cmd.go** | Backup subcommand implementation | Modifying backup behavior |
| **go.mod, go.sum** | Go dependencies | Adding packages |
| **Dockerfile** | Builds static binary, copies assets | Deployment/build issues |
| **docker-compose.yml** | Container orchestration (app + Caddy) | Changing port, env vars, services |
| **Caddyfile** | Reverse proxy config | TLS, cert, hostname setup (A2) |

### 🎯 HTTP Handlers & Middleware
| File | Purpose | Touch When |
|------|---------|-----------|
| **internal/handlers/middleware.go** | Session validation, CSRF guard, auth exemptions | Auth/security work, A2/A1 |
| **internal/handlers/auth.go** | Login/logout (currently no-op; A3/A4 later) | Auth work (deferred) |
| **internal/handlers/health_view.go** | Presentation: HealthMetricView, buildHealthTodayView | M3 integration, UI work |
| **internal/handlers/health_api.go** | `/api/health/sync` endpoint (write-only) | M3: reading Shortcut data |
| **internal/handlers/health_plan.go** | `/health/plan` view (Day view, M4) | Display/rendering work |
| **internal/handlers/** | All other routes (calendar, today, projects, etc.) | Feature/page changes |

### 🧮 Business Logic
| File | Purpose | Touch When |
|------|---------|-----------|
| **internal/services/health.go** | Health calculations (burn, energy, macros) | M3: storing observations |
| **internal/services/auth.go** | VerifyPassword, hashSessionToken (A3/A4 later) | Auth work (deferred) |
| **internal/repository/**.go** | Database queries (sessions, snapshots, etc.) | Query/data access work |

### 🎨 Web & Static Assets
| File | Purpose | Touch When |
|------|---------|-----------|
| **web/templates/health.html** | Day view rendering (never 0 for missing data) | M3, display work |
| **web/templates/login.html** | Login form (stub, auth deferred) | A3/A4 (later) |
| **web/templates/base.html** | HTML skeleton, navigation | Layout/navigation changes |
| **web/static/css/** | Plain CSS (no build system) | Styling, responsive work |
| **web/static/js/** | Vanilla JS modules (no framework) | Interactivity, UX work |

### 🌱 Seeds & Examples
| File | Purpose | Touch When |
|------|---------|-----------|
| **seeds/example.sql** | Example data (loaded once on empty DB) | Adding demo data |

### 🔧 CI/CD
| File | Purpose | Check When |
|------|---------|-----------|
| **.github/workflows/check.yml** | CI pipeline (gofmt, vet, build, test, race) | CI failures |

---

## HOW TO UPDATE TASK STATUS — Step by Step

### When You Complete a Task

**1. Update the task file:**
```bash
# Example: finishing A2
vim docs/tasks/A2-trusted-tls.md
```
Change:
```markdown
- **Status:** `IN PROGRESS`
```
To:
```markdown
- **Status:** `DONE`
- **Implementation commit(s):** `abc1234`, `def5678` (your commit SHAs)
- **Review status:** CI green — run 12345678
```

**2. Update the state table in AI_HANDOFF.md:**
```bash
vim docs/AI_HANDOFF.md
```
Find the task table (section 3). Change:
```markdown
| [A2 trusted TLS](tasks/A2-trusted-tls.md) | IN PROGRESS | — | — |
```
To:
```markdown
| [A2 trusted TLS](tasks/A2-trusted-tls.md) | DONE | `abc1234` | green |
```

**3. Update the "Next action" section in AI_HANDOFF.md:**
- Remove finished task from current action
- Add next task to current action
- Record date and what was done

Example:
```markdown
## 8. Next action

✅ A2 Trusted TLS completed (commit abc1234, CI green)

🔄 Next: Implement A1 — Remove plaintext port
```

**4. Commit and push:**
```bash
git add docs/tasks/A2-trusted-tls.md docs/AI_HANDOFF.md
git commit -m "docs: A2 completed and next action set

Implemented Tailscale MagicDNS cert in Caddyfile with 90-day renewal.
Tested from iPhone Shortcut — TLS validation passes.
A1 now unblocked (requires A2 complete).

Commits: abc1234, def5678
CI green: run 12345678"

git push origin feature/health-hardening
```

---

## THE THREE RULES THAT MATTER MOST

1. **Git is proof.** Not committed? Not pushed? Not in CI? Not done.
2. **NULL ≠ 0.** Never render missing data as zero. Keep them distinct.
3. **Never weaken a constraint.** FK enforcement, tests, Health API rules — trust them.

**Full rules:** See `docs/RULES.md` (read before every commit).

---

## YOUR FIRST THREE TASKS (In Order)

### 1️⃣ A2 — Trusted TLS (Caddyfile update)
- **Why:** Apple Shortcuts validates TLS; self-signed fails silently
- **What to do:** See `docs/tasks/A2-trusted-tls.md`
- **Files to touch:** `Caddyfile` (cert path, renewal), `docs/tasks/A2-trusted-tls.md`
- **Test:** iPhone Shortcut `Get Contents of URL` must succeed
- **Unblocks:** A1

### 2️⃣ A1 — Remove plaintext port (docker-compose.yml)
- **Why:** Only HTTPS is safe for auth and health data
- **What to do:** See `docs/tasks/A1-remove-plaintext-port.md`
- **Files to touch:** `docker-compose.yml` (remove port 3005), `Caddyfile` (HSTS header)
- **Test:** `curl http://localhost:3005` should fail; `curl https://...` should work
- **Unblocks:** M3

### 3️⃣ M3 — Apple Health Shortcut ingestion
- **Why:** Health data flows from Apple Shortcuts into the database
- **What to do:** See `docs/tasks/M3-health-ingestion.md` (create this task file)
- **Files to touch:** 
  - `internal/handlers/health_api.go` (extend `/api/health/sync`)
  - `internal/repository/` (add queries for observations)
  - `internal/services/health.go` (calculations)
  - `web/templates/health.html` (display real data)
- **Test:** Shortcut sends data → database stores it → Day view shows it
- **Unblocks:** Nothing (end of M2.5, auth deferred to later)

**After those three:** A3/A4 (Authentication) in the next phase.

---

## ARCHITECTURE GUARDRAILS

### Database
- Foreign keys enforced every connection (PRAGMA foreign_keys=1)
- 10 `ON DELETE CASCADE` + 13 `ON DELETE SET NULL` rules
- Migrations: additive only, never edited after applied
- Backup tool: `docker compose exec gyst-lifeops /app/lifeops backup`

### Health Calculations (Critical!)
- **Never sum:** Watch steps + phone steps (sources overlap)
- **Never double-count energy:** Resting + Active only
- **Never project:** Show only observed data
- **NULL ≠ 0:** "Not synced" ≠ "observed zero"

### Security
- CSRF guard: Sec-Fetch-Site + Origin validation
- Session cookie: Secure/HttpOnly/SameSite=Lax
- TLS: Tailscale MagicDNS cert (90-day renewal needed)
- Auth: Currently none; A3/A4 deferred (stub in code)

---

## COMMON TASKS

### Run tests locally
```bash
go test ./...
go test -race ./...  # Catch concurrency bugs
```

### Build the binary
```bash
go build -o ./lifeops .
```

### Start development server
```bash
docker compose up -d
docker compose logs -f gyst-lifeops
curl http://localhost:3005
```

### Stop everything
```bash
docker compose down
```

### Push a commit
```bash
git add <files>
git commit -m "message"
git push origin feature/health-hardening
```

### Check CI
After every push, GitHub Actions runs:
- gofmt (code format)
- go vet (static analysis)
- go build (build)
- go test (unit tests)
- go test -race (race detector)
- Migration uniqueness check
- No-DB-file-committed check

Watch at: https://github.com/JayDalool/gyst-lifeops/actions

---

## IF YOU GET STUCK

1. **Check the rules:** `docs/RULES.md` (most failures violate one)
2. **Check the task file:** `docs/tasks/A2-trusted-tls.md` (acceptance criteria)
3. **Check the code:** Full files (not excerpts) via `cat internal/handlers/middleware.go`
4. **Run tests locally:** `go test -run TestName -v` (see exact failure)
5. **Check git history:** `git log --oneline -- <file>` (who changed what)

---

## YOU HAVE EVERYTHING

- ✅ Branch ready: `feature/health-hardening`
- ✅ Docs complete: AI_HANDOFF.md, RULES.md, task files, CODEX_PROMPT.md
- ✅ Code tested: All tests pass locally, CI green
- ✅ Deployment live: Port 3005 (app), 3443 (Caddy/HTTPS)
- ✅ Operator steps done: OPS1 (FK audit ✅), Tailscale HTTPS ✅

**Start with A2. Push to `feature/health-hardening`. Update docs. Move to A1. Then M3. Done.**

The next agent will read this same document. Leave breadcrumbs in the docs.

---

## QUICK LINK REFERENCE

- Current state: [`docs/AI_HANDOFF.md`](docs/AI_HANDOFF.md) section 3
- All tasks: [`docs/tasks/README.md`](docs/tasks/README.md)
- Rules: [`docs/RULES.md`](docs/RULES.md)
- First task: [`docs/tasks/A2-trusted-tls.md`](docs/tasks/A2-trusted-tls.md)
- Full architecture: [`AGENTS.md`](AGENTS.md)
- Detailed workflow: [`CODEX_PROMPT.md`](CODEX_PROMPT.md)

---

**Go. You've got this. 🚀**
