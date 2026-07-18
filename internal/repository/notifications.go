package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationRepo persists the email outbox.
type NotificationRepo struct{ pool *pgxpool.Pool }

// NewNotification is the input for creating an outbox row.
type NewNotification struct {
	RecipientUserID *int64
	RecipientEmail  string
	Type            string
	Subject         string
	Body            string
}

// Create inserts a pending notification and returns its id.
func (r *NotificationRepo) Create(ctx context.Context, n *NewNotification) (int64, error) {
	const q = `
		INSERT INTO notifications (recipient_user_id, recipient_email, type, subject, body, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id`
	var id int64
	err := r.pool.QueryRow(ctx, q,
		n.RecipientUserID, n.RecipientEmail, n.Type, n.Subject, n.Body).Scan(&id)
	return id, mapErr(err)
}

// MarkSent records successful delivery.
func (r *NotificationRepo) MarkSent(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET status = 'sent', sent_at = now(), error = NULL WHERE id = $1`, id)
	return mapErr(err)
}

// MarkFailed records a delivery failure and its reason.
func (r *NotificationRepo) MarkFailed(ctx context.Context, id int64, reason string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET status = 'failed', error = $2 WHERE id = $1`, id, reason)
	return mapErr(err)
}

// PendingNotification is an undelivered outbox row awaiting a retry.
type PendingNotification struct {
	ID             int64
	RecipientEmail string
	Type           string
	Subject        string
	Body           string
}

// ListPending returns pending notifications created before olderThan (used by
// the sweeper to catch rows whose enqueue was lost), oldest first.
func (r *NotificationRepo) ListPending(ctx context.Context, olderThan time.Time, limit int) ([]PendingNotification, error) {
	const q = `
		SELECT id, recipient_email, type, subject, body
		FROM notifications
		WHERE status = 'pending' AND created_at < $1
		ORDER BY created_at
		LIMIT $2`
	rows, err := r.pool.Query(ctx, q, olderThan, limit)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []PendingNotification
	for rows.Next() {
		var n PendingNotification
		if err := rows.Scan(&n.ID, &n.RecipientEmail, &n.Type, &n.Subject, &n.Body); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, n)
	}
	return out, mapErr(rows.Err())
}
