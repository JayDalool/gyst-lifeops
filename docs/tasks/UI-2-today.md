# UI-2 — Today and daily checklist

- **Status:** `IN PROGRESS` — implementation complete; final verification/push pending.
- **Approval:** Jay approved UI-1 and requested the next task.
- **Implementation commit(s):** Pending.
- **CI:** Pending push.
- **Review:** Agent browser verification; physical iPhone/Home Screen review pending.

## Changes

Today adopts the approved light card system, greeting/date, focus banner,
checklist progress, Health snapshot, real pending focus item and next timed
reminder. Today previews four pending, two completed and two skipped rows; the dedicated checklist retains every task. Retains completion/skip actions,
study completion/postponement, prayer/Athan controls, Athkar, deadlines and XP.
The dedicated `/checklist` route reuses the same task partial and preserves its
return destination after actions. More links to it; Today remains the selected tab.

Health uses one existing HealthToday service read per Today request and the
existing Health formatter/calculations. No extra Health read on `/checklist`.
No snapshot is an empty state; failed reads are an error card. Missing metrics,
unknown sync time and stale snapshots are explicitly distinguished. Observed
zero remains numeric. Watch steps take precedence; phone fallback is labelled.
Food remaining is labelled as food target minus consumed, never energy balance.
Snapshot date is explicit because Health uses the configured timezone.

Focus is the first existing pending task, not an invented scheduler. Next up
selects the earliest valid timed reminder later on the current local date;
past/untimed reminders remain in the full existing list. No fake events, new
stored categories, business calculations, migrations or integration changes.

## Verification

- Go format/vet/build/unit/race checks passed. Chromium and WebKit passed Today/checklist at 320, 375, 390, 430, 768 and 1280px, with 200% text, 44px checkbox labels, correct navigation ownership, dark mode and no JavaScript page errors. Browser actions confirmed task creation, checklist completion/return, Today skip/recovery and saved tracker state. Final CSS previews were rechecked after visual review.
- New tests cover absent/zero/partial/stale/error Health presentation, Watch
  precedence, explicit phone fallback, reminder ordering/date boundaries,
  retained task/study actions, checklist authentication and method/path denial.
- UI shell cache/theme Node checks and JavaScript syntax checks pass. Worker
  cache version and precached CSS/chart versions updated together.
- Browser preview uses only `/tmp/gyst-ui2-review/preview.db` and fixture
  credentials/tasks. No production database access or deployment.
- During verification, fixed test literals for embedded HealthMetrics and
  browser action tests now wait for navigation to settle; scoped a phone-value assertion to rendered metrics (SVG paths contain digits).
  Visual review corrected cramped prayer labels and repeated empty Health copy.

## Next action

Jay: review UI-2 before approving UI-3 Projects and project detail.
