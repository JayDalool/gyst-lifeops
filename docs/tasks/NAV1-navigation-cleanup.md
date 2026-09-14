# NAV1 — Navigation cleanup

- **Status:** IN PROGRESS.
- **Approval:** Jay approved the proposed cleanup after UI-10.
- **Commit / CI:** Pending.

## Scope and acceptance

- Top bar contains GYST and Search; global Add dropdown removed.
- XP remains on Today. Today has a labelled Add task action.
- Desktop >=1024 px uses one left sidebar. Phone/tablet use five bottom links.
- Today, Projects, Health, Courses and Menu preserve their existing routes.
- Menu at `/more` groups 19 existing destinations into five purpose-based sections.
- New/edit actions stay on owner pages; search/quick capture and keyboard access remain.
- Galaxy retains its scene, and clearance calculations account for sidebar geometry.
- No new routes to unavailable features; no production or database changes.

## Verification

Local gofmt, vet, build, unit and race checks passed. Node syntax, shell/cache and Galaxy gesture checks passed. Chromium and WebKit passed at 320/390/768/844/1024/1440 px, including 200% text, all 19 Menu destination responses, sidebar/bottom-tab geometry, Galaxy canvas clearance, search filtering, Escape/focus return and dark mode. Desktop Menu and phone Today screenshots were visually inspected. Browser checks used only a temporary fixture database; no production deployment. Physical iPhone review remains pending. Existing shell test assumptions about globally repeated
creation links were replaced with owner-page action checks. Menu reachability
coverage now includes Health children, checklist and the main sections. Browser checks found a conflicting Galaxy minimum height in landscape and a legacy CSS rule displaying hidden search results; both were corrected.

## Next action

Finish verification and CI, then Jay reviews navigation before UI-11.
