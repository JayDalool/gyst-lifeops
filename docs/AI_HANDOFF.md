# GYST — Claude / Codex shared handoff

**Last updated:** 2026-09-12
**Updated by:** Claude
**Repository:** `JayDalool/gyst-lifeops`
**Implementation branch:** `feature/health-hardening`
**Implementation HEAD at time of writing:** `60be715ebc01234080adafdbdee00dbc003d8e32`

Read this at the start of every session, then read the task ledger. **Git is the
source of truth.** If this file disagrees with the repository, the repository
wins: correct the file, never rewrite history to match it.

→ **[Task ledger: `docs/tasks/`](tasks/README.md)** — one file per work item,
with commits, review status, acceptance criteria and next action. Per-task
detail lives there; this file is the map.

---

## Why this file moved to the implementation branch

It was previously created only on `docs/ui-v3-standards`, and merges ran **one
way** — implementation branch into docs branch — so these commits never reached
`feature/health-hardening`:

```
ce91119  docs: add UI v3 standards and shared AI handoff
c18cb5b  docs: record B1 findings fix and set the next task to the FK audit
8c39e3d  docs: correct the branch-state note after the merge
b7f4492  docs: record the CI safety gate
```

None is an ancestor of `feature/health-hardening` (verified with
`git merge-base --is-ancestor`). The one-way rule did what it was written to do —
keep documentation commits out of the code branch — but it also meant **anyone
working on the implementation branch, including Codex, could not see the handoff
at all**, which defeats the file's entire purpose. `docs/tasks/` had never
existed on any branch.

**The rule is now inverted for these two paths:** `docs/AI_HANDOFF.md` and
`docs/tasks/` live on the implementation branch, because that is where work
happens and where an agent will look. `docs/design/v3/` remains on
`docs/ui-v3-standards`. The copy of this file on that branch is superseded.

---

## 1. Project

GYST is a self-hosted LifeOps system: **Go + SQLite**, server-rendered Go
templates, plain CSS, small vanilla JS modules, Docker behind Caddy.
**Mobile-first PWA (iPhone) is a primary target.**

Areas: Today · Projects · Galaxy · Health · Courses · Reminders and supporting tools.

The Health architecture combines GYST with Apple Health, Apple Watch,
MyFitnessPal (routed through Apple Health), Apple Shortcuts ingestion, and Hevy
Pro for strength training.

> **Hevy is PLANNED, not implemented.** No Hevy code, tables or provider exists.

External apps are **providers**, not the source of truth for architecture.

---

## 2. Branch map

Verified with `git ls-remote` on 2026-09-12.

| Branch | HEAD | Meaning |
|---|---|---|
| `master` | `ae5e674` | Mainline. Contains **no** Health work. |
| `feature/health-dashboard-foundation` | `055f098` | M1 — UI shell. |
| `feature/health-core` | `3083fb9` | M2 — Health Core. B1 baseline. |
| `feature/health-hardening` | `60be715` | **Current implementation branch.** |
| `docs/ui-v3-standards` | `b7f4492` | UI v3 design reference. Carries a superseded copy of this file. |

```
3083fb9  Health baseline (feature/health-core)
fb4f1eb  B1 — SQLite PRAGMAs on pooled connections
a1dc3da  B1 review fix — seeded integrity audit, file: URI support
29adc43  B1 findings H1/M1/M2
f663f3f  CI safety gate
7835b7d  A3 — session store and auth service
fb05cc3  A4 — application authentication
18196e9  A5 — Health API write-only
d4007a3  docs — task ledger and handoff moved onto this branch
60be715  M4 — Health Day view reads real observations
```

**Do not rewrite, squash, force-push or rebase these commits.**

---

## 3. State at a glance

Full detail, including acceptance criteria, in [`docs/tasks/`](tasks/README.md).

| Task | Status | Commit(s) | CI |
|---|---|---|---|
| [B1 SQLite PRAGMA correctness](tasks/B1-sqlite-pragma-correctness.md) | DONE | `fb4f1eb` | none — predates CI |
| [B1-FIX review fixes](tasks/B1-FIX-review-fixes.md) | DONE | `a1dc3da`, `29adc43` | none — predates CI |
| [B6 CI safety gate](tasks/B6-ci-safety-gate.md) | DONE | `f663f3f` | green |
| [A3 session store](tasks/A3-session-store.md) | DONE | `7835b7d` | green |
| [A4 application auth](tasks/A4-application-auth.md) | DONE | `fb05cc3` | green |
| [A5 Health API write-only](tasks/A5-health-api-write-only.md) | DONE | `18196e9` | green |
| [A1 remove plaintext port](tasks/A1-remove-plaintext-port.md) | BLOCKED | — | — |
| [A2 trusted TLS](tasks/A2-trusted-tls.md) | BLOCKED (operator) | — | — |
| [OPS1 production FK audit](tasks/OPS1-production-fk-audit.md) | PENDING (operator) | — | — |
| [M4 live Day view](tasks/M4-live-day-view.md) | DONE | `60be715` | pending |

**Nothing above has been verified in a real browser, on a real device, or
against the production database.** CI green means the suite passes, not that the
application was used.

### Health features that do NOT exist

No AI tool layer. No foods, recipes or portion engine. No Hevy or training
tables. No progress analytics. No Apple Health Shortcut (M3). The Health Core
service layer exists, but `CreateHealthWeightEntry`, `ListHealthWeightEntries`
and `GetHealthSyncState` have no handler — there is no UI to record a weight or
see sync status.

---

## 4. Blockers and gates

