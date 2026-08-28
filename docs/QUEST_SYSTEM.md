# Quest System

## Domain model

```text
Campaign / Goal
└── Quest / main project
    └── Chapter / phase or subproject
        └── Mission / project item or standalone task
```

Projects store hierarchy, status, dependency, automatic-unlock preference, and
locked notes. Project items are project-scoped missions. Standalone tasks carry
XP and proof fields and feed the daily recurrence system.

## Project status flow

The canonical quest statuses are `active`, `upcoming`, `locked`, `done`, and
`archived`. Legacy `paused` and `completed` values remain accepted by the schema
for compatibility and are normalized by repository writes.

```mermaid
stateDiagram-v2
    locked --> upcoming: dependency completed and auto-unlock enabled
    locked --> active: manual unlock
    upcoming --> active: manual start
    active --> done: manual completion or all project items complete
    done --> active: a project item is reopened
    locked --> archived
    upcoming --> archived
    active --> archived
```

Completing a project calls `AutoUnlockDependents`. Only locked dependents with
`auto_unlock = 1` move to `upcoming`; opted-out and manually changed projects
are left untouched.

## Rewards and proof

One-time standalone tasks store their full award in `tasks.stars_awarded` when
first completed. The base XP comes from `stars_reward`; valid proof adds two XP.
Required proof blocks completion until a matching location is supplied:

- `link` requires an absolute HTTP or HTTPS URL in `proof_url`.
- `photo` and `file` require a non-empty path in `proof_path`.
- `text` requires non-empty text in `proof_path`.

Generated proof and quest-completion achievements carry their source task or
project ID. Unique partial indexes added by migration `009` prevent retries,
reopens, or repeated completion requests from creating duplicate achievements
or bonus XP.

Project completion awards ten XP once when all of its project items become
complete. Reopening and recompleting the project does not award it again.

## Markdown import

The Markdown parser maps:

- level-one headings to a campaign goal;
- level-two headings to main quests;
- level-three headings to chapters/phases;
- checklist entries to project items;
- status, notes, dependency, auto-unlock, locked notes, privacy, stars, and proof
  labels to the import plan.

Relationships are resolved after every imported project has an ID, so `Depends
On` can name a project that appears earlier or later in the document.

## Integration boundaries

- Roadmap, Rocket Route, and Cosmic views read the same project hierarchy and
  canonical project statuses; they do not maintain a second quest state.
- Course and learning-block statuses are a separate scheduling domain. Course
  completion does not award quest XP.
- Roadmap milestones are planning records, while project items are executable
  quest missions that reconcile project completion and rewards.

## Verification

Run the repository checks with an installed Go toolchain:

```sh
gofmt -l $(find . -name '*.go' -type f)
go test ./...
go vet ./...
```

Migration tests cover both a fresh database and a populated pre-quest database
upgrading through the latest migration.

## Known product gaps

- Stars and proof parsed from Markdown checklist items are preserved in the
  item description, but project items do not yet have structured reward/proof
  columns. Standalone tasks do. Unifying these two mission representations is
  the next architectural decision.
- Move-under-parent and locked-note editing are available through the project
  edit form, but dedicated contextual actions would make them easier to find.
- Required-proof completion currently returns the normal server error response;
  an inline validation message would improve the browser workflow.
- A visual browser pass is still required for the final dashboard, project tree,
  and campaign-map polish.
