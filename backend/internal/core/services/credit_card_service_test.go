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
	validCreditCardServiceUserID       = "user-123"
	validCreditCardServiceCreditCardID = "credit-card-123"
)

type mockCreditCardRepository struct {
	creditCardToReturn      *domain.CreditCard
	creditCardsToReturn     []*domain.CreditCard
	createError             error
	getByIDError            error
	getByUserIDError        error
	updateError             error
	deleteError             error
	createdCreditCard       *domain.CreditCard
	updatedCreditCard       *domain.CreditCard
	deletedCreditCardID     string
	requestedCreditCardID   string
	requestedCreditCardUser string
}

func (repository *mockCreditCardRepository) Create(ctx context.Context, creditCard *domain.CreditCard) error {
	repository.createdCreditCard = creditCard
	return repository.createError
}

func (repository *mockCreditCardRepository) GetByID(ctx context.Context, id string) (*domain.CreditCard, error) {
	repository.requestedCreditCardID = id
	return repository.creditCardToReturn, repository.getByIDError
}

func (repository *mockCreditCardRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.CreditCard, error) {
	repository.requestedCreditCardUser = userID
	return repository.creditCardsToReturn, repository.getByUserIDError
}

func (repository *mockCreditCardRepository) Update(ctx context.Context, creditCard *domain.CreditCard) error {
	repository.updatedCreditCard = creditCard
	return repository.updateError
}

func (repository *mockCreditCardRepository) Delete(ctx context.Context, id string) error {
	repository.deletedCreditCardID = id
	return repository.deleteError
}

func TestCreateCreditCard_ValidData_ReturnsCreditCardAndCreates(t *testing.T) {
	repository := &mockCreditCardRepository{}
	service := NewCreditCardService(repository, &mockTransactionRepository{}, &mockTransactionIDGenerator{id: validCreditCardServiceCreditCardID}, &mockTransactionLogger{})

	creditCard, err := service.CreateCreditCard(context.Background(), validCreditCardServiceUserID, "Clasica", "BBVA", "1234", 15, 5, 5000000, "from-blue-500 to-sky-400", domain.CreditCardNetworkVisa)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if creditCard.ID() != validCreditCardServiceCreditCardID {
		t.Errorf("expected credit card ID %s, got %s", validCreditCardServiceCreditCardID, creditCard.ID())
	}
	if repository.createdCreditCard == nil {
		t.Fatal("expected credit card to be created")
	}
}

func TestCreateCreditCard_InvalidLast4_ReturnsDomainError(t *testing.T) {
	service := NewCreditCardService(&mockCreditCardRepository{}, &mockTransactionRepository{}, &mockTransactionIDGenerator{id: validCreditCardServiceCreditCardID}, &mockTransactionLogger{})

	_, err := service.CreateCreditCard(context.Background(), validCreditCardServiceUserID, "Clasica", "BBVA", "12", 15, 5, 5000000, "from-blue-500 to-sky-400", domain.CreditCardNetworkVisa)

	if !errors.Is(err, domain.ErrInvalidCreditCardLast4) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidCreditCardLast4, err)
	}
}

func TestGetCardsWithSummary_CurrentCycleSumsStoredInstallmentAmounts(t *testing.T) {
	now := time.Date(2026, time.June, 20, 12, 0, 0, 0, time.UTC)
	previousNow := creditCardCurrentTime
	creditCardCurrentTime = func() time.Time { return now }
	defer func() { creditCardCurrentTime = previousNow }()

	card := createCreditCardServiceCard(t, validCreditCardServiceUserID, 15)
	transactionRepository := &mockTransactionRepository{
		transactionsToReturn: []*domain.Transaction{
			createCreditCardServiceTransaction(t, domain.TransactionTypeExpense, domain.TransactionStatusPaid, 12000, nil),
			createCreditCardServiceTransaction(t, domain.TransactionTypeExpense, domain.TransactionStatusPaid, 50000, creditCardServiceMSIPtr(6)),
			createCreditCardServiceTransaction(t, domain.TransactionTypeIncome, domain.TransactionStatusPaid, 999999, nil),
			createCreditCardServiceTransaction(t, domain.TransactionTypeExpense, domain.TransactionStatusPending, 999999, nil),
		},
	}
	service := NewCreditCardService(
		&mockCreditCardRepository{creditCardsToReturn: []*domain.CreditCard{card}},
		transactionRepository,
		&mockTransactionIDGenerator{},
		&mockTransactionLogger{},
	)

	summaries, err := service.GetCardsWithSummary(context.Background(), validCreditCardServiceUserID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected one summary, got %d", len(summaries))
	}
	if summaries[0].CurrentDebtCents != 62000 {
		t.Errorf("expected current debt cents 62000, got %d", summaries[0].CurrentDebtCents)
	}
	if transactionRepository.receivedFilter.From == nil || !transactionRepository.receivedFilter.From.Equal(time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected cycle start 2026-06-15, got %v", transactionRepository.receivedFilter.From)
	}
	if transactionRepository.receivedFilter.To == nil || !transactionRepository.receivedFilter.To.Equal(time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected cycle end 2026-07-15, got %v", transactionRepository.receivedFilter.To)
	}
}

