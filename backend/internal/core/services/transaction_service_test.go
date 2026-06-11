package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	validTransactionServiceUserID        = "user-123"
	validTransactionServiceTransactionID = "transaction-123"
	validTransactionServiceConcept       = "Sueldo"
	validTransactionServiceCategory      = "Ingresos"
)

type mockTransactionRepository struct {
	transactionToReturn    *domain.Transaction
	transactionsToReturn   []*domain.Transaction
	createError            error
	getByIDError           error
	getByUserIDError       error
	updateError            error
	deleteError            error
	createdTransaction     *domain.Transaction
	updatedTransaction     *domain.Transaction
	deletedTransactionID   string
	requestedTransactionID string
	requestedUserID        string
	receivedFilter         ports.TransactionDateFilter
}

func (repository *mockTransactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	repository.createdTransaction = transaction
	return repository.createError
}

func (repository *mockTransactionRepository) GetByID(ctx context.Context, id string) (*domain.Transaction, error) {
	repository.requestedTransactionID = id
	return repository.transactionToReturn, repository.getByIDError
}

func (repository *mockTransactionRepository) GetByUserID(ctx context.Context, userID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	repository.requestedUserID = userID
	repository.receivedFilter = filter
	return repository.transactionsToReturn, repository.getByUserIDError
}

func (repository *mockTransactionRepository) GetByCreditCardID(ctx context.Context, userID, creditCardID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	repository.requestedUserID = userID
	repository.receivedFilter = filter
	return repository.transactionsToReturn, repository.getByUserIDError
}

func (repository *mockTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	repository.updatedTransaction = transaction
	return repository.updateError
}

func (repository *mockTransactionRepository) Delete(ctx context.Context, id string) error {
	repository.deletedTransactionID = id
	return repository.deleteError
}

type mockTransactionIDGenerator struct {
	id string
}

func (generator *mockTransactionIDGenerator) Generate() string {
	return generator.id
}

type mockTransactionLogger struct {
	warnMessages  []string
	errorMessages []string
}

func (logger *mockTransactionLogger) Info(msg string, keysAndValues ...interface{}) {}

func (logger *mockTransactionLogger) Warn(msg string, keysAndValues ...interface{}) {
	logger.warnMessages = append(logger.warnMessages, msg)
}

func (logger *mockTransactionLogger) Error(msg string, keysAndValues ...interface{}) {
	logger.errorMessages = append(logger.errorMessages, msg)
}

func TestCreateTransaction_ValidData_ReturnsTransactionAndCreates(t *testing.T) {
	repository := &mockTransactionRepository{}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: validTransactionServiceTransactionID}, &mockTransactionLogger{})
	transactionDate := time.Now()

	transaction, err := service.CreateTransaction(context.Background(), validTransactionServiceUserID, domain.TransactionTypeIncome, validTransactionServiceConcept, validTransactionServiceCategory, 150000, transactionDate, domain.TransactionStatusPaid, nil, nil)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if transaction.ID() != validTransactionServiceTransactionID {
		t.Errorf("expected transaction ID %s, got %s", validTransactionServiceTransactionID, transaction.ID())
	}
	if repository.createdTransaction == nil {
		t.Fatal("expected transaction to be created")
	}
}

func TestCreateTransaction_InvalidAmount_ReturnsDomainError(t *testing.T) {
	service := NewTransactionService(&mockTransactionRepository{}, &mockTransactionIDGenerator{id: validTransactionServiceTransactionID}, &mockTransactionLogger{})

	_, err := service.CreateTransaction(context.Background(), validTransactionServiceUserID, domain.TransactionTypeIncome, validTransactionServiceConcept, validTransactionServiceCategory, 0, time.Now(), domain.TransactionStatusPaid, nil, nil)

	if !errors.Is(err, domain.ErrInvalidTransactionAmount) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTransactionAmount, err)
	}
}

func TestCreateTransaction_RepositoryFailure_ReturnsErrInternalProcessing(t *testing.T) {
	repository := &mockTransactionRepository{createError: ports.ErrTransactionRepositoryUnavailable}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: validTransactionServiceTransactionID}, &mockTransactionLogger{})

	_, err := service.CreateTransaction(context.Background(), validTransactionServiceUserID, domain.TransactionTypeIncome, validTransactionServiceConcept, validTransactionServiceCategory, 150000, time.Now(), domain.TransactionStatusPaid, nil, nil)

	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestGetTransaction_OwnedTransaction_ReturnsTransaction(t *testing.T) {
	transaction := createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeIncome, domain.TransactionStatusPaid, 10000)
	repository := &mockTransactionRepository{transactionToReturn: transaction}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{}, &mockTransactionLogger{})

	retrievedTransaction, err := service.GetTransaction(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if retrievedTransaction.ID() != validTransactionServiceTransactionID {
		t.Errorf("expected transaction ID %s, got %s", validTransactionServiceTransactionID, retrievedTransaction.ID())
	}
}

func TestGetTransaction_UnauthorizedTransaction_ReturnsErrTransactionNotFoundAndWarns(t *testing.T) {
	transaction := createTransactionServiceTransaction(t, "other-user-123", domain.TransactionTypeIncome, domain.TransactionStatusPaid, 10000)
	repository := &mockTransactionRepository{transactionToReturn: transaction}
	logger := &mockTransactionLogger{}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{}, logger)

	_, err := service.GetTransaction(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID)

	if !errors.Is(err, ports.ErrTransactionNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTransactionNotFound, err)
	}
	if len(logger.warnMessages) != 1 {
		t.Fatalf("expected one warning log, got %d", len(logger.warnMessages))
	}
}

