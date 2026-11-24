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
# optional; default gpt-5.6-sol (GPT-5.6 flagship; gpt-6-astra is the newer, pricier option)
HEALTH_AI_MODEL=
```

Each provider uses only its own key. `HEALTH_AI_MODEL` must be a model of the
selected provider: clear it when you switch. The OpenAI adapter talks to
`https://api.openai.com/v1/responses` only (the address is fixed in code, no
environment variable changes it), follows no redirects and sends `store: false`
so OpenAI keeps no application state for the conversation.

Without both values for the chosen provider the Coach page says "AI provider
not configured", the Coach tab is hidden and the rest of Health is unchanged.

## Enable it from Settings (Issue #218)

Settings → **AI provider** sets the provider (OpenAI or Anthropic), the chat
model, the photo model (kept for the food camera) and the API key. No server
shell or restart is needed: the Coach turns on when a key is saved and off
when it is removed.

- **Write-only key.** The key is never shown again. The page says "Key saved
  on <date>" or "Not set". Saving or removing it asks for your GYST password
  again, and failed attempts share the sign-in throttle.
- **The environment wins.** If `HEALTH_AI_PROVIDER` or a provider key is set on
  the server, Settings says "Configured by the server environment" and the
  form is off; saved settings are ignored. Unset them to manage the key here.
- **Encrypted at rest.** The key is stored in the `ai_settings` table as
  AES-256-GCM ciphertext with a random 12-byte nonce (new on every save). The
  32-byte data key is a random file, `ai_settings.key`, in the same folder as
  the database (`data/` in Docker), created with mode 0600 on the first save.
  It is never logged. A key file that is missing, not 32 bytes, or readable by
  group or other is treated as "no key", and a save fails with a fixed message.
- **Tampering.** A ciphertext that fails authentication (or a missing key file)
  counts as not configured: the Coach stays off and Settings says the saved
  key can't be read and to enter it again.
- **Audit.** `settings.ai_key.set` and `settings.ai_key.remove`, with no
  detail. A wrong password is `reauth.failed`.
- **Backups and restore.** `lifeops backup` (VACUUM INTO) copies the database
  only, so a backup holds the ciphertext and never the data key. After a
  restore, either put the original `ai_settings.key` back beside the database
  (keep it somewhere private, with the same care as the database) or enter the
  API key again in Settings. Without the key file the saved key is unreadable,
  and the next save creates a new key file. Do not copy `ai_settings.key` into
  `data/backups/` or any shared place.

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

## Food photo analysis (service only, Issue #217)

`POST /health/nutrition/photo/analyze` (multipart field `photo`, session plus
same-origin) identifies foods in a meal photo. There is no page yet (#211 adds
the review screen) and nothing is written to the diary.

- Provider: OpenAI or Anthropic, whichever is selected in Settings (both
  implement the optional `FoodPhotoAnalyzer` capability, #243). OpenAI model:
  `gpt-6.1-sol`, or `HEALTH_AI_VISION_MODEL` when set. Anthropic model: the
  photo model saved in Settings (or `HEALTH_AI_VISION_MODEL`), otherwise
  `claude-opus-5-5`. The variable must also be passed through
  `docker-compose.yml` to reach the container. When switching provider, clear
  a photo model saved for the other one.
- The upload is held in memory only: no temp file, no disk. The type is sniffed
  from the bytes (JPEG and PNG only; HEIC and WebP get a fixed "use JPEG or PNG,
  set the iPhone camera to Most Compatible" message), dimensions are checked before decoding (at most 40 MP), the image
  is re-encoded as JPEG at 1024 px or less, which drops EXIF, GPS and anything
  appended to the file. Orientation is applied to the pixels first.
- Sent to OpenAI as `input_image` (base64 data URL) with strict structured
  output and `store: false`. GYST keeps nothing after the response.
- Sent to Anthropic as one base64 `image/jpeg` block through the Messages API
  with `output_config.format` (JSON schema) and effort `low`. The request holds
  only model, max_tokens, system, messages and output_config: no tools, beta
  header, fallbacks, metadata or diary data. A `refusal` or `max_tokens` stop
  reason never reaches the parser and gives the "unreadable" message.
- Anthropic money failures (Coach and camera, shown with status 502, never
  retried): 402 `billing_error` gives "The AI credit has run out. Add credit in
  the Anthropic Console, or switch provider in Settings."; the 400 "You have
  reached your specified API usage limits" and the 429 `enforced_spend_limit_reached`
  give "The monthly AI spend limit is reached. Raise it in the Anthropic Console,
  or switch provider in Settings." The 400 "Your credit balance is too low to
  access the Anthropic API" (prepaid balance used up) also gives the credit
  sentence; it is matched on status, error type and message prefix together.
- Browser checks: `go build -tags browsercheck` adds `GYST_BROWSERCHECK_ANTHROPIC_URL`
  (only `http://127.0.0.1:<port>/`, otherwise the program refuses to start) and
  `cmd/fakeanthropic`, a loopback fake (`-mode=ok|credit|spend`). The Dockerfile,
  Makefile, compose file and CI never use the tag.
- The model identifies only. Calories and macros are never requested, and any
  nutrient field it returns is discarded. Items are validated (roles, 0-3000 g,
  confidence 0-1, at most 15 items, string lengths); invalid items are dropped.
- Rate limit: one at a time, 3 s apart, 40 an hour (separate from the Coach).
- Coach and Secretary context (GYST-180): when today's counted food includes
  AI photo estimates, `get_health_today` carries `calories_kcal.ai_estimated`
  `{items, kcal, protein_g, note}` and the day brief `health.food.ai_estimated`;
  both are absent when there are none. They are aggregates of values already in
  the totals, with no food names, photos, ids or dates, and the note tells the
  model to treat the totals as approximate.
- Failures log a fixed category only, never the image, prompt or output.
