# UI-3 — Projects and project detail

- **Status:** `IN PROGRESS` — implementation and local verification complete; commit/CI pending.
- **Approval:** Jay reviewed UI-2 and requested continuation to the next phase.
- **Implementation commit(s):** Pending.
- **CI:** Pending push.
- **Review:** Agent verification; physical iPhone/Home Screen review pending.

## Scope

Projects adopts shared cards, SVG icons, real progress/counts/status labels,
roadmap filters and All/Active/Completed/Upcoming status links. All is the
preserving default. Active includes paused/blocked states with actual labels;
Upcoming includes locked projects. A matching child keeps its parent visible
as context, without changing status or flattening the hierarchy. Status and
roadmap filters retain each other. Invalid status falls back to All.

The signature Galaxy stays one link away. The existing route map is retained
in a labelled disclosure. Project detail has Overview/Tasks/Roadmap/Notes anchor
navigation, real roadmap/milestone access, shared stop cards and explicit empty
notes/roadmap states. Project create/edit forms adopt the same visual scope.
All existing proof, external-link, status, archive, item completion/reopen/delete,
dependency, roadmap-feature and mission-creation actions remain.

No service calculations, schemas, unlock rules or integrations change. The
existing Projects GET orphan-adoption behavior is unchanged; all runtime checks
use a temporary fixture database, never production.

## Verification

Local gofmt, go vet, go build, go test and go test -race passed on 2026-09-12.
The Node UI shell cache/theme check and migration-prefix uniqueness check passed;
no database files are tracked. Existing handler/render/group tests pass. New tests
cover hierarchy-preserving filters, actual status rather than inferred progress,
query preservation, detail action retention and unavailable project item states.
The shared worker asset list/version is updated with the page stylesheet.

Chromium and WebKit passed at 320, 375, 390, 430, 768 and 1280 px with no
horizontal page overflow or JavaScript errors. Browser interactions covered
project creation, mission creation/completion/reopening, section anchors and
Galaxy navigation. Dark theme and 200% text checks passed. Mobile list and
desktop detail screenshots were visually inspected. All browser writes used
`/tmp/gyst-ui3-review/preview.db`; no production access or deployment.
Physical iPhone/Home Screen review remains pending.

## Next action

Jay: review UI-3 before approving UI-4 Mobile Galaxy.
