# Stage 6 — Security Decisions

- **Decided by:** Jay (product owner)
- **Recorded:** 2026-09-27
- **Responds to:** `09_SECURITY_FOUNDATION.md` §"Decisions for Jay" (D-S1–D-S6) and `10_THREAT_MODEL.md` (Stage 6 commit `95bddf0`, unchanged)
- **Status:** **Binding** for all later vNext work wherever it differs from the Stage 6 recommendations.
- **Precedence:** this file overrides `09` and `10` where they differ. `05A` and
  `08A` remain binding alongside it. Where this file tightens `08A`, the stricter rule applies.
- **Implementation:** none. No credentials, accounts, permissions, Cloudflare
  settings, backups or code were changed.

---

## Summary

| # | Topic | Decision | vs. Stage 6 recommendation |
|---|---|---|---|
| D-S1 | Public login / Cloudflare Access | Foundation controls first; Cloudflare Access **later, after validation**; machine endpoints never behind interactive Access | Approved with validation requirements |
| D-S2 | Claude / Codex authentication | **No default.** Verify current official provider support for unattended auth under `gyst-agent` **immediately before** Agent Bridge implementation | **Modified**: the API-key recommendation isn't adopted |
| D-S3 | Issue closing | Agent PRs use **`Refs #n`**; human verification closes | Approved |
| D-S4 | `agent:ready` | NORMAL: public + recent re-auth. ELEVATED/PRIVILEGED: tailnet only. NOT ELIGIBLE: always refused. Stop allowed publicly. | Approved |
| D-S5 | Human session | 30 d rolling, **90 d absolute**, re-auth window **≈ 15 min** | Approved, **step-up window 15 min** (was 5 min) |
| D-S6 | Initial agent repository | **`gyst-lifeops` only**, no wildcards; SamVision after demonstrated cycles | Approved with explicit exit criteria |
| O-1 | Mandatory ordering | **No `gyst-agent` account before file-permission hardening** | New binding constraint |
| O-2 | Agent tooling | Non-Docker Go toolchain for `gyst-agent`; **never** docker group | New binding constraint |
| O-3 | Prompt-injection boundary | Templates launch with flags that exclude repo-local agent config/hooks; policy from bridge config only | Approved |
| O-4 | Foundation package | Step 0 = 11 items (listed below); the local-day fix is a separate high-priority foundation fix | Approved (expanded count) |

---

## D-S1 — Public login / Cloudflare Access

**APPROVED WITH VALIDATION.**

**First, the basic GYST security foundation:**
- login throttling
- failed-login audit events
- scheduled expired-session cleanup
- sensitive-action re-authentication

**Later:** **Cloudflare Access** is approved as an additional layer **before
higher-impact public write/control functionality is enabled**.

**Validation required before enabling Cloudflare Access** (each recorded with evidence):

| # | Flow to verify |
|---|---|
| 1 | iPhone browser flow |
| 2 | Installed PWA flow (standalone mode) |
| 3 | Desktop flow |
| 4 | GYST session persistence under Access |
| 5 | Logout (GYST and Access interaction) |
| 6 | Re-authentication behaviour (GYST step-up under Access) |

**Machine endpoints must NOT require an interactive Cloudflare Access login.** Examples:
- Apple Health sync
- GitHub webhook
- explicitly approved n8n webhook endpoints
- explicitly approved SamVision webhook endpoints

**Every Access bypass must be:**
1. **exact and narrow in path**
2. **documented**
3. **tested**
4. **protected by its own strong machine authentication or signature verification**

**Never create a broad unauthenticated bypass.**

## D-S2 — Claude / Codex authentication

**MODIFIED.** The Stage 6 recommendation ("API keys") is **not adopted as a default**.

Context: Jay has **Claude Max** and will also use **Codex**.

**Required before Agent Bridge implementation:** verify with **current official
provider documentation** the supported unattended/headless authentication
mechanisms, **separately** for:
1. Claude Code
2. Codex CLI

For each, determine whether it supports:

