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
	accountRepository     ports.FinancialAccountRepository
	idGenerator           ports.IDGenerator
	logger                ports.Logger
}

func NewCreditCardService(
	creditCardRepository ports.CreditCardRepository,
	transactionRepository ports.TransactionRepository,
	idGenerator ports.IDGenerator,
	logger ports.Logger,
	accountRepository ...ports.FinancialAccountRepository,
) ports.CreditCardUseCase {
	var financialAccountRepository ports.FinancialAccountRepository
	if len(accountRepository) > 0 {
		financialAccountRepository = accountRepository[0]
	}
	return &creditCardService{
		creditCardRepository:  creditCardRepository,
		transactionRepository: transactionRepository,
		accountRepository:     financialAccountRepository,
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

func (service *creditCardService) PayCreditCardDebt(ctx context.Context, userID, creditCardID, sourceAccountID string, amountCents int64) error {
	if amountCents <= 0 {
		return domain.ErrInvalidTransactionAmount
	}
	creditCard, err := service.getAuthorizedCreditCard(ctx, userID, creditCardID)
	if err != nil {
		return err
	}
	if service.accountRepository == nil {
		service.logger.Error("credit card payment requested without financial account repository", "userID", userID, "creditCardID", creditCardID)
		return ErrInternalProcessing
	}

	sourceAccount, err := service.accountRepository.GetByID(ctx, sourceAccountID)
	if errors.Is(err, ports.ErrFinancialAccountNotFound) {
		return ports.ErrFinancialAccountNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve payment source account", "userID", userID, "accountID", sourceAccountID, "error", err)
		return ErrInternalProcessing
	}
	if sourceAccount.UserID() != userID || sourceAccount.Type() == domain.FinancialAccountTypeCreditCard {
		return ports.ErrFinancialAccountNotFound
	}

	creditAccount, err := service.accountRepository.GetByID(ctx, creditCard.ID())
	if errors.Is(err, ports.ErrFinancialAccountNotFound) {
		return ports.ErrCreditCardNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve credit financial account", "userID", userID, "creditCardID", creditCard.ID(), "error", err)
		return ErrInternalProcessing
	}
	if creditAccount.UserID() != userID || creditAccount.Type() != domain.FinancialAccountTypeCreditCard {
		return ports.ErrCreditCardNotFound
	}
	if amountCents > creditAccount.CurrentBalanceCents() {
		return domain.ErrInvalidTransactionAmount
	}
	if err := sourceAccount.ApplyDelta(-amountCents); err != nil {
		return err
	}

	now := time.Now()
	payment, err := domain.NewTransaction(
		service.idGenerator.Generate(),
		userID,
		domain.TransactionTypeDebtPayment,
		"Pago "+creditCard.Name(),
		"Tarjeta de credito",
		amountCents,
		now,
		domain.TransactionStatusCompleted,
		nil,
		&creditCardID,
		domain.TransactionRecurrenceOnce,
		nil,
	)
	if err != nil {
		return err
	}
	payment.SetAccountingDetails(&sourceAccountID, &creditCardID, nil, nil)

	deltas := []ports.AccountBalanceDelta{
		{AccountID: sourceAccountID, DeltaCents: -amountCents},
		{AccountID: creditCardID, DeltaCents: -amountCents},
	}
	ledgerEntries := []ports.LedgerEntry{
		{
			ID:            service.idGenerator.Generate(),
			UserID:        userID,
			AccountID:     sourceAccountID,
			TransactionID: payment.ID(),
			AmountCents:   -amountCents,
			EntryType:     string(domain.TransactionTypeDebtPayment),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            service.idGenerator.Generate(),
			UserID:        userID,
			AccountID:     creditCardID,
			TransactionID: payment.ID(),
			AmountCents:   -amountCents,
			EntryType:     string(domain.TransactionTypeDebtPayment),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	if err := service.transactionRepository.CreateManyWithLedger(ctx, []*domain.Transaction{payment}, deltas, ledgerEntries); err != nil {
		service.logger.Error("failed to pay credit card debt", "userID", userID, "creditCardID", creditCardID, "sourceAccountID", sourceAccountID, "error", err)
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

		currentDebtCents += transaction.AmountCents()
	}

	return currentDebtCents
}

func currentBillingCycle(cutoffDay int, now time.Time) (time.Time, time.Time) {
	year, month, _ := now.Date()
	location := now.Location()
	currentMonthCutoff := billingCycleDate(year, month, cutoffDay, location)
	currentDay := time.Date(year, month, now.Day(), 0, 0, 0, 0, location)
	if currentDay.After(currentMonthCutoff) {
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
