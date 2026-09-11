# 04 — Health Screen Specifications

Health is the most data-rich area of GYST. It should feel closer to Apple Health in hierarchy and clarity, while using GYST's own components and data model.

Important implementation state: GYST does not currently contain a Hevy API integration. Training screens below are therefore a visual/product contract for the later Hevy milestone, not permission to fake imported workout data.

## Health navigation

Primary Health tabs: Summary, Nutrition, Training, Progress, Plan. Summary may also use Day, Week, Month, Year. Each period changes the question being answered rather than simply stretching the same cards.

## Health Summary — `/health`

### Day

1. Today's Energy hero: dietary calories against target, food budget remaining only when dietary data is current, observed active energy, observed resting energy, observed total burn, and correctly labeled energy balance.
2. Nutrition card: protein, carbs, fat and fiber bars.
3. Today's Training card: future Hevy-backed workout summary or an honest not-connected/no-workout state.
4. Compact metrics: steps, active energy, exercise minutes, latest weight.
5. Next Meal card when meal-planning data exists.
6. Sync freshness/diagnostic warning only when useful.

Do not merge `food calories left` with `energy deficit`. They are separate concepts.

### Week

Show adherence and consistency: days on calorie target, days meeting protein target, workouts completed versus target, average steps, average active energy, weight trend and selected strength trend where available.

### Month

Show trend-level summaries: weight change, average calories, average protein, workout count, cardio/activity count, average steps and selected exercise progression.

### Year

Show long-term weight curve, training/session count, activity average and selected strength progression. Keep charts few and interpretable.

## Nutrition — `/health/nutrition`

Top area: date/day selector, calories consumed/target, food budget remaining if current.

Macros use stable colors and always show consumed/target values. Meals are grouped Breakfast, Lunch, Dinner, Snacks when data exists. Each meal row shows calories and a short macro summary.

Next Meal card shows the planned meal, approximate budget available, protein/fiber still needed and a Plan Meal action.

Recent foods/recipes may appear below. `Insights` should remain deterministic until an assistant phase exists.

## Meal Planner — planned

Inputs: meal type, current remaining daily targets, selected food/recipe, optional desired meal budget.

Outputs: suggested grams, calories/macros for that serving, and after-meal remaining targets. Source quality is visible: Saved Recipe, nutrient database, or Estimate.

Primary actions: Use Serving and Adjust Grams.

## Food Search — planned

Search is dominant. Rows show food name, brand/source, calories per 100g and optional protein. Source badges remain visible but quiet. Filtering may include Saved, Canadian Nutrient File, USDA and other enabled sources.

## Recipe Detail — planned

Show title/source quality, per-100g nutrition, serving calculator, ingredients, total cooked weight and actions for Plan Meal, Edit and Duplicate.

## Recipe Create/Edit — planned

Flow: name/details, ingredient rows with grams, total finished cooked weight, computed review per 100g, save. User should not manually calculate nutrition totals.

## Training — planned Hevy integration

Today's Workout is the hero: routine/workout name, expected duration if known, exercise count, schedule state, and primary Open in Hevy/Start Workout action where supported.

Below: This Week completion, Recent Workouts, Strength Progress, and Integration Status.

GYST is not the canonical live set logger while Hevy is the chosen training provider. Imported workout history is first treated as read-only normalized data.

## Workout Detail — planned

Show workout title/date/duration, exercises, sets with weight/reps and RPE/RIR only if Hevy provides them, total sets/volume, PR markers, provider source and Open in Hevy action where deep-linking is supported.

## Progress — `/health/progress` planned

Top dimension selector: Weight, Strength, Activity, Nutrition. Secondary time selector: 1M, 3M, 6M, 1Y, All where data supports it.

Weight: latest weight, change and line chart. Strength: selected exercise progression using direct observed values; estimated 1RM only if formula is documented and clearly marked estimate. Activity: steps, active energy, exercise minutes/sessions. Nutrition: calorie/protein adherence averages and days on target.

Each chart requires a text summary.

## 90-Day Plan — `/health/plan`

Hero: cycle title, inclusive date range, week count and progress. Sections: Goal, Nutrition Targets, Training Targets, Activity Targets, Energy Target, Notes.

The UI may ask for a positive Target Deficit while the internal signed energy-balance model stores a deficit as negative. A surplus goal needs an explicit future UI rather than pretending a positive-deficit field represents both.

## Activity Options — planned

Rows: Walking, Running, Cycling, Swimming, Elliptical, Other Activity. Each shows intensity and an optional estimated duration/range. The framing is optional activity to support the configured plan, not punishment for food.

A recommended active-energy gap is an estimate. The Apple Watch workout is the preferred actual measurement source. Do not claim an exact calorie burn in advance.

## Weight History

Latest value and change at top, line chart, then entries. Manual Add Weight remains available even if Apple Health body-mass sync is later introduced. Source appears on entries where multiple sources exist.

## Integrations

Apple Health via GYST Shortcut: connection/instructions, last sync and diagnostics. MyFitnessPal: explained as the nutrition source flowing through Apple Health unless a direct supported integration is deliberately added later. Hevy: future API provider row with connection state and last successful sync.

Secrets/tokens are never continuously displayed after setup.

## Freshness rules

Current metrics display normally. Stale metrics display last value plus timestamp/warning. Missing metrics show no data, never synthetic zero. `Calories left` should not be computed from stale/missing dietary intake as if trustworthy.

## Future TrainingProvider contract

The UI should depend on normalized GYST domain objects rather than raw Hevy JSON. A future provider interface can expose workouts, routines/exercise history and sync state. Provider-specific details stay in the adapter and Integrations screen.