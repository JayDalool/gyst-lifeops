# Stage 8 — Task Backlog (vNext)

- **Status:** planning document. No GitHub Issues created. IDs are **provisional** (`GYST-VN-nnn`) and are replaced by Issue numbers when Jay approves creating them.
- **Base:** `audit/vnext` after `e269df2`. Binding inputs: `05A`, `08A`, `10A`, `13A`, with `09`–`13` as the specifications they refine.
- **Galaxy:** `17A_GALAXY_PRESERVATION.md` is binding. Every Galaxy task carries its §L regression checklist; tasks touching shared surfaces carry the Galaxy quick check. IDs GYST-VN-143–145 were appended after Stage 8 so earlier IDs stay stable.
- **Roadmap:** `14_IMPLEMENTATION_ROADMAP.md` · **Dependencies:** `16_DEPENDENCY_MAP.md` · **Migration:** `17_MIGRATION_STRATEGY.md`

## How to read this backlog

| Field | Meaning |
|---|---|
| PRIORITY | P0 critical/blocking now · P1 important · P2 useful next · P3 later. Foundational alone doesn't mean P0. |
| RISK CLASS | `09` §J classes applied to the *work*: NORMAL · ELEVATED (security-relevant code) · PRIVILEGED (infrastructure-relevant code, Compose/Caddy/Docker, backup/deploy scripts, the bridge) · NOT AGENT-ELIGIBLE (operations on production, GitHub/Cloudflare/n8n settings, credentials, decisions). |
| AGENT ELIGIBILITY | Claude/Codex implementable · Human-assisted (agent prepares; Jay decides or executes) · Human-only (Jay) · NOT AGENT-ELIGIBLE operational work (Jay). Until the Agent Bridge exists, "Claude/Codex implementable" means an interactive Claude Code or Codex session supervised by Jay, as today. After go-live, only NORMAL tasks may be queued from the public host (`10A` D-S4). |
| SUGGESTED IMPLEMENTER / REVIEWER | Cross-review rule: the agent that didn't implement reviews; Jay approves every merge. ELEVATED and PRIVILEGED always need Jay's review. |
| ESTIMATED COMPLEXITY | S (one focused PR) · M (a larger PR or 2–3 PRs) · L (several PRs or a new component). No calendar estimates. |
| UI tasks | Every UI/UX task (and UI-changing bug/feature) references `11`, `12`, `13`, `13A` and carries the standard UI acceptance criteria: phone and desktop browser checks, 44 px targets, dark mode, no horizontal overflow, progressive disclosure, accessible labels, no violet state colour, no invented patterns. |
| Commits | One task → one focused PR (or a short series). Backend correctness changes never ride in a visual PR. Ledger updated per `AGENTS.md`. |

## Index

### M0 — Safe foundation

