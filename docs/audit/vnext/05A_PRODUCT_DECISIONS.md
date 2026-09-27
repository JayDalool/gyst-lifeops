# Stage 4 — Product-Owner Decisions

- **Decided by:** Jay (product owner)
- **Recorded:** 2026-09-27
- **Responds to:** `05_INFORMATION_ARCHITECTURE.md` §"Decisions for Jay" (Stage 4 commit `9c9ebb7`, unchanged)
- **Status:** **Binding** input for all later vNext stages unless Jay explicitly changes them.
- **Precedence:** where this file differs from `04_PRODUCT_VISION.md` or
  `05_INFORMATION_ARCHITECTURE.md`, **this file wins**. Those documents are not edited.
- **Implementation:** none of these decisions is implemented yet. Navigation,
  templates, routes and tests are unchanged.

---

## Summary

| # | Topic | Decision | vs. Stage 4 recommendation |
|---|---|---|---|
| D1 | Courses label | **Learn** (Courses inside) | Approved |
| D2 | Development location | Inside Projects, as a **clear first-class secondary area** | Approved, with emphasis on prominence |
| D3 | Gamified terminology | Retire Mission, Quest, Destination, Trip as functional terms | Approved |
| D4 | Completed projects in Galaxy | Shown, visually distinct, and **filterable/hideable** | Approved with a constraint |
| D5 | Attention | **One** Attention model | Approved |
| D6 | System + Apps | Phone: More. Desktop: secondary sidebar group allowed. | Approved |
| D7 | Goals | **More → Goals** (own destination) | **Modified** |
| D8 | Unbuilt Health features | Remove from primary surfaces | Approved |
| D9 | Healthy system status | Nothing now; one quiet indicator later | Approved |
| D10 | Prayer on Today | Next prayer in Now; check-ins near the top | Approved |
| D11 | Target navigation | Today · Projects · Health · Learn · More | Approved (target, not implemented) |
| C1 | Attention access | A full destination, **without** a new permanent menu item everywhere | Additional constraint |

---

## D1 — Courses label

**APPROVED.** The primary navigation concept changes from **Courses** to
**Learn**. **Courses** remains a content type inside Learn.

Learn may eventually include courses, Master's study, structured learning,
resources and study progress.

**Constraint:** don't broaden the feature implementation yet. This decision
covers naming and conceptual scope only.

## D2 — Development location

**APPROVED.** Development lives conceptually inside **Projects / Work**, and
must be a **clear, first-class secondary area**. It must not be buried several
levels deep.

Conceptual hierarchy:

```text
Projects
├── Projects
├── Development
├── Galaxy
└── Roadmaps
```

Development may later contain Tasks / GitHub Issues, Reviews, Repositories and Agents.

**Note against `05`:** this places Galaxy and Roadmaps as **siblings** of
Projects and Development within the Projects area, where `05` §K modelled them
as view modes of the project list. The design stage must present all four as
peer, directly reachable areas of the Projects tab. Exact presentation (switch,
segments, sub-navigation) is left to design.

**Constraint:** don't implement navigation yet.

## D3 — Gamified terminology

**APPROVED.** Retire as **primary functional UI terms**: **Mission, Quest,
Destination, Trip.**

Use clear product language: **Project, Task, Roadmap** (see `05` §X for the full table).

Preserve GYST personality through **Galaxy, planets/worlds, the ship, XP**, and
visual metaphors.

**Rule:** functional clarity is never sacrificed for themed vocabulary.

## D4 — Completed projects in Galaxy

**APPROVED WITH CONSTRAINT.** Completed projects may be represented in Galaxy as
dimmed, distant or outer-orbit, or otherwise visually distinct.

**Constraints:**
- Completed projects **must be filterable/hideable**.
- Completed history must **not overwhelm** the active Galaxy.
- Exact visual behaviour is decided during design work.

## D5 — Attention

**APPROVED.** There is **one** Attention model.

