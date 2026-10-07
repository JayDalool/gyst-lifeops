# AGENTS.md — GYST development governance

Every coding agent (Claude Code, Codex, others) reads this file before any work
in this repository. It is deliberately short. Detail lives in the linked files.

**GYST** is a single-user, self-hosted LifeOps PWA: Go single binary, SQLite
(WAL, enforced foreign keys), server-rendered templates, vanilla JS, plain CSS,
Three.js Galaxy, Docker Compose + Caddy. Jay is the owner, operator and the only
person who merges and touches production.

## Who does what

| Role | Is |
|---|---|
| GitHub Issue | The canonical record of a development task |
| Task branch | The implementation workspace |
| Pull request | The implementation artifact |
| Claude / Codex | Implementation and review workers |
| GYST (later) | A control surface over the same Issues |
| Jay | Final verification and merge |

There is **one** editable task system: GitHub Issues. `docs/tasks/` is a
historical ledger (see its README). The vNext backlog
(`docs/audit/vnext/15_TASK_BACKLOG.md`) is the planning source; once an item
becomes an Issue, the Issue is the live record.

## Rules for every task

1. **Read the GitHub Issue completely**, including acceptance criteria, dependencies, out of scope and risk class.
2. **Read the rules it points to:** [`docs/RULES.md`](docs/RULES.md) always; the referenced design, security and decision documents (for example `docs/audit/vnext/05A`, `08A`, `10A`, `13A`, `17A`).
3. **One Issue → one task branch → one focused PR.**
4. **Branch name:** `task/<issue-number>-<short-slug>` (for example `task/12-login-throttling`), created from **`master`**. PRs target `master` unless the Issue explicitly says otherwise.
5. **Do not expand scope silently.** If the Issue is wrong or incomplete, say so in the Issue/PR and stop or ask. Don't widen the change.
6. **No production changes** unless the Issue explicitly defines a human-controlled operational step. Agents never run commands against the production database, host services, GitHub/Cloudflare/n8n settings or credentials.
7. **Database/schema changes** need explicit acceptance criteria, a rollback consideration and migration review. Migrations are additive, timestamp-prefixed and never edited once applied (`docs/RULES.md` §2.5, §3.1).
8. **Run the required tests:** `gofmt`, `go vet`, `go build ./...`, `go test ./...`, `go test -race ./...`, plus any check the Issue names (for example `node scripts/check-ui-shell.cjs`, `node scripts/check-galaxy.cjs`).
9. **Browser-test UI work** at phone and desktop widths, in light and dark. Major mobile screens also need a real iPhone/PWA check where the Issue asks for it. Never claim browser verification you didn't do.
10. **Report limitations honestly:** what wasn't run, wasn't verified, or failed, with the output.
11. **Code existing is not Done.** Done means the acceptance criteria are verified, reviewed and accepted by Jay (then `status:done` / closed).
12. **Independent review** by a different agent/provider where practical is required before Jay's final acceptance whenever the Issue's risk class or reviewer field asks for it. ELEVATED and PRIVILEGED always need it. A self-review is not independent.
13. **Agents never merge into `master`** (GitHub branch protection isn't available on the repository's current plan, so this rule is enforced by process until GYST-VN-099 is re-planned), never force-push, never rewrite shared history, never change branch protection, secrets or repository settings.
14. **PRs reference Issues with `Refs #<issue>`**, never `Closes`/`Fixes`. Jay closes the Issue after verification.
15. **Secrets never enter the repository**, logs, Issues, PRs or screenshots.

Workflow detail: [`docs/agents/IMPLEMENTATION_WORKFLOW.md`](docs/agents/IMPLEMENTATION_WORKFLOW.md) ·
review: [`docs/agents/REVIEW_WORKFLOW.md`](docs/agents/REVIEW_WORKFLOW.md) ·
labels and Issue lifecycle: [`docs/agents/README.md`](docs/agents/README.md).

## Shared tools (jay-dev plugin)

This file and `docs/RULES.md` win over the plugin wherever they differ.

- `/jay-dev:studio [quick|standard|deep|exhaustive] <request>` picks the level
  and the specialists; code changes still follow the rules above (Issue →
  task branch from `master` → PR → independent review → Jay merges).
- `jay-dev:qa` for browser checks: local dev with `seeds/example.sql`, never
  production health data; light and dark as rule 9 says. The one exception
  (Jay, 2026-10-06): when Jay asks in the session to check his real data,
  use a read-only throwaway copy of the data, never the live database. Read
  text output only, with no screenshots. Delete the copy afterwards. Record no
  real values in commits, tests, comments, prototypes or PR and issue bodies.
- `jay-dev:security` for anything touching auth, the session cookie, the
  bearer-token Health sync (`/api/health/*`), web push or uploads.
- `jay-dev:db` for migrations (additive, timestamp-prefixed, never edited).
- `jay-dev:design-review` and the `frontend-design` skill work inside
  `13A_DESIGN_DECISIONS.md`: no new framework, spacing, colours or patterns.
  Their motion rules never override the Galaxy rules below.
- `jay-dev:ui-tests` keeps Jay's two test files, which stay the place for
  his checks: screens to check in `docs/jay/UI_TESTS.md`, steps only he can
  run in `docs/jay/MY_ACTIONS.md`. Add the line in the same PR as the work;
  `needs:jay` marks an Issue waiting on him.
- Paths the skills must use here: the design system is
  `docs/audit/vnext/13A_DESIGN_DECISIONS.md` (never create
  `docs/design/DESIGN_SYSTEM.md`).

## Non-negotiable product rules

- **GYST Galaxy is a protected signature feature. Agents may enhance and refactor it but must not delete, replace, or downgrade the real Three.js Galaxy without explicit product-owner approval.**
  - The Projects list and Galaxy **coexist**: the list is for working, Galaxy is the signature spatial view. Neither replaces the other.
  - Stage 7 prototype markup (`docs/audit/vnext/prototypes/`) is a **design reference**, never a replacement implementation.
  - Every Galaxy task follows [`docs/audit/vnext/17A_GALAXY_PRESERVATION.md`](docs/audit/vnext/17A_GALAXY_PRESERVATION.md), including its regression checklist.
  - **GYST-VN-143** establishes the browser and performance baseline. No material Galaxy change starts before it is done.
- **Health data:** `NULL` ≠ `0`; never render missing as zero; never sum Watch and phone steps; observed burn = resting + active only; never convert steps to calories (`docs/RULES.md` §3).
- **Design:** `13A_DESIGN_DECISIONS.md` > `11`/`12`/`13` > prototypes. Don't invent spacing, radii, colours, navigation or component patterns; report conflicts. Violet is identity only, never a state colour. No new UI framework, build system or component library.
- **Security:** the risk class on an Issue is informative. Tasks labelled `risk:not-agent-eligible` are human-only. Later automation re-derives risk and never trusts a label alone (`docs/audit/vnext/09_SECURITY_FOUNDATION.md` §J).

## Canonical trunk

**`master`** is the canonical development trunk and the GitHub default branch
(decided in GYST-VN-001, Issue #2). Task branches start from `master`, PRs
target `master`, and only Jay merges. `feature/health-hardening` was merged into
`master` (PR #10) and is historical.

## When in doubt

Stop and ask Jay in the Issue, with your recommendation. Git and CI are the
evidence; if a document disagrees with the repository, the repository wins and
the document gets corrected.
