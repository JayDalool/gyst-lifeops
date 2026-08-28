#!/usr/bin/env bash
# Launch Codex to build one phase of docs/BUILD_PLAN.md.
#
# Usage:
#   ./scripts/codex-build.sh 1          # build phase 1 interactively
#   ./scripts/codex-build.sh 2          # build phase 2
#   ./scripts/codex-build.sh 1 --exec   # run headless, no TUI
#
# Codex runs with full autonomy inside this repo: workspace-write sandbox and
# no approval prompts. It cannot write outside the repo directory.
set -euo pipefail

cd "$(dirname "$0")/.."

PHASE="${1:?usage: $0 <phase-number> [--exec]}"
MODE="${2:-}"

PROMPT="Read docs/BUILD_PLAN.md in full before doing anything else.

Implement Phase ${PHASE} of that plan, and only Phase ${PHASE}. Do not start
later phases.

Section 0 of the plan is your mandate: you have full authority in this repo and
should not stop to ask permission for ordinary implementation decisions. Make
the call, note it in the commit message, and keep going. Respect the four hard
guardrails in that section.

Before you finish:
  1. Run ./scripts/check.sh and make it pass. There is no Go toolchain on this
     host, so do not run 'go test' directly - it will fail. The script runs Go
     inside Docker.
  2. Rebuild and verify in the browser with 'docker compose up -d --build'.
     The app serves on http://localhost:3005.
  3. Commit the phase as a single commit.

When you are done, report: what you changed, any decision you made that the plan
did not cover, and anything in the plan you found to be wrong about the actual
codebase."

if [ "$MODE" = "--exec" ]; then
  exec codex exec --sandbox workspace-write "$PROMPT"
fi

exec codex --sandbox workspace-write --ask-for-approval never "$PROMPT"