| Capability | Claude Code | Codex CLI |
|---|---|---|
| Subscription/OAuth authentication under a dedicated service account | *to verify* | *to verify* |
| Supported machine credentials | *to verify* | *to verify* |
| API keys | *to verify* | *to verify* |
| Unattended/headless execution | *to verify* | *to verify* |
| Revocation | *to verify* | *to verify* |
| Usage/spend controls | *to verify* | *to verify* |

**Requirements for whatever mechanism is chosen:**
- works under the dedicated **`gyst-agent`** OS account
- **provider-supported** (no workarounds or terms-violating reuse)
- supports unattended execution where required
- credentials **never exposed to GYST**
- credentials **never copied into worktrees**
- credentials **revocable**
- costs **understandable and controllable**

**Do not assume:**
- that Claude Max includes Anthropic API usage
- that Claude Max subscription authentication is valid for an unattended service account
- that Codex subscription authentication and API authentication are interchangeable

**If API keys turn out to be required:**
- use **dedicated agent credentials**
- configure available **spend and rate limits**
- store them **only under the appropriate service account**
- **never** expose them in logs, agent output or repositories

**Timing:** this is a **future provider-auth decision** to be verified **immediately before
Agent Bridge implementation**, because provider capabilities and terms may
change. The result is recorded as its own decision update.

Related Stage 6 text superseded: `09` §I.2 "Provider auth" row and decision
D-S2's recommendation. Stage 5 evidence noted in `07` §B that Claude Code's
`--bare` mode reads only `ANTHROPIC_API_KEY` or an `apiKeyHelper`. That is a
**mode constraint**, not a decision; templates must match the verified auth method.

## D-S3 — Issue closing

**APPROVED.** Agent-generated pull requests use **`Refs #<issue-number>`**, not `Closes #<issue-number>`.

Issue lifecycle:

```text
implementation → review → merge → human verification → status:done → close
```

- A merged PR **doesn't** prove the task is complete.
- GYST may later provide an explicit human **"Verify & Close"** action (Attention item after merge).
- Consistent with `09` §H: after merge the issue stays `status:review` until
  Jay verifies, and closing as completed sets `status:done` (normalizer rule).

## D-S4 — `agent:ready`

**APPROVED.**

| Task class | Who / where |
|---|---|
| **NORMAL** | May receive `agent:ready` through **authenticated public GYST**, only after **recent sensitive-action re-authentication** (D-S5 window) |
| **ELEVATED / PRIVILEGED** | Approval and dispatch **only through the tailnet-only interface**, satisfying the additional Stage 6 approval requirements (`09` §J: approval record in GYST, step-up, per-run confirmation for PRIVILEGED) |
| **NOT AGENT-ELIGIBLE** | **Always refused** by the Agent Bridge |
| **Stop / cancel** a running agent | Allowed from normal authenticated GYST (stopping reduces risk) |

## D-S5 — Human session

**APPROVED.**

| Setting | Value |
|---|---|
| Rolling lifetime | **30 days** |
| Absolute maximum lifetime | **90 days** |
| Recent-authentication window for sensitive actions | **≈ 15 minutes** (initial planning target) |

Sensitive operations requiring recent re-authentication include:
- setting `agent:ready`
- meaningful GitHub writes
- elevated-task approval
- credential management
- privileged actions

**Tailnet-only privileged operations may still require explicit confirmation**
(the action-nonce confirmation page, `09` §C) even within the re-authentication window.

**Changes to `09`:**
- The step-up window becomes **≈ 15 min** (was 5 min in `09` §C).
- **"Meaningful GitHub writes"** now require recent re-authentication. `09` §G.2
  listed issue create/edit, labels and comments as *public authenticated*
  without step-up. Planning must classify which GitHub writes are "meaningful"
  (at minimum anything that changes workflow state or dispatch eligibility);
  purely cosmetic writes may stay without step-up only if explicitly listed.

## D-S6 — Initial agent repository

**APPROVED.** The initial Agent Bridge repository allowlist is **`gyst-lifeops` only**. **No wildcard repository access.**

SamVision (`samvisionai`) may be added **only after several successful GYST
implementation/review cycles** demonstrate all of the following:

