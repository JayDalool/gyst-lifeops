# GYST — Claude / Codex shared handoff

**Last updated:** 2026-09-12
**Updated by:** Claude
**Repository:** `JayDalool/gyst-lifeops`
**Implementation branch:** `feature/health-hardening`
**Implementation HEAD at time of writing:** `12fabb6` (see the log below)

Read this at the start of every session, then read the task ledger. **Git is the
source of truth.** If this file disagrees with the repository, the repository
wins: correct the file, never rewrite history to match it.

→ **[Task ledger: `docs/tasks/`](tasks/README.md)** — one file per work item,
with commits, review status, acceptance criteria and next action. Per-task
detail lives there; this file is the map.

→ **[Working rules: `docs/RULES.md`](RULES.md)** — the rules every agent follows
and the failure that produced each one. Read it before committing anything.

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

**The rule is now inverted for these paths:** `docs/AI_HANDOFF.md`,
`docs/tasks/` and `docs/RULES.md` live on the implementation branch, because that is where work
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
| `feature/health-hardening` | `12fabb6` | **Current implementation branch.** |
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
5341e46  docs — ledger corrected after M4 was committed
ccec84a  docs — working rules, REV1/DOC1/B2 task files, M4 CI result
820c704  B2 — lifeops backup, using VACUUM INTO
12fabb6  docs — B2 recorded; cp-based backup advice replaced
```

**Do not rewrite, squash, force-push or rebase these commits.**

---

## 3. State at a glance

Full detail, including acceptance criteria, in [`docs/tasks/`](tasks/README.md).

| Task | Status | Commit(s) | CI |
|---|---|---|---|
| [UI-PLAN design system](tasks/UI-PLAN-design-system.md) | DONE | `0145c9c` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34668589837) |
| [UI-9 Health Plan](tasks/UI-9-plan.md) | DONE | `694e5b7` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34778983272) |
| [UI-8 Progress](tasks/UI-8-progress.md) | DONE | `17c227b` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34774519741) |
| [UI-7 Training](tasks/UI-7-training.md) | DONE | `63d33f5` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34739294360) |
| [UI-6 Nutrition](tasks/UI-6-nutrition.md) | DONE | `f3a18f7` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34738919673) |
| [UI-5 Health summary](tasks/UI-5-health.md) | DONE | `a6ec19f` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34725554881) |
| [UI-4 Mobile Galaxy](tasks/UI-4-galaxy.md) | DONE | `0fea4fe` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34709626557) |
| [UI-3 Projects](tasks/UI-3-projects.md) | DONE | `6c86252` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34708681737) |
| [UI-2 Today](tasks/UI-2-today.md) | DONE | `3dd076d` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34671487875) |
| [UI-1 shell](tasks/UI-1-shell.md) | DONE | `a8f9105` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34670549071) |
| [UI-0 foundation](tasks/UI-0-foundation.md) | DONE | `99f5968` | [green](https://github.com/JayDalool/gyst-lifeops/actions/runs/34669485151) |
| [B1 SQLite PRAGMA correctness](tasks/B1-sqlite-pragma-correctness.md) | DONE | `fb4f1eb` | none — predates CI |
| [B1-FIX review fixes](tasks/B1-FIX-review-fixes.md) | DONE | `a1dc3da`, `29adc43` | none — predates CI |
| [B6 CI safety gate](tasks/B6-ci-safety-gate.md) | DONE | `f663f3f` | green |
| [A3 session store](tasks/A3-session-store.md) | DEFERRED (\*) | `7835b7d` | green |
| [A4 application auth](tasks/A4-application-auth.md) | DEFERRED (\*) | `fb05cc3` | green |
| [A5 Health API write-only](tasks/A5-health-api-write-only.md) | DONE | `18196e9` | green |
| [A2 trusted TLS](tasks/A2-trusted-tls.md) | IN PROGRESS | — | — |
| [A1 remove plaintext port](tasks/A1-remove-plaintext-port.md) | BLOCKED (on A2) | — | — |
| [OPS1 production FK audit](tasks/OPS1-production-fk-audit.md) | DONE (\*\*) | (manual) | n/a — operator |
| [B2 backup tooling](tasks/B2-backup-tooling.md) | DONE | `820c704` | green |
| [M4 live Day view](tasks/M4-live-day-view.md) | DONE | `60be715` | green |
| [REV1 B1 independent review](tasks/REV1-b1-independent-review.md) | DONE | findings in `29adc43` | n/a — review |
| [DOC1 documentation](tasks/DOC1-documentation.md) | DONE | `d4007a3`, `5341e46` | green |
| [M3 Apple Health ingestion](tasks/M3-health-ingestion.md) | PLANNED | — | — |

(\*) **A3/A4 deferred:** Auth is documented as complete but not deployed. Will be implemented after A2/A1/M3 ship. Currently no login required.  
(\*\*) **OPS1 completed:** FK audit ran on 2026-09-12, returned zero violations. Database certified clean.

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
- ~~No backup tooling exists.~~ **Resolved by [B2](tasks/B2-backup-tooling.md)**
  (`820c704`): `docker compose exec gyst-lifeops /app/lifeops backup`. Run it
  before any deployment or migration. **Do not back up with `cp` —** the
  database is in WAL mode and the main file alone can be near-empty.
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

The rules live in **[`docs/RULES.md`](RULES.md)**, with the failure that
produced each one recorded next to it. The two that govern every session:

**Before every task:** read this file, [`RULES.md`](RULES.md) and
[`docs/tasks/`](tasks/README.md); verify branch and HEAD; confirm the task is
not already done; check `git status`. Do not rely on chat history, and do not
rely on an agent's internal task list — neither is visible to the other agent or
to Jay.

**After every task:** update the task file and this file's table in the same
commit as the work or the one immediately after; record the commit SHA and CI
result; record blockers; set exactly one next action. **Do not claim work is
complete unless Git proves it.**

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

**Jay: review UI-9 Health Plan before UI-10 Courses.**
Jay approved UI-2 and requested continuation. See [UI-3](tasks/UI-3-projects.md)
for scope and verification evidence. A2 remains separate operational work.

Planning baseline is `5c50011`. The earlier handoff contains contradictory
operator/auth claims: newer prose says HTTPS/OPS1 are done, older blocker sections
say otherwise, and auth code exists despite being called unimplemented. These
historical claims have not been reverified in this planning task. Consult the
screen map's discrepancy notes before resuming A2; do not redeploy or reimplement
auth based only on the stale prose.

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

2026-09-12 — Claude — Audited every commit since 3083fb9 against the ledger.
                      Two had no task file (REV1, DOC1); both written. M4 CI
                      confirmed green (run 34665218837) and its PENDING status
                      corrected. Added docs/RULES.md. B2 opened as the next
                      agent task. Documentation only.      Commit: ccec84a

2026-09-12 — Claude — B2: lifeops backup, using VACUUM INTO. Source opened
                      read-only; copy verified with integrity_check before
                      success is reported. Measured the failure it replaces: a
                      live database whose main file was 4 KB while its -wal
                      held 1.28 MB, so cp would have produced a near-empty
                      backup silently. OPS1's procedure updated to use it.
                                             Commits: 820c704, 12fabb6
                      CI green: runs 34665946555 and 34666017378.

2026-09-12 — Claude — AGENTS.md: comprehensive handoff document for the next
                      agent (Codex). Teaches the complete project context:
                      architecture, rules, security guardrails, file
                      organization, workflow, mandate, and how to continue
                      from where Claude left off. Use this to onboard and
                      transition to the next agent.      Commit: 2bb4c83

2026-09-12 — Claude — OPS1 operator task COMPLETE: FK audit passed (zero
                      violations). Tailscale HTTPS enabled. Deployment live
                      on port 3005. A3/A4 (auth) documentation reviewed;
                      code not present in deployed version. Decision: defer
                      A3/A4 auth implementation to after A2/A1/M3. Update
                      ledger accordingly. Create CODEX_PROMPT.md with
                      detailed handoff for next agent to implement A2 → A1
                      → M3.                           Commits: 9cd2c2c, 9a5aefd
```

