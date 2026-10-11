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

For OpenAI instead:

```
HEALTH_AI_PROVIDER=openai
OPENAI_API_KEY=<your key>
# optional; default gpt-6-astra (OpenAI's flagship)
HEALTH_AI_MODEL=
```

Each provider uses only its own key. `HEALTH_AI_MODEL` must be a model of the
selected provider: clear it when you switch. The OpenAI adapter talks to
`https://api.openai.com/v1/responses` only (the address is fixed in code, no
environment variable changes it), follows no redirects and sends `store: false`
so OpenAI keeps no application state for the conversation.

Without both values for the chosen provider the Coach page says "AI provider
not configured", the Coach tab is hidden and the rest of Health is unchanged.

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
- OpenAI: per its data-controls guide, data sent to the API isn't used to train
  models unless you opt in, and abuse-monitoring logs are kept up to 30 days.
  GYST sends `store: false`, so the 30-day Responses application state isn't
  created. Source: https://developers.openai.com/api/docs/guides/your-data
  (checked 2026-10-11). Zero Data Retention needs OpenAI approval.
- One request at a time, at least 3 s apart, at most 40 an hour.
- Cost: each question is one to six API calls on the configured provider's
  model at effort `low`.
- General guidance only, not medical advice.
