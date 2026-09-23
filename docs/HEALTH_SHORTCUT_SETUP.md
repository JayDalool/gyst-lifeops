# GYST nutrition Shortcut: phone setup

The requested public address is `https://gyst.zaylolabs.com`. The server origin
is ready; do not use that address for sync until its Cloudflare route has been
updated and the login page works. Keep the existing working Shortcut until the
new setup has been checked. This guide syncs calories/macros, not recipe text.

## 1. Finish the domain route

In Cloudflare, find Networking → Tunnels (older UI: Zero Trust → Networks →
Tunnels), open the existing tunnel, and edit the published route for
`gyst.zaylolabs.com`:

- Service type: HTTP
- Service URL: `gyst-caddy:8081`
- Additional application settings → HTTP settings → HTTP Host Header:
  `gyst.zaylolabs.com`

Save. The origin is internal to Docker; neither 8081 nor the old 3005 is
published. Cloudflare provides public HTTPS. Open the public `/login` page and
sign in before editing the phone's sync URL. Do not disable certificate checks.

## 2. Connect MyFitnessPal to Apple Health

1. Open MyFitnessPal → More (or profile) → Settings → Sharing & Privacy →
   HealthKit Sharing.
2. Allow writing Dietary Energy, Protein, Carbohydrates and Total Fat. Enable
   Dietary Fiber too if desired.
3. For an existing connection, manage permissions in iPhone Health settings →
   Data Access & Devices → MyFitnessPal. Depending on iOS, the same app list is
   available from the Health app's profile → Apps.
4. Open today's diary and let MyFitnessPal finish syncing.
5. In Apple Health → Browse → Nutrition, check each nutrient for today's date.
   Inspect its data entries/source: the required values must come from
   MyFitnessPal. If they are missing here, fix this connection before building
   the GYST Shortcut. A saved recipe only contributes nutrition after a serving
   has actually been logged in the diary.

[MyFitnessPal's connection instructions](https://support.myfitnesspal.com/hc/en-us/articles/360032271092-Apple-Health-connection-and-syncing).

## 3. Build one nutrient Shortcut first

Use separate small Shortcuts for each nutrient, then combine them. The server
accepts partial daily totals, so missing protein cannot erase synced calories.
Labels can vary by iOS version; do not replace an unavailable action with a
fixed number.

1. In Shortcuts, tap + and name it **GYST — Calories**.
2. Add **Current Date**.
3. Add **Format Date**. Select Custom and enter `yyyy-MM-dd`. Set time format
   to None if shown. Add **Set Variable**, name it `SyncDate`.
4. Add **Find Health Samples** with all filters matching:
   - Type: Dietary Energy (or the dietary calories label in your picker).
   - Start Date: is today.
   - Source: MyFitnessPal.
   Turn Limit off. Use ungrouped samples; do not enable options that fill missing
   periods with zeros. Keep the phone/day timezone aligned with GYST's configured
   timezone (default America/Winnipeg).
5. Add **Set Variable**, name it `Samples`.
6. Add **Count**, selecting Items in `Samples`.
7. Add **If** Count is greater than 0. Put the following actions inside the If.
8. Add **Get Details of Health Samples**, select **Value**, input `Samples`.
9. Add **Calculate Statistics**, select **Sum**, input the returned values.
   Check that calories are in kcal, not kJ, before connecting the request. If
   the value/unit is unclear, inspect the action output first. Do not format the
   sum as text or append units. Macros must be numeric grams.
10. Add **Set Variable**, name it `Total`.
11. Add **Get Contents of URL**:
    - URL: `https://gyst.zaylolabs.com/api/health/sync`
    - Method: POST
    - Headers: `Authorization` = the existing `Bearer …` value from the working
      Shortcut; `Content-Type` = `application/json`.
    - Request Body: JSON. Add the three fields below.

| Key | Type | Value |
|---|---|---|
| date | Text | Select the `SyncDate` variable |
| source | Text | `apple_shortcuts` |
| dietary_calories | Number | Select the `Total` variable |

Select variables from the picker; do not type their names as literal values.
Keep the token private on your device; it is not your GYST login password.

12. During setup, add **Show Result** for the response inside the If block.
13. In Otherwise, add **Show Notification** with “No MyFitnessPal calorie samples
    today; nothing sent.” End If. Do not POST zero when there are no samples.
14. Run unlocked. Grant Shortcuts read permission for this Health category and
    permission to contact the GYST domain. A successful response includes the
    date and `last_synced_at`. Check `dietary_calories` matches Apple Health.
    If the response has an `error`, fix it before proceeding.

[Apple's HTTP request instructions](https://support.apple.com/guide/shortcuts/apd58d46713f/ios).

## 4. Duplicate for the other nutrients

Long-press **GYST — Calories** → Duplicate. For each copy change the name, the
Find Health Samples Type, the numeric JSON key, and the missing-data notification.
Keep Source MyFitnessPal and the same date filter. Re-select the local variable
if the copied action loses a variable reference.

| Shortcut | Health type | Numeric JSON key | Unit |
|---|---|---|---|
| GYST — Protein | Protein | protein_g | g |
| GYST — Carbs | Carbohydrates | carbs_g | g |
| GYST — Fat | Total Fat | fat_g | g |
| GYST — Fiber (optional) | Dietary Fiber | fiber_g | g |

Each request contains only its own nutrient plus date and source. Run and compare
one copy at a time. If Shortcuts returns no samples despite visible Health data,
check its Health read permission and source filter. Never replace missing values
with made-up numbers or a default zero.

## 5. Combine and verify

Create **GYST — Sync Nutrition** with **Run Shortcut** actions for Calories,
Protein, Carbs, Fat and optionally Fiber, in that order. During setup the child
Shortcuts show their responses; after successful validation remove Show Result
so the combined run does not require tapping through five dialogs.

Open `/health/nutrition`. Compare the same date's calories, protein, carbs and
fat with Apple Health and MyFitnessPal. Run it again without changing any food:
the totals must stay the same, because each request replaces a daily total.
Check a genuinely logged recipe serving too. A timestamp alone is not proof
that every nutrient arrived.

If you delete all source samples for a previously synced nutrient, the current
API retains its previous value; omitted/null is not a clearing instruction.
Do not send zero to hide a permissions or missing-data problem.

## 6. Automate after the manual comparison passes

In Shortcuts → Automation → + → App, choose MyFitnessPal and Is Closed. Select
Run Immediately (or disable Ask Before Running, depending on iOS), then run
**GYST — Sync Nutrition**. Test a real food log, leave MyFitnessPal, and compare
GYST afterwards. MyFitnessPal may not have finished writing Health data when
an app-close trigger fires; a manual rerun remains available.

Optional time-of-day triggers can run the same Shortcut. This is not continuous
sync: Apple Health reads may be unavailable while the phone is locked, and
network access is required. Don't show a success notification merely because
the automation started; verify the server response. The public domain, once
working, removes the Tailscale requirement for these requests.

[Apple automation guide](https://support.apple.com/guide/shortcuts/add-automations-apdfbdbd7123/ios),
[Apple Health privacy](https://www.apple.com/legal/privacy/data/en/health-app/).
