package domain

import (
	"errors"
	"strings"
	"time"
)

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "INCOME"
	TransactionTypeExpense TransactionType = "EXPENSE"
)

func (transactionType TransactionType) IsValid() bool {
	return transactionType == TransactionTypeIncome ||
		transactionType == TransactionTypeExpense
}

type TransactionStatus string

const (
	TransactionStatusPaid    TransactionStatus = "PAID"
	TransactionStatusPending TransactionStatus = "PENDING"
)

func (status TransactionStatus) IsValid() bool {
	return status == TransactionStatusPaid ||
		status == TransactionStatusPending
}

var (
	ErrEmptyTransactionID          = errors.New("domain: transaction ID cannot be empty")
	ErrEmptyTransactionUserID      = errors.New("domain: transaction user ID cannot be empty")
	ErrInvalidTransactionType      = errors.New("domain: invalid transaction type")
	ErrEmptyTransactionConcept     = errors.New("domain: transaction concept cannot be empty")
	ErrEmptyTransactionCategory    = errors.New("domain: transaction category cannot be empty")
	ErrInvalidTransactionAmount    = errors.New("domain: transaction amount must be greater than zero")
	ErrInvalidTransactionDate      = errors.New("domain: transaction date cannot be zero")
	ErrInvalidTransactionStatus    = errors.New("domain: invalid transaction status")
	ErrInvalidTransactionMSI       = errors.New("domain: transaction MSI must be at least one")
	ErrInvalidTransactionCreatedAt = errors.New("domain: transaction created at cannot be zero")
	ErrInvalidTransactionUpdatedAt = errors.New("domain: transaction updated at cannot be zero")
)

// Transaction is the aggregate root for financial ledger entries.
type Transaction struct {
	id              string
	userID          string
	transactionType TransactionType
	concept         string
	category        string
	amountCents     int64
	date            time.Time
	status          TransactionStatus
	msi             *int
	creditCardID    *string
	createdAt       time.Time
	updatedAt       time.Time
}

func NewTransaction(
	id,
	userID string,
	transactionType TransactionType,
	concept,
	category string,
	amountCents int64,
	date time.Time,
	status TransactionStatus,
	msi *int,
	creditCardID *string,
) (*Transaction, error) {
	transactionFields, err := validateTransactionFields(id, userID, transactionType, concept, category, amountCents, date, status, msi, creditCardID)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	return &Transaction{
		id:              transactionFields.id,
		userID:          transactionFields.userID,
		transactionType: transactionType,
		concept:         transactionFields.concept,
		category:        transactionFields.category,
		amountCents:     amountCents,
		date:            date,
		status:          status,
		msi:             normalizeOptionalTransactionMSI(msi),
		creditCardID:    transactionFields.creditCardID,
		createdAt:       currentTime,
		updatedAt:       currentTime,
	}, nil
}

func RehydrateTransaction(
	id,
	userID string,
	transactionType TransactionType,
	concept,
	category string,
	amountCents int64,
	date time.Time,
	status TransactionStatus,
	msi *int,
	creditCardID *string,
	createdAt,
	updatedAt time.Time,
) (*Transaction, error) {
	transactionFields, err := validateTransactionFields(id, userID, transactionType, concept, category, amountCents, date, status, msi, creditCardID)
	if err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidTransactionCreatedAt
	}
	if updatedAt.IsZero() {
		return nil, ErrInvalidTransactionUpdatedAt
	}

	return &Transaction{
		id:              transactionFields.id,
		userID:          transactionFields.userID,
		transactionType: transactionType,
		concept:         transactionFields.concept,
		category:        transactionFields.category,
		amountCents:     amountCents,
		date:            date,
		status:          status,
		msi:             normalizeOptionalTransactionMSI(msi),
		creditCardID:    transactionFields.creditCardID,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}, nil
}