func TestGetCardsWithSummary_CurrentCycleSubtractsDebtPayments(t *testing.T) {
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)
	previousNow := creditCardCurrentTime
	creditCardCurrentTime = func() time.Time { return now }
	defer func() { creditCardCurrentTime = previousNow }()

	card := createCreditCardServiceCard(t, validCreditCardServiceUserID, 5)
	transactionRepository := &mockTransactionRepository{
		transactionsToReturn: []*domain.Transaction{
			createCreditCardServiceTransaction(t, domain.TransactionTypeExpense, domain.TransactionStatusPaid, 30000, nil),
			createCreditCardServiceTransaction(t, domain.TransactionTypeDebtPayment, domain.TransactionStatusPaid, 30000, nil),
		},
	}
	service := NewCreditCardService(
		&mockCreditCardRepository{creditCardsToReturn: []*domain.CreditCard{card}},
		transactionRepository,
		&mockTransactionIDGenerator{},
		&mockTransactionLogger{},
	)

	summaries, err := service.GetCardsWithSummary(context.Background(), validCreditCardServiceUserID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected one summary, got %d", len(summaries))
	}
	if summaries[0].CurrentDebtCents != 0 {
		t.Errorf("expected current debt cents 0 after payment, got %d", summaries[0].CurrentDebtCents)
	}
}