- **The production database has not been certified** by `PRAGMA
  foreign_key_check`. Until it is, B1 is not production-safe even though the code
  is done. See [OPS1](tasks/OPS1-production-fk-audit.md).
- **No backup tooling exists** in the repository. B2 is not built. Back up by
  hand before any deployment or migration.
- **`GYST_PASSWORD_HASH` must be set** or nobody can sign in after A4 deploys.
- **Tailscale HTTPS is not enabled**, which blocks A2, therefore A1, therefore M3.
- **M3 must not begin** before the M2.5 correctness and security gates close.
- **UI v3 is design-only.** Not implemented.

---

## 5. Approved architectural decisions

Recorded so they stop being re-litigated every session.

### Database

SQLite stays. Additive migrations only — no destructive `DROP`, no broad
`UPDATE`. **Never reset or reseed the real `data/lifeops.db`.** Tests use
temporary databases. Foreign keys must actually be enforced. Do not edit an
applied migration: the runner tracks by filename with no checksum, so an edit is
silently ignored and a rename re-runs it and fails. New migrations use timestamp
prefixes (`YYYYMMDDHHMMSS_name.sql`).

### Health calculations

Never double-count energy. Do **not** add together Active Energy, step-derived
calories, and workout calories. Observed energy is **resting + active** only.
Steps are a separate activity metric, never converted to calories. Hevy's own
calorie estimates must never enter the energy engine.

Keep three concepts distinct: **food budget** (target − consumed), **observed
energy balance** (consumed − observed burn), and **projected balance** (a future
feature, always labelled projected).

### Nutrition

MyFitnessPal remains the food logger; Apple Health → Shortcut → GYST supplies
calories, protein, carbs, fat, fiber. Do not invent missing values. **Missing,
stale and current must remain distinguishable.** `NULL` means "not received";
`0` means "an observed zero". Never render missing data as zero.

### Steps

Store Watch and phone steps **separately**. **Never add them together** — the
sources overlap and summing double-counts. Watch is primary; phone is context.

### Training

Hevy is selected but unimplemented. Future shape:
`Hevy API → HevyProvider → normalized GYST training domain → SQLite → UI`.
Store generic `training_*` tables with `provider`/`provider_id`, never `hevy_*`,
and never bind the UI to raw Hevy JSON field names.

### AI

After deterministic services, not before. The assistant calls GYST functions for
nutrition, portions, energy balance, training and goals. It must **not** invent
calorie or macronutrient arithmetic.

---

## 6. Working rules for Claude and Codex

**Before every task:** read this file and [`docs/tasks/`](tasks/README.md);
verify branch and HEAD; confirm the task is not already done; check `git status`.
Do not rely on chat history, and do not rely on an agent's internal task list —
neither is visible to the other agent or to Jay.

**After every task:** update the task file and this file's table; record the
commit SHA and CI result; record blockers; set exactly one next action. **Do not
claim work is complete unless Git proves it.**

**Never** put secrets, tokens, API keys, passwords or bearer tokens in these files.

---

## 7. Do not do yet

- Do not implement Hevy.
- Do not redesign the production UI while hardening is mid-flight, unless asked.
- Do not implement several plan commits in one batch — one reviewable commit at a time.
- Do not edit applied migrations.
- Do not reset `data/lifeops.db`.
- Do not auto-repair production foreign-key violations.
- Do not replace the existing Galaxy without explicit approval.
- Do not remove existing functionality to match a mockup.
- Do not hard-code mockup sample values as production data.
- Do not weaken a foreign-key constraint, a test, or the Health API to make
  something else easier.

---

## 8. Next action

**One action, and it belongs to Jay, not to an agent:**

> Run [OPS1](tasks/OPS1-production-fk-audit.md) — `PRAGMA foreign_key_check`
> against the production database, with the app stopped and a backup taken
> first. Report the output. If it returns rows, stop and report them; repair
> nothing.

Two further operator steps unblock the rest: set `GYST_PASSWORD_HASH`, and
enable Tailscale HTTPS.

[M4](tasks/M4-live-day-view.md) is committed as of `60be715`; its CI run was
still pending when this was written. The Day view renders empty states until M3
ingestion exists.

---

## 9. Handoff log

Append-only, newest last.

```
2026-09-11 — Claude — Architecture review of the Health plan against the real code
                      on feature/health-core. Documentation only.

2026-09-11 — Claude — B1 SQLite PRAGMA correctness.        Commit: fb4f1eb
2026-09-11 — Claude — B1 review fixes (round one).         Commit: a1dc3da
2026-09-11 — Claude — Independent review of 3083fb9..a1dc3da: PASS WITH NOTES,
                      no blocking findings. Raised H1, M1, M2.
2026-09-11 — Claude — B1 findings H1/M1/M2 fixed.          Commit: 29adc43
2026-09-11 — Claude — CI safety gate; first run green.     Commit: f663f3f
2026-09-11 — Claude — A3 session store and auth service.   Commit: 7835b7d
2026-09-11 — Claude — A4 application authentication.       Commit: fb05cc3
2026-09-11 — Claude — A5 Health API write-only.            Commit: 18196e9
2026-09-11 — Claude — M4 Day view implemented and locally green.

2026-09-12 — Claude — Established docs/tasks/ and moved this handoff onto the
                      implementation branch, after verifying from Git that
                      neither had ever been present there. Backfilled ten task
                      files from Git and CI evidence only. Documentation only.
                      Commit: d4007a3

2026-09-12 — Claude — M4 committed after re-running the full suite and race
                      suite following a stash cycle. Ledger corrected: M4 was
                      recorded as uncommitted an hour earlier and that is no
                      longer true.                        Commit: 60be715
```
