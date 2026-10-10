package repository

import "context"

// AuditEvent is one row of audit_events (GYST-VN-005). The service layer
// validates and sanitises it; this type is storage only.
type AuditEvent struct {
	ID        int64
	At        string
	ActorType string
	ActorRef  string
	Ingress   string
	Action    string
	TargetRef string
	Outcome   string
	Detail    string
	RequestID string
}

// InsertAuditEvent stores exactly one row. The SQL CHECKs are the backstop
// behind the service-layer validation.
func (r *Repository) InsertAuditEvent(ctx context.Context, e AuditEvent) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO audit_events
		(at, actor_type, actor_ref, ingress, action, target_ref, outcome, detail, request_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.At, e.ActorType, e.ActorRef, e.Ingress, e.Action, e.TargetRef, e.Outcome, e.Detail, e.RequestID)
	return err
}

// RecentAuditEvents returns up to limit rows, newest first by at then id.
func (r *Repository) RecentAuditEvents(ctx context.Context, limit int) ([]AuditEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, at, actor_type, actor_ref, ingress, action,
		target_ref, outcome, detail, request_id
		FROM audit_events ORDER BY at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEvent{}
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.At, &e.ActorType, &e.ActorRef, &e.Ingress, &e.Action,
			&e.TargetRef, &e.Outcome, &e.Detail, &e.RequestID); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
