# Connected data decisions: accounts, calendar, notifications, mail, the Secretary and food

- **Decided by:** Jay (product owner), for every entry marked Decided or Dropped
- **Recorded:** 2026-11-18 (Issue #233, CF-1); updated 2026-11-19 from the owner's rulebook marks
- **Responds to:** the owner's calendar, mail, Secretary and build-order plans of
  2026-11-18 (round 7). They are kept on the server, outside this repository;
  see [Sources](#11-sources). It also records the owner's marks on every
  Secretary rule and his four clarifications of 2026-11-19.
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
| **Decider, date** | "Jay, 2026-11-18" for Decided and Dropped; "Jay (pending)" for To confirm and Open, with the date of the recommendation. "Jay, 2026-11-19" for entries decided by his rulebook marks; "Jay, 2026-11-18; … 2026-11-19" where a decided entry was amended. |
| **Source** | What the entry rests on. "Answer *n*" is the owner's numbered answer to the 36 open questions of 2026-11-18. "Earlier" and "later" are his messages before and after those answers on the same day. "Round 7" is his set of design and Secretary choices later on 2026-11-18; this record holds the three that belong here (S15, S21 with D9, and F9). "Rulebook mark rN" is a Secretary rule the owner marked right on 2026-11-19, and "rulebook note rN" one he changed with a note (aN and bN are review items, kN and pN the list's conflicts and gaps). "Clarification Qn" is one of his four answers that day, and "round 2 O-…" his answer to that rulebook question. Plan sections are listed in [Sources](#11-sources). |

**Answer 29.** Answer 29 was one yes to a group of twenty items that the lead described to the owner as engineering details (sync timings, minimum permissions, storage tables, token handling). The lead who asked the question listed the twenty as M2, M11, M12, M13, M17, C2, C3, C4, C12, C13, C15, N6, S2, S8, S9, S12, S17, D4, D5 and D10. D4 and D5 were later dropped, which leaves eighteen Decided entries that rest on answer 29 (C2 and C3 also rest on answer 8). Three of them shape what the owner sees rather than how GYST works inside: C13, C15 and S9. They stay Decided, are marked "accepted in the answer-29 group", and are shown to him for a second look.

## 2. Summary

| Area | Decided | To confirm | Open | Dropped | Total |
|---|---|---|---|---|---|
| [3. Accounts and sign-in](#3-accounts-and-sign-in) | 5 | 0 | 0 | 0 | 5 |
| [4. Storage and sync](#4-storage-and-sync) | 5 | 0 | 0 | 0 | 5 |
| [5. Calendar](#5-calendar) | 20 | 0 | 0 | 0 | 20 |
| [6. Notifications](#6-notifications) | 12 | 0 | 0 | 0 | 12 |
| [7. Secretary and AI](#7-secretary-and-ai) | 45 | 0 | 0 | 4 | 49 |
| [8. Mail](#8-mail) | 19 | 0 | 0 | 0 | 19 |
| [9. Food](#9-food) | 12 | 0 | 0 | 0 | 12 |
| **All** | **118** | **0** | **0** | **4** | **122** |

**Waiting for Jay** (none of these is decided):

- No entry is To confirm.
- Parts of Decided entries:
  - the provider for GYST's own outgoing mail (M27), through a Tool Proposal;
  - whether the server can run the local fallback model (S33).
  S28 has a design still to do, and S19, S30 and D3 wait on the VX-1 phone test.
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
| C17 | When to ask for calendar write access | Decided | At the first sign-in, so one consent covers reading and writing events. | A later "allow GYST to edit your calendar" consent step | Jay, 2026-11-19 | Round 3 C17; calendar plan §16.1 |
| C19 | Secretary changes to events | Decided | Every event change the Secretary proposes is a card the owner confirms with a tap. Undo lasts 10 minutes where recovery is supported. A delete is held 2 minutes before it is sent to the provider; held deletes survive a restart, and GYST asks again before acting. Each card states its real recovery window, lists the guests who will be told, and says what restoring can recover. | — | Jay, 2026-11-18; durations 2026-11-19 | Answers 21, 35; rulebook marks r87, r88, b14; notes k21, b6; calendar plan §16.3, §16.4 |
| C20 | Repeating events and guests | Decided | An edit applies to "this event" or "the whole series". No adding or removing guests in v1. An event with guests can be changed only when the owner organises it, and the guests are told. The Secretary sets an event's Major switch through the normal event-edit card. | Full guest editing | Jay, 2026-11-19 | Rulebook marks r56, r57, r212; note p6; calendar plan §16.3 |
| C21 | Creating repeating events | Decided | v1 creates single events only; repeating events can be edited but not created. | Repeating events can be created in v1 | Jay, 2026-11-19 | Rulebook marks r55, r212; calendar plan §16.3 |
| C22 | Replying to invites | Decided | The Secretary may propose accepting or declining an invite as a card. The card shows the event, date and time, organiser, calendar account and proposed response, and says the organiser will be notified. GYST sends the reply only after the owner taps Confirm and verifies with a password or passkey. If the invite changed since the card was made, GYST shows an updated card for review. | No invite replies in v1 | Jay, 2026-11-19 | Round 2 O-rsvp; rulebook gap p5 |

## 6. Notifications

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| N1 | Notification channel | Decided | GYST's own Web Push, on the iPhone and on the desktop browser (N9). An iPhone receives it when GYST is added to the Home Screen and opened from there. | A native app; Slack or email first | Jay, 2026-11-18; desktop 2026-11-19 | Answer 11; rulebook note r179; calendar plan §15 |
| N2 | Reminder lead times | Decided | Major events (N7) 1 day before; everyday events and timed tasks 25 minutes before. | 10 or 15 minutes before every event | Jay, 2026-11-18 | Answer 12; calendar plan §17 |
| N3 | Quiet hours | Decided | None by default. The quiet-hours switch stays, set to off. | Overnight quiet hours by default | Jay, 2026-11-18 | Answer 13; calendar plan §17 |
| N4 | What the lock screen shows | Decided | An event reminder shows the title and time, with a Hide details switch. Email and Secretary notices show no content. | Generic text for everything | Jay, 2026-11-18 | Answer 15; calendar plan §13 |
| N5 | Slack and email | Decided | Fallback only, built only if Web Push fails on the owner's iPhone. | Build both now | Jay, 2026-11-18 | Answer 11; calendar plan §13 |
| N6 | Notification storage | Decided | New notification preference and delivery tables. The prayer push tables and behaviour stay unchanged, proved by a golden test. | Widen the prayer delivery table | Jay, 2026-11-18 | Answer 29; calendar plan §15.4 |
| N7 | What counts as a "major" event | Decided | An event is major when the owner switches Major on for it. GYST may suggest Major for kinds of events such as important work meetings, client milestones or appointments; the switch is what counts. The Secretary sets it through the normal event-edit card. "All-day events are major by default" is not part of the decision. | A fixed rule (all-day, 2 hours or longer, or with guests) | Jay, 2026-11-18 | Later; rulebook note p6; calendar plan §17 |
| N8 | Morning brief | Decided | The brief job starts at 7:30 and the brief is ready by 8:00 (N10). At 08:00 a push with no content; the details are inside GYST. | — | Jay, 2026-11-18; timing 2026-11-19 | Answers 14, 24; rulebook note r202; round 2 O-briefcontent; Secretary brain §20 |
| N9 | Desktop notifications | Decided | While GYST is open, a corner popup with Open, Snooze and Dismiss, in a corner the owner chooses; with several monitors, on the monitor where GYST is open. While GYST is minimised or in the background, native desktop notifications where supported; the operating system controls which monitor and corner they appear on, and its sound and Do Not Disturb settings apply. Sound has Silent, Normal and Loud settings with a Test sound button; the owner can use the loud chime for major meetings and urgent reminders and a quieter sound for routine alerts. Privacy matches the lock screen (N4): the event title and time with a Hide details setting, and Secretary notices such as "Your morning brief is ready" or "Your answer is ready"; email, health and private details only inside GYST. Slack or email stay the fallback (N5). | iPhone only | Jay, 2026-11-19 | Rulebook note r179; round 2 O-desktop |
| N10 | When the brief runs, and the email fallback | Decided | At 7:30 GYST time the brief job starts preparing the brief across connected inboxes, calendars, tasks, projects and health logs, compared with the last completed brief. By 8:00 the full brief is ready in GYST, to read or to hear, with the source details and room for follow-up questions; it stays valid until the next day's brief. At 10:00, if the owner hasn't opened or heard it, GYST refreshes the time-sensitive details and sends it once to his verified address (M27), offering only time slots still ahead; if email delivery fails, a push. Any time he asks, a catch-up since the last brief, saying which source or messages weren't covered if a sync or the mail review didn't finish. Unchanged items say "still on"; after 3 days without change, "parked for now", with one way to get them moving again. | A brief generated at 08:00 (the earlier design) | Jay, 2026-11-19 | Rulebook notes r6, r168, r202; clarification Q2; round 2 O-briefcontent |
| N11 | What the brief holds | Decided | Eight sections. **Start here:** the top three things needing attention today, why they matter and the next step for each, calling out anything that could slip. **Today's schedule:** meetings, appointments, timed tasks and the next prayer time, flagging conflicts, gaps and prep before a major event. **Tasks and deadlines:** every overdue task and every task due today or in the next three days, grouped by deadline, with moved deadlines, what he's waiting on and the next action. **What changed since the last brief:** new, moved or cancelled meetings, changed deadlines, new assignments, replies that unblock work and anything newly urgent, today's changes first. **Inbox that needs attention:** important new or unread mail and older threads needing his reply, by thread, linked to the source in GYST. **Emails I need to send:** replies, follow-ups and new emails due today or in the next two days, from explicit commitments in sent mail, tasks, reminders, drafts and threads; inferred ones labelled as suggestions; she can offer to draft each. **Yesterday's health and today's plan:** a short Coach note from what he logged, with one practical next step, a real calendar opening for training if it fits his plan, or "No workout logged". **Waiting on others:** replies, decisions and deliverables he's waiting for, with when to follow up. "Due soon" starts at three days for tasks and two days for outgoing email. | A short summary paragraph (the earlier design) | Jay, 2026-11-19 | Round 2 O-briefcontent; rulebook note r168 |
| N12 | What the 10:00 email holds | Decided | The agenda, the due-soon task list, important overnight changes, the "Emails I need to send" list and the short Coach note, with a one-line summary and a GYST link for each item where the details matter. For sensitive items the emailed action stays useful and the details sit behind the GYST link. The original thread, quoted messages, attachments and private details stay in GYST. The Coach note may say he came up short on fiber or had a strong session; exact health numbers and food logs stay in GYST. The owner can turn off the emailed Coach note. | The full brief by email | Jay, 2026-11-19 | Round 2 O-briefcontent; rulebook note a2 |

## 7. Secretary and AI

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| D1 | Siri as the main channel | Dropped | No Siri integration. The in-app listening mode (D3, S19) replaces it. | — | Jay, 2026-11-18 | Answers 20, 25; voice plan §7 |
| D2 | What may go to the AI provider | Decided | Everything a secretary would see, sent to the provider in D7, except items the S16 guard flags. Local-only items are sent like any other (S15). | Tasks and Today only (the earlier recommendation) | Jay, 2026-11-18; Local-only 2026-11-19 | Answer 23; clarification Q1; voice plan §7 |
| D3 | In-app voice | Decided | The owner says "Secretary" while GYST is open in a listening mode; there is no separate app. The VX-1 phone test proves it works first. The speech engine is S20. | Keyboard dictation and Safari speech only | Jay, 2026-11-18 | Answer 20; voice plan §7 |
| D4 | Siri device token | Dropped | Not needed: there is no Siri channel. | — | Jay, 2026-11-18 | Answers 20, 25; voice plan §7 |
| D5 | Siri scope and public paths | Dropped | Not needed: there is no Siri channel. | — | Jay, 2026-11-18 | Answers 20, 25; voice plan §7 |
| D6 | Auto-confirm | Decided | Off for all of v1. Every change needs a tap on Confirm. Auto-confirm comes back only if the owner asks for a specific action to be automated. The daily brief is a scheduled notification, not an auto-confirmed action. | Auto-confirm per action, off by default | Jay, 2026-11-18; v1 scope 2026-11-19 | Answer 21; rulebook mark r83; note k10; voice plan §7 |
| D7 | AI provider | Decided | Anthropic (Claude) for now, paid from the owner's prepaid API credit. Gemini or ChatGPT may follow when the credit runs out, each as a new adapter and a new decision. Anthropic's privacy centre states that API inputs and outputs are not used for training unless the customer opts in ([source](https://privacy.claude.com/en/articles/7996868-is-my-data-used-for-model-training)). GYST calls Claude with regular API calls only: no saved threads, files uploaded for later, or hosted memory. The retention terms of the actual workspace and endpoints are checked and shown to the owner. | Stay on OpenAI (the earlier recommendation) | Jay, 2026-11-18; call style 2026-11-19 | Earlier; rulebook note a14; Secretary brain §13 |
| D8 | Monthly AI cap | Decided | $30 a month for now, for all AI features together (Secretary, Coach and food). Usage limits under it are S34; the local fallback is S33. | An assistant sub-cap inside the $30 | Jay, 2026-11-18 | Answer 28; Secretary brain §15 |
| D9 | How long conversations are kept | Decided | The full text of both sides for 1 year, with a delete button; no sound (S21). Replies and summaries made from mail go by default into the store that is not backed up, and raw mail lookups never enter the chat history. Deleting a conversation removes its transcript, the summaries made from it, and its pending cards and drafts; completed actions and minimal audit records stay, and GYST states what backups and the provider still hold. An activity history keeps metadata only: provider and model, data categories and internal references, action outcomes, blocks and cost. | Keep sound recordings as well | Jay, 2026-11-19 | Rulebook marks r131–r133; notes a6, a11, b11; Secretary brain §21 |
| D10 | One proposal system | Decided | One generic proposal-and-change system for every change the AI proposes. | A second, health-only system | Jay, 2026-11-18 | Answer 29; voice plan §7 |
| D11 | Reaching development or infrastructure | Decided | Never in v1: no merge, deploy, shell or backend access. App data only. Contact with Command Center is planned for v2. | — | Jay, 2026-11-18; v2 note 2026-11-19 | Answer 22; rulebook note r193; voice plan §7 |
| S2 | Freshness | Decided | Before answering, sync a source whose copy is older than its limit (mail 10 minutes, calendar 15, workouts 60) within a 10-second budget on the web; if that fails, answer from the stored copy and say "as of" the time. | Always live; never sync on demand | Jay, 2026-11-18 | Answer 29; Secretary brain §11, §22 |
| S3 | Data sent to the provider | Decided | Tasks, projects, calendar, health, nutrition, training, prayer, courses and mail details, for accounts in every group, all through the S16 guard, sent to the provider in D7. | Tasks and Today only | Jay, 2026-11-18 | Answer 23; Secretary brain §11, §17 |
| S4 | Email bodies | Decided | In a conversation she reads bodies when she needs the context, at most 3 per turn (trimmed text). The daily brief job reads new mail (S27). No other background AI triage. S16 applies. | On request only (the earlier decision) | Jay, 2026-11-19 | Rulebook note r30; marks r208, r209; clarification Q2 |
| S5 | Employer accounts and the AI | Decided | None of the accounts belongs to an employer, so Work-labelled accounts are treated like the others. Generic rule kept: an employer-owned account would get counts only until the employer agrees. | — | Jay, 2026-11-18 | Earlier; Secretary brain §11 |
| S6 | Event descriptions and guests | Decided | The Secretary may see event descriptions and guest names. Guest email addresses are not stored. Other people reach the model as names and contact ids; GYST fills in and shows the addresses. | Only on request; guest names never | Jay, 2026-11-18; contact ids 2026-11-19 | Answer 23; rulebook notes k7, b8; calendar plan §16.4 |
| S7 | Meaning of "next Friday" | Decided | The coming Friday. Every answer names the date. | Friday of next week | Jay, 2026-11-18 | Answer 10; Secretary brain §11 |
| S8 | Free-time phrasing | Decided | The model phrases it; the server checks every time against the free-time engine and falls back to its own sentence. | Always a fixed template | Jay, 2026-11-18 | Answer 29; Secretary brain §11 |
| S9 | Workout pick | Decided | A deterministic ranking from workout history and the weekly target; nutrition as context only; no medical claims. Accepted in the answer-29 group; shown to the owner for a second look. | Let the model choose freely | Jay, 2026-11-18 | Answer 29; Secretary brain §11 |
| S10 | Adding a project | Decided | Always a proposal the owner confirms, never auto-confirmed, with a 10-minute Undo. Undoing a new project that already has tasks: she tells the owner the tasks will be removed too, and removes them if he agrees. | A high-class action (web plus password) | Jay, 2026-11-18; undo 2026-11-19 | Answer 21; rulebook marks r38, r87; notes r39, k21; Secretary brain §11 |
| S11 | Proposals after reading email or invite text | Decided | Marked as tainted, labelled on the card, and never auto-applied. After she reads mail, an invite or web text, that turn may only propose additions or a reply in the same thread: no deletes, forwards, bulk changes or new recipients. Untrusted text stays marked wherever it is stored and when it comes back later. GYST's server enforces this, and it is tested with a fake model that obeys injected text. | Treated like other proposals | Jay, 2026-11-18; server limits 2026-11-19 | Answer 21; rulebook marks a1, a5, b2; Secretary brain §8, §11 |
| S12 | Secretary-ready rule | Decided | Every chunk's Issue and PR names its Secretary tool, or says N/A. | Ad hoc | Jay, 2026-11-18 | Answer 29; Secretary brain §9, §11 |
| S13 | Siri reading mail | Dropped | Not needed: there is no Siri channel. | — | Jay, 2026-11-18 | Answer 25; Secretary brain §22 |
| S14 | A Secretary that speaks first | Decided | She speaks only when asked, except for the daily brief (N8, N10) and its 10:00 email when it hasn't been opened. | No morning brief in v1 | Jay, 2026-11-18; brief timing 2026-11-19 | Answer 24; rulebook notes r6, r202; Secretary brain §20 |
| S15 | Local-only items and the AI | Decided | Local-only items are treated like any project: they go to the AI like everything else, apart from what the S16 guard withholds. The label only keeps them out of calendar feeds. | Keep Local-only items hidden from the AI (the earlier recommendation) | Jay, 2026-11-19 | Round 7 choice; rulebook note r25; clarification Q1 |
| S16 | Sensitive-content guard | Decided | A deterministic check on the server runs before any text reaches the provider. An email, event, task, note or attachment that holds a licence or product key, an API key or token, a private key, a password, a one-time or recovery code, or a card, bank, social insurance, driver's licence, passport or health-card number is withheld whole, never partly redacted. She replies with a fixed sentence that points to GYST. Skips are logged as a count only. The guard also checks the owner's own words (a hit is neither sent nor logged, and she says why), the brief job's mail, web-search words, conversation-log search and attachment text. GYST says plainly that it screens for known patterns and that some secrets will get through. A final check on every outgoing request is a go-live requirement. It is an acceptance criterion for every Secretary PR. The detector list is refined in Issue #242 (S-G). | Redact the match and send the rest; ask the model to self-censor | Jay, 2026-11-18; widened 2026-11-19 | Earlier; rulebook marks r101, r107, r108, a10, b10; note b3; round 2 O-ids; Secretary brain §8.1 |
| S17 | An event the guard flags | Decided | All its text is withheld; its times stay as "busy, details withheld", so free time stays correct. | Drop the whole event | Jay, 2026-11-18 | Answer 29; Secretary brain §8.1 |
| S18 | Secretary results as a push | Decided | "Your answer is ready" only, never content, for turns the owner started, with its own switch. | Put the answer in the notification | Jay, 2026-11-18 | Answer 15; Secretary brain §11 |
| S19 | Listening mode | Decided | Listening only while GYST is open, on screen and unlocked, with a visible sign and the wake words "Secretary" and "Lazy" (S30); it stops when the screen locks or the app is left. Listening pauses while she speaks; speech with no clear request is neither sent nor logged. The VX-1 phone test proves it works first. | — | Jay, 2026-11-18; second wake word 2026-11-19 | Answer 20; rulebook notes r119, r121; mark b5; Secretary brain §18 |
| S20 | Speech recognition engine | Decided | An in-page keyword spotter and transcriber on the phone, so audio never leaves it, validated on the owner's iPhone in VX-1. Spoken use must be a real back-and-forth conversation. Keyboard dictation is always available. A hosted speech provider only if the test fails and the owner approves a new provider. | A hosted speech provider from the start | Jay, 2026-11-19 | Rulebook notes r126, k21; mark r125; Secretary brain §18 |
| S21 | Conversation log | Decided | The full text of both sides, kept 1 year (D9); no sound recordings. | Keep sound recordings as well | Jay, 2026-11-19 | Round 7 choice; rulebook marks r131, r132; Secretary brain §21 |
| S22 | Which model answers | Decided | Sonnet for every everyday turn, coaching included. Opus only on a card the owner confirms with its cost ceiling: for a planning job after clarifying questions, or when she asks for it and gives a concrete reason. The model never chooses the model. Each turn stops at 6 model steps and a fixed token limit; a planning job stops at its cost ceiling. | Let the model choose; Opus for everything | Jay, 2026-11-18; amended 2026-11-19 | Answer 27; rulebook notes r146, k25; mark a9; Secretary brain §14 |
| S23 | Control of app data | Decided | She can view, add, edit and delete tasks, projects, events, food and health logs, goals and reminders. Every write is confirmed with a tap, with a 10-minute Undo where recovery is supported. Sending mail, deleting an event with guests and any bulk change also need a password or passkey checked within the last 5 minutes. A bulk card lists every item, at most 50. Confirm re-checks the target and fails if it changed. Settings, Apple Health, training cycles, health targets and Hevy stay with the owner: she may suggest a change and say why. Bulk means editing or deleting more than one item, or creating more than 10 at once; one meal, one event, or undoing a new project with its tasks is not bulk. | Bulk delete with Confirm alone (inferred) | Jay, 2026-11-18; amended 2026-11-19 | Answers 22, 35; rulebook marks r90, r92, a4, a16; notes k11, k12, k21, p4, r214; round 2 O-bulk; Secretary brain §16 |
| S24 | What she may see | Decided | Everything a secretary would see, including Local-only items (S15) and supported attachments once the S16 guard has screened them. S16 items are always withheld. | Fixed limits on private projects, descriptions and guests (the earlier recommendation) | Jay, 2026-11-18; amended 2026-11-19 | Answer 23; clarification Q1; rulebook notes r33, r191, p11; Secretary brain §17 |
| S25 | Roadmaps | Decided | From a goal: clarifying questions, then a planned route of courses, projects and certifications as milestones against a deadline; tracking in the morning brief; certificate or award emails proposed as milestone evidence, always as a proposal; the finished roadmap goes into an achievements book. She asks up to five questions, only about what GYST lacks: goal, deadline, weekly time and starting level first, then limits such as budget or learning format. A certificate email matches only with a passing DKIM signature from the issuer's domain. The design is as recommended. | — | Jay, 2026-11-18; amended 2026-11-19 | Answer 5; rulebook note k24; mark b13; Secretary brain §19 |
| S26 | Web search | Decided | Allowed. Searches run through GYST's approved search tool, with a minimal query made from the owner's goal; email, profile and health details stay out of it, and it passes the S16 guard. After she has read private mail, a search needs a separate request. Results count as outside content. In planning jobs, at most 10 searches a job; in a normal chat turn, when the owner asks for a link, a source or a fact, at most 3. | No web search: courses come from the model's own knowledge, each link marked "check this link" | Jay, 2026-11-19 | Rulebook notes r211, a13; mark r152; round 2 O-search |
| S27 | AI-written morning summary | Decided | Written by the AI once a day by the 7:30 brief job, within the cap. The job checks every connected inbox and may read the bodies of new messages since the last completed brief, older messages still unread, and open threads that appear to need a reply. It says what is important, what needs a reply and what changed; it groups updates by thread with links to the originals in GYST, and repeats an older item only if it still needs attention or has changed. It says if an inbox couldn't sync or some messages weren't reviewed, and leaves read and unread status unchanged. It also finds commitments in sent mail, drafts, tasks and reminders for the "Emails I need to send" list (N11). "Start here" is at most 120 words: the top three priorities, why each matters and what to do next; the other seven sections run as long as their lists need. Each section has an icon and a heading, tables where they make deadlines easier to scan, and coloured status labels paired with words such as Overdue, Moved or Waiting; it reads well on desktop and mobile. The push shows no content. S16 applies. Without any AI, the brief still arrives with its lists and "Summary unavailable today". | — | Jay, 2026-11-18; mail reading and layout 2026-11-19 | Answers 14, 24; clarification Q2; rulebook marks r170, r171, b4; round 2 O-briefcontent, O-nosummary; round 3 O-briefcap; Secretary brain §20 |
| P1 | Food-photo analysis on Claude | Decided | The camera's first read sends only the photo; it is its own Issue (#243). To estimate a home-cooked meal's macros, the photo (also a saved profile photo), the food's name, its ingredients and the owner's answers to her questions may be sent (F10). How the day's context is used is F6. | — | Jay, 2026-11-18; estimate step 2026-11-19 | Answer 35; rulebook mark r111; notes r112, k20; Secretary brain §13 |
| S29 | How the Secretary knows the owner | Decided | An editable "About me" profile (who he is, his work, foods, usual schedule, rush or calm weeks, usual plans, experience, passions, and what he wants to become), built in a short setup conversation. She may suggest new or changed entries; an entry is saved only when he taps "Remember this". GYST uses current activity to understand today and this week; temporary context expires when it stops applying. Each request gets the relevant profile details and current app data from a reused template; changes and deletions apply on the next request. | GYST gathers context afresh each time (inferred) | Jay, 2026-11-19 | Rulebook notes r17, r18; clarification Q4 |
| S30 | A second wake word | Decided | "Lazy" also wakes her, alongside "Secretary". The VX-1 test checks it for false wake-ups the same way. | "Secretary" only | Jay, 2026-11-19 | Rulebook notes r119, r121 |
| S31 | How long she listens after the wake word | Decided | Up to 90 seconds of speech, ending after 5 seconds of silence. After she answers, she listens about 10 seconds for a follow-up without the wake word, with the Listening bar showing. | 20 seconds, ending after 1.5 seconds of silence (the earlier design) | Jay, 2026-11-19 | Rulebook note r128; round 2 O-followup |
| S32 | Health advice | Decided | Small advice is allowed: gym supplements, vitamins and diet recommendations. No diagnosis and nothing involving medication; injury or pain gets only "ask a professional". | No medical or supplement advice at all (the earlier rule) | Jay, 2026-11-19 | Rulebook note r206; marks r158, r159 |
| S33 | Local fallback model | Decided | A local Qwen model (4B or 7B) on GYST's server, for an outage or exhausted credit: only if the owner has switched it on, only when really needed, and offered to him explicitly. A short failure gets one retry; an outage, a revoked key, a refusal or an exhausted budget each gets a clear status; a refusal stays a refusal. GYST's normal features keep working. Without it, the brief still arrives without its AI summary (S27). Still to confirm: whether the server can run it (a measured test first; the runtime then needs a Tool Proposal). | No fallback: AI features stop until the cap resets (the earlier rule) | Jay, 2026-11-19 | Rulebook notes r143, a8; round 2 O-nosummary |
| S34 | Usage limits and retention | Decided | 30 questions an hour and 150 a day, adjustable, under the $30 cap. A card expires after 2 minutes. Card status is kept 30 days and change records 180 days. An expired draft's contents are dropped unless saved. | — | Jay, 2026-11-19 | Rulebook note p15; mark r151 |
| S35 | Cards she makes on her own | Decided | Cards the Secretary makes without a request (for example a certificate match) appear in the next brief and on her screen, with no push. Their 2-minute confirm window starts when the owner opens the card. | A push when one is found | Jay, 2026-11-19 | Round 2 O-cards; rulebook items p8, k27 |
| S36 | The off switch | Decided | It stops all new AI calls, pending cards and held sends. | New AI calls only; held sends still go out | Jay, 2026-11-19 | Round 2 O-off; rulebook items p9, b12 |
| S37 | What she says aloud | Decided | Short answers aloud. Health numbers, mail content and other people's details stay on screen unless the owner asks to hear them. | Every reply aloud | Jay, 2026-11-19 | Round 2 O-speak; rulebook items p1, p2 |
| S38 | Languages | Decided | English only in v1. | More languages in v1 | Jay, 2026-11-19 | Round 2 O-lang; rulebook item p10 |
| S39 | The coach rule | Decided | She treats the owner like an adult: she uses his goals, schedule and recent logs to say what she would do and why. If he is pushing too hard or giving himself too little recovery, she says so plainly and suggests an adjustment he can actually use. She labels estimates and says when his logs are incomplete. If a calorie or training target crosses a visible GYST safety limit, she explains the specific concern and offers the closest safer option. Changes to his targets and training plan are his to make. | She warns, then follows his lead | Jay, 2026-11-19 | Round 2 O-safety; rulebook item p7 |

## 8. Mail

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| M1 | Scope of v1 | Decided | Read, search, triage, make tasks, and send and reply. Reading stays read-only at the provider: no delete, move or mark-read there. | Read-only first (the earlier recommendation) | Jay, 2026-11-18 | Answer 16; mail plan §16, §19 |
| M3 | Reply and compose | Decided | In v1. | Not now (the earlier recommendation) | Jay, 2026-11-18 | Answer 16; mail plan §16 |
| M4 | Which inboxes first | Decided | After the calendar: one Gmail inbox, then one Microsoft inbox, then the rest. | All inboxes at once (inferred) | Jay, 2026-11-18 | Answer 18; mail plan §14 |
| M5 | Employer inboxes | Decided | None of the inboxes belongs to an employer. Generic rule kept: an employer-owned inbox would need that employer's agreement first. | — | Jay, 2026-11-18 | Earlier; mail plan §3 |
| M6 | Sync timing | Decided | As M26: Inbox and Sent every 3 minutes, other folders every 30 minutes. | Every 5 minutes | Jay, 2026-11-18 | Answer 17; mail plan §16.5 |
| M7 | What is kept | Decided | All history, and attachments. "All history" is defined in M24. | Inbox only, 90 days, no attachments | Jay, 2026-11-18; defined 2026-11-19 | Answer 17; round 4 M24; mail plan §17, §18 |
| M9 | How email is shown and written | Decided | v1 shows formatted email, embedded photos and image attachments, with an optional plain-text view. Remote images stay blocked until the owner chooses Load images for that message; automatic link previews stay blocked. The owner can compose, reply, reply all and forward, choose the sending account, add To, Cc and Bcc recipients, and attach files or photos. The Secretary can draft; the owner reviews the recipients and attachments before he taps Send, with 20 seconds to undo (M22). Each message shows Sending, Sent, Failed or Bounced; for a bounce, which recipient was affected and why, when available. Sent means the provider accepted it for sending; Delivered shows only when delivery is confirmed. Formatted mail needs a maintained HTML sanitiser (Tool Proposal and security review) and the site's content security policy enforced before it ships. | Text only first, sanitised HTML later (the earlier recommendation) | Jay, 2026-11-19 | Round 3 M9; rulebook mark r213; note b3; mail plan §9 |
| M10 | AI summaries and triage | Decided | Summaries for every group, behind S16, when the owner asks or she needs the context (S4). The daily brief job (S27) is the one scheduled read. No other background AI triage. | Background AI triage | Jay, 2026-11-18; brief job 2026-11-19 | Earlier, amended by S27; clarification Q2; mail plan §14 |
| M12 | Unread state | Decided | Mirrors the provider. Opening a message in GYST clears the dot in GYST only. Sending doesn't change this. | Mark read at the provider (inferred) | Jay, 2026-11-18 | Answer 29; mail plan §14 |
| M13 | Removing an inbox | Decided | Deletes its cached mail and its secret at once and revokes the token. Tasks keep the saved subject and sender. | Keep the cache until it ages out (inferred) | Jay, 2026-11-18 | Answer 29; mail plan §14 |
| M14 | The Secretary and mail | Decided | She reads mail and drafts replies and new messages; the owner sends (M22). Recipients come only from names the owner says or people already in the thread, looked up by GYST, never from an address written inside a message. The send card shows the From account, To, Cc, Bcc, every attachment and every link. It warns on new recipients, anyone outside the thread, an unusually broad reply-all, outbound links, and sensitive or Local-only details. Health, financial or Local-only details, or another email's content, go into a draft only when the owner explicitly asks for that destination. The owner attaches outgoing files and photos himself and reviews them before Send (M9). | Read-only, no drafts | Jay, 2026-11-18; amended 2026-11-19 | Earlier, amended by answer 16; rulebook marks a3, b1; notes a2, b9, p11; round 3 M9; mail plan §19 |
| M15 | Zoho | Decided | A Zoho account connects through IMAP; the owner confirms that the plan includes IMAP and which region the account is in. Server hosts are taken from the provider's documentation and the account's own settings, stored per account, never hard-coded. | The Zoho Mail API | Jay, 2026-11-18 | Answer 19; mail plan §16.6 |
| M19 | Email summary | Decided | Inside the 08:00 morning brief, saying "Nothing major" when nothing needs attention. It replaces a separate daily digest. | A separate daily digest; hourly | Jay, 2026-11-18 | Answer 14; mail plan §19 |
| M20 | Send permissions | Decided | Gmail `gmail.send`; Microsoft `Mail.Send` (plus `Mail.ReadWrite` only if drafts or large attachments are kept at the provider); Yahoo and Zoho through SMTP with the same app password. Scopes follow the vendors' documentation. | `gmail.compose` or `gmail.modify` (restricted and broader) | Jay, 2026-11-18 | Answer 16; mail plan §16.1 |
| M21 | Build or adopt mail | Decided | Build mail directly into GYST on maintained open-source Go libraries and Google's and Microsoft's mail APIs. Inbox search, formatted messages, photo previews, composing with To, Cc and Bcc, attachments and send status stay in GYST. The Secretary uses GYST's own mail lookups and prepares drafts for the owner to review and send. | Adopt an open-source webmail app (those found are copyleft and add a second runtime) | Jay, 2026-11-19 | Rounds 3 and 4 M21; mail plan §16.4 |
| M22 | Send safeguards | Decided | Every send needs Confirm and a password or passkey checked within the last 5 minutes. A 20-second hold, cancellable only during it. At most 10 sends an hour, 30 a day and 5 new addresses a day. Held sends survive a restart, and GYST asks again before sending. The audit holds no addresses. The Secretary never sends. | 20 sends an hour and 100 a day (the earlier recommendation) | Jay, 2026-11-19 | Answers 16, 21; clarification Q3; rulebook marks a4, b7, b14; note b6; mail plan §16.3 |
| M23 | Attachments | Decided | Every attachment is listed on its message. The file is downloaded when the owner opens it and kept in a 5 GB cache. Pinned attachments are kept until he unpins or deletes them; automatic cache eviction never removes them. The Secretary may read supported incoming attachments, certificate PDFs included, after the S16 guard screens them. The owner attaches outgoing files himself (M9). | Download every attachment during sync (inferred) | Jay, 2026-11-19 | Round 4 M23; rulebook notes r33, r191, p11; mail plan §17 |
| M24 | What "all history" means | Decided | Searchable mail text from every synced folder except Spam and Trash, kept for as long as the inbox is connected. Removing the inbox from GYST removes its synced mail history from GYST too (M13). Attachments on demand (M23). | Also download every attachment, or keep Spam and Trash (inferred) | Jay, 2026-11-19 | Round 4 M24; mail plan §18 |
| M27 | Sending GYST's own messages | Decided | A separate outgoing-mail service for GYST's own messages, such as the 10:00 brief email (N10), usable before sending through the owner's accounts (M22) ships. It needs a new provider and a new secret, so a Tool Proposal comes first. Still to confirm: the provider, through that Tool Proposal and the owner's approval. | Wait until mail can be sent through the owner's own accounts | Jay, 2026-11-19 | Round 2 O-briefmail |

## 9. Food

| ID | Question | Status | Decision or recommendation | Alternative | Decider, date | Source |
|---|---|---|---|---|---|---|
| F1 | Food and dish profiles | Decided | A profile holds a name, notes, the dish's items and one photo. | — | Jay, 2026-11-18 | Later; Secretary brain §24.1 |
| F2 | Where a profile photo is kept | Decided | A re-encoded JPEG (at most 800 px and 250 KB, location data removed) inside the main database, so backups include it; served only behind the session; deleted with the profile. The design is as recommended. This changes "photos are never stored" for this one path only. A saved profile photo may be sent to the AI to estimate a home-cooked meal (P1, F10). | A file outside the database (not backed up) | Jay, 2026-11-18; AI use 2026-11-19 | Later; rulebook note r112; Secretary brain §24.1 |
| F3 | One photo per profile | Decided | A new photo for a profile that already has one is not kept unless the owner taps "Use this photo instead". | Keep several photos (inferred) | Jay, 2026-11-18 | Later; Secretary brain §24.1 |
| F4 | Describe a meal in words | Decided | The AI names the foods and grams; GYST's own food data supplies the values; the AI's figures are used only where there is no match, labelled and stored as an estimate; nothing is saved without Save. It ships together with the R1 line in `docs/RULES.md`. | — | Jay, 2026-11-18 | Later; rulebook marks r69, r70; Secretary brain §24.2 |
| F5 | Adjust before logging | Decided | Before logging, the owner can adjust every food in the meal, including raising or lowering its macros. The AI's original figures are kept to measure accuracy. The controls (grams per item, a meal portion from ×0.5 to ×2) follow the recommended design. | — | Jay, 2026-11-18 | Later; Secretary brain §24.3 |
| F6 | The day's context for a meal | Decided | She first identifies the meal from the photo, the name, the ingredients and the owner's answers. Then, in the same conversation, she uses the day's totals and remaining targets to advise how much of it to eat. After Save, a short "how this fits your day" note. | A separate note after Save only (the earlier recommendation) | Jay, 2026-11-19 | Rulebook notes k20, r112; mark r76; Secretary brain §24.4 |
| F7 | What the AI keeps | Decided | Note, not a choice: the AI keeps nothing between calls and is not trained on this data. What it "knows" is what GYST sends each time. GYST learns usual portions itself. | — | — (a fact, recorded 2026-11-18) | Anthropic's data terms; Secretary brain §13, §24.5 |
| F8 | Owner-entered macros | Decided | The owner can type a food's macros himself, against any unit (F9). The AI estimate is used only when he hasn't. His entry always wins, over GYST's food list too. | — | Jay, 2026-11-18 | Later; rulebook marks r67, r68, k18; round 2 O-macro |
| F9 | Macro units | Decided | He enters macros against any base (1 g, 100 g, a serving, a pack, a piece), each with its weight in grams, and logs in any of those units; GYST scales the numbers. Our own design. | Per gram only | Jay, 2026-11-18 | Round 7 choice; rulebook mark r68 |
| F10 | Questions for home-cooked food | Decided | To get as close as possible to the real macros, she asks the questions needed (for example how much oil went into the pot, per big pot or per kg of rice), then advises the portion that fits the day (F6). At most 3 questions per meal, from a starter list: which oil or fat and how much for the whole pot (per big pot, per pan or per kg of rice); how much of the pot is on his plate; hidden extras; how it was cooked; for meat, the cut and skin; whether the oil was drained or skimmed off. She skips what the photo, the name and the profile already answer. | She estimates from the photo alone | Jay, 2026-11-19 | Rulebook note r112; round 2 O-food |
| R1 | Rule change for AI nutrition estimates | Decided | A line in [`docs/RULES.md`](../../RULES.md) §3.2. The owner's own entered nutrition values take priority. Where GYST has no reliable food match, he can review, adjust and save a clearly labelled AI estimate; an unknown nutrient stays NULL. The owner approved the line on 2026-11-19; only the owner changes the rules. | No rule change, so no AI estimate is ever stored | Jay, 2026-11-19 | Rulebook marks r69, r70; note r112; round 4 R1; Secretary brain §24.2 |
| S28 | The Secretary as personal coach | Decided | The Secretary is also the owner's personal coach: one assistant with the health-coach role, reading his history through GYST's lookups. The model is not trained on him; GYST sends what is needed each time, including his About me profile (S29). Everyday coaching runs on Sonnet (S22). Still open: how the existing Coach screen and the Secretary share one assistant. | Two separate assistants (inferred) | Jay, 2026-11-18 | Later; clarification Q4; rulebook note k25 |

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
| Decision list | `decisions/decisions-r7.md` | the status of every ID, updated 2026-11-19 |
| Rulebook | `voice-assistant/secretary-rules-full.md` | the numbered rules (rN), conflicts (kN) and gaps (pN) the marks refer to |
| Rulebook marks | `voice-assistant/rulebook-marks-2026-11-19/marks-report.txt`, `…/clarifications.md`, `…/clarifications-round2.md`, `…/clarifications-round3.md`, `…/clarifications-round4.md` | the owner's marks and notes; Q1–Q4; his answers to the 16 questions (round 2) and his last answers (rounds 3 and 4) |
| Rules review | `voice-assistant/rules-review-2026-11-19.md` | A1–A16 (aN), B1–B14 (bN) |
| Rules v2 | `voice-assistant/secretary-rules-v2.md` | the decided rule set; §4 open questions |

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
- Single events only (C21). Editing a repeating event asks
  "this event" or "the whole series" (C20).
- No adding or removing guests. An event with guests can be changed only when
  the owner organises it, and the card says the guests will be told
  (C20).
- A change proposed by the Secretary is a card the owner confirms (C19), with
  Undo. A deleted event disappears from GYST at once, and the provider delete
  waits a short time (10-minute undo and 2-minute hold, C19).
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
