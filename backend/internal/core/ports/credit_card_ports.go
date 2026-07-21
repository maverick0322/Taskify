package ports

import (
	"context"
	"errors"
	"time"

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
	CreditCard        *domain.CreditCard
	CurrentDebtCents  int64
	TotalBalanceCents int64
	PaymentDueDate    *time.Time
	Status            string
}

type CreditCardStatement struct {
	ID                   string
	UserID               string
	CreditAccountID      string
	CycleStart           time.Time
	CycleEnd             time.Time
	PaymentDueDate       time.Time
	StatementAmountCents int64
	PaidAmountCents      int64
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type CreditCardStatementItem struct {
	ID            string
	UserID        string
	StatementID   string
	TransactionID string
	AmountCents   int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreditCardPaymentAllocation struct {
	ID                   string
	UserID               string
	StatementID          string
	PaymentTransactionID string
	AmountCents          int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type CreditCardPaymentResult struct {
	PaymentTransactionID string   `json:"paymentTransactionId"`
	AppliedAmountCents   int64    `json:"appliedAmountCents"`
	RemainingDebtCents   int64    `json:"remainingDebtCents"`
	AffectedStatementIDs []string `json:"affectedStatementIds"`
}

type FinancialPayable struct {
	ID              string     `json:"id"`
	Type            string     `json:"type"`
	SourceID        string     `json:"sourceId"`
	Name            string     `json:"name"`
	AmountCents     int64      `json:"amountCents"`
	DueDate         time.Time  `json:"dueDate"`
	Status          string     `json:"status"`
	CreditCardID    *string    `json:"creditCardId,omitempty"`
	TransactionID   *string    `json:"transactionId,omitempty"`
	Category        string     `json:"category,omitempty"`
	SourceDate      *time.Time `json:"sourceDate,omitempty"`
	Recurrence      string     `json:"recurrence,omitempty"`
	RecurrenceLimit *int       `json:"recurrenceLimit,omitempty"`
}

type CreditCardStatementRepository interface {
	Reconcile(ctx context.Context, userID, creditCardID string, statements []CreditCardStatement, items []CreditCardStatementItem, allocations []CreditCardPaymentAllocation) error
	GetByCreditCardID(ctx context.Context, userID, creditCardID string) ([]CreditCardStatement, error)
	ApplyPayment(ctx context.Context, payment *domain.Transaction, accountDeltas []AccountBalanceDelta, ledgerEntries []LedgerEntry, allocations []CreditCardPaymentAllocation, statements []CreditCardStatement) error
}

type CreditCardUseCase interface {
	CreateCreditCard(ctx context.Context, userID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color, network string) (*domain.CreditCard, error)
	GetCardsWithSummary(ctx context.Context, userID string) ([]CreditCardWithSummary, error)
	GetPayables(ctx context.Context, userID, timezone string) ([]FinancialPayable, error)
	UpdateCreditCard(ctx context.Context, userID, creditCardID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color, network string) error
	PayCreditCardDebt(ctx context.Context, userID, creditCardID, sourceAccountID string, amountCents int64, timezone ...string) (CreditCardPaymentResult, error)
	DeleteCreditCard(ctx context.Context, userID, creditCardID string) error
}

var (
	ErrCreditCardNotFound              = errors.New("repository: credit card not found")
	ErrCreditCardAlreadyExists         = errors.New("repository: credit card already exists")
	ErrCreditCardRepositoryUnavailable = errors.New("repository: credit card persistence layer is unavailable or corrupted")
	ErrCreditCardStatementUnavailable  = errors.New("repository: credit card statement persistence layer is unavailable or corrupted")
	ErrCreditCardNoPayableDebt         = errors.New("service: credit card has no payable debt")
	ErrCreditCardPaymentExceedsDebt    = errors.New("service: credit card payment exceeds payable debt")
	ErrCreditCardPaymentConflict       = errors.New("service: credit card payment changed concurrently")
)
