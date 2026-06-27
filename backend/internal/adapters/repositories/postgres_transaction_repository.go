package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	createTransactionQuery = `
		INSERT INTO transactions (id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	getTransactionByIDQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE id = $1 AND deleted_at IS NULL
	`

	getTransactionsByUserIDQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDFromQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND date >= $2 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDToQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND date < $2 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDRangeQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND date >= $2 AND date < $3 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDAndCreditCardIDQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND credit_card_id = $2 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDAndCreditCardIDFromQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND credit_card_id = $2 AND date >= $3 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDAndCreditCardIDToQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND credit_card_id = $2 AND date < $3 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDAndCreditCardIDRangeQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND credit_card_id = $2 AND date >= $3 AND date < $4 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDAndPaymentAccountIDQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND payment_account_id = $2 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDAndPaymentAccountIDFromQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND payment_account_id = $2 AND date >= $3 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDAndPaymentAccountIDToQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND payment_account_id = $2 AND date < $3 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	getTransactionsByUserIDAndPaymentAccountIDRangeQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND payment_account_id = $2 AND date >= $3 AND date < $4 AND deleted_at IS NULL
		ORDER BY date DESC
	`

	updateTransactionQuery = `
		UPDATE transactions
		SET type = $2,
			concept = $3,
			category = $4,
			amount_cents = $5,
			date = $6,
			status = $7,
			msi = $8,
			credit_card_id = $9,
			payment_account_id = $10,
			destination_account_id = $11,
			installment_number = $12,
			installment_count = $13,
			is_historical = $14,
			recurrence = $15,
			recurrence_limit = $16,
			last_paid_at = $17,
			updated_at = $18
		WHERE id = $1 AND deleted_at IS NULL AND updated_at < $18
	`

	deleteTransactionQuery = `
		UPDATE transactions
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL AND updated_at < $2
	`

	createAccountPayablePaymentQuery = `
		INSERT INTO account_payable_payments (id, account_payable_id, user_id, due_date, paid_at, amount_cents, concept, category, created_transaction_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	getPaidCyclesByUserIDQuery = `
		SELECT account_payable_id, due_date, paid_at
		FROM account_payable_payments
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY due_date ASC
	`

	getPaidCyclesByAccountPayableIDQuery = `
		SELECT due_date, paid_at
		FROM account_payable_payments
		WHERE account_payable_id = $1 AND deleted_at IS NULL
		ORDER BY due_date ASC
	`
)

type postgresTransactionDatabase interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

type postgresTransactionBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PostgresTransactionRepository implements transaction persistence using PostgreSQL.
type PostgresTransactionRepository struct {
	database postgresTransactionDatabase
	logger   ports.Logger
}

func NewPostgresTransactionRepository(pool *pgxpool.Pool, logger ports.Logger) ports.TransactionRepository {
	return &PostgresTransactionRepository{
		database: pool,
		logger:   logger,
	}
}