| # | Criterion |
|---|---|
| 1 | Correct worktree isolation |
| 2 | Correct repository isolation |
| 3 | No credential exposure |
| 4 | Correct branch behaviour |
| 5 | Successful independent review |
| 6 | Correct cleanup |
| 7 | Useful execution logs |
| 8 | No host privilege violations |

Evidence for each criterion is recorded before the allowlist changes.

---

## O-1 — Mandatory ordering: file permissions before the agent account

**BINDING.** **Do not create or run the `gyst-agent` account before
file-permission hardening is complete.** (This also applies to `gyst-bridge`,
which is an unprivileged "other" account in the same sense.)

Scope includes the verified sensitive files and their classes:
- GYST `.env`
- FMS `.env` (and MyFamilyExpenses `.env`, `.env.test`)
- databases
- database backups
- any other sensitive file under `/srv/server` that an unprivileged agent account shouldn't be able to read

Stage 6 evidence: world-readable `.env` in three repositories and 45
world-readable DB/backup files. **File-permission hardening is a hard
prerequisite, not optional cleanup.** Completion requires a recorded
verification (a `find … -perm -o=r` check returning no sensitive files).

## O-2 — Agent tooling prerequisite

**BINDING.** The host has **no Go toolchain**, and `gyst-agent` gets **no Docker
access**. The Agent Bridge setup must therefore give the dedicated agent account
an appropriate **non-Docker Go toolchain** able to run GYST's:
- build
- vet
- tests
- race tests where practical

**Do not solve this by adding `gyst-agent` to the `docker` group.** (The race
detector needs cgo and a C toolchain; availability for the agent account is
verified during setup. If it's unavailable, the gap is recorded rather than
worked around with privileges.)

## O-3 — Repository prompt-injection boundary

**APPROVED.**
- Claude Code and Codex role templates **launch with supported flags and
  settings that prevent repository-local agent configuration and hooks from
  redefining the execution policy** (Stage 6 evidence: Claude `--setting-sources`,
  `--bare`, `--safe-mode`, `--strict-mcp-config`, `--allowedTools`; Codex
  `--ignore-user-config`, `--ignore-rules`, sandbox modes, and hooks untrusted by
  default). The exact flags are re-verified against the installed versions at implementation.
- **Repository contents remain untrusted input.**
- The permanent Agent Bridge **role templates, risk rules and allowed
  capabilities come from bridge-controlled configuration**, never from the
  repository being inspected.

## O-4 — Foundation package (Step 0)

**APPROVED** security and operations Step 0:

| # | Item |
|---|---|
| 1 | Login throttling |
| 2 | Failed-login auditing |
| 3 | Expired-session cleanup |
| 4 | File-permission hardening |
| 5 | Scheduled GYST backup |
| 6 | Off-host backup copy |
| 7 | One tested restore |
| 8 | `/healthz` |
| 9 | Basic application logging |
| 10 | Log rotation |
| 11 | UTC instants for new audit/event records |

**Separately:** the verified **local-day correctness bug** (Stage 2A S2-02,
Stage 2B D1) remains a **high-priority product foundation fix** and must appear
**near the start of the final roadmap**. It isn't part of this security package.

**Relation to `09` §P:** `09` listed the audit-log table and Kuma monitors in
Step 0. Failed-login *auditing* (item 2) needs a minimal audit record store;
the remaining Stage 6 "MUST BEFORE WRITE/CONTROL" items keep their placement.

---

## Carry-forward for the roadmap (Stage 7+)

1. Provider-auth verification task (D-S2), placed immediately before Agent Bridge implementation.
2. Cloudflare Access validation checklist and bypass inventory (D-S1), placed before public higher-impact writes.
3. Classification of "meaningful" GitHub writes that need step-up (D-S5).
4. File-permission hardening with a recorded verification, **before** any agent/bridge account (O-1).
5. Agent Go toolchain setup and race-detector feasibility (O-2).
6. SamVision allowlist exit-criteria evidence (D-S6).
7. Local-day fix near the start of the roadmap (O-4).
