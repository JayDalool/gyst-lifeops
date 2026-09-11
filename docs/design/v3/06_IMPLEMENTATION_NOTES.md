# 06 — Implementation Notes for Codex

## Scope discipline

UI v3 is documentation and presentation work. It must not be mixed into the M2.5/M3 database/security hardening commits. Implement screen families on a dedicated branch after the relevant hardening baseline is approved or intentionally rebased.

Hard rules: preserve existing routes/actions unless a separate route migration is approved; do not reset real data; do not weaken auth/same-origin/Health API behavior; do not change XP/reward semantics merely for presentation; and never use fake production data to make a planned screen look complete.

## Current production UI architecture

Production presentation lives under `web/templates/`, `web/static/css/styles.css`, `web/static/js/app.js`, `web/static/js/charts.js`, `web/static/js/galaxy.js`, and vendored Three.js modules. Keep this server-rendered architecture. The older `docs/design/v2/` remains historical reference; this v3 directory is the new UI decision source.

## Recommended template/component organization

Where the existing loader permits it, introduce reusable partials for the bottom tabs, top bar, Health tabs, metric cards, list rows, empty states and integration rows. If nested partials complicate the current Go template loader, keep files flat but use the same conceptual component boundaries.

Suggested production concepts: `layout.html`, `partials/bottom_tabs.html`, `partials/top_bar.html`, `partials/health_tabs.html`, ordinary page templates, and existing page-specific JS modules.

## CSS architecture

Use CSS custom properties for design tokens and semantic component classes rather than a large utility framework. Suggested classes include `.app-shell`, `.page-header`, `.section`, `.ui-card`, `.ui-row`, `.ui-button`, `.ui-segment`, `.metric-ring`, `.project-card`, `.course-card`, `.health-summary`, and `.galaxy-stage`.

All phone pages must reserve bottom padding for the fixed tabs and `env(safe-area-inset-bottom)`.

## Icons

Vendor a small Lucide SVG set locally or build a local SVG sprite. Do not load icons from a CDN. Icon-only buttons have visible tooltip/accessible labels and 44px targets.

## Charts

Extend the existing charts module rather than adding a framework. Rings can be SVG/CSS. Full trend charts must include a textual summary for accessibility.

## Galaxy

Keep the existing dynamic Three.js import and `/galaxy/data` source. Mobile changes should focus on stage sizing, touch behavior, renderer pixel ratio, context card/sheet, explicit project opening, reduced motion and animation suspension when hidden. Preserve the accessible list and fallback.

## Health sequencing

Stage H0: render the new Health shell using only currently supported server data and honest missing/stale states.

Stage H1: after Shortcut ingestion is production-ready, populate Apple Health/MyFitnessPal metrics and sync diagnostics.

Stage H2: after food/recipe services exist, enable meal planner, food search, recipe screens and Next Meal calculations.

Stage H3: after Hevy integration exists, enable Training, imported workout detail and strength history.

Stage H4: only after deterministic services are stable, add richer assistant/AI interactions.

## Hevy status

At the time this v3 spec was written there is no Hevy integration in the GYST repository. Do not bind templates to raw Hevy response fields yet. The future integration should normalize provider data into GYST training domain models and keep provider-specific JSON/auth details inside a Hevy adapter.

A future training model may include provider/provider ID, workout title, start time, duration, exercise count, working set count, volume and exercise history. Exact fields are decided during the Hevy milestone after reviewing the live API contract.

## Data freshness

Health presentation uses current, stale and missing states. A missing metric never becomes numeric zero unless zero was actually observed. A stale dietary intake must not produce a confident `calories left` value.

## Responsive QA

Each screen family is checked at 320×568, 390×844, 430×932, 768×1024 and 1440×900 where practical. Test light/dark mode, reduced motion, long content, empty states and stale/error states. Acceptance requires no horizontal page overflow, no bottom-tab overlap, no safe-area clipping, and no tiny touch targets.

## Recommended implementation order

1. Token layer and typography.
2. App shell and five primary tabs.
3. Today and task family.
4. Projects list/detail.
5. Galaxy mobile adaptation.
6. Courses family.
7. More, Settings, Goals, Reminders, Calendar, Analytics, Archive and Plan Import.
8. Health shell with currently supported data.
9. Health subpages unlocked in step with backend milestones.
10. Final accessibility, responsive, dark-mode and PWA polish.

Do not implement every template in one commit. Each family should be independently reviewable.

## Definition of done

A screen is done when existing functionality still works, core content remains understandable without JavaScript, mobile/tablet/desktop layouts are deliberate, light/dark themes work, keyboard and touch accessibility work, empty/error states exist, private-data caching rules remain safe, tests stay green and no planned feature is represented by fake production data.