# GYST — working rules for agents

> **Note (2026-09-27):** these rules stay in force. For new work, two sections are read with [`../AGENTS.md`](../AGENTS.md): **§5 Branches**, where work happens on `task/<issue>-<slug>` branches off `master`, merged into `master` by Jay via PR; and **§6 The ledger**, where the GitHub Issue is the ledger and `docs/tasks/` is historical. "One reviewable commit at a time" (§2.1) becomes one focused PR per Issue.

**Audience:** Claude, Codex, and any other agent that commits to this
repository. Jay is the operator and the only one who runs anything against the
production database.

These are not style preferences. **Every rule below exists because something
went wrong**, and the failure is named next to it so the rule can be argued with
rather than obeyed blindly. If a rule is wrong, say so and change it in a commit
— do not quietly ignore it.

Read this with [`AI_HANDOFF.md`](AI_HANDOFF.md) and [`tasks/`](tasks/README.md)
at the start of every session.

---

## 1. Evidence

### 1.1 Git is the only proof of completion

A task is DONE when a commit on the implementation branch implements it.
Anything not provable from Git or a CI run is `UNKNOWN` or `PENDING`. Never
infer completion from a conversation — the other agent and Jay cannot read it.

> *Why:* a full session of work existed only in chat. `docs/AI_HANDOFF.md` and
> the task ledger were not on the implementation branch at all, so anyone
> picking the branch up would have seen nothing.

### 1.2 A commit message is evidence of intent, not of outcome

Where a claim is something CI can check, cite the CI run instead of the message.

### 1.3 Verify before asserting — read the whole file

Never report an absence from a partial read. Open the actual range before
claiming a guard, a test or a branch does not exist.

> *Why:* a review claimed `/health/plan` had no CSRF protection. It did, at
> `internal/handlers/health_plan.go:136-149`, with a test. Lines 1–100 and
> 150–225 had been read; the guard was in the gap. ChatGPT caught it.

### 1.4 Measure before fixing

Before restructuring something because it is slow, hung or broken, get the
number. Re-reading a stale snapshot is not observation.

> *Why:* `go vet` was reported as having hung for ten minutes in CI and a
> workflow rewrite was nearly shipped. It had taken 83 seconds. The `sleep`
> calls meant to space out the polling were backgrounded and returned instantly,
> so the same API response was read repeatedly.

### 1.5 Probe the dependency, do not recall it

When behaviour depends on a library, driver or SQLite itself, write the probe
and run it. Memory of an API is not a source.

> *Why:* in-memory detection was made case-insensitive on the assumption that
> `:MEMORY:` and `mode=MEMORY` were accepted spellings. Probing SQLite showed
> `:MEMORY:` is an ordinary *file* and `mode=MEMORY` is "no such access mode".
> The code now mirrors SQLite's case-sensitivity exactly.

### 1.6 Negative-test every security property

A security control is not proven by a test that passes through it. Prove the
denial: unauthenticated requests rejected, cross-site POSTs rejected, orphan
inserts refused, open redirects refused.

> *Why:* the original PRAGMA defect was invisible to every passing test. Only a
> test that opened several pooled connections and attempted an orphan insert
> showed 3 of 4 accepting it.

---

## 2. Changing code

### 2.1 One reviewable commit at a time

One task, one commit, reviewed before the next begins. Do not batch plan items.

### 2.2 Never weaken a test, a constraint or an API to make something pass

Not a foreign key, not an assertion, not a validation. If a real constraint
breaks an old test, the test encoded the bug. Fix the cause or raise it — never
loosen the constraint.

### 2.3 Keep the fix minimal

Fix what the failure or the review comment needs, and nothing else. Do not widen
a change on your own initiative.

### 2.4 Do not blindly concatenate paths into URIs

A filesystem path is not a URI. Escape it, or do not build a URI from it.

### 2.5 Do not edit an applied migration

The runner tracks migrations by filename with **no checksum**: an edit to an
applied file is silently ignored, and a rename re-runs it and fails. New
migrations are additive and use timestamp prefixes (`YYYYMMDDHHMMSS_name.sql`).
No destructive `DROP`, no broad `UPDATE`.

### 2.6 Clean up after a failing test run

