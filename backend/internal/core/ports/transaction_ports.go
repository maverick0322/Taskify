package ports

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

type TransactionDateFilter struct {
	From *time.Time
	To   *time.Time
}

type FinancialSummary struct {
	TotalIncomeCents  int64
	TotalExpenseCents int64
	ProfitMarginCents int64
}

type AccountBalanceDelta struct {
	AccountID  string
	DeltaCents int64
}

type LedgerEntry struct {
	ID            string
	UserID        string
	AccountID     string
	TransactionID string
	AmountCents   int64
	EntryType     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AccountPayableCyclePayment struct {
	ID                   string
	AccountPayableID     string
	UserID               string
	DueDate              time.Time
	PaidAt               time.Time
	AmountCents          int64
	Concept              string
	Category             string
	CreatedTransactionID string
	CreatedAt            time.Time
}

// TransactionRepository defines the outbound port for financial transaction persistence.
type TransactionRepository interface {
	Create(ctx context.Context, transaction *domain.Transaction) error
	GetByID(ctx context.Context, id string) (*domain.Transaction, error)
	GetByUserID(ctx context.Context, userID string, filter TransactionDateFilter) ([]*domain.Transaction, error)
	GetByCreditCardID(ctx context.Context, userID, creditCardID string, filter TransactionDateFilter) ([]*domain.Transaction, error)
	GetByPaymentAccountID(ctx context.Context, userID, paymentAccountID string, filter TransactionDateFilter) ([]*domain.Transaction, error)
	GetPaidCyclesByUserID(ctx context.Context, userID string) (map[string][]domain.PaidCycle, error)
	GetPaidCyclesByAccountPayableID(ctx context.Context, accountPayableID string) ([]domain.PaidCycle, error)
	Update(ctx context.Context, transaction *domain.Transaction) error
	CreateAndUpdate(ctx context.Context, transactionToCreate, transactionToUpdate *domain.Transaction) error
	CreatePaymentCycleAndUpdate(ctx context.Context, transactionToCreate, transactionToUpdate *domain.Transaction, cyclePayment AccountPayableCyclePayment) error
	CreateManyWithLedger(ctx context.Context, transactions []*domain.Transaction, accountDeltas []AccountBalanceDelta, ledgerEntries []LedgerEntry) error
	Delete(ctx context.Context, id string) error
}

// TransactionUseCase defines user-scoped application operations for financial transactions.
type TransactionUseCase interface {
	CreateTransaction(ctx context.Context, userID string, transactionType domain.TransactionType, concept, category string, amountCents int64, date time.Time, status domain.TransactionStatus, msi *int, creditCardID *string, recurrence domain.TransactionRecurrence, recurrenceLimit *int, paymentAccountID *string) (*domain.Transaction, error)
	GetTransaction(ctx context.Context, userID, transactionID string) (*domain.Transaction, error)
	GetUserTransactions(ctx context.Context, userID string, filter TransactionDateFilter) ([]*domain.Transaction, error)
	UpdateTransaction(ctx context.Context, userID, transactionID string, transactionType domain.TransactionType, concept, category string, amountCents int64, date time.Time, status domain.TransactionStatus, msi *int, creditCardID *string, recurrence domain.TransactionRecurrence, recurrenceLimit *int) error
	PayAccountPayable(ctx context.Context, userID, transactionID string, dueDate *time.Time) error
	DeleteTransaction(ctx context.Context, userID, transactionID string) error
	GetFinancialSummary(ctx context.Context, userID string, startDate, endDate time.Time) (FinancialSummary, error)
}

var (
	ErrTransactionNotFound              = errors.New("repository: transaction not found")
	ErrTransactionAlreadyExists         = errors.New("repository: transaction already exists")
	ErrAccountPayableCycleAlreadyPaid   = errors.New("repository: account payable cycle already paid")
	ErrTransactionRepositoryUnavailable = errors.New("repository: transaction persistence layer is unavailable or corrupted")
)
