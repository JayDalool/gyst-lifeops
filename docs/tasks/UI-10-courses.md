# UI-10 — Courses

- **Status:** IN PROGRESS.
- **Approval:** Jay requested the next phase after UI-9.
- **Implementation commit / CI:** Pending.

## Scope and acceptance

- List cards show actual provider, instructor, status, progress, blocks, next deadline and target date.
- Detail uses Overview/Learning blocks/Notes/Add block anchor navigation.
- Create/edit forms and block controls adopt shared tokens and responsive sizing.
- Existing course status/archive, block complete/reopen/postpone/delete, notes, URLs, dates, times and repeat controls remain.
- Calendar export gating is preserved. No service/schema or production changes.
- Catch-up previews use distinct IDs; canonical block action anchors are preserved.
- No invented covers, locked lessons, completion percentages or integrations.

## Verification

Local format, vet, build, unit and race checks passed, plus Node shell/cache and Galaxy checks. Chromium and WebKit passed course create/edit/pause/resume/complete/archive, block complete/reopen/postpone/delete, catch-up anchor uniqueness, notes and field persistence, navigation, dark mode and 200% text at 320/390/430/768/844/1440 px across list/detail/create/edit pages. Phone detail and desktop list screenshots were visually inspected. WebKit exposed native-select overflow at enlarged text; scoped select appearance fixed it. A rerun initially matched an earlier fixture with the same title after archive; checking the specific course URL resolved the test ambiguity. The full rerun passed. All browser writes used a temporary loopback fixture database with auth disabled only there. Physical iPhone/software-keyboard review remains pending.

## Next action

Finish local/browser/CI verification; then Jay reviews before UI-11.