func TestGetCardsWithSummary_OnCutoffDayUsesClosingCycle(t *testing.T) {
	now := time.Date(2026, time.June, 15, 23, 0, 0, 0, time.UTC)
	previousNow := creditCardCurrentTime
	creditCardCurrentTime = func() time.Time { return now }
	defer func() { creditCardCurrentTime = previousNow }()

	card := createCreditCardServiceCard(t, validCreditCardServiceUserID, 15)
	transactionRepository := &mockTransactionRepository{}
	service := NewCreditCardService(
		&mockCreditCardRepository{creditCardsToReturn: []*domain.CreditCard{card}},
		transactionRepository,
		&mockTransactionIDGenerator{},
		&mockTransactionLogger{},
	)

	_, err := service.GetCardsWithSummary(context.Background(), validCreditCardServiceUserID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if transactionRepository.receivedFilter.From == nil || !transactionRepository.receivedFilter.From.Equal(time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected cycle start 2026-05-15, got %v", transactionRepository.receivedFilter.From)
	}
	if transactionRepository.receivedFilter.To == nil || !transactionRepository.receivedFilter.To.Equal(time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected cycle end 2026-06-15, got %v", transactionRepository.receivedFilter.To)
	}
}

func TestGetCardsWithSummary_BeforeCutoffUsesPreviousCycle(t *testing.T) {
	now := time.Date(2026, time.June, 10, 12, 0, 0, 0, time.UTC)
	previousNow := creditCardCurrentTime
	creditCardCurrentTime = func() time.Time { return now }
	defer func() { creditCardCurrentTime = previousNow }()

	card := createCreditCardServiceCard(t, validCreditCardServiceUserID, 15)
	transactionRepository := &mockTransactionRepository{}
	service := NewCreditCardService(
		&mockCreditCardRepository{creditCardsToReturn: []*domain.CreditCard{card}},
		transactionRepository,
		&mockTransactionIDGenerator{},
		&mockTransactionLogger{},
	)

	_, err := service.GetCardsWithSummary(context.Background(), validCreditCardServiceUserID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if transactionRepository.receivedFilter.From == nil || !transactionRepository.receivedFilter.From.Equal(time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected cycle start 2026-05-15, got %v", transactionRepository.receivedFilter.From)
	}
	if transactionRepository.receivedFilter.To == nil || !transactionRepository.receivedFilter.To.Equal(time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected cycle end 2026-06-15, got %v", transactionRepository.receivedFilter.To)
	}
}

func TestUpdateCreditCard_UnauthorizedCreditCard_ReturnsErrCreditCardNotFound(t *testing.T) {
	card := createCreditCardServiceCard(t, "other-user-123", 15)
	service := NewCreditCardService(&mockCreditCardRepository{creditCardToReturn: card}, &mockTransactionRepository{}, &mockTransactionIDGenerator{}, &mockTransactionLogger{})

	err := service.UpdateCreditCard(context.Background(), validCreditCardServiceUserID, validCreditCardServiceCreditCardID, "Oro", "BBVA", "1234", 15, 5, 5000000, "from-blue-500 to-sky-400", domain.CreditCardNetworkVisa)

	if !errors.Is(err, ports.ErrCreditCardNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrCreditCardNotFound, err)
	}
}

func TestDeleteCreditCard_OwnedCreditCard_DeletesCreditCard(t *testing.T) {
	card := createCreditCardServiceCard(t, validCreditCardServiceUserID, 15)
	repository := &mockCreditCardRepository{creditCardToReturn: card}
	service := NewCreditCardService(repository, &mockTransactionRepository{}, &mockTransactionIDGenerator{}, &mockTransactionLogger{})

	err := service.DeleteCreditCard(context.Background(), validCreditCardServiceUserID, validCreditCardServiceCreditCardID)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.deletedCreditCardID != validCreditCardServiceCreditCardID {
		t.Errorf("expected deleted credit card ID %s, got %s", validCreditCardServiceCreditCardID, repository.deletedCreditCardID)
	}
}

func TestPayCreditCardDebt_AssignsCreditCardIDToPaymentTransaction(t *testing.T) {
	card := createCreditCardServiceCard(t, validCreditCardServiceUserID, 15)
	sourceAccount := createTransactionServiceFinancialAccount(t, domain.FinancialAccountTypeDebitCard, "cash-123")
	if err := sourceAccount.ApplyDelta(50000); err != nil {
		t.Fatalf("expected source account balance setup to succeed, got %v", err)
	}
	creditAccount := createTransactionServiceFinancialAccount(t, domain.FinancialAccountTypeCreditCard, validCreditCardServiceCreditCardID)
	if err := creditAccount.ApplyDelta(30000); err != nil {
		t.Fatalf("expected credit account balance setup to succeed, got %v", err)
	}

	transactionRepository := &mockTransactionRepository{}
	accountRepository := &multiAccountRepository{
		accountsByID: map[string]*domain.FinancialAccount{
			sourceAccount.ID(): sourceAccount,
			creditAccount.ID(): creditAccount,
		},
	}
	service := NewCreditCardService(
		&mockCreditCardRepository{creditCardToReturn: card},
		transactionRepository,
		&mockTransactionIDGenerator{id: "payment-123"},
		&mockTransactionLogger{},
		accountRepository,
	)

	_, err := service.PayCreditCardDebt(context.Background(), validCreditCardServiceUserID, validCreditCardServiceCreditCardID, sourceAccount.ID(), 30000)

	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(transactionRepository.ledgerTransactions) != 1 {
		t.Fatalf("expected one payment transaction, got %d", len(transactionRepository.ledgerTransactions))
	}
	payment := transactionRepository.ledgerTransactions[0]
	if payment.CreditCardID() == nil || *payment.CreditCardID() != validCreditCardServiceCreditCardID {
		t.Fatalf("expected credit card ID %s, got %v", validCreditCardServiceCreditCardID, payment.CreditCardID())
	}
	if payment.PaymentAccountID() == nil || *payment.PaymentAccountID() != sourceAccount.ID() {
		t.Fatalf("expected payment account ID %s, got %v", sourceAccount.ID(), payment.PaymentAccountID())
	}
	if payment.DestinationAccountID() == nil || *payment.DestinationAccountID() != validCreditCardServiceCreditCardID {
		t.Fatalf("expected destination account ID %s, got %v", validCreditCardServiceCreditCardID, payment.DestinationAccountID())
	}
}

func TestProjectCreditCardStatements_UsesClosedCycleAndNetsPayments(t *testing.T) {
	card := createCreditCardServiceCard(t, validCreditCardServiceUserID, 5)
	cardID := card.ID()
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	purchase, _ := domain.NewTransaction("purchase", card.UserID(), domain.TransactionTypeExpense, "Compra de contado", "Otros", 50000, time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC), domain.TransactionStatusCompleted, nil, &cardID, domain.TransactionRecurrenceOnce, nil)
	purchase.SetCreditCardID(&cardID)
	installments := 12
	installmentNumber := 1
	installment, _ := domain.NewTransaction("installment", card.UserID(), domain.TransactionTypeExpense, "Computadora 1/12", "Equipo", 10000, time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC), domain.TransactionStatusPending, &installments, &cardID, domain.TransactionRecurrenceOnce, nil)
	installment.SetAccountingDetails(&cardID, nil, &installmentNumber, &installments)
	installment.SetCreditCardID(&cardID)
	payment, _ := domain.NewTransaction("old-payment", card.UserID(), domain.TransactionTypeDebtPayment, "Pago", "Tarjeta de crédito", 15000, time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC), domain.TransactionStatusCompleted, nil, &cardID, domain.TransactionRecurrenceOnce, nil)
	payment.SetCreditCardID(&cardID)

	statements, items, allocations := projectCreditCardStatements(card, []*domain.Transaction{purchase, installment, payment}, now)
	if len(statements) != 1 || len(items) != 2 || len(allocations) != 1 {
		t.Fatalf("expected one statement, two items and one allocation; got %d, %d, %d", len(statements), len(items), len(allocations))
	}
	statement := statements[0]
	if statement.StatementAmountCents != 60000 || statement.PaidAmountCents != 15000 {
		t.Fatalf("expected statement 60000/15000, got %d/%d", statement.StatementAmountCents, statement.PaidAmountCents)
	}
	if statement.PaymentDueDate.Format("2006-01-02") != "2026-08-05" {
		t.Fatalf("expected payment due date 2026-08-05, got %s", statement.PaymentDueDate.Format("2006-01-02"))
	}
}

