# Health Coach

The Coach answers nutrition, training and "how do I use GYST" questions from
live Health data. Issue #64.

## How it works

- GYST builds a small context: stable rules, the profile you typed (More →
  Profile for the coach) and today's observed summary (including today's water: litres logged or null, taps_today, target, bottle size).
- The model can call allowlisted, **read-only** tools: today, plan, weight
  trend, training summary, recent workouts, exercise progress, Hevy routines,
  My foods and the daily checklist. That's all it can see. It can't reach
  tasks, projects, prayer, courses, settings, SQL, files, the shell or URLs.
- The model can attach one button from a fixed list of Health pages
  (`internal/healthcoach/provider.go`, `Destinations`). It can't render links of
  its own.
- Visible turns are stored in `health_coach_messages`. The last 8 go back to
  the model; **Clear chat** deletes them all. No vendor-side chat storage.

## Enable it

Set these in the server's private `.env`, then recreate the container:

```
HEALTH_AI_PROVIDER=anthropic
ANTHROPIC_API_KEY=<your key>
# optional; default claude-opus-5-5
HEALTH_AI_MODEL=
```

Without both values the Coach page says "AI provider not configured", the
Coach tab is hidden and the rest of Health is unchanged.
`HEALTH_AI_PROVIDER=openai` is accepted but stays disabled until the OpenAI
adapter exists.

## Privacy and limits

- Calls happen only on the server. The key never appears in HTML, JS,
  browser storage, logs or error messages.
- What leaves GYST per question: the rules, your profile, today's summary,
  recent visible turns and whatever tool results the model asks for (Health
  data only). Today's water is part of the summary: litres logged (null when
  nothing is logged), bottle taps, target and bottle size.
- Prompts and replies are not logged. Vendor errors are replaced by fixed
  messages.
- Anthropic's API doesn't train on API data by default. Retention follows
  your Anthropic organisation's settings.
- One request at a time, at least 3 s apart, at most 40 an hour.
- Cost: each question is one to six Messages API calls on the configured
  model at effort `low`.
- General guidance only, not medical advice.
