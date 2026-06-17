package repository

import "lifeops-tracker/internal/models"

func (r *Repository) ListStandaloneRemindersThrough(date string) ([]models.Reminder, error) {
	rows, err := r.db.Query(`SELECT id, source_type, COALESCE(source_id, 0), title,
		COALESCE(reminder_date, ''), reminder_time, status
		FROM reminders
		WHERE status = 'active'
			AND (source_type != 'task' OR source_id IS NULL)
			AND (reminder_date IS NULL OR reminder_date <= ?)
		ORDER BY reminder_date, reminder_time`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var reminder models.Reminder
		if err := rows.Scan(
			&reminder.ID,
			&reminder.SourceType,
			&reminder.SourceID,
			&reminder.Title,
			&reminder.ReminderDate,
			&reminder.ReminderTime,
			&reminder.Status,
		); err != nil {
			return nil, err
		}
		reminders = append(reminders, reminder)
	}
	return reminders, rows.Err()
}
