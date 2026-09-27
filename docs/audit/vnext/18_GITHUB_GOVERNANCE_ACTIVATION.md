# Stage 9 — GitHub Governance Activation

- **Recorded:** 2026-09-27 on `audit/vnext` (worktree `/srv/server/repos/gyst-lifeops-audit`)
- **Status:** governance files and templates committed on `audit/vnext` (not pushed, not merged). Labels and the first eight Issues **created on GitHub** (`JayDalool/gyst-lifeops`).
- **Not done:**
  - no application code changed
  - no production, database, deploy, branch-protection, secrets, visibility or credential change
  - no `agent:ready` applied
  - no agent execution authorised

---

## A. Permanent task source of truth

| Role | Is |
|---|---|
| **GitHub Issue** | The canonical development task record (one editable task system) |
| Git branch | The implementation workspace: `task/<issue>-<slug>` |
| Pull request | The implementation artifact: `Refs #<issue>` |
| Claude / Codex | Implementation and review workers |
| GYST (later) | A control surface over the same Issues and labels |
| Jay | Final verification, merge, close |

Rules:
- `docs/audit/vnext/15_TASK_BACKLOG.md` remains the **planning** source for the vNext migration. Once an item is instantiated, **the Issue is the live record**. The backlog entry isn't edited to track progress.
- Every instantiated Issue keeps its provisional `GYST-VN-nnn` ID in the title and body, so roadmap and dependency references stay stable.
- **Labels are canonical for workflow state** (Stage 5 decision). No GitHub Project exists, and none was created.

## B. Governance files

| File | Change |
|---|---|
| `AGENTS.md` (root) | **Rewritten** as the concise permanent governance (about 90 lines): roles, 15 task rules, the non-negotiable Galaxy rule, Health/design/security rules, the GYST-VN-001 base-branch gate |
| `docs/history/AGENTS_CODEX_HANDOFF_2026-09-12.md` | The previous root `AGENTS.md` (a 422-line Codex handoff), **moved, not deleted**, with a "historical" banner |
| `CLAUDE.md` (new) | Thin pointer: `AGENTS.md`, `docs/RULES.md`, the Issue, referenced vNext documents. No duplicated policy. |
| `docs/agents/README.md` (new) | Source of truth, label reference, Issue state rules |
| `docs/agents/IMPLEMENTATION_WORKFLOW.md` (new) | Issue → branch → PR → review → merge → close |
| `docs/agents/REVIEW_WORKFLOW.md` (new) | Independent review format and outcomes |
| `docs/RULES.md` | **Kept in place and in force.** One note at the top explains how §5 (branches) and §6 (ledger) apply under Issues. No rule removed. |
| `docs/AI_HANDOFF.md`, `CODEX_PROMPT.md`, `HANDOFF_TO_CODEX.md` | Kept; a "superseded as workflow entry point" banner added to each |
| `docs/tasks/README.md` | "Historical ledger" banner added |

Future Codex-specific guidance follows the same pattern as `CLAUDE.md`: a thin pointer to `AGENTS.md`.

The workspace-level `/srv/server/repos/AGENTS.md` (outside this repository) was not changed.

## C. Branch convention