func TestGetUserTransactions_RepositorySuccess_ReturnsTransactions(t *testing.T) {
	transactions := []*domain.Transaction{createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeIncome, domain.TransactionStatusPaid, 10000)}
	repository := &mockTransactionRepository{transactionsToReturn: transactions}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{}, &mockTransactionLogger{})

	retrievedTransactions, err := service.GetUserTransactions(context.Background(), validTransactionServiceUserID, ports.TransactionDateFilter{})

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(retrievedTransactions) != 1 {
		t.Fatalf("expected one transaction, got %d", len(retrievedTransactions))
	}
	if repository.requestedUserID != validTransactionServiceUserID {
		t.Errorf("expected user ID %s, got %s", validTransactionServiceUserID, repository.requestedUserID)
	}
}

func TestUpdateTransaction_OwnedTransaction_UpdatesAndPersists(t *testing.T) {
	transaction := createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeExpense, domain.TransactionStatusPending, 12000)
	repository := &mockTransactionRepository{transactionToReturn: transaction}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{}, &mockTransactionLogger{})
	transactionDate := time.Now().Add(-24 * time.Hour)

	err := service.UpdateTransaction(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID, domain.TransactionTypeExpense, "CFE - Luz", "Servicios", 45000, transactionDate, domain.TransactionStatusPaid, transactionServiceMSIPtr(3), nil)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.updatedTransaction == nil {
		t.Fatal("expected transaction to be updated")
	}
	if repository.updatedTransaction.Concept() != "CFE - Luz" {
		t.Errorf("expected concept CFE - Luz, got %s", repository.updatedTransaction.Concept())
	}
	if repository.updatedTransaction.AmountCents() != 45000 {
		t.Errorf("expected amount cents 45000, got %d", repository.updatedTransaction.AmountCents())
	}
}

func TestDeleteTransaction_OwnedTransaction_DeletesTransaction(t *testing.T) {
	transaction := createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeIncome, domain.TransactionStatusPaid, 10000)
	repository := &mockTransactionRepository{transactionToReturn: transaction}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{}, &mockTransactionLogger{})

	err := service.DeleteTransaction(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.deletedTransactionID != validTransactionServiceTransactionID {
		t.Errorf("expected deleted transaction ID %s, got %s", validTransactionServiceTransactionID, repository.deletedTransactionID)
	}
}

func TestGetFinancialSummary_PaidTransactions_CalculatesMarginWithAmountCents(t *testing.T) {
	startDate := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	repository := &mockTransactionRepository{
		transactionsToReturn: []*domain.Transaction{
			createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeIncome, domain.TransactionStatusPaid, 10000),
			createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeIncome, domain.TransactionStatusPaid, 500),
			createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeExpense, domain.TransactionStatusPaid, 2500),
			createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeExpense, domain.TransactionStatusPending, 9999),
		},
	}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{}, &mockTransactionLogger{})

	summary, err := service.GetFinancialSummary(context.Background(), validTransactionServiceUserID, startDate, endDate)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if summary.TotalIncomeCents != 10500 {
		t.Errorf("expected total income cents 10500, got %d", summary.TotalIncomeCents)
	}
	if summary.TotalExpenseCents != 2500 {
		t.Errorf("expected total expense cents 2500, got %d", summary.TotalExpenseCents)
	}
	if summary.ProfitMarginCents != 8000 {
		t.Errorf("expected profit margin cents 8000, got %d", summary.ProfitMarginCents)
	}
	if repository.receivedFilter.From == nil || !repository.receivedFilter.From.Equal(startDate) {
		t.Errorf("expected summary start date filter %v, got %v", startDate, repository.receivedFilter.From)
	}
	if repository.receivedFilter.To == nil || !repository.receivedFilter.To.Equal(endDate) {
		t.Errorf("expected summary end date filter %v, got %v", endDate, repository.receivedFilter.To)
	}
}

func TestGetFinancialSummary_InvalidDateRange_ReturnsDomainError(t *testing.T) {
	service := NewTransactionService(&mockTransactionRepository{}, &mockTransactionIDGenerator{}, &mockTransactionLogger{})
	startDate := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

	_, err := service.GetFinancialSummary(context.Background(), validTransactionServiceUserID, startDate, startDate)

	if !errors.Is(err, domain.ErrInvalidTransactionDate) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTransactionDate, err)
	}
}

func TestGetFinancialSummary_RepositoryFailure_ReturnsErrInternalProcessing(t *testing.T) {
	repository := &mockTransactionRepository{getByUserIDError: ports.ErrTransactionRepositoryUnavailable}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{}, &mockTransactionLogger{})
	startDate := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	_, err := service.GetFinancialSummary(context.Background(), validTransactionServiceUserID, startDate, endDate)

	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func createTransactionServiceTransaction(t *testing.T, userID string, transactionType domain.TransactionType, status domain.TransactionStatus, amountCents int64) *domain.Transaction {
	t.Helper()

	transaction, err := domain.NewTransaction(
		validTransactionServiceTransactionID,
		userID,
		transactionType,
		validTransactionServiceConcept,
		validTransactionServiceCategory,
		amountCents,
		time.Now(),
		status,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("expected transaction to be valid, got: %v", err)
	}

	return transaction
}

func transactionServiceMSIPtr(msi int) *int {
	return &msi
}