func (transaction *Transaction) Update(
	transactionType TransactionType,
	concept,
	category string,
	amountCents int64,
	date time.Time,
	status TransactionStatus,
	msi *int,
	creditCardID *string,
) error {
	transactionFields, err := validateTransactionFields(transaction.id, transaction.userID, transactionType, concept, category, amountCents, date, status, msi, creditCardID)
	if err != nil {
		return err
	}

	transaction.transactionType = transactionType
	transaction.concept = transactionFields.concept
	transaction.category = transactionFields.category
	transaction.amountCents = amountCents
	transaction.date = date
	transaction.status = status
	transaction.msi = normalizeOptionalTransactionMSI(msi)
	transaction.creditCardID = transactionFields.creditCardID
	transaction.touch()
	return nil
}

func (transaction *Transaction) ID() string {
	return transaction.id
}

func (transaction *Transaction) UserID() string {
	return transaction.userID
}

func (transaction *Transaction) Type() TransactionType {
	return transaction.transactionType
}

func (transaction *Transaction) Concept() string {
	return transaction.concept
}

func (transaction *Transaction) Category() string {
	return transaction.category
}

func (transaction *Transaction) AmountCents() int64 {
	return transaction.amountCents
}

func (transaction *Transaction) Date() time.Time {
	return transaction.date
}

func (transaction *Transaction) Status() TransactionStatus {
	return transaction.status
}

func (transaction *Transaction) MSI() *int {
	return normalizeOptionalTransactionMSI(transaction.msi)
}

func (transaction *Transaction) CreditCardID() *string {
	return normalizeOptionalTransactionCreditCardID(transaction.creditCardID)
}

func (transaction *Transaction) CreatedAt() time.Time {
	return transaction.createdAt
}

func (transaction *Transaction) UpdatedAt() time.Time {
	return transaction.updatedAt
}

func (transaction *Transaction) touch() {
	transaction.updatedAt = time.Now()
}

func validateTransactionFields(
	id,
	userID string,
	transactionType TransactionType,
	concept,
	category string,
	amountCents int64,
	date time.Time,
	status TransactionStatus,
	msi *int,
	creditCardID *string,
) (validatedTransactionFields, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return validatedTransactionFields{}, ErrEmptyTransactionID
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return validatedTransactionFields{}, ErrEmptyTransactionUserID
	}

	if !transactionType.IsValid() {
		return validatedTransactionFields{}, ErrInvalidTransactionType
	}

	trimmedConcept := strings.TrimSpace(concept)
	if trimmedConcept == "" {
		return validatedTransactionFields{}, ErrEmptyTransactionConcept
	}

	trimmedCategory := strings.TrimSpace(category)
	if trimmedCategory == "" {
		return validatedTransactionFields{}, ErrEmptyTransactionCategory
	}

	if amountCents <= 0 {
		return validatedTransactionFields{}, ErrInvalidTransactionAmount
	}

	if date.IsZero() {
		return validatedTransactionFields{}, ErrInvalidTransactionDate
	}

	if !status.IsValid() {
		return validatedTransactionFields{}, ErrInvalidTransactionStatus
	}

	if msi != nil && *msi < 1 {
		return validatedTransactionFields{}, ErrInvalidTransactionMSI
	}

	return validatedTransactionFields{
		id:           trimmedID,
		userID:       trimmedUserID,
		concept:      trimmedConcept,
		category:     trimmedCategory,
		creditCardID: normalizeOptionalTransactionCreditCardID(creditCardID),
	}, nil
}

type validatedTransactionFields struct {
	id           string
	userID       string
	concept      string
	category     string
	creditCardID *string
}

func normalizeOptionalTransactionMSI(msi *int) *int {
	if msi == nil {
		return nil
	}

	normalizedMSI := *msi
	return &normalizedMSI
}

func normalizeOptionalTransactionCreditCardID(creditCardID *string) *string {
	if creditCardID == nil {
		return nil
	}

	trimmedCreditCardID := strings.TrimSpace(*creditCardID)
	if trimmedCreditCardID == "" {
		return nil
	}

	return &trimmedCreditCardID
}
