package domain

import (
	"errors"
	"strings"
	"time"
)

type FinancialAccountType string

const (
	FinancialAccountTypeCash       FinancialAccountType = "CASH"
	FinancialAccountTypeDebitCard  FinancialAccountType = "DEBIT_CARD"
	FinancialAccountTypeCreditCard FinancialAccountType = "CREDIT_CARD"
)

func (accountType FinancialAccountType) IsValid() bool {
	return accountType == FinancialAccountTypeCash ||
		accountType == FinancialAccountTypeDebitCard ||
		accountType == FinancialAccountTypeCreditCard
}

var (
	ErrInvalidFinancialAccountID           = errors.New("domain: financial account ID cannot be empty")
	ErrInvalidFinancialAccountUserID       = errors.New("domain: financial account user ID cannot be empty")
	ErrInvalidFinancialAccountType         = errors.New("domain: invalid financial account type")
	ErrInvalidFinancialAccountName         = errors.New("domain: financial account name cannot be empty")
	ErrInvalidFinancialAccountBalance      = errors.New("domain: financial account balance cannot be negative")
	ErrInvalidFinancialAccountCreditLimit  = errors.New("domain: credit account limit must be greater than zero")
	ErrInvalidFinancialAccountCutoffDay    = errors.New("domain: credit account cutoff day must be between 1 and 31")
	ErrInvalidFinancialAccountPaymentDay   = errors.New("domain: credit account payment day must be between 1 and 31")
	ErrInsufficientFinancialAccountFunds   = errors.New("domain: insufficient financial account funds")
	ErrFinancialAccountCreditLimitExceeded = errors.New("domain: credit account limit exceeded")
)

type FinancialAccount struct {
	id                  string
	userID              string
	accountType         FinancialAccountType
	name                string
	institution         string
	last4               *string
	openingBalanceCents int64
	currentBalanceCents int64
	creditLimitCents    *int64
	cutoffDay           *int
	paymentDay          *int
	color               string
	createdAt           time.Time
	updatedAt           time.Time
}

func NewFinancialAccount(id, userID string, accountType FinancialAccountType, name, institution string, last4 *string, openingBalanceCents, currentBalanceCents int64, creditLimitCents *int64, cutoffDay, paymentDay *int, color string) (*FinancialAccount, error) {
	fields, err := validateFinancialAccountFields(id, userID, accountType, name, openingBalanceCents, currentBalanceCents, creditLimitCents, cutoffDay, paymentDay)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &FinancialAccount{
		id:                  fields.id,
		userID:              fields.userID,
		accountType:         accountType,
		name:                fields.name,
		institution:         strings.TrimSpace(institution),
		last4:               normalizeOptionalFinancialString(last4),
		openingBalanceCents: openingBalanceCents,
		currentBalanceCents: currentBalanceCents,
		creditLimitCents:    normalizeOptionalInt64(creditLimitCents),
		cutoffDay:           normalizeOptionalInt(cutoffDay),
		paymentDay:          normalizeOptionalInt(paymentDay),
		color:               strings.TrimSpace(color),
		createdAt:           now,
		updatedAt:           now,
	}, nil
}

func RehydrateFinancialAccount(id, userID string, accountType FinancialAccountType, name, institution string, last4 *string, openingBalanceCents, currentBalanceCents int64, creditLimitCents *int64, cutoffDay, paymentDay *int, color string, createdAt, updatedAt time.Time) (*FinancialAccount, error) {
	account, err := NewFinancialAccount(id, userID, accountType, name, institution, last4, openingBalanceCents, currentBalanceCents, creditLimitCents, cutoffDay, paymentDay, color)
	if err != nil {
		return nil, err
	}
	account.createdAt = createdAt
	account.updatedAt = updatedAt
	return account, nil
}

func (account *FinancialAccount) ApplyDelta(deltaCents int64) error {
	nextBalance := account.currentBalanceCents + deltaCents
	if account.accountType != FinancialAccountTypeCreditCard && nextBalance < 0 {
		return ErrInsufficientFinancialAccountFunds
	}
	if account.accountType == FinancialAccountTypeCreditCard && account.creditLimitCents != nil && nextBalance > *account.creditLimitCents {
		return ErrFinancialAccountCreditLimitExceeded
	}
	account.currentBalanceCents = nextBalance
	account.updatedAt = time.Now()
	return nil
}

func (account *FinancialAccount) ID() string                 { return account.id }
func (account *FinancialAccount) UserID() string             { return account.userID }
func (account *FinancialAccount) Type() FinancialAccountType { return account.accountType }
func (account *FinancialAccount) Name() string               { return account.name }
func (account *FinancialAccount) Institution() string        { return account.institution }
func (account *FinancialAccount) Last4() *string {
	return normalizeOptionalFinancialString(account.last4)
}
func (account *FinancialAccount) OpeningBalanceCents() int64 { return account.openingBalanceCents }
func (account *FinancialAccount) CurrentBalanceCents() int64 { return account.currentBalanceCents }
func (account *FinancialAccount) CreditLimitCents() *int64 {
	return normalizeOptionalInt64(account.creditLimitCents)
}
func (account *FinancialAccount) CutoffDay() *int      { return normalizeOptionalInt(account.cutoffDay) }
func (account *FinancialAccount) PaymentDay() *int     { return normalizeOptionalInt(account.paymentDay) }
func (account *FinancialAccount) Color() string        { return account.color }
func (account *FinancialAccount) CreatedAt() time.Time { return account.createdAt }
func (account *FinancialAccount) UpdatedAt() time.Time { return account.updatedAt }

func validateFinancialAccountFields(id, userID string, accountType FinancialAccountType, name string, openingBalanceCents, currentBalanceCents int64, creditLimitCents *int64, cutoffDay, paymentDay *int) (struct{ id, userID, name string }, error) {
	fields := struct{ id, userID, name string }{id: strings.TrimSpace(id), userID: strings.TrimSpace(userID), name: strings.TrimSpace(name)}
	if fields.id == "" {
		return fields, ErrInvalidFinancialAccountID
	}
	if fields.userID == "" {
		return fields, ErrInvalidFinancialAccountUserID
	}
	if !accountType.IsValid() {
		return fields, ErrInvalidFinancialAccountType
	}
	if fields.name == "" {
		return fields, ErrInvalidFinancialAccountName
	}
	if accountType != FinancialAccountTypeCreditCard && (openingBalanceCents < 0 || currentBalanceCents < 0) {
		return fields, ErrInvalidFinancialAccountBalance
	}
	if accountType == FinancialAccountTypeCreditCard {
		if creditLimitCents == nil || *creditLimitCents <= 0 {
			return fields, ErrInvalidFinancialAccountCreditLimit
		}
		if cutoffDay == nil || *cutoffDay < 1 || *cutoffDay > 31 {
			return fields, ErrInvalidFinancialAccountCutoffDay
		}
		if paymentDay == nil || *paymentDay < 1 || *paymentDay > 31 {
			return fields, ErrInvalidFinancialAccountPaymentDay
		}
	}
	return fields, nil
}

func normalizeOptionalFinancialString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeOptionalInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	normalized := *value
	return &normalized
}

func normalizeOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	normalized := *value
	return &normalized
}
