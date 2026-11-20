# Stage 7 — Design Decisions, part B: the makeover

- **Decided by:** Jay (product owner)
- **Recorded:** 2026-11-20 (Issue #255, makeover chunk 0)
- **Responds to:** the makeover design, rounds 1 to 7 (2026-11-16 to 2026-11-18). Jay approved the round 7 design and gave the go to build it on 2026-11-20.
- **Status:** **Binding** for all makeover work. Where this file differs from `13A`, `11` or `13`, this file wins. Everything it doesn't mention stays as those files say.
- **Amends:** `13A` DD-1 and DD-4. `13A` itself is not rewritten; DD-2 and DD-3 are unchanged.
- **Does not decide:** any Galaxy visual or performance work ([Galaxy](#galaxy)), or anything listed under [Open](#open).
- **Implementation:** none. No production code, CSS, templates or data changed.
- **Privacy:** every value here is a design value or fictional. The sources are on the owner's server, outside this repository ([Sources](#sources)).

---

## How to read an entry

Each entry gives the **old** value (with the section it came from), the **new** value, **why**, and the **source**: Jay's dated answer, or the design round that settled it. Jay's answers win over every design document.

Source short names:

| Short name | Meaning |
|---|---|
| Answer n | Jay's numbered answer of 2026-11-18 (`decisions/jay-answers-2026-11-18.md`) |
| UI1 | Jay's round 7 choice of 2026-11-18 on the fourth tab (same file; `decisions/decisions-r7.md`) |
| Round n | The design round's assessment (`design-makeover/assessment*.md`), approved as part of the round 7 design on 2026-11-20 |
| Direction | `design-makeover/direction.md`, rounds 1 and 2 (2026-11-16) |

## Summary

| # | Topic | Decision | Supersedes | Source |
|---|---|---|---|---|
| DD-5 | Appearance setting | One setting holds Theme, Look, Accent, Cards, Density, Text size and Navigation | 13A DD-1 (extended); 11 §J | Direction; round 6; answer 4 |
| DD-6 | Accent | One accent, chosen by Jay from seven; violet is no longer the fixed identity colour; never good, warning or bad | 13A DD-4 (revised); 11 §A, §H | Direction; round 7; lead, 2026-11-20 (fixed values) |
| DD-7 | Looks | Three: Aurora (blue, default), Lagoon (green), Sunrise (orange) | 11 §F, §H | Answer 4; round 7 |
| DD-8 | Section colours | Six, one per launcher group | new | Round 6 |
| DD-9 | Typography | Geist, eight sizes on one scale factor, three weights | 11 §C | Direction |
| DD-10 | Radii | 8 · 12 · 20 · 28 · pill | 11 §E | Direction |
| DD-11 | Spacing and density | Same seven steps; tablet gutter 28; Comfortable and Compact | 11 §D | Direction |
| DD-12 | Elevation, glass and card styles | Inner hairline, one floating shadow, blur 20, three card styles | 11 §G | Direction; round 6 |
| DD-13 | State colours | Fixed values; "info" uses the accent; state text is checked per Look | 11 §I | Direction; round 6 |
| DD-14 | Identity marks | Six project colours; one shape per kind of thing | 11 §K, §N Project Row | Direction; rounds 4 to 7 |
| DD-15 | Icons | One inline SVG set, 1.75 stroke | 11 §K | Round 3 |
| DD-16 | Motion | 120 · 220 · 380 ms | 11 §L; 13 §18 | Direction |
| DD-17 | Breakpoints and split view | 700 and 1000; split view replaces the inspector | 13 §1, §3, §7; 11 App Shell | Direction |
| DD-18 | Phone tab bar | Floating pill: Today · Calendar · Mail · Tasks · Health · All, plus + | 11 Mobile Tab Bar; 13 §2, §13 | Round 6; answer 3 |
| DD-19 | Tasks hub | The fourth tab opens Tasks · Projects · Galaxy | 13 §5 | Answer 3; UI1; round 7 |
| DD-20 | All pages (launcher) and pins | Replaces More; six groups; Jay pins and orders pages | 11 Mobile Tab Bar; 13 §2 | Round 6; answer 5 |
| DD-21 | Desktop sidebar | 248 wide, Pinned, six groups, 68 rail | 11 Desktop Sidebar; 13 §4, §10 | Direction; round 6 |
| DD-22 | Health navigation | The main tab bar stays; four icon sections | 11 Secondary navigation; 13 §5 | Rounds 1 and 3 |
| DD-23 | Phone Today | New colours; layout frozen; events in the existing Now and Later rows | none (confirms the layout) | Answers 1 and 2; round 7 |
| DD-24 | Core components | Buttons 44, inputs 48, chips, stat tile, hero metric, sheet | 11 §N controls, Sheet; 13 §6 | Direction |
| DD-25 | Task row, task sheet and + | Chip sheet replaces the long form; + on every screen | 11 Task Row, Quick Capture; 13 §12 | Direction; rounds 3 and 7 |
| DD-26 | Settings | Grouped list with search and state rows | new (12 has no grouped Settings) | Direction; rounds 3 to 7 |
| DD-27 | Secretary surfaces | The assistant is labelled Secretary; one overlay; a Listening pill | new | Rounds 3 and 7; answers 20, 21 and 25 |
| DD-28 | Screens added in rounds 3 to 7 | The round 7 mockup is their design reference | new | Rounds 3 to 7 |

---

## Amended 13A entries

| 13A entry | Effect | Where |
|---|---|---|
| DD-1 Default appearance | **Extended.** System · Light · Dark stays exactly as written. The same single setting gains more options. | DD-5 |
| DD-2 Today Rhythm strip | **Unchanged.** Still to be implemented: Today still shows three large prayer sections. | — |
| DD-3 XP on Today | **Unchanged.** A small "Lv" chip in the header. | — |
| DD-4 Violet identity accent | **Revised.** There is one accent and Jay chooses it. The rule behind DD-4 is kept: the accent never means a health or status state (good, warning or bad). Info and active (Working, Reviewing) are drawn in the accent, with their word and icon. Section and project colours never mean any state. | DD-6, DD-13 |

13A's "Implementation discipline" section still applies. This file is the "recorded decision" it asks for.

## DD-5 — Appearance setting (amends 13A DD-1)

| | |
|---|---|
| **Old** | One setting, System · Light · Dark (13A DD-1). A fixed palette. |
| **New** | The same single setting, Settings › Appearance, now holds: **Theme** (System · Light · Dark, unchanged), **Look** (DD-7), **Accent** (DD-6), **Cards** (DD-12), **Density** (DD-11), **Text size** (DD-9) and **Navigation** (pins, DD-20). |
| **Storage** | On the device. Applied before first paint, as the theme is today, so there is no flash of the default. The page must work when storage is blocked. |
| **Greeting name** | The launcher greets Jay by a name he types. Empty by default, at most 24 characters, stored on the device only. |
| **Why** | Jay asked for a more personal and colourful app (round 6) and wants to choose the colours himself in Settings (answer 4). |
| **Source** | Direction (accent, density, text size); round 6 (Look, cards, pins, name); answer 4. |

Still true from DD-1: one setting for the whole app, no per-area appearance settings, live reaction to the OS in System mode.

## DD-6 — Accent (amends 13A DD-4)

| | |
|---|---|
| **Old** | A fixed blue `--accent` (#2563eb / #6ea8ff) for action, and violet `--xp` / `--xp-soft` as the identity accent (13A DD-4; 11 §A, §H). |
| **New** | **One accent, chosen by Jay** from seven: Iris, Blue, Teal, Volt, Orchid, Ember, Graphite. It means "act" or "selected", and it is the colour of the info and active state (Working, Reviewing; DD-13). `--xp` and `--xp-soft` merge into `--accent` and `--accent-soft`. |
| **Default** | Each Look starts with its own accent (DD-7): Aurora with Iris, Lagoon with Teal, Sunrise with Ember. Any of the seven works on any Look. |
| **Never good, warning or bad** | The accent never means a health or status state: not success, healthy, warning, error, blocked, destructive or needs attention. Info and active is the one state drawn in it, always with its word and a dot or icon. No pure green, amber or red accent is offered. Volt and Ember sit closest to state colours, so a state always carries a dot or icon and a word (DD-13). |
| **Why** | One colour for "act" is easier to read than blue for action and violet for identity, and Jay wanted to pick it. |
| **Source** | Direction, "Changes to recorded decisions" ("Info uses the accent"); round 7 (the seven accents stay); lead, 2026-11-20 (the wording of the state rule, following Direction). |

Each accent has four variants per theme. These are the audited values, and they are **binding as fixed values**. R1 ships them as written. The generator does not derive them from hue seeds; it asserts the contrast pairs below and fails the build on a value that breaks one. Direction described each accent as a stored seed (a hue and a strength), but no round recorded seed numbers, so re-deriving the hues is not required. Decided by the lead on 2026-11-20; it follows the round 7 drawings Jay approved.

| Accent | Light `--accent` | Light `--accent-text` | Light `--accent-soft` | Dark `--accent` and `--accent-text` | Dark `--accent-soft` |
|---|---|---|---|---|---|
| Iris | #6163F2 | #4842D0 | #EDEFFD | #96A2FE | #1F2343 |
| Blue | #306FEC | #104FCA | #E9F1FD | #87B1FD | #162644 |
| Teal | #0F828D | #06626B | #D6F8FB | #44D4E2 | #032C31 |
| Volt | #5A8209 | #436202 | #E9F5DC | #B2F340 | #1E2C07 |
| Orchid | #B149C2 | #9126A1 | #FCE9FE | #EA7FFB | #341C38 |
| Ember | #C4530A | #963D03 | #FEECE4 | #FE8F5B | #3E1B0B |
| Graphite | #2C2E31 | #57595D | #EEF0F4 | #DCDEE2 | #252729 |

`--on-accent` is #FFFFFF in light and #0B0D12 in dark.

Guarantees the build must assert, failing if one doesn't hold:

| Pair | Minimum |
|---|---|
| `--on-accent` on `--accent` | 4.5:1 |
| `--accent` against `--surface` | 3:1 |
| `--accent-text` on every neutral of every Look (canvas, surface, surface-2, elevated) | 4.5:1 (7:1 on surface is the design target) |
| `--accent-text` on its own `--accent-soft` | 4.5:1 |

The lowest measured pair is 4.53:1, white on Volt in light. Volt is the bright lime only in dark; in light it is a deep green so text stays readable.

What 13A DD-4 still means after this change:

- XP and level use the accent. They stay small and quiet (13A DD-3).
- Status chips, badges, banners, Attention severity and state dots never use the accent to mean success, warning, error, blocked, destructive or needs attention.
- Info and active states (Working, Reviewing, Queued, Syncing) use the accent, with their word and icon (DD-13). This follows 13A DD-4's own ruling that "Reviewing" is an active state drawn in the info colour like "Working"; the info colour is now the chosen accent instead of the fixed blue.
- One consequence is open: when the chosen accent is Iris or Orchid, an active state is drawn in a violet, which 13A DD-4 and `AGENTS.md` forbid in their current words (O-12).

## DD-7 — Looks

| | |
|---|---|
| **Old** | One set of cool navy neutrals (11 §F, §H). |
| **New** | Three Looks. Each is a set of neutrals, section colours and state-text values, in light and dark. **Aurora** is the default; **Lagoon** and **Sunrise** are the other two. Jay picks in Settings › Appearance › Look. |
| **Not graphite** | The default neutrals are **Aurora's blue-violet** (hue 248), not graphite. Direction's graphite neutrals (#F3F4F6, #08090B and the rest) were rounds 2 to 5; they survive only as the declined Instrument Look and are not shipped. With no Look chosen, the app shows Aurora. Where an Issue or plan says "graphite", this entry wins. |
| **Declined** | The round 6 Looks **Instrument** (the round 5 graphite palette) and **Court** (near-white and true black) are removed. |
| **Why** | Jay liked all five and asked to keep one blue, one green and one orange, and to choose after the deploy (answer 4). The default until he chooses is blue. |
| **Source** | Answer 4 (2026-11-18); round 7. |

| Look | Role | Neutral hue | Starts with accent | Character |
|---|---|---|---|---|
| Aurora (default) | the blue Look | 248 | Iris | Cool blue-violet neutrals |
| Lagoon | the green Look | 188 | Teal | Blue-green neutrals |
| Sunrise | the orange Look | 32 | Ember | Warm cream neutrals |

Neutrals, light:

| Token | Aurora | Lagoon | Sunrise |
|---|---|---|---|
| `--canvas` | #F1EFF9 | #F0F7F8 | #FAF4EF |
| `--surface` | #FFFFFF | #FFFFFF | #FFFFFF |
| `--surface-2` | #EAE9F4 | #E9F2F4 | #F5EFE8 |
| `--surface-3` | #DEDCEC | #DDEAEC | #EEE5DB |
| `--elevated` | #FFFFFF | #FFFFFF | #FFFFFF |
| `--ink` | #0D0C16 | #0C1416 | #16110C |
| `--ink-2` | #4D4961 | #425457 | #594F43 |
| `--ink-3` | #646277 | #5A6B6D | #6F655B |
| `--separator` | #DEDCEC | #DDEAEC | #EEE5DB |
| `--glass` | white at 74% | white at 74% | white at 74% |

Neutrals, dark:

| Token | Aurora | Lagoon | Sunrise |
|---|---|---|---|
| `--canvas` | #09080F | #070D0E | #0D0C0A |
| `--surface` | #13111F | #0F1D1F | #1C1814 |
| `--surface-2` | #1B182D | #16292C | #28231D |
| `--surface-3` | #25223A | #1F373A | #352E27 |
| `--elevated` | #1C192E | #172B2E | #29241E |
| `--ink` | #F3F2F5 | #F2F5F5 | #F5F4F2 |
| `--ink-2` | #A7A5B6 | #A8B6B8 | #B6AEA5 |
| `--ink-3` | #8B899A | #89989A | #999187 |
| `--separator` | white at 9% | white at 9% | white at 9% |
| `--glass` | the Look's `--surface` at 70% | same | same |

Two more neutrals are the same in every Look: `--hairline` is #0B0D12 at 8% in light and white at 7% in dark; `--scrim` is #0B0D12 at 32% in light and black at 55% in dark.

**Looks are generated, not hand-picked.** The generator takes one seed per Look and searches lightness until every text and fill pair clears its target, then asserts each pair. The seeds:

| Seed | Aurora | Lagoon | Sunrise |
|---|---|---|---|
| Neutral hue | 248 | 188 | 32 |
| Neutral saturation, light / dark | 0.42 / 0.30 | 0.40 / 0.34 | 0.50 / 0.16 |
| Dark canvas lightness | 0.045 | 0.04 | 0.045 |
| Section saturation (System is 0.10 in every Look) | 0.82 | 0.78 | 0.80 |
| Section hues (Plan, Work, Connect, Health, Learn, System) | 222, 16, 268, 335, 172, 248 | 215, 24, 256, 332, 165, 192 | 206, 12, 282, 346, 174, 30 |

The generator asserts 714 pairs across the three Looks, with all seven accents on each. Search targets are 4.7:1 for text and 7:1 for strong text; the assertion is AA (4.5:1 text, 3:1 fills). See [Token tools](#token-tools-for-r1).

## DD-8 — Section colours

| | |
|---|---|
| **Old** | None. Navigation used the accent only. |
| **New** | Six section colours, one per launcher group (DD-20): **Plan** blue, **Work** orange, **Connect** violet, **Health** pink, **Learn** teal, **System** grey. |
| **Used on** | Launcher tiles, the current tab, sidebar rows, the date line above a page title, small header icons and chart lines. |
| **Never** | A state. A section colour says where you are, never how something is doing. Violet is now the Connect section's colour; as a section colour it never means a state. |
| **Why** | Jay asked for a more colourful app where every place is easy to find (round 6). |
| **Source** | Round 6; carried into round 7 unchanged. |

Each section has three tokens per theme: `--s-<name>` (fill, with white text or icon in light and #0B0D12 in dark), `--s-<name>-t` (text) and `--s-<name>-s` (soft tint). In dark the fill and the text are the same value.

Light (fill / text / soft):

| Section | Aurora | Lagoon | Sunrise |
|---|---|---|---|
| plan | #336AEB / #1756E8 / #E4ECFE | #1C6FE3 / #1A65CF / #E5EFFD | #1577C1 / #146EB3 / #E4F3FE |
| work | #CC4514 / #B73E12 / #FEEBE4 | #B85717 / #AA5115 / #FDEFE5 | #D33D17 / #BE3715 / #FEE9E4 |
| connect | #9548ED / #7917E8 / #F0E4FE | #7B53EA / #511CE3 / #EBE5FD | #B130E8 / #A218DC / #F6E4FE |
| health | #DF166A / #C5145E / #FEE4EF | #DC1B75 / #C51869 / #FDE5F0 | #E11948 / #CA1640 / #FEE4EA |
| learn | #0D8272 / #0B7466 / #E4FEFA | #108165 / #0F785E / #E5FDF7 | #0E8175 / #0D776D / #E4FEFB |
| system | #747089 / #67647A / #EBEAF1 | #64767A / #5C6C70 / #EAEFF1 | #7D7166 / #72675D / #F1EDEA |

Dark (fill and text / soft):

| Section | Aurora | Lagoon | Sunrise |
|---|---|---|---|
| plan | #6A95F9 / #18243F | #5E9DF5 / #19283E | #40A6F5 / #182E3E |
| work | #F7784A / #3F2218 | #F28338 / #3E2819 | #F67655 / #3E2018 |
| connect | #B578F9 / #2A183F | #A78AF8 / #23193E | #CF70F7 / #33183E |
| health | #F965A2 / #3F1828 | #F66AAB / #3E192A | #F7698A / #3E1821 |
| learn | #23F6DA / #183F3A | #27F1BF / #193E35 | #25F4DF / #183E3B |
| system | #A6A2C1 / #313036 | #92AFB7 / #303536 | #B8A694 / #363330 |

## DD-9 — Typography

| | |
|---|---|
| **Old** | System font stack, no web fonts. Seven sizes. Weights from 400 to 700, including 550 and 650. Display 34, title 28 (11 §C). |
| **New** | **Geist Variable**, one self-hosted WOFF2 (about 57 KB, SIL Open Font License 1.1) under `web/static/fonts/`, its licence under `web/static/licenses/`. The system stack is the fallback. No request leaves the server. |
| **Weights** | 400, 500 and 600 only. |
| **Scale factor** | Every size is multiplied by one factor, `--ts`: **iOS** 1 (follows the iPhone text-size setting), **S** 0.92, **M** 1, **L** 1.12. |
| **Case** | Sentence case everywhere, section labels included. No ALL-CAPS and no uppercase section labels. |
| **Why** | The stylesheet named a font that never shipped, so the face differed by device; 26 sizes and 7 weights were live (round 1 audit). Fixed pixel sizes ignored the iPhone text-size setting. |
| **Source** | Direction. |

| Token | Size / line | Weight and tracking | Use | Was (11 §C) |
|---|---|---|---|---|
| `--t-display` | 56 / 56 | 600, −0.05em | One hero metric per screen | 34 / 40, 700 |
| `--t-metric` | 26 / 30 | 600, −0.035em | Stat tile numbers | new |
| `--t-title` | 30 / 34 | 600, −0.03em | Page title | 28 / 32, 700 |
| `--t-head` | 20 / 25 | 600, −0.02em | Now line, sheet and task titles | `--t-section`, 650 |
| `--t-body` | 16 / 24 | 400; row titles 500; project names 600 | Rows, inputs | 16 / 23; row titles 550 |
| `--t-sub` | 14 / 19 | 400–500 | Secondary values, small buttons | 400–550 |
| `--t-foot` | 13 / 18 | 400–500 | Meta, section labels | 400–600, uppercase labels |
| `--t-cap` | 12 / 16 | 500 | Chips, tile labels | `--t-caption`, 500–650 |

Tab labels are 10.5 px on the six-tab phone bar (DD-18). Input text is never under 16 px at any scale factor (`max(16px, …)`), so iPhone does not zoom the page on focus. Numbers that align keep `font-variant-numeric: tabular-nums`.

## DD-10 — Radii

| Token | Old (11 §E) | New | Use |
|---|---|---|---|
| `--r-sm` | 8 | **8** | Tiles |
| `--r-md` | 12 | **12** | Buttons, inputs, segmented track; project tile (40 px) |
| `--r-lg` | 16 | **20** | Cards, grouped lists, stat tiles |
| `--r-xl` | 24 | **28** | Sheets |
| `--r-pill` | 999 | **999** | Chips, tab bar |

Why: larger, softer containers against the hairline edges. Source: Direction.

## DD-11 — Spacing and density

| | |
|---|---|
| **Old** | Seven steps, 4 to 48. Page gutter 16 phone · 24 tablet · 32 desktop. Rows 56, compact 48 (11 §D). |
| **New** | The same seven steps: **4 · 8 · 12 · 16 · 24 · 32 · 48**. Page gutter **16** under 700, **28** from 700, **32** from 1000. |
| **Density** | A setting. **Comfortable:** rows 56, padding 16, section gap 24. **Compact:** rows 48, padding 12, section gap 16. No target drops under 44 px in Compact. |
| **Why** | Density lets Jay see more rows on the phone without a second layout. |
| **Source** | Direction. |

## DD-12 — Elevation, glass and card styles

| | |
|---|---|
| **Old** | `--e1` hairline shadow on lists, `--e2` and `--e3` for popovers and sheets. Blur 14–16 with saturate 1.6 on three sticky bars (11 §G). |
| **New: edges** | Cards and lists have **no shadow**, only a 1 px inner hairline (`inset 0 0 0 1px var(--hairline)`). |
| **New: floating** | One floating shadow, `--e-float`, for the tab bar, +, toasts and the command bar. Light: `0 10px 30px` #0B0D12 at 12% plus `0 1px 2px` at 6%. Dark: `0 16px 40px` black at 55%. |
| **New: glass** | Tab bar, sticky top bar and the Listening pill only. `--glass` (DD-7) with `backdrop-filter: saturate(1.8) blur(20px)`. Under `prefers-reduced-transparency: reduce` it is solid `--elevated` with no filter. |
| **New: card styles** | A setting with three values. **Hairline** (default): as above. **Lifted:** adds `0 1px 2px` #0B0D12 at 5% and `0 10px 28px` at 8%. **Tinted:** the card's surface mixed with 5% of its section colour, and a hairline of the section colour at 22%. |
| **Why** | Depth from tone and a hairline reads calmer than shadows; only things that float get a shadow. |
| **Source** | Direction (hairline, blur 20); round 6 (card styles), audited again in round 7. |

## DD-13 — State colours

| | |
|---|---|
| **Old** | ok / info / warn / crit / idle, with "info / active" drawn in the fixed blue accent (11 §I). |
| **New** | The graphic values are unchanged and **do not change with the accent or the Look**: ok #16A34A / #4ADE80, warn #D97706 / #FBBF24, crit #DC2626 / #F87171 (light / dark). Soft backgrounds: warn #FDF1E1 / #33260C, crit #FDEAEA / #3A1618. |
| **Info / active** | Uses the chosen accent (`--accent`, `--accent-text`, `--accent-soft`), always with its word and a dot or icon. It is the only state drawn in the accent (DD-6). |
| **State text** | Checked per Look. Dark is the same in every Look: ok #6EE7A0, warn #FCD062, crit #FCA5A5. Light is in the table below. |
| **Rule kept** | State is never colour alone. Healthy stays quiet. |
| **Why** | Green "on track" text measured 4.31:1 on grey tiles in round 5, under AA. The generator now darkens state text until it passes on every neutral of the Look. |
| **Source** | Direction ("info uses the accent"); round 6 (state text). |

| Light | Aurora | Lagoon | Sunrise |
|---|---|---|---|
| `--ok-text` | #137637 | #147B3A | #147939 |
| `--warn-text` | #A64C07 | #AC4F07 | #AC4F07 |
| `--crit-text` | #B91C1C | #B91C1C | #B91C1C |

## DD-14 — Identity marks

| | |
|---|---|
| **Old** | A project is a "world orb" in lists (11 §K, §N Project Row). No rule for accounts, events or goals. |
| **New** | Each kind of thing has its own shape, so none can be mistaken for another with colour vision off. In lists a project is a **glyph tile**. The orb stays in Galaxy. |
| **Why** | Projects and tasks looked alike, and rounds 4 to 7 added accounts, events, goals and milestones. |
| **Source** | Direction (project and task); round 4 (account); round 5 (event, free time, house rule, prayer); round 6 (goal, launcher tile); round 7 (milestones, nutrition sources). |

| Thing | Mark |
|---|---|
| Project | Square glyph tile, 40 px, radius 12, filled with 20% of the project colour on the surface, with a 1.5 px ring in that colour and the project's glyph. Small size 28 px, radius 8. |
| Task | Hollow check circle, 24 px drawn in a 44 px target |
| Task with no project | A hollow square and the words "No project" |
| Account (inbox, calendar account) | Round tinted badge with an icon |
| Calendar event | A vertical colour bar beside its time; in grids, a tinted block with that bar on its left edge |
| Free time | Dashed outline, with the word "Free" and the length |
| House rule | Diagonal hatching, with the rule's name |
| Prayer | Dotted line with the name and time |
| Goal | Six-sided mark |
| Roadmap milestone | Course: circle. Project: square. Certification: badge shape. |
| Launcher tile | Solid section colour (a project tile is tinted with a ring) |
| Nutrition number source | Your numbers: solid. GYST food data: outlined. AI estimate: dashed. Each with its words. |

Six **project identity colours**, in this fixed order. They are identity only and never mean a state. A tile always carries a glyph and the project name, so colour is never the only cue.

| # | Name | Light | Dark |
|---|---|---|---|
| `--p1` | blue | #2A78D6 | #3987E5 |
| `--p2` | orange | #EB6834 | #D95926 |
| `--p3` | aqua | #1BAF7A | #199E70 |
| `--p4` | yellow | #EDA100 | #C98500 |
| `--p5` | magenta | #E87BA4 | #D55181 |
| `--p6` | violet | #6250D6 | #9085E9 |

Inbox and calendar colours reuse these six plus grey. No new colour tokens.

## DD-15 — Icons

| | |
|---|---|
| **Old** | Line icons, 1.8 stroke (11 §K). |
| **New** | One set of inline SVG symbols, **1.75 stroke**, round caps and joins, no fill, `currentColor`. Used with `<use>` from one same-origin sprite. No icon font. |
| **The set** | The sprite in `design-makeover/mockup-r7.html`: **93 symbols**, including `i-mail` and `i-all` for the six-tab bar. Not round 3's 62. R1 may ship only the symbols in use, in round 7's sprite format (`<symbol id="i-…" viewBox="0 0 24 24">`); later chunks add theirs from the same set and none from outside it. |
| **Sizes** | 16 inline and chips · 20 rows and buttons · 22 tab bar · 24 large. |
| **Why** | Jay asked for more icons (round 3); one sprite keeps them consistent and inside `script-src 'self'` and `img-src 'self'`. |
| **Source** | Round 3. |

## DD-16 — Motion

| Token | Old (11 §L) | New | Use |
|---|---|---|---|
| `--fast` | 150 ms | **120 ms** | Press (scale 0.97; the + button 0.94), hover |
| `--base` | 240 ms | **220 ms** | Standard transitions, centred sheets |
| `--sheet` | — | **380 ms** | Bottom sheet, with `--ease-sheet` |
| `--ease` | `cubic-bezier(.2,.8,.2,1)` | unchanged | |
| `--ease-sheet` | — | `cubic-bezier(.32,.72,0,1)` | |

With reduced motion everything collapses to about 1 ms and the Secretary's waveform is static. Motion animates transform and opacity only. Galaxy's own timings (flight, pulse) are not changed here.

Why: shorter presses feel more direct; a sheet needs its own longer curve. Source: Direction.

## DD-17 — Breakpoints and split view

| | |
|---|---|
| **Old** | Layout breakpoints at 600, 1024, 1280 and 1600. An inspector from 1280 (13 §1, §3, §7; 11 App Shell). |
| **New** | **Two** layout breakpoints, **700** and **1000**, measured on the app's own width. One set of templates. |
| **Split view** | From 1000, list screens that have a detail (Projects, Settings, Mail) show the list on the left, `minmax(320px, 380px)`, and the detail on the right with a 28 px gap. The selected row is tinted and the back link is hidden. Under 1000 the same detail is a pushed page with a back link. **The inspector at 1280 is replaced by this.** |
| **Why** | The iPad in portrait and landscape are the real middle sizes; one rule for "list beside detail" is simpler than a separate inspector tier. |
| **Source** | Direction; round 4 (Mail split). |

| App width | Devices | Navigation | Content | Detail opens as | Sheets |
|---|---|---|---|---|---|
| Under 700 | iPhone | Floating tab bar and + (DD-18) | One column, gutter 16 | A pushed page with a back link | Bottom sheet, max 88% height |
| 700–999 | iPad portrait | The same tab bar, centred, max 480 wide | Two columns on Today, Nutrition and Progress; lists centred, max 720 (Calendar and the launcher 760); gutter 28 | A pushed page | Centred card, 560 wide |
| 1000 and up | iPad landscape, desktop | Sidebar 248 (DD-21) | Max 1180, gutter 32, two columns | Split view | Centred card, 560 wide; the command bar drops from the top, 640 wide |

Rules:

- The tab bar and the sidebar never show together.
- Touch targets stay 44 px at every width, because iPads are touch.
- The mockup has two content tweaks at 1280 (the desktop home grid and a 400 px Mail list). They are not shell breakpoints.
- Makeover screens are browser-checked at **390, 834, 1194 and 1440** wide, in light and dark. `AGENTS.md` rule 9 (390 and 1280) still applies as the minimum.

## DD-18 — Phone tab bar

| | |
|---|---|
| **Old** | A fixed bar of five tabs, Today · Projects · Health · Learn · More, 56 high plus the safe area; a badge on Today only (11 Mobile Tab Bar; 13 §2, §13). In production, Health replaced it with its own bar. |
| **New: order** | **Today · Calendar · Mail · Tasks · Health · All**, then a separate **+**. |
| **New: shape** | A floating glass pill (DD-12), 60 high, 4 px inner padding, 12 px from the left, right and bottom edges plus the safe area. Under 700 it fills the width; from 700 it is centred, max 480. |
| **Tabs** | Six. Each is 49 × 52 px at 390 wide (measured), min 44 wide. Icon 22, label 10.5 px, weight 500. The current tab takes its section's colour: text `--s-<name>-t` on `--s-<name>-s`. |
| **+** | A 56 px round button in the accent, 8 px to the right of the pill, with `--e-float`. Tap opens the capture sheet (DD-25). Holding it talks to the Secretary (DD-27). |
| **Mail count** | The Mail tab shows its unread count. |
| **All** | Always the last tab. It opens the launcher (DD-20) and can't be removed, so every page is always reachable. |
| **Never** | Replaced by a section's own bar. Health keeps the main tab bar (DD-22). |
| **Why** | Jay asked for the main tab bar to be reordered for fast access to every page (round 6). The order is the day first (Today, Calendar), then what arrives (Mail), then what to do (Tasks), then Health. |
| **Source** | Round 6; answer 3 (the fourth tab is Tasks). |

Declined: a seventh tab (at 390 wide seven tabs are 43 px each, under the 44 px minimum), and a planet button in the tab bar (it would take the place of All or +). Both in round 7.

## DD-19 — Tasks hub

| | |
|---|---|
| **Old** | Projects was a primary tab with the areas Projects · Development · Galaxy · Roadmaps (13 §5). All tasks was a page under it. |
| **New** | The fourth tab is **Tasks**. It opens one screen whose first control is a three-way switch: **Tasks · Projects · Galaxy**. Galaxy carries the same switch, top right, to step back. |
| **Always Tasks** | The tab always opens on Tasks. It does not remember the last view (UI1). |
| **Reach** | From the Tasks tab, Projects or Galaxy is one tap. From anywhere else, two. Jay can pin Projects or Galaxy to a tab for one tap (DD-20). |
| **Task list** | Every row says its project (a small colour square and the name) or **No project** (a hollow square and the words). Filter chips: All · In a project · No project, each with its count. A By date / By project switch. Late tasks get their own group at the top. |
| **Why** | Jay chose Tasks for the fourth tab, asked for Projects or the planet view to stay one tap away, and said the task list holds tasks with and without a project (answer 3). |
| **Source** | Answer 3 (2026-11-18); UI1 (2026-11-18); round 7. |

Declined: Projects as the fourth tab (answer 3), and a tab that remembers the last of the three views (UI1; a tab labelled Tasks that opens Galaxy would surprise).

The hub is navigation only. It keeps the Projects list a separate view and Galaxy its peer, as `17A` §C.6 and §M require, and it changes nothing inside the Galaxy scene ([Galaxy](#galaxy)). No existing way into Galaxy is removed before the hub ships (`17A` §C.7).

## DD-20 — All pages (launcher) and pins

| | |
|---|---|
| **Old** | A More tab with a list of rows (11 Mobile Tab Bar; 13 §2). |
| **New** | **All pages**, the launcher, replaces the More list. A greeting, a search field, a Pinned row, then every destination as a coloured tile in six groups: **Plan, Work, Connect, Health, Learn, System** (DD-8). Tiles are 84 × 104 px on phone and carry a live count where one exists. |
| **Search** | Finds pages by name and by what was merged into them: "month" finds Calendar, "reminders" finds Tasks. |
| **Pins** | One ordered list. Positions 1 to 5 are the phone tab bar; the rest lead the launcher and the sidebar. Between 5 and 8 pins. Edited from the launcher's Edit or Settings › Appearance › Navigation, with up, down and unpin buttons. Stored on the device (DD-5). |
| **Default pins** | Today, Calendar, Mail, Tasks, Health, then Projects and Secretary. |
| **Merged pages** | Daily checklist becomes a row on Today. Reminders become a reminder chip on a task. The monthly view becomes Calendar › Month. Roadmaps leave the Projects area; they are a larger feature of their own (see below). |
| **Depth** | No page is more than two taps from anywhere on the phone. |
| **Why** | Jay asked for every page to be close and easy to reach (round 6), and accepted the merges (answer 5). |
| **Source** | Round 6; answer 5 (2026-11-18). |

Roadmaps: answer 5 accepted the merge but made roadmaps larger than a row of milestones under a goal. The Secretary builds a roadmap from a goal and tracks it; what it does is recorded in `19A` (S25). Its screens are in DD-28.

## DD-21 — Desktop sidebar

| | |
|---|---|
| **Old** | 232 wide (256 from 1600). Brand, ⌘K field, five primary items, an "Operate" group, More (11 Desktop Sidebar; 13 §4). Shown from 1024. |
| **New** | Shown from **1000**. **248** wide, full height. ⌘K field, a New button, a **Pinned** section, the six groups as coloured rows that open, **All pages**, then Secretary and Settings at the bottom. Every item has an icon. |
| **Rail** | A button, or the `[` key, collapses it to a **68 px** icon rail. |
| **Current page** | Its section's soft colour and text colour, with a bar on the left. |
| **Why** | The same places as the phone, in the same groups, with room for more of them. The old sidebar also stopped at 800 px while the page went on (round 1 audit). |
| **Source** | Direction (248, New, Settings); round 3 (icons, Secretary); round 6 (Pinned, groups, rail). |

Keyboard, replacing the short list in 13 §4 and adding to 13 §10. Shortcuts do nothing while typing in a field.

| Key | Action |
|---|---|
| ⌘K | Command bar |
| N | New |
| ⌘, | Settings |
| ⌘↵ | Save |
| Esc | Close the top layer |
| `[` | Collapse or open the sidebar |
| Hold Space | Talk to the Secretary |
| G then T / C / M / D / H / A | Today / Calendar / Mail / Tasks / Health / All pages |
| G then P / X / S / B | Projects / Galaxy / Secretary / Morning brief |
| G then N / W / R / G / L / V | Nutrition / Training / Progress / Goals / Courses / Achievements |

## DD-22 — Health navigation

| | |
|---|---|
| **Old** | Spec: one segmented control, Summary · Nutrition · Training · Progress · Plan (11 Secondary navigation; 13 §5). Production: a Health-only tab bar replaced the main one, and some screens showed two Health navigations. |
| **New** | The main tab bar stays on every Health screen. Exactly one section switcher under the title, with **four icon tabs: Summary · Nutrition · Progress · Training**, 44 high, fitting 390 px without scrolling. **Plan** opens from a row on Summary. |
| **Why** | Entering Health hid Today, Projects and the rest behind a wordmark tap, the highest-impact problem in the round 1 audit. Five text tabs scrolled off at 390 px. |
| **Source** | Round 1 audit and Direction (one switcher, main tab bar stays); round 3 (four icon tabs). |

## DD-23 — Phone Today

| | |
|---|---|
| **Old** | The Today layout Jay approved in round 2 and froze in round 3. |
| **New: colours** | The new colours apply to phone Today: the Look's neutrals, the section colour on the date line and small icons, the six-tab bar, and the chosen accent. Card style follows Jay's Cards setting. |
| **New: events** | The next calendar event sits on the existing "Now" label line. Upcoming events join the existing "Later today" list. |
| **Layout** | **Unchanged.** No element moves except that "Later today" grows by the rows it gains. |
| **Why** | Jay said to keep Today as it is (round 3), then allowed the new colours with the layout untouched (answer 1) and events inside the existing rows (answer 2). |
| **Source** | Answers 1 and 2 (2026-11-18); round 7 proof. |

Declined: round 5's separate 48 px event line under the Now card. Round 7 puts the event on the existing line instead.

Round 7 measured this on a 390 px phone, light and dark: of 141 elements on Today and the tab bar, 132 keep the same box; 8 of the other 9 are the "Later today" list growing by one row, and the ninth is the "Now" label, which keeps its box.

## DD-24 — Core components

Changes against 11 §N. Anything not listed is unchanged.

| Component | Old | New |
|---|---|---|
| Card | Reserved for the most important block | One per screen for the lead item (Now, hero metric, project header). Radius 20, inner hairline. |
| List | Grouped list, radius 16, `--e1` | The default container. Radius 20, inner hairline, rows 56 (compact 48), one trailing element, no per-row text buttons. |
| Stat tile | Health Metric in a 2 to 4 column grid | Label 12, number 26 (`--t-metric`), one sub-line, optional 4 px bar. Three across on phone Today. |
| Hero metric | Display 34 | Label, 56 px number with its unit, 6 px bar, one legend line. One per screen. |
| Button | 44 high, radius 12; `.small` 34 | 44 high, radius 12. Filled accent = commit, one per view. Soft accent = secondary. Grey = neutral. Red-tinted = destructive, only in sheets and detail panes. |
| Input | 44 high | **48 high**, text 16 px or larger, filled `--surface-2`, radius 12, accent ring on focus. Number fields get − and + steppers. |
| Section switcher | Segmented control | Segmented control under the title, 44 high, track `--surface-2`, radius 12. |
| Chip | 24 high | 24 high as a label, 44 high when it is a control. Grey by default; amber or red only for problems; a green dot for good. |
| Sheet | Radius 24; centred from 600, max 560 | Radius **28**; bottom sheet with a grab handle under 700, max 88% height; centred card 560 wide **from 700**. |
| Chart | Accent by default | One accent hue. 2 px line, dashed grey reference, last value labelled, tap or hover for any point. |
| Focus ring | 2 px accent, 2 px offset | Unchanged. |

Why: the round 1 audit found five button looks, inputs at 13.5 and 15 px that zoom the page on iPhone, and controls at 42–43 px. Source: Direction.

## DD-25 — Task row, task sheet and +

| | |
|---|---|
| **Old** | Task Row with a swipe or long-press menu "later"; Quick Capture as one field with parsed chips, opened from + on Today (11 Task Row, Quick Capture; 13 §12). In production, a 12-field task form. |
| **New: task row** | Check circle, title (weight 500), one meta signal (DD-19). Trailing: nothing, or one amber chip if late. No per-row Skip, Edit or Delete buttons; they live in the task sheet. |
| **New: task sheet** | Replaces the long form for create and edit. Title, then five chips (date, project, priority, repeat, reminder), notes and a checklist. Category, proof, status and XP sit under "More options". Tapping a chip changes it in place. Skip has an icon; Delete is an icon button. |
| **New: +** | On every screen: the round button beside the tab bar, and New in the sidebar. It opens the capture sheet with icon tabs for **Task, Food, Weight, Water and Event**, starting on the tab that matches the current screen. Weight starts at the last reading with − and + steppers. |
| **Project detail** | A header card (tile, name, status, "n of m", a bar in the project colour), then Next up with an inline add row (type, Return), Sub-projects, Done. Sub-projects show inside their project only, not in the Projects list. |
| **Project row** | Glyph tile, name (weight 600), one signal (next task, reviews waiting, or "Blocked · reason" in amber), then the count and a progress ring in the project colour. Status is a word with a dot or icon. Done projects collapse into "Done · n". |
| **Why** | Adding a task took three taps and a long scroll; the quick capture existed but was hidden (round 1 audit). |
| **Source** | Direction; round 3 (icons, sub-projects, All tasks grouping); round 7 (Event tab). |

Nothing in the task sheet removes a field: every field of the old form stays reachable (`docs/RULES.md` §8).

## DD-26 — Settings

| | |
|---|---|
| **Old** | One long page of forms. |
| **New** | A grouped list. Each group is a row with an icon, a one-line summary of its state, and an amber "Setup" chip when something needs Jay. A search field at the top filters every setting by name and shows its group. |
| **Groups** | Eight: Account & security · Appearance · Health & integrations · Accounts · Secretary · Notifications & prayer · Data & backups · Advanced. |
| **Rows inside a group** | State on the right: a dot and a word for connections, a value with a chevron, a switch, or one action button. |
| **Forms** | Long forms open as their own page or sheet from their row. Settings never shows a form inline. Their behaviour and guards do not change. |
| **Destructive** | Red-tinted, and asks for the password first. |
| **Layout** | Phone: the group list, then the group. From 1000: split view (DD-17). The group list fits one phone screen. |
| **Why** | Settings was 3,318 px tall on the phone in a third visual style, with inputs under 16 px (round 1 audit). |
| **Source** | Direction (six groups); round 3 (Secretary group, icons); rounds 4 and 5 (Accounts group); round 7. |

## DD-27 — Secretary surfaces

This entry records the patterns on screen. What the Secretary may read, do and keep is decided in `19A`, not here.

| | |
|---|---|
| **Old** | A ⌘K launcher with Go, Do and Capture modes (13 §11). No assistant. |
| **New: name** | The assistant is called **Secretary** in every label. |
| **New: one overlay** | A bottom sheet on phone; a command bar on wide screens, 640 wide, dropping from the top. It serves typing and talking alike. |
| **New: ways in** | Hold +, hold Space, ⌘K, the sidebar item, or **listening mode**. No Siri, Shortcuts or Action Button path. |
| **New: Listening pill** | While listening mode is on, a glass pill reading "Listening" with a moving mark stays at the top of every screen. Its × stops it. It says that GYST only listens while it is open on screen. The same switch is in Settings › Secretary. |
| **New: states** | Listening, Thinking, Answering, Proposals, Done, Error, Offline, No credit. |
| **New: proposal card** | One card per change: an icon, a plain sentence, the facts that will be saved, then Confirm · Edit · dismiss, under the label "nothing is saved yet". After Confirm, each change is a row with its own Undo. |
| **Rule on screen** | Nothing is saved before Confirm, and the numbers on a card are the numbers saved. |
| **Why** | Jay named the assistant, ruled out Siri and asked to say "Secretary" while GYST is open in a listening mode (answers 20 and 25); every change needs his Confirm and has Undo (answer 21). |
| **Source** | Round 3 (name, overlay); round 7 (listening mode, states); answers 20, 21 and 25 (2026-11-18). |

Superseded inside the makeover's own sources: Direction's Siri, Shortcuts and Action Button rows, and round 3's "On call" wake word for iPad and desktop only.

## DD-28 — Screens added in rounds 3 to 7

`12_SCREEN_ARCHITECTURE.md` has none of these. Until a screen spec is written for one, the round 7 mockup is its design reference, inside the tokens and components above. Behaviour and data rules are in `19A` where it covers them. Each screen is still its own Issue.

| Screen | Round | Notes |
|---|---|---|
| Overview | 3, 4 | Read-only: tasks as four counts, projects as tiles, health as six tiles. Three cards and a Health band from 1000. Reached from the launcher and the sidebar. |
| Desktop Today (home board) | 4, 5 | From 1000 only: a large Now card beside a Fuel cluster, then To do, Mail, Rhythm and Later today, and a full-width day line. Phone Today is DD-23. |
| Health Summary | 3 | Six icon tiles (calories, protein, water, steps, weight, training) and the plan as one row. |
| Mail | 4, 7 | Inbox list and reading view; split view from 1000. Rules in `19A`. |
| Calendar | 5, 6, 7 | Day, Week and Month; agenda list on phone, time grid from 700. Rules in `19A`. |
| Morning brief | 7 | Launcher group Plan. |
| Goals | 6 | Launcher group Work. Its own page, not folded into Projects. |
| Roadmap and Achievements | 7 | Achievements is in launcher group Learn. |
| Courses and Training | 6 | Training is read-only. |
| My foods, food profile and the meal review | 7 | Under Nutrition. |
| Health plan, Analytics, Import a plan, Archive, Apps | 6 | **Not redesigned.** Each keeps today's layout, restyled by the tokens only. |

Source: rounds 3 to 7, approved with the round 7 design on 2026-11-20.

---

## Alternatives

Every option that a round offered and did not build as the main design.

| Alternative | Offered in | Result | Source |
|---|---|---|---|
| A free hue slider for the accent | Direction | **Declined.** Seven curated accents ship. A slider would need a new decision. | Direction; round 7 (the seven stay) |
| "All tasks" as a switch on Today (Today · Upcoming · All), leaving the Projects tab to projects and Galaxy | Direction | **Declined.** Removed in round 3; the Tasks hub (DD-19) answers the same need. | Round 3; answer 3 |
| Typed commands first, voice after | Direction | **Accepted.** The Secretary's typed web chat is built before listening mode in the build order of record. | `design-makeover/issues-r7.md` part (c) |
| Galaxy on the phone as "Path", one scrolling route | Round 3 | **Declined** as the drawn target; "Focus" was carried through round 7. Neither is approved to build ([Galaxy](#galaxy)). | Round 3 |
| A separate event line under the Now card on phone Today | Round 5 | **Declined.** The event sits on the existing Now line. | Answer 2; round 7 |
| Looks "Instrument" and "Court" | Round 6 | **Declined.** | Answer 4; round 7 |
| Projects as the fourth tab | Round 6 | **Declined.** | Answer 3 |
| The four page merges (DD-20) | Round 6 | **Accepted**, with roadmaps made larger. | Answer 5 |
| Card styles Lifted and Tinted, beside Hairline | Round 6 | **Accepted** as a setting; Hairline is the default. | Rounds 6 and 7 |
| A fourth tab that remembers the last of Tasks · Projects · Galaxy | Round 7 | **Declined.** | UI1 |
| A seventh tab, or a planet button in the tab bar | Round 7 | **Declined.** | Round 7 |

## Galaxy

**Galaxy visual work stays behind its own gate. This file decides none of it.**

- `17A_GALAXY_PRESERVATION.md` is unchanged and stays binding. Visual and performance Galaxy changes still wait for the iPhone timings. Data-only Galaxy work is unaffected.
- The makeover's Galaxy pictures (always dark, project colours on worlds, glass chrome, an accent selection ring, the phone "Focus" view) are **mockups of a possible target, not a build plan**. Nothing in them is approved.
- Makeover chunks do not touch `galaxy.html`, `ui-galaxy.css` or `galaxy*.js`, and `node scripts/check-galaxy.cjs` must still pass after each one. A chunk that changes shared CSS, the shell or tokens runs the `17A` §L quick check.
- **Every value the Galaxy page reads stays unchanged** until the Galaxy chunk. `web/static/css/ui-galaxy.css` reads `--gyst-*` variables (background, surface, text, border, blue, radius, shadow), the legacy variables and the body font, so a token change can restyle Galaxy without touching a Galaxy file. R1 and every later chunk therefore **pin the whole `/galaxy` page, its shell included**, to today's values, and prove it with a computed-style comparison of that page before and after, in light and dark, that shows **zero differences**. The `17A` §L quick check alone does not catch this. If the pin can't be made without editing a Galaxy file, the chunk stops and reports to the lead. (Lead, 2026-11-20.)
- What this file does decide near Galaxy is navigation only: the Tasks · Projects · Galaxy switch (DD-19) and the `G X` shortcut. The switch that sits on the Galaxy screen is part of the Galaxy chunk, not the shell chunks.
- 13A DD-1 still holds: Galaxy may keep its dark, immersive look in both themes.

## Not changed

- Server-rendered pages, no new framework, build system or component library.
- The security headers: `script-src 'self'`, `font-src 'self'`, `img-src 'self'`. No inline scripts and no remote assets.
- The state vocabulary and the state colours' meaning.
- Honest missing-data display: missing is "—", never 0 (`docs/RULES.md` §3.2).
- 44 px touch targets and the 2 px focus ring.
- Existing functionality: no screen loses a feature to match a mockup (`docs/RULES.md` §8).
- Mockup sample values are fictional and never become production data.

## Open

Not decided. Nothing below may be built as if it were. (O-1, the accent seeds, was settled on 2026-11-20 and is now part of DD-6.)

| # | Open item | Why it is open |
|---|---|---|
| O-2 | **Aurora's starting accent.** Aurora is "the blue Look" and starts with Iris, an indigo, not with the accent named Blue. | **For Jay.** He asked for one blue, one green and one orange Look. Aurora is blue by its neutrals and its Plan colour, but it starts on Iris. It stays Iris for now, as drawn in the round 7 mockups he approved (lead, 2026-11-20). He picks the accent in Settings either way. |
| O-3 | **Listening mode on iPhone.** Whether an installed web app can keep the microphone open and spot the word while on screen is untested. | Waits for the iPhone listening test in the build order. DD-27 records the design only. |
| O-4 | **Real devices.** Every round was checked in headless Chromium only. Glass, safe areas, the six-tab bar under a thumb, the font in the installed app and iOS text-size following are unverified. | Jay's checks after the shell chunks. If iOS text size can't be made reliable, ship S · M · L only and say so. |
| O-5 | **Pages the makeover did not draw.** Development, Reviews, Agents, System, Automations and Attention from `11` and `12` have no place yet in the six groups, and the "Operate" sidebar group is not in the mockups. | No round covered them. `05A` D2 and D6 still describe where they belong. |
| O-6 | **Widths from 1600.** `13` §1 has a wide tier (sidebar 256, larger caps). The makeover draws one tier from 1000 with content capped at 1180 and was checked up to 1440. | Not drawn or checked in any round. |
| O-7 | **Attention badge.** `11` puts a badge on the Today tab only. Round 6 adds an unread count on the Mail tab and says nothing about the Today badge. | Not addressed in any round. |
| O-8 | **Time format.** The calendar shows 24-hour times; the Secretary speaks 12-hour times. Round 5 says one setting should decide both. | No answer recorded. |
| O-9 | **Pointers outside 11 and 13.** These files still name 13A as the design record or describe the old five tabs and Projects areas: `AGENTS.md` (the design hierarchy line and the shared-tools lines), `19A` (precedence line and §12), `05A` (D1, D2, D6) and `17A` (§C.6, §H, §M). | Left untouched on purpose (lead, 2026-11-20). `AGENTS.md` is a rule file and needs Jay's word; `17A` must not change here. |
| O-10 | **Screen specs.** The screens in DD-28 have no entry in `12`. | Written with each screen's own Issue. |
| O-11 | **Galaxy visual work.** | Behind the `17A` gate ([Galaxy](#galaxy)). |
| O-12 | **Violet and the active state.** `AGENTS.md` ("Non-negotiable product rules", Design) and 13A DD-4 say violet is never a state colour. Under DD-6 and DD-13, info and active states use the chosen accent, so with Iris or Orchid chosen an active state is drawn in a violet. | **For Jay.** Changing the wording of a rule file is his call alone. Until he decides, `AGENTS.md` stands as written and this file changes nothing in it. |

## Token tools for R1

The tools are on the owner's server under `artifacts/gyst/design-makeover/tools-r7/`, not in this repository:

| File | What it does |
|---|---|
| `gen_looks7.py` | Derives the three Looks from the seeds in DD-7 and asserts every pair. Writes `looks7.css` (the tokens) and `looks7.json` (values and the lowest ratios) beside itself. It carries the seven accents as fixed values and checks them on each Look; it does not derive them (DD-6). |
| `audit7.cjs` | Renders the mockup and measures every visible text node and icon against what is behind it. |
| `proof7.cjs` | Compares phone Today between rounds, element by element. |
| `kbd7.cjs` | The scripted keyboard pass. |
| `build7.py` | Builds `mockup-r7.html`. Mockup only; nothing to port. |

Notes for R1:

- The generated `looks7.css` and `looks7.json` are not kept in the folder. The values in DD-7, DD-8 and DD-13 were read from the generated block inside `design-makeover/mockup-r7.html`, and R1's output must equal them.
- Port the generator into the repository so the build asserts the pairs; a failing pair fails the build. The mockup's CSS is a reference, not a file to copy. In particular, do not copy its reduced-motion rule that delays focus inside sheets (round 7).
- Token names here are the mockup's. R1 maps them onto the production `--vn-*` tokens and points `--gyst-*` and the legacy variables at those, as `13` "Implementation risks" already says, **except on the `/galaxy` page**, where every value stays as it is today ([Galaxy](#galaxy)).
- The mockup switches with `data-theme`, `data-look`, `data-accent` and `data-density` on the root, and `data-card` on the app. Production maps `data-theme` onto its existing theme attribute.
- Round 7's rendered audit: 40,342 pairs on 78 screens, sheets and states, three Looks plus Tinted cards, light and dark, phone and desktop; none below AA. Lowest text 4.57:1 (white on the Teal button, Lagoon, light); lowest icon 4.02:1 (a project glyph on its tile, Sunrise, light; icons need 3:1). The audit can't see through glass or gradients, and it left out Galaxy.

## Design source of truth

After this file, the design hierarchy is:

```text
13B_DESIGN_DECISIONS.md            ← the makeover; overrides where it differs
13A_DESIGN_DECISIONS.md            ← still binding; DD-1 and DD-4 as amended here
    ↓
11_DESIGN_SYSTEM_VNEXT.md
12_SCREEN_ARCHITECTURE.md
13_RESPONSIVE_INTERACTION_SPEC.md
    ↓ (reference only, non-binding where it conflicts)
the round 7 mockup · docs/audit/vnext/prototypes/
```

`05A`, `08A`, `10A`, `17A` and `19A` stay binding for product, integration, security, Galaxy and connected-data scope.

A later conflict or a new pattern is recorded as a new entry in this file, never as an ad-hoc change (13A, "Implementation discipline").

## Sources

On the owner's server under `artifacts/gyst/`, not in this repository. Jay's answers are paraphrased here, never quoted.

| Short name | File | Used for |
|---|---|---|
| Direction | `design-makeover/direction.md` | Rounds 1 and 2: tokens, components, "Changes to recorded decisions", the Alternatives |
| Round 1 audit | `design-makeover/assessment.md` | The problems the makeover answers |
| Rounds 3 to 7 | `design-makeover/assessment-r3.md` to `assessment-r7.md` | What each round changed; round 7 is the final state |
| Plan | `design-makeover/plan.md` | Chunk 0 scope and the chunk list |
| Build order | `design-makeover/issues-r7.md` part (c); `design-makeover/build-order.md` | The order of record |
| Answers | `decisions/jay-answers-2026-11-18.md` | Answers 1 to 5, 20, 21 and 25; the round 7 choices |
| Decision list | `decisions/decisions-r7.md` | UI1 |
| Mockup | `design-makeover/mockup-r7.html` | The exact token values |
| Token tools | `design-makeover/tools-r7/` | See [Token tools for R1](#token-tools-for-r1) |