- `task/<issue-number>-<short-slug>`, for example `task/4-login-throttling`.
- The branch is created from the base branch recorded by **GYST-VN-001 (#2)**. One Issue, one branch, one focused PR.
- No force-push, no rewriting shared history; only Jay merges.

## D. Issue lifecycle

```text
status:inbox → status:needs-spec → status:ready → status:in-progress → status:review → (Jay closes as completed) status:done
                                        ↘ status:blocked (named dependency) ↗
```

- Open Issue: its single `status:*` label is the stage.
- Closed as completed: `status:done`. Closed as not planned: no `status:*`. Reopened: `status:inbox`.
- Agent PRs use `Refs #n`, so a merge doesn't close the Issue. Jay verifies and closes.
- Several or unknown `status:*` labels: fixed by Jay, never auto-repaired.
- Only Jay applies `agent:ready`. It is never set on `risk:not-agent-eligible` or blocked Issues.

These rules come from `09` §H. Until GYST's label normalizer exists, they are applied by hand.

## E. Review lifecycle

- An independent reviewer (a different agent/provider where practical) inspects the Issue, the acceptance criteria, the diff, the tests, the design and security rules, and the regression risk.
- Each criterion gets PASS / FAIL / PARTIAL / NOT VERIFIED.
- Findings are graded BLOCKING / IMPORTANT / MINOR.
- The outcome is **READY FOR HUMAN REVIEW** or **CHANGES REQUIRED**.
- The reviewer never merges.
- ELEVATED and PRIVILEGED work always gets an independent review; PRIVILEGED also gets `needs:security-review`.

## F. Labels created

Created on GitHub on 2026-09-27. 30 labels; none existed before. GitHub's default labels were left untouched and aren't part of the workflow.

| Group | Labels |
|---|---|
| Status (7) | `status:inbox`, `status:needs-spec`, `status:ready`, `status:in-progress`, `status:review`, `status:blocked`, `status:done` |
| Priority (4) | `priority:p0`, `priority:p1`, `priority:p2`, `priority:p3` |
| Type (11) | `type:bug`, `type:feature`, `type:ui`, `type:ux`, `type:refactor`, `type:security`, `type:infrastructure`, `type:integration`, `type:operations`, `type:documentation`, `type:research` |
| Risk (4, informative) | `risk:normal`, `risk:elevated`, `risk:privileged`, `risk:not-agent-eligible` |
| Agent (3) | `agent:claude`, `agent:codex`, `agent:ready` |
| Other (1) | `needs:security-review`, which has a defined workflow purpose (`09` §J PRIVILEGED PRs) |

State colours avoid violet (DD-4 spirit). Risk labels are informative: GYST and the future Agent Bridge re-derive risk and never trust a label alone.

## G. Templates created

| File | Purpose |
|---|---|
| `.github/ISSUE_TEMPLATE/gyst-task.yml` | "GYST task" Issue Form. Required: GYST-VN ID, summary, requirements, acceptance criteria, verification, risk class, agent eligibility. Optional: current problem, dependencies, design references, security references, out of scope, and rollback/risks/reviewer. Defaults to `status:inbox`. |
| `.github/ISSUE_TEMPLATE/config.yml` | Blank Issues stay allowed for quick capture |
| `.github/pull_request_template.md` | `Refs #`, summary, files/areas, tests, browser/device verification, acceptance-criteria table, security impact, database/migration impact, known limitations, screenshots |

**Activation:** GitHub uses Issue and PR templates only from the **default branch** (`master`). They're committed on `audit/vnext`, which isn't pushed. **They become active when this branch's content reaches the default branch through GYST-VN-001.** Stage 9 deliberately didn't push or merge. The eight initial Issues were created through the API with bodies that follow the same structure.

## H. Historical `docs/tasks/` treatment

- **Nothing deleted, nothing converted.**
- `docs/tasks/README.md` carries a banner: historical ledger; new work uses GitHub Issues; the files aren't canonical for new work.
- Open items there (NAV1, M3, A1/A2, `GYST_PASSWORD_HASH`) are reconciled as part of GYST-VN-001 (U-11).

## I. Initial Issues created

Bodies were generated from the Stage 8 backlog data, so priority, risk, type, eligibility, acceptance criteria and dependencies match `15` exactly. Each Issue references its source documents by repository path.

| GYST-VN | Issue | Title | Labels | Initial state reason |
|---|---|---|---|---|
| GYST-VN-001 | [#2](https://github.com/JayDalool/gyst-lifeops/issues/2) | Resolve the vNext implementation branch baseline | `status:ready` `priority:p0` `type:operations` `risk:not-agent-eligible` | Human decision; contains the branch/stash evidence and the exact decisions needed |
| GYST-VN-003 | [#3](https://github.com/JayDalool/gyst-lifeops/issues/3) | One app timezone and local-day correctness | `status:blocked` `priority:p0` `type:bug` `risk:normal` | Needs the base branch (#2) |
| GYST-VN-006 | [#4](https://github.com/JayDalool/gyst-lifeops/issues/4) | Login throttling | `status:blocked` `priority:p0` `type:security` `risk:elevated` | Needs #2 |
| GYST-VN-009 | [#5](https://github.com/JayDalool/gyst-lifeops/issues/5) | Harden host file permissions and record verification | `status:ready` `priority:p0` `type:operations` `risk:not-agent-eligible` | Human host operation; can start before #2 (only its runbook PR needs the base) |
| GYST-VN-010 | [#6](https://github.com/JayDalool/gyst-lifeops/issues/6) | Scheduled GYST backup job | `status:blocked` `priority:p0` `type:infrastructure` `risk:privileged` `needs:security-review` | Needs #5 and #2 |
| GYST-VN-018 | [#7](https://github.com/JayDalool/gyst-lifeops/issues/7) | vNext design tokens in the existing ui-* CSS layer | `status:blocked` `priority:p1` `type:ui` `risk:normal` | Needs #2 |
| GYST-VN-052 | [#8](https://github.com/JayDalool/gyst-lifeops/issues/8) | Hide unbuilt Health placeholders | `status:blocked` `priority:p1` `type:ui` `risk:normal` | Needs #2 |
| GYST-VN-143 | [#9](https://github.com/JayDalool/gyst-lifeops/issues/9) | Document the production Galaxy baseline | `status:blocked` `priority:p1` `type:documentation` `risk:normal` | Needs #2. States that `17A` §B is provisional until verified at runtime, and lists the 14 items to confirm. No Galaxy change allowed. |

Each blocked Issue has a comment naming its blocker. **`agent:ready` count: 0.**

**Validation performed:**
- labels listed back (30 present)
- each Issue body has an acceptance-criteria checklist (3–18 items), a dependency section and its `GYST-VN` ID
- no Issue carries `agent:ready`
- the Issue form YAML parses with unique field ids
- `git status` shows only documentation, governance and template files changed
- the production checkout is still `dab7995`, clean

## J. Remaining backlog instantiation strategy

1. **Validate first.**
   - Work #2 (baseline) and #5 (permissions) as the human-operational pilots.
   - Then one agent task (for example #4 or #8) through the full branch → PR → independent review → merge → close flow.
   - Adjust the template, labels or workflow docs based on what the pilots show.
2. **Instantiate by milestone, just ahead of need.**
   - Create the rest of M0 after the pilot, M1 when M0's P0/P1 items are nearly done, and so on.
   - Don't bulk-create all 145 Issues. Far-future Issues would go stale and duplicate the planning document.
3. **Instantiate the same way:** the body is generated from the backlog entry; the `GYST-VN` ID is kept; dependencies link to existing Issue numbers or name un-instantiated IDs.
4. **Initial status:**
   - `status:blocked` when dependencies are open
   - `status:needs-spec` when the backlog entry needs refinement
   - `status:ready` only when fully workable
5. **Backlog edits** after instantiation happen in the Issue. The planning document is updated only for structural roadmap changes (new tasks keep appended IDs, as with 143–145).

## K. Galaxy permanent rule

**U-13 is resolved.** Root `AGENTS.md` contains, as a non-negotiable product rule:

> "GYST Galaxy is a protected signature feature. Agents may enhance and refactor it but must not delete, replace, or downgrade the real Three.js Galaxy without explicit product-owner approval."

`AGENTS.md` also states:
- the Projects list and Galaxy coexist
- Stage 7 prototype markup is a design reference, not a replacement
- Galaxy tasks follow `17A_GALAXY_PRESERVATION.md`
- GYST-VN-143 (#9) establishes the browser/performance baseline before material Galaxy changes

`CLAUDE.md` points to it.

## L. GYST-VN-001 baseline gate

**U-1 isn't resolved by this stage.** Issue #2 records the evidence (default branch `master` 41 commits behind `feature/health-hardening`; production on `dab7995`; local `master` no longer shows unpushed commits; three stashes including `stash@{2}` activity-events WIP; `docs/ui-v3-standards` 15 commits ahead; `audit/vnext` unpushed). It asks Jay to decide:
1. the implementation base branch (recommended: `master` brought up to date by a normal merge and kept as the default)
2. how `audit/vnext` lands
3. the stash disposition (recommended: preserve `stash@{2}` on a branch; drop nothing unreviewed)
4. historical branches (recommended: keep)
5. old ledger reconciliation

**Until #2 is resolved:**
- Issues may exist.
- Implementation branches that depend on the base don't start.
- No stash, commit, branch or mainline change is made by any agent.

## M. What Stage 10 may do after baseline resolution

Once Jay has resolved #2, Stage 10 (or the first implementation session Jay starts) may:
1. Land `audit/vnext` on the chosen base via Jay's PR and merge, which activates the Issue and PR templates.
2. Record the base branch in `AGENTS.md` and set #2 to done.
3. Unblock #3, #4, #7, #8 and #9 (`status:ready`). #6 unblocks after #5.
4. Run the pilot described in §J with one agent task end to end, including an independent review.
5. Instantiate the next Issues per §J.

Stage 10 still doesn't apply `agent:ready` or start automated agent dispatch. That remains gated by the Phase 10–12 security work and Jay's explicit authorisation.