func (repository *PostgresTransactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	if transaction == nil {
		repository.logger.Error("cannot create nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	_, err := execPostgresCreateTransaction(ctx, repository.database, transaction)
	if err == nil {
		return nil
	}

	return repository.mapWriteError(err, "failed to create transaction", "userID", transaction.UserID(), "transactionID", transaction.ID())
}

func (repository *PostgresTransactionRepository) GetByID(ctx context.Context, id string) (*domain.Transaction, error) {
	transaction, err := repository.scanTransaction(repository.database.QueryRow(ctx, getTransactionByIDQuery, id))
	if err == nil {
		return transaction, nil
	}

	return repository.mapReadError(err, "failed to retrieve transaction by id", "transactionID", id)
}

func (repository *PostgresTransactionRepository) GetByUserID(ctx context.Context, userID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	query, arguments := buildTransactionsByUserIDQuery(userID, filter)
	return repository.queryTransactions(ctx, query, arguments, "failed to retrieve transactions by user id", "userID", userID)
}

func (repository *PostgresTransactionRepository) queryTransactions(ctx context.Context, query string, arguments []interface{}, message string, keysAndValues ...interface{}) ([]*domain.Transaction, error) {
	rows, err := repository.database.Query(ctx, query, arguments...)
	if err != nil {
		repository.logger.Error(message, append(keysAndValues, "error", err)...)
		return nil, ports.ErrTransactionRepositoryUnavailable
	}
	defer rows.Close()

	transactions := make([]*domain.Transaction, 0)
	for rows.Next() {
		transaction, err := repository.scanTransaction(rows)
		if err != nil {
			repository.logger.Error("failed to scan transaction row", append(keysAndValues, "error", err)...)
			return nil, ports.ErrTransactionRepositoryUnavailable
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		repository.logger.Error("failed while iterating transaction rows", append(keysAndValues, "error", err)...)
		return nil, ports.ErrTransactionRepositoryUnavailable
	}

	return transactions, nil
}

func (repository *PostgresTransactionRepository) GetByCreditCardID(ctx context.Context, userID, creditCardID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	query, arguments := buildTransactionsByUserIDAndCreditCardIDQuery(userID, creditCardID, filter)
	return repository.queryTransactions(ctx, query, arguments, "failed to retrieve transactions by credit card id", "userID", userID, "creditCardID", creditCardID)
}

func (repository *PostgresTransactionRepository) GetByPaymentAccountID(ctx context.Context, userID, paymentAccountID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	query, arguments := buildTransactionsByUserIDAndPaymentAccountIDQuery(userID, paymentAccountID, filter)
	return repository.queryTransactions(ctx, query, arguments, "failed to retrieve transactions by payment account id", "userID", userID, "paymentAccountID", paymentAccountID)
}

func (repository *PostgresTransactionRepository) GetPaidCyclesByUserID(ctx context.Context, userID string) (map[string][]domain.PaidCycle, error) {
	rows, err := repository.database.Query(ctx, getPaidCyclesByUserIDQuery, userID)
	if err != nil {
		return nil, ports.ErrTransactionRepositoryUnavailable
	}
	defer rows.Close()

	paidCycles := make(map[string][]domain.PaidCycle)
	for rows.Next() {
		var accountPayableID string
		var dueDate time.Time
		var paidAt time.Time
		if err := rows.Scan(&accountPayableID, &dueDate, &paidAt); err != nil {
			return nil, ports.ErrTransactionRepositoryUnavailable
		}
		paidCycles[accountPayableID] = append(paidCycles[accountPayableID], domain.PaidCycle{DueDate: dueDate, PaidAt: paidAt})
	}
	if err := rows.Err(); err != nil {
		return nil, ports.ErrTransactionRepositoryUnavailable
	}

	return paidCycles, nil
}

func (repository *PostgresTransactionRepository) GetPaidCyclesByAccountPayableID(ctx context.Context, accountPayableID string) ([]domain.PaidCycle, error) {
	rows, err := repository.database.Query(ctx, getPaidCyclesByAccountPayableIDQuery, accountPayableID)
	if err != nil {
		return nil, ports.ErrTransactionRepositoryUnavailable
	}
	defer rows.Close()

	paidCycles := make([]domain.PaidCycle, 0)
	for rows.Next() {
		var dueDate time.Time
		var paidAt time.Time
		if err := rows.Scan(&dueDate, &paidAt); err != nil {
			return nil, ports.ErrTransactionRepositoryUnavailable
		}
		paidCycles = append(paidCycles, domain.PaidCycle{DueDate: dueDate, PaidAt: paidAt})
	}
	if err := rows.Err(); err != nil {
		return nil, ports.ErrTransactionRepositoryUnavailable
	}

	return paidCycles, nil
}

func (repository *PostgresTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	if transaction == nil {
		repository.logger.Error("cannot update nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := execPostgresUpdateTransaction(ctx, repository.database, transaction); err != nil {
		repository.logger.Error("failed to update transaction", "userID", transaction.UserID(), "transactionID", transaction.ID(), "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresTransactionRepository) CreateAndUpdate(ctx context.Context, transactionToCreate, transactionToUpdate *domain.Transaction) error {
	if transactionToCreate == nil || transactionToUpdate == nil {
		repository.logger.Error("cannot create and update nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	beginner, ok := repository.database.(postgresTransactionBeginner)
	if !ok {
		repository.logger.Error("postgres transaction database cannot start transactions")
		return ports.ErrTransactionRepositoryUnavailable
	}

	tx, err := beginner.Begin(ctx)
	if err != nil {
		repository.logger.Error("failed to start transaction payment unit", "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}
	defer tx.Rollback(ctx)

	if _, err := execPostgresCreateTransaction(ctx, tx, transactionToCreate); err != nil {
		repository.logger.Error("failed to create payable payment transaction", "userID", transactionToCreate.UserID(), "transactionID", transactionToCreate.ID(), "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := execPostgresUpdateTransaction(ctx, tx, transactionToUpdate); err != nil {
		repository.logger.Error("failed to update paid account payable", "userID", transactionToUpdate.UserID(), "transactionID", transactionToUpdate.ID(), "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	if err := tx.Commit(ctx); err != nil {
		repository.logger.Error("failed to commit payable payment unit", "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresTransactionRepository) CreatePaymentCycleAndUpdate(ctx context.Context, transactionToCreate, transactionToUpdate *domain.Transaction, cyclePayment ports.AccountPayableCyclePayment) error {
	if transactionToCreate == nil || transactionToUpdate == nil {
		return ports.ErrTransactionRepositoryUnavailable
	}
	beginner, ok := repository.database.(postgresTransactionBeginner)
	if !ok {
		return ports.ErrTransactionRepositoryUnavailable
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}
	defer tx.Rollback(ctx)

	if _, err := execPostgresCreateTransaction(ctx, tx, transactionToCreate); err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := tx.Exec(ctx, createAccountPayablePaymentQuery, cyclePayment.ID, cyclePayment.AccountPayableID, cyclePayment.UserID, cyclePayment.DueDate, cyclePayment.PaidAt, cyclePayment.AmountCents, cyclePayment.Concept, cyclePayment.Category, cyclePayment.CreatedTransactionID, cyclePayment.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return ports.ErrAccountPayableCycleAlreadyPaid
		}
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := execPostgresUpdateTransaction(ctx, tx, transactionToUpdate); err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}

	if err := tx.Commit(ctx); err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}
	return nil
}

func (repository *PostgresTransactionRepository) CreateManyWithLedger(ctx context.Context, transactions []*domain.Transaction, accountDeltas []ports.AccountBalanceDelta, ledgerEntries []ports.LedgerEntry) error {
	beginner, ok := repository.database.(postgresTransactionBeginner)
	if !ok {
		return ports.ErrTransactionRepositoryUnavailable
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}
	defer tx.Rollback(ctx)
	for _, transaction := range transactions {
		if transaction == nil {
			return ports.ErrTransactionRepositoryUnavailable
		}
		if _, err := execPostgresCreateTransaction(ctx, tx, transaction); err != nil {
			return ports.ErrTransactionRepositoryUnavailable
		}
	}
	now := time.Now().UTC()
	for _, delta := range accountDeltas {
		if _, err := tx.Exec(ctx, "UPDATE financial_accounts SET current_balance_cents = current_balance_cents + $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL", delta.DeltaCents, now, delta.AccountID); err != nil {
			return ports.ErrTransactionRepositoryUnavailable
		}
	}
	for _, entry := range ledgerEntries {
		if _, err := tx.Exec(ctx, "INSERT INTO ledger_entries (id, user_id, account_id, transaction_id, amount_cents, entry_type, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", entry.ID, entry.UserID, entry.AccountID, entry.TransactionID, entry.AmountCents, entry.EntryType, entry.CreatedAt, entry.UpdatedAt); err != nil {
			return ports.ErrTransactionRepositoryUnavailable
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}
	return nil
}

func (repository *PostgresTransactionRepository) Delete(ctx context.Context, id string) error {
	deletedAt := time.Now().UTC()
	if _, err := repository.database.Exec(ctx, deleteTransactionQuery, id, deletedAt); err != nil {
		repository.logger.Error("failed to delete transaction", "transactionID", id, "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func execPostgresCreateTransaction(ctx context.Context, executor postgresTransactionDatabase, transaction *domain.Transaction) (pgconn.CommandTag, error) {
	return executor.Exec(ctx, createTransactionQuery, transaction.ID(), transaction.UserID(), nullableTransactionCreditCardID(transaction.CreditCardID()), nullableTransactionCreditCardID(transaction.PaymentAccountID()), nullableTransactionCreditCardID(transaction.DestinationAccountID()), string(transaction.Type()), transaction.Concept(), transaction.Category(), transaction.AmountCents(), transaction.Date(), string(transaction.Status()), nullableTransactionMSI(transaction.MSI()), nullableTransactionMSI(transaction.InstallmentNumber()), nullableTransactionMSI(transaction.InstallmentCount()), transaction.IsHistorical(), string(transaction.Recurrence()), nullableTransactionRecurrenceLimit(transaction.RecurrenceLimit()), nullableTransactionTime(transaction.LastPaidAt()), transaction.CreatedAt(), transaction.UpdatedAt())
}

func execPostgresUpdateTransaction(ctx context.Context, executor postgresTransactionDatabase, transaction *domain.Transaction) (pgconn.CommandTag, error) {
	return executor.Exec(ctx, updateTransactionQuery, transaction.ID(), string(transaction.Type()), transaction.Concept(), transaction.Category(), transaction.AmountCents(), transaction.Date(), string(transaction.Status()), nullableTransactionMSI(transaction.MSI()), nullableTransactionCreditCardID(transaction.CreditCardID()), nullableTransactionCreditCardID(transaction.PaymentAccountID()), nullableTransactionCreditCardID(transaction.DestinationAccountID()), nullableTransactionMSI(transaction.InstallmentNumber()), nullableTransactionMSI(transaction.InstallmentCount()), transaction.IsHistorical(), string(transaction.Recurrence()), nullableTransactionRecurrenceLimit(transaction.RecurrenceLimit()), nullableTransactionTime(transaction.LastPaidAt()), transaction.UpdatedAt())
}

func (repository *PostgresTransactionRepository) scanTransaction(row pgx.Row) (*domain.Transaction, error) {
	var storedTransaction storedTransaction
	if err := row.Scan(
		&storedTransaction.id,
		&storedTransaction.userID,
		&storedTransaction.creditCardID,
		&storedTransaction.paymentAccountID,
		&storedTransaction.destinationAccountID,
		&storedTransaction.transactionType,
		&storedTransaction.concept,
		&storedTransaction.category,
		&storedTransaction.amountCents,
		&storedTransaction.date,
		&storedTransaction.status,
		&storedTransaction.msi,
		&storedTransaction.installmentNumber,
		&storedTransaction.installmentCount,
		&storedTransaction.isHistorical,
		&storedTransaction.recurrence,
		&storedTransaction.recurrenceLimit,
		&storedTransaction.lastPaidAt,
		&storedTransaction.createdAt,
		&storedTransaction.updatedAt,
	); err != nil {
		return nil, err
	}

	transaction, err := domain.RehydrateTransaction(
		storedTransaction.id,
		storedTransaction.userID,
		domain.TransactionType(storedTransaction.transactionType),
		storedTransaction.concept,
		storedTransaction.category,
		storedTransaction.amountCents,
		storedTransaction.date,
		domain.TransactionStatus(storedTransaction.status),
		storedTransaction.msi,
		storedTransaction.creditCardID,
		domain.TransactionRecurrence(storedTransaction.recurrence),
		storedTransaction.recurrenceLimit,
		storedTransaction.lastPaidAt,
		storedTransaction.createdAt,
		storedTransaction.updatedAt,
		storedTransaction.isHistorical,
	)
	if err != nil {
		return nil, err
	}
	transaction.SetAccountingDetails(storedTransaction.paymentAccountID, storedTransaction.destinationAccountID, storedTransaction.installmentNumber, storedTransaction.installmentCount)
	return transaction, nil
}

func (repository *PostgresTransactionRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.Transaction, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrTransactionNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrTransactionRepositoryUnavailable
}

func (repository *PostgresTransactionRepository) mapWriteError(err error, message string, keysAndValues ...interface{}) error {
	if isUniqueViolation(err) {
		repository.logger.Warn("transaction already exists")
		return ports.ErrTransactionAlreadyExists
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return ports.ErrTransactionRepositoryUnavailable
}

type storedTransaction struct {
	id                   string
	userID               string
	creditCardID         *string
	paymentAccountID     *string
	destinationAccountID *string
	transactionType      string
	concept              string
	category             string
	amountCents          int64
	date                 time.Time
	status               string
	msi                  *int
	installmentNumber    *int
	installmentCount     *int
	isHistorical         bool
	recurrence           string
	recurrenceLimit      *int
	lastPaidAt           *time.Time
	createdAt            time.Time
	updatedAt            time.Time
}

func buildTransactionsByUserIDQuery(userID string, filter ports.TransactionDateFilter) (string, []interface{}) {
	if filter.From != nil && filter.To != nil {
		return getTransactionsByUserIDRangeQuery, []interface{}{userID, *filter.From, *filter.To}
	}
	if filter.From != nil {
		return getTransactionsByUserIDFromQuery, []interface{}{userID, *filter.From}
	}
	if filter.To != nil {
		return getTransactionsByUserIDToQuery, []interface{}{userID, *filter.To}
	}

	return getTransactionsByUserIDQuery, []interface{}{userID}
}

func buildTransactionsByUserIDAndCreditCardIDQuery(userID, creditCardID string, filter ports.TransactionDateFilter) (string, []interface{}) {
	if filter.From != nil && filter.To != nil {
		return getTransactionsByUserIDAndCreditCardIDRangeQuery, []interface{}{userID, creditCardID, *filter.From, *filter.To}
	}
	if filter.From != nil {
		return getTransactionsByUserIDAndCreditCardIDFromQuery, []interface{}{userID, creditCardID, *filter.From}
	}
	if filter.To != nil {
		return getTransactionsByUserIDAndCreditCardIDToQuery, []interface{}{userID, creditCardID, *filter.To}
	}

	return getTransactionsByUserIDAndCreditCardIDQuery, []interface{}{userID, creditCardID}
}

func buildTransactionsByUserIDAndPaymentAccountIDQuery(userID, paymentAccountID string, filter ports.TransactionDateFilter) (string, []interface{}) {
	if filter.From != nil && filter.To != nil {
		return getTransactionsByUserIDAndPaymentAccountIDRangeQuery, []interface{}{userID, paymentAccountID, *filter.From, *filter.To}
	}
	if filter.From != nil {
		return getTransactionsByUserIDAndPaymentAccountIDFromQuery, []interface{}{userID, paymentAccountID, *filter.From}
	}
	if filter.To != nil {
		return getTransactionsByUserIDAndPaymentAccountIDToQuery, []interface{}{userID, paymentAccountID, *filter.To}
	}

	return getTransactionsByUserIDAndPaymentAccountIDQuery, []interface{}{userID, paymentAccountID}
}

func nullableTransactionMSI(msi *int) interface{} {
	if msi == nil {
		return nil
	}

	return *msi
}

func nullableTransactionCreditCardID(creditCardID *string) interface{} {
	if creditCardID == nil {
		return nil
	}

	return *creditCardID
}

func nullableTransactionRecurrenceLimit(recurrenceLimit *int) interface{} {
	if recurrenceLimit == nil {
		return nil
	}

	return *recurrenceLimit
}

func nullableTransactionTime(value *time.Time) interface{} {
	if value == nil || value.IsZero() {
		return nil
	}

	return value.UTC()
}
