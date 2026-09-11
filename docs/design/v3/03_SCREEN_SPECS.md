# 03 — Screen Specifications

This file defines the intended UI for the whole application. Existing features are redesigned; planned features remain visually specified but must not be treated as already implemented backend functionality.

## Global shell

Phone uses a fixed five-tab bar: Today, Projects, Health, Courses, More. Context-specific actions such as Add/Search belong in the page header. Desktop may convert the same information architecture to a rail or top navigation.

## Today — `/today`

Purpose: answer `What should I do today?`

Order: greeting/date, short focus message, Today's Checklist, compact Health Snapshot, Upcoming items, focused Projects, Next Up recommendation. Today should not contain full analytics.

## Tasks / Daily Checklist — `/tasks`

Header with title/date and add action. Optional segmented filter: Today, Upcoming, All, Completed. Group task rows by meaningful context or due date. Each row includes completion, title, optional time/project and only non-default priority.

## Task detail / task form

Detail page shows title/status, schedule, project/goal links, category/priority, checklist/repeat/proof metadata where those features already exist, and edit/complete actions. Forms are grouped into Basics, Timing, Context and advanced fields.

## Projects — `/projects`

Header with search/add. Segmented views: Galaxy, List, Roadmap, Completed. Project cards show icon/domain, title, roadmap/category, completion %, progress bar, next milestone/task and target date when useful.

## Galaxy — `/galaxy`

The interactive universe is the dominant object. Phone layout uses a compact dark header, large interactive stage, GYST core, project worlds, and a bottom context card for the selected project. Tap selects; an explicit Open Project action navigates. Drag orbits; pinch zoom may be supported if smooth. A List control and conventional project list remain available for accessibility and fallback.

## Project detail

Header with project title/status/edit. Progress hero shows completion, current phase and target. Sections or tabs: Overview, Tasks, Milestones, Notes where real data exists. Overview contains the project brief, next milestone and current roadmap context.

## Project form

Grouped sections for Identity, Outcome, Dates, roadmap/goal link and supported Galaxy visual settings.

## Roadmaps — `/roadmap`

Phone uses a roadmap selector, current milestone summary and vertical milestone path. Desktop may use a selector/sidebar with main roadmap. Current milestone is visually strongest.

## Courses — `/courses`

Header with search/add. Segments: In Progress, Library, Completed. Course cards show title, subject, progress, next lesson and next scheduled study block when available.

## Course detail / form

Detail shows title, subject, progress and next action first. Lessons are clear rows with status/title/duration. Notes/resources appear only where backed by real data. Course forms group basics, schedule and structure.

## Goals — `/goals`

Active goals first, completed separate. Goal row/card shows title, type, progress, current/target values and due date when relevant. Detail shows progress history and linked work.

## Reminders — `/reminders`

List-oriented page. Optional views Today, Scheduled, All. Rows show title, next date/time, recurrence and state.

## Monthly / Calendar — `/monthly`

Month strip or grid at top, selected-day agenda below. Tasks, reminders, courses and other event types use small semantic markers instead of large rainbow cards.

## Analytics — `/analytics`

Period selector Week, Month, Year. Summary first, then a small number of meaningful charts for completion, project/course progress and life-domain trends. Avoid decorative chart types.

## Archive — `/archive`

Filter by type. Rows show item title, type and archived/completed date. Restore actions appear only where supported.

## Plan Import — `/plan-import`

Input/paste area followed by preview of items to create. User confirms before writes. Parsing errors identify the affected input instead of silently dropping content.

## More / Settings

More groups secondary features: Reminders, Goals, Calendar, Analytics, Archive, Settings and Plan Import. Settings groups General, Appearance, Notifications, Prayer configuration, Calendar integration, security when implemented, and diagnostics when intentionally exposed.

## Global search / command palette

Searches routes and existing entities, preserving quick-task capture. Desktop can remain a dialog/command palette. Phone should use a full-screen search sheet.

## Login — planned

Simple page containing product identity, password field, primary sign-in action and clear error state. No private application data appears before authentication.

## Offline / error / empty states

Offline state explicitly says data may be stale. 404 provides `Go to Today`. Server errors show a plain retry/back path without implementation details. Empty lists provide the relevant single creation action.

## Implementation rule

A screen is complete only when existing behavior remains available, the layout works on phone/tablet/desktop, touch and keyboard use are supported, light/dark themes work, empty/error states exist, and no hard-coded fake data is used to imitate a backend feature that does not exist yet.