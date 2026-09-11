# GYST UI/UX Design System v3

Status: approved visual direction and implementation reference.

The v3 direction combines Apple-like clarity and spacing, openGym-style workout usability, and the existing GYST Galaxy identity. The primary target is the iPhone PWA, with deliberate tablet and desktop adaptations.

## Primary navigation

The five primary destinations are Today, Projects, Health, Courses, and More. On iPhone these appear in a fixed bottom tab bar. Wider layouts may use a rail or top navigation while keeping the same labels and route meaning.

## Reference documents

`GYST_UI_V3_STANDARDS.md` is the canonical consolidated design reference. It is the approved high-level document and wins where it and a split document below disagree. The numbered files remain the working detail for each area.

- `GYST_UI_V3_STANDARDS.md` — canonical consolidated standard: principles, tokens, components, screen specs and implementation guide in one document.
- `01_VISUAL_SYSTEM.md` — colour, type, spacing, responsive rules, accessibility and approved libraries.
- `02_COMPONENT_LIBRARY.md` — reusable UI components and required states.
- `03_SCREEN_SPECS.md` — screen-by-screen reference for the whole application.
- `04_HEALTH_SPECS.md` — detailed Health, nutrition, training, progress and 90-day-plan screens.
- `05_GALAXY_SPEC.md` — mobile and desktop Galaxy interaction and layout.
- `06_IMPLEMENTATION_NOTES.md` — mapping into the current Go templates, CSS and JavaScript architecture.

## Visual references

The approved concept boards are stored under `docs/design/v3/assets/` when available. Written specifications take priority if a visual board and the written reference differ.

## Product principle

GYST should feel calm, native, legible, and information-dense without feeling crowded. Each page has one dominant element: today's actions on Today, the universe/map on Galaxy, the current health state on Health, active work on Projects, and active learning on Courses. Supporting information stays visually quieter.

This v3 folder supersedes v2 for new UI decisions. The existing `docs/design/v2/` remains historical reference.