# Connected data decisions: accounts, calendar, notifications, mail, the Secretary and food

- **Decided by:** Jay (product owner), for every entry marked Decided or Dropped
- **Recorded:** 2026-11-18 (Issue #233, CF-1)
- **Responds to:** the owner's calendar, mail, Secretary and build-order plans of
  2026-11-18 (round 7). They are kept on the server, outside this repository;
  see [Sources](#11-sources).
- **Status:** Decided and Dropped entries are **binding** for later work. To
  confirm and Open entries are recommendations only: they bind nothing until
  Jay accepts them, and no Issue may build on them as if they were decided.
- **Precedence:** where this file differs from the server plans, this file
  wins. `05A`, `08A`, `10A`, `13A` and `17A` stay binding. The one known
  conflict with them, a second database file against "no new databases" in
  [`09_SECURITY_FOUNDATION.md`](09_SECURITY_FOUNDATION.md) §B.10, is an
  exception the owner accepted on 2026-11-18. It is recorded in
  [ADR 0002](../../adr/0002-secretary-reads-synced-store.md), and `09` and
  [`10A`](10A_SECURITY_DECISIONS.md) are brought in line in a later change.
- **Implementation:** none. Nothing here is built, configured or credentialed.
- **Privacy:** this file records decisions only. It holds no account names,
  addresses, regions, counts or personal details. Every example is generic.

---

## 1. How to read an entry

Each area has one table. Every decision ID appears exactly once, in the first
column.

| Column | Meaning |
|---|---|
| **ID** | The decision's ID in the plans. A plan row that held several IDs (for example `C6 / C7 / C8 / C9`) is split into one entry per ID. |
| **Question** | What was being decided. |
| **Status** | One of four words. **Decided**: the owner decided it. **To confirm**: there is a recommendation and the owner's yes is still needed. **Open**: not asked yet. **Dropped**: no longer applies. |
| **Decision or recommendation** | For Decided, the decision in plain words. For To confirm and Open, the recommendation, starting "Recommended:". A Decided entry that still has one part waiting says so after "Still to confirm:"; that part is not decided. |
| **Alternative** | For To confirm and Open, the main other option. For Decided, the option it replaced or rejected, or "—". "(inferred)" marks an alternative the plans don't name. |
| **Decider, date** | "Jay, 2026-11-18" for Decided and Dropped; "Jay (pending)" for To confirm and Open, with the date of the recommendation. |
| **Source** | What the entry rests on. "Answer *n*" is the owner's numbered answer to the 36 open questions of 2026-11-18. "Earlier" and "later" are his messages before and after those answers on the same day. "Round 7" is his set of design and Secretary choices later on 2026-11-18; this record holds the three that belong here (S15, S21 with D9, and F9). Plan sections are listed in [Sources](#11-sources). |

**Answer 29.** Answer 29 was one yes to a group of twenty items that the lead described to the owner as engineering details (sync timings, minimum permissions, storage tables, token handling). The lead who asked the question listed the twenty as M2, M11, M12, M13, M17, C2, C3, C4, C12, C13, C15, N6, S2, S8, S9, S12, S17, D4, D5 and D10. D4 and D5 were later dropped, which leaves eighteen Decided entries that rest on answer 29 (C2 and C3 also rest on answer 8). Three of them shape what the owner sees rather than how GYST works inside: C13, C15 and S9. They stay Decided, are marked "accepted in the answer-29 group", and are shown to him for a second look.

## 2. Summary

| Area | Decided | To confirm | Open | Dropped | Total |
|---|---|---|---|---|---|
| [3. Accounts and sign-in](#3-accounts-and-sign-in) | 5 | 0 | 0 | 0 | 5 |
| [4. Storage and sync](#4-storage-and-sync) | 5 | 0 | 0 | 0 | 5 |
| [5. Calendar](#5-calendar) | 16 | 3 | 0 | 0 | 19 |
| [6. Notifications](#6-notifications) | 8 | 0 | 0 | 0 | 8 |
| [7. Secretary and AI](#7-secretary-and-ai) | 32 | 1 | 1 | 4 | 38 |
| [8. Mail](#8-mail) | 14 | 4 | 0 | 0 | 18 |
| [9. Food](#9-food) | 9 | 2 | 0 | 0 | 11 |
| **All** | **89** | **10** | **1** | **4** | **104** |

**Waiting for Jay** (none of these is decided):

- To confirm: C17, C20, C21, M9, M21, M23, M24, S20, F6, R1.
- Open: S26.
- Parts of Decided entries: the undo and hold durations in C19, S10 and
  S23, the M22 send limits, the S23 bulk-delete rule, the meaning of "all
  history" in M7 (= M24), how the
  day's context reaches the AI in P1 (= F6), and F4's rule change (= R1).
  S28 has a design still to do, and S19 and D3 wait on the VX-1 phone test.
- A second look: C13, C15 and S9, accepted in the answer-29 group.

---

## 3. Accounts and sign-in

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| M2 | How each kind of inbox connects | Decided | Gmail through the Gmail API and Microsoft through Microsoft Graph, each with the provider's own sign-in; Yahoo and Zoho through IMAP with an app password. Sending is added by M20. | IMAP with XOAUTH2 for Gmail (needs full mail access) | Jay, 2026-11-18 | Answer 29; mail plan §2, §14 |
| M11 | New Go libraries | Decided | `golang.org/x/oauth2` now. The IMAP, MIME and SMTP libraries only when their chunks start, each probed before use (`docs/RULES.md` §1.5). Which mail libraries is M21. | Hand-written OAuth on `net/http` (inferred) | Jay, 2026-11-18 | Answer 29; mail plan §4, §14, §20 |
| M16 | Google app status | Decided | One Google Cloud app in Personal mode per Google account: no verification, no unverified-app screen, no seven-day token expiry. | External, in production, unverified (only if Personal mode can't give a web client) | Jay, 2026-11-18 | Answer 6; mail plan §14 |
| M17 | If Google mail access fails | Decided | Gmail falls back to IMAP with an app password; calendars ask to reconnect. | Reconnect only, no fallback (inferred) | Jay, 2026-11-18 | Answer 29; mail plan §14 |
| M18 | One Google grant or two | Decided | Two grants per Google account, calendar and mail, each with its own encrypted refresh token, so a mail revocation never ends the calendar. | One combined grant | Jay, 2026-11-18 | Answer 6; mail plan §14 |

**Secrets layout** (the design these entries and ADR 0002 rely on; mail plan §5):

- OAuth **client IDs and secrets** come from the environment file, set by Jay.
  They are never stored in the database or the repository.
- Each account's **refresh token**, or its IMAP app password, is encrypted in
  the main database (AES-256-GCM, the same pattern as the AI key, with the
  data key in a `0600` file beside the database). Refresh tokens are never
  kept in environment variables or the repository.
- Access tokens stay in memory. Connecting, reconnecting and removing an
  account need the GYST password again.

## 4. Storage and sync

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| M8 | Where synced mail is kept | Decided | In a second SQLite file, `data/sync.db`, which is not backed up. Accounts, encrypted secrets, task links, calendar choices and house rules stay in the main database. An exception to "no new databases" (`09` §B.10), accepted by the owner; see ADR 0002. | Everything in the main database | Jay, 2026-11-18 | Answer 7; mail plan §6, §14 |
| C5 | Where calendar events are kept | Decided | In `data/sync.db`, not backed up. Calendar choices and house rules stay in the main database. | Everything in the main database | Jay, 2026-11-18 | Answer 7; calendar plan §5, §13 |
| S1 | Where the Secretary reads from | Decided | GYST's synced copy, through typed read-only lookups. Live provider calls only for "Sync now", a bounded refresh of a stale copy, an attachment download and the unstored rest of a long email, all made by the sync service. | A live provider call per question; provider tools handed to the model | Jay, 2026-11-18 | Answer 7; Secretary brain §0, §2, §11 |
| C18 | Keeping GYST aligned with the provider | Decided | The provider is the source of truth. GYST writes to the provider first, stores the provider's reply, then re-syncs. Edits, moves and deletions made at the provider always reach GYST. A conflicting change is detected and shown, never overwritten silently. | — | Jay, 2026-11-18 | Answer 7; calendar plan §16.2 |
| M26 | Mail sync timing | Decided | Inbox and Sent every 3 minutes per account, with jitter; other folders every 30 minutes. | Every 5 minutes (the earlier recommendation) | Jay, 2026-11-18 | Answer 17; mail plan §16.5 |

## 5. Calendar

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| C1 | Scope of the first calendar release | Decided | Read and write. The owner picks which calendars to show; new calendars are off by default; events can be created, edited and deleted from GYST. | Read-only first (the earlier recommendation) | Jay, 2026-11-18 | Answer 8; calendar plan §13, §16 |
| C2 | Google calendar permission | Decided | `calendar.events` plus `calendar.calendarlist.readonly`. | The full `calendar` scope (broader) | Jay, 2026-11-18 | Answers 8, 29; calendar plan §16.1 |
| C3 | Microsoft calendar permission | Decided | `Calendars.ReadWrite` plus `offline_access`. | `Calendars.Read` (read-only, replaced) | Jay, 2026-11-18 | Answers 8, 29; calendar plan §16.1 |
| C4 | Sync window and timing | Decided | 14 days back to 120 days ahead, every 15 minutes, with a nightly window roll. | 30 days back to 365 days ahead | Jay, 2026-11-18 | Answer 29; calendar plan §4.1, §13 |
| C6 | All-day events and free time | Decided | An all-day event blocks free time only when the provider marks it busy; otherwise it shows as a note. | Always block; never block | Jay, 2026-11-18 | Answer 9; calendar plan §13 |
| C7 | Tentative events | Decided | Count as busy. | Count as free | Jay, 2026-11-18 | Answer 9; calendar plan §13 |
| C8 | Timed GYST tasks and free time | Decided | A timed task blocks 30 minutes from its time (an editable house rule; 0 turns it off). | Never block | Jay, 2026-11-18 | Answer 9; calendar plan §13 |
| C9 | Prayer times and free time | Decided | Each prayer blocks 15 minutes (editable; 0 turns it off). | Don't block | Jay, 2026-11-18 | Answer 9; calendar plan §13 |
| C10 | Where events appear | Decided | Inside Today's existing Now and Later rows with no layout change, on a new Agenda screen, and in an Overview band built with makeover chunk 6A. See [§12](#12-calendar-ui-record). | Agenda only | Jay, 2026-11-18 | Answer 2 (Today rows); calendar plan §8 (Agenda, Overview band) |
| C11 | Creating events | Decided | In the first calendar release, not later. | A later, separate decision | Jay, 2026-11-18 | Answer 8; calendar plan §16 |
| C12 | GYST's own calendar feed | Decided | Provider events are never re-exported into GYST's `.ics` feeds. A calendar that is GYST's own feed is detected and left off. | Re-export everything | Jay, 2026-11-18 | Answer 29; calendar plan §3, §9 |
| C13 | Time zone shown | Decided | Always GYST's application time zone. Accepted in the answer-29 group; shown to the owner for a second look. | Also show the event's own zone | Jay, 2026-11-18 | Answer 29; calendar plan §13 |
| C14 | Employer calendars | Decided | None of the connected calendars belongs to an employer. Generic rule kept: an employer-owned calendar would need that employer's agreement before it is connected or its data is sent to the AI. | — | Jay, 2026-11-18 | Earlier; calendar plan §2, §13 |
| C15 | House rules in v1 | Decided | Six kinds: day frame, home-by time, weekly block, travel buffer, prayer length and task length, with a default day frame when none is set. Accepted in the answer-29 group; shown to the owner for a second look. | Free-text rules read by the AI | Jay, 2026-11-18 | Answer 29; calendar plan §7 |
| C16 | Build priority | Decided | Calendar first: the calendar Issues come before the Secretary on the web, and mail after that. | Mail first | Jay, 2026-11-18 | Earlier; build order |
| C17 | When to ask for calendar write access | To confirm | Recommended: at the first sign-in, so one consent covers reading and writing. | A later "allow GYST to edit your calendar" consent step | Jay (pending); recommended 2026-11-18 | Calendar plan §16.1 |
| C19 | Secretary changes to events | Decided | Every event change the Secretary proposes is a card the owner confirms, with Undo, and a delete is held briefly before it is sent to the provider. Still to confirm, the durations: Undo for 10 minutes, a 2-minute hold. | — | Jay, 2026-11-18 | Answers 21, 35; calendar plan §16.3, §16.4 |
| C20 | Repeating events and guests | To confirm | Recommended: an edit applies to "this event" or "the whole series"; no adding or removing guests in v1; an event with guests can be changed only when the owner organises it, and the guests are told. | Full guest editing | Jay (pending); recommended 2026-11-18 | Calendar plan §16.3 |
| C21 | Creating repeating events | To confirm | Recommended: v1 creates single events only; repeating events can be edited but not created. | Repeating events can be created in v1 | Jay (pending); recommended 2026-11-18 | Calendar plan §16.3 |

## 6. Notifications

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| N1 | Notification channel | Decided | GYST's own Web Push. An iPhone receives it when GYST is added to the Home Screen and opened from there. | A native app; Slack or email first | Jay, 2026-11-18 | Answer 11; calendar plan §15 |
| N2 | Reminder lead times | Decided | Major events (N7) 1 day before; everyday events and timed tasks 25 minutes before. | 10 or 15 minutes before every event | Jay, 2026-11-18 | Answer 12; calendar plan §17 |
| N3 | Quiet hours | Decided | None by default. The quiet-hours switch stays, set to off. | Overnight quiet hours by default | Jay, 2026-11-18 | Answer 13; calendar plan §17 |
| N4 | What the lock screen shows | Decided | An event reminder shows the title and time, with a Hide details switch. Email and Secretary notices show no content. | Generic text for everything | Jay, 2026-11-18 | Answer 15; calendar plan §13 |
| N5 | Slack and email | Decided | Fallback only, built only if Web Push fails on the owner's iPhone. | Build both now | Jay, 2026-11-18 | Answer 11; calendar plan §13 |
| N6 | Notification storage | Decided | New notification preference and delivery tables. The prayer push tables and behaviour stay unchanged, proved by a golden test. | Widen the prayer delivery table | Jay, 2026-11-18 | Answer 29; calendar plan §15.4 |
| N7 | What counts as a "major" event | Decided | An event is major when the owner switches Major on for it. GYST may suggest Major for kinds of events such as important work meetings, client milestones or appointments; the switch is what counts. "All-day events are major by default" is not part of the decision. | A fixed rule (all-day, 2 hours or longer, or with guests) | Jay, 2026-11-18 | Later; calendar plan §17 |
| N8 | Morning brief | Decided | A morning brief at 08:00 every day: a push with no content, with the details inside GYST. | — | Jay, 2026-11-18 | Answers 14, 24; Secretary brain §20 |

## 7. Secretary and AI

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| D1 | Siri as the main channel | Dropped | No Siri integration. The in-app listening mode (D3, S19) replaces it. | — | Jay, 2026-11-18 | Answers 20, 25; voice plan §7 |
| D2 | What may go to the AI provider | Decided | Everything a secretary would see, sent to the provider in D7, except items the S16 guard flags. Local-only items are included (S15). | Tasks and Today only (the earlier recommendation) | Jay, 2026-11-18 | Answer 23; round 7; voice plan §7 |
| D3 | In-app voice | Decided | The owner says "Secretary" while GYST is open in a listening mode; there is no separate app. The VX-1 phone test proves it works first. The speech engine is S20. | Keyboard dictation and Safari speech only | Jay, 2026-11-18 | Answer 20; voice plan §7 |
| D4 | Siri device token | Dropped | Not needed: there is no Siri channel. | — | Jay, 2026-11-18 | Answers 20, 25; voice plan §7 |
| D5 | Siri scope and public paths | Dropped | Not needed: there is no Siri channel. | — | Jay, 2026-11-18 | Answers 20, 25; voice plan §7 |
| D6 | Auto-confirm | Decided | Off. Every change needs Confirm. | Auto-confirm per action, off by default | Jay, 2026-11-18 | Answer 21; voice plan §7 |
| D7 | AI provider | Decided | Anthropic (Claude) for now, paid from the owner's prepaid API credit. Gemini or ChatGPT may follow when the credit runs out, each as a new adapter and a new decision. Anthropic's privacy centre states that API inputs and outputs are not used for training unless the customer opts in ([source](https://privacy.claude.com/en/articles/7996868-is-my-data-used-for-model-training)). | Stay on OpenAI (the earlier recommendation) | Jay, 2026-11-18 | Earlier; Secretary brain §13 |
| D8 | Monthly AI cap | Decided | $30 a month for now, for all AI features together (Secretary, Coach and food). | An assistant sub-cap inside the $30 | Jay, 2026-11-18 | Answer 28; Secretary brain §15 |
| D9 | How long conversations are kept | Decided | The full text of both sides for 1 year, with a delete button; no sound (S21). | Keep sound recordings as well | Jay, 2026-11-18 | Round 7; Secretary brain §21 |
| D10 | One proposal system | Decided | One generic proposal-and-change system for every change the AI proposes. | A second, health-only system | Jay, 2026-11-18 | Answer 29; voice plan §7 |
| D11 | Reaching development or infrastructure | Decided | Never: no merge, deploy, shell or backend access. App data only. | — | Jay, 2026-11-18 | Answer 22; voice plan §7 |
| S2 | Freshness | Decided | Before answering, sync a source whose copy is older than its limit (mail 10 minutes, calendar 15, workouts 60) within a 10-second budget on the web; if that fails, answer from the stored copy and say "as of" the time. | Always live; never sync on demand | Jay, 2026-11-18 | Answer 29; Secretary brain §11, §22 |
| S3 | Data sent to the provider | Decided | Tasks, projects, calendar, health, nutrition, training, prayer, courses and mail details, for accounts in every group, all through the S16 guard, sent to the provider in D7. | Tasks and Today only | Jay, 2026-11-18 | Answer 23; Secretary brain §11, §17 |
| S4 | Email bodies | Decided | On request (at most 3 per turn, trimmed text), plus the needs-reply snippets in the scheduled morning brief (S27). No background AI triage. S16 applies. | Off entirely | Jay, 2026-11-18 | Earlier, amended by answers 14, 24; Secretary brain §11, §17 |
| S5 | Employer accounts and the AI | Decided | None of the accounts belongs to an employer, so Work-labelled accounts are treated like the others. Generic rule kept: an employer-owned account would get counts only until the employer agrees. | — | Jay, 2026-11-18 | Earlier; Secretary brain §11 |
| S6 | Event descriptions and guests | Decided | The Secretary may see event descriptions and guest names. Guest email addresses are not stored. | Only on request; guest names never | Jay, 2026-11-18 | Answer 23; calendar plan §16.4 |
| S7 | Meaning of "next Friday" | Decided | The coming Friday. Every answer names the date. | Friday of next week | Jay, 2026-11-18 | Answer 10; Secretary brain §11 |
| S8 | Free-time phrasing | Decided | The model phrases it; the server checks every time against the free-time engine and falls back to its own sentence. | Always a fixed template | Jay, 2026-11-18 | Answer 29; Secretary brain §11 |
| S9 | Workout pick | Decided | A deterministic ranking from workout history and the weekly target; nutrition as context only; no medical claims. Accepted in the answer-29 group; shown to the owner for a second look. | Let the model choose freely | Jay, 2026-11-18 | Answer 29; Secretary brain §11 |
| S10 | Adding a project | Decided | Always a proposal the owner confirms, never auto-confirmed, with Undo. Still to confirm: the 10-minute undo. | A high-class action (web plus password) | Jay, 2026-11-18 | Answer 21; Secretary brain §11 |
| S11 | Proposals after reading email or invite text | Decided | Marked as tainted, labelled on the card, and never auto-applied. | Treated like other proposals | Jay, 2026-11-18 | Answer 21; Secretary brain §8, §11 |
| S12 | Secretary-ready rule | Decided | Every chunk's Issue and PR names its Secretary tool, or says N/A. | Ad hoc | Jay, 2026-11-18 | Answer 29; Secretary brain §9, §11 |
| S13 | Siri reading mail | Dropped | Not needed: there is no Siri channel. | — | Jay, 2026-11-18 | Answer 25; Secretary brain §22 |
| S14 | A Secretary that speaks first | Decided | She speaks only when asked, except for the 08:00 morning brief. | No morning brief in v1 | Jay, 2026-11-18 | Answer 24; Secretary brain §20 |
| S15 | Local-only items and the AI | Decided | The Secretary can see items marked Local-only and their tasks, and they go to the AI like any other item, except what the S16 guard flags. They still never appear in calendar feeds. | Keep Local-only items hidden from the AI (the earlier recommendation) | Jay, 2026-11-18 | Round 7; Secretary brain §17 |
| S16 | Sensitive-content guard | Decided | A deterministic check on the server runs before any text reaches the provider. An email, event, task or note that holds a licence or product key, an API key or token, a private key, a password, a one-time or recovery code, or a card, bank, social insurance or driver's licence number is withheld whole, never partly redacted. She replies with a fixed sentence that points to GYST. Skips are logged as a count only. It is an acceptance criterion for every Secretary PR. The detector list is refined in Issue #242 (S-G). | Redact the match and send the rest; ask the model to self-censor | Jay, 2026-11-18 | Earlier; Secretary brain §8.1 |
| S17 | An event the guard flags | Decided | All its text is withheld; its times stay as "busy, details withheld", so free time stays correct. | Drop the whole event | Jay, 2026-11-18 | Answer 29; Secretary brain §8.1 |
| S18 | Secretary results as a push | Decided | "Your answer is ready" only, never content, for turns the owner started, with its own switch. | Put the answer in the notification | Jay, 2026-11-18 | Answer 15; Secretary brain §11 |
| S19 | Listening mode | Decided | Listening only while GYST is open, on screen and unlocked, with a visible sign and the wake word "Secretary"; it stops when the screen locks or the app is left. The VX-1 phone test proves it works first. | — | Jay, 2026-11-18 | Answer 20; Secretary brain §18 |
| S20 | Speech recognition engine | To confirm | Recommended: an in-page keyword spotter and transcriber on the phone, so audio never leaves it, chosen after the VX-1 test; the browser's own recogniser where it works; keyboard dictation always available. | A hosted speech provider, only if the test fails and the owner approves a new provider | Jay (pending); recommended 2026-11-18 | Secretary brain §18 |
| S21 | Conversation log | Decided | Keep the full text of both sides for 1 year (D9); no sound recordings. | Keep sound recordings as well | Jay, 2026-11-18 | Answer 26; round 7; Secretary brain §21 |
| S22 | Which model answers | Decided | Sonnet for every turn. Opus only through a fixed planning-job rule, after clarifying questions and a Confirm. The model never chooses the model. The rule's design is as recommended. | Let the model choose; Opus for everything | Jay, 2026-11-18 | Answer 27; Secretary brain §14 |
| S23 | Control of app data | Decided | She can view, add, edit and delete tasks, projects, events, food and health logs, goals and reminders. Every write is confirmed, with single-item Undo. Still to confirm: the 10-minute undo, and that a bulk delete needs the password and lists every item, at most 50 per card. | Bulk delete with Confirm alone (inferred) | Jay, 2026-11-18 | Answers 22, 35; Secretary brain §16 |
| S24 | What she may see | Decided | Everything a secretary would see, Local-only items included (S15). S16 items are always withheld. | Fixed limits on private projects, descriptions and guests (the earlier recommendation) | Jay, 2026-11-18 | Answer 23; round 7; Secretary brain §17 |
| S25 | Roadmaps | Decided | From a goal: clarifying questions, then a planned route of courses, projects and certifications as milestones against a deadline; tracking in the morning brief; certificate or award emails proposed as milestone evidence, always as a proposal; the finished roadmap goes into an achievements book. The design is as recommended. | — | Jay, 2026-11-18 | Answer 5; Secretary brain §19 |
| S26 | Web search to find real courses | Open | Recommended: allow it in planning jobs only, at most 10 searches a job. Each search costs money and sends the search words outside GYST. | No web search: courses come from the model's own knowledge, each link marked "check this link" | Jay (pending); recommended 2026-11-18 | Secretary brain §19 |
| S27 | AI-written morning summary | Decided | Written by the AI once a day for the morning brief, within the cap. The push shows no content. S16 applies. | — | Jay, 2026-11-18 | Answers 14, 24; Secretary brain §20 |
| P1 | Food-photo analysis on Claude | Decided | Photo identification only; the photo is the only data in that call. It is its own Issue (#243). The owner later asked for the day's context to go with the meal; how that is sent is F6 (to confirm). | — | Jay, 2026-11-18 | Answer 35; Secretary brain §13 |

## 8. Mail

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| M1 | Scope of v1 | Decided | Read, search, triage, make tasks, and send and reply. Reading stays read-only at the provider: no delete, move or mark-read there. | Read-only first (the earlier recommendation) | Jay, 2026-11-18 | Answer 16; mail plan §16, §19 |
| M3 | Reply and compose | Decided | In v1. | Not now (the earlier recommendation) | Jay, 2026-11-18 | Answer 16; mail plan §16 |
| M4 | Which inboxes first | Decided | After the calendar: one Gmail inbox, then one Microsoft inbox, then the rest. | All inboxes at once (inferred) | Jay, 2026-11-18 | Answer 18; mail plan §14 |
| M5 | Employer inboxes | Decided | None of the inboxes belongs to an employer. Generic rule kept: an employer-owned inbox would need that employer's agreement first. | — | Jay, 2026-11-18 | Earlier; mail plan §3 |
| M6 | Sync timing | Decided | As M26: Inbox and Sent every 3 minutes, other folders every 30 minutes. | Every 5 minutes | Jay, 2026-11-18 | Answer 17; mail plan §16.5 |
| M7 | What is kept | Decided | All history, and attachments. Still to confirm: what "all history" means in detail (M24). | Inbox only, 90 days, no attachments | Jay, 2026-11-18 | Answer 17; mail plan §17, §18 |
| M9 | How email is shown | To confirm | Recommended: text only first. Sanitised HTML later, only after the site's content security policy is enforced. Remote images blocked. | Sanitised HTML in v1 (inferred) | Jay (pending); recommended 2026-11-18 | Mail plan §9 |
| M10 | AI summaries and triage | Decided | Summaries on request for every group, behind S16. No background AI triage. Amended: the scheduled morning-brief summary (S27) is allowed. | Background AI triage | Jay, 2026-11-18 | Earlier, amended by S27; mail plan §14 |
| M12 | Unread state | Decided | Mirrors the provider. Opening a message in GYST clears the dot in GYST only. Sending doesn't change this. | Mark read at the provider (inferred) | Jay, 2026-11-18 | Answer 29; mail plan §14 |
| M13 | Removing an inbox | Decided | Deletes its cached mail and its secret at once and revokes the token. Tasks keep the saved subject and sender. | Keep the cache until it ages out (inferred) | Jay, 2026-11-18 | Answer 29; mail plan §14 |
| M14 | The Secretary and mail | Decided | She reads mail and may draft replies. Sending always needs the owner's Confirm (M22). | Read-only, no drafts | Jay, 2026-11-18 | Earlier, amended by answer 16; mail plan §19 |
| M15 | Zoho | Decided | A Zoho account connects through IMAP; the owner confirms that the plan includes IMAP and which region the account is in. Server hosts are taken from the provider's documentation and the account's own settings, stored per account, never hard-coded. | The Zoho Mail API | Jay, 2026-11-18 | Answer 19; mail plan §16.6 |
| M19 | Email summary | Decided | Inside the 08:00 morning brief, saying "Nothing major" when nothing needs attention. It replaces a separate daily digest. | A separate daily digest; hourly | Jay, 2026-11-18 | Answer 14; mail plan §19 |
| M20 | Send permissions | Decided | Gmail `gmail.send`; Microsoft `Mail.Send` (plus `Mail.ReadWrite` only if drafts or large attachments are kept at the provider); Yahoo and Zoho through SMTP with the same app password. Scopes follow the vendors' documentation. | `gmail.compose` or `gmail.modify` (restricted and broader) | Jay, 2026-11-18 | Answer 16; mail plan §16.1 |
| M21 | Build or adopt mail | To confirm | Recommended: build inside GYST on permissively licensed Go libraries. No permissively licensed mail application fits. | Adopt an open-source webmail app (those found are copyleft and add a second runtime) | Jay (pending); recommended 2026-11-18 | Mail plan §16.4 |
| M22 | Send safeguards | Decided | Every send needs Confirm and has an undo window. Sends are rate-limited. The audit holds no addresses. The Secretary never sends without Confirm. Still to confirm, the numbers: a 20-second undo, at most 20 sends an hour and 100 a day. | Other numbers, set by the owner | Jay, 2026-11-18 | Answers 16, 21; mail plan §16.3 |
| M23 | Attachments | To confirm | Recommended: metadata for every message; content fetched on demand into a 5 GB cache that is not backed up; files the owner pins kept for good in the backed-up main database; always served as downloads. | Download every attachment during sync (inferred) | Jay (pending); recommended 2026-11-18 | Mail plan §17 |
| M24 | What "all history" means | To confirm | Recommended: the text of every message in every folder except Spam and Trash, kept until the inbox is removed; attachments on demand (M23). | Also download every attachment, or keep Spam and Trash (inferred) | Jay (pending); recommended 2026-11-18 | Mail plan §18 |

## 9. Food

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| F1 | Food and dish profiles | Decided | A profile holds a name, notes, the dish's items and one photo. | — | Jay, 2026-11-18 | Later; Secretary brain §24.1 |
| F2 | Where a profile photo is kept | Decided | A re-encoded JPEG (at most 800 px and 250 KB, location data removed) inside the main database, so backups include it; served only behind the session; deleted with the profile. The design is as recommended. This changes "photos are never stored" for this one path only. | A file outside the database (not backed up) | Jay, 2026-11-18 | Later; Secretary brain §24.1 |
| F3 | One photo per profile | Decided | A new photo for a profile that already has one is not kept unless the owner taps "Use this photo instead". | Keep several photos (inferred) | Jay, 2026-11-18 | Later; Secretary brain §24.1 |
| F4 | Describe a meal in words | Decided | The AI names the foods and grams; GYST's own food data supplies the values; the AI's figures are used only where there is no match, labelled and stored as an estimate; nothing is saved without Save. Still to confirm: it can't ship until R1 is answered. | — | Jay, 2026-11-18 | Later; Secretary brain §24.2 |
| F5 | Adjust before logging | Decided | Before logging, the owner can adjust every food in the meal, including raising or lowering its macros. The AI's original figures are kept to measure accuracy. The controls (grams per item, a meal portion from ×0.5 to ×2) follow the recommended design. | — | Jay, 2026-11-18 | Later; Secretary brain §24.3 |
| F6 | The day's context for a meal | To confirm | Recommended: the day's diary and health numbers go in a separate "how this fits your day" note after Save, not in the photo identification call. | Send them with the photo identification call | Jay (pending); recommended 2026-11-18 | Secretary brain §24.4 |
| F7 | What the AI keeps | Decided | Note, not a choice: the AI keeps nothing between calls and is not trained on this data. What it "knows" is what GYST sends each time. GYST learns usual portions itself. | — | — (a fact, recorded 2026-11-18) | Anthropic's data terms; Secretary brain §13, §24.5 |
| F8 | Owner-entered macros | Decided | The owner can type a food's macros himself, in any unit he sets up (F9). The AI estimate is used only when he hasn't. | Macros per gram only (his earlier wording, replaced by F9) | Jay, 2026-11-18 | Later; round 7 |
| F9 | Units for a food's macros | Decided | Like a food tracker's serving list: each food can have several units (1 g, 100 g, a serving, a pack, a piece), each with its weight in grams, and GYST scales the macros to the amount logged. | Macros per gram only (F8 as first written) | Jay, 2026-11-18 | Round 7 |
| R1 | Rule change for AI nutrition estimates | To confirm | Recommended: a line in [`docs/RULES.md`](../../RULES.md) §3.2 that allows a labelled AI nutrition estimate, saved by the owner, only where GYST's own food data has no match; an unknown nutrient stays NULL. Only the owner changes the rules. | No rule change, so no AI estimate is ever stored | Jay (pending); recommended 2026-11-18 | Secretary brain §24.2 |
| S28 | The Secretary as personal coach | Decided | The Secretary is also the owner's personal coach: one assistant with the health-coach role, reading his history through GYST's lookups. The model is not trained on him; GYST sends what is needed each time. Still open: how the existing Coach screen and the Secretary share one assistant. | Two separate assistants (inferred) | Jay, 2026-11-18 | Later |

## 10. Not in this record

- **ID gaps.** There is no M25. The plans never used it.
- **Makeover design answers** (colours, the fourth tab, themes, folding small
  pages) belong to the makeover's design record (13B), not here.
- **Operational answers** (refreshing and merging reviewed PRs, closing old
  tasks, deploys, branch names) are process steps, not design decisions.

## 11. Sources

The plans are on the owner's server under `artifacts/gyst/`, not in this
repository. Entries cite them by short name:

| Short name | File | Sections used |
|---|---|---|
| Calendar plan | `design-makeover/calendar-plan.md` | §2–§9, §13, §15–§17 |
| Mail plan | `design-makeover/mail-plan.md` | §2–§6, §9, §14, §16–§20 |
| Build order | `design-makeover/build-order.md`, `design-makeover/issues-r7.md` | the calendar-first order |
| Secretary brain | `voice-assistant/secretary-brain.md` | §0–§2, §7–§9, §11, §13–§24 |
| Voice plan | `voice-assistant/plan.md` | §7 |
| Charter | `voice-assistant/secretary-charter.md` | the Secretary's rules, for meaning |
| Decision list | `decisions/decisions-r7.md` | the status of every ID |

## 12. Calendar UI record

A short design source for the calendar screens, so CF-8 (#240) and the
event-write Issue have one before makeover chunk 0 merges. Everything is built
in the **current** shell and inside
[`13A_DESIGN_DECISIONS.md`](13A_DESIGN_DECISIONS.md); the makeover restyles it
later. Screenshots and checks use `seeds/example.sql` with fictional events.

**Today (phone and desktop)**
- Events join Today's **existing Now and Later rows** as a new item kind. There
  is no new section and **no layout change** (C10).
- An all-day event is one row in Later today, marked "All day".
- A screenshot comparison before and after proves the layout is unchanged.

**Agenda screen (`/agenda`)**
- A list of seven days, today first, with a day picker for the next 30 days.
- Each day shows all-day chips, then timed rows in time order: events, timed
  GYST tasks, course blocks, prayers as thin rows, and free windows.
- Every free window equals the Secretary's answer for the same day, because
  both come from the same free-time engine (S8).
- States: loading, empty, no accounts connected, needs reconnect, and stale
  (shows "as of" the time).
- Navigation: on desktop, under Today in the sidebar; on phone, from Today's
  date header and the More tab.

**Event sheet** (tapping an event)
- Time, calendar, place, the description as plain text, and guest names (S6).
- A **Major** switch for the event (N7), kept by GYST, not at the provider.
- "Open in Google" or "Open in Outlook" only when the link is an `https`
  provider link, opened with `rel="noopener noreferrer"`.
- Edit and Delete, once event writes ship.

**Creating and editing events** (the event-write Issue, CF-10 in the plans,
not yet opened; CF-8 doesn't add the New event button)
- A New event form: title, start and end or all-day, place, description, and
  the calendar, choosing only calendars the owner may write to. The default
  calendar for new events is set in Settings.
- Single events only (C21, to confirm). Editing a repeating event asks
  "this event" or "the whole series" (C20, to confirm).
- No adding or removing guests. An event with guests can be changed only when
  the owner organises it, and the card says the guests will be told
  (C20, to confirm).
- A change proposed by the Secretary is a card the owner confirms (C19), with
  Undo. A deleted event disappears from GYST at once, and the provider delete
  waits a short time (10-minute undo and 2-minute hold: to confirm, C19).
- If the event changed at the provider in the meantime, GYST shows the new
  version and the owner re-applies or drops the change (C18).
- With the provider unreachable, the change fails with a clear message.
  Nothing is queued.

**Settings rows**
- **Accounts:** Sign in with Google or Microsoft, status, Reconnect, and
  Remove (needs the password).
- **Calendars:** each calendar with its colour and name, a Show switch and a
  Counts as busy switch, both off by default; GYST's own feed is marked "This
  is GYST's own feed" and left off (C12); the default calendar for new events.
- **House rules:** the six kinds in C15, edited through a normal form.
- **Notifications:** Enable (on a tap), a switch per category (event
  reminders, morning brief, Secretary results), lead times (major 1 day,
  everyday 25 minutes; N2), the quiet-hours switch (off; N3), Hide details
  (N4), the list of devices with Remove, and on an iPhone not running GYST
  from the Home Screen, a note on how to add it.

**Deferred**
- The Overview **Calendar band** (the next event and today's free windows) is
  built inside makeover chunk **6A**, because Overview doesn't exist yet. The
  Agenda and Today parts don't wait for it.
- The round-5 Day and Week grid belongs to the makeover era.

**For CF-8's design review** (not approved here): dashed rows for free windows
and calendar-colour dots on event rows are patterns 13A doesn't define yet.
