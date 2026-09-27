# Stage 5 — Integration Architecture Decisions

- **Decided by:** Jay (product owner)
- **Recorded:** 2026-09-27
- **Responds to:** `06_INTEGRATION_ARCHITECTURE.md` §"Decisions for Jay" (D-J1–D-J8), `07`, `08` (Stage 5 commit `6e64ca0`, unchanged)
- **Status:** **Binding** for all subsequent vNext planning unless Jay explicitly revises them.
- **Precedence:** where this file differs from `06`, `07` or `08`, **this file wins**.
  Those documents are not edited. Product decisions in `05A_PRODUCT_DECISIONS.md` remain binding alongside this file.
- **Implementation:** nothing here is implemented, installed, configured or credentialed.

---

## Summary

| # | Topic | Decision | vs. Stage 5 recommendation |
|---|---|---|---|
| D-J1 | Orchestration / "Herder" | Purpose-built **Agent Bridge**; herdr optional and local only; GYST never depends on herdr | Approved |
| D-J2 | GitHub authentication | **Private GitHub App** long-term; a read-only fine-grained PAT only for a limited bootstrap; separate agent identity; human merge | Approved, with explicit limits |
| D-J3 | Development status | **GitHub labels are canonical**; Projects is visualisation only | Approved, with **explicit `status:inbox` and `status:done`** |
| D-J4 | Agent autonomy | `agent:ready` = entry to the AI queue, **with 10 safeguards** incl. high-risk-class approval | **Approved with added safeguards** |
| D-J5 | Agent OS user | Dedicated account **without sudo, docker, lxd** or equivalent | Strongly approved |
| D-J6 | n8n | App-specific authenticated webhooks/events; no full-access n8n key for routine use | Approved |
| D-J7 | Public vs tailnet | Public shows **sanitized state only**; all control and diagnostics **tailnet-only, enforced server-side** | **Approved with a stronger boundary** |
| D-J8 | Console | No terminal in GYST; "Open Console" → separately secured Tailscale SSH / terminal | Approved |
| SoT | Source-of-truth model | Approved as listed below | Approved |
| SEQ | Implementation order | Stage 5 sequence 0–13 is the planning baseline | Approved |

---

## D-J1 — Agent orchestration / "Herder"

**APPROVED.** GYST will eventually use a small, purpose-built **Agent Bridge**:
the integration boundary between GYST and coding-agent processes.

```text
GYST → Agent Bridge → Claude Code / Codex CLI
```

- The bridge may use **isolated git worktrees** and **structured/headless CLI
  execution** (capabilities verified in `07` §B–C).
- **herdr** may be used optionally as a **local** terminal monitoring and
  convenience tool.
- herdr is **not**:
  - the task source of truth
  - the remote API
  - the authentication layer
  - the PR manager
  - the canonical orchestrator state store
- **GYST must not depend on herdr.** Any bridge use of herdr is optional and
  must degrade to CLI-only operation.

## D-J2 — GitHub authentication

**APPROVED TARGET:** a **private GitHub App** is the long-term GYST integration identity.

Preferences:
- repository-scoped installation
- least privilege
- **read permissions first**
- write permissions added only when a specific feature needs them

**Bootstrap exception:** a **fine-grained personal access token** may be used
**temporarily** for an explicitly limited **read-only** bootstrap or spike, if
that materially simplifies first implementation. It **must not become the
permanent architecture by accident**. Any plan that uses it must name its
expiry and the step that replaces it with the App.

**Agent identity:** agents use a **separate automation identity and
credential** from GYST.

| Agent credential **may** | Agent credential **may not** |
|---|---|
| Read approved repositories | Merge protected branches |
| Push task branches | Administer repositories |
| Create or update pull requests as required | Change branch protection |

**Human approval remains required for merge** (branch protection).

## D-J3 — Development status

**APPROVED.** **GitHub labels are the canonical machine-readable development
workflow state.** GYST reads and writes canonical issue and label state
**through GitHub**.

Label families (examples; the complete set is fixed during planning):

| Family | Labels |
|---|---|
| Status | `status:inbox` · `status:needs-spec` · `status:ready` · `status:in-progress` · `status:review` · `status:blocked` · `status:done` |
| Priority | `priority:p0` · `priority:p1` · `priority:p2` · `priority:p3` |
| Type | `type:feature` · `type:bug` · `type:ui` · `type:security` · … |
| Agent | `agent:claude` · `agent:codex` · `agent:ready` |

- **GitHub Projects** may later be a visualisation or planning surface, but
  **Projects fields must not become a second, conflicting canonical status system.**
- **Difference from `06` §E.1:** `06` treated *Inbox* as "no status label" and
  *Done* as "issue closed". D-J3 makes both **explicit labels**
  (`status:inbox`, `status:done`). Planning must define the consistency rule
  between issue open/closed state and `status:*` labels (e.g. which actor applies
  `status:done` on close and how a conflict is surfaced), without introducing a
  second status store.
- The priority scale is **`p0`–`p3`** (replacing the `priority:*` words implied in `06`).

## D-J4 — Agent autonomy

**APPROVED WITH SAFEGUARDS.** The label **`agent:ready`** means Jay has
authorised the task to **enter the AI execution queue**. It does **not** mean
unrestricted autonomous execution.

Required safeguards:

| # | Safeguard |
|---|---|
| 1 | **Repository allowlist**: only approved repositories are eligible |
| 2 | **Maximum concurrent jobs** |
| 3 | Each issue executes in an **isolated task branch and worktree** |
| 4 | **Agents never merge** |
| 5 | **Protected / high-risk classes require additional approval before dispatch** (list below) |
| 6 | The bridge executes **only predefined agent roles and templates** |
| 7 | GYST **never sends arbitrary shell commands** through the normal agent-task API |
| 8 | Agent execution state is **visible and auditable** |
| 9 | Implementation and review **normally use different agents or providers** |
| 10 | **Human merge remains the final gate** |

