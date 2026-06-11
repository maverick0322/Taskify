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

type CreditCardWithSummary struct {
	CreditCard       *domain.CreditCard
	CurrentDebtCents int64
}

type CreditCardUseCase interface {
	CreateCreditCard(ctx context.Context, userID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) (*domain.CreditCard, error)
	GetCardsWithSummary(ctx context.Context, userID string) ([]CreditCardWithSummary, error)
	UpdateCreditCard(ctx context.Context, userID, creditCardID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) error
	DeleteCreditCard(ctx context.Context, userID, creditCardID string) error
}

var (
	ErrCreditCardNotFound              = errors.New("repository: credit card not found")
	ErrCreditCardAlreadyExists         = errors.New("repository: credit card already exists")
	ErrCreditCardRepositoryUnavailable = errors.New("repository: credit card persistence layer is unavailable or corrupted")
)
