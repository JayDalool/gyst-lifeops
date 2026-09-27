# Independent review workflow

The reviewer is normally a **different agent/provider** from the implementer
(Codex reviews Claude's PRs and vice versa) where practical. A review by the
implementing agent isn't independent and must say so (`docs/RULES.md` §7).

## Inputs the reviewer must inspect

- the Issue (requirements, acceptance criteria, out of scope, risk class)
- the full diff
- tests: which were added and which were run (re-run them where possible)
- the relevant design rules (`13A`, `11`–`13`; `17A` for Galaxy)
- security rules (`09`, `10A`; `docs/RULES.md` §4)
- regression risk: hotspot files, shared CSS/JS, migrations, auth paths

## Review output (posted as a PR review or comment)

```markdown
## Independent review: Refs #<issue> (GYST-VN-nnn)

Reviewer: <agent/provider>   Implementer: <agent/provider>

### Acceptance criteria
| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | ... | PASS / FAIL / PARTIAL / NOT VERIFIED | test name, command output, file:line, screenshot note |

### Findings
- **BLOCKING**: must be fixed before human review (wrong behaviour, failing or missing required test, security issue, rule violation, scope creep, Galaxy regression)
- **IMPORTANT**: should be fixed or explicitly accepted by Jay
- **MINOR**: optional polish

### Checks run by the reviewer
<commands and results, or "not run" with the reason>

### Outcome
READY FOR HUMAN REVIEW   or   CHANGES REQUIRED
```

## Rules

- Every acceptance criterion gets exactly one of **PASS**, **FAIL**, **PARTIAL** or **NOT VERIFIED**. NOT VERIFIED is honest and allowed; guessing PASS isn't.
- Any FAIL or BLOCKING finding means **CHANGES REQUIRED**.
- The reviewer **never merges**, never pushes to the implementer's branch unless asked, and never changes the Issue's scope.
- `needs:security-review` PRs get an explicit security section: auth, CSRF, secrets, input handling, file/host access, migrations.
- After fixes, the reviewer re-checks only what changed plus any affected criteria, and restates the outcome.
