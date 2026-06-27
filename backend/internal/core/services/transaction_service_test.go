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
	transactionToReturn              *domain.Transaction
	transactionsToReturn             []*domain.Transaction
	createError                      error
	getByIDError                     error
	getByUserIDError                 error
	updateError                      error
	createAndUpdateError             error
	createPaymentCycleAndUpdateError error
	deleteError                      error
	createdTransaction               *domain.Transaction
	updatedTransaction               *domain.Transaction
	paidTransaction                  *domain.Transaction
	paidAccountPayable               *domain.Transaction
	cyclePayment                     ports.AccountPayableCyclePayment
	paidCyclesByUserID               map[string][]domain.PaidCycle
	paidCyclesByAccountPayableID     []domain.PaidCycle
	ledgerTransactions               []*domain.Transaction
	accountDeltas                    []ports.AccountBalanceDelta
	ledgerEntries                    []ports.LedgerEntry
	deletedTransactionID             string
	requestedTransactionID           string
	requestedUserID                  string
	receivedFilter                   ports.TransactionDateFilter
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

func (repository *mockTransactionRepository) GetByPaymentAccountID(ctx context.Context, userID, paymentAccountID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	repository.requestedUserID = userID
	repository.receivedFilter = filter
	return repository.transactionsToReturn, repository.getByUserIDError
}

func (repository *mockTransactionRepository) GetPaidCyclesByUserID(ctx context.Context, userID string) (map[string][]domain.PaidCycle, error) {
	repository.requestedUserID = userID
	return repository.paidCyclesByUserID, nil
}

func (repository *mockTransactionRepository) GetPaidCyclesByAccountPayableID(ctx context.Context, accountPayableID string) ([]domain.PaidCycle, error) {
	repository.requestedTransactionID = accountPayableID
	return repository.paidCyclesByAccountPayableID, nil
}

func (repository *mockTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	repository.updatedTransaction = transaction
	return repository.updateError
}

func (repository *mockTransactionRepository) CreateAndUpdate(ctx context.Context, transactionToCreate, transactionToUpdate *domain.Transaction) error {
	repository.paidTransaction = transactionToCreate
	repository.paidAccountPayable = transactionToUpdate
	return repository.createAndUpdateError
}

func (repository *mockTransactionRepository) CreatePaymentCycleAndUpdate(ctx context.Context, transactionToCreate, transactionToUpdate *domain.Transaction, cyclePayment ports.AccountPayableCyclePayment) error {
	repository.paidTransaction = transactionToCreate
	repository.paidAccountPayable = transactionToUpdate
	repository.cyclePayment = cyclePayment
	return repository.createPaymentCycleAndUpdateError
}

func (repository *mockTransactionRepository) CreateManyWithLedger(ctx context.Context, transactions []*domain.Transaction, accountDeltas []ports.AccountBalanceDelta, ledgerEntries []ports.LedgerEntry) error {
	repository.ledgerTransactions = transactions
	repository.accountDeltas = accountDeltas
	repository.ledgerEntries = ledgerEntries
	return repository.createError
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

type mockFinancialAccountRepository struct {
	accountToReturn *domain.FinancialAccount
}

func (repository *mockFinancialAccountRepository) Create(ctx context.Context, account *domain.FinancialAccount) error {
	return nil
}

func (repository *mockFinancialAccountRepository) GetByID(ctx context.Context, id string) (*domain.FinancialAccount, error) {
	if repository.accountToReturn == nil {
		return nil, ports.ErrFinancialAccountNotFound
	}
	return repository.accountToReturn, nil
}

func (repository *mockFinancialAccountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.FinancialAccount, error) {
	return nil, nil
}

func (repository *mockFinancialAccountRepository) Update(ctx context.Context, account *domain.FinancialAccount) error {
	return nil
}

func (repository *mockFinancialAccountRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func TestCreateTransaction_ValidData_ReturnsTransactionAndCreates(t *testing.T) {
	repository := &mockTransactionRepository{}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: validTransactionServiceTransactionID}, &mockTransactionLogger{})
	transactionDate := time.Now()

	transaction, err := service.CreateTransaction(context.Background(), validTransactionServiceUserID, domain.TransactionTypeIncome, validTransactionServiceConcept, validTransactionServiceCategory, 150000, transactionDate, domain.TransactionStatusPaid, nil, nil, domain.TransactionRecurrenceOnce, nil, nil)

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

func TestCreateTransaction_CreditPaymentAccountAssignsCreditCardID(t *testing.T) {
	repository := &mockTransactionRepository{}
	creditAccount := createTransactionServiceFinancialAccount(t, domain.FinancialAccountTypeCreditCard, "joy-card-123")
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: validTransactionServiceTransactionID}, &mockTransactionLogger{}, &mockFinancialAccountRepository{accountToReturn: creditAccount})
	paymentAccountID := "joy-card-123"

	_, err := service.CreateTransaction(context.Background(), validTransactionServiceUserID, domain.TransactionTypeExpense, "Collar", "Otros", 12000, time.Now(), domain.TransactionStatusPaid, nil, nil, domain.TransactionRecurrenceOnce, nil, &paymentAccountID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(repository.ledgerTransactions) != 1 {
		t.Fatalf("expected one ledger transaction, got %d", len(repository.ledgerTransactions))
	}
	if repository.ledgerTransactions[0].CreditCardID() == nil || *repository.ledgerTransactions[0].CreditCardID() != paymentAccountID {
		t.Fatalf("expected credit card ID %s, got %v", paymentAccountID, repository.ledgerTransactions[0].CreditCardID())
	}
}

