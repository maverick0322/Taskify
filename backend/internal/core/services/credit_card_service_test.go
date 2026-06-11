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

	creditCard, err := service.CreateCreditCard(context.Background(), validCreditCardServiceUserID, "Clasica", "BBVA", "1234", 15, 5, 5000000, "from-blue-500 to-sky-400")

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

	_, err := service.CreateCreditCard(context.Background(), validCreditCardServiceUserID, "Clasica", "BBVA", "12", 15, 5, 5000000, "from-blue-500 to-sky-400")

	if !errors.Is(err, domain.ErrInvalidCreditCardLast4) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidCreditCardLast4, err)
	}
}

func TestGetCardsWithSummary_CurrentCycleCalculatesDebtAndMSIWithoutLosingCents(t *testing.T) {
	now := time.Date(2026, time.June, 20, 12, 0, 0, 0, time.UTC)
	previousNow := creditCardCurrentTime
	creditCardCurrentTime = func() time.Time { return now }
	defer func() { creditCardCurrentTime = previousNow }()

	card := createCreditCardServiceCard(t, validCreditCardServiceUserID, 15)
	transactionRepository := &mockTransactionRepository{
		transactionsToReturn: []*domain.Transaction{
			createCreditCardServiceTransaction(t, domain.TransactionTypeExpense, domain.TransactionStatusPaid, 12000, nil),
			createCreditCardServiceTransaction(t, domain.TransactionTypeExpense, domain.TransactionStatusPaid, 10000, creditCardServiceMSIPtr(3)),
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
	if summaries[0].CurrentDebtCents != 15334 {
		t.Errorf("expected current debt cents 15334, got %d", summaries[0].CurrentDebtCents)
	}
	if transactionRepository.receivedFilter.From == nil || !transactionRepository.receivedFilter.From.Equal(time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected cycle start 2026-06-15, got %v", transactionRepository.receivedFilter.From)
	}
	if transactionRepository.receivedFilter.To == nil || !transactionRepository.receivedFilter.To.Equal(time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected cycle end 2026-07-15, got %v", transactionRepository.receivedFilter.To)
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

	err := service.UpdateCreditCard(context.Background(), validCreditCardServiceUserID, validCreditCardServiceCreditCardID, "Oro", "BBVA", "1234", 15, 5, 5000000, "from-blue-500 to-sky-400")

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

func createCreditCardServiceCard(t *testing.T, userID string, cutoffDay int) *domain.CreditCard {
	t.Helper()

	card, err := domain.NewCreditCard(validCreditCardServiceCreditCardID, userID, "Clasica", "BBVA", "1234", cutoffDay, 5, 5000000, "from-blue-500 to-sky-400")
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
	)
	if err != nil {
		t.Fatalf("expected transaction to be valid, got: %v", err)
	}

	return transaction
}

func creditCardServiceMSIPtr(msi int) *int {
	return &msi
}
