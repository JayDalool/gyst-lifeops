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

Reference: [official Hevy API documentation](https://api.hevyapp.com/docs/).