func TestCreateTransaction_CreditMSIInstallmentsKeepCreditCardID(t *testing.T) {
	repository := &mockTransactionRepository{}
	creditAccount := createTransactionServiceFinancialAccount(t, domain.FinancialAccountTypeCreditCard, "joy-card-123")
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: validTransactionServiceTransactionID}, &mockTransactionLogger{}, &mockFinancialAccountRepository{accountToReturn: creditAccount})
	paymentAccountID := "joy-card-123"

	_, err := service.CreateTransaction(context.Background(), validTransactionServiceUserID, domain.TransactionTypeExpense, "Collar", "Otros", 300000, time.Now(), domain.TransactionStatusPaid, transactionServiceMSIPtr(6), nil, domain.TransactionRecurrenceOnce, nil, &paymentAccountID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(repository.ledgerTransactions) != 6 {
		t.Fatalf("expected six MSI transactions, got %d", len(repository.ledgerTransactions))
	}
	for _, transaction := range repository.ledgerTransactions {
		if transaction.CreditCardID() == nil || *transaction.CreditCardID() != paymentAccountID {
			t.Fatalf("expected installment credit card ID %s, got %v", paymentAccountID, transaction.CreditCardID())
		}
		if transaction.PaymentAccountID() == nil || *transaction.PaymentAccountID() != paymentAccountID {
			t.Fatalf("expected installment payment account ID %s, got %v", paymentAccountID, transaction.PaymentAccountID())
		}
	}
}

func TestCreateTransaction_InvalidAmount_ReturnsDomainError(t *testing.T) {
	service := NewTransactionService(&mockTransactionRepository{}, &mockTransactionIDGenerator{id: validTransactionServiceTransactionID}, &mockTransactionLogger{})

	_, err := service.CreateTransaction(context.Background(), validTransactionServiceUserID, domain.TransactionTypeIncome, validTransactionServiceConcept, validTransactionServiceCategory, 0, time.Now(), domain.TransactionStatusPaid, nil, nil, domain.TransactionRecurrenceOnce, nil, nil)

	if !errors.Is(err, domain.ErrInvalidTransactionAmount) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTransactionAmount, err)
	}
}

func TestCreateTransaction_RepositoryFailure_ReturnsErrInternalProcessing(t *testing.T) {
	repository := &mockTransactionRepository{createError: ports.ErrTransactionRepositoryUnavailable}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: validTransactionServiceTransactionID}, &mockTransactionLogger{})

	_, err := service.CreateTransaction(context.Background(), validTransactionServiceUserID, domain.TransactionTypeIncome, validTransactionServiceConcept, validTransactionServiceCategory, 150000, time.Now(), domain.TransactionStatusPaid, nil, nil, domain.TransactionRecurrenceOnce, nil, nil)

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

	err := service.UpdateTransaction(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID, domain.TransactionTypeExpense, "CFE - Luz", "Servicios", 45000, transactionDate, domain.TransactionStatusPaid, transactionServiceMSIPtr(3), nil, domain.TransactionRecurrenceOnce, nil)

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

func TestPayAccountPayable_OncePayable_CreatesPaymentAndCompletesOriginal(t *testing.T) {
	accountPayable := createTransactionServiceTransaction(t, validTransactionServiceUserID, domain.TransactionTypeExpense, domain.TransactionStatusPending, 12000)
	repository := &mockTransactionRepository{transactionToReturn: accountPayable}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: "payment-123"}, &mockTransactionLogger{})

	err := service.PayAccountPayable(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID, nil)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.paidTransaction == nil {
		t.Fatal("expected payment transaction to be created")
	}
	if repository.paidTransaction.ID() != "payment-123" {
		t.Errorf("expected payment ID payment-123, got %s", repository.paidTransaction.ID())
	}
	if repository.paidTransaction.Status() != domain.TransactionStatusCompleted {
		t.Errorf("expected payment status COMPLETED, got %s", repository.paidTransaction.Status())
	}
	if repository.paidAccountPayable == nil || repository.paidAccountPayable.Status() != domain.TransactionStatusCompleted {
		t.Fatalf("expected original account payable to be completed")
	}
	if repository.paidAccountPayable.LastPaidAt() == nil {
		t.Fatalf("expected original account payable last paid at to be set")
	}
}

