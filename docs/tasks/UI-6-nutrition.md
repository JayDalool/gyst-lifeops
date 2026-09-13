# UI-6 — Nutrition detail

- **Status:** `IN PROGRESS`
- **Approval:** Jay requested continuation after UI-5 and handed Shortcut setup to another conversation.
- **Implementation commit:** Pending.
- **CI:** Pending.

## Scope

GET `/health/nutrition` reads the existing HealthToday service and retains session
protection. Health summary links to the detail page. The page presents calories,
protein, carbs, fat and fiber with independent consumed/target/availability
states. The calorie ring requires both observation and target; graphic fill
clamps at 100 while actual intake and over-target amounts remain visible.

The view formats negative remaining amounts as positive “over target” text.
Nutrition presence depends on nutrition fields, not an activity-only snapshot.
Snapshot age/staleness is explicitly snapshot-wide, not per-nutrient freshness.
Meal history, recipes and portion planning are unavailable states, not fabricated
records or new backend features. No schema/calculation/provider changes.

## Verification

Local format/vet/build/unit/race and Node shell/gesture checks passed.
Chromium and WebKit passed empty and populated views at 320, 390, 768, 844
landscape and 1440 px, including 200% text, active navigation, target editing
links and dark mode. Synthetic populated data included zero protein, missing
fiber, stale age and over-target calories. Screenshots were visually inspected.
Physical iPhone/Home Screen review remains pending.
Route/method and presentation tests cover missing,
activity-only, zero, partial, stale, over-target and targetless data.
Browser checks use a fresh temporary fixture database, never production.

## Next action

Jay: review UI-6 before proceeding to UI-7 Training.
