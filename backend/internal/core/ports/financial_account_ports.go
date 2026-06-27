package ports

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

type FinancialAccountRepository interface {
	Create(ctx context.Context, account *domain.FinancialAccount) error
	GetByID(ctx context.Context, id string) (*domain.FinancialAccount, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.FinancialAccount, error)
	Update(ctx context.Context, account *domain.FinancialAccount) error
	Delete(ctx context.Context, id string) error
}

type FinancialAccountUseCase interface {
	CreateAccount(ctx context.Context, userID string, accountType domain.FinancialAccountType, name, institution string, last4 *string, openingBalanceCents int64, creditLimitCents *int64, cutoffDay, paymentDay *int, color, network string) (*domain.FinancialAccount, error)
	GetAccounts(ctx context.Context, userID string) ([]*domain.FinancialAccount, error)
	GetAccountSummary(ctx context.Context, userID, accountID string) (FinancialAccountSummary, error)
	GetAccountTransactions(ctx context.Context, userID, accountID string, filter TransactionDateFilter) ([]*domain.Transaction, error)
	UpdateAccount(ctx context.Context, userID, accountID string, accountType domain.FinancialAccountType, name, institution string, last4 *string, openingBalanceCents int64, creditLimitCents *int64, cutoffDay, paymentDay *int, color, network string) error
	DeleteAccount(ctx context.Context, userID, accountID string) error
}

type FinancialAccountSummary struct {
	AccountID            string
	CurrentBalanceCents  int64
	OpeningBalanceCents  int64
	CreditLimitCents     *int64
	CurrentDebtCents     int64
	AvailableCreditCents int64
	TotalIncomeCents     int64
	TotalExpenseCents    int64
	CalculatedAt         time.Time
}

var (
	ErrFinancialAccountNotFound              = errors.New("repository: financial account not found")
	ErrFinancialAccountAlreadyExists         = errors.New("repository: financial account already exists")
	ErrFinancialAccountRepositoryUnavailable = errors.New("repository: financial account persistence layer is unavailable or corrupted")
)
