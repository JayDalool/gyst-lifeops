# M3 — Apple Health Shortcut ingestion

- **Status:** PENDING — existing phone Shortcut; HTTPS request and nutrition validation pending
- **Scope:** Jay requested calories and macros from MyFitnessPal, including logged
  recipe servings. A recipe library is explicitly out of scope.
- **Backend:** Already implemented by Health Core and A5; do not create a second
  ingestion endpoint or duplicate the existing storage.

## Current evidence

Trusted HTTPS is deployed, application authentication is enabled, and Jay
confirmed iPhone login without a certificate warning. Jay also confirmed an
existing GYST Shortcut that reads real Health samples rather than fixed test
numbers. The agent has not inspected that Shortcut or observed its response.
A2's Shortcuts-specific check remains open until its HTTPS request succeeds.

## Existing contract to match

POST to the deployed HTTPS hostname on port 3443, path `/api/health/sync`.
Keep the existing private Bearer token on the phone; never paste it in chat or
commit it. Send a JSON object, with `source` equal to `apple_shortcuts` and a
calendar `date` in `yyyy-MM-dd` matching the day being aggregated.

| Apple Health measurement | JSON field | Numeric unit |
|---|---|---|
| Dietary Energy | dietary_calories | kcal |
| Protein | protein_g | g |
| Carbohydrates | carbs_g | g |
| Total Fat | fat_g | g |
| Dietary Fiber (optional) | fiber_g | g |

Each value must be the day's total, not a single sample or the change since the
last sync. Verify the Shortcut's Health source filter selects MyFitnessPal for
nutrition, so another app's copy of the same meals is not added again. Verify
units explicitly. Date boundaries must match the intended local calendar day;
GYST's Today page uses its configured timezone (default America/Winnipeg).

Missing samples or denied read access do not establish zero intake. Omit those
fields; never convert an empty result to zero. Only send numeric zero when the
source actually records zero. If there are no received metrics, do not POST.
Omitted/null fields preserve earlier received values; they do not clear them.
The current API has no clearing operation, so deleting all source samples will
not automatically erase an earlier total in GYST.

Retries replace that date's received totals, without accumulating them. The API
returns the merged snapshot and `last_synced_at`; this timestamp confirms a
successful request, not that every nutrient was received. Other retained
metrics may appear in the merged response even if this run did not send them.

Watch and phone steps remain separate if included in the existing Shortcut.
Never add them, and never add workout/step calories on top of active energy.

## MyFitnessPal connection

In MyFitnessPal: More (or profile) → Settings → Sharing & Privacy → HealthKit
Sharing. Grant the relevant nutrition writing permissions, and grant Shortcuts
permission to read those nutrition categories. Existing sharing permissions can
be managed through iPhone Health settings → Data Access & Devices → MyFitnessPal.
Confirm actual samples appear in Apple Health before diagnosing GYST.

MyFitnessPal documents meal summaries and most nutrients flowing to Apple
Health, not full recipe definitions or meal timestamps. Verify a logged recipe's
nutrition in Apple Health and then GYST using actual observations.

Source: [MyFitnessPal Apple Health connection and syncing](https://support.myfitnesspal.com/hc/en-us/articles/360032271092-Apple-Health-connection-and-syncing).

## Acceptance criteria

- [x] Trusted HTTPS deployed; real iPhone browser login reported successful
- [x] Existing Shortcut reported to read real Health samples
- [ ] Existing Shortcut successfully POSTs to the new HTTPS URL (closes A2)
- [ ] Date, source filter and numeric units inspected/verified on phone
- [ ] Calories, protein, carbs and fat actually arrive and match Apple Health
- [ ] Logged MyFitnessPal recipe serving contributes its nutrition once
- [ ] Re-running unchanged samples leaves the same totals
- [ ] Missing nutrition remains absent; observed zero remains zero
- [ ] Automation tested on the actual phone; failed/locked/offline runs are not
      represented as successful syncs

## Automation after manual validation

Configure a personal automation to run the existing Shortcut after food logging
and optionally at selected times. Test on the actual iPhone. Do not promise
continuous background sync: connectivity and access to Health data are required.
Apple states that Health data is inaccessible by default while the phone is
locked. A manual retry with the phone unlocked is the initial fallback.

Sources: [Apple Shortcuts automations](https://support.apple.com/guide/shortcuts/add-automations-apdfbdbd7123/ios),
[Apple Health privacy](https://www.apple.com/legal/privacy/data/en/health-app/).

## Next action

Jay updates the existing Shortcut's URL, runs it unlocked with Tailscale
connected, and reports success (`last_synced_at`) or the error without sharing
the Bearer token. Then verify nutrient presence and totals in `/health/nutrition`.
