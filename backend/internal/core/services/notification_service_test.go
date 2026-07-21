package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

func TestCheckDueNotificationsCreatesUrgentDailyAlertsThreeDaysBeforeDueDate(t *testing.T) {
	repository := &fakeNotificationRepository{
		payables: []ports.NotificationAccountPayableSource{
			{
				ID:          "payable-1",
				UserID:      "user-1",
				Concept:     "Luz",
				AmountCents: 12000,
				DueDate:     notificationTestDate(2026, time.July, 13),
			},
		},
		creditStatements: []ports.NotificationCreditAccountSource{
			{
				ID:             "statement-1",
				UserID:         "user-1",
				Name:           "Joy",
				AmountCents:    30000,
				PaymentDueDate: notificationTestDate(2026, time.July, 13),
			},
		},
	}
	service := NewNotificationService(repository, &mockLogger{})

	if err := service.CheckDueNotifications(context.Background(), notificationTestDate(2026, time.July, 10)); err != nil {
		t.Fatalf("expected first check to succeed: %v", err)
	}
	if err := service.CheckDueNotifications(context.Background(), notificationTestDate(2026, time.July, 11)); err != nil {
		t.Fatalf("expected second check to succeed: %v", err)
	}

	if len(repository.created) != 4 {
		t.Fatalf("expected 4 daily urgent notifications, got %d", len(repository.created))
	}
	for _, notification := range repository.created {
		if !strings.HasPrefix(notification.Title(), "Urgente:") {
			t.Fatalf("expected urgent notification title, got %q", notification.Title())
		}
	}
	if repository.created[0].ID() == repository.created[2].ID() {
		t.Fatalf("expected a distinct notification per alert day")
	}
}

func TestCheckDueNotificationsDoesNotNotifyPaidSources(t *testing.T) {
	repository := &fakeNotificationRepository{}
	service := NewNotificationService(repository, &mockLogger{})

	if err := service.CheckDueNotifications(context.Background(), notificationTestDate(2026, time.July, 10)); err != nil {
		t.Fatalf("expected check to succeed: %v", err)
	}

	if len(repository.created) != 0 {
		t.Fatalf("expected no notifications for paid or absent sources, got %d", len(repository.created))
	}
}

type fakeNotificationRepository struct {
	created          []*domain.Notification
	payables         []ports.NotificationAccountPayableSource
	creditStatements []ports.NotificationCreditAccountSource
}

func (repository *fakeNotificationRepository) CreateIfNotExists(ctx context.Context, notification *domain.Notification) error {
	for _, existing := range repository.created {
		if existing.ID() == notification.ID() {
			return nil
		}
	}
	repository.created = append(repository.created, notification)
	return nil
}

func (repository *fakeNotificationRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Notification, error) {
	return nil, nil
}

func (repository *fakeNotificationRepository) MarkAsRead(ctx context.Context, userID, notificationID string) error {
	return nil
}

func (repository *fakeNotificationRepository) GetDueAccountPayables(ctx context.Context, dueBefore time.Time) ([]ports.NotificationAccountPayableSource, error) {
	return filterNotificationPayables(repository.payables, dueBefore), nil
}

func (repository *fakeNotificationRepository) GetDueCreditCardStatements(ctx context.Context, dueBefore time.Time) ([]ports.NotificationCreditAccountSource, error) {
	return filterNotificationCreditStatements(repository.creditStatements, dueBefore), nil
}

func filterNotificationPayables(payables []ports.NotificationAccountPayableSource, dueBefore time.Time) []ports.NotificationAccountPayableSource {
	filtered := make([]ports.NotificationAccountPayableSource, 0, len(payables))
	for _, payable := range payables {
		if payable.DueDate.Before(dueBefore) {
			filtered = append(filtered, payable)
		}
	}
	return filtered
}

func filterNotificationCreditStatements(statements []ports.NotificationCreditAccountSource, dueBefore time.Time) []ports.NotificationCreditAccountSource {
	filtered := make([]ports.NotificationCreditAccountSource, 0, len(statements))
	for _, statement := range statements {
		if statement.PaymentDueDate.Before(dueBefore) {
			filtered = append(filtered, statement)
		}
	}
	return filtered
}

func notificationTestDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
