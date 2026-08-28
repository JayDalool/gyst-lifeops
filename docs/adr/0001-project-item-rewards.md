# ADR 0001: Structured rewards on project items

## Context

LifeOps has two executable work records: standalone recurring tasks and
project-scoped mission items. Markdown imports already parse XP and proof
metadata for missions, but project items previously stored that metadata inside
their description. It could not be validated or awarded.

## Decision

Add structured reward and proof columns directly to `project_items`, mirroring
the established standalone-task fields. Project-item completion owns mission XP
and proof awards, then invokes the existing project reconciliation that can
complete the quest and unlock dependencies.

Generated mission-proof achievements retain `source_project_item_id`; a unique
index makes retries idempotent.

## Alternatives considered

### Replace tasks and project items with one mission table

This would remove duplication, but recurring tasks and project checklist items
currently have different lifecycle, scheduling, and foreign-key behavior. A
replacement would broaden this milestone into a risky migration and handler
rewrite.

### Store reward metadata in descriptions

This preserves the schema but cannot validate proof, calculate player totals,
or prevent duplicate awards reliably.

## Why this option was selected

The additive migration is backward-safe, uses existing proof conventions, and
completes the Markdown-to-reward workflow without changing Course, Roadmap,
Rocket Route, or Cosmic ownership boundaries.

## Consequences

- Standalone tasks and project items temporarily have parallel reward fields.
- Player totals aggregate awards from both tables.
- Project items gain a proof-submission action in the project UI.
- Locked and upcoming projects cannot complete missions until manually or
  automatically activated.

## Future considerations

If recurring project missions or cross-project scheduling are added, reassess a
shared mission interface or table. Do not merge the records solely to remove
field duplication.
