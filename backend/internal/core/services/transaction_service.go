package services

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type transactionService struct {
	transactionRepository ports.TransactionRepository
	idGenerator           ports.IDGenerator
	logger                ports.Logger
}

func NewTransactionService(
	transactionRepository ports.TransactionRepository,
	idGenerator ports.IDGenerator,
	logger ports.Logger,
) ports.TransactionUseCase {
	return &transactionService{
		transactionRepository: transactionRepository,
		idGenerator:           idGenerator,
		logger:                logger,
	}
}

func (service *transactionService) CreateTransaction(
	ctx context.Context,
	userID string,
	transactionType domain.TransactionType,
	concept,
	category string,
	amountCents int64,
	date time.Time,
	status domain.TransactionStatus,
	msi *int,
	creditCardID *string,
) (*domain.Transaction, error) {
	transactionID := service.idGenerator.Generate()
	transaction, err := domain.NewTransaction(transactionID, userID, transactionType, concept, category, amountCents, date, status, msi, creditCardID)
	if err != nil {
		return nil, err
	}

	if err := service.transactionRepository.Create(ctx, transaction); err != nil {
		service.logger.Error("failed to create transaction", "userID", userID, "transactionID", transactionID, "error", err)
		return nil, ErrInternalProcessing
	}

	return transaction, nil
}

func (service *transactionService) GetTransaction(ctx context.Context, userID, transactionID string) (*domain.Transaction, error) {
	return service.getAuthorizedTransaction(ctx, userID, transactionID)
}

func (service *transactionService) GetUserTransactions(ctx context.Context, userID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	transactions, err := service.transactionRepository.GetByUserID(ctx, userID, filter)
	if err != nil {
		service.logger.Error("failed to retrieve user transactions", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	return transactions, nil
}

func (service *transactionService) UpdateTransaction(
	ctx context.Context,
	userID,
	transactionID string,
	transactionType domain.TransactionType,
	concept,
	category string,
	amountCents int64,
	date time.Time,
	status domain.TransactionStatus,
	msi *int,
	creditCardID *string,
) error {
	transaction, err := service.getAuthorizedTransaction(ctx, userID, transactionID)
	if err != nil {
		return err
	}

	if err := transaction.Update(transactionType, concept, category, amountCents, date, status, msi, creditCardID); err != nil {
		return err
	}

	return service.persistTransactionUpdate(ctx, transaction)
}

func (service *transactionService) DeleteTransaction(ctx context.Context, userID, transactionID string) error {
	transaction, err := service.getAuthorizedTransaction(ctx, userID, transactionID)
	if err != nil {
		return err
	}

	if err := service.transactionRepository.Delete(ctx, transaction.ID()); err != nil {
		service.logger.Error("failed to delete transaction", "userID", userID, "transactionID", transaction.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *transactionService) GetFinancialSummary(ctx context.Context, userID string, startDate, endDate time.Time) (ports.FinancialSummary, error) {
	if startDate.IsZero() || endDate.IsZero() || !endDate.After(startDate) {
		return ports.FinancialSummary{}, domain.ErrInvalidTransactionDate
	}

	filter := ports.TransactionDateFilter{From: &startDate, To: &endDate}
	transactions, err := service.transactionRepository.GetByUserID(ctx, userID, filter)
	if err != nil {
		service.logger.Error("failed to retrieve transactions for financial summary", "userID", userID, "error", err)
		return ports.FinancialSummary{}, ErrInternalProcessing
	}

	return calculateFinancialSummary(transactions), nil
}

func (service *transactionService) getAuthorizedTransaction(ctx context.Context, userID, transactionID string) (*domain.Transaction, error) {
	transaction, err := service.transactionRepository.GetByID(ctx, transactionID)
	if errors.Is(err, ports.ErrTransactionNotFound) {
		return nil, ports.ErrTransactionNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve transaction", "userID", userID, "transactionID", transactionID, "error", err)
		return nil, ErrInternalProcessing
	}
	if transaction == nil {
		return nil, ports.ErrTransactionNotFound
	}
	if transaction.UserID() != userID {
		service.logger.Warn("unauthorized transaction access attempt", "userID", userID, "transactionID", transactionID)
		return nil, ports.ErrTransactionNotFound
	}

	return transaction, nil
}

func (service *transactionService) persistTransactionUpdate(ctx context.Context, transaction *domain.Transaction) error {
	if err := service.transactionRepository.Update(ctx, transaction); err != nil {
		service.logger.Error("failed to update transaction", "userID", transaction.UserID(), "transactionID", transaction.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func calculateFinancialSummary(transactions []*domain.Transaction) ports.FinancialSummary {
	var summary ports.FinancialSummary
	for _, transaction := range transactions {
		if transaction == nil || transaction.Status() != domain.TransactionStatusPaid {
			continue
		}

		switch transaction.Type() {
		case domain.TransactionTypeIncome:
			summary.TotalIncomeCents += transaction.AmountCents()
		case domain.TransactionTypeExpense:
			summary.TotalExpenseCents += transaction.AmountCents()
		}
	}

	summary.ProfitMarginCents = summary.TotalIncomeCents - summary.TotalExpenseCents
	return summary
}
