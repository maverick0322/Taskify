package domain

import (
	"errors"
	"strings"
	"time"
)

type TransactionType string

const (
	TransactionTypeIncome      TransactionType = "INCOME"
	TransactionTypeExpense     TransactionType = "EXPENSE"
	TransactionTypeDebtPayment TransactionType = "DEBT_PAYMENT"
	TransactionTypeTransfer    TransactionType = "TRANSFER"
)

func (transactionType TransactionType) IsValid() bool {
	return transactionType == TransactionTypeIncome ||
		transactionType == TransactionTypeExpense ||
		transactionType == TransactionTypeDebtPayment ||
		transactionType == TransactionTypeTransfer
}

type TransactionStatus string

const (
	TransactionStatusPaid      TransactionStatus = "PAID"
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
)

func (status TransactionStatus) IsValid() bool {
	return status == TransactionStatusPaid ||
		status == TransactionStatusPending ||
		status == TransactionStatusCompleted
}

type TransactionRecurrence string

const (
	TransactionRecurrenceOnce      TransactionRecurrence = "once"
	TransactionRecurrenceMonthly   TransactionRecurrence = "monthly"
	TransactionRecurrenceQuarterly TransactionRecurrence = "quarterly"
	TransactionRecurrenceBiannual  TransactionRecurrence = "biannual"
	TransactionRecurrenceAnnual    TransactionRecurrence = "annual"
)

func (recurrence TransactionRecurrence) IsValid() bool {
	return recurrence == TransactionRecurrenceOnce ||
		recurrence == TransactionRecurrenceMonthly ||
		recurrence == TransactionRecurrenceQuarterly ||
		recurrence == TransactionRecurrenceBiannual ||
		recurrence == TransactionRecurrenceAnnual
}

func (recurrence TransactionRecurrence) IsRecurring() bool {
	return recurrence != TransactionRecurrenceOnce
}

func (recurrence TransactionRecurrence) NextDate(date time.Time) time.Time {
	switch recurrence {
	case TransactionRecurrenceMonthly:
		return date.AddDate(0, 1, 0)
	case TransactionRecurrenceQuarterly:
		return date.AddDate(0, 3, 0)
	case TransactionRecurrenceBiannual:
		return date.AddDate(0, 6, 0)
	case TransactionRecurrenceAnnual:
		return date.AddDate(1, 0, 0)
	default:
		return date
	}
}

var (
	ErrEmptyTransactionID                = errors.New("domain: transaction ID cannot be empty")
	ErrEmptyTransactionUserID            = errors.New("domain: transaction user ID cannot be empty")
	ErrInvalidTransactionType            = errors.New("domain: invalid transaction type")
	ErrEmptyTransactionConcept           = errors.New("domain: transaction concept cannot be empty")
	ErrEmptyTransactionCategory          = errors.New("domain: transaction category cannot be empty")
	ErrInvalidTransactionAmount          = errors.New("domain: transaction amount must be greater than zero")
	ErrInvalidTransactionDate            = errors.New("domain: transaction date cannot be zero")
	ErrInvalidTransactionStatus          = errors.New("domain: invalid transaction status")
	ErrInvalidTransactionMSI             = errors.New("domain: transaction MSI must be at least one")
	ErrInvalidTransactionRecurrence      = errors.New("domain: invalid transaction recurrence")
	ErrInvalidTransactionRecurrenceLimit = errors.New("domain: transaction recurrence limit cannot be negative")
	ErrInvalidTransactionCreatedAt       = errors.New("domain: transaction created at cannot be zero")
	ErrInvalidTransactionUpdatedAt       = errors.New("domain: transaction updated at cannot be zero")
)

// Transaction is the aggregate root for financial ledger entries.
type Transaction struct {
	id                   string
	userID               string
	transactionType      TransactionType
	concept              string
	category             string
	amountCents          int64
	date                 time.Time
	status               TransactionStatus
	msi                  *int
	creditCardID         *string
	paymentAccountID     *string
	destinationAccountID *string
	installmentNumber    *int
	installmentCount     *int
	recurrence           TransactionRecurrence
	recurrenceLimit      *int
	lastPaidAt           *time.Time
	paidCycles           []PaidCycle
	createdAt            time.Time
	updatedAt            time.Time
}

