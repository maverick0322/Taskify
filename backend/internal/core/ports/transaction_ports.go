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

// TransactionRepository defines the outbound port for financial transaction persistence.
type TransactionRepository interface {
	Create(ctx context.Context, transaction *domain.Transaction) error
	GetByID(ctx context.Context, id string) (*domain.Transaction, error)
	GetByUserID(ctx context.Context, userID string, filter TransactionDateFilter) ([]*domain.Transaction, error)
	Update(ctx context.Context, transaction *domain.Transaction) error
	Delete(ctx context.Context, id string) error
}

// TransactionUseCase defines user-scoped application operations for financial transactions.
type TransactionUseCase interface {
	CreateTransaction(ctx context.Context, userID string, transactionType domain.TransactionType, concept, category string, amountCents int64, date time.Time, status domain.TransactionStatus, msi *int) (*domain.Transaction, error)
	GetTransaction(ctx context.Context, userID, transactionID string) (*domain.Transaction, error)
	GetUserTransactions(ctx context.Context, userID string, filter TransactionDateFilter) ([]*domain.Transaction, error)
	UpdateTransaction(ctx context.Context, userID, transactionID string, transactionType domain.TransactionType, concept, category string, amountCents int64, date time.Time, status domain.TransactionStatus, msi *int) error
	DeleteTransaction(ctx context.Context, userID, transactionID string) error
	GetFinancialSummary(ctx context.Context, userID string, startDate, endDate time.Time) (FinancialSummary, error)
}

var (
	ErrTransactionNotFound              = errors.New("repository: transaction not found")
	ErrTransactionAlreadyExists         = errors.New("repository: transaction already exists")
	ErrTransactionRepositoryUnavailable = errors.New("repository: transaction persistence layer is unavailable or corrupted")
)
