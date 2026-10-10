# GYST Health — Current State (2026-10-09)

**Production:** master `4ee5bc4` (PR #81), deployed 2026-10-08; it includes the water work (Issue #60 / PR #75, Issue #76 / PR #77, Issue #78 / PR #79). Later merges to master (PR #82) are docs only. Earlier: PR #74 (real-data rule), #71 (Today redesign), #68 (load rounding), #65 (Coach, off until a key is set).  
**Last updated:** 2026-10-09, catching up with everything merged 2026-10-06 and 2026-10-07: the Coach merge, Nutrition week fixes, 0.1 kg loads, the Today redesign, water and the daily water stop  
**Status:** Health v3 Today, Workout, Nutrition and More are live, with the Daily checklist, My foods & products and protein-gap suggestions. Integrations operational. Today redesigned to the approved mockup (Issue #70 / PR #71), with water logging (Issue #60 / PR #75, Issue #76 / PR #77) that stops at the daily target (Issue #78 / PR #79). Nutrition week marks recorded-zero and partial days (Issue #69 / PR #72). Health Coach merged 2026-10-06 (Issue #64 / PR #65) and off until Jay sets `HEALTH_AI_PROVIDER` and `ANTHROPIC_API_KEY`. Remaining: Jay's real-device QA.

**Note:** This document reflects the **production state** of implemented features. Production has already reported successful Hevy syncs and Apple Health data ingestion. Treat operational features as running unless proven otherwise.

---

## Live & Implemented

### Apple Health Integration (M3 backend complete)
- `/api/health/sync` POST endpoint accepts Shortcut data
- Parses: dietary calories, macros (protein, carbs, fat, fiber), activity (steps, exercise minutes), energy (resting + active)
- Distinguishes NULL (not received) vs 0 (observed zero)
- Health sync state tracked; stale data flagged (>8 hours)
- **Current state:** An existing, working Shortcut (see `docs/HEALTH_SHORTCUT_SETUP.md`) already sends data to GYST. Production has today's dietary calories as well as activity, so nutrition totals are arriving.
- **Remaining operator task:** finish/verify the MyFitnessPal → Apple Health → Shortcut flow and compare real values end-to-end against MyFitnessPal.

### Health v3 UI (PR #54 `6efa27e`; Nutrition added in PR #58)
Today, Workout and More from PR #54; Nutrition v3 from PR #58 (see Real Gaps below):

#### 1. Health → Today (`/health`, `web/templates/health.html`; redesigned in Issue #70 / PR #71)
- **Calories today card:** food budget left (target − eaten) in a ring coloured by state, always with words: *Within budget*, *Close to budget* (the last tenth of the target), *Over budget*, *No food logged*, *Waiting for sync*. Beside it, three separate numbers: **Eaten** (of target), **Burned so far** (resting + active) and **Balance so far** (eaten − burned, "Observed, not final"). Numbers are rounded before they're combined, so the parts on screen always add up.
- **Movement nudge** (`internal/handlers/health_home.go`): deterministic, server-side, never mentions calories.
  - Hidden when there's no snapshot, the data is stale, before 11:00, or from 21:00.
  - Hidden after 30+ exercise minutes or once the step target is reached.
  - After a Hevy workout that day it shows a short recovery note instead.
  - Otherwise it shows the steps left to the plan's step target, with Walk / Swim / Bike ideas.
- **Macro cards:** protein, carbs, fat, fiber in their category colours with bars; "over" warnings on limits, "Target met" on minimums. Each card opens Nutrition.
- **Steps card:** Apple Watch when observed, else iPhone, labelled; the other source is listed separately and never summed; exercise minutes.
- **Weight card:** latest reading, change since plan start, sparkline, history link and Log weight.
- **Water card** (Issue #60 / PR #75, Issue #76 / PR #77, live; the daily stop at the target is Issue #78 / PR #79; `internal/handlers/health_water.go`, `health_components.html` `health.bottle`):
  - Jay's water bottle (Issue #76 / PR #77) fills the card. **Tapping the bottle** adds one bottle of the plan's size to today, and the water rises with waves to the share of the target: for example (illustrative sizes only), with a bottle half the size of the target, one tap fills it halfway and two fill it. **Undo** removes the last tap.
  - The waves and bubbles settle after about a minute. Motion stops under reduced motion.
  - Target and bottle size are set in Health → Plan (blank = not set).
  - Taps are stored per local day in `health_water_log`, so the bottle starts empty at midnight (America/Winnipeg). Taps stop at the plan's target (Issue #78 / PR #79): the check is in the same statement as the insert, and the tap that reaches or passes the target is the last one. Once full, the bottle isn't a button ("Done for today · resets at midnight"); Undo stays. No taps reads "No water logged yet", never 0 L.
  - The card sits right after the calorie card (and nudge) at every width, so the screen order and the reading order match.
- **Daily checklist:** "n of m complete" with a bar and round toggles. The protein and water items are measured, not ticked; water is added from checklist settings and stored as the `health.checklist.water` setting.
- **Stale data warning:** flagged when Apple Health data is >8 hours old.
- Design record: `docs/design/health-home/` (approved mockup, prototype, states).

#### 2. Health → Workout (`/health/training`, `web/templates/training.html`)  
- **Weekly summary:** workouts completed vs target, volume (kg), training time
- **Muscle focus:** visual body map showing primary muscle groups by volume percentage
- **Top progress:** 3 exercises with best improvement in last 12 weeks (sparkline per exercise)
- **Latest workout detail:** exercises, sets, rep ranges, load, RPE
- **Loads:** shown to 0.1 kg instead of raw decimals from Hevy (Issue #67 / PR #68)
- **Earlier workouts:** scrollable list of recent 30 sessions
- **Sync status:** last synced time, Hevy connection state, manual sync button
- **Stale data warning:** flagged when Hevy data is >2 hours old

#### 3. Health → Nutrition (`/health/nutrition`, `web/templates/nutrition.html`, `internal/handlers/nutrition.go`)
- **Calories vs plan** and **macro tiles** against the plan's targets (Issue #55 / PR #58).
- **Last 7 days** against target, with the MyFitnessPal → Apple Health source row. A day with a recorded zero is not counted as logged, and a partial day is marked as partial (Issue #69 / PR #72).
- **Close the protein gap** card: picks from My foods, best protein per kcal, fits-budget flag, no AI (Issue #62 / PR #63).

#### 4. Health → More (`/health/more`)
- Health plan, log weight and weight history, Daily checklist, My foods & products, **Profile for the coach** (`/health/profile`), data sources with sync state, health goals, health tasks, gym check-in trend.

### Weight Logging (PR #48 merged)
**Page:** Health → Progress (`/health/progress`, `web/templates/health_progress.html`)
- **Log weight form:** input kg + measured timestamp
- **Undo button:** remove last entry
- **Weight history chart:**
  - SVG sparkline plot with actual readings
  - Range filter: 30 days, 90 days, all time
  - Table of measurements with dates + sources
  - Ranges respect application timezone
- **Stored in DB:** `health_weight_entries` table

### Hevy Training Sync (Live & Operational)
**Status:** Syncing in production; auto-sync every 37 minutes confirmed

**Hevy provider:** `internal/providers/hevy/hevy.go`
- API client reads-only (no logging in Hevy from GYST)
- Pagination support for large histories
- Error handling: auth (401/403), rate limit (429), incomplete responses
- Syncs every 37 minutes automatically
- Manual sync button on Workout page

**Training cache:** `internal/services/training.go`
- Fetches: workouts, exercises, sets, templates, muscle mappings
- Caches last sync time + error state
- Protects against concurrent syncs (mutex)
- Normalizes Hevy's JSON to GYST models (`internal/models/training.go`)

**Workout analytics:**
- `TrainingSummary`: weekly workouts, volume, training time, unknown-time entries
- `ExerciseTrend`: per-exercise progress (best set, change %, sparkline series)
- `MuscleGroupShare`: primary/secondary breakdown for body map
- `WorkoutsPerWeek`: bar chart data for progress view

---

## Architecture: Data Sources (Not Local Logging)

### Foods & Nutrition
**Logging:**  MyFitnessPal (not GYST)
- GYST reads daily aggregate calories + macros from Apple Health (MyFitnessPal is the source)
- GYST does not and will not replace MyFitnessPal as the food logger

**Lightweight support (implemented):**
- **My Foods & Products** (PR #61): a lightweight list of foods Jay usually has (name, serving, kcal, macros, notes, active), managed in More; the Coach reads it through its `get_food_products` tool when enabled (`internal/handlers/coach.go`)
- **Portion suggestions** (PR #63): "Close the protein gap" on Nutrition picks from My Foods, with no AI. Coach-based meal suggestions come with the Coach.
- **Not:** Full food database, recipe import, or competitive food logging. Only food logging stays in MyFitnessPal.

### Hevy Logging / Workout Entry in GYST
**By design:** Hevy is read-only  
- Users log workouts in Hevy app
- GYST imports and summarizes
- No "start workout" or "log set" in GYST

### Health Coach (LLM Layer)
**Status:** Merged 2026-10-06 (Issue #64 / PR #65, `cbc5411`). Off in production unless `HEALTH_AI_PROVIDER` and `ANTHROPIC_API_KEY` are both set (`internal/app/app.go`, `docs/HEALTH_COACH.md`). Jay's step is the #64 line in `docs/jay/MY_ACTIONS.md`.  
**Shape:** Live GYST context + retrieval + allowlisted **read-only** tools first (today, plan, weight trend, training, foods, checklist). Any later write needs explicit user confirmation. Provider-neutral (OpenAI or Anthropic), keys server-side only.  
- Not model fine-tuning on Jay's data
- Not scheduled work
- Anthropic adapter built; OpenAI adapter is a follow-up

---

## Real Gaps (Feature Work Remaining)

**Implemented: Health v3 Today, Workout, Nutrition and More, plus the Daily checklist, My foods and protein-gap suggestions.**


Done 2026-10-06 and 2026-10-07 (merge dates in UTC; all in production since the 2026-10-08 deploy):

- **Health Coach** (Issue #64 / PR #65, merged 2026-10-06): provider-neutral backend, read-only tools, Anthropic adapter. Off until the key is set. Migration `20261006020000_health_coach_messages.sql`.
- **Workout loads to 0.1 kg** (Issue #67 / PR #68, merged 2026-10-06).
- **Today redesign** (Issue #70 / PR #71, merged 2026-10-06): calorie card, movement nudge, macro cards, checklist.
- **Nutrition week** (Issue #69 / PR #72, merged 2026-10-07): recorded-zero days not logged, partial days marked.
- **Water target and bottle taps** (Issue #60 / PR #75, merged 2026-10-07). Migration `20261007010000_health_water.sql`.
- **Tap the bottle itself** (Issue #76 / PR #77, merged 2026-10-07).
- **Stop at the daily target** (Issue #78 / PR #79, merged 2026-10-07).

Done 2026-10-05 (merged and deployed, Winnipeg time):

- **Nutrition v3** (Issue #55 / PR #58): calories vs plan, macro tiles, last 7 days vs target, MyFitnessPal → Apple Health source row.
- **Daily checklist** (Issue #59 / PR #61): habits ticked per local day on Today; protein measured from food logged. Settings at `/health/checklist`. Migration `20261006010000_health_checklist_foods.sql`.
- **My foods & products** (Issue #59 / PR #61): `/health/foods`, per-serving nutrition with unknown kept blank.
- **Portion support** (Issue #62 / PR #63): "Close the protein gap" card on Nutrition from My foods, best protein per kcal, fits-budget flag. No AI.

Still open:

1. **Health Coach** (Issue #64 / PR #65): merged 2026-10-06 and off until Jay sets `HEALTH_AI_PROVIDER` and `ANTHROPIC_API_KEY` (see `docs/HEALTH_COACH.md`). The OpenAI adapter is a follow-up. Since GYST-063 (Issue #101) the Coach's today tool also carries today's water (logged litres, taps, target); no taps is null, never 0.

2. **Real device QA** — End-to-end testing on iPhone + Watch
   - Apple Health Shortcut nutrition flow
   - Hevy sync accuracy
   - Weight logging + charting
   - Daily Checklist UX

---

## Data Sources & Ownership Model

| Source | Tool | GYST Role |
|--------|------|-----------|
| **Nutrition** | MyFitnessPal | Read via Apple Health → daily total, macros |
| **Activity** | Apple Health (from Apple Watch + iPhone) | Read → steps Watch/Phone, exercise minutes |
| **Energy** | Apple Health (from Apple Watch) | Read → resting + active energy |
| **Workouts** | Hevy Pro | Read → routines, exercises, sets, muscle mapping |
| **Weight** | Manual entry in GYST | Direct input → `/health/progress` form |
| **Targets & Plans** | GYST | Direct input → `/health/plan` form |

**Do not duplicate logging.** GYST is aggregator + summaries + targets, not a food diary or workout log.

---

## Health Calculations Implemented

### Energy (Calories)
- **Budget left** = target − eaten
- **Observed burn** = resting + active (never step-derived)
- **Observed balance** = eaten − observed burn
- No projection; step-derived calories explicitly **not** added

### Macros
- **Remaining** = target − consumed
- **Progress %** = (consumed / target) × 100
- Targets default to 0 → displays as absent in UI, not target

### Steps  
- **Watch and iPhone tracked separately**, never summed
- Today shows Apple Watch steps when observed, otherwise iPhone steps, labelled with the source; the other source is listed separately
- The plan's step target drives the bar

### Training
- **Volume** = sum of load × reps over work sets (normal, failure, dropset) that have both; warmups and bodyweight sets excluded
- **Training time** = sum of (end − start) for workouts with an end time; others counted as unknown, not zero
- **Top progress** = change in best estimated 1-rep max (Epley, 1–12 reps) from first to latest session, exercises with 2+ sessions, over 4 wk / 12 wk / 6 mo
- **Weekly target** = the active plan's strength sessions per week (never hardcoded)
- **Muscle focus** = primary work sets by muscle group (excluding warmups)
- Windows: muscle focus and group shares 30 days (or 7 / routine targets via Details); top-progress card 12 weeks; Progress view 4 wk / 12 wk / 6 mo

---

## Next Actions

**For Jay (Operator QA):**

1. **Verify Apple Health Shortcut nutrition flow:**
   - Log meal in MyFitnessPal
   - Ensure Apple Shortcuts automation runs
   - Check `/health` Today view for updated calories + macros
   - Compare accuracy end-to-end

2. **Real device QA (iPhone + Watch):**
   - Verify Hevy syncs workouts every 37 minutes
   - Check weight logging UX (form, undo, chart accuracy)
   - Test the Daily Checklist on iPhone
   - Monitor stale-data warnings (>8 hours without sync)

3. **Health Coach key:** set the provider and key, or leave the Coach off (the #64 line in `docs/jay/MY_ACTIONS.md`).

**For Agent (Feature Work):**

1. **Health Coach**: the OpenAI adapter

---

## Summary

**Health v3 Today, Workout, Nutrition and More are live, with the checklist, My foods and protein-gap suggestions. The Coach is merged and off until Jay sets the key.**

- **Today page:** Real calories, macros, activity, weight data via Apple Health + manual entry
- **Workout page:** Real Hevy training syncing every 37 minutes; exercise progress analytics live
- **Nutrition page:** calories vs plan, macro tiles, last 7 days (recorded-zero and partial days marked), protein-gap suggestions from My foods
- **Water:** tap-the-bottle card on Today, stops at the daily target
- **More page:** plan, log weight and weight history, Daily checklist, My foods & products, data sources with sync state, health goals, health tasks, gym check-in trend
- **Weight logging:** Live with undo, history chart, trend tracking

**Integrations are operational:**
- Hevy: Syncing automatically, production tested
- Apple Health: activity, energy and daily nutrition totals reaching GYST; MyFitnessPal end-to-end comparison still to verify

**Next feature sequence:**
1. Verify Apple Health / MyFitnessPal nutrition end-to-end (Jay)
2. Health Coach: Jay sets the provider key or leaves it off
3. OpenAI adapter (follow-up to Issue #64)
4. Real iPhone/PWA QA (Jay)

---

## Code References

- **Health Today / More:** `internal/handlers/handlers.go` (`health`, `healthBase`) + `health_v3.go` + `health_view.go` + `web/templates/health.html`, `health_more.html`
- **Health v3 styles:** `web/static/css/ui-health-v3.css`; tab bar and sidebar sub-links in `web/templates/ui_shell.html`
- **Training summaries:** `internal/services/training_progress.go`
- **Hevy Sync:** `internal/providers/hevy/`, `internal/services/training.go`, `internal/handlers/training.go`
- **Weight:** `internal/handlers/health_progress.go`, `health_weight.go` + `web/templates/health_progress.html`
- **Nutrition:** `internal/handlers/nutrition.go` (week, protein gap) + `web/templates/nutrition.html`
- **Checklist & foods:** `internal/handlers/health_config.go`, `internal/services/health_config.go`, `internal/repository/health_config.go`, templates `health_checklist.html`, `health_foods.html`, `health_food.html`, `health_components.html`
- **Models:** `internal/models/health.go`, `internal/models/training.go`
- **DB:** `internal/repository/` (health_* and training_* tables)
- **Water:** `internal/handlers/health_water.go`, `internal/services/health_water.go`, `internal/repository/health_water.go`; the bottle is `health.bottle` in `web/templates/health_components.html`; migration `migrations/20261007010000_health_water.sql`
- **Today extras:** `internal/handlers/health_home.go` (movement nudge, calorie card)
- **Health Coach:** `internal/handlers/coach.go`, `internal/healthcoach/provider.go`, `internal/healthcoach/anthropic.go`, `web/templates/health_coach.html`, `web/templates/health_profile.html`, `web/templates/health_plan.html`; wiring in `internal/app/app.go`; migration `migrations/20261006020000_health_coach_messages.sql`; setup in `docs/HEALTH_COACH.md`
