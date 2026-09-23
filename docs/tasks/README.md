# GYST task ledger

One file per work item. The ledger is the shared record between Claude, Codex
and Jay, so it lives on the implementation branch where the work happens.

The rules that govern how it is kept — and how work is done at all — are in
[`../RULES.md`](../RULES.md).

## The rule this ledger exists to enforce

**Git is the only proof of completion.** A task is DONE when a commit on
`feature/health-hardening` implements it. Anything not provable from Git or a
CI run is marked `UNKNOWN` or `PENDING` — never assumed, never inferred from a
conversation nobody else can read.

Commit messages are evidence of intent, not of outcome. Where a message claims
something CI can check, the CI run is cited instead.

## Status vocabulary

| Status | Meaning |
|---|---|
| `DONE` | A commit on the implementation branch implements it. |
| `IN PROGRESS` | Started; not committed. Not provable from Git yet. |
| `NOT STARTED` | No commit, no working-tree change. |
| `BLOCKED` | Cannot proceed until a named blocker clears. |
| `PENDING` | Waiting on an action outside this repository (usually the operator). |
| `UNKNOWN` | Cannot be established from Git or CI. |

## CI evidence

CI was added in `f663f3f`. **Commits before it have no CI run and never will** —
their `review status` says so rather than implying they went unchecked.

Workflow: `.github/workflows/check.yml` — gofmt, go vet, go build, go test,
go test -race, migration-prefix uniqueness, no-database-file-committed.

## Files

| Task | Status |
|---|---|
| [UI-PLAN — UI design system and screen map](UI-PLAN-design-system.md) | DONE |
| [NAV1 — Navigation cleanup](NAV1-navigation-cleanup.md) | IN PROGRESS |
| [UI-10 — Courses](UI-10-courses.md) | DONE |
| [UI-9 — Health Plan](UI-9-plan.md) | DONE |
| [UI-8 — Progress](UI-8-progress.md) | DONE |
| [UI-7 — Training](UI-7-training.md) | DONE |
| [UI-6 — Nutrition detail](UI-6-nutrition.md) | DONE |
| [UI-5 — Responsive Health summary](UI-5-health.md) | DONE |
| [UI-4 — Mobile Galaxy](UI-4-galaxy.md) | DONE |
| [UI-3 — Projects and detail](UI-3-projects.md) | DONE |
| [UI-2 — Today and checklist](UI-2-today.md) | DONE |
| [UI-1 — App shell and navigation](UI-1-shell.md) | DONE |
| [UI-0 — Reusable foundation](UI-0-foundation.md) | DONE |
| [B1 — SQLite PRAGMA correctness](B1-sqlite-pragma-correctness.md) | DONE |
| [B1-FIX — B1 review fixes](B1-FIX-review-fixes.md) | DONE |
| [B6 — CI safety gate](B6-ci-safety-gate.md) | DONE |
| [A3 — Session store and auth service](A3-session-store.md) | DONE |
| [A4 — Application authentication](A4-application-auth.md) | DONE |
| [A5 — Health API write-only](A5-health-api-write-only.md) | DONE |
| [A1 — Remove plaintext port](A1-remove-plaintext-port.md) | IN PROGRESS |
| [A2 — Trusted TLS for Shortcuts](A2-trusted-tls.md) | IN PROGRESS |
| [OPS1 — Production foreign_key_check](OPS1-production-fk-audit.md) | PENDING |
| [B2 — Backup tooling](B2-backup-tooling.md) | DONE |
| [M4 — Live Health Day view](M4-live-day-view.md) | DONE |
| [REV1 — Independent review of the B1 range](REV1-b1-independent-review.md) | DONE |
| [DOC1 — Project documentation](DOC1-documentation.md) | DONE |

## Updating

When a task's state changes, edit its file and the row above, then say what
changed in [`../AI_HANDOFF.md`](../AI_HANDOFF.md) — in the same commit as the
work, or in the commit immediately after. If the ledger disagrees with Git, Git
wins: correct the ledger, never rewrite history to match it.
