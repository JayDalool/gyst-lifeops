# Connect Hevy

Hevy's public API currently requires **Hevy Pro**. Generate a key at
[Hevy developer settings](https://hevy.com/settings?developer).

Set `HEVY_API_KEY` in the server's private deployment environment (the existing
untracked `.env` used by Docker Compose). Never paste the key into chat, commit
it, or enter it into a public URL. Recreate the app container with
`docker compose up -d --no-build gyst-lifeops` after configuring it.

Open https://gyst.zaylolabs.com/health/training . The first import starts when
the server starts. Automatic refresh runs every 37 minutes; “Sync now” provides
an immediate refresh (minimum one minute between attempts).

GYST reads your exercise catalog, routines and workout history. It never edits
Hevy. Failed imports preserve the last complete snapshot and show an error.
Imports are bounded to 1,000 pages per endpoint and three minutes total; a
history exceeding these limits reports a failure instead of silently truncating.
The API key remains on the server, never in page HTML or browser storage.

The muscle map counts primary work sets (normal, failure and dropset), excluding
warmups. Secondary involvement is reported separately. Routine views represent
one pass through the chosen routines, not weekly training frequency. Workout
views count sessions starting in the selected rolling period. Unmapped exercises
remain unclassified; no guessed muscle assignments or calorie calculations.

## Treadmill incline and speed (exercise notes)

Hevy's API has no incline or speed field. For a treadmill (or any timed cardio)
exercise, write them in that exercise's **notes** in Hevy. GYST reads them
(read-only) and shows `20 min · 8% · 5.0 km/h` on the workout page, a cardio
trend on Progress (last 8 weeks), and gives them to the Coach.

Accepted, case-insensitive:

- `incline 8% · 5.0 km/h`
- `incline 8 5 kmh` (units `km/h`, `kmh`, `kph`)
- `incline 10% 3.1 mph` (mph is converted to km/h, rounded to 0.1)
- `8%/5.0` (percent, slash, speed; km/h when no unit is given)
- Comma decimals work (`5,5 km/h`). Separate per-set tokens with spaces.
- Incline is a percent: `incline 30 degrees` is ignored.

Which sets they apply to:

1. The **keyword form** (`incline ...`) applies to every timed set of the exercise.
2. **Per-set tokens** (`8%/5.0`): when the number of tokens equals the number of
   timed sets, the nth token belongs to the nth timed set (for example
   `6%/4.5 8%/5.0 10%/5.5` for three intervals).
3. Anything else gives **no** incline or speed for any set: keyword and tokens
   that disagree, different keyword values, or a token count that does not match
   the timed sets.

A bare percent is never an incline (`felt 20% harder` is ignored); write the
word `incline` or use the `8%/5.0` form. Incline must be 0-40% and speed
0.5-25 km/h; anything else, or notes GYST cannot read, leaves the value absent
(never shown as 0). Duration always comes from Hevy's own set duration.

Reference: [official Hevy API documentation](https://api.hevyapp.com/docs/).
