package ports

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

type NotificationAccountPayableSource struct {
	ID          string
	UserID      string
	Concept     string
	AmountCents int64
	DueDate     time.Time
}

type NotificationCreditAccountSource struct {
	ID              string
	UserID          string
	Name            string
	AmountCents     int64
	PaymentDueDate  time.Time
	StatementStatus string
}

type NotificationRepository interface {
	CreateIfNotExists(ctx context.Context, notification *domain.Notification) error
	GetByUserID(ctx context.Context, userID string) ([]*domain.Notification, error)
	MarkAsRead(ctx context.Context, userID, notificationID string) error
	GetDueAccountPayables(ctx context.Context, dueBefore time.Time) ([]NotificationAccountPayableSource, error)
	GetDueCreditCardStatements(ctx context.Context, dueBefore time.Time) ([]NotificationCreditAccountSource, error)
}

type NotificationUseCase interface {
	CheckDueNotifications(ctx context.Context, now time.Time) error
	GetNotifications(ctx context.Context, userID string) ([]*domain.Notification, error)
	MarkNotificationAsRead(ctx context.Context, userID, notificationID string) error
}

var (
	ErrNotificationNotFound              = errors.New("repository: notification not found")
	ErrNotificationRepositoryUnavailable = errors.New("repository: notification persistence layer is unavailable or corrupted")
)
