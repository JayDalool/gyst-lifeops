# Stage 7 — Design Decisions

- **Decided by:** Jay (product owner)
- **Recorded:** 2026-09-27
- **Responds to:** `12_SCREEN_ARCHITECTURE.md` §"Design decisions for Jay" (DD-1–DD-4) (Stage 7 commit `ce9b61b`, unchanged)
- **Status:** **Binding** for all later vNext implementation work wherever it differs from the Stage 7 recommendations.
- **Implementation:** none. No production code, CSS, templates or data changed.

---

## Summary

| # | Topic | Decision | vs. Stage 7 recommendation |
|---|---|---|---|
| DD-1 | Default appearance | **System** default; user setting **System / Light / Dark**; live reaction to OS changes | Approved, with an explicit three-way setting |
| DD-2 | Today Rhythm strip | Prayer · Athkar · gym/training check-in; compact; **not configurable initially** | Approved, with no widget framework |
| DD-3 | XP on Today | Very small, quiet, tappable level indicator | Approved with restraint |
| DD-4 | Violet identity accent | Identity only; **never** a semantic state colour | Approved |

---

## DD-1 — Default appearance

**APPROVED.**

| Rule | Requirement |
|---|---|
| Default | Follow the operating-system appearance preference (**System**) |
| Setting | One user setting with three values: **System · Light · Dark**; System is the default |
| Persistence | A manual override (Light or Dark) persists until changed |
| Live change | In System mode, the app reacts correctly when the OS appearance changes while GYST is open (no reload required) |
| Single source | One appearance setting for the whole app. **No independent appearance settings** in different GYST areas. |
| Galaxy | May keep its intentionally immersive dark treatment regardless of the surrounding app appearance, where appropriate |

Effect on Stage 7: `11` §J ("follow the system, with a manual override stored
per device") is made concrete as this three-value setting. The production
`data-gyst-theme` toggle (light/dark only) is superseded when implemented.

## DD-2 — Today Rhythm strip

**APPROVED.** For the initial vNext design, Rhythm contains:
- prayer (the five daily prayers)
- Athkar
- gym / training check-in

Requirements:
- **Keep it compact.** One strip; it must not become a large dashboard section.
- **Configurability isn't required** for the initial vNext implementation. It may come later.
- **Don't build a generic widget framework** to support future customisation.
- **Prayer remains an intentional, first-class LifeOps feature.** Placement stays as approved in `05A` D10: next prayer in Now, check-ins in the upper portion of Today.

## DD-3 — XP on Today

**APPROVED WITH RESTRAINT.** XP/level may appear on Today as a **very small
identity/progress element**, e.g. `Lv 12` or a compact XP indicator.

| Requirement | |
|---|---|
| Visually quiet | Small chip in the header area (as in the prototype) |
| Doesn't compete with **Now** | Never larger or more prominent than the Now block |
| Doesn't compete with **Attention** | Never uses attention/state colours or badges |
| Not a game dashboard | No large XP cards, progress banners or motivational blocks on Today |
| Tappable | May open additional progress detail (e.g. a sheet with level progress and recent achievements) if useful |

XP remains part of GYST's personality, not the product's primary purpose.

## DD-4 — Violet identity accent

**APPROVED.** Violet is GYST's **identity accent**.

**Allowed uses:**
- Galaxy-related identity (entry points, orb/orbit motifs)
- the subtle brand mark (orbit dot beside the wordmark)
- XP/level identity
- selected decorative elements
- restrained highlights

**Violet must never be a semantic state colour.** It may not mean:
- success
- healthy
- warning
- error
- blocked
- destructive
- needs attention

Semantic states keep their own colours **and** icons/words (`11` §I:
ok/info/warn/crit/idle). This distinction must hold in **both light and dark mode**.

Implementation check: in `11` §H the `--xp` / `--xp-soft` tokens are the only
violet tokens. They must not be used by status chips, badges, banners,
Attention severity or state dots.

One prototype detail to note: the Agents prototype uses the violet chip for the
"Reviewing" state. Under DD-4 that is **not allowed**. "Reviewing" is an
active/in-progress state and uses the **info** (blue) state colour like "Working",
distinguished by its word and icon. Implementation follows this decision, not
the prototype.

---

## Design source of truth

After this file, the design hierarchy is:

```text
13A_DESIGN_DECISIONS.md            ← overrides where applicable
    ↓
11_DESIGN_SYSTEM_VNEXT.md
12_SCREEN_ARCHITECTURE.md
13_RESPONSIVE_INTERACTION_SPEC.md
    ↓ (reference implementation, non-binding where it conflicts)
docs/audit/vnext/prototypes/
```

These remain binding alongside it for product, integration and security scope:
`05A_PRODUCT_DECISIONS.md`, `08A_INTEGRATION_DECISIONS.md`, `10A_SECURITY_DECISIONS.md`.

**Older design documentation is historical/context material only** and must not
override the vNext documents. That includes `docs/UI_SYSTEM.md`,
`docs/UI_SCREEN_MAP.md`, `docs/design/v2/`, the `docs/design/v3/` material on the
`docs/ui-v3-standards` branch, and the reference boards in `docs/`.

## Implementation discipline

Future implementation agents **must not invent** alternate:
- spacing systems
- radii
- navigation
- state colours
- page hierarchy
- component patterns

without an explicit task that requires a design change.

If implementation finds a **genuine conflict or an impossible design
requirement**, the agent **reports it** (in its task record and to Jay) instead
of silently creating a new pattern. A resolved conflict is recorded as a new
decision entry, not an ad-hoc change.
