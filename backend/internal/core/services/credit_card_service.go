package services

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

var creditCardCurrentTime = time.Now

type creditCardService struct {
	creditCardRepository  ports.CreditCardRepository
	transactionRepository ports.TransactionRepository
	idGenerator           ports.IDGenerator
	logger                ports.Logger
}

func NewCreditCardService(
	creditCardRepository ports.CreditCardRepository,
	transactionRepository ports.TransactionRepository,
	idGenerator ports.IDGenerator,
	logger ports.Logger,
) ports.CreditCardUseCase {
	return &creditCardService{
		creditCardRepository:  creditCardRepository,
		transactionRepository: transactionRepository,
		idGenerator:           idGenerator,
		logger:                logger,
	}
}

func (service *creditCardService) CreateCreditCard(ctx context.Context, userID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) (*domain.CreditCard, error) {
	creditCardID := service.idGenerator.Generate()
	creditCard, err := domain.NewCreditCard(creditCardID, userID, name, bank, last4, cutoffDay, paymentDay, limitCents, color)
	if err != nil {
		return nil, err
	}

	if err := service.creditCardRepository.Create(ctx, creditCard); err != nil {
		service.logger.Error("failed to create credit card", "userID", userID, "creditCardID", creditCardID, "error", err)
		return nil, ErrInternalProcessing
	}

	return creditCard, nil
}

func (service *creditCardService) GetCardsWithSummary(ctx context.Context, userID string) ([]ports.CreditCardWithSummary, error) {
	creditCards, err := service.creditCardRepository.GetByUserID(ctx, userID)
	if err != nil {
		service.logger.Error("failed to retrieve credit cards", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	summaries := make([]ports.CreditCardWithSummary, 0, len(creditCards))
	for _, creditCard := range creditCards {
		if creditCard == nil {
			continue
		}

		startDate, endDate := currentBillingCycle(creditCard.CutoffDay(), creditCardCurrentTime())
		filter := ports.TransactionDateFilter{From: &startDate, To: &endDate}
		transactions, err := service.transactionRepository.GetByCreditCardID(ctx, userID, creditCard.ID(), filter)
		if err != nil {
			service.logger.Error("failed to retrieve credit card transactions", "userID", userID, "creditCardID", creditCard.ID(), "error", err)
			return nil, ErrInternalProcessing
		}

		summaries = append(summaries, ports.CreditCardWithSummary{
			CreditCard:       creditCard,
			CurrentDebtCents: calculateCreditCardDebt(transactions),
		})
	}

	return summaries, nil
}

func (service *creditCardService) UpdateCreditCard(ctx context.Context, userID, creditCardID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) error {
	creditCard, err := service.getAuthorizedCreditCard(ctx, userID, creditCardID)
	if err != nil {
		return err
	}

	if err := creditCard.Update(name, bank, last4, cutoffDay, paymentDay, limitCents, color); err != nil {
		return err
	}

	if err := service.creditCardRepository.Update(ctx, creditCard); err != nil {
		service.logger.Error("failed to update credit card", "userID", userID, "creditCardID", creditCard.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *creditCardService) DeleteCreditCard(ctx context.Context, userID, creditCardID string) error {
	creditCard, err := service.getAuthorizedCreditCard(ctx, userID, creditCardID)
	if err != nil {
		return err
	}

	if err := service.creditCardRepository.Delete(ctx, creditCard.ID()); err != nil {
		service.logger.Error("failed to delete credit card", "userID", userID, "creditCardID", creditCard.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *creditCardService) getAuthorizedCreditCard(ctx context.Context, userID, creditCardID string) (*domain.CreditCard, error) {
	creditCard, err := service.creditCardRepository.GetByID(ctx, creditCardID)
	if errors.Is(err, ports.ErrCreditCardNotFound) {
		return nil, ports.ErrCreditCardNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve credit card", "userID", userID, "creditCardID", creditCardID, "error", err)
		return nil, ErrInternalProcessing
	}
	if creditCard == nil {
		return nil, ports.ErrCreditCardNotFound
	}
	if creditCard.UserID() != userID {
		service.logger.Warn("unauthorized credit card access attempt", "userID", userID, "creditCardID", creditCardID)
		return nil, ports.ErrCreditCardNotFound
	}

	return creditCard, nil
}

func calculateCreditCardDebt(transactions []*domain.Transaction) int64 {
	var currentDebtCents int64
	for _, transaction := range transactions {
		if transaction == nil ||
			transaction.Status() != domain.TransactionStatusPaid ||
			transaction.Type() != domain.TransactionTypeExpense {
			continue
		}

		currentDebtCents += creditCardTransactionInstallment(transaction.AmountCents(), transaction.MSI())
	}

	return currentDebtCents
}

func creditCardTransactionInstallment(amountCents int64, msi *int) int64 {
	if msi == nil || *msi <= 1 {
		return amountCents
	}

	months := int64(*msi)
	return amountCents/months + amountCents%months
}

func currentBillingCycle(cutoffDay int, now time.Time) (time.Time, time.Time) {
	year, month, _ := now.Date()
	location := now.Location()
	currentMonthCutoff := billingCycleDate(year, month, cutoffDay, location)
	if !now.Before(currentMonthCutoff) {
		return currentMonthCutoff, billingCycleDate(year, month+1, cutoffDay, location)
	}

	return billingCycleDate(year, month-1, cutoffDay, location), currentMonthCutoff
}

func billingCycleDate(year int, month time.Month, cutoffDay int, location *time.Location) time.Time {
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, location)
	lastDay := firstOfMonth.AddDate(0, 1, -1).Day()
	if cutoffDay > lastDay {
		cutoffDay = lastDay
	}

	return time.Date(firstOfMonth.Year(), firstOfMonth.Month(), cutoffDay, 0, 0, 0, 0, location)
}