type PaidCycle struct {
	DueDate time.Time
	PaidAt  time.Time
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
	recurrence TransactionRecurrence,
	recurrenceLimit *int,
) (*Transaction, error) {
	transactionFields, err := validateTransactionFields(id, userID, transactionType, concept, category, amountCents, date, status, msi, creditCardID, recurrence, recurrenceLimit)
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
		recurrence:      recurrence,
		recurrenceLimit: normalizeOptionalTransactionRecurrenceLimit(recurrence, recurrenceLimit),
		lastPaidAt:      nil,
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
	recurrence TransactionRecurrence,
	recurrenceLimit *int,
	lastPaidAt *time.Time,
	createdAt,
	updatedAt time.Time,
) (*Transaction, error) {
	transactionFields, err := validateTransactionFields(id, userID, transactionType, concept, category, amountCents, date, status, msi, creditCardID, recurrence, recurrenceLimit)
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
		recurrence:      recurrence,
		recurrenceLimit: normalizeOptionalTransactionRecurrenceLimit(recurrence, recurrenceLimit),
		lastPaidAt:      normalizeOptionalTransactionTime(lastPaidAt),
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
	recurrence TransactionRecurrence,
	recurrenceLimit *int,
) error {
	transactionFields, err := validateTransactionFields(transaction.id, transaction.userID, transactionType, concept, category, amountCents, date, status, msi, creditCardID, recurrence, recurrenceLimit)
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
	transaction.recurrence = recurrence
	transaction.recurrenceLimit = normalizeOptionalTransactionRecurrenceLimit(recurrence, recurrenceLimit)
	transaction.touch()
	return nil
}

func (transaction *Transaction) SetAccountingDetails(paymentAccountID, destinationAccountID *string, installmentNumber, installmentCount *int) {
	transaction.paymentAccountID = normalizeOptionalTransactionCreditCardID(paymentAccountID)
	transaction.destinationAccountID = normalizeOptionalTransactionCreditCardID(destinationAccountID)
	transaction.installmentNumber = normalizeOptionalTransactionMSI(installmentNumber)
	transaction.installmentCount = normalizeOptionalTransactionMSI(installmentCount)
	transaction.touch()
}

func (transaction *Transaction) SetCreditCardID(creditCardID *string) {
	transaction.creditCardID = normalizeOptionalTransactionCreditCardID(creditCardID)
	transaction.touch()
}

func (transaction *Transaction) Complete() {
	completedLimit := 0
	transaction.status = TransactionStatusCompleted
	transaction.recurrenceLimit = &completedLimit
	transaction.touch()
}

func (transaction *Transaction) AdvanceRecurrence() {
	if !transaction.recurrence.IsRecurring() {
		transaction.Complete()
		return
	}

	transaction.date = transaction.recurrence.NextDate(transaction.date)
	if transaction.recurrenceLimit != nil {
		nextLimit := *transaction.recurrenceLimit - 1
		transaction.recurrenceLimit = &nextLimit
		if nextLimit <= 0 {
			transaction.status = TransactionStatusCompleted
		}
	}
	transaction.touch()
}

func (transaction *Transaction) MarkPaidAt(paidAt time.Time) {
	transaction.lastPaidAt = normalizeOptionalTransactionTime(&paidAt)
	transaction.touch()
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

func (transaction *Transaction) PaymentAccountID() *string {
	return normalizeOptionalTransactionCreditCardID(transaction.paymentAccountID)
}

func (transaction *Transaction) DestinationAccountID() *string {
	return normalizeOptionalTransactionCreditCardID(transaction.destinationAccountID)
}

func (transaction *Transaction) InstallmentNumber() *int {
	return normalizeOptionalTransactionMSI(transaction.installmentNumber)
}

func (transaction *Transaction) InstallmentCount() *int {
	return normalizeOptionalTransactionMSI(transaction.installmentCount)
}

func (transaction *Transaction) Recurrence() TransactionRecurrence {
	return transaction.recurrence
}

func (transaction *Transaction) RecurrenceLimit() *int {
	return normalizeOptionalTransactionRecurrenceLimit(transaction.recurrence, transaction.recurrenceLimit)
}

func (transaction *Transaction) LastPaidAt() *time.Time {
	return normalizeOptionalTransactionTime(transaction.lastPaidAt)
}

func (transaction *Transaction) PaidCycles() []PaidCycle {
	paidCycles := make([]PaidCycle, len(transaction.paidCycles))
	copy(paidCycles, transaction.paidCycles)
	return paidCycles
}

func (transaction *Transaction) SetPaidCycles(paidCycles []PaidCycle) {
	transaction.paidCycles = make([]PaidCycle, len(paidCycles))
	copy(transaction.paidCycles, paidCycles)
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
	recurrence TransactionRecurrence,
	recurrenceLimit *int,
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

	if !recurrence.IsValid() {
		return validatedTransactionFields{}, ErrInvalidTransactionRecurrence
	}

	if recurrenceLimit != nil && *recurrenceLimit < 0 {
		return validatedTransactionFields{}, ErrInvalidTransactionRecurrenceLimit
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

func normalizeOptionalTransactionRecurrenceLimit(recurrence TransactionRecurrence, recurrenceLimit *int) *int {
	if recurrenceLimit == nil {
		return nil
	}

	if *recurrenceLimit == 0 {
		normalizedLimit := 0
		return &normalizedLimit
	}

	if recurrence == TransactionRecurrenceOnce {
		return nil
	}

	normalizedLimit := *recurrenceLimit
	return &normalizedLimit
}

func normalizeOptionalTransactionTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	normalizedValue := value.UTC()
	return &normalizedValue
}