func TestPayAccountPayable_MonthlyPayableWithLimit_AdvancesDateAndDecrementsLimit(t *testing.T) {
	limit := 2
	accountPayable := createTransactionServiceTransactionWithRecurrence(
		t,
		validTransactionServiceUserID,
		domain.TransactionRecurrenceMonthly,
		&limit,
		time.Date(2026, time.June, 10, 0, 0, 0, 0, time.UTC),
	)
	repository := &mockTransactionRepository{transactionToReturn: accountPayable}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: "payment-123"}, &mockTransactionLogger{})

	err := service.PayAccountPayable(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID, nil)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !repository.paidAccountPayable.Date().Equal(time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected next due date 2026-07-10, got %v", repository.paidAccountPayable.Date())
	}
	if repository.paidAccountPayable.Status() != domain.TransactionStatusPending {
		t.Errorf("expected original status PENDING, got %s", repository.paidAccountPayable.Status())
	}
	if repository.paidAccountPayable.RecurrenceLimit() == nil || *repository.paidAccountPayable.RecurrenceLimit() != 1 {
		t.Errorf("expected remaining recurrence limit 1, got %v", repository.paidAccountPayable.RecurrenceLimit())
	}
}

func TestPayAccountPayable_FutureMonthlyCycleWithOverdueCycles_RecordsCycleWithoutAdvancingOriginal(t *testing.T) {
	accountPayable := createTransactionServiceTransactionWithRecurrence(
		t,
		validTransactionServiceUserID,
		domain.TransactionRecurrenceMonthly,
		nil,
		time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC),
	)
	repository := &mockTransactionRepository{transactionToReturn: accountPayable}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: "payment-123"}, &mockTransactionLogger{})
	juneCycle := time.Date(2026, time.June, 16, 0, 0, 0, 0, time.UTC)

	err := service.PayAccountPayable(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID, &juneCycle)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !repository.paidAccountPayable.Date().Equal(time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected original due date to remain 2026-04-16, got %v", repository.paidAccountPayable.Date())
	}
	if repository.paidAccountPayable.Status() != domain.TransactionStatusPending {
		t.Errorf("expected original status PENDING, got %s", repository.paidAccountPayable.Status())
	}
	if !repository.cyclePayment.DueDate.Equal(juneCycle) {
		t.Errorf("expected paid cycle due date 2026-06-16, got %v", repository.cyclePayment.DueDate)
	}
	if repository.paidTransaction == nil || repository.paidTransaction.Status() != domain.TransactionStatusCompleted {
		t.Fatalf("expected completed payment transaction to be created")
	}
}

func TestPayAccountPayable_MonthlyPayableLimitReachesZero_CompletesOriginal(t *testing.T) {
	limit := 1
	accountPayable := createTransactionServiceTransactionWithRecurrence(
		t,
		validTransactionServiceUserID,
		domain.TransactionRecurrenceMonthly,
		&limit,
		time.Date(2026, time.June, 10, 0, 0, 0, 0, time.UTC),
	)
	repository := &mockTransactionRepository{transactionToReturn: accountPayable}
	service := NewTransactionService(repository, &mockTransactionIDGenerator{id: "payment-123"}, &mockTransactionLogger{})

	err := service.PayAccountPayable(context.Background(), validTransactionServiceUserID, validTransactionServiceTransactionID, nil)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.paidAccountPayable.Status() != domain.TransactionStatusCompleted {
		t.Errorf("expected original status COMPLETED, got %s", repository.paidAccountPayable.Status())
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
		domain.TransactionRecurrenceOnce,
		nil,
	)
	if err != nil {
		t.Fatalf("expected transaction to be valid, got: %v", err)
	}

	return transaction
}

func createTransactionServiceTransactionWithRecurrence(t *testing.T, userID string, recurrence domain.TransactionRecurrence, recurrenceLimit *int, date time.Time) *domain.Transaction {
	t.Helper()

	transaction, err := domain.NewTransaction(
		validTransactionServiceTransactionID,
		userID,
		domain.TransactionTypeExpense,
		validTransactionServiceConcept,
		validTransactionServiceCategory,
		10000,
		date,
		domain.TransactionStatusPending,
		nil,
		nil,
		recurrence,
		recurrenceLimit,
	)
	if err != nil {
		t.Fatalf("expected transaction to be valid, got: %v", err)
	}

	return transaction
}

func createTransactionServiceFinancialAccount(t *testing.T, accountType domain.FinancialAccountType, id string) *domain.FinancialAccount {
	t.Helper()

	creditLimit := int64(5000000)
	cutoffDay := 15
	paymentDay := 5
	account, err := domain.NewFinancialAccount(
		id,
		validTransactionServiceUserID,
		accountType,
		"Joy",
		"BBVA",
		nil,
		0,
		0,
		&creditLimit,
		&cutoffDay,
		&paymentDay,
		"from-blue-700 to-blue-900",
		domain.CreditCardNetworkVisa,
	)
	if err != nil {
		t.Fatalf("expected financial account to be valid, got: %v", err)
	}

	return account
}

func transactionServiceMSIPtr(msi int) *int {
	return &msi
}
