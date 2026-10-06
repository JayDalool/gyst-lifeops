# Health Today redesign (#70)

- `approved-mockup.jpg`: the dark-mode direction Jay approved on 2026-10-05.
- `index.html`: the prototype. It uses the real `ui-tokens.css` and
  `ui-health-v3.css` with fictional data. Choose a state with `?state=A`…`G`
  and a theme with `&theme=dark`.
- `states-390.jpg`: states A–G at 390 px plus A in dark mode (state H).
  `states-430-768.jpg` shows C at 430 px and E at 768 px, dark.
  `state-A-1280-light.jpg` and `state-C-1280-dark.jpg` show desktop.

| State | Shows |
|---|---|
| A | Normal day |
| B | Near the food target (amber, "Close to budget") |
| C | Over the food target, lightly active (red, "Over budget", fat/carbs over) |
| D | Low activity day (steps nudge) |
| E | Hevy workout done (recovery note instead of a nudge) |
| F | Stale Apple Health data (banner, no nudge) |
| G | No nutrition logged yet, no weight, no checklist |
| H | Dark mode (A in dark) |

## Kept from the mockup, and changed on purpose

- **Kept:**
  - Ring with the food budget left.
  - Eaten / burned / balance rows with their own colours.
  - Macro tiles in category colours.
  - Steps and weight cards.
  - Checklist summary with round toggles.
- **Not copied:**
  - "~400 cal" walk estimates and "400 calories to go! Go for a walk to close
    the gap". GYST has no active-energy goal, so any burn target would be
    invented, and food is never something to work off.
  - "Net left to reach goal" (target + burned). The balance shown is
    eaten − burned, labelled "Observed, not final".
- **Colours:** the mockup made "more than 150 left" orange, which would be
  amber all morning. Here, green means within budget, amber means the last
  tenth of the target, and red means over. Every state also has words and an
  icon.
- **Header:** "Health · date / Today", as Jay's brief asked, instead of a
  greeting and a status line. The calorie card's state pill is the status, so
  it isn't repeated.
- **Nudge:** decided on the server (`internal/handlers/health_home.go`):
  - It never shows with stale or missing data, before 11:00, from 21:00, after
    30+ exercise minutes, or once the step goal is reached.
  - After a Hevy workout it becomes a recovery note.