2026-09-11 — Codex — Inspected both approved design boards and implementation
source at 5c50011. Prepared UI_SYSTEM.md and UI_SCREEN_MAP.md with exact UI-0/UI-1
files, data gaps and preservation requirements. UI-PLAN awaits Jay's approval.
No runtime changes or production access. Commit: 0145c9c. CI green: run 34668589837. Local format/vet/build/unit/race checks passed. Approval remains pending.

2026-09-11 — Codex — Jay approved UI-0. Added the opt-in component foundation,
Lucide subset and standalone development gallery. Chromium/WebKit review passed;
real iPhone review pending. Commit: 99f5968; CI green: run 34669485151.

2026-09-11 — Codex — Jay liked UI-0 and approved UI-1. Implemented the GYST
shell, More, five SVG navigation links, theme ownership and private-safe offline
behavior. Verification and commit evidence recorded in UI-1 task; no production
deployment or data access.

2026-09-11 — Codex — Jay approved UI-1 and requested the next phase. UI-2
adopts Today and adds the shared daily checklist route. Existing task/study,
prayer/Athan/trackers and XP actions remain. Health uses existing read services;
no business logic/schema/production data changes. Evidence: UI-2 task file.

2026-09-11 — Codex — Jay requested continuation after UI-2. UI-3 adopts Projects
cards, status/roadmap filters, detail sections and forms while preserving
hierarchy, proof, external work and action guards. Evidence: UI-3 task file.
No production access or deployment.

