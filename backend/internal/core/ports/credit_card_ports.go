package ports

import (
	"context"
	"errors"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

type CreditCardRepository interface {
	Create(ctx context.Context, creditCard *domain.CreditCard) error
	GetByID(ctx context.Context, id string) (*domain.CreditCard, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.CreditCard, error)
	Update(ctx context.Context, creditCard *domain.CreditCard) error
	Delete(ctx context.Context, id string) error
}

var (
	ErrCreditCardNotFound              = errors.New("repository: credit card not found")
	ErrCreditCardAlreadyExists         = errors.New("repository: credit card already exists")
	ErrCreditCardRepositoryUnavailable = errors.New("repository: credit card persistence layer is unavailable or corrupted")
)
