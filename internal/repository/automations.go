package repository

import "strings"

// AutomationEvent is one row of automation_events (GYST-VN-110). The service
// layer validates and normalises it; this type is storage only.
type AutomationEvent struct {
	EventID      string
	WorkflowID   string
	Status       string
	OccurredAt   string
	ReceivedAt   string
	ErrorSummary string
}

// InsertAutomationEvent stores the event unless its event_id already exists.
// inserted is false for a duplicate, and nothing changes in that case.
func (r *Repository) InsertAutomationEvent(e AutomationEvent) (inserted bool, err error) {
	result, err := r.db.Exec(`INSERT OR IGNORE INTO automation_events
		(event_id, workflow_id, status, occurred_at, received_at, error_summary)
		VALUES (?, ?, ?, ?, ?, ?)`,
		e.EventID, e.WorkflowID, e.Status, e.OccurredAt, e.ReceivedAt, e.ErrorSummary)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// PruneAutomationEventsBefore deletes events that occurred before cutoff
// (UTC text in the table's format) and returns how many.
func (r *Repository) PruneAutomationEventsBefore(cutoff string) (int64, error) {
	result, err := r.db.Exec(`DELETE FROM automation_events WHERE occurred_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// AutomationEventsForWorkflows returns the stored events of the given
// workflows, ordered by workflow then newest occurred_at first.
func (r *Repository) AutomationEventsForWorkflows(ids []string) ([]AutomationEvent, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := r.db.Query(`SELECT event_id, workflow_id, status, occurred_at, received_at, error_summary
		FROM automation_events
		WHERE workflow_id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")+`)
		ORDER BY workflow_id, occurred_at DESC, received_at DESC, event_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AutomationEvent
	for rows.Next() {
		var e AutomationEvent
		if err := rows.Scan(&e.EventID, &e.WorkflowID, &e.Status, &e.OccurredAt, &e.ReceivedAt, &e.ErrorSummary); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