2026-09-12 — Codex — Resumed UI-3 with Jay’s approval. Implementation committed
as 6c86252; CI green: run 34708681737. Local format/vet/build/unit/race and
Chromium/WebKit checks passed. Physical iPhone review remains pending.
Next action: Jay reviews UI-3 before approving UI-4. No production deployment.

2026-09-12 — Codex — Jay authorized UI-4 while testing UI-3. Mobile Galaxy
gestures, selection panel and category navigator committed as 0fea4fe. CI green:
run 34709626557. Chromium/WebKit and local Go/Node checks passed; real iPhone
review pending. No production access or deployment. Next: Jay reviews UI-4
before approving UI-5 Health summary.

2026-09-12 — Codex — Jay requested continuation and identified Health sizing.
UI-5 committed as a6ec19f, CI green: run 34725554881. Phone/landscape/desktop,
200% text, dark theme and gym interaction checks passed in Chromium/WebKit.
Local Go and Node suites passed. No production deployment; physical device
review pending. Next: Jay reviews Health sizing before UI-6 Nutrition.

2026-09-12 — Codex — UI-6 Nutrition detail committed as f3a18f7; CI green:
run 34738919673. Local Go/Node suites and Chromium/WebKit empty/populated
responsive checks passed. No production deployment. Physical device review
pending. Next: Jay reviews UI-6 before UI-7 Training. Shortcut setup remains
with Jay’s other conversation; authenticated connection test was user-reported,
not evidence that actual health measurements have synced.

2026-09-13 — Codex — Jay requested UI-7. Training onboarding and shared gym
check-ins committed as 63d33f5; CI green: run 34739294360. Local Go/Node and
Chromium/WebKit responsive, large-text, dark-mode and persistent tracker checks
passed. No production deployment. Physical device review pending. Next: Jay
reviews UI-7 before UI-8 Progress. Hevy remains unimplemented.

2026-09-13 — Codex — Continued the unfinished UI-8 Progress work. Recorded weight
history, date ranges and accessible chart/table committed as 17c227b. CI green:
run 34774519741. Local Go/Node checks and Chromium/WebKit responsive checks
passed using a temporary fixture database. Physical iPhone review pending.
No production deployment. Next: Jay reviews UI-8 before UI-9 Plan.

2026-09-13 — Codex — Jay requested UI-9. Grouped Health Plan editor and saved
cycle cards committed as 694e5b7. CI green: run 34778983272. Local Go/Node
checks and Chromium/WebKit responsive create/edit/error-recovery checks passed
using a temporary fixture database. No production deployment. Physical iPhone
and software-keyboard review pending. Next: Jay reviews UI-9 before UI-10 Courses.
