package services

import (
	"context"
	"fmt"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type notificationService struct {
	repository ports.NotificationRepository
	logger     ports.Logger
}

func NewNotificationService(repository ports.NotificationRepository, logger ports.Logger) ports.NotificationUseCase {
	return &notificationService{repository: repository, logger: logger}
}

func (service *notificationService) CheckDueNotifications(ctx context.Context, now time.Time) error {
	if service == nil || service.repository == nil {
		return ports.ErrNotificationRepositoryUnavailable
	}

	today := startOfDay(now)
	alertDate := today
	dueBefore := today.AddDate(0, 0, 4)
	payables, err := service.repository.GetDueAccountPayables(ctx, dueBefore)
	if err != nil {
		return err
	}
	for _, payable := range payables {
		notification, err := domain.NewNotification(
			accountPayableNotificationID(payable, alertDate),
			payable.UserID,
			"Urgente: cuenta por pagar",
			urgentPayableMessage(payable.Concept, payable.DueDate, today),
		)
		if err != nil {
			service.logger.Warn("skipping invalid account payable notification", "payableID", payable.ID, "error", err)
			continue
		}
		if err := service.repository.CreateIfNotExists(ctx, notification); err != nil {
			return err
		}
	}

	creditAccounts, err := service.repository.GetDueCreditCardStatements(ctx, dueBefore)
	if err != nil {
		return err
	}
	for _, account := range creditAccounts {
		notification, err := domain.NewNotification(
			creditCardNotificationID(account, alertDate),
			account.UserID,
			"Urgente: pago de tarjeta",
			urgentPayableMessage(account.Name, account.PaymentDueDate, today),
		)
		if err != nil {
			service.logger.Warn("skipping invalid credit card notification", "accountID", account.ID, "error", err)
			continue
		}
		if err := service.repository.CreateIfNotExists(ctx, notification); err != nil {
			return err
		}
	}

	return nil
}

func (service *notificationService) GetNotifications(ctx context.Context, userID string) ([]*domain.Notification, error) {
	return service.repository.GetByUserID(ctx, userID)
}

func (service *notificationService) MarkNotificationAsRead(ctx context.Context, userID, notificationID string) error {
	return service.repository.MarkAsRead(ctx, userID, notificationID)
}

func accountPayableNotificationID(payable ports.NotificationAccountPayableSource, alertDate time.Time) string {
	return fmt.Sprintf("payable:%s:%s:%s:%s", payable.UserID, payable.ID, payable.DueDate.Format("20060102"), alertDate.Format("20060102"))
}

func creditCardNotificationID(account ports.NotificationCreditAccountSource, alertDate time.Time) string {
	return fmt.Sprintf("credit-card:%s:%s:%s:%s", account.UserID, account.ID, account.PaymentDueDate.Format("20060102"), alertDate.Format("20060102"))
}

func urgentPayableMessage(name string, dueDate, today time.Time) string {
	daysUntilDue := int(startOfDay(dueDate).Sub(today).Hours() / 24)
	switch {
	case daysUntilDue < 0:
		return fmt.Sprintf("%s venció el %s. Regulariza este pago cuanto antes.", name, dueDate.Format("02/01/2006"))
	case daysUntilDue == 0:
		return fmt.Sprintf("%s vence hoy. Realiza el pago para mantener tus cuentas al corriente.", name)
	case daysUntilDue == 1:
		return fmt.Sprintf("%s vence mañana (%s).", name, dueDate.Format("02/01/2006"))
	default:
		return fmt.Sprintf("%s vence en %d días (%s).", name, daysUntilDue, dueDate.Format("02/01/2006"))
	}
}

func notificationCycleDate(year int, month time.Month, day int, location *time.Location) time.Time {
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, location)
	lastDay := firstOfMonth.AddDate(0, 1, -1).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(firstOfMonth.Year(), firstOfMonth.Month(), day, 0, 0, 0, 0, location)
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
