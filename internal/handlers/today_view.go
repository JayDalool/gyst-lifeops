package handlers

import (
	"lifeops-tracker/internal/models"
	"time"
)

// TodayHealthSnapshot reuses the Health presentation layer. It never computes
// energy or merges step sources, and a Health read failure leaves Today usable.
type TodayHealthSnapshot struct {
	Date    string
	Status  UIStatus
	Metrics []UIMetric
}

func buildTodayHealthSnapshot(view HealthTodayView, err error) TodayHealthSnapshot {
	out := TodayHealthSnapshot{Date: view.Date}
	if err != nil {
		out.Status = UIStatus{State: "error", Message: "Health snapshot unavailable. Try opening Health again."}
		return out
	}
	out.Status = UIStatus{State: "normal", Message: "Snapshot synced " + view.LastSynced}
	switch {
	case !view.HasSnapshot:
		out.Status = UIStatus{State: "empty", Message: "No Health data synced for this day yet."}
		return out
	case view.LastSyncedStale:
		out.Status = UIStatus{State: "stale", Message: "Snapshot synced " + view.LastSynced}
	case !view.HasLastSynced:
		out.Status = UIStatus{State: "partial", Message: "Snapshot sync time unavailable."}
	}
	steps, stepsLabel := view.StepsWatch, "Watch steps"
	if !steps.Present && view.StepsPhone.Present {
		steps, stepsLabel = view.StepsPhone, "Phone steps (fallback)"
	}
	for _, item := range []struct {
		label, detail string
		metric        HealthMetricView
	}{
		{"Food remaining", "Food target minus consumed", view.FoodRemaining},
		{stepsLabel, "Activity measurement", steps},
	} {
		status := out.Status
		if !item.metric.Present {
			status = UIStatus{State: "empty", Message: "Not available yet"}
			if out.Status.Kind() == "normal" {
				out.Status = UIStatus{State: "partial", Message: "Some values missing · snapshot synced " + view.LastSynced}
			}
		}
		out.Metrics = append(out.Metrics, UIMetric{Label: item.label, Detail: item.detail, Value: item.metric.Value, Unit: item.metric.Unit, Present: item.metric.Present, Status: status})
	}
	return out
}

// Only existing, timed reminders later today qualify. Past and all-day rows
// remain visible in the full reminders/deadlines list.
func nextTodayReminder(reminders []models.ReminderView, now time.Time) *models.ReminderView {
	var next *models.ReminderView
	var nextAt time.Time
	for _, reminder := range reminders {
		at, err := time.ParseInLocation("2006-01-02 15:04", reminder.ReminderDate+" "+reminder.ReminderTime, now.Location())
		if err != nil || reminder.ReminderDate != now.Format("2006-01-02") || !at.After(now) {
			continue
		}
		if next == nil || at.Before(nextAt) {
			copy := reminder
			next = &copy
			nextAt = at
		}
	}
	return next
}

func (d TodayData) ChecklistReturnPath() string {
	if d.ChecklistOnly {
		return "/checklist"
	}
	return "/today"
}

// standaloneReminders drops task-sourced reminders: the task row already
// carries its reminder time, so listing it again would duplicate it.
func standaloneReminders(rows []models.ReminderView) []models.ReminderView {
	out := make([]models.ReminderView, 0, len(rows))
	for _, r := range rows {
		if r.SourceType != "task" {
			out = append(out, r)
		}
	}
	return out
}
