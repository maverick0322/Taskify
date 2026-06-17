package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type SQLiteNotificationRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteNotificationRepository(database *sql.DB, logger ports.Logger) ports.NotificationRepository {
	return &SQLiteNotificationRepository{database: database, logger: logger}
}

func (repository *SQLiteNotificationRepository) CreateIfNotExists(ctx context.Context, notification *domain.Notification) error {
	if notification == nil {
		return ports.ErrNotificationRepositoryUnavailable
	}
	_, err := repository.database.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO notifications (id, user_id, title, message, is_read, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		notification.ID(),
		notification.UserID(),
		notification.Title(),
		notification.Message(),
		notification.IsRead(),
		timeValue(notification.CreatedAt()),
		timeValue(notification.UpdatedAt()),
	)
	if err != nil {
		repository.logger.Error("failed to create notification", "error", err)
		return ports.ErrNotificationRepositoryUnavailable
	}
	return nil
}

func (repository *SQLiteNotificationRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Notification, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT id, user_id, title, message, is_read, created_at, updated_at
		 FROM notifications
		 WHERE user_id = ? AND deleted_at IS NULL
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, ports.ErrNotificationRepositoryUnavailable
	}
	defer rows.Close()
	return scanNotifications(rows)
}

func (repository *SQLiteNotificationRepository) MarkAsRead(ctx context.Context, userID, notificationID string) error {
	result, err := repository.database.ExecContext(
		ctx,
		`UPDATE notifications
		 SET is_read = 1, updated_at = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		timeValue(time.Now()),
		notificationID,
		userID,
	)
	if err != nil {
		return ports.ErrNotificationRepositoryUnavailable
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ports.ErrNotificationRepositoryUnavailable
	}
	if rowsAffected == 0 {
		return ports.ErrNotificationNotFound
	}
	return nil
}

func (repository *SQLiteNotificationRepository) GetDueAccountPayables(ctx context.Context, dueBefore time.Time) ([]ports.NotificationAccountPayableSource, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT id, user_id, concept, amount_cents, date
		 FROM transactions
		 WHERE type = 'EXPENSE'
		   AND status = 'PENDING'
		   AND date < ?
		   AND deleted_at IS NULL
		   AND NOT EXISTS (
		     SELECT 1 FROM account_payable_payments
		     WHERE account_payable_id = transactions.id
		       AND due_date = transactions.date
		   )
		 ORDER BY date ASC`,
		timeValue(dueBefore),
	)
	if err != nil {
		return nil, ports.ErrNotificationRepositoryUnavailable
	}
	defer rows.Close()

	sources := make([]ports.NotificationAccountPayableSource, 0)
	for rows.Next() {
		var source ports.NotificationAccountPayableSource
		if err := rows.Scan(&source.ID, &source.UserID, &source.Concept, &source.AmountCents, &source.DueDate); err != nil {
			return nil, ports.ErrNotificationRepositoryUnavailable
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, ports.ErrNotificationRepositoryUnavailable
	}
	return sources, nil
}

func (repository *SQLiteNotificationRepository) GetCreditAccountsWithDebt(ctx context.Context) ([]ports.NotificationCreditAccountSource, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT id, user_id, name, current_balance_cents, cutoff_day, payment_day
		 FROM financial_accounts
		 WHERE type = 'CREDIT_CARD'
		   AND current_balance_cents > 0
		   AND cutoff_day IS NOT NULL
		   AND payment_day IS NOT NULL
		   AND deleted_at IS NULL`,
	)
	if err != nil {
		return nil, ports.ErrNotificationRepositoryUnavailable
	}
	defer rows.Close()

	sources := make([]ports.NotificationCreditAccountSource, 0)
	for rows.Next() {
		var source ports.NotificationCreditAccountSource
		if err := rows.Scan(&source.ID, &source.UserID, &source.Name, &source.CurrentBalanceCents, &source.CutoffDay, &source.PaymentDay); err != nil {
			return nil, ports.ErrNotificationRepositoryUnavailable
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, ports.ErrNotificationRepositoryUnavailable
	}
	return sources, nil
}

func scanNotifications(rows *sql.Rows) ([]*domain.Notification, error) {
	notifications := make([]*domain.Notification, 0)
	for rows.Next() {
		var stored struct {
			id, userID, title, message string
			isRead                     bool
			createdAt, updatedAt       time.Time
		}
		if err := rows.Scan(&stored.id, &stored.userID, &stored.title, &stored.message, &stored.isRead, &stored.createdAt, &stored.updatedAt); err != nil {
			return nil, err
		}
		notification, err := domain.RehydrateNotification(stored.id, stored.userID, stored.title, stored.message, stored.isRead, stored.createdAt, stored.updatedAt)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return notifications, nil
}