**High-risk classes (additional approval before dispatch)**, examples:
- production database changes
- migrations with destructive potential
- authentication or security architecture
- secrets or credential rotation
- Docker or host privilege changes
- Caddy, network or firewall changes
- server-control functionality
- production deployment operations

**Planning implications:**
- A task must be classifiable as high-risk *before* dispatch. The mechanism (e.g.
  a label such as `type:security`, a dedicated risk label, or path-based rules in
  the bridge) is decided in planning. The bridge must refuse high-risk tasks
  lacking the additional approval.
- **The initial implementation is conservative:** read-only visibility
  (sequence step 8) comes before any dispatch capability (step 10).

## D-J5 — Agent operating-system user

**STRONGLY APPROVED.** Agent jobs run under a **dedicated OS account** that must
**not** belong to **sudo**, **docker** or **lxd**, and must not otherwise have
equivalent host-root capability.

Agents get access only to:
- approved repositories and worktrees
- required build and test tooling
- narrowly scoped credentials needed for their role

**Autonomous agents must not run as `jay`**, because `jay` effectively has
host-root capability through those groups (Stage 5 evidence: `id -nG`).

## D-J6 — n8n integration

**APPROVED.**
- GYST does **not** receive a full-access n8n API key merely to power routine status or events.
- Preferred model:
  - **n8n → authenticated app-specific webhooks/events → GYST**
  - where required, **GYST → narrowly defined workflow webhook → n8n**
- **Separate identities and secrets** for GYST workflows and SamVision workflows.
  **No universal shared secret.**
- GYST provides a **native Automation status and control surface**. The n8n
  editor remains a specialised external application reached through an
  explicit **"Open n8n"** action.
- If a future requirement genuinely needs the n8n management API, it is **reassessed separately**.

## D-J7 — Public domain vs tailnet

**APPROVED WITH A STRONGER BOUNDARY** than `08` §J.

**Public authenticated domain: sanitized operational state only.** Acceptable examples:

```text
Server     ● Healthy
n8n        ● Healthy
SamVision  ● Healthy
Backup     ● Current
```

Simple, non-sensitive CPU/RAM/disk summaries are possible **if later judged appropriate**.

**The public path must NOT expose:**
- raw logs
- environment or config details
- internal network topology
- container environment variables
- secrets
- privileged diagnostic output
- host paths
- detailed security information
- control actions

**TAILNET ONLY:**
- restart service
- deployment actions
- backup trigger
- detailed logs
- advanced diagnostics
- any future privileged actions

**Enforcement:** sensitive System endpoints **must enforce this server-side**
(e.g. the Caddy public-vhost deny plus the application's ingress check, `08` §J).
**Hiding UI buttons is not sufficient.**

**Effect on `08` §J:** the public column narrows to sanitized health states for
System. Service-detail views with images, restart counts, deploy SHAs, host
paths or topology are **tailnet-only** unless planning explicitly classifies a
field as sanitized. GitHub write actions aren't System actions and aren't
changed by D-J7; their ingress policy is confirmed during planning.

## D-J8 — Console

**APPROVED.**
- **No unrestricted terminal inside GYST.**
- If console access is provided later, GYST offers **"Open Console"**, which
  leads to a **separately secured Tailscale SSH / terminal** experience.
- The console keeps **its own authentication and privilege boundary**.
- GYST **must not proxy arbitrary shell commands** through its normal web API.

---

## Approved source-of-truth model

| Data | Source of truth |
|---|---|
| GYST personal tasks | **GYST** |
| Development issues / tasks | **GitHub Issues** |
| Development workflow state | **GitHub labels** |
| Branches, PRs, checks, reviews | **GitHub** |
| Agent execution state | **Agent Bridge** |
| Agent implementation artifacts | **Git + GitHub** |
| Server telemetry | **Server Provider / Uptime Kuma** |
| Automations | **n8n** |
| SamVision business data | **SamVision** |
| Health observations | **GYST**, sourced from Apple Health integrations |
| Attention | **GYST** derived/reference model pointing to the owning source |

This refines `06` §C: that matrix remains the detailed storage guide (NONE /
REFERENCE / CACHE / SNAPSHOT / AUTHORITATIVE), subject to this table.

## Approved implementation order (planning baseline)

```text
0. Foundations
1. Apps + Launcher
2. Attention from GYST sources
3. Machine credentials + audit log
4. Uptime Kuma status
5. GitHub read-only
6. n8n events
7. Server provider
8. Agent bridge (read-only)
9. GitHub write
10. Agent dispatch
11. Server actions / n8n Run Now
12. Logs
13. Console link/access (much later)
```

This order may be **refined** by the final roadmap but **should not be casually
reversed**. The security gates per step in `08` §L remain the prerequisites.

---

## Carry-forward notes for planning

Items these decisions create or sharpen, to resolve during roadmap planning
(not new product decisions):

1. **Label consistency rule** between issue state and `status:*` labels (D-J3).
2. **High-risk classification mechanism** and its extra-approval step (D-J4 #5).
3. **Bootstrap PAT**: if used, record its read-only scope, expiry and replacement step (D-J2).
4. **Sanitized-field classification** for public System views (D-J7).
5. **Agent account setup:** CLI authentication for a non-`jay` user (API keys vs
   subscriptions) is an open operational question (`07` §M).
6. **Repository allowlist and concurrency defaults** (D-J4 #1–2).
