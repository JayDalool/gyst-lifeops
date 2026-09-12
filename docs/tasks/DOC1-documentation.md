# DOC1 — Project documentation

- **ID:** DOC1
- **Title:** Architecture review, UI v3 standards, handoff and task ledger
- **Status:** `DONE`
- **Owner:** Claude
- **Implementation commit(s):**
  - `d4007a3` — task ledger and handoff on the implementation branch
  - `5341e46` — ledger corrected after M4 was committed
  - the commit carrying this file — `docs/RULES.md`, REV1, DOC1, B2, and the
    M4 CI result
  - On `docs/ui-v3-standards`: `ce91119`, `c18cb5b`, `8c39e3d`, `b7f4492`
    (handoff, now superseded) and the `docs/design/v3/` set
- **Review status:** CI green for `d4007a3`
  ([run 34665123234](https://github.com/JayDalool/gyst-lifeops/actions/runs/34665123234)).
  Content review by a second party: **UNKNOWN**.

## What exists

- **`docs/AI_HANDOFF.md`** — the map: branch state, decisions, blockers, next
  action. On the implementation branch.
- **`docs/tasks/`** — this ledger, one file per work item.
- **`docs/RULES.md`** — the working rules agents must follow.
- **`docs/design/v3/GYST_UI_V3_STANDARDS.md`** — the canonical UI reference, on
  `docs/ui-v3-standards`, alongside the split `01`–`06` documents.

The architecture review of the Health plan was delivered as a conversation
artifact rather than a repository file. **It is not in Git** and is therefore not
recoverable from this repository — only its consequences are, as the B1/A3/A4/A5
work and the decisions recorded in the handoff.

## The mistake this task exists to remember

`AI_HANDOFF.md` was first created only on `docs/ui-v3-standards`, and merges ran
one way — implementation branch into docs branch. The rule kept documentation
commits out of the code branch as intended, but it also meant **nobody working
on the implementation branch could see the handoff**, which defeated its entire
purpose. It went unnoticed until Jay asked why the file was missing.

The rule is now inverted for `docs/AI_HANDOFF.md`, `docs/tasks/` and
`docs/RULES.md`; `docs/design/v3/` stays on the docs branch.

## Acceptance criteria

- [x] Handoff and ledger live where the work happens
- [x] Every code commit since the baseline is named by exactly one task file
- [x] Nothing claimed as complete without Git or CI evidence
- [x] Working rules written down in one canonical place

## Blockers

None.

## Next action

Keep it current. The rule is in [`../RULES.md`](../RULES.md): update the task
file and the handoff in the same commit as the work, or in the commit
immediately after.
