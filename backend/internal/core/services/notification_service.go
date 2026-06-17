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
	dueBefore := today.AddDate(0, 0, 2)
	payables, err := service.repository.GetDueAccountPayables(ctx, dueBefore)
	if err != nil {
		return err
	}
	for _, payable := range payables {
		notification, err := domain.NewNotification(
			accountPayableNotificationID(payable),
			payable.UserID,
			"Cuenta por pagar próxima",
			fmt.Sprintf("%s vence el %s.", payable.Concept, payable.DueDate.Format("02/01/2006")),
		)
		if err != nil {
			service.logger.Warn("skipping invalid account payable notification", "payableID", payable.ID, "error", err)
			continue
		}
		if err := service.repository.CreateIfNotExists(ctx, notification); err != nil {
			return err
		}
	}

	creditAccounts, err := service.repository.GetCreditAccountsWithDebt(ctx)
	if err != nil {
		return err
	}
	for _, account := range creditAccounts {
		dueDate := creditCardPaymentDueDate(account.CutoffDay, account.PaymentDay, now)
		if !dueDate.Before(dueBefore) {
			continue
		}
		notification, err := domain.NewNotification(
			creditCardNotificationID(account, dueDate),
			account.UserID,
			"Pago de tarjeta próximo",
			fmt.Sprintf("%s vence el %s.", account.Name, dueDate.Format("02/01/2006")),
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

func accountPayableNotificationID(payable ports.NotificationAccountPayableSource) string {
	return fmt.Sprintf("payable:%s:%s:%s", payable.UserID, payable.ID, payable.DueDate.Format("20060102"))
}

func creditCardNotificationID(account ports.NotificationCreditAccountSource, dueDate time.Time) string {
	return fmt.Sprintf("credit-card:%s:%s:%s", account.UserID, account.ID, dueDate.Format("20060102"))
}

func creditCardPaymentDueDate(cutoffDay, paymentDay int, now time.Time) time.Time {
	currentCutoff := notificationCycleDate(now.Year(), now.Month(), cutoffDay, now.Location())
	offset := 1
	if startOfDay(now).After(currentCutoff) {
		offset = 2
	}
	return notificationCycleDate(now.Year(), now.Month()+time.Month(offset), paymentDay, now.Location())
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
