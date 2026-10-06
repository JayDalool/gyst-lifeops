package repository

// Health Coach memory (Issue #64).

// CoachMessage is one stored, visible turn.
type CoachMessage struct {
	ID        int64
	Role      string
	Text      string
	Action    string
	CreatedAt string
}

// ListCoachMessages returns the latest n turns, oldest first.
func (r *Repository) ListCoachMessages(n int) ([]CoachMessage, error) {
	rows, err := r.db.Query(`SELECT id, role, text, action, created_at FROM (
		SELECT id, role, text, action, created_at FROM health_coach_messages ORDER BY id DESC LIMIT ?) ORDER BY id`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CoachMessage{}
	for rows.Next() {
		var m CoachMessage
		if err := rows.Scan(&m.ID, &m.Role, &m.Text, &m.Action, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) AddCoachMessage(role, text, action string) error {
	_, err := r.db.Exec(`INSERT INTO health_coach_messages (role, text, action) VALUES (?, ?, ?)`, role, text, action)
	return err
}

// ClearCoachMessages deletes the whole conversation.
func (r *Repository) ClearCoachMessages() error {
	_, err := r.db.Exec(`DELETE FROM health_coach_messages`)
	return err
}