| ID | Name | Type | Pri | Risk | Eligibility | Cx |
|---|---|---|---|---|---|---|
| [GYST-VN-001](#gyst-vn-001) | Resolve the vNext implementation branch baseline | Operations | P0 | NOT AGENT-ELIGIBLE | Human-only | S |
| [GYST-VN-002](#gyst-vn-002) | Record accepted vNext decisions as ADRs and update ARCHITECTURE.md | Documentation | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-003](#gyst-vn-003) | One app timezone and local-day correctness | Bug | P0 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-004](#gyst-vn-004) | UTC instants for new audit and event records | Infrastructure | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-005](#gyst-vn-005) | Minimal audit_events store | Security | P1 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-006](#gyst-vn-006) | Login throttling | Security | P0 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-007](#gyst-vn-007) | Failed-login audit events | Security | P1 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-008](#gyst-vn-008) | Scheduled expired-session cleanup | Security | P1 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-009](#gyst-vn-009) | Harden host file permissions and record verification | Operations | P0 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-010](#gyst-vn-010) | Scheduled GYST backup job | Infrastructure | P0 | PRIVILEGED | Claude/Codex implementable | S |
| [GYST-VN-011](#gyst-vn-011) | Decide remediation for historical wrong-day rows | Research | P1 | NOT AGENT-ELIGIBLE | Human-assisted | S |
| [GYST-VN-012](#gyst-vn-012) | Off-host backup copy | Operations | P1 | NOT AGENT-ELIGIBLE | Human-assisted | S |
| [GYST-VN-013](#gyst-vn-013) | Restore runbook and one tested restore | Operations | P1 | NOT AGENT-ELIGIBLE | Human-assisted | S |
| [GYST-VN-014](#gyst-vn-014) | /healthz endpoint | Infrastructure | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-015](#gyst-vn-015) | Structured application logging and sanitized client errors | Security | P1 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-016](#gyst-vn-016) | Log rotation and Docker healthcheck in Compose | Infrastructure | P1 | PRIVILEGED | Claude/Codex implementable | S |
| [GYST-VN-017](#gyst-vn-017) | OPS1 production foreign-key audit | Operations | P1 | NOT AGENT-ELIGIBLE | Human-only | S |

### M1 — vNext core experience

| ID | Name | Type | Pri | Risk | Eligibility | Cx |
|---|---|---|---|---|---|---|
| [GYST-VN-018](#gyst-vn-018) | vNext design tokens in the existing ui-* CSS layer | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-019](#gyst-vn-019) | Appearance setting: System / Light / Dark | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-020](#gyst-vn-020) | Grouped list and row components | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-021](#gyst-vn-021) | Icon set and icon/action button components | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-022](#gyst-vn-022) | State components: chips, banners, empty, stale, error, offline | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-023](#gyst-vn-023) | Sheet, popover, inspector and confirm dialog components | UI | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-024](#gyst-vn-024) | 44 px tap targets and overflow guard for shared components | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-025](#gyst-vn-025) | Navigation shell: Today · Projects · Health · Learn · More | UI | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-026](#gyst-vn-026) | Simplify More to the grouped model | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-027](#gyst-vn-027) | Terminology migration in the UI: Project / Task / Roadmap | UX | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-143](#gyst-vn-143) | Document the production Galaxy baseline | Documentation | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-028](#gyst-vn-028) | Remove database writes from GET requests | Bug | P1 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-029](#gyst-vn-029) | Today: remove the Focus / Next up duplication | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-030](#gyst-vn-030) | Today: Now selection logic | Feature | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-031](#gyst-vn-031) | Today: Now block presentation | UI | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-032](#gyst-vn-032) | Today: overdue reminders and tasks | Feature | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-033](#gyst-vn-033) | Today: Rhythm strip (prayer, Athkar, gym) | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-034](#gyst-vn-034) | Today: quiet Lv chip | UI | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-035](#gyst-vn-035) | Today: exceptions-only status area | UI | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-036](#gyst-vn-036) | Today: responsive layout (phone order, desktop two columns) | UI | P1 | NORMAL | Claude/Codex implementable | S |

### M2 — vNext Life areas

| ID | Name | Type | Pri | Risk | Eligibility | Cx |
|---|---|---|---|---|---|---|
| [GYST-VN-037](#gyst-vn-037) | One project status → state mapping | Refactor | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-038](#gyst-vn-038) | Transactional service use cases for multi-write operations | Refactor | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-039](#gyst-vn-039) | Projects area sub-navigation: Projects · Development · Galaxy · Roadmaps | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-040](#gyst-vn-040) | Projects list vNext | UI | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-041](#gyst-vn-041) | Project detail vNext | UI | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-042](#gyst-vn-042) | Roadmaps list and route page | UI | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-043](#gyst-vn-043) | Galaxy discoverability | UX | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-044](#gyst-vn-044) | Galaxy 56 px touch targets | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-045](#gyst-vn-045) | Galaxy preview close and focus accessibility | Bug | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-046](#gyst-vn-046) | Galaxy completed worlds on the outer orbit | Feature | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-047](#gyst-vn-047) | Galaxy completed filter / hide | UI | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-048](#gyst-vn-048) | Galaxy sparse and empty states | UI | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-049](#gyst-vn-049) | Galaxy phone layout and premium preview | UI | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-144](#gyst-vn-144) | Galaxy visual depth (measured) | UI | P3 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-145](#gyst-vn-145) | Galaxy subtle project-state encoding | Feature | P3 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-050](#gyst-vn-050) | Courses → Learn (IA and presentation) | UX | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-051](#gyst-vn-051) | Learn: zero, one and many courses with Continue | UI | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-052](#gyst-vn-052) | Hide unbuilt Health placeholders | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-053](#gyst-vn-053) | Tie the Hevy card to real connection state | Bug | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-054](#gyst-vn-054) | Health Summary simplification | UI | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-055](#gyst-vn-055) | Health navigation normalization | UX | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-056](#gyst-vn-056) | Nutrition vNext | UI | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-057](#gyst-vn-057) | Training: connected and disconnected states | UI | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-058](#gyst-vn-058) | Progress: real-data trends only | UI | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-059](#gyst-vn-059) | Decide the training storage shape | Research | P3 | NORMAL | Human-assisted | S |
| [GYST-VN-060](#gyst-vn-060) | Goals at More → Goals, with edit | Feature | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-061](#gyst-vn-061) | Secondary screens adopt the vNext shell | UI | P3 | NORMAL | Claude/Codex implementable | L |

### M3 — Command centre

| ID | Name | Type | Pri | Risk | Eligibility | Cx |
|---|---|---|---|---|---|---|
| [GYST-VN-062](#gyst-vn-062) | Apps registry and Apps page | Feature | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-063](#gyst-vn-063) | Launcher v2: shell and Go mode | Feature | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-064](#gyst-vn-064) | Launcher Do mode | Feature | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-065](#gyst-vn-065) | Launcher Capture mode | Feature | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-066](#gyst-vn-066) | Attention service with GYST sources | Feature | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-067](#gyst-vn-067) | Attention stored conditions, snooze and dismiss | Feature | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-068](#gyst-vn-068) | Attention page and badge | UI | P1 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-069](#gyst-vn-069) | Today: Attention section | UI | P1 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-070](#gyst-vn-070) | Critical Attention push | Feature | P3 | NORMAL | Claude/Codex implementable | S |

### M4 — Read-only integrations

| ID | Name | Type | Pri | Risk | Eligibility | Cx |
|---|---|---|---|---|---|---|
| [GYST-VN-071](#gyst-vn-071) | Decide packaging and repository for Bridge, Server Provider and Action Runner | Research | P1 | NOT AGENT-ELIGIBLE | Human-assisted | S |
| [GYST-VN-072](#gyst-vn-072) | Consolidate configuration into one struct | Refactor | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-073](#gyst-vn-073) | Dual listener: public, privileged (tailnet) and internal machine | Security | P1 | PRIVILEGED | Claude/Codex implementable | L |
| [GYST-VN-074](#gyst-vn-074) | Bind the tailnet site to the Tailscale address only | Infrastructure | P1 | PRIVILEGED | Human-assisted | S |
| [GYST-VN-075](#gyst-vn-075) | Security headers including frame-ancestors | Security | P1 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-076](#gyst-vn-076) | Run the GYST container as a non-root user | Infrastructure | P2 | PRIVILEGED | Claude/Codex implementable | S |
| [GYST-VN-077](#gyst-vn-077) | Session lifetime: 30-day rolling, 90-day absolute | Security | P1 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-078](#gyst-vn-078) | Sign out everywhere and password-change revocation | Security | P2 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-079](#gyst-vn-079) | Step-up re-authentication, action nonce and confirmation pattern | Security | P1 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-080](#gyst-vn-080) | Unify password hashing at cost 12 | Security | P2 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-081](#gyst-vn-081) | Machine credentials with scopes | Security | P1 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-082](#gyst-vn-082) | Machine credential admin (tailnet, step-up) | Security | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-083](#gyst-vn-083) | Processed-event idempotency store | Security | P2 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-084](#gyst-vn-084) | Audit viewer and retention pruning | Security | P2 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-085](#gyst-vn-085) | Bound the Health sync date range | Security | P2 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-086](#gyst-vn-086) | Dependency vulnerability check in CI | Security | P2 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-087](#gyst-vn-087) | Jira sync: send credentials only to the configured host | Security | P2 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-088](#gyst-vn-088) | Deploy recording and running-version stamp | Infrastructure | P2 | PRIVILEGED | Claude/Codex implementable | S |
| [GYST-VN-089](#gyst-vn-089) | Configure Uptime Kuma monitors | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-090](#gyst-vn-090) | Uptime Kuma status provider in GYST | Integration | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-091](#gyst-vn-091) | Server Provider and filtering Docker proxy (read-only) | Infrastructure | P2 | PRIVILEGED | Claude/Codex implementable | L |
| [GYST-VN-092](#gyst-vn-092) | Deploy the Server Provider | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-093](#gyst-vn-093) | System page: sanitized public view and tailnet detail | UI | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-094](#gyst-vn-094) | System Attention and quiet healthy indicator | Feature | P3 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-095](#gyst-vn-095) | Schedule the n8n Postgres backup | Operations | P2 | NOT AGENT-ELIGIBLE | Human-assisted | S |
| [GYST-VN-096](#gyst-vn-096) | GitHub auth decision: App-first or temporary PAT | Research | P1 | NOT AGENT-ELIGIBLE | Human-only | S |
| [GYST-VN-097](#gyst-vn-097) | Temporary fine-grained PAT (only if chosen) | Operations | P3 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-098](#gyst-vn-098) | Register the GYST GitHub App (read permissions) | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-099](#gyst-vn-099) | Branch protection / ruleset on gyst-lifeops | Operations | P1 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-100](#gyst-vn-100) | Create the GitHub label taxonomy | Operations | P2 | NOT AGENT-ELIGIBLE | Human-assisted | S |
| [GYST-VN-101](#gyst-vn-101) | GitHub read provider: repositories and issues | Integration | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-102](#gyst-vn-102) | GitHub PR, review and check status | Integration | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-103](#gyst-vn-103) | GitHub webhook receiver: verify, refetch, dedupe | Integration | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-104](#gyst-vn-104) | Catch-up polling and stale/failure detection | Integration | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-105](#gyst-vn-105) | Development area: task list | UI | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-106](#gyst-vn-106) | Development task detail (read-only) | UI | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-107](#gyst-vn-107) | Reviews and Repositories views | UI | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-108](#gyst-vn-108) | GitHub Attention conditions | Feature | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-109](#gyst-vn-109) | Separate n8n identities for GYST and SamVision | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-110](#gyst-vn-110) | n8n event ingestion | Integration | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-111](#gyst-vn-111) | Automations page (status only) | UI | P2 | NORMAL | Claude/Codex implementable | S |

### M5 — Agent-assisted development

| ID | Name | Type | Pri | Risk | Eligibility | Cx |
|---|---|---|---|---|---|---|
| [GYST-VN-112](#gyst-vn-112) | Verify current official Claude Code authentication for unattended use | Research | P2 | NOT AGENT-ELIGIBLE | Human-assisted | S |
| [GYST-VN-113](#gyst-vn-113) | Verify current official Codex CLI authentication for unattended use | Research | P2 | NOT AGENT-ELIGIBLE | Human-assisted | S |
| [GYST-VN-114](#gyst-vn-114) | Create gyst-bridge and gyst-agent service accounts | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-115](#gyst-vn-115) | Non-Docker Go toolchain for gyst-agent | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-116](#gyst-vn-116) | Agent Bridge read-only: heartbeat and status | Feature | P2 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-117](#gyst-vn-117) | Agents page and run detail (read-only) | UI | P2 | NORMAL | Claude/Codex implementable | M |
| [GYST-VN-118](#gyst-vn-118) | Confirm the classification of meaningful GitHub writes | Documentation | P2 | NOT AGENT-ELIGIBLE | Human-only | S |
| [GYST-VN-119](#gyst-vn-119) | Grant the GYST App Issues write | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-120](#gyst-vn-120) | Create issues and edit title/body from GYST | Feature | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-121](#gyst-vn-121) | Edit labels and status, assign an agent | Feature | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-122](#gyst-vn-122) | Comment on issues | Feature | P3 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-123](#gyst-vn-123) | Label normalizer (single automated status writer) | Feature | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-124](#gyst-vn-124) | Verify & close from GYST (re-auth) | Feature | P2 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-125](#gyst-vn-125) | Cloudflare Access validation and machine-endpoint bypass inventory | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | M |
| [GYST-VN-126](#gyst-vn-126) | Bridge execution templates and path policy | Feature | P2 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-127](#gyst-vn-127) | Repository mirror and worktree isolation | Feature | P2 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-128](#gyst-vn-128) | Risk classifier in the bridge | Feature | P2 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-129](#gyst-vn-129) | Approval records and start_run commands in GYST | Feature | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-130](#gyst-vn-130) | Set agent:ready from GYST | Feature | P2 | ELEVATED | Claude/Codex implementable | M |
| [GYST-VN-131](#gyst-vn-131) | Agent dispatch: queue, run and stop | Feature | P2 | PRIVILEGED | Claude/Codex implementable | L |
| [GYST-VN-132](#gyst-vn-132) | Agent review flow | Feature | P2 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-133](#gyst-vn-133) | Agent limits configuration and cleanup | Feature | P2 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-134](#gyst-vn-134) | Agent Attention conditions | Feature | P2 | NORMAL | Claude/Codex implementable | S |
| [GYST-VN-135](#gyst-vn-135) | Enable agent dispatch for gyst-lifeops (first supervised runs) | Operations | P2 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-136](#gyst-vn-136) | Evaluate adding SamVision to the agent allowlist | Research | P3 | NOT AGENT-ELIGIBLE | Human-only | S |

### M6 — Controlled operations

| ID | Name | Type | Pri | Risk | Eligibility | Cx |
|---|---|---|---|---|---|---|
| [GYST-VN-137](#gyst-vn-137) | Action runner (pull-only, fixed allow-list) | Infrastructure | P3 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-138](#gyst-vn-138) | Deploy the action runner | Operations | P3 | NOT AGENT-ELIGIBLE | NOT AGENT-ELIGIBLE operational work | S |
| [GYST-VN-139](#gyst-vn-139) | System actions: run backup, restart allow-listed service | Feature | P3 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-140](#gyst-vn-140) | n8n Run Now (tailnet, confirm) | Feature | P3 | ELEVATED | Claude/Codex implementable | S |
| [GYST-VN-141](#gyst-vn-141) | Bounded, redacted service logs (tailnet, step-up) | Feature | P3 | PRIVILEGED | Claude/Codex implementable | M |
| [GYST-VN-142](#gyst-vn-142) | Console link via Tailscale SSH (tailnet only) | Feature | P3 | PRIVILEGED | Human-assisted | S |

---

## Phase 0 — Foundation (M0)

<a id="gyst-vn-001"></a>
### GYST-VN-001 — Resolve the vNext implementation branch baseline

- **ID:** GYST-VN-001
- **NAME:** Resolve the vNext implementation branch baseline
- **PROJECT:** Governance
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Operations
- **PRIORITY:** P0
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-only (Jay)
- **SUMMARY:** Decide which branch vNext work is based on and merged into, and land the audit/vnext documents there.
- **CURRENT PROBLEM:** Production runs `feature/health-hardening` (dab7995). `master` has unpushed commits, `stash@{2}` exists and `audit/vnext` holds all vNext decisions. Stage 0 recorded master-vs-feature as unresolved, and no vNext PR can have a defined base until it is resolved.
- **DETAILED REQUIREMENTS:**
  - Jay decides the integration branch for vNext work (for example keep `feature/health-hardening` as the trunk, or promote it to `master`)
  - Jay decides what happens to the unpushed `master` commits and `stash@{2}` (the agents keep hands off)
  - `docs/audit/vnext/` reaches the chosen base via a normal reviewed merge or PR
  - `AGENTS.md`/`docs/AI_HANDOFF.md` state the base branch and point to `14`–`17`
- **PROPOSED BEHAVIOR:** Every later task branches from, and opens its PR against, one named base branch.
- **RELEVANT FILES / LIKELY AREAS:** Git branches only; `AGENTS.md`, `docs/AI_HANDOFF.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** None
- **ACCEPTANCE CRITERIA:**
  - The base branch is named in `AGENTS.md` and `docs/AI_HANDOFF.md`
  - `docs/audit/vnext/14`–`17` are present on the base branch
  - No history rewritten; no stash or unpushed commit discarded without Jay's explicit decision
- **TEST PLAN:** `git log`/`git status` on the base; CI green on the base after the docs merge.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Docs-only merge; revert the merge commit if needed.
- **OUT OF SCOPE:** Any code change; deleting old branches.
- **KNOWN RISKS:** Choosing the wrong trunk would force rebases later. The docs merge is low risk.
- **SUGGESTED IMPLEMENTER:** Jay (decision and merge); an agent may prepare the PR description
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-002"></a>
### GYST-VN-002 — Record accepted vNext decisions as ADRs and update ARCHITECTURE.md

- **ID:** GYST-VN-002
- **NAME:** Record accepted vNext decisions as ADRs and update ARCHITECTURE.md
- **PROJECT:** Governance
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Documentation
- **PRIORITY:** P2
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Turn the binding decisions (`05A`, `08A`, `10A`, `13A`) into short ADRs and keep `docs/ARCHITECTURE.md` describing the current system.
- **CURRENT PROBLEM:** The decisions exist only as audit files. The working agreement asks for ADRs for material decisions and an accurate ARCHITECTURE.md.
- **DETAILED REQUIREMENTS:**
  - ADRs (Context / Decision / Alternatives / Why / Consequences / Future), for example: `0001` app timezone and local-day rule, `0002` Attention as a derived model, `0003` two-listener public/tailnet split, `0004` Agent Bridge execution model, `0005` GitHub Issues canonical with labels, `0006` vNext design system hierarchy
  - ADRs link to the audit file that holds the full reasoning; no duplication of long text
  - ARCHITECTURE.md is updated only for what exists now, with the target marked as target
- **PROPOSED BEHAVIOR:** Future agents find the decision record without reading 20 audit files.
- **RELEVANT FILES / LIKELY AREAS:** `docs/adr/`, `docs/ARCHITECTURE.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - One ADR per listed decision, each under about 80 lines
  - ARCHITECTURE.md has no claim about unbuilt features presented as current
- **TEST PLAN:** Link check (relative links resolve); no secrets (grep for token/password patterns).
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Docs only; revert.
- **OUT OF SCOPE:** Rewriting the audit files.
- **KNOWN RISKS:** ADRs drifting from the audit files; mitigated by linking rather than copying.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-003"></a>
### GYST-VN-003 — One app timezone and local-day correctness

- **ID:** GYST-VN-003
- **NAME:** One app timezone and local-day correctness
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Bug
- **PRIORITY:** P0
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Introduce a single configured app timezone (America/Winnipeg) and use it for every calendar-day computation: tasks, completions, trackers, prayer, Health and quick capture.
- **CURRENT PROBLEM:** Stage 2A S2-02 / 2B D1: task, tracker and completion dates use `Today()` in container UTC, while Health and prayer use Winnipeg time and some forms use browser dates. Between 18:00/19:00 local and midnight, new rows land on tomorrow's date.
- **DETAILED REQUIREMENTS:**
  - One clock/timezone component (for example `services.Clock` with `Now()` and `Today()` returning a local date) created from config; the prayer timezone setting becomes a reader of it, not a second source
  - Replace every day-bucketing `time.Now()`/`Today()` call in `internal/services` and `internal/handlers` (tasks, completions, trackers, check-ins, reminders, courses, Health day, prayer, quick capture)
  - Stored day values remain `YYYY-MM-DD` local dates; instants are not changed by this task (see the UTC-instants task)
  - Browser-supplied dates are validated but defaults come from the server's local day
  - Injectable clock so tests can pin time
  - **No automatic rewrite of existing rows** (separate decision task)
- **PROPOSED BEHAVIOR:** At 23:30 Winnipeg time a completion is recorded for today's local date, and it still is at 00:30 UTC. Today, Health and prayer agree on what "today" is.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/lifeops.go`, `internal/services/prayer.go`, `internal/services/health.go`, `internal/services/course.go`, `internal/services/calendar.go`, `internal/handlers/handlers.go`, `internal/handlers/today_view.go`, `internal/handlers/health_progress.go`, `internal/app/app.go`
- **DESIGN REFERENCES:** Stage 2A S2-02; Stage 2B D1
- **SECURITY REFERENCES:** Not security. Correctness foundation for Now, Attention and every Today feature.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - A single timezone source; `grep` finds no remaining day-bucketing `time.Now()` outside the clock
  - Table-driven tests at 23:59 local, 00:00 local, 00:00 UTC (18:00/19:00 local, CST and CDT), and the two DST transition days, for task completion, tracker entry, prayer check-in, Health day and quick capture
  - Existing data untouched (row counts and values unchanged on a backup copy before and after)
- **TEST PLAN:** `go test ./...` including the new clock table tests; `go test -race ./...`; a manual check on a local copy with the clock pinned to 23:30 local.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Code-only; revert the PR. No data migration to undo.
- **OUT OF SCOPE:** Remediating historical wrong-day rows; changing stored instant formats; multi-timezone support.
- **KNOWN RISKS:** Many call sites: a missed site keeps the bug. Mitigate with a grep checklist in the PR and a lint-style test that forbids `time.Now()` in day-bucketing packages.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-004"></a>
### GYST-VN-004 — UTC instants for new audit and event records

- **ID:** GYST-VN-004
- **NAME:** UTC instants for new audit and event records
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Infrastructure
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Define and enforce the convention that every stored instant in new tables is UTC RFC 3339 with `Z`, with display converted to the app timezone.
- **CURRENT PROBLEM:** Stage 6 N.7 / G9: new tables (audit, attention, runs, credentials) need one instant format. Existing tables mix formats.
- **DETAILED REQUIREMENTS:**
  - A helper (for example `timeutil.FormatInstant`/`ParseInstant`) producing `2026-09-27T14:03:00Z`
  - A display helper converting instants to the app timezone (uses the clock from the local-day task when present)
  - Documented rule in ARCHITECTURE.md: instants UTC; day buckets stored as explicit local dates
  - Existing columns are not migrated
- **PROPOSED BEHAVIOR:** New tables from later tasks all use one format, which makes Attention ordering and audit retention trivial.
- **RELEVANT FILES / LIKELY AREAS:** New `internal/timeutil/` (or in `internal/services`), `docs/ARCHITECTURE.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.7, gate G9.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Helper with round-trip tests including sub-second and offset inputs
  - Rule documented
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert; nothing uses it yet.
- **OUT OF SCOPE:** Converting existing columns.
- **KNOWN RISKS:** Low. Pairs with the local-day task; keep the two in separate PRs.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-005"></a>
### GYST-VN-005 — Minimal audit_events store

- **ID:** GYST-VN-005
- **NAME:** Minimal audit_events store
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add the `audit_events` table and a small writer service, as specified in `09` §M, so security events can be recorded.
- **CURRENT PROBLEM:** GYST keeps no record of logins, failures or privileged actions (S2-01, S2-05).
- **DETAILED REQUIREMENTS:**
  - Additive migration `YYYYMMDDHHMMSS_audit_events.sql` with the `09` §M columns; UTC instants
  - `AuditService.Record(ctx, event)` with a fixed action list; `detail` truncated to 200 chars and never containing secrets, bodies or cookie values
  - `actor_ref` for sessions uses the first 8 chars of the session hash only
  - Retention pruning is **not** in this task (see the audit viewer task)
- **PROPOSED BEHAVIOR:** Other tasks call one writer; nothing is shown in the UI yet.
- **RELEVANT FILES / LIKELY AREAS:** `migrations/`, `internal/repository/audit.go`, `internal/services/audit.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §M; gate G2; S2-01, S2-05.
- **DEPENDENCIES:** GYST-VN-004 (UTC instants for new audit and event records)
- **ACCEPTANCE CRITERIA:**
  - Migration additive with a unique timestamp prefix (CI check passes)
  - Test proves that a record containing a password-like field is rejected or stripped
  - Foreign keys and PRAGMA behaviour unchanged
- **TEST PLAN:** `go test ./...`, migration applied to a copy of a backup; `PRAGMA foreign_key_check` clean on the copy.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Additive table; roll back by reverting code (the table can stay unused).
- **OUT OF SCOPE:** Audit UI, retention, export.
- **KNOWN RISKS:** Logging sensitive detail by mistake; mitigated by the field allowlist and a negative test.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-006"></a>
### GYST-VN-006 — Login throttling

- **ID:** GYST-VN-006
- **NAME:** Login throttling
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P0
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Throttle failed password attempts on `/login` so the public login cannot be brute-forced.
- **CURRENT PROBLEM:** S2-01: the public login at the Cloudflare hostname has no rate limit, lockout or delay.
- **DETAILED REQUIREMENTS:**
  - Per-client-key and global failure counters (client key = `CF-Connecting-IP` only when the request came through the trusted tunnel path, otherwise the remote address)
  - Progressive delay, then a temporary block (for example 5 failures in 15 min: 15-min block for that key; global ceiling: 50 failures/hour → all logins need a 30 s wait)
  - Constant generic error text; no hint of account existence or remaining attempts
  - Successful login resets that key's counter
  - In-memory store is acceptable (single process), documented as reset on restart
- **PROPOSED BEHAVIOR:** Repeated wrong passwords slow down, then block for a while; Jay's normal login is unaffected.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/auth.go`, `internal/handlers/middleware.go`, `internal/services/auth.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §C; gate G1; `10A` D-S1 foundation list; S2-01.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Negative test: the 6th failure within the window is refused even with the correct password until the block expires
  - Test: spoofed `CF-Connecting-IP` from a non-tunnel source does not change the key
  - Test: success resets the counter; global ceiling engages
- **TEST PLAN:** `go test ./...` including the new throttle tests; `go test -race ./...`; manual wrong-password run against a local instance.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert the PR; there is no data change.
- **OUT OF SCOPE:** CAPTCHA, account lockout requiring admin action, passkeys (later).
- **KNOWN RISKS:** Locking Jay out; mitigated by a time-based block (no permanent lockout) and a documented tailnet path.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-007"></a>
### GYST-VN-007 — Failed-login audit events

- **ID:** GYST-VN-007
- **NAME:** Failed-login audit events
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Record `login.failed`, `login.succeeded`, `logout` and throttle blocks in `audit_events`.
- **CURRENT PROBLEM:** Failed logins are invisible today (S2-01).
- **DETAILED REQUIREMENTS:**
  - Write the events via the audit writer with ingress (public/tailnet) and a sanitized reason
  - Never record the submitted password or username value beyond a fixed 'owner' label
  - One structured log line per event too (once the logging task exists)
- **PROPOSED BEHAVIOR:** Jay can later see failed attempts (the viewer comes in Phase 6).
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/auth.go`, `internal/services/auth.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §M; gate G1/G2.
- **DEPENDENCIES:** GYST-VN-005 (Minimal audit_events store), GYST-VN-006 (Login throttling)
- **ACCEPTANCE CRITERIA:**
  - Tests prove each event is written once, with no secret fields
  - Throttle block writes `login.failed` with detail `throttled`
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Audit viewer; alerting.
- **KNOWN RISKS:** Same files as throttling; run strictly after it.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-008"></a>
### GYST-VN-008 — Scheduled expired-session cleanup

- **ID:** GYST-VN-008
- **NAME:** Scheduled expired-session cleanup
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Run the expired-session sweep on a schedule inside the process and prove it runs.
- **CURRENT PROBLEM:** S2-11: `SweepExpiredSessions` exists but the audit found it is never scheduled, so expired rows accumulate.
- **DETAILED REQUIREMENTS:**
  - Hourly ticker started from `internal/app/app.go`, stopped on shutdown
  - Logs the count removed (no tokens)
  - Test with an injected clock
- **PROPOSED BEHAVIOR:** Expired sessions disappear within an hour.
- **RELEVANT FILES / LIKELY AREAS:** `internal/app/app.go`, `internal/services/auth.go`, `internal/repository/sessions.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** S2-11; `10A` D-S1 foundation list.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Test: an expired row is deleted by the scheduled sweep; a live row is not
  - Goroutine exits on context cancel (race test clean)
- **TEST PLAN:** `go test -race ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Absolute lifetime (Phase 6), sign out everywhere.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-009"></a>
### GYST-VN-009 — Harden host file permissions and record verification

- **ID:** GYST-VN-009
- **NAME:** Harden host file permissions and record verification
- **PROJECT:** Server / host operations
- **REPOSITORY:** Host (`/srv/server`), runbook in `JayDalool/gyst-lifeops`
- **TYPE:** Operations
- **PRIORITY:** P0
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Make secrets, databases and backups on the host readable only by their owner, and record a verification before any agent or bridge account is created.
- **CURRENT PROBLEM:** S2-04 / `09` §I.1: 3 `.env` files and 45 DB/backup files were world-readable. Any new local account (such as `gyst-agent`) could read them.
- **DETAILED REQUIREMENTS:**
  - An agent drafts a runbook with exact `find`-based check commands that list offending paths **without printing contents**
  - Jay applies: `.env*` `0600`; `data/` `0700`; DB/backup files `0600`; backup dirs `0700`; `/srv/server/private` `0700`
  - Verify the GYST, n8n and SamVision containers still start and read their files (current containers run as root; record which ones)
  - Record the verification output (paths and modes only) in `docs/tasks/`
  - **Binding (O-1):** the service-account task cannot start until this record exists
- **PROPOSED BEHAVIOR:** No world-readable secret or database file remains under `/srv/server`.
- **RELEVANT FILES / LIKELY AREAS:** Host paths under `/srv/server`; runbook in `docs/runbooks/file-permissions.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.11, §I.1; gate G4; `10A` O-1; S2-04.
- **DEPENDENCIES:** None
- **ACCEPTANCE CRITERIA:**
  - The check command returns no world-readable secret/DB/backup files
  - All services healthy after the change
  - Verification record committed (no secret values)
- **TEST PLAN:** Run the check before and after; container health checks; login to GYST works.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Restore previous modes from the recorded before-list (modes only).
- **OUT OF SCOPE:** Changing container users (non-root task); moving secrets to a vault.
- **KNOWN RISKS:** A container that reads a file as a non-root UID may fail; checked service by service.
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-010"></a>
### GYST-VN-010 — Scheduled GYST backup job

- **ID:** GYST-VN-010
- **NAME:** Scheduled GYST backup job
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Infrastructure
- **PRIORITY:** P0
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Run `lifeops backup` daily on a schedule, write a status file, and keep a bounded local history.
- **CURRENT PROBLEM:** S2-03: backups exist only when run by hand; there's no schedule, status or retention.
- **DETAILED REQUIREMENTS:**
  - A host systemd timer (or cron) invoking `docker compose exec gyst-lifeops /app/lifeops backup` daily
  - Writes `data/backups/last_run.json` (time UTC, file, size, integrity result), mode `0600` in a `0700` dir
  - Local retention: 7 daily copies
  - Non-zero exit on failure; a push hook placeholder for Uptime Kuma (wired in Phase 7)
  - Script lives in the repo (`scripts/backup-daily.sh`); Jay installs the timer
- **PROPOSED BEHAVIOR:** A verified backup exists every day without Jay remembering.
- **RELEVANT FILES / LIKELY AREAS:** `scripts/`, `internal/app/backup_cmd.go` (status file), host timer unit (documented in `docs/runbooks/backup.md`).
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.1; gate G5; S2-03.
- **DEPENDENCIES:** GYST-VN-009 (Harden host file permissions and record verification)
- **ACCEPTANCE CRITERIA:**
  - Two consecutive scheduled runs observed with `integrity_check` = ok in `last_run.json`
  - Retention keeps exactly 7
  - The backup and status files are not world-readable
- **TEST PLAN:** `go test ./...` for the status writer; manual run of the script against a local copy; Jay confirms timer runs on the host.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable the timer; the script is additive.
- **OUT OF SCOPE:** Off-host copy, restore test.
- **KNOWN RISKS:** Disk usage growth; bounded by retention. The script path counts as PRIVILEGED under the path policy.
- **SUGGESTED IMPLEMENTER:** Agent writes the script and status writer; Jay installs and enables the timer
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-011"></a>
### GYST-VN-011 — Decide remediation for historical wrong-day rows

- **ID:** GYST-VN-011
- **NAME:** Decide remediation for historical wrong-day rows
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Research
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Measure how many existing rows carry the UTC day instead of the local day, then decide whether to leave, annotate or correct them.
- **CURRENT PROBLEM:** Rows created between local evening and UTC midnight before the local-day fix may be dated a day late. Their exact creation instant may not be recoverable for every table.
- **DETAILED REQUIREMENTS:**
  - An agent writes read-only analysis queries run **against a backup copy**, never the live DB
  - Report per table: candidate rows, whether a creation instant exists to prove the correct day, sample (non-private) counts
  - Options: A leave as is; B annotate only; C correct rows with provable instants via a reviewed, backed-up, one-off operation
  - Jay decides; any correction is a separate NOT AGENT-ELIGIBLE operation with its own backup
- **PROPOSED BEHAVIOR:** A written decision; no data changed by this task.
- **RELEVANT FILES / LIKELY AREAS:** Analysis SQL in `docs/tasks/`; no application files.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `AGENTS.md` data rules: never auto-repair production; backup before any production operation.
- **DEPENDENCIES:** GYST-VN-003 (One app timezone and local-day correctness), GYST-VN-010 (Scheduled GYST backup job)
- **ACCEPTANCE CRITERIA:**
  - Decision recorded with counts and the chosen option
  - No production write occurred (evidence: queries ran on a backup file path)
- **TEST PLAN:** Queries run on a restored copy; integrity_check before and after on the copy.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Nothing to roll back (read-only).
- **OUT OF SCOPE:** Performing any correction.
- **KNOWN RISKS:** Misclassifying legitimate late-evening UTC-dated rows; mitigate by only acting where an instant proves the day.
- **SUGGESTED IMPLEMENTER:** Agent drafts the runbook, script or analysis; Jay executes on the host or decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-012"></a>
### GYST-VN-012 — Off-host backup copy

- **ID:** GYST-VN-012
- **NAME:** Off-host backup copy
- **PROJECT:** Server / host operations
- **REPOSITORY:** Host; script template in `JayDalool/gyst-lifeops`
- **TYPE:** Operations
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Copy the daily GYST backup to a second machine or storage with 7 daily / 4 weekly / 6 monthly retention.
- **CURRENT PROBLEM:** S2-03: every copy lives on the same disk as production.
- **DETAILED REQUIREMENTS:**
  - Jay chooses the destination (open question: second machine on the tailnet vs. encrypted cloud storage)
  - Agent drafts the copy script (for example `rsync`/`rclone` with encryption where cloud), with no secret in the repo
  - Credentials for the destination stored `0600` outside the repo
  - Status recorded next to `last_run.json`
- **PROPOSED BEHAVIOR:** Losing the server does not lose GYST data.
- **RELEVANT FILES / LIKELY AREAS:** `scripts/backup-offsite.sh` (template), host credentials outside the repo.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.2; gate G5.
- **DEPENDENCIES:** GYST-VN-010 (Scheduled GYST backup job)
- **ACCEPTANCE CRITERIA:**
  - A copy exists off-host with the retention shape
  - The destination credential is not readable by other users
- **TEST PLAN:** Jay lists the remote copy and compares size/hash with the local backup.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable the timer.
- **OUT OF SCOPE:** Backing up other apps (n8n Postgres is a separate task).
- **KNOWN RISKS:** Cloud destination exposes health data if unencrypted; require client-side encryption for any non-owned storage.
- **SUGGESTED IMPLEMENTER:** Agent drafts the runbook, script or analysis; Jay executes on the host or decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-013"></a>
### GYST-VN-013 — Restore runbook and one tested restore

- **ID:** GYST-VN-013
- **NAME:** Restore runbook and one tested restore
- **PROJECT:** Server / host operations
- **REPOSITORY:** Runbook in `JayDalool/gyst-lifeops`; execution on host
- **TYPE:** Operations
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Write the restore runbook and perform one real restore into a scratch path, checking integrity, foreign keys and row counts.
- **CURRENT PROBLEM:** S2-03: a restore has never been tested.
- **DETAILED REQUIREMENTS:**
  - Runbook covers stopping the app, restoring the file, WAL handling, starting, and verification
  - Test restore goes to a scratch path, never over `data/lifeops.db`
  - Checks: `PRAGMA integrity_check`, `PRAGMA foreign_key_check`, row counts of the main tables vs. the source backup
  - Monthly repeat is documented (the reminder becomes an Attention item later)
- **PROPOSED BEHAVIOR:** Jay knows a backup is usable and how long a restore takes.
- **RELEVANT FILES / LIKELY AREAS:** `docs/runbooks/restore.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.3; gate G5.
- **DEPENDENCIES:** GYST-VN-010 (Scheduled GYST backup job)
- **ACCEPTANCE CRITERIA:**
  - One restore recorded with the three checks passing and a time taken
  - No production file was overwritten
- **TEST PLAN:** As described in the runbook.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Scratch files deleted after the test.
- **OUT OF SCOPE:** Automated monthly restore job.
- **KNOWN RISKS:** Restoring over production by mistake; mitigated by a scratch-path-only runbook.
- **SUGGESTED IMPLEMENTER:** Agent drafts the runbook, script or analysis; Jay executes on the host or decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-014"></a>
### GYST-VN-014 — /healthz endpoint

- **ID:** GYST-VN-014
- **NAME:** /healthz endpoint
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Infrastructure
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add an unauthenticated `/healthz` that reports only that the process is up and the database answers.
- **CURRENT PROBLEM:** S2-05: there is no health endpoint and no Docker healthcheck.
- **DETAILED REQUIREMENTS:**
  - `GET /healthz` → `200 ok` or `503` with no body detail; runs `SELECT 1` with a 1 s timeout
  - Auth-exempt; exposes no version, path or data
  - Initially on the existing listener; it moves to the internal listener when the dual-listener task lands; Caddy doesn't route it publicly
- **PROPOSED BEHAVIOR:** Docker and Uptime Kuma can tell whether GYST is alive.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/handlers.go`, `internal/handlers/middleware.go` (exempt list).
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.4; gate G6.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Test: 200 when the DB is reachable, 503 when closed
  - Response has no identifying detail
- **TEST PLAN:** `go test ./...`; `curl` locally.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Readiness detail pages; metrics.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-015"></a>
### GYST-VN-015 — Structured application logging and sanitized client errors

- **ID:** GYST-VN-015
- **NAME:** Structured application logging and sanitized client errors
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Log server errors, auth events and provider failures as structured lines, and stop returning internal error text to browsers.
- **CURRENT PROBLEM:** S2-05: no error, request or auth logs. S2-06: internal error strings reach the browser.
- **DETAILED REQUIREMENTS:**
  - `log/slog` JSON handler to stdout with a request id
  - One line per 5xx, auth event and provider failure; never bodies, cookies, tokens or env values
  - Browser gets a generic message plus the request id; detail only in logs
  - A central helper replaces direct `http.Error(w, err.Error(), …)` calls
- **PROPOSED BEHAVIOR:** Problems leave a trace Jay can find; users see a clean message.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/handlers.go`, `internal/handlers/middleware.go`, `internal/app/app.go`, other handlers using `err.Error()`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.5; gates G6/G14; S2-05, S2-06.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - `grep` finds no `err.Error()` written to responses
  - Test: a forced repository error returns generic text and logs one line with the request id
  - Negative test: a request with an `Authorization` header or password field never appears in the log
- **TEST PLAN:** `go test ./...`; manual forced error locally.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Log shipping, dashboards, log viewing in GYST.
- **KNOWN RISKS:** `handlers.go` is a contention hotspot; land early and small.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-016"></a>
### GYST-VN-016 — Log rotation and Docker healthcheck in Compose

- **ID:** GYST-VN-016
- **NAME:** Log rotation and Docker healthcheck in Compose
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Infrastructure
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Bound the GYST container logs and add a healthcheck using `/healthz`.
- **CURRENT PROBLEM:** S2-05: container logs are unbounded; Docker has no health signal for GYST.
- **DETAILED REQUIREMENTS:**
  - `logging: json-file` with `max-size: 10m`, `max-file: 5` for GYST containers
  - `healthcheck` calling `/healthz` (using the binary or a tiny probe, since the image may not have curl)
  - Jay deploys; the agent doesn't restart services
- **PROPOSED BEHAVIOR:** Logs can't fill the disk; `docker ps` shows GYST health.
- **RELEVANT FILES / LIKELY AREAS:** `docker-compose.yml`, `Dockerfile` (probe only if needed).
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.6; gate G6.
- **DEPENDENCIES:** GYST-VN-014 (/healthz endpoint)
- **ACCEPTANCE CRITERIA:**
  - `docker compose config` valid
  - After Jay's deploy, the container reports `healthy`
  - Log files rotate at the size limit (verified in a local compose run)
- **TEST PLAN:** Local `docker compose up` in a scratch directory; Jay's production deploy with before/after `docker ps`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert the compose change and redeploy.
- **OUT OF SCOPE:** Other stacks' logging.
- **KNOWN RISKS:** Compose is PRIVILEGED path; one reviewed change at a time.
- **SUGGESTED IMPLEMENTER:** Agent edits Compose; Jay deploys
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-017"></a>
### GYST-VN-017 — OPS1 production foreign-key audit

- **ID:** GYST-VN-017
- **NAME:** OPS1 production foreign-key audit
- **PROJECT:** Server / host operations
- **REPOSITORY:** Host; ledger in `JayDalool/gyst-lifeops`
- **TYPE:** Operations
- **PRIORITY:** P1
- **PHASE:** 0 — Foundation · Milestone M0
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-only (Jay)
- **SUMMARY:** Run the pending `PRAGMA foreign_key_check` on production per the existing OPS1 procedure and record the result.
- **CURRENT PROBLEM:** OPS1 status is UNKNOWN; B1 FK enforcement is only certified once production has zero violations.
- **DETAILED REQUIREMENTS:**
  - Follow `docs/tasks/OPS1-production-fk-audit.md`: backup, stop, check, start
  - Zero rows → record certified. Any rows → **stop** and report; no repair
- **PROPOSED BEHAVIOR:** Confidence that FK enforcement matches production data.
- **RELEVANT FILES / LIKELY AREAS:** `docs/tasks/OPS1-production-fk-audit.md`, `docs/AI_HANDOFF.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `AGENTS.md` guardrail 1 (never touch production without backup; never auto-repair).
- **DEPENDENCIES:** GYST-VN-010 (Scheduled GYST backup job)
- **ACCEPTANCE CRITERIA:**
  - Result recorded with date and row count (no private row contents)
- **TEST PLAN:** The procedure itself.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Nothing is changed by the check.
- **OUT OF SCOPE:** Repairing violations.
- **KNOWN RISKS:** Brief downtime during the check.
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

## Phase 1 — vNext UI foundation (M1)

<a id="gyst-vn-018"></a>
### GYST-VN-018 — vNext design tokens in the existing ui-* CSS layer

- **ID:** GYST-VN-018
- **NAME:** vNext design tokens in the existing ui-* CSS layer
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 1 — vNext UI foundation · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add the `11` token set (type scale, spacing, radii, colours, state colours, shadows, breakpoints) to `web/static/css/ui-tokens.css`, alongside the existing tokens.
- **CURRENT PROBLEM:** The `ui-*` layer and legacy `styles.css` define overlapping values; vNext needs one token source before any component work.
- **DETAILED REQUIREMENTS:**
  - Type 12/13/14/16/20/28/34; spacing 4/8/12/16/24/32/48; radii 8/12/16/24/pill; `--canvas` #f4f6fa, `--accent` #2563eb, `--xp` #6d5dfc and the state set ok/info/warn/crit/idle with dark equivalents
  - Tokens are namespaced so they can coexist with current tokens; old tokens are not deleted in this task
  - `--xp`/`--xp-soft` documented as identity-only (DD-4)
  - Update `ui_gallery.html` with a token swatch section as the living reference
- **PROPOSED BEHAVIOR:** Components built from now on read one token set; existing screens look the same.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/css/ui-tokens.css`, `web/templates/ui_gallery.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `11` §B–§I; prototype `prototypes/assets/proto.css` (reference, non-binding); `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Existing screens unchanged (before/after screenshots of Today, Health, Projects at 390 and 1280 in both modes)
  - Every `11` §H token present, light and dark
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert the CSS file.
- **OUT OF SCOPE:** Restyling screens; deleting legacy tokens.
- **KNOWN RISKS:** Name collisions with existing tokens; use a `--vn-` prefix if needed and record the choice.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-019"></a>
### GYST-VN-019 — Appearance setting: System / Light / Dark

- **ID:** GYST-VN-019
- **NAME:** Appearance setting: System / Light / Dark
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 1 — vNext UI foundation · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Replace the light/dark toggle with one three-value appearance setting (System default) that follows OS changes live.
- **CURRENT PROBLEM:** The current `data-gyst-theme` toggle has only light/dark and doesn't follow the OS by default (DD-1).
- **DETAILED REQUIREMENTS:**
  - Setting stored per device (existing `theme.js` storage) with values system/light/dark; default system
  - System mode listens to `prefers-color-scheme` changes without reload
  - Set before first paint to avoid a flash
  - Galaxy keeps its immersive dark scene regardless (DD-1); its page chrome follows the setting
  - One setting for the whole app; the control lives in Settings → Appearance (and the existing toggle location until Settings is migrated)
- **PROPOSED BEHAVIOR:** Jay's phone switches GYST between light and dark with iOS automatically, unless he picks a fixed mode.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/theme.js`, `web/templates/layout.html`, `web/templates/settings.html`, `web/static/css/ui-tokens.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; DD-1 in `13A`; `11` §J; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-018 (vNext design tokens in the existing ui-* CSS layer)
- **ACCEPTANCE CRITERIA:**
  - System mode follows an emulated OS change live
  - Manual override persists across reloads
  - No flash of the wrong theme on load
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert JS; the old stored value is mapped (`dark`→dark, `light`→light).
- **OUT OF SCOPE:** Per-area appearance settings.
- **KNOWN RISKS:** Service-worker-cached old `theme.js`; bump the cache version.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-020"></a>
### GYST-VN-020 — Grouped list and row components

- **ID:** GYST-VN-020
- **NAME:** Grouped list and row components
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 1 — vNext UI foundation · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Build the `11` grouped list (inset group, row with tile/title/meta/trailing, chevron, disclosure) as reusable template partials and CSS.
- **CURRENT PROBLEM:** Screens use ad-hoc cards; vNext uses grouped rows throughout (More, Health, Settings, Development).
- **DETAILED REQUIREMENTS:**
  - Template partial(s) for group, row, row-with-action and disclosure row
  - Row min height 44 px; whole row is the target where it navigates
  - Supports leading tile with state, title + meta truncation rules, trailing value/chip/chevron
  - Added to `ui_gallery.html` with empty/one/many examples
- **PROPOSED BEHAVIOR:** One consistent list pattern is available to every migrated screen.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/ui_components.html`, `web/static/css/ui-components.css`, `web/templates/ui_gallery.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `11` §K (lists); prototype `prototypes/more.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-018 (vNext design tokens in the existing ui-* CSS layer)
- **ACCEPTANCE CRITERIA:**
  - Gallery shows all variants in both modes
  - Long titles truncate without overflow at 320 px
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert; nothing uses it yet.
- **OUT OF SCOPE:** Migrating screens.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-021"></a>
### GYST-VN-021 — Icon set and icon/action button components

- **ID:** GYST-VN-021
- **NAME:** Icon set and icon/action button components
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 1 — vNext UI foundation · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Adopt the vNext icon sprite and build icon button, primary/secondary/destructive buttons and the check circle to `11` sizes.
- **CURRENT PROBLEM:** Icons and buttons vary per screen; several icon buttons are below 44 px (Stage 3).
- **DETAILED REQUIREMENTS:**
  - Inline SVG sprite served statically (no icon library dependency)
  - `icon-btn` 44 px hit area with a visually smaller glyph
  - Button variants per `11`; destructive never uses violet or success colours
  - Accessible name required for icon-only buttons (template helper enforces a label argument)
- **PROPOSED BEHAVIOR:** Screens can use consistent, accessible actions.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/ui_icons.html`, `web/static/css/ui-components.css`, `web/templates/ui_gallery.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `11` §L (icons, buttons); prototype `prototypes/assets/proto.js` sprite (reference)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-018 (vNext design tokens in the existing ui-* CSS layer)
- **ACCEPTANCE CRITERIA:**
  - Every icon button in the gallery measures ≥ 44 px
  - A render test fails if an icon button has no label
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Migrating screens.
- **KNOWN RISKS:** Sprite size; keep only icons used by `12`.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-022"></a>
### GYST-VN-022 — State components: chips, banners, empty, stale, error, offline

- **ID:** GYST-VN-022
- **NAME:** State components: chips, banners, empty, stale, error, offline
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 1 — vNext UI foundation · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Build the state vocabulary components: state chip, severity tile, inline banner, empty line, stale chip, error and offline banners.
- **CURRENT PROBLEM:** Stage 3 found missing or inconsistent empty/stale/error states; `11` §I defines one vocabulary.
- **DETAILED REQUIREMENTS:**
  - States ok/info/warn/crit/idle, each with colour + icon + word (never colour alone)
  - Empty state is one line plus one action, not a paragraph
  - Stale chip shows age; error banner never says "healthy"
  - No violet in any state (DD-4)
- **PROPOSED BEHAVIOR:** Every screen can show honest status in the same way.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/ui_components.html`, `web/static/css/ui-components.css`, `web/templates/ui_gallery.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `11` §I; `13` state rules
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-018 (vNext design tokens in the existing ui-* CSS layer), GYST-VN-021 (Icon set and icon/action button components)
- **ACCEPTANCE CRITERIA:**
  - Gallery shows each state in both modes with sufficient contrast
  - Each state is distinguishable in greyscale (icon + word)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Attention logic.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-023"></a>
### GYST-VN-023 — Sheet, popover, inspector and confirm dialog components

- **ID:** GYST-VN-023
- **NAME:** Sheet, popover, inspector and confirm dialog components
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 1 — vNext UI foundation · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Build the bottom sheet (phone), popover (desktop), right inspector (≥ 1280 px) and in-page confirm dialog with focus management.
- **CURRENT PROBLEM:** Galaxy preview, the launcher, the XP chip, re-auth and ⋯ menus all need these; none exist as shared components.
- **DETAILED REQUIREMENTS:**
  - Vanilla JS module: open/close, focus trap, Escape, restore focus, scroll lock, swipe-down optional
  - Close button ≥ 44 px with an accessible name
  - Inspector only at ≥ 1280 px; below that the same content opens as a page or sheet
  - Confirm dialog is in-page (never `window.confirm`)
- **PROPOSED BEHAVIOR:** Consistent overlays that work with keyboard and screen reader.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/app.js` (or a new `ui-sheet.js` module), `web/static/css/ui-components.css`, `web/templates/ui_gallery.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `13` §Overlays; prototype `prototypes/galaxy.html`, `launcher.html`; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-018 (vNext design tokens in the existing ui-* CSS layer), GYST-VN-021 (Icon set and icon/action button components)
- **ACCEPTANCE CRITERIA:**
  - Keyboard: open, Tab stays inside, Escape closes and focus returns
  - VoiceOver announces the sheet title (iPhone check)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Screen-specific content.
- **KNOWN RISKS:** iOS Safari scroll-lock quirks; test on a real iPhone.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-024"></a>
### GYST-VN-024 — 44 px tap targets and overflow guard for shared components

- **ID:** GYST-VN-024
- **NAME:** 44 px tap targets and overflow guard for shared components
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 1 — vNext UI foundation · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Fix undersized targets in the shell and shared components, and extend the UI check script to fail on targets under 44 px and on horizontal overflow.
- **CURRENT PROBLEM:** Stage 3 measured several targets under 44 px and horizontal overflow on narrow screens.
- **DETAILED REQUIREMENTS:**
  - Extend `scripts/check-ui-shell.cjs` (existing tooling) to measure interactive elements and document width at 320/390/1280
  - Fix shell-level offenders (tab bar, top bar, shared buttons)
  - The check measures the Galaxy page's DOM controls; scene targets (56 px) are measured by the Galaxy tasks, not excluded or flattened
  - Screen-specific offenders are recorded for their screen tasks, not fixed here
- **PROPOSED BEHAVIOR:** Regression protection for the most common mobile defect.
- **RELEVANT FILES / LIKELY AREAS:** `scripts/check-ui-shell.cjs`, `web/static/css/ui-shell.css`, `web/static/css/ui-components.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `13` §Touch targets; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-018 (vNext design tokens in the existing ui-* CSS layer)
- **ACCEPTANCE CRITERIA:**
  - The script fails on a deliberately undersized fixture and passes on the shell
  - List of remaining offenders recorded per screen
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Fixing every screen.
- **KNOWN RISKS:** Headless measurements differ from iOS; confirm on a device.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

## Phase 2 — Today + navigation + cleanup (M1)

<a id="gyst-vn-025"></a>
### GYST-VN-025 — Navigation shell: Today · Projects · Health · Learn · More

- **ID:** GYST-VN-025
- **NAME:** Navigation shell: Today · Projects · Health · Learn · More
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Implement the approved target navigation (D11) as the phone tab bar and desktop sidebar, replacing NAV1.
- **CURRENT PROBLEM:** NAV1 is implemented; D11 approved the target but deferred implementation. `scripts/check-ui-shell.cjs` pins NAV1.
- **DETAILED REQUIREMENTS:**
  - Five tabs with icon + label; active state not violet
  - Desktop sidebar ≥ 1024 px: the same five plus a secondary Operate group (System, Apps) **only once those destinations exist** (no empty destinations)
  - Learn tab routes to `/learn`, served by the existing courses list until the Learn task (alias, not a copy)
  - No Attention tab (C1)
  - Update the UI shell test to the D11 spec as an explicit, reviewed spec change (not a weakened test); Jay acknowledges in the PR
  - Old routes keep working (bookmarks, PWA start URL)
  - Galaxy entry points are kept (Projects-list Galaxy tile, command menu "Open Galaxy") until their successors ship (`17A` §C.7)
- **PROPOSED BEHAVIOR:** The app's top-level structure matches vNext on phone and desktop.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/layout.html`, `web/templates/ui_shell.html`, `internal/handlers/ui_shell.go`, `web/static/css/ui-shell.css`, `scripts/check-ui-shell.cjs`, `web/static/service-worker.js`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D11, C1; `12` §More; `13` §Navigation; prototype `prototypes/today.html`; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-020 (Grouped list and row components), GYST-VN-021 (Icon set and icon/action button components), GYST-VN-022 (State components: chips, banners, empty, stale, error, offline), GYST-VN-024 (44 px tap targets and overflow guard for shared components)
- **ACCEPTANCE CRITERIA:**
  - Tabs correct on phone; sidebar correct at 1024/1280
  - Every old top-level URL still resolves
  - Installed PWA opens to Today with the new bar (iPhone check)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert the PR; routes are unchanged so no link breaks.
- **OUT OF SCOPE:** Moving pages between areas beyond D11; Attention entry.
- **KNOWN RISKS:** Highest-contention files in the repo; nothing else touching the shell runs in parallel.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-026"></a>
### GYST-VN-026 — Simplify More to the grouped model

- **ID:** GYST-VN-026
- **NAME:** Simplify More to the grouped model
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Rebuild More as the 4-group list from `12` with no empty destinations.
- **CURRENT PROBLEM:** More has 19 rows in 5 groups including duplicates of tab destinations and placeholders (Stage 3).
- **DETAILED REQUIREMENTS:**
  - Groups: Attention + Goals; Operate (System, Automations, Apps: each shown **only when it exists**); Tools (All tasks & reminders, Calendar, Review & analytics, Import & export, Archive); Settings
  - Remove rows that duplicate tabs (Today, Projects, Health sub-pages, Learn, Development); **keep the Galaxy row until the Projects → Galaxy segment (GYST-VN-039) ships** (`17A` §C.7)
  - Attention row shown only once the Attention page exists; a search field only once the launcher exists
- **PROPOSED BEHAVIOR:** More is a short list of real destinations.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/more.html`, `internal/handlers/ui_shell.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D6, D7; `12` §More; prototype `prototypes/more.html`; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-025 (Navigation shell: Today · Projects · Health · Learn · More), GYST-VN-020 (Grouped list and row components)
- **ACCEPTANCE CRITERIA:**
  - No row leads to a placeholder or 404
  - Row count matches the groups whose destinations exist
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Building System/Apps/Automations.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-027"></a>
### GYST-VN-027 — Terminology migration in the UI: Project / Task / Roadmap

- **ID:** GYST-VN-027
- **NAME:** Terminology migration in the UI: Project / Task / Roadmap
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UX
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Replace Mission, Quest, Destination and Trip as functional UI terms with Project, Task and Roadmap, without renaming persisted data.
- **CURRENT PROBLEM:** D3 retired the gamified functional terms; templates and flash messages still use them.
- **DETAILED REQUIREMENTS:**
  - Change visible strings only (templates, flash text, page titles, aria labels)
  - Keep route paths, table and column names, JSON keys and CSS hooks; add aliases where a URL contains a retired term only if a visible URL matters
  - Keep Galaxy/world/ship/XP personality words; Galaxy changes are strings only (no behaviour)
  - Record a terminology table in `17` style in the PR
- **PROPOSED BEHAVIOR:** Plain labels everywhere; nothing in the database changes.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/*.html`, `internal/handlers/*.go` (user-facing strings), `internal/services/roadmap.go` (messages), `web/static/js/galaxy.js` (preview strings such as "missions" / "Next mission" only).
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D3; `05` §X terminology table; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - `grep -i` for Mission/Quest/Destination/Trip in templates returns only allowed personality uses (listed)
  - No migration, no route change
  - Existing tests pass unchanged except string assertions updated with the terminology table
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Renaming tables/columns/routes.
- **KNOWN RISKS:** Touches many templates: do it early and keep it strings-only to limit conflicts.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-143"></a>
### GYST-VN-143 — Document the production Galaxy baseline

- **ID:** GYST-VN-143
- **NAME:** Document the production Galaxy baseline
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Documentation
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Record the current production Galaxy as the regression baseline before any task that materially changes it (`17A` §F).
- **CURRENT PROBLEM:** `17A`: Galaxy is a protected signature feature. Without a recorded baseline, later tasks can't prove that no behaviour was lost.
- **DETAILED REQUIREMENTS:**
  - Verify `17A` §B at runtime: entry points, Three.js architecture, `/galaxy/data` flow, world generation and position rules, ship, selection, preview, camera, desktop and mobile interactions, filters, the unfinished-project rule, visual details
  - Measure first-frame time and frame time (idle, drag, fly-to-world) on desktop Chrome and a real iPhone (Safari and PWA) with 3, 10 and 30 seeded worlds
  - Screenshots/short recordings at 390×844 and 1280×800 (idle, selected, zoomed, empty, sparse, WebGL fallback) using **fictional seeded data only**
  - Write the parity checklist every later Galaxy task re-checks
  - Store under `docs/design/galaxy-baseline/`; no production data, no private project names
- **PROPOSED BEHAVIOR:** A written, measured baseline that every Galaxy change is compared against.
- **RELEVANT FILES / LIKELY AREAS:** `docs/design/galaxy-baseline/` (new), a local seed script or fixture for fictional projects.
- **DESIGN REFERENCES:** `17A_GALAXY_PRESERVATION.md` §B, §F; `12` §Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Every §B row confirmed or corrected with evidence
  - Performance numbers recorded for desktop and iPhone
  - Parity checklist committed
  - No private data in any media
- **TEST PLAN:** Run the app locally against a seeded copy (never the production DB); `node scripts/check-galaxy.cjs`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800; real iPhone in Safari and the installed PWA (Jay) for the performance and interaction baseline.
- **ROLLBACK CONSIDERATION:** Docs only; revert.
- **OUT OF SCOPE:** Any Galaxy code change.
- **KNOWN RISKS:** iPhone measurements need Jay's device; the agent prepares the seed and the measurement steps.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex (document and seed); Jay runs the iPhone measurements
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-028"></a>
### GYST-VN-028 — Remove database writes from GET requests

- **ID:** GYST-VN-028
- **NAME:** Remove database writes from GET requests
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Bug
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Stop `GET /projects` and `GET /galaxy/data` from writing to the database.
- **CURRENT PROBLEM:** S2-15 / D4: GET handlers reconcile and write state; the CSRF guard assumes safe methods don't change data.
- **DETAILED REQUIREMENTS:**
  - Move reconciliation into the write paths that cause it (task complete, status change) or into an explicit POST
  - GET handlers become read-only
  - Test asserting no row changes after GET on a seeded DB
  - **Galaxy positions stay identical:** compute the deterministic UUID-hash position on read without writing, or persist it at project creation / via an explicit step; never re-randomise existing worlds
  - The `/galaxy/data` JSON contract is unchanged
- **PROPOSED BEHAVIOR:** Viewing never changes data.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/handlers.go`, `internal/handlers/projects_view.go`, `internal/services/galaxy.go`, `internal/services/rocket.go`.
- **DESIGN REFERENCES:** None (no UI change).; `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** S2-15; `02B` D4; CSRF guard assumption.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline), GYST-VN-143 (Document the production Galaxy baseline)
- **ACCEPTANCE CRITERIA:**
  - Tests: row counts and `updated_at` unchanged after both GETs
  - Galaxy and Projects render the same data as before on a seeded DB
  - Test: every world's x/y/z is identical before and after on a seeded DB, including projects that had no stored position
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `go test ./...`; `node scripts/check-galaxy.cjs`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Changing project status semantics.
- **KNOWN RISKS:** Hidden dependence on the GET-time reconcile; measured with the seeded comparison.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-029"></a>
### GYST-VN-029 — Today: remove the Focus / Next up duplication

- **ID:** GYST-VN-029
- **NAME:** Today: remove the Focus / Next up duplication
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Show each open item once on Today by merging the duplicated Focus and Next up blocks into the single Today list.
- **CURRENT PROBLEM:** Stage 3: the same task appears in Focus and in Next up.
- **DETAILED REQUIREMENTS:**
  - One Today list using grouped rows
  - The item chosen as Now (next task) is not repeated in the list
  - Done items collapse into `Done · n`
- **PROPOSED BEHAVIOR:** Today lists each thing once.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/today.html`, `web/templates/today_components.html`, `internal/handlers/today_view.go`, `web/static/css/ui-today.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Today; prototype `prototypes/today.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-020 (Grouped list and row components)
- **ACCEPTANCE CRITERIA:**
  - A seeded day with 5 items shows 5 rows, none duplicated
  - Done collapse works with keyboard
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Now selection logic (separate backend task).
- **KNOWN RISKS:** Same files as other Today tasks; run sequentially.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-030"></a>
### GYST-VN-030 — Today: Now selection logic

- **ID:** GYST-VN-030
- **NAME:** Today: Now selection logic
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** A service function that returns the Now set: next action, next prayer and next timed item, using the app clock.
- **CURRENT PROBLEM:** Today has no Now concept; the next prayer is computed separately, in its own timezone.
- **DETAILED REQUIREMENTS:**
  - `TodayService.Now(ctx)` returns next task (priority, due, then manual order), next prayer (time + name), next timed item (reminder/course session/event) within a horizon
  - Fallback order when there's no open task: next learning session, then next routine
  - Pure function over repositories plus clock; no UI in this task
- **PROPOSED BEHAVIOR:** The Today page can render Now from one deterministic source.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/lifeops.go` (or new `internal/services/today.go`), `internal/services/prayer.go`, `internal/handlers/today_view.go` (view model only).
- **DESIGN REFERENCES:** `12` §Today (Now); `05A` D10
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-003 (One app timezone and local-day correctness)
- **ACCEPTANCE CRITERIA:**
  - Table tests with the pinned clock around prayer boundaries and midnight
  - Deterministic tie-breaking documented
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Presentation.
- **KNOWN RISKS:** Choosing the "right" next task is product behaviour: tie-breaks documented and shown to Jay in the PR.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-031"></a>
### GYST-VN-031 — Today: Now block presentation

- **ID:** GYST-VN-031
- **NAME:** Today: Now block presentation
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Render the Now block at the top of Today: complete-able next action, next prayer (tap → prayer sheet) and next timed item.
- **CURRENT PROBLEM:** Today opens with duplicated lists and no single "what now" answer.
- **DETAILED REQUIREMENTS:**
  - Uses the Now view model
  - Check circle completes the Now item (existing complete endpoint)
  - Prayer sheet shows today's schedule using the sheet component
  - Empty: "Nothing else today" line under Now
- **PROPOSED BEHAVIOR:** Opening GYST answers "what now" in one glance.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/today.html`, `web/templates/today_components.html`, `web/static/css/ui-today.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Today; `13A` DD-3 (Now outranks XP); prototype `prototypes/today.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-030 (Today: Now selection logic), GYST-VN-029 (Today: remove the Focus / Next up duplication), GYST-VN-022 (State components: chips, banners, empty, stale, error, offline), GYST-VN-023 (Sheet, popover, inspector and confirm dialog components)
- **ACCEPTANCE CRITERIA:**
  - Now shows exactly one next action
  - Completing it updates Now without a full reload (or with a quick reload if that's the existing pattern)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Rhythm, status, Attention.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-032"></a>
### GYST-VN-032 — Today: overdue reminders and tasks

- **ID:** GYST-VN-032
- **NAME:** Today: overdue reminders and tasks
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Surface overdue reminders and tasks as a single warn row each on Today, with a "Do today" action.
- **CURRENT PROBLEM:** Overdue items are mixed into lists without distinction or are missed after midnight UTC shifts (fixed by the local-day task).
- **DETAILED REQUIREMENTS:**
  - Overdue = due local date < today's local date and not done
  - One warn row per item ("Overdue since Friday"); many → grouped "n overdue" row
  - "Do today" moves the due date to today (existing update path, CSRF-protected POST)
  - Warnings don't badge (Attention rules)
  - When Attention exists, this computation becomes an Attention source (Phase 5); keep it in the service layer
- **PROPOSED BEHAVIOR:** Jay notices overdue things without them dominating Today.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/lifeops.go`, `internal/handlers/today_view.go`, `web/templates/today.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Today scenario "One overdue task"
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-003 (One app timezone and local-day correctness), GYST-VN-022 (State components: chips, banners, empty, stale, error, offline)
- **ACCEPTANCE CRITERIA:**
  - Tests for overdue boundaries at local midnight
  - "Do today" is a POST with CSRF protection
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Attention page, push.
- **KNOWN RISKS:** Mixes backend + UI: keep the service change and the template change in separate commits.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-033"></a>
### GYST-VN-033 — Today: Rhythm strip (prayer, Athkar, gym)

- **ID:** GYST-VN-033
- **NAME:** Today: Rhythm strip (prayer, Athkar, gym)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Place a compact Rhythm strip directly under Now with prayer check-ins, Athkar and gym/training check-in.
- **CURRENT PROBLEM:** Check-ins are spread across Today sections; DD-2 fixes their content and position (upper portion, D10).
- **DETAILED REQUIREMENTS:**
  - Fixed contents per DD-2; not configurable; no widget framework
  - Uses existing check-in endpoints; each check-in ≥ 44 px with a label
  - Wraps to two rows on narrow phones without overflow; width capped on desktop
- **PROPOSED BEHAVIOR:** Daily rhythm is one tap each, right under Now.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/today.html`, `web/templates/today_components.html`, `web/static/css/ui-today.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `13A` DD-2; `05A` D10; prototype `prototypes/today.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-031 (Today: Now block presentation), GYST-VN-021 (Icon set and icon/action button components)
- **ACCEPTANCE CRITERIA:**
  - Order: header → Now → Rhythm
  - All 8 check-ins reachable and labelled
  - No change to check-in data semantics
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Configurable Rhythm.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-034"></a>
### GYST-VN-034 — Today: quiet Lv chip

- **ID:** GYST-VN-034
- **NAME:** Today: quiet Lv chip
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Show a small tappable level chip in the Today header that opens a sheet with level progress and recent achievements.
- **CURRENT PROBLEM:** XP presentation on Today is larger than DD-3 allows or missing in places.
- **DETAILED REQUIREMENTS:**
  - Chip smaller than Now; uses identity violet only (DD-4 allowed use)
  - Tap opens a sheet using existing XP data
  - No XP cards, banners or motivational blocks on Today
- **PROPOSED BEHAVIOR:** XP stays part of GYST's personality without competing with Now.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/today.html`, `internal/handlers/today_view.go`, `web/static/css/ui-today.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `13A` DD-3, DD-4
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-023 (Sheet, popover, inspector and confirm dialog components), GYST-VN-031 (Today: Now block presentation)
- **ACCEPTANCE CRITERIA:**
  - Chip ≤ the size set in `11`; tap target 44 px
  - Sheet shows level progress from real data
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** New XP mechanics.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-035"></a>
### GYST-VN-035 — Today: exceptions-only status area

- **ID:** GYST-VN-035
- **NAME:** Today: exceptions-only status area
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add the status strip that shows only exceptions (for now: Health sync stale) and nothing when everything is normal.
- **CURRENT PROBLEM:** D9: before System exists Today shows nothing for system health; a status area is still needed for sync staleness.
- **DETAILED REQUIREMENTS:**
  - Strip appears only when there's at least one exception
  - Initial source: Health sync stale (existing `staleAfter` rule)
  - Designed so System/agent chips can be added later without a framework
- **PROPOSED BEHAVIOR:** Jay sees problems on Today; normal days stay clean.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/today.html`, `internal/handlers/today_view.go`, `web/static/css/ui-today.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D9; `12` §Today scenarios
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-022 (State components: chips, banners, empty, stale, error, offline), GYST-VN-031 (Today: Now block presentation)
- **ACCEPTANCE CRITERIA:**
  - No strip on a normal seeded day
  - Stale chip shows the age on a stale seeded day
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** System health indicator (Phase 7).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-036"></a>
### GYST-VN-036 — Today: responsive layout (phone order, desktop two columns)

- **ID:** GYST-VN-036
- **NAME:** Today: responsive layout (phone order, desktop two columns)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 2 — Today + navigation + cleanup · Milestone M1
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Finish Today's layout: phone order header → Now → Rhythm → (Attention) → Today → Status → Health glance; desktop two columns up to 1120 px.
- **CURRENT PROBLEM:** Today's current layout doesn't follow the vNext order and stretches on desktop.
- **DETAILED REQUIREMENTS:**
  - Phone single column in the specified order
  - Desktop: left Now/Rhythm/Attention/Today; right Status/Health glance; no extra widgets
  - Health glance shows real values only, with "—" for missing (never 0)
- **PROPOSED BEHAVIOR:** Today reads well on iPhone and desktop.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/today.html`, `web/static/css/ui-today.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Today (Phone/Desktop); `13` §Breakpoints; prototype `prototypes/today.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-031 (Today: Now block presentation), GYST-VN-033 (Today: Rhythm strip (prayer, Athkar, gym)), GYST-VN-032 (Today: overdue reminders and tasks), GYST-VN-035 (Today: exceptions-only status area)
- **ACCEPTANCE CRITERIA:**
  - Real iPhone PWA check recorded
  - Missing Health values show "—"
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Attention section (Phase 5).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

## Phase 3 — Projects / Galaxy / Learn / Health migrations (M2)

<a id="gyst-vn-037"></a>
### GYST-VN-037 — One project status → state mapping

- **ID:** GYST-VN-037
- **NAME:** One project status → state mapping
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Refactor
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Replace the three disagreeing status-to-state functions with one mapping used by Projects, Galaxy and the rocket route.
- **CURRENT PROBLEM:** `02B` D3: `rocket.go`, `handlers.go` and `projects_view.go` map project status to state differently.
- **DETAILED REQUIREMENTS:**
  - One function in the service layer with a table test
  - All three callers use it
  - No stored value changes
  - Galaxy world states, colours and legend (Active/Upcoming/Locked) unchanged except the listed edge statuses, checked against the Galaxy baseline
- **PROPOSED BEHAVIOR:** A project shows the same state on every screen.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/rocket.go`, `internal/handlers/handlers.go`, `internal/handlers/projects_view.go`, `internal/services/galaxy.go`.
- **DESIGN REFERENCES:** `02B` D3; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-028 (Remove database writes from GET requests), GYST-VN-143 (Document the production Galaxy baseline)
- **ACCEPTANCE CRITERIA:**
  - Table test covers every stored status
  - `grep` shows one mapping
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...`, `go test -race ./...`, `gofmt`, `go vet`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** New statuses.
- **KNOWN RISKS:** Visible state may change for edge statuses; list them in the PR.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-038"></a>
### GYST-VN-038 — Transactional service use cases for multi-write operations

- **ID:** GYST-VN-038
- **NAME:** Transactional service use cases for multi-write operations
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Refactor
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Move complete task/mission, set project status and adopt roadmap into service methods that run in one transaction each.
- **CURRENT PROBLEM:** S2-13 / D2: these operations write several rows from handlers without a transaction and can leave partial state or orphans.
- **DETAILED REQUIREMENTS:**
  - Service methods own the transaction (`db.BeginTx`); handlers call one method
  - Foreign keys stay enforced; no constraint weakened
  - Tests force a failure mid-operation and assert no partial writes
- **PROPOSED BEHAVIOR:** Multi-step changes either fully happen or not at all.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/lifeops.go`, `internal/services/roadmap.go`, `internal/handlers/handlers.go`, `internal/repository/*`.
- **DESIGN REFERENCES:** `02B` D2
- **SECURITY REFERENCES:** S2-13 (data integrity).
- **DEPENDENCIES:** GYST-VN-037 (One project status → state mapping)
- **ACCEPTANCE CRITERIA:**
  - Failure-injection tests pass
  - No handler performs more than one write directly for these operations
- **TEST PLAN:** `go test ./...`, `go test -race ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Other handlers.
- **KNOWN RISKS:** Large diff in `handlers.go`; land before project UI work.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-039"></a>
### GYST-VN-039 — Projects area sub-navigation: Projects · Development · Galaxy · Roadmaps

- **ID:** GYST-VN-039
- **NAME:** Projects area sub-navigation: Projects · Development · Galaxy · Roadmaps
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add the sticky segmented sub-navigation that makes the four Projects areas peers (D2), showing Development only once it exists.
- **CURRENT PROBLEM:** Galaxy and Roadmaps are reached from separate places; Development doesn't exist yet.
- **DETAILED REQUIREMENTS:**
  - Segmented control under the title on phone; sidebar sub-items on desktop
  - The Galaxy segment opens the real `/galaxy` Three.js view (`17A`); Galaxy stays a peer view, never replaced by a list
  - Development segment hidden until the Development area ships (no empty destination)
  - State preserved in the URL (deep-linkable)
- **PROPOSED BEHAVIOR:** All Projects areas are one tap apart.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/projects_list.html`, `web/templates/galaxy.html`, `web/templates/roadmap.html`, `internal/handlers/ui_shell.go`, `web/static/css/ui-projects.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D2; `12` §Projects; prototype `prototypes/projects.html`; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-025 (Navigation shell: Today · Projects · Health · Learn · More)
- **ACCEPTANCE CRITERIA:**
  - Each segment deep-links
  - Sidebar sub-items at ≥ 1024 px
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Development content.
- **KNOWN RISKS:** Touches the shell; not parallel with other shell tasks.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-040"></a>
### GYST-VN-040 — Projects list vNext

- **ID:** GYST-VN-040
- **NAME:** Projects list vNext
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Rebuild the Projects list with grouped rows (orb, name, one signal, progress ring), Active/Planned/Blocked/Paused filters and collapsed Done.
- **CURRENT PROBLEM:** Stage 3: the list is dense and mixes states; completed projects crowd the view.
- **DETAILED REQUIREMENTS:**
  - Default filter Active
  - Completed collapsed as `Done · n`
  - One signal per row (next task or blocked reason)
  - Roadmaps summary row
  - The existing Galaxy entry stays visible (via the sub-nav segment); the list is a peer of Galaxy, not a replacement (`17A` §M)
- **PROPOSED BEHAVIOR:** Jay sees active work first.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/projects_list.html`, `internal/handlers/projects_view.go`, `web/static/css/ui-projects.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Projects; prototype `prototypes/projects.html`; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-039 (Projects area sub-navigation: Projects · Development · Galaxy · Roadmaps), GYST-VN-037 (One project status → state mapping), GYST-VN-027 (Terminology migration in the UI: Project / Task / Roadmap)
- **ACCEPTANCE CRITERIA:**
  - Empty/one/many states
  - Filters are links or buttons with state (no JS-only state)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Project detail.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-041"></a>
### GYST-VN-041 — Project detail vNext

- **ID:** GYST-VN-041
- **NAME:** Project detail vNext
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Rebuild project detail: hero, Next task, sections Tasks/Roadmap/Activity/Resources, and a ⋯ menu for edit, dependencies & proof, move, archive, delete.
- **CURRENT PROBLEM:** Project detail shows everything at once, including destructive actions.
- **DETAILED REQUIREMENTS:**
  - Complete Next task from the hero
  - Delete only via ⋯ → in-page confirm
  - Inspector for a selected task at ≥ 1280 px (optional)
- **PROPOSED BEHAVIOR:** Working a project starts from its next task.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/project_detail.html`, `web/templates/project_components.html`, `internal/handlers/projects_view.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Project Detail; prototype `prototypes/project-detail.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-040 (Projects list vNext), GYST-VN-023 (Sheet, popover, inspector and confirm dialog components), GYST-VN-038 (Transactional service use cases for multi-write operations)
- **ACCEPTANCE CRITERIA:**
  - Destructive actions not visible by default
  - Sections switch without losing scroll on phone
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** New project fields.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-042"></a>
### GYST-VN-042 — Roadmaps list and route page

- **ID:** GYST-VN-042
- **NAME:** Roadmaps list and route page
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Replace the long Roadmap page with a roadmap list and a route page of ordered stops.
- **CURRENT PROBLEM:** Stage 3: the Roadmap page repeats every project in full across about 16 screens of scrolling.
- **DETAILED REQUIREMENTS:**
  - List row: name · "3 of 5 projects" · next project
  - Route page: vertical ordered stops with state
  - Import/export Markdown under ⋯
- **PROPOSED BEHAVIOR:** Roadmaps are scannable.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/roadmap.html`, `web/templates/rocket_route.html`, `internal/services/roadmap.go` (read model only).
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Roadmaps
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-039 (Projects area sub-navigation: Projects · Development · Galaxy · Roadmaps), GYST-VN-027 (Terminology migration in the UI: Project / Task / Roadmap)
- **ACCEPTANCE CRITERIA:**
  - Empty/one/many
  - No project repeated in full
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Weekly review.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-043"></a>
### GYST-VN-043 — Galaxy discoverability

- **ID:** GYST-VN-043
- **NAME:** Galaxy discoverability
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UX
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Make the real Galaxy easy to find from Projects (the Projects → Galaxy segment and the sidebar), a launcher Go result, and keep and improve the existing accessible navigator under the scene.
- **CURRENT PROBLEM:** Stage 3: Galaxy is hard to find. Its entry points today are a Projects-list tile, the command menu, a More row and the direct URL. The existing navigator (world list, category filter, Preview buttons) is below the fold.
- **DETAILED REQUIREMENTS:**
  - The Projects → Galaxy segment and the desktop sidebar open the real `/galaxy` (Three.js scene)
  - Keep the existing entry points until their successors exist (`17A` §C.7)
  - Keep and improve the **existing** accessible navigator. It supplements the scene and is the WebGL-failure fallback; it is never promoted to replace a working scene
  - Legend and hint text visible and plain (terminology per `05` §X)
- **PROPOSED BEHAVIOR:** Galaxy is easy to reach, and usable without precise tapping, while staying a spatial view.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/galaxy.html`, `web/static/js/galaxy.js`, `web/static/css/ui-galaxy.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Galaxy; prototype `prototypes/galaxy.html` (reference); `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-039 (Projects area sub-navigation: Projects · Development · Galaxy · Roadmaps), GYST-VN-143 (Document the production Galaxy baseline)
- **ACCEPTANCE CRITERIA:**
  - Galaxy reachable in one tap from Projects on phone and desktop
  - Navigator Preview opens the same preview as tapping a world
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `node scripts/check-galaxy.cjs` plus the standard UI checks and the `17A` §L checklist.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Replacing or rebuilding the Three.js Galaxy; making Galaxy the default Projects view.
- **KNOWN RISKS:** Galaxy files are one contention group.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-044"></a>
### GYST-VN-044 — Galaxy 56 px touch targets

- **ID:** GYST-VN-044
- **NAME:** Galaxy 56 px touch targets
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Give every world about a 56 px usable touch target (or equivalent comfortable area) and make the zoom, reset and close controls at least 44 px, building on the existing hit logic.
- **CURRENT PROBLEM:** Stage 3: small worlds are hard to hit on a phone. The current hit test is a raycast, then the nearest world whose screen edge is within 28 px (44 px for the selected world), with a 5 px minimum screen radius. The +/−/Reset buttons are small.
- **DETAILED REQUIREMENTS:**
  - Measure the effective target sizes on the baseline first
  - Extend the existing screen-space padding logic (`planetAtPointer`) so the effective diameter is ≥ 56 px at default and max zoom, with a nearest-centre rule for close worlds
  - Keep the gesture model in `galaxy-gestures.js` (tap vs drag vs pinch vs cancel) and extend its tests
  - Controls use 44 px icon buttons
  - Keep Three.js
- **PROPOSED BEHAVIOR:** Worlds are easy to tap on an iPhone without changing the scene.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/galaxy.js`, `web/static/js/galaxy-gestures.js`, `web/static/css/ui-galaxy.css`, `scripts/check-galaxy.cjs`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `13` §Touch targets (56 px Galaxy); `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-143 (Document the production Galaxy baseline)
- **ACCEPTANCE CRITERIA:**
  - Effective target measured ≥ 56 px at default and max zoom
  - Real iPhone check recorded
  - Drag and pinch never trigger selection (existing tests still pass)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `node scripts/check-galaxy.cjs`; standard UI checks; `17A` §L checklist.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Visual redesign; replacing the hit logic or the renderer.
- **KNOWN RISKS:** Overlapping targets when worlds are close; nearest-centre rule.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-045"></a>
### GYST-VN-045 — Galaxy preview close and focus accessibility

- **ID:** GYST-VN-045
- **NAME:** Galaxy preview close and focus accessibility
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Bug
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Make the existing preview's close control clearly visible and ≥ 44 px, add outside-tap close, and return focus to the world or navigator item that opened it.
- **CURRENT PROBLEM:** Stage 3: the preview close control (a small "×" with an aria-label) is hard to see and hit, especially on phones. Escape recentres, but focus handling is incomplete.
- **DETAILED REQUIREMENTS:**
  - Keep the existing preview content and the "Open project" action
  - Close button ≥ 44 px, visible on dark and light surrounds
  - Escape (existing) and outside tap close; closing recentres as today
  - Focus returns to the invoking world (stage) or navigator Preview button
- **PROPOSED BEHAVIOR:** The preview never traps the user, and the Galaxy flow is otherwise unchanged.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/galaxy.js`, `web/templates/galaxy.html`, `web/static/css/ui-galaxy.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `13` §Overlays; `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-044 (Galaxy 56 px touch targets)
- **ACCEPTANCE CRITERIA:**
  - Keyboard and VoiceOver can close the preview
  - Open project still works from the preview
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `node scripts/check-galaxy.cjs`; standard UI checks; `17A` §L checklist.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Preview redesign (see the phone layout task).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-046"></a>
### GYST-VN-046 — Galaxy completed worlds on the outer orbit

- **ID:** GYST-VN-046
- **NAME:** Galaxy completed worlds on the outer orbit
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Include completed projects in Galaxy data with a completed flag, and render them as dimmed, distant, outer-orbit worlds, hidden by default.
- **CURRENT PROBLEM:** `/galaxy/data` currently excludes done/completed/archived projects ("Completed worlds have left the Galaxy"). D4 approves showing them distinctly and hideably.
- **DETAILED REQUIREMENTS:**
  - Additive JSON field (e.g. `completed: true`); the existing fields and the contract are unchanged
  - Completed state comes from the single status mapping
  - Dimmed, smaller glow, outer orbit by a documented rule (or deterministic position kept); never out-shine active worlds
  - **Hidden by default** until the filter task ships the visible Done toggle
  - Never uses a state colour alone to mean done; the navigator marks it in words
  - Performance measured with many completed worlds
- **PROPOSED BEHAVIOR:** History is visible on request and stays quiet.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/galaxy.js`, `internal/services/galaxy.go`, `internal/models/models.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D4; `12` §Galaxy; `17A` §I; `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-037 (One project status → state mapping), GYST-VN-045 (Galaxy preview close and focus accessibility)
- **ACCEPTANCE CRITERIA:**
  - Seeded galaxy with 3 active and 10 completed: active remain dominant when completed are shown
  - Default view unchanged (completed hidden)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `go test ./...`; `node scripts/check-galaxy.cjs`; before/after frame-time comparison.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert; the additive field is ignored by old clients.
- **OUT OF SCOPE:** Archive view; moving existing worlds without approval.
- **KNOWN RISKS:** Performance with many worlds; measured.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-047"></a>
### GYST-VN-047 — Galaxy completed filter / hide

- **ID:** GYST-VN-047
- **NAME:** Galaxy completed filter / hide
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Extend the existing category filter with an Active · Done toggle that applies to both the scene and the navigator; completed worlds are hidden by default, and the choice is remembered per device.
- **CURRENT PROBLEM:** D4 requires completed projects to be filterable and hideable. Today the category filter only filters the navigator list, not the scene.
- **DETAILED REQUIREMENTS:**
  - Toggle state in the URL and local storage (with try/catch)
  - Scene and navigator follow the same filter
  - The existing category filter keeps working
- **PROPOSED BEHAVIOR:** Jay chooses whether history is visible.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/galaxy.js`, `web/templates/galaxy.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D4; `17A` §I; `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-046 (Galaxy completed worlds on the outer orbit)
- **ACCEPTANCE CRITERIA:**
  - Default hides completed
  - Toggle is keyboard accessible with its state announced
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `node scripts/check-galaxy.cjs`; standard UI checks; `17A` §L checklist.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Category management.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-048"></a>
### GYST-VN-048 — Galaxy sparse and empty states

- **ID:** GYST-VN-048
- **NAME:** Galaxy sparse and empty states
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Make Galaxy look intentional with only a few worlds. Investigate orbital composition, camera framing, ambient elements, completed/distant worlds and depth; improve the empty state.
- **CURRENT PROBLEM:** With ≤ 3 active projects the scene looks empty. With none, the page shows a status line and a list link, and no scene renders.
- **DETAILED REQUIREMENTS:**
  - With ≤ 3 worlds: better framing and scale, labels always on, composition that uses the existing core, rings and nebulae; no meaningless clutter
  - Empty: keep the scene (core, stars, ship) with "Create a project to launch your first world" instead of a scene-less page
  - **Never** falls back to a grid of cards
  - Performance unchanged
- **PROPOSED BEHAVIOR:** Galaxy feels deliberate at any size.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/galaxy.js`, `web/static/js/galaxy-camera.js`, `web/templates/galaxy.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Galaxy (sparse); `17A` §G; `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-046 (Galaxy completed worlds on the outer orbit)
- **ACCEPTANCE CRITERIA:**
  - 0, 1, 3 and many worlds verified with screenshots against the baseline
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `node scripts/check-galaxy.cjs` (camera-fit cases for sparse); standard UI checks; `17A` §L checklist.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Over-decoration; judged against the baseline.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-049"></a>
### GYST-VN-049 — Galaxy phone layout and premium preview

- **ID:** GYST-VN-049
- **NAME:** Galaxy phone layout and premium preview
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Size the scene to about 64 vh (minimum 420 px) on phones, and make the preview a readable bottom sheet (side panel on desktop) with clear hierarchy, useful actions and a visible close.
- **CURRENT PROBLEM:** The scene and preview don't fit phone screens well; recent fit work helped, and `12`/`17A` §G define the target.
- **DETAILED REQUIREMENTS:**
  - The preview uses the shared sheet component on phones with the same content, plus name · state · progress · next item · Open project
  - Camera flight to the selected world is preserved
  - The ship (the Oliver sprite that rides with the camera and follows the pointer) is preserved
  - The header hide-on-scroll behaviour is kept or deliberately replaced, and noted
  - Page scroll and scene gestures don't conflict
- **PROPOSED BEHAVIOR:** Galaxy is comfortable and polished on an iPhone.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/galaxy.html`, `web/static/js/galaxy.js`, `web/static/js/galaxy-camera.js`, `web/static/css/ui-galaxy.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Galaxy; `17A` §G; prototype `prototypes/galaxy.html` (reference); `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-045 (Galaxy preview close and focus accessibility), GYST-VN-023 (Sheet, popover, inspector and confirm dialog components)
- **ACCEPTANCE CRITERIA:**
  - Real iPhone Safari and PWA check recorded
  - Preview readable at 320 px
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `node scripts/check-galaxy.cjs`; standard UI checks; `17A` §L checklist.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Replacing the scene or the ship.
- **KNOWN RISKS:** iOS gesture conflicts.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-144"></a>
### GYST-VN-144 — Galaxy visual depth (measured)

- **ID:** GYST-VN-144
- **NAME:** Galaxy visual depth (measured)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P3
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Improve depth and atmosphere within the existing scene: lighting, atmospheres, rings, subtle particles, world variation, orbit paths, transitions and ship movement. Each change must be performance-measured.
- **CURRENT PROBLEM:** `17A` amplification goal 6: the scene can feel richer, but must not trade usability or iPhone performance for spectacle.
- **DETAILED REQUIREMENTS:**
  - Proposal first: a short list of changes with expected cost, reviewed by Jay before implementation
  - Each effect behind reduced-motion rules; no effect that hurts label legibility or targets
  - Before/after frame-time numbers on desktop and iPhone at 10 and 30 worlds
- **PROPOSED BEHAVIOR:** Galaxy feels deeper without becoming slower or harder to use.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/galaxy.js`, `web/static/css/ui-galaxy.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `17A` §D, §J; `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-143 (Document the production Galaxy baseline), GYST-VN-049 (Galaxy phone layout and premium preview), GYST-VN-048 (Galaxy sparse and empty states)
- **ACCEPTANCE CRITERIA:**
  - Proposal approved
  - No regression beyond an agreed budget
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `node scripts/check-galaxy.cjs`; performance comparison; `17A` §L checklist.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert individual effects.
- **OUT OF SCOPE:** Changing the renderer; new 3D libraries.
- **KNOWN RISKS:** Performance on iPhone.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex (proposal reviewed by Jay first)
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-145"></a>
### GYST-VN-145 — Galaxy subtle project-state encoding

- **ID:** GYST-VN-145
- **NAME:** Galaxy subtle project-state encoding
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P3
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Encode useful project information subtly on worlds: progress, active/paused state, completed state and (once Attention exists) attention needed, without turning worlds into dashboards.
- **CURRENT PROBLEM:** `17A` amplification goal 7: Galaxy should communicate meaning, not only look good.
- **DETAILED REQUIREMENTS:**
  - Design proposal first (for example a progress arc or ring, paused as dimmed rotation, attention as a quiet marker plus a label in the preview, **added only once Attention (Phase 5) exists**); reviewed by Jay
  - Reads the single status mapping and Attention; no state stored in Galaxy
  - Encoded information is also present in words in the preview and navigator (never colour alone)
- **PROPOSED BEHAVIOR:** A glance at Galaxy says which worlds are moving, paused or need Jay.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/galaxy.js`, `internal/services/galaxy.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `17A` §D, §O; `13A` DD-4; `17A_GALAXY_PRESERVATION.md` (binding); Stage 7 `prototypes/galaxy.html` is a design reference only, not a substitute for the production Three.js Galaxy
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-143 (Document the production Galaxy baseline), GYST-VN-046 (Galaxy completed worlds on the outer orbit)
- **ACCEPTANCE CRITERIA:**
  - Proposal approved
  - Tests for the data fields
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy regression checklist (`17A` §L) passes and is recorded in the task record:
  - The real Galaxy still exists at `/galaxy`; Three.js/spatial interaction still works
  - Existing project worlds remain accessible (scene and accessible navigator)
  - The ship remains part of the experience (unless this task was explicitly approved to change it)
  - Project selection works (tap, click, keyboard, navigator Preview) and preview/Open project works
  - Desktop interaction works (drag, wheel, hover roam, click/double-click open); mobile works (tap, drag, pinch) on a real iPhone for material changes
  - No existing Galaxy behaviour silently lost versus the baseline parity list; no list/grid substituted for Galaxy
  - The normal Projects list remains available separately; Galaxy remains an alternate/signature project view
  - Performance no worse than the baseline without documented justification (frame time and first frame, desktop and iPhone)
  - Reduced-motion and accessibility behaviour considered and checked
  - Completed-project enhancements don't clutter active worlds; completed projects can be hidden/filtered (once GYST-VN-047 has shipped)
  - `node scripts/check-galaxy.cjs` passes (extended when behaviour is added)
- **TEST PLAN:** `go test ./...`; `node scripts/check-galaxy.cjs`; `17A` §L checklist.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Badges or numbers on every world.
- **KNOWN RISKS:** Clutter; judged against `17A` §I and DD-4.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex (proposal reviewed by Jay first)
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-050"></a>
### GYST-VN-050 — Courses → Learn (IA and presentation)

- **ID:** GYST-VN-050
- **NAME:** Courses → Learn (IA and presentation)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UX
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Present Courses as Learn: route `/learn`, page title and Continue-first structure; Courses stays a content type inside.
- **CURRENT PROBLEM:** D1 renamed the area; routes and titles still say Courses.
- **DETAILED REQUIREMENTS:**
  - `/learn` is canonical; `/courses*` redirects (301) or keeps working as aliases
  - No data model or table change
  - No Master's/program features (D1 constraint)
- **PROPOSED BEHAVIOR:** The Learn tab lands on a Learn page.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/courses.go`, `web/templates/courses_list.html`, `web/templates/course_detail.html`, `web/static/css/ui-courses.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D1; `12` §Learn; prototype `prototypes/learn.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-025 (Navigation shell: Today · Projects · Health · Learn · More), GYST-VN-027 (Terminology migration in the UI: Project / Task / Roadmap)
- **ACCEPTANCE CRITERIA:**
  - Old course URLs and calendar feed URLs still work
  - Calendar feed tokens untouched
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Programs/degrees.
- **KNOWN RISKS:** Calendar feed paths are auth-exempt; don't move them.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-051"></a>
### GYST-VN-051 — Learn: zero, one and many courses with Continue

- **ID:** GYST-VN-051
- **NAME:** Learn: zero, one and many courses with Continue
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Implement the Continue block and the empty/one/many states, choosing the most urgent due or catch-up session first.
- **CURRENT PROBLEM:** The course list doesn't answer "what should I study next?".
- **DETAILED REQUIREMENTS:**
  - Continue selection uses the app clock
  - One course: Continue + progress + next deadline only
  - Paused/completed collapsed
- **PROPOSED BEHAVIOR:** Learn answers what to study next.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/course.go`, `web/templates/courses_list.html`, `internal/handlers/courses.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Learn
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-050 (Courses → Learn (IA and presentation)), GYST-VN-003 (One app timezone and local-day correctness)
- **ACCEPTANCE CRITERIA:**
  - Tests for Continue selection
  - Empty/one/many verified
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-052"></a>
### GYST-VN-052 — Hide unbuilt Health placeholders

- **ID:** GYST-VN-052
- **NAME:** Hide unbuilt Health placeholders
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Remove primary controls for functionality that doesn't work: Week/Month/Year, Plan meal, Start workout, incomplete trends, weight entry without a writer.
- **CURRENT PROBLEM:** D8: placeholders make Health look broken and promise features that don't exist.
- **DETAILED REQUIREMENTS:**
  - Templates stop rendering those controls
  - **Keep backend work** (`CreateHealthWeightEntry`, Hevy provider, training cache, weight queries) per D8
  - No data or API change
- **PROPOSED BEHAVIOR:** Health shows only what works.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/health.html`, `web/templates/health_components.html`, `web/templates/health_progress.html`, `web/templates/training.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D8; `12` §Health Summary (Never shown)
- **SECURITY REFERENCES:** Health invariants (`AGENTS.md`): missing ≠ zero; Watch and phone steps separate; burn = resting + active; no step→calorie conversion.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - None of the listed controls render
  - Backend functions and tests remain
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Building the missing features.
- **KNOWN RISKS:** Low; good early visible win.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-053"></a>
### GYST-VN-053 — Tie the Hevy card to real connection state

- **ID:** GYST-VN-053
- **NAME:** Tie the Hevy card to real connection state
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Bug
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Replace the static Hevy text with the provider's real connected / not connected / unreachable state.
- **CURRENT PROBLEM:** `02B` D8 / C34: the Hevy card shows fixed text that contradicts the actual provider state.
- **DETAILED REQUIREMENTS:**
  - Read state from the training provider/cache
  - Three states with honest wording and last-sync age
- **PROPOSED BEHAVIOR:** The card tells the truth.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/training.go`, `internal/handlers/training.go`, `web/templates/health.html`, `web/templates/training.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `02B` D8; `12` §Training
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Tests for the three states
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...`, `go test -race ./...`, `gofmt`, `go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Hevy write features.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-054"></a>
### GYST-VN-054 — Health Summary simplification

- **ID:** GYST-VN-054
- **NAME:** Health Summary simplification
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Rebuild the Health Summary as the 4-metric grid (Intake, Burned, Steps, Exercise), the Energy block and rows for Training, Weight and Plan.
- **CURRENT PROBLEM:** Health Summary is long and mixes placeholders with real data (Stage 3).
- **DETAILED REQUIREMENTS:**
  - Intake vs target; Burned = resting + active; Steps: Watch and phone shown separately; Exercise
  - Energy: Food left and Balance so far are separate, never summed
  - Missing values show "—", never 0
  - Info sheet for formulas and provenance
- **PROPOSED BEHAVIOR:** Today's body at a glance, from real data only.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/health.html`, `internal/handlers/health_view.go`, `web/static/css/ui-health.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Health Summary; prototype `prototypes/health.html`
- **SECURITY REFERENCES:** Health invariants (`AGENTS.md`): missing ≠ zero; Watch and phone steps separate; burn = resting + active; no step→calorie conversion.
- **DEPENDENCIES:** GYST-VN-052 (Hide unbuilt Health placeholders), GYST-VN-022 (State components: chips, banners, empty, stale, error, offline), GYST-VN-020 (Grouped list and row components)
- **ACCEPTANCE CRITERIA:**
  - Existing absent-vs-zero tests still pass unchanged
  - Seeded day with missing steps shows "—"
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** New metrics.
- **KNOWN RISKS:** Violating a Health invariant; reviewer checks each against `AGENTS.md`.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-055"></a>
### GYST-VN-055 — Health navigation normalization

- **ID:** GYST-VN-055
- **NAME:** Health navigation normalization
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UX
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** One Health sub-navigation: Summary · Nutrition · Training · Progress · Plan, with consistent back behaviour.
- **CURRENT PROBLEM:** Health sub-pages are reached inconsistently (Stage 3).
- **DETAILED REQUIREMENTS:**
  - Segmented control or row links per `12`
  - Deep-linkable
  - No sub-page without real content (Progress hidden if no pipeline has data)
- **PROPOSED BEHAVIOR:** Health areas are predictable.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/health*.html`, `web/templates/nutrition.html`, `web/templates/training.html`, `internal/handlers/health_*.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Health
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-025 (Navigation shell: Today · Projects · Health · Learn · More), GYST-VN-054 (Health Summary simplification)
- **ACCEPTANCE CRITERIA:**
  - Every Health page reachable in ≤ 2 taps from Health
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-056"></a>
### GYST-VN-056 — Nutrition vNext

- **ID:** GYST-VN-056
- **NAME:** Nutrition vNext
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Rebuild Nutrition as the calories ring plus Protein/Carbs/Fat/Fiber rows, with provenance and stale chip.
- **CURRENT PROBLEM:** Nutrition doesn't follow the vNext pattern.
- **DETAILED REQUIREMENTS:**
  - Missing macro → "— not received" with a dashed bar
  - Target not configured → value only
  - No food logging UI
- **PROPOSED BEHAVIOR:** A clear daily nutrition view.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/nutrition.html`, `internal/handlers/nutrition.go`, `web/static/css/ui-health.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Nutrition; prototype `prototypes/nutrition.html`
- **SECURITY REFERENCES:** Health invariants (`AGENTS.md`): missing ≠ zero; Watch and phone steps separate; burn = resting + active; no step→calorie conversion.
- **DEPENDENCIES:** GYST-VN-055 (Health navigation normalization)
- **ACCEPTANCE CRITERIA:**
  - Missing fiber shows the not-received state, not 0
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Meal planning.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-057"></a>
### GYST-VN-057 — Training: connected and disconnected states

- **ID:** GYST-VN-057
- **NAME:** Training: connected and disconnected states
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Training shows last workout, week count, next routine and muscle focus when Hevy is connected, and an honest not-connected pattern when it isn't.
- **CURRENT PROBLEM:** Training UI doesn't distinguish provider states clearly.
- **DETAILED REQUIREMENTS:**
  - Uses the provider state from the Hevy task
  - "Sync now" rate-limited; no Start workout
  - Manual gym check-ins behind a disclosure
- **PROPOSED BEHAVIOR:** Training is useful when connected and honest when not.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/training.html`, `internal/handlers/training.go`, `web/static/css/ui-training.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Training
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-053 (Tie the Hevy card to real connection state), GYST-VN-055 (Health navigation normalization)
- **ACCEPTANCE CRITERIA:**
  - Connected, disconnected and unreachable states verified
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Workout logging.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-058"></a>
### GYST-VN-058 — Progress: real-data trends only

- **ID:** GYST-VN-058
- **NAME:** Progress: real-data trends only
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Progress shows the weight history chart (30/90/all) when readings exist, and nothing speculative otherwise.
- **CURRENT PROBLEM:** Progress shows placeholder trend cards (D8).
- **DETAILED REQUIREMENTS:**
  - Empty: "No weight readings yet" (no disabled entry button)
  - Future trend cards appear only when their data pipeline exists
- **PROPOSED BEHAVIOR:** Progress never shows fake charts.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/health_progress.html`, `internal/handlers/health_progress.go`, `web/static/js/charts.js`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Progress
- **SECURITY REFERENCES:** Health invariants (`AGENTS.md`): missing ≠ zero; Watch and phone steps separate; burn = resting + active; no step→calorie conversion.
- **DEPENDENCIES:** GYST-VN-055 (Health navigation normalization)
- **ACCEPTANCE CRITERIA:**
  - No card renders without data
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** New pipelines.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-059"></a>
### GYST-VN-059 — Decide the training storage shape

- **ID:** GYST-VN-059
- **NAME:** Decide the training storage shape
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Research
- **PRIORITY:** P3
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Decide JSON-blob cache vs normalized tables for training history before any training-history feature.
- **CURRENT PROBLEM:** `02B` D7: storage shape blocks strength trends and history queries.
- **DETAILED REQUIREMENTS:**
  - Options, query examples, migration impact
  - Jay decides; ADR recorded
- **PROPOSED BEHAVIOR:** A recorded decision; no code.
- **RELEVANT FILES / LIKELY AREAS:** `docs/adr/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - ADR merged
- **TEST PLAN:** —
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** —
- **OUT OF SCOPE:** Implementing trends.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Agent drafts the runbook, script or analysis; Jay executes on the host or decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-060"></a>
### GYST-VN-060 — Goals at More → Goals, with edit

- **ID:** GYST-VN-060
- **NAME:** Goals at More → Goals, with edit
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Make Goals its own destination under More and add the missing edit action.
- **CURRENT PROBLEM:** D7 moved Goals to More → Goals; `12` notes goal edit is missing.
- **DETAILED REQUIREMENTS:**
  - Goals list: icon · name · period · progress · pace
  - Edit goal (form, CSRF-protected POST)
  - Completed goals collapsed
- **PROPOSED BEHAVIOR:** Goals are easy to reach and correct.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/goals_list.html`, `web/templates/goal_detail.html`, `web/templates/goal_form.html`, `internal/handlers/handlers.go`, `internal/repository/goals.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D7; `12` §Goals
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-026 (Simplify More to the grouped model)
- **ACCEPTANCE CRITERIA:**
  - Edit round-trips all fields
  - UI checks pass
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...`, `go test -race ./...`, `gofmt`, `go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Weekly review.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-061"></a>
### GYST-VN-061 — Secondary screens adopt the vNext shell

- **ID:** GYST-VN-061
- **NAME:** Secondary screens adopt the vNext shell
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P3
- **PHASE:** 3 — Projects / Galaxy / Learn / Health migrations · Milestone M2
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Bring Tasks, Reminders, Calendar, Analytics, Archive and Settings into the vNext tokens, lists and states without changing behaviour.
- **CURRENT PROBLEM:** These screens still use legacy `styles.css` patterns.
- **DETAILED REQUIREMENTS:**
  - One PR per screen
  - No behaviour change
  - Remove per-screen legacy CSS only when unused (grep proof, including `galaxy.html`/`galaxy.js` class usage)
- **PROPOSED BEHAVIOR:** The whole app feels consistent.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/tasks_list.html`, `web/templates/reminders.html`, `web/templates/monthly.html`, `web/templates/analytics.html`, `web/templates/archive.html`, `web/templates/settings.html`, `web/static/css/styles.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `17` screen-by-screen table; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-025 (Navigation shell: Today · Projects · Health · Learn · More), GYST-VN-020 (Grouped list and row components), GYST-VN-022 (State components: chips, banners, empty, stale, error, offline)
- **ACCEPTANCE CRITERIA:**
  - Each screen passes the UI checks
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert per screen.
- **OUT OF SCOPE:** New features.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** L

## Phase 4 — Apps + Launcher (M3)

<a id="gyst-vn-062"></a>
### GYST-VN-062 — Apps registry and Apps page

- **ID:** GYST-VN-062
- **NAME:** Apps registry and Apps page
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P1
- **PHASE:** 4 — Apps + Launcher · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** A non-secret, config-backed registry of external tools (name, icon/monogram, URL, group, tailnet-only flag, optional health source) and the More → Apps page.
- **CURRENT PROBLEM:** External tools (SamVision, n8n, Portainer, Kuma and others) are reached by bookmarks; `06` §G defines one registry.
- **DETAILED REQUIREMENTS:**
  - Registry in a config file in the repo (no DB table needed yet; no secrets)
  - Tailnet-only apps show a marker and aren't linked on the public host
  - Groups Work · Operate · Learning
  - Links open in a new tab with `rel=noopener`
  - Health dot only when a health source exists (later phases)
- **PROPOSED BEHAVIOR:** One place to open every tool, on phone and desktop.
- **RELEVANT FILES / LIKELY AREAS:** New `internal/services/apps.go`, `config/apps.json` (or similar), `web/templates/apps.html` (new), `internal/handlers/ui_shell.go`, `web/templates/more.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Apps; prototype `prototypes/apps.html`
- **SECURITY REFERENCES:** `06` §G (no secrets in registry); `09` §G.2 (links public, tailnet apps marked).
- **DEPENDENCIES:** GYST-VN-026 (Simplify More to the grouped model), GYST-VN-020 (Grouped list and row components)
- **ACCEPTANCE CRITERIA:**
  - Tailnet-only URLs are not present in the HTML served on the public host
  - Apps row appears in More only now that the page exists
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert; config file removal.
- **OUT OF SCOPE:** Registry editing UI; health checks.
- **KNOWN RISKS:** Leaking internal hostnames on the public host; tested.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-063"></a>
### GYST-VN-063 — Launcher v2: shell and Go mode

- **ID:** GYST-VN-063
- **NAME:** Launcher v2: shell and Go mode
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P1
- **PHASE:** 4 — Apps + Launcher · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Build the launcher (⌘/Ctrl-K panel on desktop, full-height sheet on phone) with Go mode searching GYST destinations, projects, courses, Health pages and Apps.
- **CURRENT PROBLEM:** The current launcher mixes navigation and actions; vNext separates Go, Do and Capture.
- **DETAILED REQUIREMENTS:**
  - Server-side search endpoint returning grouped results (read-only GET)
  - Keyboard: arrows, Enter, Escape; recents + suggestions when empty
  - Mode prefixes: none = Go, `>` = Do, `+` = Capture (Do and Capture arrive in their own tasks)
  - Results are links (no `location.href` hacks)
  - Galaxy is a Go result opening the real `/galaxy` (replaces the old command-menu "Open Galaxy" entry, `17A` §C.7)
- **PROPOSED BEHAVIOR:** Jay can reach anything in two keystrokes or one tap.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/app.js` (launcher module), `internal/handlers/handlers.go` (search route), new `internal/services/search.go`, `web/templates/layout.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Launcher; `13` §Launcher; prototype `prototypes/launcher.html`; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-023 (Sheet, popover, inspector and confirm dialog components), GYST-VN-062 (Apps registry and Apps page), GYST-VN-025 (Navigation shell: Today · Projects · Health · Learn · More)
- **ACCEPTANCE CRITERIA:**
  - Search results respect the public/tailnet host (tailnet apps hidden publicly)
  - Works with VoiceOver on iPhone
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** External search; Do and Capture.
- **KNOWN RISKS:** Shell and `app.js` contention.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-064"></a>
### GYST-VN-064 — Launcher Do mode

- **ID:** GYST-VN-064
- **NAME:** Launcher Do mode
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 4 — Apps + Launcher · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add Do mode: a fixed list of allowed commands (complete Now item, start learning session, open prayer times, toggle appearance…) that map to existing endpoints.
- **CURRENT PROBLEM:** Commands are scattered; `05` defines Do as an allow-listed command set with no implicit capture.
- **DETAILED REQUIREMENTS:**
  - Command registry in code; each command is either navigation or an existing CSRF-protected POST
  - Destructive or infrastructure commands are **not** in the launcher
  - Confirmation for any state change
- **PROPOSED BEHAVIOR:** Frequent actions are one command away.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/app.js`, `internal/services/search.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `13` §Launcher (Do)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-063 (Launcher v2: shell and Go mode)
- **ACCEPTANCE CRITERIA:**
  - Every command maps to an existing, tested endpoint
  - No command triggers a GET write
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Agent/system commands (later phases add their own entries with their own gates).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-065"></a>
### GYST-VN-065 — Launcher Capture mode

- **ID:** GYST-VN-065
- **NAME:** Launcher Capture mode
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 4 — Apps + Launcher · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add Capture mode: an explicit quick-capture form (task or reminder, optional date) using the app clock for defaults.
- **CURRENT PROBLEM:** Quick capture exists but can create items implicitly; `05` requires explicit capture only.
- **DETAILED REQUIREMENTS:**
  - `+` prefix or Capture tab shows a form; nothing is created without Enter on the form
  - Date defaults use the app timezone
  - Uses the existing create endpoints
- **PROPOSED BEHAVIOR:** Capturing is fast and never accidental.
- **RELEVANT FILES / LIKELY AREAS:** `web/static/js/app.js`, `internal/handlers/handlers.go` (existing capture endpoint), `internal/services/lifeops.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `13` §Launcher (Capture)
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-063 (Launcher v2: shell and Go mode), GYST-VN-003 (One app timezone and local-day correctness)
- **ACCEPTANCE CRITERIA:**
  - Typing in Go mode never creates an item
  - Captured item has today's local date at 23:30 local
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Natural-language parsing.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

## Phase 5 — Attention (M3)

<a id="gyst-vn-066"></a>
### GYST-VN-066 — Attention service with GYST sources

- **ID:** GYST-VN-066
- **NAME:** Attention service with GYST sources
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P1
- **PHASE:** 5 — Attention · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** An Attention service that computes items on read from GYST data: overdue tasks/reminders, learning session due, Health sync stale, GYST backup overdue.
- **CURRENT PROBLEM:** D5: one Attention model; today every screen invents its own warnings.
- **DETAILED REQUIREMENTS:**
  - Service returns items with source, source_key, kind (decision/problem), severity (info/needs_action/warning/critical), title, target
  - Producer interface so later sources (GitHub, agents, n8n, System) register without changing the service
  - Items point at their source; no editable work stored (not a task store)
  - Backup overdue reads `last_run.json` (36 h warning, 72 h critical)
  - Ordering: severity, then first seen
- **PROPOSED BEHAVIOR:** One answer to "what needs me?" from GYST's own data.
- **RELEVANT FILES / LIKELY AREAS:** New `internal/services/attention.go`, `internal/services/lifeops.go` (overdue query reuse), `internal/services/health.go`.
- **DESIGN REFERENCES:** `05A` D5; `06` §H.1–H.2
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-004 (UTC instants for new audit and event records), GYST-VN-003 (One app timezone and local-day correctness), GYST-VN-032 (Today: overdue reminders and tasks), GYST-VN-010 (Scheduled GYST backup job)
- **ACCEPTANCE CRITERIA:**
  - Table tests for each source and severity
  - Adding a fake producer in a test needs no service change
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Stored conditions, snooze/dismiss (next task).
- **KNOWN RISKS:** Becoming a second task store; review against D5.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-067"></a>
### GYST-VN-067 — Attention stored conditions, snooze and dismiss

- **ID:** GYST-VN-067
- **NAME:** Attention stored conditions, snooze and dismiss
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 5 — Attention · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add the `attention_conditions` table (for external sources) and owner snooze/dismiss that returns when the condition's fingerprint changes.
- **CURRENT PROBLEM:** Computed items can't be snoozed or dismissed without storage; external sources need upsert/resolve.
- **DETAILED REQUIREMENTS:**
  - Additive migration per `06` §H.1 with UTC instants
  - Upsert/resolve API for producers; auto-resolve when the source clears
  - Snooze 1 h / tomorrow; dismiss hides until fingerprint changes
  - Per-namespace cap (20 open) for machine-fed sources
- **PROPOSED BEHAVIOR:** Attention stays accurate and quiet.
- **RELEVANT FILES / LIKELY AREAS:** `migrations/`, new `internal/repository/attention.go`, `internal/services/attention.go`.
- **DESIGN REFERENCES:** `06` §H
- **SECURITY REFERENCES:** `09` §L (namespace caps).
- **DEPENDENCIES:** GYST-VN-066 (Attention service with GYST sources)
- **ACCEPTANCE CRITERIA:**
  - Tests: dismissed item reappears when fingerprint changes; resolved items leave the list
  - FK check clean
- **TEST PLAN:** `go test ./...`; migration on a backup copy.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert code; the table can stay.
- **OUT OF SCOPE:** Push delivery.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-068"></a>
### GYST-VN-068 — Attention page and badge

- **ID:** GYST-VN-068
- **NAME:** Attention page and badge
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 5 — Attention · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** The full Attention page (Critical · Needs you · Warnings) and a badge for needs_action + critical, reachable from Today, the badge, the launcher and deep links; no permanent menu item.
- **CURRENT PROBLEM:** C1: Attention must be a full destination without adding a permanent menu item.
- **DETAILED REQUIREMENTS:**
  - Row: severity tile · title · source · age · one action or ⋯
  - Filter All / Decisions / Problems
  - Empty: "Nothing needs you"
  - "Some sources unavailable" banner listing sources
  - Launcher Go entry "Attention"; More row per `12`
- **PROPOSED BEHAVIOR:** Jay can see everything that needs him in one place.
- **RELEVANT FILES / LIKELY AREAS:** New `web/templates/attention.html`, `internal/handlers/ui_shell.go` (badge), `web/templates/ui_shell.html`, `internal/services/search.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D5, C1; `12` §Attention; prototype `prototypes/attention.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-066 (Attention service with GYST sources), GYST-VN-067 (Attention stored conditions, snooze and dismiss), GYST-VN-022 (State components: chips, banners, empty, stale, error, offline), GYST-VN-025 (Navigation shell: Today · Projects · Health · Learn · More), GYST-VN-063 (Launcher v2: shell and Go mode)
- **ACCEPTANCE CRITERIA:**
  - Badge counts only needs_action + critical
  - No check circles or editing on the page (not a task manager)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Push.
- **KNOWN RISKS:** Shell contention (badge).
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-069"></a>
### GYST-VN-069 — Today: Attention section

- **ID:** GYST-VN-069
- **NAME:** Today: Attention section
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P1
- **PHASE:** 5 — Attention · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Show up to 3 Attention rows plus "All n" on Today, absent when empty; overdue items move from their Phase 2 rows into Attention.
- **CURRENT PROBLEM:** Today needs the Attention section in its specified position (after Rhythm).
- **DETAILED REQUIREMENTS:**
  - Uses the Attention service; overdue rows now come from Attention (no duplicate rendering)
  - Section absent when there are no items
- **PROPOSED BEHAVIOR:** Today shows what needs Jay, in one place, without duplication.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/today.html`, `internal/handlers/today_view.go`, `web/static/css/ui-today.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Today scenarios
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-068 (Attention page and badge), GYST-VN-036 (Today: responsive layout (phone order, desktop two columns))
- **ACCEPTANCE CRITERIA:**
  - Overdue item appears once (Attention), not twice
  - Section absent on a normal day
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Today file contention.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-070"></a>
### GYST-VN-070 — Critical Attention push

- **ID:** GYST-VN-070
- **NAME:** Critical Attention push
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P3
- **PHASE:** 5 — Attention · Milestone M3
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Send a Web Push for new critical items (and needs_action where a source opts in), with a deep link to the item.
- **CURRENT PROBLEM:** Push exists (`internal/services/push.go`) but isn't connected to Attention.
- **DETAILED REQUIREMENTS:**
  - Only on transition to critical or new critical
  - Rate-limited; no push for info/warning
  - Deep link opens the Attention item
- **PROPOSED BEHAVIOR:** Urgent problems reach Jay's phone.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/push.go`, `internal/services/attention.go`.
- **DESIGN REFERENCES:** `06` §H.2
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-067 (Attention stored conditions, snooze and dismiss)
- **ACCEPTANCE CRITERIA:**
  - Test: no duplicate push for an unchanged fingerprint
- **TEST PLAN:** `go test ./...`; manual push to Jay's device.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Digest notifications.
- **KNOWN RISKS:** Notification noise; conservative defaults.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

## Phase 6 — Machine auth / integration foundation (M4)

<a id="gyst-vn-071"></a>
### GYST-VN-071 — Decide packaging and repository for Bridge, Server Provider and Action Runner

- **ID:** GYST-VN-071
- **NAME:** Decide packaging and repository for Bridge, Server Provider and Action Runner
- **PROJECT:** Governance
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Research
- **PRIORITY:** P1
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Decide where the Agent Bridge, Server Provider and Action Runner code lives and how it's built and deployed.
- **CURRENT PROBLEM:** No repository is decided. If the bridge lived in `gyst-lifeops`, the allowlisted agents could modify their own controller.
- **DETAILED REQUIREMENTS:**
  - Option A: `cmd/` binaries inside `gyst-lifeops` (path-policy PRIVILEGED)
  - Option B: a separate private `gyst-ops` repo, **not** on the agent allowlist
  - Recommendation: B, so agents can never change their own bridge, policy or runner
  - Build/deploy approach for each (systemd service vs container)
  - ADR recorded
- **PROPOSED BEHAVIOR:** Later infrastructure tasks have a defined repository.
- **RELEVANT FILES / LIKELY AREAS:** `docs/adr/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - ADR merged with the chosen option
- **TEST PLAN:** —
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** —
- **OUT OF SCOPE:** Implementation.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** An agent drafts the ADR options; Jay decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-072"></a>
### GYST-VN-072 — Consolidate configuration into one struct

- **ID:** GYST-VN-072
- **NAME:** Consolidate configuration into one struct
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Refactor
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Load all configuration (including VAPID and provider tokens) into one validated struct at startup; honour or remove `DB_PATH`/`PORT`.
- **CURRENT PROBLEM:** `02B` D6: configuration is read ad hoc; some env vars are ignored.
- **DETAILED REQUIREMENTS:**
  - One `config.Load()` with validation and clear startup errors
  - Secrets never logged (only CONFIGURED/NOT CONFIGURED)
  - Secret-file support (`*_FILE`) for new credentials
- **PROPOSED BEHAVIOR:** Startup shows what is configured without printing secrets.
- **RELEVANT FILES / LIKELY AREAS:** `internal/app/app.go`, new `internal/config/`, `.env.example`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §D; S2-21 (dead config).
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Test: a missing required value fails startup with a named error
  - Log shows presence only
- **TEST PLAN:** `go test ./...`, `go test -race ./...`, `gofmt`, `go vet`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Moving secrets to a vault.
- **KNOWN RISKS:** Contention with `app.go`.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-073"></a>
### GYST-VN-073 — Dual listener: public, privileged (tailnet) and internal machine

- **ID:** GYST-VN-073
- **NAME:** Dual listener: public, privileged (tailnet) and internal machine
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Run GYST with a public listener (public routes only), a privileged listener (public + privileged routes) and an internal machine listener, and point Caddy's sites at the right one.
- **CURRENT PROBLEM:** `09` §G.1: one listener serves both hostnames, so tailnet-only features could leak through Cloudflare by a Caddy mistake.
- **DETAILED REQUIREMENTS:**
  - Route sets registered per listener; privileged handlers simply absent on public
  - Machine API (`/api/machine/*`) and `/healthz` only on the internal listener
  - Caddy: public site → :8080 only; tailnet site → :8090 only; explicit deny of `/api/machine/*` and `/system/*/detail|logs|actions` on public
  - No host port published for any listener
  - Tests assert privileged routes 404 on the public listener
- **PROPOSED BEHAVIOR:** Tailnet-only features cannot be reached from the internet even if Caddy is misconfigured.
- **RELEVANT FILES / LIKELY AREAS:** `internal/app/app.go`, `internal/handlers/handlers.go` (route registration), `Caddyfile`, `docker-compose.yml`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §G.1; gate G8; `10A` D-S4.
- **DEPENDENCIES:** GYST-VN-072 (Consolidate configuration into one struct), GYST-VN-014 (/healthz endpoint), GYST-VN-016 (Log rotation and Docker healthcheck in Compose)
- **ACCEPTANCE CRITERIA:**
  - Negative test per privileged route on the public listener
  - Existing public routes unchanged
  - Jay's deploy verified from phone (public) and tailnet
- **TEST PLAN:** `go test ./...`; local compose run; post-deploy curl checks from both networks.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert code and Caddy; redeploy.
- **OUT OF SCOPE:** Tailscale identity headers.
- **KNOWN RISKS:** Misrouting breaks login; staged deploy with rollback plan.
- **SUGGESTED IMPLEMENTER:** Agent implements; Jay deploys
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** L

<a id="gyst-vn-074"></a>
### GYST-VN-074 — Bind the tailnet site to the Tailscale address only

- **ID:** GYST-VN-074
- **NAME:** Bind the tailnet site to the Tailscale address only
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Infrastructure
- **PRIORITY:** P1
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Change the host port 3443 binding from all interfaces to the Tailscale IP.
- **CURRENT PROBLEM:** S2-08: a LAN client sending the tailnet SNI reaches the tailnet site.
- **DETAILED REQUIREMENTS:**
  - Compose port binding `100.x.y.z:3443:3443` (value from host config, not committed if considered sensitive)
  - Verify from a LAN device (refused) and a tailnet device (works)
- **PROPOSED BEHAVIOR:** The tailnet site is reachable only over Tailscale.
- **RELEVANT FILES / LIKELY AREAS:** `docker-compose.yml`, host `.env`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §G.1 layer 3; S2-08.
- **DEPENDENCIES:** GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine)
- **ACCEPTANCE CRITERIA:**
  - LAN access refused; tailnet works; public unaffected
- **TEST PLAN:** Manual network checks recorded.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert the binding.
- **OUT OF SCOPE:** Firewall rules.
- **KNOWN RISKS:** Tailscale IP change breaks the bind; documented.
- **SUGGESTED IMPLEMENTER:** Agent prepares the Compose change; Jay applies and verifies
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-075"></a>
### GYST-VN-075 — Security headers including frame-ancestors

- **ID:** GYST-VN-075
- **NAME:** Security headers including frame-ancestors
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Send CSP (with `frame-ancestors 'none'`), `X-Content-Type-Options`, `Referrer-Policy` and `Permissions-Policy` on all responses.
- **CURRENT PROBLEM:** S2-07: no browser security headers.
- **DETAILED REQUIREMENTS:**
  - Middleware sets headers; CSP allows only self plus the specific needs (Three.js vendor module is local and loaded by dynamic `import()`; Galaxy uses WebGL, canvas-generated textures and `/static/images`)
  - Report-only first for one deploy, then enforce
  - PWA and service worker still work
- **PROPOSED BEHAVIOR:** GYST can't be framed, and injected content is constrained.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/middleware.go`.
- **DESIGN REFERENCES:** None (no UI change).; `17A` §N.5 (shared-change awareness)
- **SECURITY REFERENCES:** `09` gate G15; S2-07.
- **DEPENDENCIES:** GYST-VN-015 (Structured application logging and sanitized client errors)
- **ACCEPTANCE CRITERIA:**
  - Tests assert headers
  - No CSP violation in the console on every migrated screen
  - Galaxy quick regression check (`17A` §L, §N.5): `/galaxy` loads the real Three.js scene, a world can be selected and opened, the ship is visible, and `node scripts/check-galaxy.cjs` passes
- **TEST PLAN:** `go test ./...`; browser console on each main screen.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390 and 1280 on every top-level screen, including `/galaxy` with the 3D scene running: console free of CSP violations; installed PWA on iPhone still launches.
- **ROLLBACK CONSIDERATION:** Revert or return to report-only.
- **OUT OF SCOPE:** Subresource integrity.
- **KNOWN RISKS:** Inline scripts in templates break under CSP; inventory first.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-076"></a>
### GYST-VN-076 — Run the GYST container as a non-root user

- **ID:** GYST-VN-076
- **NAME:** Run the GYST container as a non-root user
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Infrastructure
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Run GYST under a fixed non-root UID with the data directory owned by it.
- **CURRENT PROBLEM:** S2-10: containers run as root.
- **DETAILED REQUIREMENTS:**
  - `USER` in the Dockerfile with a fixed UID
  - Data directory ownership change is a Jay-run step with a backup first
  - Read-only root filesystem noted as later
- **PROPOSED BEHAVIOR:** A GYST compromise doesn't give root in the container.
- **RELEVANT FILES / LIKELY AREAS:** `Dockerfile`, `docker-compose.yml`, `docs/runbooks/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` gate G10; S2-10.
- **DEPENDENCIES:** GYST-VN-009 (Harden host file permissions and record verification), GYST-VN-016 (Log rotation and Docker healthcheck in Compose), GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine)
- **ACCEPTANCE CRITERIA:**
  - Container runs as the UID; backups and writes work
- **TEST PLAN:** Local compose run; post-deploy backup run.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert image; restore ownership.
- **OUT OF SCOPE:** Other containers.
- **KNOWN RISKS:** Permission mismatch on `data/`; backup first.
- **SUGGESTED IMPLEMENTER:** Agent edits Dockerfile/Compose; Jay deploys and changes ownership
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-077"></a>
### GYST-VN-077 — Session lifetime: 30-day rolling, 90-day absolute

- **ID:** GYST-VN-077
- **NAME:** Session lifetime: 30-day rolling, 90-day absolute
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Keep the 30-day rolling session and add a 90-day absolute cap from session creation.
- **CURRENT PROBLEM:** `10A` D-S5: sessions can currently be refreshed indefinitely.
- **DETAILED REQUIREMENTS:**
  - Store `created_at` (additive column; existing rows get their current value treated as created now)
  - Refresh never extends beyond created + 90 days
  - On cap: redirect to login with a neutral message
- **PROPOSED BEHAVIOR:** Even an active stolen cookie expires within 90 days.
- **RELEVANT FILES / LIKELY AREAS:** `migrations/`, `internal/repository/sessions.go`, `internal/services/auth.go`, `internal/handlers/middleware.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `10A` D-S5; `09` §C.
- **DEPENDENCIES:** GYST-VN-008 (Scheduled expired-session cleanup)
- **ACCEPTANCE CRITERIA:**
  - Tests with the pinned clock at 29, 31, 89 and 91 days
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert; column stays.
- **OUT OF SCOPE:** Device list UI.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-078"></a>
### GYST-VN-078 — Sign out everywhere and password-change revocation

- **ID:** GYST-VN-078
- **NAME:** Sign out everywhere and password-change revocation
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add "Sign out everywhere" and revoke all sessions when the password hash changes.
- **CURRENT PROBLEM:** S2-11: sessions survive password changes; no global sign-out.
- **DETAILED REQUIREMENTS:**
  - Button in Settings → Security (re-auth required once step-up exists)
  - On startup, if the configured hash fingerprint differs from the stored one, revoke all sessions
  - Audit `session.revoked_all`
- **PROPOSED BEHAVIOR:** Jay can kill all sessions; a password change does it automatically.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/auth.go`, `internal/handlers/auth.go`, `web/templates/settings.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** S2-11; `09` §P (before write/control).
- **DEPENDENCIES:** GYST-VN-077 (Session lifetime: 30-day rolling, 90-day absolute), GYST-VN-005 (Minimal audit_events store)
- **ACCEPTANCE CRITERIA:**
  - Tests for both paths
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-079"></a>
### GYST-VN-079 — Step-up re-authentication, action nonce and confirmation pattern

- **ID:** GYST-VN-079
- **NAME:** Step-up re-authentication, action nonce and confirmation pattern
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Require the password again for sensitive actions, valid for about 15 minutes per session, and bind each sensitive POST to a single-use action nonce and a confirmation page.
- **CURRENT PROBLEM:** `09` G7 / `10A` D-S5: no re-auth exists; sensitive writes must not rely on the long session alone.
- **DETAILED REQUIREMENTS:**
  - `reauth_until` per session (UTC instant); re-auth sheet on phone, dialog on desktop
  - Single-use nonce tied to action + target + session, expiring in 10 min
  - Throttling and audit (`reauth.succeeded/failed`) reuse the login throttle
  - A helper to mark a route as requiring step-up (used later by agent:ready, credentials, system actions)
- **PROPOSED BEHAVIOR:** Money-spending, agent-starting and infrastructure actions need a fresh password.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/auth.go`, `internal/handlers/middleware.go`, `internal/handlers/auth.go`, `migrations/`, `web/templates/` (re-auth sheet).
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` gate G7; `10A` D-S5.
- **DEPENDENCIES:** GYST-VN-077 (Session lifetime: 30-day rolling, 90-day absolute), GYST-VN-005 (Minimal audit_events store), GYST-VN-023 (Sheet, popover, inspector and confirm dialog components), GYST-VN-006 (Login throttling)
- **ACCEPTANCE CRITERIA:**
  - Negative tests: expired re-auth, reused nonce, nonce for another target all refused
  - Re-auth sheet passes UI checks
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Passkeys.
- **KNOWN RISKS:** Friction if the window is too short; 15 min per D-S5.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-080"></a>
### GYST-VN-080 — Unify password hashing at cost 12

- **ID:** GYST-VN-080
- **NAME:** Unify password hashing at cost 12
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Make both hash generators use cost 12; Jay regenerates the production hash.
- **CURRENT PROBLEM:** S2-12: the two generators use different costs, and differ from the documented cost.
- **DETAILED REQUIREMENTS:**
  - One shared function used by `lifeops passwordhash` and `cmd/passwordhash`
  - Jay regenerates and sets `GYST_PASSWORD_HASH` (then all sessions revoke via the revoke task)
- **PROPOSED BEHAVIOR:** One documented cost.
- **RELEVANT FILES / LIKELY AREAS:** `internal/app/passwordhash_cmd.go`, `cmd/passwordhash/`, `internal/services/auth.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** S2-12; `09` §C.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Test asserts cost 12 from both entry points
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Argon2.
- **KNOWN RISKS:** Jay must update the env; documented.
- **SUGGESTED IMPLEMENTER:** Agent unifies code; Jay regenerates the production hash
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-081"></a>
### GYST-VN-081 — Machine credentials with scopes

- **ID:** GYST-VN-081
- **NAME:** Machine credentials with scopes
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P1
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add `machine_credentials` (hashed tokens, scopes, namespace, status, expiry) and middleware that authenticates machine requests on the internal listener.
- **CURRENT PROBLEM:** `02B` D5: only a single Health token exists; integrations need scoped, revocable credentials.
- **DETAILED REQUIREMENTS:**
  - Schema per `09` §D.1; SHA-256 of token stored; token shown once at creation (tailnet admin task)
  - Scopes: `health:sync`, `automation:event`, `app:event`, `agent:report`, `agent:commands`, `server:commands`
  - Constant-time compare; per-credential rate limit; `credential.denied` audit
  - Existing Health token keeps working (mapped to a `health:sync` credential or left as is until migrated)
- **PROPOSED BEHAVIOR:** Every machine caller has its own revocable, least-privilege key.
- **RELEVANT FILES / LIKELY AREAS:** `migrations/`, new `internal/repository/credentials.go`, new `internal/services/credentials.go`, `internal/handlers/middleware.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §D, §E; gate G3.
- **DEPENDENCIES:** GYST-VN-005 (Minimal audit_events store), GYST-VN-004 (UTC instants for new audit and event records), GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine), GYST-VN-007 (Failed-login audit events), GYST-VN-009 (Harden host file permissions and record verification), GYST-VN-012 (Off-host backup copy), GYST-VN-013 (Restore runbook and one tested restore)
- **ACCEPTANCE CRITERIA:**
  - Negative tests: wrong scope, revoked, expired, wrong listener all refused
  - Health sync unaffected
- **TEST PLAN:** `go test ./...`, `-race`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert; table stays.
- **OUT OF SCOPE:** Credential admin UI (next task).
- **KNOWN RISKS:** Root of machine trust; ELEVATED review.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-082"></a>
### GYST-VN-082 — Machine credential admin (tailnet, step-up)

- **ID:** GYST-VN-082
- **NAME:** Machine credential admin (tailnet, step-up)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Create, rotate and revoke machine credentials from Settings → Security on the privileged listener only, with step-up.
- **CURRENT PROBLEM:** Credentials need a safe lifecycle UI.
- **DETAILED REQUIREMENTS:**
  - Route only on the privileged listener
  - Token displayed once; never logged
  - Audit `credential.created/rotated/revoked`
- **PROPOSED BEHAVIOR:** Jay manages integration keys safely.
- **RELEVANT FILES / LIKELY AREAS:** New `web/templates/settings_security.html`, `internal/handlers/` (privileged routes).
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` §G.2 (TO + step-up).
- **DEPENDENCIES:** GYST-VN-081 (Machine credentials with scopes), GYST-VN-079 (Step-up re-authentication, action nonce and confirmation pattern), GYST-VN-074 (Bind the tailnet site to the Tailscale address only)
- **ACCEPTANCE CRITERIA:**
  - Public listener 404s these routes
  - Token not present in any log line
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-083"></a>
### GYST-VN-083 — Processed-event idempotency store

- **ID:** GYST-VN-083
- **NAME:** Processed-event idempotency store
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add `processed_events` to deduplicate incoming machine events and webhooks, with a ±5 min freshness window.
- **CURRENT PROBLEM:** Event ingestion (GitHub, n8n, bridge) must be replay-safe (`09` §F.2).
- **DETAILED REQUIREMENTS:**
  - Unique (source, event_id); retention 14 days
  - Rejects stale `occurred_at`; audit on rejection
- **PROPOSED BEHAVIOR:** Replayed events do nothing.
- **RELEVANT FILES / LIKELY AREAS:** `migrations/`, new `internal/repository/events.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §F.2.
- **DEPENDENCIES:** GYST-VN-081 (Machine credentials with scopes)
- **ACCEPTANCE CRITERIA:**
  - Tests: duplicate ignored; stale rejected
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-084"></a>
### GYST-VN-084 — Audit viewer and retention pruning

- **ID:** GYST-VN-084
- **NAME:** Audit viewer and retention pruning
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Show audit events on the tailnet (Settings → Security) and prune by retention (security 1 year, lifecycle 180 days) daily.
- **CURRENT PROBLEM:** The audit log has no viewer and grows without bound.
- **DETAILED REQUIREMENTS:**
  - Privileged listener only; filter by action/outcome
  - Daily prune job with a count log line
- **PROPOSED BEHAVIOR:** Jay can review security events.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/` (privileged), `internal/services/audit.go`, `web/templates/`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` §M.
- **DEPENDENCIES:** GYST-VN-005 (Minimal audit_events store), GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine)
- **ACCEPTANCE CRITERIA:**
  - Prune test with the pinned clock
  - Public listener 404
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Export off-host.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-085"></a>
### GYST-VN-085 — Bound the Health sync date range

- **ID:** GYST-VN-085
- **NAME:** Bound the Health sync date range
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Limit Health sync writes to a window around today (for example today and the previous 7 local days).
- **CURRENT PROBLEM:** S2-18: the Health token can overwrite any date.
- **DETAILED REQUIREMENTS:**
  - Reject out-of-window dates with 422 and an audit event
  - Window uses the app clock
- **PROPOSED BEHAVIOR:** A leaked Health token can't rewrite history.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/health_api.go`, `internal/services/health.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** S2-18; Health invariants unchanged.
- **DEPENDENCIES:** GYST-VN-003 (One app timezone and local-day correctness), GYST-VN-005 (Minimal audit_events store)
- **ACCEPTANCE CRITERIA:**
  - Tests at the window edges
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Backfill tool.
- **KNOWN RISKS:** Legitimate late backfills; window documented.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-086"></a>
### GYST-VN-086 — Dependency vulnerability check in CI

- **ID:** GYST-VN-086
- **NAME:** Dependency vulnerability check in CI
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add `govulncheck` to CI and record the current result.
- **CURRENT PROBLEM:** S2-19: dependency vulnerability status is unknown.
- **DETAILED REQUIREMENTS:**
  - CI step using the pinned Go version
  - Failures block; documented exception process
- **PROPOSED BEHAVIOR:** Known vulnerable dependencies are caught.
- **RELEVANT FILES / LIKELY AREAS:** `.github/workflows/check.yml`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** S2-19.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - CI run shows the step
- **TEST PLAN:** CI.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Dependabot.
- **KNOWN RISKS:** CI workflow is an ELEVATED path.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-087"></a>
### GYST-VN-087 — Jira sync: send credentials only to the configured host

- **ID:** GYST-VN-087
- **NAME:** Jira sync: send credentials only to the configured host
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Security
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Ensure the Jira/provider sync sends credentials only to the configured base host, never to a host taken from a stored link.
- **CURRENT PROBLEM:** S2-09: the sync would send credentials to a stored link's host.
- **DETAILED REQUIREMENTS:**
  - Host allowlist from config; mismatched links refused and logged
- **PROPOSED BEHAVIOR:** Credentials never leave to an unexpected host.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/provider_sync.go`, `internal/services/providers/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** S2-09; gate G13.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Negative test with a foreign host
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-088"></a>
### GYST-VN-088 — Deploy recording and running-version stamp

- **ID:** GYST-VN-088
- **NAME:** Deploy recording and running-version stamp
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Infrastructure
- **PRIORITY:** P2
- **PHASE:** 6 — Machine auth / integration foundation · Milestone M4
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Stamp the build with the git SHA, expose it on the privileged listener, and have the deploy procedure write `deploys/gyst-lifeops.json` (repo, SHA, time, result).
- **CURRENT PROBLEM:** S2-14: deployments are not traceable; nobody can tell which commit is running.
- **DETAILED REQUIREMENTS:**
  - `-ldflags` build stamp in the Dockerfile
  - Version shown on the tailnet only (public shows "deployed n days ago" at most, per `09` §K.3)
  - Deploy script (Jay runs) takes a backup first and records the result
- **PROPOSED BEHAVIOR:** Jay always knows which commit is live and when it was deployed.
- **RELEVANT FILES / LIKELY AREAS:** `Dockerfile`, `scripts/deploy.sh` (new), `internal/app/app.go`, `docs/runbooks/deploy.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.10; S2-14.
- **DEPENDENCIES:** GYST-VN-072 (Consolidate configuration into one struct), GYST-VN-010 (Scheduled GYST backup job), GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine)
- **ACCEPTANCE CRITERIA:**
  - Running SHA equals the deployed commit after one deploy
  - The deploy record exists and is `0600`
- **TEST PLAN:** Local build; one real deploy by Jay.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert; deploy manually as before.
- **OUT OF SCOPE:** CI/CD automation; deploying from GYST.
- **KNOWN RISKS:** Script is PRIVILEGED; it must never deploy without the backup step succeeding.
- **SUGGESTED IMPLEMENTER:** Agent writes the stamp and script; Jay runs deploys
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** S

## Phase 7 — Uptime / System read-only (M4)

<a id="gyst-vn-089"></a>
### GYST-VN-089 — Configure Uptime Kuma monitors

- **ID:** GYST-VN-089
- **NAME:** Configure Uptime Kuma monitors
- **PROJECT:** Server / host operations
- **REPOSITORY:** Uptime Kuma (live config)
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 7 — Uptime / System read-only · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Create Kuma monitors for GYST (public and tailnet), n8n and SamVision, plus push monitors for every backup job.
- **CURRENT PROBLEM:** No monitoring alerts exist for GYST (S2-05).
- **DETAILED REQUIREMENTS:**
  - Monitors per `09` §N.8
  - Backup script pushes to its Kuma push URL (URL stored `0600`, not in the repo)
  - Disk warning at 85 %, critical at 95 %
- **PROPOSED BEHAVIOR:** Jay is alerted when GYST or a backup fails.
- **RELEVANT FILES / LIKELY AREAS:** Uptime Kuma configuration (live).
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.8, §N.9.
- **DEPENDENCIES:** GYST-VN-014 (/healthz endpoint), GYST-VN-010 (Scheduled GYST backup job)
- **ACCEPTANCE CRITERIA:**
  - Each monitor green; a deliberately failed push turns red
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Delete monitors.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Alert fatigue.
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-090"></a>
### GYST-VN-090 — Uptime Kuma status provider in GYST

- **ID:** GYST-VN-090
- **NAME:** Uptime Kuma status provider in GYST
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Integration
- **PRIORITY:** P2
- **PHASE:** 7 — Uptime / System read-only · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Read monitor states from Kuma (read-only) and store a snapshot for System and Attention.
- **CURRENT PROBLEM:** GYST has no view of availability.
- **DETAILED REQUIREMENTS:**
  - Read-only API/status-page access; credential via secret file, fixed host (G13)
  - 2 s timeout, cached snapshot, stale after 10 min
  - Down > 5 min → critical Attention; certificate expiring → warning
- **PROPOSED BEHAVIOR:** System and Attention know what's down.
- **RELEVANT FILES / LIKELY AREAS:** New `internal/services/uptime.go`, `internal/repository/`, `internal/services/attention.go` (producer).
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §O (Uptime: G3 if key, G13).
- **DEPENDENCIES:** GYST-VN-089 (Configure Uptime Kuma monitors), GYST-VN-072 (Consolidate configuration into one struct), GYST-VN-067 (Attention stored conditions, snooze and dismiss), GYST-VN-081 (Machine credentials with scopes)
- **ACCEPTANCE CRITERIA:**
  - Tests with a fake Kuma
  - Unreachable Kuma → "Status unavailable", never healthy
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable via config.
- **OUT OF SCOPE:** Kuma configuration from GYST.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-091"></a>
### GYST-VN-091 — Server Provider and filtering Docker proxy (read-only)

- **ID:** GYST-VN-091
- **NAME:** Server Provider and filtering Docker proxy (read-only)
- **PROJECT:** Server / host operations
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist
- **TYPE:** Infrastructure
- **PRIORITY:** P2
- **PHASE:** 7 — Uptime / System read-only · Milestone M4
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Build the read-only Server Provider (host metrics, services, backups, deploys) behind a filtering Docker API proxy, per `09` §K.
- **CURRENT PROBLEM:** GYST can't see disk, memory, containers or backups; mounting the Docker socket into GYST is forbidden.
- **DETAILED REQUIREMENTS:**
  - Fixed GET endpoints; metric and output allowlists
  - Non-root, read-only root filesystem, internal network only
  - Filtering proxy allows only the listed GETs; the provider is its only network member
  - Provider authenticates GYST's token (SHA-256 stored)
  - No actions, exec or arbitrary paths
- **PROPOSED BEHAVIOR:** A safe source of host status.
- **RELEVANT FILES / LIKELY AREAS:** Repository per the packaging decision; Compose service definitions.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §K.1–K.3; gates G3, G4, G10, G13.
- **DEPENDENCIES:** GYST-VN-071 (Decide packaging and repository for Bridge, Server Provider and Action Runner), GYST-VN-072 (Consolidate configuration into one struct), GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine)
- **ACCEPTANCE CRITERIA:**
  - Tests that denied Docker paths are refused
  - Output never contains env, mounts, labels or container IDs
- **TEST PLAN:** Unit tests; local compose run with a test socket.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Stop the services.
- **OUT OF SCOPE:** Actions (Phase 13), logs (Phase 14).
- **KNOWN RISKS:** Anything reaching the proxy can read container env; network isolation tested.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** L

<a id="gyst-vn-092"></a>
### GYST-VN-092 — Deploy the Server Provider

- **ID:** GYST-VN-092
- **NAME:** Deploy the Server Provider
- **PROJECT:** Server / host operations
- **REPOSITORY:** Host
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 7 — Uptime / System read-only · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Deploy the provider and proxy on the host and connect GYST's credential.
- **CURRENT PROBLEM:** Deployment touches Docker and the socket; it is human-only.
- **DETAILED REQUIREMENTS:**
  - Backup first
  - Deploy; verify the network membership; verify denied paths from inside the provider network
  - Record the deploy
- **PROPOSED BEHAVIOR:** The provider runs.
- **RELEVANT FILES / LIKELY AREAS:** Host Compose.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §K.2.
- **DEPENDENCIES:** GYST-VN-091 (Server Provider and filtering Docker proxy (read-only)), GYST-VN-009 (Harden host file permissions and record verification)
- **ACCEPTANCE CRITERIA:**
  - Denied-path checks recorded; GYST reads status
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Stop and remove the services.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Socket exposure mistakes.
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-093"></a>
### GYST-VN-093 — System page: sanitized public view and tailnet detail

- **ID:** GYST-VN-093
- **NAME:** System page: sanitized public view and tailnet detail
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 7 — Uptime / System read-only · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Build the System page from Kuma and provider snapshots, with the public projection enforced server-side and detail only on the privileged listener.
- **CURRENT PROBLEM:** D6/D9 and `09` §K.3: System status is needed, but public exposure must be sanitized.
- **DETAILED REQUIREMENTS:**
  - Public: health state, backup age/state, rounded CPU/memory/disk (5 %), certificate state, rounded uptime
  - Never exposed: environment, mounts, container IDs, network layout, host paths, raw logs, deploy internals
  - Tailnet detail: image tag, restart count, exact metrics, deploy SHA/time
  - "Status unavailable" when stale (> 10 min), never healthy
  - More → Operate → System row and desktop sidebar group appear now (they exist)
- **PROPOSED BEHAVIOR:** Jay sees server health from anywhere, with detail only on the tailnet.
- **RELEVANT FILES / LIKELY AREAS:** New `web/templates/system.html`, `internal/handlers/` (public + privileged routes), `internal/services/system.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §System; prototype `prototypes/system.html`
- **SECURITY REFERENCES:** `09` §K.3; `08A` D-J7.
- **DEPENDENCIES:** GYST-VN-090 (Uptime Kuma status provider in GYST), GYST-VN-092 (Deploy the Server Provider), GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine), GYST-VN-022 (State components: chips, banners, empty, stale, error, offline), GYST-VN-026 (Simplify More to the grouped model)
- **ACCEPTANCE CRITERIA:**
  - Test: the public projection struct has no forbidden field (compile-time type separation)
  - Public HTML contains no path, ID or env
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Actions, logs.
- **KNOWN RISKS:** A new field accidentally added to the public view; mitigated by separate types.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-094"></a>
### GYST-VN-094 — System Attention and quiet healthy indicator

- **ID:** GYST-VN-094
- **NAME:** System Attention and quiet healthy indicator
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P3
- **PHASE:** 7 — Uptime / System read-only · Milestone M4
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Feed System problems into Attention and show one quiet "All systems normal" chip in the Today status area and a sidebar dot.
- **CURRENT PROBLEM:** D9: later, one quiet healthy indicator is acceptable; problems get prominence.
- **DETAILED REQUIREMENTS:**
  - Problems as Attention items (service down, backup stale/failed, disk)
  - Healthy: one muted chip; never a banner
- **PROPOSED BEHAVIOR:** Server problems surface on Today; normal operation stays quiet.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/attention.go`, `web/templates/today.html`, `web/templates/ui_shell.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D9
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-093 (System page: sanitized public view and tailnet detail), GYST-VN-067 (Attention stored conditions, snooze and dismiss), GYST-VN-035 (Today: exceptions-only status area)
- **ACCEPTANCE CRITERIA:**
  - Normal day shows one muted chip at most
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-095"></a>
### GYST-VN-095 — Schedule the n8n Postgres backup

- **ID:** GYST-VN-095
- **NAME:** Schedule the n8n Postgres backup
- **PROJECT:** Server / host operations
- **REPOSITORY:** Host
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 7 — Uptime / System read-only · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Schedule a daily dump of n8n's Postgres with off-host copy and Kuma push.
- **CURRENT PROBLEM:** Stage 2A: n8n's Postgres backup isn't scheduled.
- **DETAILED REQUIREMENTS:**
  - `pg_dump` via a host timer; `0600` output; retention like GYST
  - Off-host copy reuses the off-host mechanism
- **PROPOSED BEHAVIOR:** n8n workflows and history survive a disk loss.
- **RELEVANT FILES / LIKELY AREAS:** Host scripts.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §N.2.
- **DEPENDENCIES:** GYST-VN-012 (Off-host backup copy)
- **ACCEPTANCE CRITERIA:**
  - Two scheduled dumps observed; one test restore into a scratch database
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable the timer.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Dump includes credentials: storage permissions matter.
- **SUGGESTED IMPLEMENTER:** Agent drafts the runbook, script or analysis; Jay executes on the host or decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

## Phase 8 — GitHub read-only / Development (M4)

<a id="gyst-vn-096"></a>
### GYST-VN-096 — GitHub auth decision: App-first or temporary PAT

- **ID:** GYST-VN-096
- **NAME:** GitHub auth decision: App-first or temporary PAT
- **PROJECT:** Governance
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Research
- **PRIORITY:** P1
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-only (Jay)
- **SUMMARY:** Decide whether the read phase starts with the GitHub App directly or with a short-lived fine-grained PAT.
- **CURRENT PROBLEM:** `08A`/`09` §H: the App is the target; a bootstrap PAT is allowed with strict limits. Unresolved.
- **DETAILED REQUIREMENTS:**
  - Recommendation: **App-first** (setup cost is small; avoids a credential replacement step)
  - If PAT: the PAT task's constraints apply in full
  - Recorded in the task ledger/ADR
- **PROPOSED BEHAVIOR:** A clear credential path for GitHub read.
- **RELEVANT FILES / LIKELY AREAS:** `docs/adr/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Decision recorded
- **TEST PLAN:** —
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** —
- **OUT OF SCOPE:** Write permissions.
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-097"></a>
### GYST-VN-097 — Temporary fine-grained PAT (only if chosen)

- **ID:** GYST-VN-097
- **NAME:** Temporary fine-grained PAT (only if chosen)
- **PROJECT:** Server / host operations
- **REPOSITORY:** GitHub account settings; host secret file
- **TYPE:** Operations
- **PRIORITY:** P3
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** If the PAT path is chosen: create, store, track and later revoke a read-only fine-grained PAT.
- **CURRENT PROBLEM:** Conditional on the auth decision.
- **DETAILED REQUIREMENTS:**
  - Repositories: `gyst-lifeops` only
  - Permissions: Metadata R, Issues R, Pull requests R, Checks R, Commit statuses R; **no write**
  - Expiry ≤ 30 days; no renewal
  - Stored in a `0600` secret file; never in DB, logs or pages
  - Replacement milestone: the GitHub App is in place before GitHub write **and** before expiry
  - Revocation verification: revoke in GitHub; a request with it returns 401; remove from the secret file; record in the ledger
- **PROPOSED BEHAVIOR:** GitHub read can start before the App exists, for at most 30 days.
- **RELEVANT FILES / LIKELY AREAS:** Host secret file; ledger.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §H (bootstrap PAT).
- **DEPENDENCIES:** GYST-VN-096 (GitHub auth decision: App-first or temporary PAT)
- **ACCEPTANCE CRITERIA:**
  - Scope and expiry recorded (not the value)
  - Revocation evidence recorded
- **TEST PLAN:** Manual API check (401 after revoke).
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revoke immediately.
- **OUT OF SCOPE:** Any write.
- **KNOWN RISKS:** Forgotten expiry; Attention reminder at day 21.
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-098"></a>
### GYST-VN-098 — Register the GYST GitHub App (read permissions)

- **ID:** GYST-VN-098
- **NAME:** Register the GYST GitHub App (read permissions)
- **PROJECT:** Server / host operations
- **REPOSITORY:** GitHub (App settings)
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Create the private GYST App with read permissions and webhooks, installed on `gyst-lifeops` only.
- **CURRENT PROBLEM:** `08A`: a private GitHub App is the target identity for GYST.
- **DETAILED REQUIREMENTS:**
  - Permissions: Metadata R, Issues R, Pull requests R, Checks R, Commit statuses R, Actions R; webhook secret
  - Private key `0600` on the host; never in the repo
  - Installation limited to `gyst-lifeops`
- **PROPOSED BEHAVIOR:** GYST has its own least-privilege GitHub identity.
- **RELEVANT FILES / LIKELY AREAS:** GitHub settings; host secret files.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §H.
- **DEPENDENCIES:** GYST-VN-096 (GitHub auth decision: App-first or temporary PAT), GYST-VN-009 (Harden host file permissions and record verification)
- **ACCEPTANCE CRITERIA:**
  - App permissions screenshot/summary recorded (no secrets)
  - Installation shows one repository
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Uninstall/delete the App.
- **OUT OF SCOPE:** Write permissions.
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-099"></a>
### GYST-VN-099 — Branch protection / ruleset on gyst-lifeops

- **ID:** GYST-VN-099
- **NAME:** Branch protection / ruleset on gyst-lifeops
- **PROJECT:** Governance
- **REPOSITORY:** GitHub settings
- **TYPE:** Operations
- **PRIORITY:** P1
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Protect the base branch: PR required, 1 approval, stale approvals dismissed, status checks required, no force push or deletion, no agent bypass.
- **CURRENT PROBLEM:** `09` §H / G12: required before any agent PR; useful immediately for all work.
- **DETAILED REQUIREMENTS:**
  - Ruleset on the base branch chosen in the branch-baseline task
  - Jay is the only bypass actor for his own PRs
  - No GYST or agent credential has Administration permission
- **PROPOSED BEHAVIOR:** Nothing reaches the base branch without review and green CI.
- **RELEVANT FILES / LIKELY AREAS:** GitHub repository settings.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` gate G12.
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - Ruleset summary recorded
  - A test push to the base branch is refused
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Relax the ruleset (Jay).
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Slows Jay's own quick fixes; bypass documented.
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-100"></a>
### GYST-VN-100 — Create the GitHub label taxonomy

- **ID:** GYST-VN-100
- **NAME:** Create the GitHub label taxonomy
- **PROJECT:** Governance
- **REPOSITORY:** GitHub labels; manifest in `JayDalool/gyst-lifeops`
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Create the canonical labels in `gyst-lifeops`: `status:*`, `priority:p0–p3`, `type:*`, `agent:claude/codex/ready`, `risk:*`, `needs:security-review`.
- **CURRENT PROBLEM:** `08A`: labels are canonical for workflow state; they don't exist yet.
- **DETAILED REQUIREMENTS:**
  - Agent prepares a label manifest (name, colour, description); violet not used for states
  - Jay applies with `gh label create` or the UI
- **PROPOSED BEHAVIOR:** Development has a consistent vocabulary.
- **RELEVANT FILES / LIKELY AREAS:** Manifest in `docs/`; GitHub labels.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §H (status rules).
- **DEPENDENCIES:** GYST-VN-001 (Resolve the vNext implementation branch baseline)
- **ACCEPTANCE CRITERIA:**
  - All labels exist exactly once
- **TEST PLAN:** `gh label list`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Delete labels.
- **OUT OF SCOPE:** Creating issues (a separate, Jay-approved step).
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Agent drafts the runbook, script or analysis; Jay executes on the host or decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-101"></a>
### GYST-VN-101 — GitHub read provider: repositories and issues

- **ID:** GYST-VN-101
- **NAME:** GitHub read provider: repositories and issues
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Integration
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Sync repository metadata and issues (title, state, labels, assignees, updated time) into cache tables, read-only.
- **CURRENT PROBLEM:** Development needs GitHub data; GitHub Issues are canonical and GYST keeps only a cache.
- **DETAILED REQUIREMENTS:**
  - Cache/reference tables only where needed (issues, repos); GitHub remains the source of truth
  - Credential from the App (installation token) or the PAT; fixed host `api.github.com` (G13)
  - ETags and rate-limit handling; no write calls exist in the client (compile-time: read-only client type)
  - Sync freshness stored for stale detection
- **PROPOSED BEHAVIOR:** GYST mirrors GitHub issues for display.
- **RELEVANT FILES / LIKELY AREAS:** New `internal/services/github/`, `migrations/`, new `internal/repository/github.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §H; `08A` source-of-truth model; gates G3/G13.
- **DEPENDENCIES:** GYST-VN-096 (GitHub auth decision: App-first or temporary PAT), GYST-VN-072 (Consolidate configuration into one struct), GYST-VN-004 (UTC instants for new audit and event records), GYST-VN-081 (Machine credentials with scopes)
- **ACCEPTANCE CRITERIA:**
  - The client type has no write methods
  - Tests with recorded fixtures
  - Issue bodies stored/rendered safely (escaped Markdown subset)
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable via config; tables stay.
- **OUT OF SCOPE:** Writes.
- **KNOWN RISKS:** Rendering untrusted issue Markdown; escape and restrict.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-102"></a>
### GYST-VN-102 — GitHub PR, review and check status

- **ID:** GYST-VN-102
- **NAME:** GitHub PR, review and check status
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Integration
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Sync PRs linked to issues, their review state and check/workflow status.
- **CURRENT PROBLEM:** Reviews and CI failures must be visible for the Development and Attention flows.
- **DETAILED REQUIREMENTS:**
  - PR ↔ issue link via `Refs #n`/`Fixes #n`
  - Review state, check conclusion, draft flag
  - No diffs stored
- **PROPOSED BEHAVIOR:** Jay sees which PRs need review and which failed CI.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/`, `internal/repository/github.go`, `migrations/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §H.
- **DEPENDENCIES:** GYST-VN-101 (GitHub read provider: repositories and issues)
- **ACCEPTANCE CRITERIA:**
  - Fixture tests for each review/check state
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Merge (never).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-103"></a>
### GYST-VN-103 — GitHub webhook receiver: verify, refetch, dedupe

- **ID:** GYST-VN-103
- **NAME:** GitHub webhook receiver: verify, refetch, dedupe
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Integration
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Receive GitHub webhooks, verify the signature, deduplicate by delivery id and **refetch** the object rather than trusting the payload.
- **CURRENT PROBLEM:** Polling alone is slow; webhooks must be treated as signals only.
- **DETAILED REQUIREMENTS:**
  - HMAC-SHA256 verification (constant time)
  - Dedupe via `processed_events`
  - Refetch the issue/PR through the read client
  - Route reachable publicly only at the documented exact path (Cloudflare Access bypass inventory)
- **PROPOSED BEHAVIOR:** Development updates within seconds.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/` (webhook route), `internal/services/github/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §F; `10A` D-S1 (machine endpoints bypass Access).
- **DEPENDENCIES:** GYST-VN-101 (GitHub read provider: repositories and issues), GYST-VN-083 (Processed-event idempotency store), GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine)
- **ACCEPTANCE CRITERIA:**
  - Negative tests: bad signature, replayed delivery
  - Audit `github.webhook.rejected`
- **TEST PLAN:** `go test ./...`; GitHub redelivery test after deploy.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable the webhook in GitHub.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Public endpoint; strict size limits.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-104"></a>
### GYST-VN-104 — Catch-up polling and stale/failure detection

- **ID:** GYST-VN-104
- **NAME:** Catch-up polling and stale/failure detection
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Integration
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Poll periodically to catch missed webhooks and mark data stale when GitHub is unreachable.
- **CURRENT PROBLEM:** Webhooks can be missed; the UI must show freshness honestly.
- **DETAILED REQUIREMENTS:**
  - Every 10 min incremental sync; backoff on errors
  - Stale threshold → Attention warning and disabled writes (later)
- **PROPOSED BEHAVIOR:** Data stays correct even when webhooks fail.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-101 (GitHub read provider: repositories and issues)
- **ACCEPTANCE CRITERIA:**
  - Tests with a failing fake API
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Rate limits; conditional requests.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-105"></a>
### GYST-VN-105 — Development area: task list

- **ID:** GYST-VN-105
- **NAME:** Development area: task list
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add Development as the fourth Projects segment: Needs you → In progress → Ready → Inbox link → Repositories, from the GitHub cache.
- **CURRENT PROBLEM:** D2: Development is a first-class area inside Projects; it can only appear once GitHub data exists.
- **DETAILED REQUIREMENTS:**
  - Rows: status tile · title · #ID · repo · agent/review signal · risk/priority tags only when notable · state chip
  - Footer "Synced from GitHub n min ago"; stale chip
  - Unhide the Development segment now
  - Inspector at ≥ 1280 px
- **PROPOSED BEHAVIOR:** Jay sees development work next to his projects.
- **RELEVANT FILES / LIKELY AREAS:** New `web/templates/development.html`, `internal/handlers/` (dev routes), `web/static/css/ui-projects.css`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `05A` D2; `12` §Development; `13A` DD-4; prototype `prototypes/development.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-101 (GitHub read provider: repositories and issues), GYST-VN-039 (Projects area sub-navigation: Projects · Development · Galaxy · Roadmaps)
- **ACCEPTANCE CRITERIA:**
  - Empty state "No development tasks"
  - State colours: Reviewing uses info blue, not violet (DD-4)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert; hide the segment.
- **OUT OF SCOPE:** Writes.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-106"></a>
### GYST-VN-106 — Development task detail (read-only)

- **ID:** GYST-VN-106
- **NAME:** Development task detail (read-only)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Issue detail: hero, workflow strip (Spec · Ready · Working · Review · Done), agent + risk row, acceptance checklist, disclosures, Open on GitHub.
- **CURRENT PROBLEM:** Needed before any write action.
- **DETAILED REQUIREMENTS:**
  - Risk display per `12`: Normal neutral, Elevated amber, Privileged red, Not eligible "Human only"
  - Issue Markdown rendered safely
  - Actions shown disabled with reason until the write phase
- **PROPOSED BEHAVIOR:** Jay can understand an issue without leaving GYST.
- **RELEVANT FILES / LIKELY AREAS:** New `web/templates/dev_task.html`, `internal/handlers/`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Development Task; prototype `prototypes/dev-task.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-105 (Development area: task list)
- **ACCEPTANCE CRITERIA:**
  - Untrusted Markdown can't inject script (test)
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Writes.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-107"></a>
### GYST-VN-107 — Reviews and Repositories views

- **ID:** GYST-VN-107
- **NAME:** Reviews and Repositories views
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Reviews queue (waiting for Jay; merged · verify & close) and the Repositories list, as filtered views of the same data.
- **CURRENT PROBLEM:** D5: Reviews must be a view of the one model, not a separate system.
- **DETAILED REQUIREMENTS:**
  - Reviews rows open a summary sheet/inspector with "Open PR" (GitHub)
  - No diffs shown; GYST never merges
  - Repos: name · linked project · open count · CI dot
- **PROPOSED BEHAVIOR:** A fast review queue.
- **RELEVANT FILES / LIKELY AREAS:** New `web/templates/reviews.html`, `web/templates/development.html` (repos section).
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Reviews, §Repositories; prototype `prototypes/reviews.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-102 (GitHub PR, review and check status), GYST-VN-105 (Development area: task list)
- **ACCEPTANCE CRITERIA:**
  - Empty "No reviews waiting"
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Request changes (write phase).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-108"></a>
### GYST-VN-108 — GitHub Attention conditions

- **ID:** GYST-VN-108
- **NAME:** GitHub Attention conditions
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 8 — GitHub read-only / Development · Milestone M4
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Producers for PR needs review (needs_action), CI failure on an open PR (warning), inconsistent `status:*` labels (warning) and merged-not-verified (needs_action).
- **CURRENT PROBLEM:** Development signals must flow into the one Attention model.
- **DETAILED REQUIREMENTS:**
  - Upsert/resolve via the stored-conditions API
  - Grouped Today row "2 reviews waiting"
  - Inconsistent labels are shown, never auto-fixed
- **PROPOSED BEHAVIOR:** Reviews and failures reach Today and the badge.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/`, `internal/services/attention.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §H (consistency rules).
- **DEPENDENCIES:** GYST-VN-102 (GitHub PR, review and check status), GYST-VN-067 (Attention stored conditions, snooze and dismiss)
- **ACCEPTANCE CRITERIA:**
  - Tests for each condition and auto-resolve
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Noise; grouping rules.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

## Phase 9 — n8n events (M4)

<a id="gyst-vn-109"></a>
### GYST-VN-109 — Separate n8n identities for GYST and SamVision

- **ID:** GYST-VN-109
- **NAME:** Separate n8n identities for GYST and SamVision
- **PROJECT:** Server / host operations
- **REPOSITORY:** n8n (live config)
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 9 — n8n events · Milestone M4
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Create GYST's n8n machine credential (`automation:event`, namespace `n8n`) and separate per-app Header-Auth credentials in n8n; SamVision keeps its own.
- **CURRENT PROBLEM:** `08A` D-J6 / `09` §L: no full n8n API key in GYST; apps must not share credentials.
- **DETAILED REQUIREMENTS:**
  - GYST credential created via the admin UI (token shown once) and stored in n8n
  - Naming rule `<APP>_N8N_*`
  - Registered workflow ids recorded in GYST config
- **PROPOSED BEHAVIOR:** A compromised workflow can only send GYST events.
- **RELEVANT FILES / LIKELY AREAS:** n8n credentials (live); GYST config.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §L.
- **DEPENDENCIES:** GYST-VN-081 (Machine credentials with scopes), GYST-VN-082 (Machine credential admin (tailnet, step-up))
- **ACCEPTANCE CRITERIA:**
  - Credential list reviewed; no full API key in GYST
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revoke the credential.
- **OUT OF SCOPE:** Run Now.
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-110"></a>
### GYST-VN-110 — n8n event ingestion

- **ID:** GYST-VN-110
- **NAME:** n8n event ingestion
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Integration
- **PRIORITY:** P2
- **PHASE:** 9 — n8n events · Milestone M4
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Accept `automation:event` posts on the internal listener for registered workflows, deduplicated, and update automation status snapshots.
- **CURRENT PROBLEM:** GYST can't see automation results.
- **DETAILED REQUIREMENTS:**
  - Registered workflow ids only; unknown ids ignored + audited
  - Idempotency and ±5 min window
  - Namespace cap of 20 open Attention items
  - Failed runs → warning; repeated failures → critical
- **PROPOSED BEHAVIOR:** Automation status reaches GYST.
- **RELEVANT FILES / LIKELY AREAS:** `internal/handlers/` (machine routes), new `internal/services/automations.go`, `migrations/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §L; `08A` D-J6.
- **DEPENDENCIES:** GYST-VN-081 (Machine credentials with scopes), GYST-VN-083 (Processed-event idempotency store), GYST-VN-109 (Separate n8n identities for GYST and SamVision), GYST-VN-067 (Attention stored conditions, snooze and dismiss)
- **ACCEPTANCE CRITERIA:**
  - Negative tests: wrong namespace (SamVision credential), unknown workflow, replay
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revoke the credential.
- **OUT OF SCOPE:** Run Now.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-111"></a>
### GYST-VN-111 — Automations page (status only)

- **ID:** GYST-VN-111
- **NAME:** Automations page (status only)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 9 — n8n events · Milestone M4
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** List GYST's automations (name · last/next run · state) with failure detail and Open n8n; no Run Now yet.
- **CURRENT PROBLEM:** Status visibility is separate from Run Now by decision.
- **DETAILED REQUIREMENTS:**
  - Failed first; SamVision automations not listed
  - "n8n unreachable" banner when stale
  - More → Operate → Automations appears now
- **PROPOSED BEHAVIOR:** Jay sees which automations failed.
- **RELEVANT FILES / LIKELY AREAS:** New `web/templates/automations.html`, `internal/handlers/`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Automations; prototype `prototypes/automations.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-110 (n8n event ingestion), GYST-VN-026 (Simplify More to the grouped model)
- **ACCEPTANCE CRITERIA:**
  - Empty state; failed detail sheet
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Run Now; editing workflows (never).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

## Phase 10 — Agent visibility (M5)

<a id="gyst-vn-112"></a>
### GYST-VN-112 — Verify current official Claude Code authentication for unattended use

- **ID:** GYST-VN-112
- **NAME:** Verify current official Claude Code authentication for unattended use
- **PROJECT:** Agent Bridge
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Research
- **PRIORITY:** P2
- **PHASE:** 10 — Agent visibility · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Immediately before bridge implementation, verify from current official Anthropic documentation how Claude Code may authenticate for headless runs under a service account.
- **CURRENT PROBLEM:** `10A` D-S2: Jay has Claude Max; it must not be assumed that Max covers API usage, is valid for unattended runs, or how budgets apply.
- **DETAILED REQUIREMENTS:**
  - Read current official docs (no third-party posts) and record the URLs and dates
  - Answer: allowed auth methods for headless `claude -p`; terms for unattended/service use; budget controls (`--max-budget-usd`); where credentials live
  - Recommendation with cost implications; Jay decides
  - No API key created by an agent
- **PROPOSED BEHAVIOR:** The bridge uses a documented, permitted auth method.
- **RELEVANT FILES / LIKELY AREAS:** `docs/adr/` or `docs/tasks/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `10A` D-S2.
- **DEPENDENCIES:** GYST-VN-071 (Decide packaging and repository for Bridge, Server Provider and Action Runner)
- **ACCEPTANCE CRITERIA:**
  - Findings cite official sources with retrieval date
  - Jay's decision recorded
- **TEST PLAN:** —
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** —
- **OUT OF SCOPE:** Codex (separate task).
- **KNOWN RISKS:** Docs change often; must be done right before implementation, not earlier.
- **SUGGESTED IMPLEMENTER:** An agent researches and drafts; Jay decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-113"></a>
### GYST-VN-113 — Verify current official Codex CLI authentication for unattended use

- **ID:** GYST-VN-113
- **NAME:** Verify current official Codex CLI authentication for unattended use
- **PROJECT:** Agent Bridge
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Research
- **PRIORITY:** P2
- **PHASE:** 10 — Agent visibility · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Immediately before bridge implementation, verify from current official OpenAI documentation how Codex CLI may authenticate for headless runs.
- **CURRENT PROBLEM:** `10A` D-S2: subscription and API authentication must not be assumed interchangeable.
- **DETAILED REQUIREMENTS:**
  - Official docs only, with URLs and dates
  - Answer: `codex exec` auth options, sandbox flags (`--sandbox`, `--ignore-user-config`), usage limits and budgets
  - Jay decides
- **PROPOSED BEHAVIOR:** The bridge uses a documented, permitted auth method.
- **RELEVANT FILES / LIKELY AREAS:** `docs/adr/` or `docs/tasks/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `10A` D-S2.
- **DEPENDENCIES:** GYST-VN-071 (Decide packaging and repository for Bridge, Server Provider and Action Runner)
- **ACCEPTANCE CRITERIA:**
  - Findings cite official sources
  - Decision recorded
- **TEST PLAN:** —
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** —
- **OUT OF SCOPE:** Claude (separate task).
- **KNOWN RISKS:** Same as the Claude task.
- **SUGGESTED IMPLEMENTER:** An agent researches and drafts; Jay decides
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-114"></a>
### GYST-VN-114 — Create gyst-bridge and gyst-agent service accounts

- **ID:** GYST-VN-114
- **NAME:** Create gyst-bridge and gyst-agent service accounts
- **PROJECT:** Server / host operations
- **REPOSITORY:** Host; runbook in the ops repository
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 10 — Agent visibility · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Create the two service accounts (no sudo, docker or lxd groups; `nologin`), `/srv/agents` (`0700`), and the narrow polkit rule for the template unit.
- **CURRENT PROBLEM:** `09` §I.2: agent processes need an unprivileged OS identity separate from the bridge.
- **DETAILED REQUIREMENTS:**
  - **Precondition (O-1):** the file-permission verification record exists and is re-run the same day with no findings
  - An agent drafts the exact commands and a verification script; Jay runs them
  - Verify: `id` shows no supplementary groups; the agent can't read `/srv/server` secrets or other homes; can't reach the Docker socket
- **PROPOSED BEHAVIOR:** An isolated identity exists for agent runs.
- **RELEVANT FILES / LIKELY AREAS:** Host accounts; runbook `docs/runbooks/agent-accounts.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §I (agent security), §J (risk classes); `10A` D-S2, D-S3, D-S4, D-S6, O-1, O-2, O-3.
- **DEPENDENCIES:** GYST-VN-009 (Harden host file permissions and record verification), GYST-VN-071 (Decide packaging and repository for Bridge, Server Provider and Action Runner)
- **ACCEPTANCE CRITERIA:**
  - Verification script output recorded (no secrets)
  - Negative checks (secret read, docker) fail as expected
- **TEST PLAN:** The verification script.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** `userdel` and remove `/srv/agents`.
- **OUT OF SCOPE:** Installing CLIs (bridge tasks).
- **KNOWN RISKS:** Group membership mistakes; verified by script.
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-115"></a>
### GYST-VN-115 — Non-Docker Go toolchain for gyst-agent

- **ID:** GYST-VN-115
- **NAME:** Non-Docker Go toolchain for gyst-agent
- **PROJECT:** Server / host operations
- **REPOSITORY:** Host
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 10 — Agent visibility · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Install a pinned Go SDK under `/srv/agents/sdk` for the agent account and check race-detector feasibility.
- **CURRENT PROBLEM:** `10A` O-2: the host has no Go and the agent has no Docker; tests must run in the worktree.
- **DETAILED REQUIREMENTS:**
  - Pinned version matching `Dockerfile` (Go 1.25)
  - Checksum-verified download from the official source
  - Check whether `go test -race` works (CGO/toolchain); record the result and the fallback
- **PROPOSED BEHAVIOR:** Agents can run the repository's checks.
- **RELEVANT FILES / LIKELY AREAS:** `/srv/agents/sdk`; runbook.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `10A` O-2.
- **DEPENDENCIES:** GYST-VN-114 (Create gyst-bridge and gyst-agent service accounts)
- **ACCEPTANCE CRITERIA:**
  - `go version` as gyst-agent
  - `go test ./...` passes on a clean clone as gyst-agent
  - Race-detector result recorded
- **TEST PLAN:** As listed.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Remove the SDK directory.
- **OUT OF SCOPE:** Other toolchains.
- **KNOWN RISKS:** Version drift from CI; pin both.
- **SUGGESTED IMPLEMENTER:** Jay, with an agent-drafted runbook
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-116"></a>
### GYST-VN-116 — Agent Bridge read-only: heartbeat and status

- **ID:** GYST-VN-116
- **NAME:** Agent Bridge read-only: heartbeat and status
- **PROJECT:** Agent Bridge
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist (GYST-side handlers in `JayDalool/gyst-lifeops`)
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 10 — Agent visibility · Milestone M5
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** First bridge release: runs as `gyst-bridge`, reports availability of Claude/Codex and a heartbeat to GYST via `agent:report`; no dispatch.
- **CURRENT PROBLEM:** GYST can't show agent availability; the bridge must exist before dispatch.
- **DETAILED REQUIREMENTS:**
  - Pull-only toward GYST's internal machine listener (no inbound listener)
  - Heartbeat every 30 s; GYST marks Unknown after 3 min
  - CLI versions pinned; no auto-update
  - Config owned by `gyst-bridge`, not writable by `gyst-agent`
  - GYST side: `agent:report` handler, runs/agents tables (additive)
- **PROPOSED BEHAVIOR:** The Agents page can show whether agents are available.
- **RELEVANT FILES / LIKELY AREAS:** Bridge repository per packaging decision; GYST `internal/handlers/` (machine routes), `migrations/`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §I (agent security), §J (risk classes); `10A` D-S2, D-S3, D-S4, D-S6, O-1, O-2, O-3.
- **DEPENDENCIES:** GYST-VN-114 (Create gyst-bridge and gyst-agent service accounts), GYST-VN-081 (Machine credentials with scopes), GYST-VN-112 (Verify current official Claude Code authentication for unattended use), GYST-VN-113 (Verify current official Codex CLI authentication for unattended use), GYST-VN-083 (Processed-event idempotency store)
- **ACCEPTANCE CRITERIA:**
  - Negative test: `agent:report` credential can't call other machine routes
  - Heartbeat loss → Unknown
- **TEST PLAN:** Unit tests both sides; supervised host run.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Stop the bridge unit.
- **OUT OF SCOPE:** Dispatch.
- **KNOWN RISKS:** The bridge is PRIVILEGED code; reviewed accordingly.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-117"></a>
### GYST-VN-117 — Agents page and run detail (read-only)

- **ID:** GYST-VN-117
- **NAME:** Agents page and run detail (read-only)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** UI
- **PRIORITY:** P2
- **PHASE:** 10 — Agent visibility · Milestone M5
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Agents: Now, Queue, Recent, Available with limits line and heartbeat footer; run detail with facts and safe activity.
- **CURRENT PROBLEM:** Agent visibility precedes any dispatch.
- **DETAILED REQUIREMENTS:**
  - States Idle · Queued · Working · Reviewing · Needs input · Failed · Done only
  - Reviewing uses info blue (DD-4), not violet
  - Never shows credentials, raw prompts or transcripts; logs tailnet-only (later)
  - Stop button shown disabled until dispatch exists
  - Development segment pill "Agents"
- **PROPOSED BEHAVIOR:** Jay can see what agents are doing.
- **RELEVANT FILES / LIKELY AREAS:** New `web/templates/agents.html`, `web/templates/agent_detail.html`, `internal/handlers/`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`; `12` §Agents, §Agent Detail; `13A` DD-4; prototype `prototypes/agents.html`, `agent-detail.html`
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-116 (Agent Bridge read-only: heartbeat and status), GYST-VN-022 (State components: chips, banners, empty, stale, error, offline), GYST-VN-105 (Development area: task list)
- **ACCEPTANCE CRITERIA:**
  - Unknown state after heartbeat loss
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** Dispatch controls.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** M

## Phase 11 — GitHub write (M5)

<a id="gyst-vn-118"></a>
### GYST-VN-118 — Confirm the classification of meaningful GitHub writes

- **ID:** GYST-VN-118
- **NAME:** Confirm the classification of meaningful GitHub writes
- **PROJECT:** Governance
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Documentation
- **PRIORITY:** P2
- **PHASE:** 11 — GitHub write · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-only (Jay)
- **SUMMARY:** Confirm the write classification proposed in `14` §E.3 and encode it as the policy table used by the write handlers.
- **CURRENT PROBLEM:** `10A` D-S5 carried forward: which GitHub writes need re-authentication was left for the roadmap.
- **DETAILED REQUIREMENTS:**
  - Jay confirms or edits `14` §E.3
  - The confirmed table becomes a code-level policy map (action → PA / PR / TO) with a test per action
- **PROPOSED BEHAVIOR:** Write handlers enforce the agreed re-auth rules.
- **RELEVANT FILES / LIKELY AREAS:** `docs/audit/vnext/14_IMPLEMENTATION_ROADMAP.md` §E.3; later `internal/services/github/policy.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-079 (Step-up re-authentication, action nonce and confirmation pattern)
- **ACCEPTANCE CRITERIA:**
  - Decision recorded
  - Policy test exists in the first write task
- **TEST PLAN:** —
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** —
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-119"></a>
### GYST-VN-119 — Grant the GYST App Issues write

- **ID:** GYST-VN-119
- **NAME:** Grant the GYST App Issues write
- **PROJECT:** Server / host operations
- **REPOSITORY:** GitHub (App settings)
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 11 — GitHub write · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Add Issues read/write to the GYST App; revoke the bootstrap PAT if one exists and verify the revocation.
- **CURRENT PROBLEM:** Writes require the App identity (a PAT must be replaced before write).
- **DETAILED REQUIREMENTS:**
  - Issues RW only; nothing else
  - If a PAT exists: revoke, check 401, remove from the secret file, record
- **PROPOSED BEHAVIOR:** GYST can write issues as its own identity.
- **RELEVANT FILES / LIKELY AREAS:** GitHub App settings.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §H.
- **DEPENDENCIES:** GYST-VN-098 (Register the GYST GitHub App (read permissions)), GYST-VN-099 (Branch protection / ruleset on gyst-lifeops)
- **ACCEPTANCE CRITERIA:**
  - Permission change recorded
  - PAT revocation evidence (if applicable)
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Remove the permission.
- **OUT OF SCOPE:** Contents or admin permissions (never).
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-120"></a>
### GYST-VN-120 — Create issues and edit title/body from GYST

- **ID:** GYST-VN-120
- **NAME:** Create issues and edit title/body from GYST
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 11 — GitHub write · Milestone M5
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** "New task" in Development creates a GitHub issue; the task page edits title and body. Both are public-authenticated writes, audited.
- **CURRENT PROBLEM:** Development is read-only after Phase 8.
- **DETAILED REQUIREMENTS:**
  - Service-layer use case per write (G14); sanitized errors
  - Policy map from the classification task
  - Audit `github.write`; writes disabled when data is stale
  - Rate limit per session
- **PROPOSED BEHAVIOR:** Jay can capture development work from GYST.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/`, `internal/handlers/`, `web/templates/dev_task.html`, `web/templates/development.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` §G.2 (PA), gates G1, G2, G14, G15.
- **DEPENDENCIES:** GYST-VN-119 (Grant the GYST App Issues write), GYST-VN-106 (Development task detail (read-only)), GYST-VN-075 (Security headers including frame-ancestors), GYST-VN-015 (Structured application logging and sanitized client errors), GYST-VN-118 (Confirm the classification of meaningful GitHub writes)
- **ACCEPTANCE CRITERIA:**
  - Negative tests: CSRF, stale data, missing session
  - UI checks
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Disable the write feature flag.
- **OUT OF SCOPE:** Labels, comments.
- **KNOWN RISKS:** Untrusted content round-trip; escape.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-121"></a>
### GYST-VN-121 — Edit labels and status, assign an agent

- **ID:** GYST-VN-121
- **NAME:** Edit labels and status, assign an agent
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 11 — GitHub write · Milestone M5
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Change normal labels, set the workflow status label, and assign `agent:claude`/`agent:codex` (intent only) from GYST.
- **CURRENT PROBLEM:** Workflow changes currently require GitHub.
- **DETAILED REQUIREMENTS:**
  - `agent:ready`, `risk:*` and `status:done` excluded here (other tasks)
  - One `status:*` at a time
  - Audit each write
- **PROPOSED BEHAVIOR:** Jay moves work through the workflow from his phone.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/`, `web/templates/dev_task.html` (label sheet).
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` §G.2.
- **DEPENDENCIES:** GYST-VN-120 (Create issues and edit title/body from GYST), GYST-VN-100 (Create the GitHub label taxonomy)
- **ACCEPTANCE CRITERIA:**
  - Test: attempt to set `agent:ready` through this path is refused
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Feature flag.
- **OUT OF SCOPE:** `agent:ready`.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-122"></a>
### GYST-VN-122 — Comment on issues

- **ID:** GYST-VN-122
- **NAME:** Comment on issues
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P3
- **PHASE:** 11 — GitHub write · Milestone M5
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Add an issue comment from the task page.
- **CURRENT PROBLEM:** Answering agent questions needs comments.
- **DETAILED REQUIREMENTS:**
  - Plain-text/Markdown input; preview optional
  - Audit
- **PROPOSED BEHAVIOR:** Jay can answer from GYST.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/`, `web/templates/dev_task.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` §G.2 (PA).
- **DEPENDENCIES:** GYST-VN-120 (Create issues and edit title/body from GYST)
- **ACCEPTANCE CRITERIA:**
  - Tests
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Feature flag.
- **OUT OF SCOPE:** Editing others' comments.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-123"></a>
### GYST-VN-123 — Label normalizer (single automated status writer)

- **ID:** GYST-VN-123
- **NAME:** Label normalizer (single automated status writer)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 11 — GitHub write · Milestone M5
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Apply the `status:*` consistency rules on `closed`/`reopened` events as the only automated writer.
- **CURRENT PROBLEM:** `09` §H: issue state and labels must stay consistent without loops.
- **DETAILED REQUIREMENTS:**
  - Rules: closed-completed → `status:done`; not planned → remove `status:*`; reopened → `status:inbox`
  - Ignore events sent by the GYST App; write only when desired ≠ current; one write per event
  - Multiple/unknown `status:*` are not auto-fixed (Attention warning)
- **PROPOSED BEHAVIOR:** Status labels stay trustworthy.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/normalizer.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §H.
- **DEPENDENCIES:** GYST-VN-103 (GitHub webhook receiver: verify, refetch, dedupe), GYST-VN-119 (Grant the GYST App Issues write)
- **ACCEPTANCE CRITERIA:**
  - Loop-prevention tests
  - Tests per rule
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable the normalizer flag.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Loops; covered by tests.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-124"></a>
### GYST-VN-124 — Verify & close from GYST (re-auth)

- **ID:** GYST-VN-124
- **NAME:** Verify & close from GYST (re-auth)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 11 — GitHub write · Milestone M5
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** "Verify & close" on merged-not-verified issues closes the issue as completed after re-authentication.
- **CURRENT PROBLEM:** D-S3: agent PRs use `Refs #n`, so Jay closes after verification.
- **DETAILED REQUIREMENTS:**
  - Step-up required (classification `14` §E.3)
  - Audit; Attention item resolves
- **PROPOSED BEHAVIOR:** Jay closes the loop from GYST.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/`, `web/templates/reviews.html`, `web/templates/dev_task.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `10A` D-S3, D-S5.
- **DEPENDENCIES:** GYST-VN-079 (Step-up re-authentication, action nonce and confirmation pattern), GYST-VN-121 (Edit labels and status, assign an agent), GYST-VN-123 (Label normalizer (single automated status writer))
- **ACCEPTANCE CRITERIA:**
  - Refused without fresh re-auth
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Feature flag.
- **OUT OF SCOPE:** Merging.
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-125"></a>
### GYST-VN-125 — Cloudflare Access validation and machine-endpoint bypass inventory

- **ID:** GYST-VN-125
- **NAME:** Cloudflare Access validation and machine-endpoint bypass inventory
- **PROJECT:** Server / host operations
- **REPOSITORY:** Cloudflare (live config)
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 11 — GitHub write · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Validate Cloudflare Access in front of the public host before higher-impact public writes, and document the exact paths that bypass it.
- **CURRENT PROBLEM:** `10A` D-S1: Access is approved as an extra layer only after validation; machine endpoints must not need interactive login.
- **DETAILED REQUIREMENTS:**
  - Test and record: iPhone browser, installed PWA (standalone), desktop, GYST session persistence under Access, logout (GYST + Access), re-auth behaviour
  - Bypass inventory: exact paths only (Apple Health sync, GitHub webhook, and any other machine endpoint) with the reason and the non-Access authentication each uses
  - No wildcard bypasses
  - Rollback plan (disable the Access app)
- **PROPOSED BEHAVIOR:** An extra login layer protects the public host without breaking the PWA or machine endpoints.
- **RELEVANT FILES / LIKELY AREAS:** Cloudflare Zero Trust (live); record in `docs/runbooks/cloudflare-access.md`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `10A` D-S1.
- **DEPENDENCIES:** GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine), GYST-VN-103 (GitHub webhook receiver: verify, refetch, dedupe)
- **ACCEPTANCE CRITERIA:**
  - All six flows pass and are recorded
  - Each bypass path authenticates on its own (signature or scoped credential)
- **TEST PLAN:** Manual on devices.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable the Access application.
- **OUT OF SCOPE:** Changing tunnel config beyond Access.
- **KNOWN RISKS:** PWA standalone cookie behaviour on iOS; must be tested on the device.
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** M

## Phase 12 — Agent dispatch (M5)

<a id="gyst-vn-126"></a>
### GYST-VN-126 — Bridge execution templates and path policy

- **ID:** GYST-VN-126
- **NAME:** Bridge execution templates and path policy
- **PROJECT:** Agent Bridge
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Fixed implementer/reviewer templates and the per-repo path policy, owned by `gyst-bridge`, excluding repository-local config.
- **CURRENT PROBLEM:** `10A` O-3: templates must not load repo config; `09` §I.1 lists the flags.
- **DETAILED REQUIREMENTS:**
  - Claude: `--setting-sources user` (or `--bare` with explicit settings), `--strict-mcp-config`, explicit `--allowedTools`, permission mode from template
  - Codex: `--ignore-user-config`, `--ignore-rules`, `--sandbox workspace-write` (review: `read-only`)
  - Git: `core.hooksPath=/dev/null`, no includes, `protocol.file.allow=never`
  - Bridge preamble: repository and issue text are untrusted data
  - Path policy for `gyst-lifeops` per `09` §J
- **PROPOSED BEHAVIOR:** Agents run with fixed, reviewable instructions and limits.
- **RELEVANT FILES / LIKELY AREAS:** Bridge repository (templates, policy).
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §I (agent security), §J (risk classes); `10A` D-S2, D-S3, D-S4, D-S6, O-1, O-2, O-3.
- **DEPENDENCIES:** GYST-VN-116 (Agent Bridge read-only: heartbeat and status)
- **ACCEPTANCE CRITERIA:**
  - Tests: a repo-local settings/hook file has no effect
  - Policy file not writable by gyst-agent
- **TEST PLAN:** Unit tests; a supervised dry run on a scratch repo.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert templates.
- **OUT OF SCOPE:** Other repositories.
- **KNOWN RISKS:** Flag behaviour changes with CLI versions; pinned versions.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-127"></a>
### GYST-VN-127 — Repository mirror and worktree isolation

- **ID:** GYST-VN-127
- **NAME:** Repository mirror and worktree isolation
- **PROJECT:** Agent Bridge
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Bare mirrors and per-run worktrees under `/srv/agents`, fetched with a read token, pushing task branches only with 1-hour installation tokens.
- **CURRENT PROBLEM:** Runs must never share a working tree or touch production checkouts.
- **DETAILED REQUIREMENTS:**
  - Mirror `/srv/agents/repos/<owner>/<repo>.git`; worktree `/srv/agents/worktrees/<repo>/<issue>-<run>`
  - Clean environment allowlist; `umask 077`
  - Agent App private key readable only by gyst-bridge; per-run 1 h single-repo token passed to the child
  - Never follows symlinks when reading worktree files for summaries
- **PROPOSED BEHAVIOR:** Each run is isolated.
- **RELEVANT FILES / LIKELY AREAS:** Bridge repository.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §I (agent security), §J (risk classes); `10A` D-S2, D-S3, D-S4, D-S6, O-1, O-2, O-3.
- **DEPENDENCIES:** GYST-VN-116 (Agent Bridge read-only: heartbeat and status), GYST-VN-115 (Non-Docker Go toolchain for gyst-agent)
- **ACCEPTANCE CRITERIA:**
  - Test: two runs never share a path
  - Token scope verified (single repo)
- **TEST PLAN:** Unit tests; supervised run.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Delete worktrees.
- **OUT OF SCOPE:** Other repos.
- **KNOWN RISKS:** Disk growth; bounded by the cleanup task.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-128"></a>
### GYST-VN-128 — Risk classifier in the bridge

- **ID:** GYST-VN-128
- **NAME:** Risk classifier in the bridge
- **PROJECT:** Agent Bridge
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Derive the risk class from labels, implied minimums and the path policy; refuse when the derived class exceeds the approved class; post-run path check.
- **CURRENT PROBLEM:** `09` §J: labels alone can't approve; keywords never lower a class.
- **DETAILED REQUIREMENTS:**
  - Explicit `risk:*` > implied (`type:security` ≥ ELEVATED, `type:infrastructure` ≥ PRIVILEGED) > path policy; unknown → ELEVATED
  - Post-run changed-path check raises the class; excess → PR stays draft + Attention "Policy: re-approval needed"
  - NOT AGENT-ELIGIBLE always refused
- **PROPOSED BEHAVIOR:** Agents can't be tricked into higher-risk work.
- **RELEVANT FILES / LIKELY AREAS:** Bridge repository.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §I (agent security), §J (risk classes); `10A` D-S2, D-S3, D-S4, D-S6, O-1, O-2, O-3.
- **DEPENDENCIES:** GYST-VN-126 (Bridge execution templates and path policy)
- **ACCEPTANCE CRITERIA:**
  - Table tests for every rule; keyword-lowering attempt fails
- **TEST PLAN:** Unit tests.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Classifier gaps; default to ELEVATED.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-129"></a>
### GYST-VN-129 — Approval records and start_run commands in GYST

- **ID:** GYST-VN-129
- **NAME:** Approval records and start_run commands in GYST
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Store approvals (issue, class, approver session, time; single use for PRIVILEGED) and issue `start_run` commands via `agent:commands` only after checking them.
- **CURRENT PROBLEM:** Approval must live in GYST's audit log, not only in a label.
- **DETAILED REQUIREMENTS:**
  - Approval requires step-up; ELEVATED/PRIVILEGED only on the privileged listener
  - Command carries the approved class; bridge re-derives and refuses on mismatch
  - Audit `agent.approval`
- **PROPOSED BEHAVIOR:** Only approved work can start.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/agents.go`, `migrations/`, `internal/handlers/` (privileged routes).
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §J; `10A` D-S4.
- **DEPENDENCIES:** GYST-VN-079 (Step-up re-authentication, action nonce and confirmation pattern), GYST-VN-081 (Machine credentials with scopes), GYST-VN-073 (Dual listener: public, privileged (tailnet) and internal machine)
- **ACCEPTANCE CRITERIA:**
  - Negative tests: reuse of a PRIVILEGED approval, approval from public for ELEVATED
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Feature flag.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-130"></a>
### GYST-VN-130 — Set agent:ready from GYST

- **ID:** GYST-VN-130
- **NAME:** Set agent:ready from GYST
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** "Ready for agent" sets `agent:ready`: NORMAL from public with re-auth; ELEVATED/PRIVILEGED tailnet-only with step-up; NOT AGENT-ELIGIBLE refused.
- **CURRENT PROBLEM:** `10A` D-S4 defines where each class may be approved.
- **DETAILED REQUIREMENTS:**
  - 10 safeguards from `08A` checked before the label is set (acceptance present, risk derived, repo allowlisted, limits, etc.)
  - Public use only after Cloudflare Access validation
- **PROPOSED BEHAVIOR:** Jay can queue NORMAL work from his phone safely.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/github/`, `internal/services/agents.go`, `web/templates/dev_task.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `10A` D-S4; `08A` agent:ready safeguards.
- **DEPENDENCIES:** GYST-VN-129 (Approval records and start_run commands in GYST), GYST-VN-121 (Edit labels and status, assign an agent), GYST-VN-125 (Cloudflare Access validation and machine-endpoint bypass inventory), GYST-VN-099 (Branch protection / ruleset on gyst-lifeops)
- **ACCEPTANCE CRITERIA:**
  - Tests per class and ingress
  - Refusal wording shown for NOT AGENT-ELIGIBLE
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Feature flag.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-131"></a>
### GYST-VN-131 — Agent dispatch: queue, run and stop

- **ID:** GYST-VN-131
- **NAME:** Agent dispatch: queue, run and stop
- **PROJECT:** Agent Bridge
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** The bridge pulls approved `start_run` commands, starts runs as systemd template units under `gyst-agent`, reports progress, and stops on request or timeout.
- **CURRENT PROBLEM:** No agent can run from GYST yet.
- **DETAILED REQUIREMENTS:**
  - Enforces from the start: 1 implementation job, 45 min timeout, 0 automatic retries, `MemoryMax`/`CPUQuota`/`TasksMax`
  - Stop kills the process group; Stop is public (fail-safe)
  - Implementer opens a PR with `Refs #n` from the agent identity
  - Dispatch stays disabled in production until the go-live task
- **PROPOSED BEHAVIOR:** Approved issues become agent PRs.
- **RELEVANT FILES / LIKELY AREAS:** Bridge repository; GYST `internal/services/agents.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §I (agent security), §J (risk classes); `10A` D-S2, D-S3, D-S4, D-S6, O-1, O-2, O-3.
- **DEPENDENCIES:** GYST-VN-127 (Repository mirror and worktree isolation), GYST-VN-128 (Risk classifier in the bridge), GYST-VN-129 (Approval records and start_run commands in GYST), GYST-VN-099 (Branch protection / ruleset on gyst-lifeops)
- **ACCEPTANCE CRITERIA:**
  - Supervised run on a scratch issue produces a draft PR
  - Timeout and Stop verified
- **TEST PLAN:** Unit tests; supervised host runs.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable dispatch in bridge config.
- **OUT OF SCOPE:** Review flow; cleanup.
- **KNOWN RISKS:** Runaway spend; per-run budget flag.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** L

<a id="gyst-vn-132"></a>
### GYST-VN-132 — Agent review flow

- **ID:** GYST-VN-132
- **NAME:** Agent review flow
- **PROJECT:** Agent Bridge
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Independent review job (the other agent, read-only sandbox, 15 min) on a PR, results summarised in GYST; "Request agent review" requires re-auth.
- **CURRENT PROBLEM:** `08A`: an independent agent review precedes Jay's review.
- **DETAILED REQUIREMENTS:**
  - 1 review job at a time; never on the PR being implemented
  - Summary without full diffs; findings as a PR review comment
  - Moves issue to `status:review` via a GYST command acknowledgement
- **PROPOSED BEHAVIOR:** Jay gets a second opinion before he reviews.
- **RELEVANT FILES / LIKELY AREAS:** Bridge repository; GYST Reviews view.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §I (agent security), §J (risk classes); `10A` D-S2, D-S3, D-S4, D-S6, O-1, O-2, O-3.
- **DEPENDENCIES:** GYST-VN-131 (Agent dispatch: queue, run and stop)
- **ACCEPTANCE CRITERIA:**
  - Supervised review of a scratch PR
- **TEST PLAN:** Unit tests; supervised run.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable review in config.
- **OUT OF SCOPE:** Auto-merge (never).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-133"></a>
### GYST-VN-133 — Agent limits configuration and cleanup

- **ID:** GYST-VN-133
- **NAME:** Agent limits configuration and cleanup
- **PROJECT:** Agent Bridge
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Make every limit explicit configuration, enforce the remaining ones, and clean up worktrees and logs.
- **CURRENT PROBLEM:** Limits must be visible and changeable only by the owner.
- **DETAILED REQUIREMENTS:**
  - Config table: repositories `gyst-lifeops` only; implementation jobs 1; review jobs 1; implementation timeout 45 min; review timeout 15 min; worktrees 5 max; automatic retries 0; runs per day 6; spend: per-run cap (for example Claude `--max-budget-usd 5`) plus provider budget alerts
  - Worktree removed on merge, cancel or 7 days idle; logs kept 14 days
  - Limits shown on the Agents page
- **PROPOSED BEHAVIOR:** Agent work stays small, bounded and tidy.
- **RELEVANT FILES / LIKELY AREAS:** Bridge config and repository; GYST Agents page.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §I.3.
- **DEPENDENCIES:** GYST-VN-131 (Agent dispatch: queue, run and stop), GYST-VN-132 (Agent review flow)
- **ACCEPTANCE CRITERIA:**
  - Tests: the 7th run in a day is refused; the 6th worktree triggers GC
- **TEST PLAN:** Unit tests.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert config.
- **OUT OF SCOPE:** Quiet hours (optional later).
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-134"></a>
### GYST-VN-134 — Agent Attention conditions

- **ID:** GYST-VN-134
- **NAME:** Agent Attention conditions
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** NORMAL
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Needs input (needs_action), failed run (warning; repeated → critical), policy violation (warning) and PR ready for review (needs_action).
- **CURRENT PROBLEM:** Agent events must reach Jay through the one Attention model.
- **DETAILED REQUIREMENTS:**
  - Upsert/resolve via stored conditions
- **PROPOSED BEHAVIOR:** Jay knows when an agent needs him.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/agents.go`, `internal/services/attention.go`.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** —
- **DEPENDENCIES:** GYST-VN-116 (Agent Bridge read-only: heartbeat and status), GYST-VN-067 (Attention stored conditions, snooze and dismiss)
- **ACCEPTANCE CRITERIA:**
  - Tests per condition
- **TEST PLAN:** `go test ./...`.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Revert.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex
- **SUGGESTED REVIEWER:** The other agent (Codex reviews Claude and vice versa), then Jay
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-135"></a>
### GYST-VN-135 — Enable agent dispatch for gyst-lifeops (first supervised runs)

- **ID:** GYST-VN-135
- **NAME:** Enable agent dispatch for gyst-lifeops (first supervised runs)
- **PROJECT:** Agent Bridge
- **REPOSITORY:** Host (bridge config)
- **TYPE:** Operations
- **PRIORITY:** P2
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Turn dispatch on for `gyst-lifeops` after all gates pass and supervise the first runs.
- **CURRENT PROBLEM:** Dispatch is built disabled; enabling it is an owner decision.
- **DETAILED REQUIREMENTS:**
  - Gate checklist: file permissions re-verified, branch protection on, Cloudflare Access validated (for public ready), limits configured, review flow working, Attention conditions working
  - First 3 runs on NORMAL documentation or test-only issues
  - Record outcomes
- **PROPOSED BEHAVIOR:** Agent-assisted development is live, within limits.
- **RELEVANT FILES / LIKELY AREAS:** Bridge config.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §O (Agent dispatch gates).
- **DEPENDENCIES:** GYST-VN-132 (Agent review flow), GYST-VN-133 (Agent limits configuration and cleanup), GYST-VN-134 (Agent Attention conditions), GYST-VN-130 (Set agent:ready from GYST), GYST-VN-125 (Cloudflare Access validation and machine-endpoint bypass inventory)
- **ACCEPTANCE CRITERIA:**
  - Checklist recorded; 3 supervised runs reviewed
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Disable dispatch.
- **OUT OF SCOPE:** SamVision.
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-136"></a>
### GYST-VN-136 — Evaluate adding SamVision to the agent allowlist

- **ID:** GYST-VN-136
- **NAME:** Evaluate adding SamVision to the agent allowlist
- **PROJECT:** Governance
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Research
- **PRIORITY:** P3
- **PHASE:** 12 — Agent dispatch · Milestone M5
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** Human-only (Jay)
- **SUMMARY:** After the trial period, decide on adding `samvisionai` using the eight exit criteria.
- **CURRENT PROBLEM:** `10A` D-S6: gyst-lifeops only, until evidence supports expansion.
- **DETAILED REQUIREMENTS:**
  - Evidence for each: worktree isolation, repository isolation, no credential exposure, branch behaviour, independent review, cleanup, useful logs, no host-privilege violations
  - SamVision path policy drafted before any run
  - Jay decides
- **PROPOSED BEHAVIOR:** Expansion is evidence-based.
- **RELEVANT FILES / LIKELY AREAS:** Ledger/ADR.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `10A` D-S6.
- **DEPENDENCIES:** GYST-VN-135 (Enable agent dispatch for gyst-lifeops (first supervised runs))
- **ACCEPTANCE CRITERIA:**
  - All eight criteria evidenced or the decision is no
- **TEST PLAN:** —
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** —
- **OUT OF SCOPE:** Implementation for SamVision.
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

## Phase 13 — Controlled System / n8n actions (M6)

<a id="gyst-vn-137"></a>
### GYST-VN-137 — Action runner (pull-only, fixed allow-list)

- **ID:** GYST-VN-137
- **NAME:** Action runner (pull-only, fixed allow-list)
- **PROJECT:** Server / host operations
- **REPOSITORY:** Per the packaging decision (GYST-VN-071); recommended: separate `gyst-ops` repository, not on the agent allowlist
- **TYPE:** Infrastructure
- **PRIORITY:** P3
- **PHASE:** 13 — Controlled System / n8n actions · Milestone M6
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** A separate runner with no listener that pulls `server:commands` from GYST and runs only allow-listed actions (restart n8n / samvision-web / gyst-lifeops, backup gyst / postgres).
- **CURRENT PROBLEM:** `09` §K.2: actions must not come from the provider or GYST directly.
- **DETAILED REQUIREMENTS:**
  - Root-owned allow-list file; unknown actions refused
  - Pulls via the internal machine listener with its own credential
  - Reports results; audit `server.action`
- **PROPOSED BEHAVIOR:** Safe remote actions become possible.
- **RELEVANT FILES / LIKELY AREAS:** Ops repository.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §K.2; gates G5, G7, G8.
- **DEPENDENCIES:** GYST-VN-092 (Deploy the Server Provider), GYST-VN-071 (Decide packaging and repository for Bridge, Server Provider and Action Runner), GYST-VN-081 (Machine credentials with scopes)
- **ACCEPTANCE CRITERIA:**
  - Refuses anything not on the allow-list
  - No inbound port
- **TEST PLAN:** Unit tests; supervised host test.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Stop the runner.
- **OUT OF SCOPE:** Deploys.
- **KNOWN RISKS:** The runner has Docker access; small code, strict review.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-138"></a>
### GYST-VN-138 — Deploy the action runner

- **ID:** GYST-VN-138
- **NAME:** Deploy the action runner
- **PROJECT:** Server / host operations
- **REPOSITORY:** Host
- **TYPE:** Operations
- **PRIORITY:** P3
- **PHASE:** 13 — Controlled System / n8n actions · Milestone M6
- **RISK CLASS:** NOT AGENT-ELIGIBLE
- **AGENT ELIGIBILITY:** NOT AGENT-ELIGIBLE operational work (Jay)
- **SUMMARY:** Install the runner on the host with its credential and allow-list.
- **CURRENT PROBLEM:** Host deployment is human-only.
- **DETAILED REQUIREMENTS:**
  - Backup first; install; test each allowed action once under supervision
- **PROPOSED BEHAVIOR:** Actions work end to end.
- **RELEVANT FILES / LIKELY AREAS:** Host.
- **DESIGN REFERENCES:** None (no UI change).
- **SECURITY REFERENCES:** `09` §K.2.
- **DEPENDENCIES:** GYST-VN-137 (Action runner (pull-only, fixed allow-list))
- **ACCEPTANCE CRITERIA:**
  - Each action tested and recorded
- **TEST PLAN:** Manual.
- **BROWSER / DEVICE VERIFICATION:** Not applicable (no UI change).
- **ROLLBACK CONSIDERATION:** Stop and remove.
- **OUT OF SCOPE:** —
- **KNOWN RISKS:** —
- **SUGGESTED IMPLEMENTER:** Jay
- **SUGGESTED REVIEWER:** Jay; an agent may review the recorded evidence for completeness
- **ESTIMATED COMPLEXITY:** S

<a id="gyst-vn-139"></a>
### GYST-VN-139 — System actions: run backup, restart allow-listed service

- **ID:** GYST-VN-139
- **NAME:** System actions: run backup, restart allow-listed service
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P3
- **PHASE:** 13 — Controlled System / n8n actions · Milestone M6
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Tailnet-only actions on the System page with step-up and in-page confirmation.
- **CURRENT PROBLEM:** Operations currently need SSH.
- **DETAILED REQUIREMENTS:**
  - Only on the privileged listener
  - Restart: step-up + confirm; backup: confirm
  - Result shown and audited
- **PROPOSED BEHAVIOR:** Jay can fix common problems from the tailnet.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/system.html`, `internal/handlers/` (privileged), `internal/services/system.go`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` §G.2 (TO + step-up + confirm).
- **DEPENDENCIES:** GYST-VN-138 (Deploy the action runner), GYST-VN-079 (Step-up re-authentication, action nonce and confirmation pattern), GYST-VN-074 (Bind the tailnet site to the Tailscale address only), GYST-VN-093 (System page: sanitized public view and tailnet detail)
- **ACCEPTANCE CRITERIA:**
  - Public listener 404
  - Negative tests for missing step-up/nonce
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Feature flag.
- **OUT OF SCOPE:** Deploy from GYST.
- **KNOWN RISKS:** Availability impact; confirm dialogs.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

<a id="gyst-vn-140"></a>
### GYST-VN-140 — n8n Run Now (tailnet, confirm)

- **ID:** GYST-VN-140
- **NAME:** n8n Run Now (tailnet, confirm)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P3
- **PHASE:** 13 — Controlled System / n8n actions · Milestone M6
- **RISK CLASS:** ELEVATED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Trigger approved GYST workflows via their app-specific webhook on the internal network; tailnet with confirmation by default, `public_runnable` only for safe workflows with re-auth.
- **CURRENT PROBLEM:** Separate from event status by decision.
- **DETAILED REQUIREMENTS:**
  - Allow-listed workflow webhook paths; Header-Auth credential per app
  - Audit `automation.run`
- **PROPOSED BEHAVIOR:** Jay can rerun an automation.
- **RELEVANT FILES / LIKELY AREAS:** `internal/services/automations.go`, `web/templates/automations.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` §G.2, §L; `08A` D-J6.
- **DEPENDENCIES:** GYST-VN-111 (Automations page (status only)), GYST-VN-079 (Step-up re-authentication, action nonce and confirmation pattern), GYST-VN-074 (Bind the tailnet site to the Tailscale address only)
- **ACCEPTANCE CRITERIA:**
  - Refused for unregistered workflows
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Feature flag.
- **OUT OF SCOPE:** Editing workflows (never).
- **KNOWN RISKS:** Low.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required) plus the other agent as security reviewer
- **ESTIMATED COMPLEXITY:** S

## Phase 14 — Logs (M6)

<a id="gyst-vn-141"></a>
### GYST-VN-141 — Bounded, redacted service logs (tailnet, step-up)

- **ID:** GYST-VN-141
- **NAME:** Bounded, redacted service logs (tailnet, step-up)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P3
- **PHASE:** 14 — Logs · Milestone M6
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Claude/Codex implementable
- **SUMMARY:** Show the last ≤ 200 lines of an allow-listed service's logs on the tailnet after step-up, redacted.
- **CURRENT PROBLEM:** Debugging needs logs; logs may contain secrets.
- **DETAILED REQUIREMENTS:**
  - Provider Level-3 logs endpoint enabled deliberately
  - Redaction of token/password/key patterns; bounded size
  - Audit `server.logs_viewed`
- **PROPOSED BEHAVIOR:** Jay can diagnose without SSH.
- **RELEVANT FILES / LIKELY AREAS:** Provider (ops repository), `web/templates/system.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `09` §G.2 (TO + step-up), §K.1.
- **DEPENDENCIES:** GYST-VN-092 (Deploy the Server Provider), GYST-VN-079 (Step-up re-authentication, action nonce and confirmation pattern), GYST-VN-074 (Bind the tailnet site to the Tailscale address only), GYST-VN-084 (Audit viewer and retention pruning)
- **ACCEPTANCE CRITERIA:**
  - Redaction tests
  - Public 404
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Disable Level 3.
- **OUT OF SCOPE:** Search/streaming.
- **KNOWN RISKS:** Secret leakage despite redaction; step-up and tailnet only.
- **SUGGESTED IMPLEMENTER:** Claude Code or Codex in an interactive, Jay-supervised session
- **SUGGESTED REVIEWER:** Jay (required, line by line) plus an independent agent security review; PR labelled `needs:security-review`
- **ESTIMATED COMPLEXITY:** M

## Phase 15 — Console link (much later) (M6)

<a id="gyst-vn-142"></a>
### GYST-VN-142 — Console link via Tailscale SSH (tailnet only)

- **ID:** GYST-VN-142
- **NAME:** Console link via Tailscale SSH (tailnet only)
- **PROJECT:** GYST LifeOps
- **REPOSITORY:** `JayDalool/gyst-lifeops`
- **TYPE:** Feature
- **PRIORITY:** P3
- **PHASE:** 15 — Console link (much later) · Milestone M6
- **RISK CLASS:** PRIVILEGED
- **AGENT ELIGIBILITY:** Human-assisted (agent prepares; Jay decides or executes)
- **SUMMARY:** Show a link to Tailscale SSH for the host on the tailnet only, after a separate review of Tailscale SSH ACLs.
- **CURRENT PROBLEM:** `08A` D-J8: the console is never proxied through GYST.
- **DETAILED REQUIREMENTS:**
  - Separate ACL review (human)
  - Link rendered only on the privileged listener
  - No terminal in GYST
- **PROPOSED BEHAVIOR:** A quick path to the console for Jay.
- **RELEVANT FILES / LIKELY AREAS:** `web/templates/system.html`.
- **DESIGN REFERENCES:** `11_DESIGN_SYSTEM_VNEXT.md`, `12_SCREEN_ARCHITECTURE.md`, `13_RESPONSIVE_INTERACTION_SPEC.md`, `13A_DESIGN_DECISIONS.md`
- **SECURITY REFERENCES:** `08A` D-J8; `09` §O (Console link).
- **DEPENDENCIES:** GYST-VN-074 (Bind the tailnet site to the Tailscale address only), GYST-VN-093 (System page: sanitized public view and tailnet detail)
- **ACCEPTANCE CRITERIA:**
  - Public listener never renders the link
  - Checked in a browser at phone (390×844) and desktop (1280×800) widths; no console errors or failed requests
  - Every interactive target ≥ 44 × 44 px (Galaxy scene targets 56 px)
  - Correct in light and dark mode; in System mode it follows an OS appearance change without reload (DD-1)
  - No horizontal page overflow at 320, 390, 768, 1024 and 1280 px
  - Progressive disclosure as specified in `12`: hidden-until-requested content is not shown by default
  - Icon-only controls have accessible names; focus is visible; heading order is correct
  - Violet (`--xp`, `--xp-soft`) is not used for any state, badge or severity (DD-4)
  - No spacing, radius, colour, navigation or component pattern outside `11`/`13A`; any conflict is reported in the task record, not invented
- **TEST PLAN:** `go test ./...` (template/handler render tests for the empty, one and many states), `node scripts/check-ui-shell.cjs` (updated only where this task deliberately changes the specified behaviour), `gofmt`/`go vet`.
- **BROWSER / DEVICE VERIFICATION:** Chrome at 390×844 and 1280×800 (and 1440 when an inspector is involved), light and dark, keyboard pass, console and network clean. For a major mobile screen, verify on a real iPhone in Safari and in the installed PWA where practical; record the result (not private screenshots) in the task record.
- **ROLLBACK CONSIDERATION:** Remove the link.
- **OUT OF SCOPE:** Web terminal.
- **KNOWN RISKS:** ACL mistakes; human review.
- **SUGGESTED IMPLEMENTER:** Jay (ACL review); an agent adds the link
- **SUGGESTED REVIEWER:** Jay
- **ESTIMATED COMPLEXITY:** S

---

## Backlog statistics

**Total tasks: 145**

| Priority | Tasks |
|---|---|
| P0 | 5 |
| P1 | 52 |
| P2 | 73 |
| P3 | 15 |
| **Total** | **145** |

| Risk class | Tasks |
|---|---|
| NORMAL | 68 |
| ELEVATED | 33 |
| PRIVILEGED | 18 |
| NOT AGENT-ELIGIBLE | 26 |
| **Total** | **145** |

| Type | Tasks |
|---|---|
| Bug | 4 |
| Feature | 33 |
| UI | 39 |
| UX | 4 |
| Refactor | 3 |
| Security | 18 |
| Infrastructure | 9 |
| Integration | 6 |
| Operations | 19 |
| Documentation | 3 |
| Research | 7 |
| **Total** | **145** |

| Agent eligibility | Tasks |
|---|---|
| Claude/Codex implementable | 116 |
| Human-assisted (agent prepares; Jay decides or executes) | 11 |
| Human-only (Jay) | 5 |
| NOT AGENT-ELIGIBLE operational work (Jay) | 13 |
| **Total** | **145** |

| Complexity | Tasks |
|---|---|
| S | 89 |
| M | 52 |
| L | 4 |
| **Total** | **145** |

| Milestone | Phases | Tasks | P0/P1/P2/P3 | User-visible outcome |
|---|---|---|---|---|
| M0 Safe foundation | 0 | 17 | 5 / 11 / 1 / 0 | GYST data is backed up off-host and restorable, the public login resists guessing, "today" means Jay's local day everywhere, and failures are visible in logs and /healthz. |
| M1 vNext core experience | 1, 2 | 20 | 0 / 17 / 3 / 0 | The app looks and navigates like vNext: Today · Projects · Health · Learn · More, a calm Today with Now, Rhythm and no duplicate blocks, a simpler More, and System/Light/Dark appearance. |
| M2 vNext Life areas | 3 | 27 | 0 / 10 / 13 / 4 | Projects, Galaxy, Learn, Health and Goals use the vNext patterns and plain terminology; Health shows only what really works; Galaxy is usable on a phone. |
| M3 Command centre | 4, 5 | 9 | 0 / 5 / 3 / 1 | One launcher (Go · Do · Capture) reaches everything; Apps opens external tools; one Attention model tells Jay what needs him, from GYST's own data. |
| M4 Read-only integrations | 6, 7, 8, 9 | 41 | 0 / 9 / 30 / 2 | GYST shows sanitized server health, GitHub issues/PRs/reviews in Development, and n8n automation status, all read-only, on a hardened two-listener foundation. |
| M5 Agent-assisted development | 10, 11, 12 | 25 | 0 / 0 / 23 / 2 | Jay can write GitHub issues from GYST, mark a NORMAL issue ready for an agent after re-auth, watch the run, review the PR in GitHub and verify-and-close from GYST. |
| M6 Controlled operations | 13, 14, 15 | 6 | 0 / 0 / 0 / 6 | From the tailnet, with re-auth and confirmation, Jay can run a backup, restart an allow-listed service, run an approved n8n workflow and read bounded redacted logs. |

