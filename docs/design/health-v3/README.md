# Health v3 — design prototype (Phase 1)

Refs #45. Design only. Fictional data, no backend, no network calls, no LLM.
Production Health pages, the database, the syncs and the APIs are unchanged.

Open `index.html` in a browser. Pages: `index.html` (Today), `workout.html`,
`nutrition.html`, `coach.html`, `settings.html` (More).

The **Prototype** strip at the bottom of each page switches the data
(Full day · Partial · Empty · Long values) and the theme (System · Light · Dark).
URL form: `index.html?data=partial&theme=dark`.

Screenshots: [`screenshots/`](screenshots/).

## Key design decisions

1. **One number first.** Today leads with calories left in a ring. Eaten, target
   and active sit under it, and energy balance comes last.
2. **"So far", never "deficit".** Balance = eaten − (resting + active), labelled
   "Energy balance so far · Observed today, not a final deficit". The formula is
   in an info sheet, not on the card.
3. **Honest data.** Missing values show "—" with a dashed ring or bar. A recorded
   zero shows "0" with "Recorded zero today". Watch and phone steps stay separate
   and are never added together. Steps are never turned into calories.
4. **Rings only where they help.** Calories and four macros use rings. Everything
   else uses numbers, bars or sparklines. Protein and fiber turn green when met;
   carbs and fat turn amber when over, because those are limits.
5. **Colour discipline.** Blue = progress and action, green = target met,
   amber = over or stale, every state paired with a word. Violet only on the
   GYST brand dot (13A DD-4). Macros don't get their own hues.
6. **GYST reads, it doesn't log.** No food diary, no workout logger. Workout shows
   Hevy data read-only; Nutrition has one action: Open MyFitnessPal.
   The only manual input is weight, because no source supplies it today.
7. **Progressive disclosure.** Muscle table, workout sets, energy formula, food
   details and "What the coach can see" sit behind a tap.
8. **Checklist is the user's.** Supplements are user-added checkboxes with no
   doses. Protein and water are measured targets (132 / 190 g), not checkboxes.
9. **Coach shows its sources.** Each reply ends with "Used: …" listing the data
   it drew on.

## Conflicts with the vNext design documents (need Jay's decision)

| # | This prototype | vNext rule | Why |
|---|---|---|---|
| C1 | Inside Health on phone, a **Health tab bar** (Today · Workout · Nutrition · Coach · More) replaces the app tab bar; "‹ GYST" top-left goes back | `11` §N: one five-tab app bar; `11`/`12`: Health sections are a top segmented control | Jay asked for a fixed bottom Health nav. Two stacked bottom bars would be worse. |
| C2 | Health sections renamed: Summary → Today, Training → Workout, Progress + Plan → More; Coach is new | `12` Health sections | Matches the requested IA |
| C3 | Today uses several cards (calorie, macros, activity, weight) | `11` §B.3: one card per screen; grouped lists by default | Requested Apple-Fitness-style visual summary |
| C4 | Cards get a 1 px hairline ring plus a very soft shadow | `11` §F: no borders on lists | Requested "thin borders" |
| C5 | Desktop sidebar shows Health's five areas as sub-items; the sixth is labelled "Settings" (phone tab says "More") | `11` §N sidebar: More is app-level | Avoids two "More" items on desktop |

If approved, these become a new entry in `13A_DESIGN_DECISIONS.md`.

## What current data can't support yet

| UI element | Gap |
|---|---|
| Log weight | No writer: no form, no API, and the Apple Health sync doesn't send weight (`health_weight_entries` has 0 rows) |
| Weight change since cycle start | Needs weight readings |
| Training time | Hevy `end_time` isn't stored (`TrainingEntry` keeps only `StartedAt`) |
| Exercise progress % and sparklines | Computable from stored sets, but no code does it yet. Prototype numbers are fake |
| This week n / target | Needs workouts per week from Hevy history (computable) plus the existing `strength_sessions_per_week` target |
| Volume | Computable (load × reps over work sets); not computed yet |
| Muscle focus as % per group | Existing `SummarizeTraining` counts sets per muscle; a grouping to Legs/Back/… is new |
| Water target | No water field in the daily sync or plan |
| Meals | The sync stores daily totals only. The section is conditional and hidden when absent |
| Daily checklist, My foods & products | No tables |
| Health Coach | No provider, context builder or storage |
| Status line ("On track so far") | Rule needs agreeing (calorie target only, today) |

Note on the brief's example numbers: 2,290 target − 1,450 eaten = 840 left, not
1,750. The prototype uses 840.

## Recommended backend changes for Phase 2 (one Issue each)

1. **Weight entry**: `POST /health/weight` (CSRF, same-origin, validated kg),
   Log weight sheet. Optional: accept weight in the Apple Shortcut sync.
2. **Hevy summary**: store workout `end_time`; compute weekly count, volume,
   training time, per-exercise best set and e1RM trend, muscle-group shares.
   Pure functions in `internal/services` with tests.
3. **Health v3 templates**: server-render the five screens from existing view
   models, progressive enhancement for sheets. Retire `health.html` sections as
   they move. Health tab bar only after C1 is approved.
4. **Daily checklist**: `health_checklist_items` + `health_checklist_log`
   (additive migration). Measured targets derived, never stored as ticks.
5. **My foods & products**: `health_foods` table (name, serving, kcal, P/C/F/fiber, notes). CRUD in More.
6. **Water**: add optional `water_ml` to the daily sync and `water_target_ml` to cycles.
7. **Health Coach**, risk elevated with security review:
   - provider interface (Claude or OpenAI), key server-side only
   - a `CoachContext` struct built from named fields, not a database dump
   - rate limit, size limits, no context in logs
   - optional conversation storage with a delete action
8. **Status rule**: define "on track" (for example eaten ≤ target so far), then test it.

## Verified

- 390, 430, 768, 1280 widths × 4 data states × light and dark: no horizontal overflow, no clipped cards, no console errors
- Keyboard: logical tab order, visible focus, sheet takes focus, Escape closes and returns focus, arrow keys switch Overview/Progress, Enter sends in Coach
- Typed text in Coach and forms is escaped
- Touch targets ≥ 44 px on phone widths (desktop sidebar rows are 36–40 px, pointer only)
- Existing checks unchanged and passing: `scripts/check.sh` (gofmt, vet, tests), `check-ui-shell.cjs`, `check-galaxy.cjs`
- Not verified: real iPhone or PWA, VoiceOver, 200% text zoom
