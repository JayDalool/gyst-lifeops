# ADR 0002: The Secretary reads a synced store

**Status:** Accepted, 2026-11-18, by Jay (product owner). The decisions behind
it are S1, C5, M8 and C18 in
[`19A_CONNECTED_DATA_DECISIONS.md`](../audit/vnext/19A_CONNECTED_DATA_DECISIONS.md).

## Context

GYST is going to connect the owner's Google and Microsoft calendars and his
mail accounts (19A §3 to §8). Three parts need that data at once: the Agenda
screen and Today, the free-time engine ("When am I free on Friday?"), and the
Secretary, an AI assistant that answers questions and proposes changes.

GYST already keeps outside data in its own database instead of calling the
source on every page: Hevy workouts are cached by a sync loop
(`migrations/20260924010000_training_cache.sql`), and Health arrives from the
Shortcut. The AI provider key is already encrypted in the database with a data
key in a separate file (`internal/services/ai_settings.go`, `loadDataKey` and
`ensureDataKey`).

Calendar and mail data differ from what GYST stores today in two ways:

- **It is other people's text.** Meeting titles, guest names and email bodies
  are written by third parties. Some of it is hostile (prompt injection), and
  it should not spread into every backup copy.
- **The provider owns it.** Events and mail change at Google or Microsoft as
  well as in GYST, and since the 2026-11-18 decisions GYST also writes to them
  (C1, C11, M1).

The security foundation says "no new databases"
([`09_SECURITY_FOUNDATION.md`](../audit/vnext/09_SECURITY_FOUNDATION.md)
§B.10). This decision needs one more database file, so it conflicts with that
principle (see [Exception](#exception-to-09-b10)).

## Decision

1. **Synced, not live.** Background sync loops pull calendar and mail data into
   GYST, using each account's refresh token. The screens and the Secretary read
   the same stored rows, so the Agenda and the Secretary's answer can't
   disagree.
2. **A second file, not backed up.** Provider data (event and mail caches, sync
   cursors) lives in `data/sync.db`, a second SQLite file opened with the same
   driver and settings as the main database and migrated by the same runner
   (`internal/db/db.go`, `Open` and `Migrate`) from its own migrations
   directory. It is **left out of backups**: `lifeops backup` copies only the
   main database (`VACUUM INTO`, `internal/db/backup.go`). The owner's own
   choices stay in the main database, which is backed up: connected accounts
   and their encrypted secrets, calendar choices, house rules, notification
   preferences, task links and pinned files.
3. **The AI tool layer holds no provider token.** The Secretary reads GYST's
   rows through typed, read-only lookups. Only the sync service holds tokens.
   The few live provider calls ("Sync now", a bounded refresh of a stale copy,
   an attachment download, the rest of a long email) are made by the sync
   service, never by the model.
4. **Secrets.** OAuth client IDs and secrets come from the environment file.
   Refresh tokens (and IMAP app passwords) are encrypted in the main database
   with the existing AI-key pattern. Access tokens stay in memory. No refresh
   token is ever kept in an environment variable or the repository.
5. **The provider stays the source of truth.** GYST writes a change to the
   provider first, stores the provider's reply, then re-syncs that calendar or
   folder. Edits, moves and deletions made at the provider reach GYST through
   the next sync. A conflicting change is shown to the owner, never overwritten
   silently, and nothing is queued while the provider can't be reached.

## Alternatives considered

- **A live provider call per question:** slow, fails whenever one provider does, and puts tokens on the AI path.
- **Giving the model provider tools or tokens:** broad mailbox access for the model, injected text could reach the provider, and free time would not be deterministic.
- **Sending everything in every prompt:** costly, and far more personal data leaves GYST than each answer needs.
- **Keeping the caches in the main database:** every backup would carry other people's mail and meeting text and grow with mail history.
- **A database server (for example PostgreSQL):** a second engine and service for a single-user app, against the one-binary, one-engine design.
- **Refresh tokens in environment variables:** Microsoft replaces the refresh token on every use, so GYST must write it, and a leaked environment file would expose every account.
- **GYST edits its own copy first and pushes later:** two sources of truth, and stale changes replayed after an outage.

## Why this option was selected

It reuses what GYST already does for Hevy and the AI key, keeps one binary and
one database engine, and puts a hard boundary between the AI and the provider
accounts: a manipulated model can reach only what GYST chose to store. Leaving
the cache out of backups keeps third-party text out of backup copies, and the
data can always be downloaded again from the provider.

## Consequences

- **Restore means re-sync.** Restoring a backup brings back accounts, choices
  and rules; `data/sync.db` is rebuilt by a full sync. For mail with all
  history (M7, M24) that can take hours.
- **Start-up opens two files.** If `data/sync.db` is missing or broken, GYST
  recreates it and runs a full sync. A test proves the main backup never
  includes it.
- **No foreign keys across files.** Rows in `data/sync.db` refer to main-database
  ids by value. Removing an account or calendar deletes its rows from both
  files together (M13).
- **Anything that must survive a restore goes in the main database.** Pinned
  attachments (M23) and food profile photos (F2) are stored there for that
  reason.
- **Answers can be stale.** The freshness rule (S2) syncs a stale source
  before answering and otherwise says "as of" the time.
- **File permissions.** `data/sync.db` and the attachment cache are files with
  third-party data: mode `0600` inside `data/` at `0700` (09 §N.11), and in
  scope of the file-permission hardening that
  [`10A_SECURITY_DECISIONS.md`](../audit/vnext/10A_SECURITY_DECISIONS.md) O-1
  requires before any agent account exists.
- **Audit and logs** carry account and source ids and counts, never titles,
  addresses or text (09 §B.8). Event times are stored as UTC instants
  (09 §B.9).
- **Writes are only as good as the provider's reply.** GYST's copy changes only
  after the provider accepts a write, so a failed write leaves GYST unchanged.
- **Disk use grows** with mail history. The mail plan's disk guard pauses the
  history download when free space runs low.

## Exception to 09 §B.10

`09_SECURITY_FOUNDATION.md` §B.10 says: "Small enough to finish. No enterprise
IAM, no brokers, no new databases." `data/sync.db` is a new database file, so
this decision conflicts with that principle.

**The owner accepted this exception on 2026-11-18** (19A M8, C5 and S1). It is
narrow: the same engine (SQLite), driver and migration runner, no server, no
new technology. Any further database needs its own decision.

`10A_SECURITY_DECISIONS.md` overrides `09` where they differ, and its O-1 makes
file-permission hardening of "databases" a binding prerequisite. Neither file
is edited here. A later change brings them in line:

- **09 §B.10:** note the approved exception for `data/sync.db`, with a link to
  this ADR.
- **09 §N.11:** name `data/sync.db` and the attachment cache in the
  file-permission list (`0600` files inside `data/` at `0700`).
- **09 §N.1 (backups):** say that `data/sync.db` and the attachment cache are
  deliberately not backed up, and that a restore is followed by a re-sync.
- **10A:** record the §B.10 exception as an owner decision, so the precedence
  chain (10A over 09) holds.
- **10A O-1:** list `data/sync.db` and the attachment cache among the
  sensitive files its recorded verification checks.

## Future considerations

- Another AI provider (D7) changes only the adapter; the boundary in point 3
  stays.
- If mail history makes `data/sync.db` too large, reassess retention (M24)
  before reaching for another store.
- If push sync from the providers is added later, it feeds the same store; the
  Secretary still reads only GYST's rows.
