# 05 — Galaxy Specification

The Galaxy is GYST's signature visual metaphor and the one screen allowed to feel cinematic. It must still remain practical, accessible and fast on a phone.

## Existing behavior to preserve

The current implementation already has an interactive Three.js canvas, project data loaded from `/galaxy/data`, selectable project worlds, a project context panel, a conventional project list and a fallback when 3D cannot load. The redesign must preserve these capabilities rather than replacing Galaxy with a static mockup.

## Mobile layout

1. Compact dark top bar with `My Galaxy`, back/navigation where needed, and one secondary action such as List or Info.
2. Interactive stage occupying roughly 58–70vh.
3. GYST/core hub centered visually.
4. Project worlds distributed around the core. Domain grouping/rings are allowed only when they remain understandable at phone size.
5. Selected project opens a bottom-anchored context card or bottom sheet.
6. Bottom tab navigation stays visible unless the user explicitly enters an immersive Galaxy mode.

## Selection and navigation

- One tap selects a project world.
- Selection shows project name, category/domain, progress, next task/milestone, optional target date and an explicit `Open project` action.
- Navigation must not depend on double-tapping.
- One-finger drag orbits/roams.
- Pinch-to-zoom may be implemented if it does not harm scroll/gesture behavior.
- A clear List control always exists.

## Planet semantics

Visual size must not become an undocumented proxy for importance. State/progress should use visible text, ring/halo treatment and context details. Completed, active, upcoming and locked states should remain distinguishable without relying only on color.

Project-specific accent colors may exist inside Galaxy, but ordinary app pages continue using the restrained global system.

## Context card

Phone fields:

- project title;
- domain/category;
- progress percentage and bar;
- next mission/task;
- target date where available;
- `Open project` primary action;
- optional `View tasks` secondary action if route behavior supports it.

Desktop may render the same data in a side panel.

## Accessible fallback

The conventional list remains in the DOM and usable with keyboard/screen reader. If WebGL, Three.js or Galaxy data loading fails, the app clearly offers the project list rather than leaving an empty canvas.

## Performance rules

Three.js loads only on the Galaxy route. Renderer pixel ratio should be capped, for example at `Math.min(devicePixelRatio, 2)`. Respect `prefers-reduced-motion`. Animation should pause or substantially reduce when the document/stage is hidden. Avoid unnecessary high-poly geometry, large textures and post-processing effects.

The existing project-data fetch keeps a timeout/fallback path. Mobile touch handling must not trap page scrolling unexpectedly.

## Visual style

Galaxy may use dark navy/black space, subtle stars, restrained nebula color, orbit lines and soft project glows. The selected world may use a bright focus ring. Labels should remain readable and avoid overlapping excessively. GYST's core should be visually recognizable without becoming a giant glowing obstruction.

## Projects integration

The Projects family exposes `Galaxy | List | Roadmap | Completed`. Galaxy is a peer view of the same project data, not a disconnected feature. Selecting/opening a world leads to the same project detail used by List/Roadmap views.

## Mobile QA

Required checks include 320px, 390px and 430px widths; portrait orientation; low-power/reduced-motion behavior; touch selection; orientation/resize; fallback mode; long project names; many projects; zero projects; and returning to Galaxy after visiting a project detail.