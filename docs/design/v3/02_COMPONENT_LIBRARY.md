# 02 — Component Library

GYST v3 uses a small reusable component set across all pages.

## Shell components

`AppShell` owns safe areas, page padding and responsive navigation. `TopBar` contains back navigation, title and at most two page actions. `BottomTabBar` contains exactly Today, Projects, Health, Courses and More.

## Structure components

`PageTitle` is the main heading. `SectionHeader` is a section title with an optional single action. `Card` is the standard grouped surface. `ListGroup` and `ListRow` are the default for settings, reminders, integrations and compact information.

## Action components

`PrimaryButton` is filled with the main tint. `SecondaryButton` is outlined or tint text. `IconButton` always has a 44px target and an accessible label. `SegmentedControl` switches peer views. `FilterChip` is only for optional filters.

## Work components

`TaskRow` contains completion state, title, optional time/project/priority and detail navigation. `ProjectCard` contains title, domain/category, progress and next milestone. `CourseCard` contains title, subject, progress and next lesson.

## Data components

`ProgressBar` is 4–8px thick. `MetricRing` is SVG/CSS and represents one dominant target. `MetricTile` contains one large value, label and optional trend. `MacroBar` uses stable colors: protein blue/cyan, carbs orange/yellow, fat purple, fiber green. `MiniTrendChart` is a sparkline; `TrendChart` is a full chart with a text summary.

## Health components

`NutrientSummaryCard` combines calories and macros while respecting freshness. `NextMealCard` shows the next meal budget and planning action. `WorkoutHeroCard` is reserved for the future Hevy-backed training summary. `ActivityOptionRow` represents walking, swimming, cycling and similar alternatives without promising exact calorie burn. `Plan90DayCard` summarizes the active health cycle. `IntegrationStatusRow` shows provider state and last sync. `SyncHealthCard` shows ingestion freshness/diagnostics when supported.

## Galaxy components

`GalaxyWorld` is a Three.js project planet. Progress/state should be communicated by ring, halo and text rather than arbitrary size. `GalaxyContextCard` appears as a bottom card/sheet on phone and a side card on desktop. The conventional accessible project list remains available at all times.

## Feedback components

`EmptyState` contains a short title, sentence and one useful action. `ErrorState` explains the problem and gives a retry/back action where useful. `Toast` is short transient feedback only. `BottomSheet` is used for compact mobile choices and must be accessible. `ConfirmationDialog` is reserved for important irreversible actions.

## Form standards

Labels are always visible. Inputs and controls are at least 44px high. Related fields are grouped. Validation appears beside the relevant field and never depends on color alone. Health numeric fields display units clearly.

## Suggested CSS names

`app-shell`, `top-bar`, `bottom-tabs`, `page-title`, `section-header`, `ui-card`, `ui-row`, `ui-button`, `ui-segment`, `metric-ring`, `metric-tile`, `macro-bar`, `mini-trend`, `project-card`, `course-card`, `workout-hero`, `integration-row`, `activity-row`, `empty-state`, `bottom-sheet`, `confirm-dialog`.
