// Package secretary is the Secretary's read-only tool catalogue (Issue #236,
// CF-4). It decides what the Secretary may read, how much of it, and how fresh;
// it never decides what to say, and it calls no AI provider.
//
// # Rules
//
//   - Read-only. Every tool reads stored rows. The one side effect is a Hevy
//     refresh through services.SyncTraining, started by EnsureFresh.
//   - Explicit allowlist. A tool exists only if its name is in the allowlist in
//     catalogue.go; New rejects any other name, a duplicate, an empty
//     description, a nil Run and an unknown freshness source.
//   - No AI call, no route, no environment. This package imports services,
//     models, clock, healthcoach (only the Tool type) and the standard library.
//     It does not import handlers, net/http, os or os/exec (noprovider_test.go
//     checks this).
//   - Time comes from the application clock (Service.Clock), never time.Now.
//   - Errors the model can see are fixed strings that never repeat the input.
//   - Free text written by someone else (an event title, a mail subject, a
//     name) leaves this package inside an Envelope after Clean. Task, project,
//     course and workout titles are Jay's own and are cleaned and capped.
//   - Logs hold the source, the outcome and the duration. Never a title, a name
//     or a tool result.
//
// # GET versus POST (Issue #188)
//
// BuildDayBrief is pure: it reads stored rows and the freshness state with
// Peek, and never syncs or writes, so a GET may call it. Catalogue.Tools is for
// POST paths only: each tool it returns first runs EnsureFresh for its Sources,
// which can start a sync. Build one Catalogue per turn; a source is checked at
// most once per Catalogue.
//
// # The brief and project privacy
//
// list_projects carries each project's effective privacy (public, private or
// local_only, S15 round 7). The day brief holds no project-derived item yet:
// tasks have no project or privacy column. When a later chunk adds project items
// to the brief, each must carry its privacy the same way.
//
// # Adding a tool (calendar and mail chunks)
//
//  1. Write a builder, func(*Catalogue) []ToolSpec, and add it to builders in
//     catalogue.go. A spec that reads a synced source lists it in Sources; its
//     Run returns a value (encoded as compact JSON) or a fixed-string error.
//  2. Add the tool's name to allowlist, in offer order, and to the literal in
//     TestRegistryEqualsAllowlist. This is the reviewed step.
//  3. Pass the data source in Deps (for example Deps.Calendar, a
//     CalendarReader) and, for a source that syncs, a Syncer in Deps.Syncers
//     under "calendar" or "mail". Sync must bound itself (its own timeout and
//     lock) and must not rely on the caller's context: it keeps running after
//     the time budget, so a timeout never becomes a false failure.
//  4. Test it with secretarytest: a migrated database, a pinned clock, fictional
//     rows, FakeHealth, FakeCalendar, SlowHevy and Run. The Example shows the
//     wiring; TestBuilderHook proves the steps above.
//
// Untrusted text must go through Wrap (source "email" or "calendar") with the
// exported caps TitleCap, SubjectCap, NameCap or DescriptionCap.
package secretary
