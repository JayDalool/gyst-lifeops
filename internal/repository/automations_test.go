package repository

import (
	"strings"
	"testing"
)

// GYST-131 security blocker: the retention prune must use an index, or every
// stored event scans the whole table.
func TestPruneAutomationEventsUsesIndex(t *testing.T) {
	r := newTestRepo(t)
	rows, err := r.db.Query("EXPLAIN QUERY PLAN "+pruneAutomationEventsSQL, "2026-01-01T00:00:00.000000000Z")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan []string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		plan = append(plan, detail)
	}
	joined := strings.Join(plan, "; ")
	if !strings.Contains(joined, "SEARCH") || strings.Contains(joined, "SCAN") || !strings.Contains(joined, "idx_automation_events_occurred_at") {
		t.Fatalf("prune plan is not an index search: %s", joined)
	}
}

// OR IGNORE would hide CHECK violations; ON CONFLICT DO NOTHING must not.
func TestInsertAutomationEventSurfacesCheckViolations(t *testing.T) {
	r := newTestRepo(t)
	ok := AutomationEvent{EventID: "e1", WorkflowID: "wf", Status: "started",
		OccurredAt: "2026-10-11T12:00:00.000000000Z", ReceivedAt: "2026-10-11T12:00:00.000000000Z"}
	if inserted, err := r.InsertAutomationEvent(ok); err != nil || !inserted {
		t.Fatalf("valid: %v %v", inserted, err)
	}
	if inserted, err := r.InsertAutomationEvent(ok); err != nil || inserted {
		t.Fatalf("duplicate: %v %v", inserted, err)
	}
	bad := ok
	bad.EventID, bad.Status = "e2", "bogus"
	if _, err := r.InsertAutomationEvent(bad); err == nil {
		t.Fatal("CHECK violation was swallowed")
	}
}