A failing run can leave artefacts behind — a stray `internal/db/file:` directory
or a `:MEMORY:` file. Check `git status` before committing; those must never be
committed. Tests use `t.TempDir()`.

---

## 3. Data

### 3.1 Never touch the real database

`data/lifeops.db` is never reset, reseeded, migrated ad hoc, or committed. Do
not auto-repair foreign-key violations found in production — report them and
stop. Only Jay runs anything against it, with the app stopped and a backup taken:
`docker compose exec gyst-lifeops /app/lifeops backup`
([B2](tasks/B2-backup-tooling.md)). One exception (Jay, 2026-10-06; see
`AGENTS.md`, `jay-dev:qa`): when Jay asks for a real-data check, an agent may
make one copy with `lifeops backup` for QA, outside `data/backups/`.

**Never back up with `cp`.** WAL mode means the main file alone can be
near-empty while the real data sits in `-wal`, and the copy looks fine.

### 3.2 `NULL` is not `0`

`NULL` means "not received". `0` means "an observed zero". Missing, stale and
current must remain distinguishable all the way to the screen. Never render
missing data as zero, and never invent a value to fill a gap.

### 3.3 Never sum Watch and phone steps

The sources overlap, so adding them double-counts. Store and display them
separately. Watch is primary; phone is context.

### 3.4 Never double-count energy

Observed burn is **resting + active** only. Do not add step-derived calories or
workout calories on top. Steps are an activity metric, never converted to
calories. Hevy's own calorie estimates never enter the energy engine.

### 3.5 Keep the three energy concepts separate

**Food budget** = target − consumed. **Observed energy balance** = consumed −
observed burn. **Projected balance** is a future feature and must always be
labelled projected. They have similar magnitudes, which is exactly why
conflating them is easy and harmful.

### 3.6 Providers are not the architecture

Apple Health, MyFitnessPal and Hevy are providers. Store generic tables with
`provider`/`provider_id`. Never name a table `hevy_*`, never bind the UI to a
provider's raw JSON field names.

---

## 4. Secrets

Never put a secret, token, API key, password, password hash or bearer token in
any file in this repository — documentation included. `GYST_PASSWORD_HASH` is
set in the environment by Jay and appears here only by name.

Only a SHA-256 hash of a session token is stored. The raw token exists only in
the `Secure`/`HttpOnly`/`SameSite=Lax` cookie.

---

## 5. Branches

| Path | Lives on |
|---|---|
| `docs/AI_HANDOFF.md`, `docs/tasks/`, `docs/RULES.md` | the implementation branch |
| `docs/design/v3/` | `docs/ui-v3-standards` |
| everything else | the implementation branch |

Handoff, ledger and rules live where the work happens, because that is where an
agent will look.

> *Why:* they were first created only on the docs branch, and merges ran one way
> — implementation into docs. The rule worked as written and still hid the
> handoff from everyone on the code branch.

**Never rewrite history.** No force-push, no rebase, no squash, no amend of a
pushed commit on `feature/health-core`, `feature/health-hardening` or any branch
someone else may have checked out — unless Jay explicitly instructs it.

Never push to a branch other than the designated one without explicit
permission.

---

## 6. The ledger

Update the task file **in the same commit as the work**, or in the commit
immediately after. Record the commit SHA, the CI result, any blocker, and
exactly one next action.

When a task's state changes, edit its file, its row in
[`tasks/README.md`](tasks/README.md), and the status table in
[`AI_HANDOFF.md`](AI_HANDOFF.md).

If the ledger disagrees with Git, **Git wins**: correct the ledger, never
rewrite history to match it.

---

## 7. Honesty

- "CI green" means the suite passed. It does not mean the feature works.
- "Not verified in a browser" is written down when it is true.
- A review by the same agent that wrote the code is not an independent review.
  Say so.
- Report what happened, including what was skipped and why. If a step failed,
  say it failed and show the output.
- Do not simply agree. If the plan is wrong, say which part and why, with the
  evidence.

---

## 8. Scope guards

Do not, without being asked:

- implement Hevy;
- redesign the production UI while hardening is mid-flight;
- replace the existing Galaxy;
- remove existing functionality to match a mockup;
- hard-code mockup sample values as production data;
- begin M3 before the M2.5 correctness and security gates close.
