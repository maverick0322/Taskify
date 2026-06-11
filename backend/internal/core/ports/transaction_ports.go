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

// TransactionRepository defines the outbound port for financial transaction persistence.
type TransactionRepository interface {
	Create(ctx context.Context, transaction *domain.Transaction) error
	GetByID(ctx context.Context, id string) (*domain.Transaction, error)
	GetByUserID(ctx context.Context, userID string, filter TransactionDateFilter) ([]*domain.Transaction, error)
	Update(ctx context.Context, transaction *domain.Transaction) error
	Delete(ctx context.Context, id string) error
}

var (
	ErrTransactionNotFound              = errors.New("repository: transaction not found")
	ErrTransactionAlreadyExists         = errors.New("repository: transaction already exists")
	ErrTransactionRepositoryUnavailable = errors.New("repository: transaction persistence layer is unavailable or corrupted")
)