func TestPaymentDueDate_UsesFirstPaymentDayAfterCutoffAndClampsMonth(t *testing.T) {
	location := time.FixedZone("Mexico", -6*60*60)
	if got := paymentDueDate(time.Date(2026, 7, 5, 0, 0, 0, 0, location), 25); got.Format("2006-01-02") != "2026-07-25" {
		t.Fatalf("expected same-month due date, got %s", got.Format("2006-01-02"))
	}
	if got := paymentDueDate(time.Date(2026, 7, 25, 0, 0, 0, 0, location), 10); got.Format("2006-01-02") != "2026-08-10" {
		t.Fatalf("expected next-month due date, got %s", got.Format("2006-01-02"))
	}
	if got := paymentDueDate(time.Date(2026, 2, 28, 0, 0, 0, 0, location), 31); got.Format("2006-01-02") != "2026-03-31" {
		t.Fatalf("expected clamped cycle to advance to March 31, got %s", got.Format("2006-01-02"))
	}
}

func TestAllocatePayment_AppliesOldestStatementFirst(t *testing.T) {
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	statements := []ports.CreditCardStatement{
		{ID: "old", UserID: "user", StatementAmountCents: 30000, PaidAmountCents: 0, PaymentDueDate: time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC)},
		{ID: "new", UserID: "user", StatementAmountCents: 20000, PaidAmountCents: 0, PaymentDueDate: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)},
	}
	allocations, updated := allocatePayment("user", "payment", 35000, statements, now)
	if len(allocations) != 2 || allocations[0].StatementID != "old" || allocations[0].AmountCents != 30000 || allocations[1].AmountCents != 5000 {
		t.Fatalf("expected oldest-first allocations, got %+v", allocations)
	}
	if updated[0].Status != "PAID" || updated[1].PaidAmountCents != 5000 {
		t.Fatalf("expected paid oldest and partial newest, got %+v", updated)
	}
}

func createCreditCardServiceCard(t *testing.T, userID string, cutoffDay int) *domain.CreditCard {
	t.Helper()

	card, err := domain.NewCreditCard(validCreditCardServiceCreditCardID, userID, "Clasica", "BBVA", "1234", cutoffDay, 5, 5000000, "from-blue-500 to-sky-400", domain.CreditCardNetworkVisa)
	if err != nil {
		t.Fatalf("expected credit card to be valid, got: %v", err)
	}

	return card
}

func createCreditCardServiceTransaction(t *testing.T, transactionType domain.TransactionType, status domain.TransactionStatus, amountCents int64, msi *int) *domain.Transaction {
	t.Helper()

	transaction, err := domain.NewTransaction(
		"transaction-123",
		validCreditCardServiceUserID,
		transactionType,
		"Compra",
		"General",
		amountCents,
		time.Now(),
		status,
		msi,
		&[]string{validCreditCardServiceCreditCardID}[0],
		domain.TransactionRecurrenceOnce,
		nil,
	)
	if err != nil {
		t.Fatalf("expected transaction to be valid, got: %v", err)
	}

	return transaction
}

func creditCardServiceMSIPtr(msi int) *int {
	return &msi
}

type multiAccountRepository struct {
	accountsByID map[string]*domain.FinancialAccount
}

func (repository *multiAccountRepository) Create(ctx context.Context, account *domain.FinancialAccount) error {
	return nil
}

func (repository *multiAccountRepository) GetByID(ctx context.Context, id string) (*domain.FinancialAccount, error) {
	account := repository.accountsByID[id]
	if account == nil {
		return nil, ports.ErrFinancialAccountNotFound
	}
	return account, nil
}

func (repository *multiAccountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.FinancialAccount, error) {
	return nil, nil
}

func (repository *multiAccountRepository) Update(ctx context.Context, account *domain.FinancialAccount) error {
	return nil
}

func (repository *multiAccountRepository) Delete(ctx context.Context, id string) error {
	return nil
}