**Don't** create competing conceptual sources such as a separate Inbox,
Notifications, Review Queue or Alerts.

Attention may surface through:
- Today
- a badge
- a full Attention page
- the launcher
- push, for urgent items only

**Rules:**
- The **underlying event or problem remains the source of truth**. Attention is a view over it.
- Items **resolve automatically** when their source condition resolves, where appropriate.
- Presentations such as a Development "Reviews" list are filtered views of the
  same model, not separate systems.

## D6 — System + Apps

**APPROVED.**
- **Phone:** System and Apps live under **More**.
- **Desktop:** they may appear as a **secondary sidebar group**.
- System status may surface elsewhere **only when relevant**.
- Healthy infrastructure must not dominate Today.

## D7 — Goals (modified)

**MODIFIED FROM THE STAGE 4 RECOMMENDATION.** Goals has its own clear destination:

```text
More
└── Goals
```

- Goals must **not** live primarily under a generic Review area (this replaces `05` §U's "Review → Goals" placement).
- A future weekly/monthly **Review** feature may *summarise* Goals, but Goals
  remains its own product concept.
- Goals may surface **contextually on Today when relevant** (this relaxes
  `04` §F, which had "Goals progress: never daily").

Resulting More structure (conceptual, for design):

```text
More
├── Goals
├── System        (Server · Automations · Backups · Integrations)
├── Apps
├── Review        (future: weekly/monthly summaries; Analytics · Archive)
├── Tools
└── Settings
```

The position of Analytics and Archive within More is left to design, as long as
Goals stays a direct destination.

## D8 — Unbuilt Health features

**APPROVED.** Remove unfinished functionality from primary surfaces. Don't show
primary controls for functionality that doesn't work yet.

Examples (current placeholders, per Stage 2B/3):
- Week / Month / Year
- Plan meal
- Start workout
- incomplete Trends (nutrition/activity, strength)
- weight input or entry while there's no working writer

**Principle:** a smaller complete Health product is better than a larger misleading one.

**Constraint:** **don't remove supporting backend work** merely because the UI
is hidden (e.g. `Service.CreateHealthWeightEntry`, Hevy provider and training
cache, weight history queries remain).

## D9 — Healthy system status

**APPROVED.**
- **Before the System subsystem exists:** Today shows **nothing** for system health.
- **Later:** a **single quiet healthy indicator** is acceptable.
- Problems get **more visual prominence** than normal operation.

## D10 — Prayer on Today

**APPROVED.**
- The **next prayer** belongs in the Today / **Now** context.
- **Prayer and Athkar check-ins** stay easy to reach in the **upper portion** of Today.
- The spiritual rhythm is preserved as an intentional LifeOps feature.

## D11 — Target navigation

**APPROVED.** The vNext target mobile navigation is:

```text
Today · Projects · Health · Learn · More
```

This is the approved **target information architecture**. It supersedes NAV1
as the *target*. NAV1 remains what's implemented until an approved
implementation task changes it.

**Constraint:** don't implement it yet.

---

## Additional product constraint

### C1 — Attention access

Attention must be accessible as a **full destination**, but **don't
mechanically add another permanent visible menu item everywhere**.

Reach it through context and signals:
- the Today section (when non-empty)
- a badge
- the launcher ("Attention")
- deep links from push and from items

This is consistent with `05` (no Attention tab). A dedicated sidebar or More
entry isn't required and shouldn't be added by default.

---

## Items from Stage 4 not changed by these decisions

Everything else in `04_PRODUCT_VISION.md` and `05_INFORMATION_ARCHITECTURE.md`
stands as the working basis, including:
- product principles
- Today's structure and signal rules (as amended by D7)
- progressive-disclosure levels
- launcher intents (Go / Do / Capture, with no implicit capture)
- state vocabulary
- the terminology table (as affirmed by D3)
- capability tiers
- migration order

Where design work needs to resolve open presentation questions (D2
sub-navigation, D4 visuals, D7 More layout), those are **design choices within
these decisions**, not reopened product decisions.
