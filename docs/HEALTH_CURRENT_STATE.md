# GYST Health — Current State (2026-10-05)

**Master branch HEAD:** `6efa27e` (2026-09-14)  
**Last updated:** Operator reality audit  
**Status:** Health v3 UI live; integrations operational; next: Nutrition v3, Daily Checklist, My Foods & Products, Health Coach

**Note:** This document reflects the **production state** of implemented features. Production has already reported successful Hevy syncs and Apple Health data ingestion. Treat operational features as running unless proven otherwise.

---

## Live & Implemented

### Apple Health Integration (M3 backend complete)
- `/api/health/sync` POST endpoint accepts Shortcut data
- Parses: dietary calories, macros (protein, carbs, fat, fiber), activity (steps, exercise minutes), energy (resting + active)
- Distinguishes NULL (not received) vs 0 (observed zero)
- Health sync state tracked; stale data flagged (>8 hours)
- **Current state:** Existing Shortcut sends activity data to GYST; nutrition flow partially complete

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
- Framework exists; awaiting Daily Checklist + My Foods & Products features

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
- **My Foods & Products:** Quick reference for frequently-logged items (for Health Coach context only)
- **Meal suggestions:** Portion sizes + hydration reminders (powered by Health Coach)
- **Not:** Full food database, recipe import, or competitive food logging

**Implementation status:**
- Not yet started; depends on Health Coach context layer

### Hevy Logging / Workout Entry in GYST
**By design:** Hevy is read-only  
- Users log workouts in Hevy app
- GYST imports and summarizes
- No "start workout" or "log set" in GYST

### Health Coach (LLM Layer)
**Status:** Not implemented  
**Planned shape:** Live GYST context (today's health state) + safe tools (target setter, daily log query)  
- Not model fine-tuning on Jay's data
- Not scheduled work
- Backend architecture unbuilt

---

## Real Gaps (Feature Work Remaining)

**Phase 2 complete (Today, Workout, More pages live with Health v3 UI)**

**Phase 3 — Remaining features:**

1. **Nutrition v3 redesign** — Current Nutrition page exists but has not been redesigned to Health v3 standards (layout, components, real-time sync status)

2. **Daily Checklist** — Water intake, supplement log, sleep notes (appears in `/health/more`)
   - Database tables exist; UI not yet built
   - Manual entry + daily streak tracking

3. **My Foods & Products** — Quick reference for frequently-logged items + meal suggestions
   - Blocked on Health Coach context layer
   - Portion size suggestions

4. **Lightweight Portion Support** — Hydration reminders, meal size suggestions
   - Blocked on Health Coach

5. **Health Coach context + safe tools** — LLM-powered live context
   - Safe tools: set targets, query daily logs
   - Not: fine-tuning on Jay's data, scheduled work
   - Backend architecture unbuilt

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
- **Watch and Phone tracked separately**, never summed
- Both shown on Today card with notation of source
- Step target used only for progress ring

### Training
- **Volume** = sum of working-set kg weight
- **Training time** = sum of (end - start) minutes per session
- **Top progress** = exercises with largest improvement in weight × reps across last 12 weeks
- **Muscle focus** = primary work sets by muscle group (excluding warmups)
- All period-aware (last 7/30/84+ days configurable)

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
   - Test Daily Checklist when available (Water, sleep, supplements)
   - Monitor stale-data warnings (>8 hours without sync)

**For Agent (Feature Work):**

1. **Nutrition v3** — Redesign the current Nutrition page to Health v3 UI standards
2. **Daily Checklist** — Build UI for Water/Supplements/Sleep in `/health/more`
3. **My Foods & Products** — Quick reference UI (blocked on Health Coach)
4. **Health Coach context layer** — Safe tools for live AI interaction
5. **Real device testing** — Smoke tests on production iPhone/Watch

---

## Summary

**Phase 2 (Health v3 UI redesign) is live and operational.**

- **Today page:** Real calories, macros, activity, weight data via Apple Health + manual entry
- **Workout page:** Real Hevy training syncing every 37 minutes; exercise progress analytics live
- **More page:** Framework exists; awaiting Daily Checklist + My Foods & Products features
- **Weight logging:** Live with undo, history chart, trend tracking

**Integrations are operational:**
- Hevy: Syncing automatically, production tested
- Apple Health: Activity data reaching GYST; nutrition flow partially complete (MyFitnessPal sync verification needed)

**Next feature sequence:**
1. Verify Apple Health nutrition end-to-end
2. Nutrition v3 page redesign
3. Daily Checklist (Water, Supplements, Sleep)
4. My Foods & Products lightweight support
5. Health Coach LLM context layer
6. Real device QA

---

## Code References

- **Health Today:** `internal/handlers/health.go` + `health_view.go` + `web/templates/health.html`
- **Hevy Sync:** `internal/providers/hevy/`, `internal/services/training.go`, `internal/handlers/training.go`
- **Weight:** `internal/handlers/health_progress.go` + `web/templates/health_progress.html`
- **Models:** `internal/models/health.go`, `internal/models/training.go`
- **DB:** `internal/repository/` (health_* and training_* tables)
