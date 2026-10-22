package handlers

import (
	"lifeops-tracker/internal/models"
	"strconv"
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

// ListPending is the checklist's pending rows: every pending occurrence except
// the one the Now block already shows, so that task appears once on Today. The
// checklist-only page has no Now block and lists them all. Counts and progress
// keep reading Pending.
func (d TodayData) ListPending() []models.TaskOccurrence {
	action := d.Now.Action
	if d.ChecklistOnly || action == nil || action.Kind != "task" || action.Task == nil {
		return d.Pending
	}
	out := make([]models.TaskOccurrence, 0, len(d.Pending))
	for _, item := range d.Pending {
		if item.Task.ID == action.Task.Task.ID && item.Date == action.Task.Date {
			continue
		}
		out = append(out, item)
	}
	return out
}

// ListBlocks is today's study blocks without the one shown as the Now action.
func (d TodayData) ListBlocks() []models.LearningBlock {
	action := d.Now.Action
	if d.ChecklistOnly || action == nil || action.Kind != "session" || action.Block == nil {
		return d.Learning.Blocks
	}
	out := make([]models.LearningBlock, 0, len(d.Learning.Blocks))
	for _, block := range d.Learning.Blocks {
		if block.ID != action.Block.ID {
			out = append(out, block)
		}
	}
	return out
}

// NowTimedHref is where the next timed item links: the course for a study
// session, otherwise the reminders page.
func (d TodayData) NowTimedHref() string {
	timed := d.Now.Timed
	if timed != nil && timed.Kind == "session" {
		for _, block := range d.Learning.Blocks {
			if block.ID == timed.SourceID {
				return "/courses/" + strconv.FormatInt(block.CourseID, 10) + "#block-" + strconv.FormatInt(block.ID, 10)
			}
		}
	}
	return "/reminders"
}

func (d TodayData) ChecklistReturnPath() string {
	if d.ChecklistOnly {
		return "/checklist"
	}
	return "/today"
}

// standaloneReminders drops task-sourced reminders: the task row already
// carries its reminder time, so listing it again would duplicate it. A task-typed
// row without a source id (the column default) is standalone and stays.
func standaloneReminders(rows []models.ReminderView) []models.ReminderView {
	out := make([]models.ReminderView, 0, len(rows))
	for _, r := range rows {
		if r.SourceType != "task" || r.SourceID == 0 {
			out = append(out, r)
		}
	}
	return out
}
