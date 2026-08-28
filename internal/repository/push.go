package repository

import "lifeops-tracker/internal/models"

func (r *Repository) UpsertPushSubscription(subscription models.PushSubscription) error {
	_, err := r.db.Exec(`INSERT INTO push_subscriptions (uuid, endpoint, p256dh, auth, user_agent)
		VALUES (lower(hex(randomblob(16))), ?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET p256dh = excluded.p256dh, auth = excluded.auth,
		user_agent = excluded.user_agent, updated_at = CURRENT_TIMESTAMP`,
		subscription.Endpoint, subscription.P256DH, subscription.Auth, subscription.UserAgent)
	return err
}

func (r *Repository) DeletePushSubscription(endpoint string) error {
	_, err := r.db.Exec(`DELETE FROM push_subscriptions WHERE endpoint = ?`, endpoint)
	return err
}

func (r *Repository) PushSubscriptions() ([]models.PushSubscription, error) {
	rows, err := r.db.Query(`SELECT id, endpoint, p256dh, auth, user_agent FROM push_subscriptions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subscriptions []models.PushSubscription
	for rows.Next() {
		var item models.PushSubscription
		if err := rows.Scan(&item.ID, &item.Endpoint, &item.P256DH, &item.Auth, &item.UserAgent); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, item)
	}
	return subscriptions, rows.Err()
}

func (r *Repository) ClaimPushDelivery(subscriptionID int64, date, prayer string) (bool, error) {
	result, err := r.db.Exec(`INSERT OR IGNORE INTO push_deliveries (subscription_id, prayer_date, prayer_name) VALUES (?, ?, ?)`, subscriptionID, date, prayer)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (r *Repository) ReleasePushDelivery(subscriptionID int64, date, prayer string) error {
	_, err := r.db.Exec(`DELETE FROM push_deliveries WHERE subscription_id = ? AND prayer_date = ? AND prayer_name = ?`, subscriptionID, date, prayer)
	return err
}
