# Implementation workflow

```text
Issue
 ↓ read / specify
Confirm acceptance criteria
 ↓
Create task branch            task/<issue-number>-<short-slug>
 ↓
Implement
 ↓
Tests                         gofmt · go vet · go build · go test · go test -race · named checks
 ↓
Browser verification          UI work: phone + desktop, light + dark (+ iPhone/PWA where required)
 ↓
Self-review                   against every acceptance criterion
 ↓
PR                            "Refs #<issue>", PR template filled in
 ↓
Independent review            different agent/provider where practical (REVIEW_WORKFLOW.md)
 ↓
Human verification            Jay, where applicable
 ↓
Merge                         Jay only
 ↓
status:done / close           Jay
```

## Steps

1. **Read.** Read the whole Issue and every document it references. Check its dependencies are closed (or explicitly waived by Jay in the Issue).
2. **Specify.** If acceptance criteria are missing, ambiguous or wrong, comment on the Issue with a proposal and stop. The Issue goes to `status:needs-spec`. Don't guess.
3. **Confirm eligibility.** Agents start only Issues that Jay has made startable. That means `status:ready` for interactive sessions Jay asked you to run, or `agent:ready` for automated dispatch (later). Never start `risk:not-agent-eligible` work. For human-operational Issues, agents only prepare what the Issue asks for (runbooks, scripts, queries).
4. **Branch.** From the base branch recorded by GYST-VN-001: `task/<issue-number>-<short-slug>`. One Issue, one branch.
5. **Implement.** Smallest change that meets the criteria. Follow `docs/RULES.md` and the design/security documents the Issue names. Backend correctness changes don't ride inside visual PRs. No drive-by refactors.
6. **Test.** Run the required checks and record the actual output. Never weaken a test, constraint or API to pass (`docs/RULES.md` §2.2). A deliberate specification change to a test is stated as such in the PR.
7. **Verify in the browser** (UI work). Phone (390×844) and desktop (1280×800), light and dark, keyboard, console and network clean. Real iPhone/PWA when the Issue requires it; otherwise write "not verified on device". Galaxy work also runs the `17A` §L checklist.
8. **Self-review.** Go through the diff against each acceptance criterion. Look for scope creep, secrets, debug output and stray files (`git status`).
9. **Open the PR.** Use the PR template. Body contains `Refs #<issue>`. Set the Issue to `status:review`. Add `needs:security-review` for PRIVILEGED work.
10. **Independent review.** A different agent reviews per `REVIEW_WORKFLOW.md`. Fix BLOCKING findings on the same branch (new commits, no force-push).
11. **Human verification and merge.** Jay merges. For operational steps (deploy, host changes), Jay performs them as the Issue describes.
12. **Close.** Jay closes the Issue as completed once the outcome is verified; it becomes `status:done`.

## Human-operational Issues

Issues typed `type:operations` or labelled `risk:not-agent-eligible` are
performed by Jay. An agent may draft the runbook, script or checklist in a PR
when the Issue asks for it. Evidence is recorded in the Issue as paths, modes,
counts and outcomes, never secret values.

## Blocked and dependency handling

A dependency is another Issue (or a `GYST-VN-nnn` ID not yet instantiated).
While a dependency is open, the Issue stays `status:blocked` with a comment
naming it. Don't start blocked work "to save time".
