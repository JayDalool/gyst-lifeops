# Agent workflow: Issues, labels and lifecycle

Permanent reference for how development work is tracked. Governance summary:
[`../../AGENTS.md`](../../AGENTS.md).

| File | Purpose |
|---|---|
| [`IMPLEMENTATION_WORKFLOW.md`](IMPLEMENTATION_WORKFLOW.md) | Issue → branch → PR → merge, step by step |
| [`REVIEW_WORKFLOW.md`](REVIEW_WORKFLOW.md) | Independent review format and outcomes |

## Source of truth

- **GitHub Issues** in `JayDalool/gyst-lifeops` are the only editable development-task record. **`master`** is the canonical trunk: task branches start from it and PRs target it (GYST-VN-001).
- An Issue created from the vNext backlog keeps its provisional ID in the title and body (for example `GYST-VN-003`). The ID stays the stable roadmap reference; GitHub's `#number` is the working reference.
- **Labels are canonical for workflow state.** GitHub Projects (if ever added) is a visualization only and never holds canonical fields.
- `docs/tasks/` is a historical ledger. It isn't updated for new work.

## Labels

Exactly one `status:*`, one `priority:*`, one `type:*` and one `risk:*` per open Issue.

| Label | Meaning |
|---|---|
| `status:inbox` | New, not yet triaged. An open Issue with no status counts as inbox. |
| `status:needs-spec` | Missing acceptance criteria or a decision; not workable |
| `status:ready` | Fully specified; dependencies met; may be started (by the eligible worker) |
| `status:in-progress` | A task branch exists and work is under way |
| `status:review` | PR open, or merged and awaiting Jay's verification |
| `status:blocked` | Waiting on a named dependency or decision (named in a comment) |
| `status:done` | Verified and accepted; set when the Issue is closed as completed |
| `priority:p0` … `priority:p3` | P0 critical/blocking · P1 important · P2 useful next · P3 later |
| `type:bug` · `type:feature` · `type:ui` · `type:ux` · `type:refactor` · `type:security` · `type:infrastructure` · `type:integration` · `type:operations` · `type:documentation` · `type:research` | Task type (matches the vNext backlog) |
| `risk:normal` · `risk:elevated` · `risk:privileged` · `risk:not-agent-eligible` | Risk class from `docs/audit/vnext/09_SECURITY_FOUNDATION.md` §J. **Informative**: GYST and any future Agent Bridge re-derive risk and never trust an editable label alone. |
| `agent:claude` · `agent:codex` | Intended implementer (intent only; starts nothing) |
| `agent:ready` | Jay has authorised an agent to start this Issue. **Only Jay sets it.** Never on `risk:not-agent-eligible` Issues or while blocked. |
| `needs:security-review` | PR/Issue needs an independent security review before Jay's acceptance (always for PRIVILEGED work) |
| `needs:jay` | Waiting on something only Jay can do: merge authorization, an owner decision, or an owner-run step. Set by the agent or coordinator that hit it; removed once Jay has done it |
| `needs:ui-review` | Merged and verified by agents; Jay's visual check is pending. Removed when Jay approves |
| `needs:deploy` | Merged, not yet on production. Removed after the deploy and its post-deploy checks pass |
| `needs:legal` | Waiting on a lawyer's answer. Set and removed by Jay |

`needs:*` flags add to the single `status:*` label; they never replace it. The
same flags exist in `samvisionai` and `myfamilyexpenses`, so a cross-repo view
can list everything waiting on Jay.

GitHub's default labels (`bug`, `enhancement`, `documentation`, …) remain in the
repository but aren't part of the GYST workflow. Use the `type:*` labels.

## Issue state rules

From `docs/audit/vnext/09_SECURITY_FOUNDATION.md` §H:

| Situation | Rule |
|---|---|
| Open Issue | Its single `status:*` label is the workflow stage |
| Closed as **completed** | Done: `status:done`, other `status:*` removed |
| Closed as **not planned** | Won't do: all `status:*` removed; not counted as done |
| **Reopened** | Remove `status:done`, set `status:inbox` (re-triage) |
| Agent PR merged, verification pending | PR used `Refs #n`, so the Issue stays open in `status:review` until Jay verifies and closes it |
| Several or unknown `status:*` on one Issue | Inconsistent: fixed by Jay, never auto-"repaired" by an agent |

Until GYST's label normalizer exists, Jay (or an agent on Jay's instruction) applies these rules by hand.

## Creating Issues

Use the **GYST task** Issue form (`.github/ISSUE_TEMPLATE/gyst-task.yml`). Small
tasks can leave optional fields short. A blank Issue is allowed for quick
capture; it starts in `status:inbox` or `status:needs-spec`.
