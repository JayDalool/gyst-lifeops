package repository

import (
	"context"
	"errors"
	"strings"
)

// DeleteAuditEventsBefore removes rows with at strictly before cutoff (a row
// exactly at the cutoff stays). actions narrows the delete: include=true
// deletes only those actions, include=false deletes every action except
// those. include=true with no actions deletes nothing. cutoff must be a
// stored-instant string (clock.FormatInstant); an empty one is refused so the
// statement can never match every row. Action names are bound parameters.
func (r *Repository) DeleteAuditEventsBefore(ctx context.Context, cutoff string, actions []string, include bool) (int64, error) {
	if cutoff == "" {
		return 0, errors.New("audit delete: empty cutoff")
	}
	query := "DELETE FROM audit_events WHERE at < ?"
	args := []any{cutoff}
	if len(actions) > 0 {
		marks := strings.TrimSuffix(strings.Repeat("?,", len(actions)), ",")
		if include {
			query += " AND action IN (" + marks + ")"
		} else {
			query += " AND action NOT IN (" + marks + ")"
		}
		for _, a := range actions {
			args = append(args, a)
		}
	} else if include {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

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
