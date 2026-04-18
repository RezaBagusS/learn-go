package repository

import (
	"context"
	"database/sql"
	"fmt"
	"my-microservices/notification-service/internal/domain"
	"time"
)

type NotificationRepository interface {
	CreateLog(ctx context.Context, data domain.NotificationLog) error
	UpdateLogStatus(ctx context.Context, id string, status domain.NotificationStatus, retryCount int, lastError string, httpCode int, sentAt *time.Time) error
}

type notificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) NotificationRepository {
	return &notificationRepository{
		db: db,
	}
}

func (r *notificationRepository) CreateLog(ctx context.Context, data domain.NotificationLog) error {
	query := `
		INSERT INTO notification_logs (
			id, event_type, reference_id, partner_id, callback_url,
			payload, status, retry_count, last_error, http_status_code,
			created_at, updated_at, sent_at
		) VALUES (
			$1, $2, $3, $4, $5, 
			$6, $7, $8, $9, $10, 
			$11, $12, $13
		)
	`
	_, err := r.db.ExecContext(
		ctx, query,
		data.ID, data.EventType, data.ReferenceID, data.PartnerID, data.CallbackURL,
		data.Payload, data.Status, data.RetryCount, data.LastError, data.HttpStatusCode,
		data.CreatedAt, data.UpdatedAt, data.SentAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert notification log: %w", err)
	}
	return nil
}

func (r *notificationRepository) UpdateLogStatus(ctx context.Context, id string, status domain.NotificationStatus, retryCount int, lastError string, httpCode int, sentAt *time.Time) error {
	query := `
		UPDATE notification_logs 
		SET status = $1, retry_count = $2, last_error = $3, http_status_code = $4, 
		    sent_at = $5, updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query, status, retryCount, lastError, httpCode, sentAt, id)
	if err != nil {
		return fmt.Errorf("failed to update notification log status: %w", err)
	}
	return nil
}
