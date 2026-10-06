# GYST Health — Current State (2026-10-05)

**Production:** `c2c8ae7` (PR #63, merged and deployed 2026-10-06)  
**Last updated:** 2026-10-05, corrected against code and production (read-only checks)  
**Status:** Health v3 Today, Workout, Nutrition and More are live, with the Daily checklist, My foods & products and protein-gap suggestions. Integrations operational. Remaining: Health Coach (PR #65, awaiting Jay), water target (#60), final real-device QA.

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

### Health v3 UI (PR #54 merged, commit `6efa27e`)
Three pages implemented, replacing old Health view:

#### 1. Health → Today (`/health`, `web/templates/health.html`)
- **Calories card:** budget left vs target, progress ring, eaten + burned breakdown
- **Macros section:** protein, carbs, fat, fiber with targets + progress rings
- **Activity section:** steps (Watch and Phone tracked separately), exercise minutes
- **Weight section:** latest reading, trend since plan start, sparkline chart
- **Stale data warning:** flagged when Apple Health data is >8 hours old

#### 2. Health → Workout (`/health/training`, `web/templates/training.html`)  
- **Weekly summary:** workouts completed vs target, volume (kg), training time
- **Muscle focus:** visual body map showing primary muscle groups by volume percentage
- **Top progress:** 3 exercises with best improvement in last 12 weeks (sparkline per exercise)
- **Latest workout detail:** exercises, sets, rep ranges, load, RPE
- **Earlier workouts:** scrollable list of recent 30 sessions
- **Sync status:** last synced time, Hevy connection state, manual sync button
- **Stale data warning:** flagged when Hevy data is >2 hours old

#### 3. Health → More (`/health/more`)
- Health plan, log weight and weight history, data sources with sync state, health goals, health tasks, gym check-in trend. Checklist and My Foods settings will be added here.

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

**Planned lightweight support:**
- **My Foods & Products:** a lightweight list of foods Jay usually has (name, serving, kcal, macros, notes, active), managed in More and later used by the Coach
- **Portion/meal suggestions:** ways to close the day's gaps using My Foods, delivered through the Coach
- **Not:** Full food database, recipe import, or competitive food logging

**Implementation status:**
- Not started. My Foods & Products is built **before** the Coach and becomes part of its context; it isn't blocked by the Coach.
- Food and product support is planned, not out of scope; only full food logging stays in MyFitnessPal.

### Hevy Logging / Workout Entry in GYST
**By design:** Hevy is read-only  
- Users log workouts in Hevy app
- GYST imports and summarizes
- No "start workout" or "log set" in GYST

### Health Coach (LLM Layer)
**Status:** Not implemented  
**Planned shape:** Live GYST context + retrieval + allowlisted **read-only** tools first (today, plan, weight trend, training, foods, checklist). Any later write needs explicit user confirmation. Provider-neutral (OpenAI or Anthropic), keys server-side only.  
- Not model fine-tuning on Jay's data
- Not scheduled work
- Backend architecture unbuilt

---

## Real Gaps (Feature Work Remaining)

**Implemented: Health v3 Today, Workout and More. Not yet complete:** the items below.

**Remaining features:**

Done since the last update (all merged and deployed 2026-10-06):

- **Nutrition v3** (#55, PR #58): calories vs plan, macro tiles, last 7 days vs target, MyFitnessPal → Apple Health source row.
- **Daily checklist** (#59, PR #61): habits ticked per local day on Today; protein measured from food logged. Settings at `/health/checklist`. Migration `20261006010000_health_checklist_foods.sql`.
- **My foods & products** (#59, PR #61): `/health/foods`, per-serving nutrition with unknown kept blank.
- **Portion support** (#62, PR #63): "Close the protein gap" card on Nutrition from My foods, best protein per kcal, fits-budget flag. No AI.

Still open:

1. **Health Coach** (#64, PR #65): provider-neutral backend, read-only Health tools, Anthropic adapter, GYST-owned memory. Elevated risk: **Jay decides the merge and the key**. See `docs/HEALTH_COACH.md` on that branch. The OpenAI adapter is a follow-up.

2. **Water target** (#60): needs a water field in the Apple Health sync, a plan target and a Shortcut update.

6. **Real device QA** — End-to-end testing on iPhone + Watch
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
   - Test Daily Checklist when available
   - Monitor stale-data warnings (>8 hours without sync)

**For Agent (Feature Work):**

1. **Health Coach**: address Jay's review of PR #65; then the OpenAI adapter
2. **Water target** (#60), after Jay adds water to the Shortcut
3. **Real device testing**: production iPhone/Watch

---

## Summary

**Health v3 Today, Workout, Nutrition and More are live, with the checklist, My foods and protein-gap suggestions. The Coach awaits Jay's decision.**

- **Today page:** Real calories, macros, activity, weight data via Apple Health + manual entry
- **Workout page:** Real Hevy training syncing every 37 minutes; exercise progress analytics live
- **Nutrition page:** calories vs plan, macro tiles, last 7 days, protein-gap suggestions from My foods
- **More page:** plan, log weight and weight history, Daily checklist, My foods & products, data sources with sync state, health goals, health tasks, gym check-in trend
- **Weight logging:** Live with undo, history chart, trend tracking

**Integrations are operational:**
- Hevy: Syncing automatically, production tested
- Apple Health: activity, energy and daily nutrition totals reaching GYST; MyFitnessPal end-to-end comparison still to verify

**Next feature sequence:**
1. Verify Apple Health / MyFitnessPal nutrition end-to-end (Jay)
2. Health Coach: Jay's decision on PR #65, then the provider key
3. OpenAI adapter (follow-up to #64)
4. Water target (#60)
5. Real iPhone/PWA QA

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
