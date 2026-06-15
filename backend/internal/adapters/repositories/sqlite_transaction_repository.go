package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	sqliteCreateTransactionQuery = `
		INSERT INTO transactions (id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	sqliteGetTransactionByIDQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE id = ? AND deleted_at IS NULL
	`

	sqliteGetTransactionsByUserIDQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDFromQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND date >= ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDToQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDRangeQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND date >= ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndCreditCardIDQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND credit_card_id = ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndCreditCardIDFromQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND credit_card_id = ? AND date >= ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndCreditCardIDToQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND credit_card_id = ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndCreditCardIDRangeQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND credit_card_id = ? AND date >= ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndPaymentAccountIDQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND payment_account_id = ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndPaymentAccountIDFromQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND payment_account_id = ? AND date >= ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndPaymentAccountIDToQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND payment_account_id = ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndPaymentAccountIDRangeQuery = `
		SELECT id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, recurrence, recurrence_limit, last_paid_at, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND payment_account_id = ? AND date >= ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteUpdateTransactionQuery = `
		UPDATE transactions
		SET type = ?,
			concept = ?,
			category = ?,
			amount_cents = ?,
			date = ?,
			status = ?,
			msi = ?,
			credit_card_id = ?,
			payment_account_id = ?,
			destination_account_id = ?,
			installment_number = ?,
			installment_count = ?,
			recurrence = ?,
			recurrence_limit = ?,
			last_paid_at = ?,
			updated_at = ?
		WHERE id = ?
	`

	sqliteDeleteTransactionQuery = `
		UPDATE transactions
		SET deleted_at = ?, updated_at = ?
		WHERE id = ?
	`

	sqliteCreateAccountPayablePaymentQuery = `
		INSERT INTO account_payable_payments (id, account_payable_id, user_id, due_date, paid_at, amount_cents, concept, category, created_transaction_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	sqliteGetPaidCyclesByUserIDQuery = `
		SELECT account_payable_id, due_date, paid_at
		FROM account_payable_payments
		WHERE user_id = ?
		ORDER BY due_date ASC
	`

	sqliteGetPaidCyclesByAccountPayableIDQuery = `
		SELECT due_date, paid_at
		FROM account_payable_payments
		WHERE account_payable_id = ?
		ORDER BY due_date ASC
	`
)

type SQLiteTransactionRepository struct {
	database *sql.DB
	logger   ports.Logger
}

type sqliteTransactionExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewSQLiteTransactionRepository(database *sql.DB, logger ports.Logger) ports.TransactionRepository {
	return &SQLiteTransactionRepository{database: database, logger: logger}
}

func (repository *SQLiteTransactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	if transaction == nil {
		repository.logger.Error("cannot create nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	_, err := execSQLiteCreateTransaction(ctx, repository.database, transaction)
	if err == nil {
		return nil
	}

	return repository.mapWriteError(err, "failed to create transaction", "userID", transaction.UserID(), "transactionID", transaction.ID())
}

func (repository *SQLiteTransactionRepository) GetByID(ctx context.Context, id string) (*domain.Transaction, error) {
	transaction, err := repository.scanTransaction(repository.database.QueryRowContext(ctx, sqliteGetTransactionByIDQuery, id))
	if err == nil {
		return transaction, nil
	}

	return repository.mapReadError(err, "failed to retrieve transaction by id", "transactionID", id)
}

func (repository *SQLiteTransactionRepository) GetByUserID(ctx context.Context, userID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	query, arguments := buildSQLiteTransactionsByUserIDQuery(userID, filter)
	return repository.queryTransactions(ctx, query, arguments, "failed to retrieve transactions by user id", "userID", userID)
}

func (repository *SQLiteTransactionRepository) GetByCreditCardID(ctx context.Context, userID, creditCardID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	query, arguments := buildSQLiteTransactionsByUserIDAndCreditCardIDQuery(userID, creditCardID, filter)
	return repository.queryTransactions(ctx, query, arguments, "failed to retrieve transactions by credit card id", "userID", userID, "creditCardID", creditCardID)
}

func (repository *SQLiteTransactionRepository) GetByPaymentAccountID(ctx context.Context, userID, paymentAccountID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	query, arguments := buildSQLiteTransactionsByUserIDAndPaymentAccountIDQuery(userID, paymentAccountID, filter)
	return repository.queryTransactions(ctx, query, arguments, "failed to retrieve transactions by payment account id", "userID", userID, "paymentAccountID", paymentAccountID)
}

func (repository *SQLiteTransactionRepository) GetPaidCyclesByUserID(ctx context.Context, userID string) (map[string][]domain.PaidCycle, error) {
	rows, err := repository.database.QueryContext(ctx, sqliteGetPaidCyclesByUserIDQuery, userID)
	if err != nil {
		repository.logger.Error("failed to retrieve account payable paid cycles", "userID", userID, "error", err)
		return nil, ports.ErrTransactionRepositoryUnavailable
	}
	defer rows.Close()

	paidCycles := make(map[string][]domain.PaidCycle)
	for rows.Next() {
		var accountPayableID string
		var dueDate time.Time
		var paidAt time.Time
		if err := rows.Scan(&accountPayableID, &dueDate, &paidAt); err != nil {
			repository.logger.Error("failed to scan account payable paid cycle", "userID", userID, "error", err)
			return nil, ports.ErrTransactionRepositoryUnavailable
		}
		paidCycles[accountPayableID] = append(paidCycles[accountPayableID], domain.PaidCycle{DueDate: dueDate, PaidAt: paidAt})
	}
	if err := rows.Err(); err != nil {
		return nil, ports.ErrTransactionRepositoryUnavailable
	}

	return paidCycles, nil
}

func (repository *SQLiteTransactionRepository) GetPaidCyclesByAccountPayableID(ctx context.Context, accountPayableID string) ([]domain.PaidCycle, error) {
	rows, err := repository.database.QueryContext(ctx, sqliteGetPaidCyclesByAccountPayableIDQuery, accountPayableID)
	if err != nil {
		repository.logger.Error("failed to retrieve account payable paid cycles", "transactionID", accountPayableID, "error", err)
		return nil, ports.ErrTransactionRepositoryUnavailable
	}
	defer rows.Close()

	paidCycles := make([]domain.PaidCycle, 0)
	for rows.Next() {
		var dueDate time.Time
		var paidAt time.Time
		if err := rows.Scan(&dueDate, &paidAt); err != nil {
			repository.logger.Error("failed to scan account payable paid cycle", "transactionID", accountPayableID, "error", err)
			return nil, ports.ErrTransactionRepositoryUnavailable
		}
		paidCycles = append(paidCycles, domain.PaidCycle{DueDate: dueDate, PaidAt: paidAt})
	}
	if err := rows.Err(); err != nil {
		return nil, ports.ErrTransactionRepositoryUnavailable
	}

	return paidCycles, nil
}

func (repository *SQLiteTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	if transaction == nil {
		repository.logger.Error("cannot update nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := execSQLiteUpdateTransaction(ctx, repository.database, transaction); err != nil {
		repository.logger.Error("failed to update transaction", "userID", transaction.UserID(), "transactionID", transaction.ID(), "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteTransactionRepository) CreateAndUpdate(ctx context.Context, transactionToCreate, transactionToUpdate *domain.Transaction) error {
	if transactionToCreate == nil || transactionToUpdate == nil {
		repository.logger.Error("cannot create and update nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		repository.logger.Error("failed to start transaction payment unit", "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}
	defer tx.Rollback()

	if _, err := execSQLiteCreateTransaction(ctx, tx, transactionToCreate); err != nil {
		repository.logger.Error("failed to create payable payment transaction", "userID", transactionToCreate.UserID(), "transactionID", transactionToCreate.ID(), "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := execSQLiteUpdateTransaction(ctx, tx, transactionToUpdate); err != nil {
		repository.logger.Error("failed to update paid account payable", "userID", transactionToUpdate.UserID(), "transactionID", transactionToUpdate.ID(), "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	if err := tx.Commit(); err != nil {
		repository.logger.Error("failed to commit payable payment unit", "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteTransactionRepository) CreatePaymentCycleAndUpdate(ctx context.Context, transactionToCreate, transactionToUpdate *domain.Transaction, cyclePayment ports.AccountPayableCyclePayment) error {
	if transactionToCreate == nil || transactionToUpdate == nil {
		repository.logger.Error("cannot create payable cycle with nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		repository.logger.Error("failed to start payable cycle payment unit", "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}
	defer tx.Rollback()

	if _, err := execSQLiteCreateTransaction(ctx, tx, transactionToCreate); err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := tx.ExecContext(
		ctx,
		sqliteCreateAccountPayablePaymentQuery,
		cyclePayment.ID,
		cyclePayment.AccountPayableID,
		cyclePayment.UserID,
		timeValue(cyclePayment.DueDate),
		timeValue(cyclePayment.PaidAt),
		cyclePayment.AmountCents,
		cyclePayment.Concept,
		cyclePayment.Category,
		cyclePayment.CreatedTransactionID,
		timeValue(cyclePayment.CreatedAt),
	); err != nil {
		if isSQLiteConstraintViolation(err) {
			return ports.ErrAccountPayableCycleAlreadyPaid
		}
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := execSQLiteUpdateTransaction(ctx, tx, transactionToUpdate); err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}

	if err := tx.Commit(); err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteTransactionRepository) CreateManyWithLedger(ctx context.Context, transactions []*domain.Transaction, accountDeltas []ports.AccountBalanceDelta, ledgerEntries []ports.LedgerEntry) error {
	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}
	defer tx.Rollback()
	for _, transaction := range transactions {
		if transaction == nil {
			return ports.ErrTransactionRepositoryUnavailable
		}
		if _, err := execSQLiteCreateTransaction(ctx, tx, transaction); err != nil {
			return ports.ErrTransactionRepositoryUnavailable
		}
	}
	for _, delta := range accountDeltas {
		if _, err := tx.ExecContext(ctx, "UPDATE financial_accounts SET current_balance_cents = current_balance_cents + ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", delta.DeltaCents, timeValue(time.Now()), delta.AccountID); err != nil {
			return ports.ErrTransactionRepositoryUnavailable
		}
	}
	for _, entry := range ledgerEntries {
		if _, err := tx.ExecContext(ctx, "INSERT INTO ledger_entries (id, user_id, account_id, transaction_id, amount_cents, entry_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", entry.ID, entry.UserID, entry.AccountID, entry.TransactionID, entry.AmountCents, entry.EntryType, timeValue(entry.CreatedAt), timeValue(entry.UpdatedAt)); err != nil {
			return ports.ErrTransactionRepositoryUnavailable
		}
	}
	if err := tx.Commit(); err != nil {
		return ports.ErrTransactionRepositoryUnavailable
	}
	return nil
}

func (repository *SQLiteTransactionRepository) Delete(ctx context.Context, id string) error {
	deletedAt := timeValue(time.Now())
	if _, err := repository.database.ExecContext(ctx, sqliteDeleteTransactionQuery, deletedAt, deletedAt, id); err != nil {
		repository.logger.Error("failed to delete transaction", "transactionID", id, "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func execSQLiteCreateTransaction(ctx context.Context, executor sqliteTransactionExecutor, transaction *domain.Transaction) (sql.Result, error) {
	return executor.ExecContext(ctx, sqliteCreateTransactionQuery, transaction.ID(), transaction.UserID(), nullableString(transaction.CreditCardID()), nullableString(transaction.PaymentAccountID()), nullableString(transaction.DestinationAccountID()), string(transaction.Type()), transaction.Concept(), transaction.Category(), transaction.AmountCents(), timeValue(transaction.Date()), string(transaction.Status()), nullableInt(transaction.MSI()), nullableInt(transaction.InstallmentNumber()), nullableInt(transaction.InstallmentCount()), string(transaction.Recurrence()), nullableInt(transaction.RecurrenceLimit()), nullableTimePtr(transaction.LastPaidAt()), timeValue(transaction.CreatedAt()), timeValue(transaction.UpdatedAt()))
}

func execSQLiteUpdateTransaction(ctx context.Context, executor sqliteTransactionExecutor, transaction *domain.Transaction) (sql.Result, error) {
	return executor.ExecContext(ctx, sqliteUpdateTransactionQuery, string(transaction.Type()), transaction.Concept(), transaction.Category(), transaction.AmountCents(), timeValue(transaction.Date()), string(transaction.Status()), nullableInt(transaction.MSI()), nullableString(transaction.CreditCardID()), nullableString(transaction.PaymentAccountID()), nullableString(transaction.DestinationAccountID()), nullableInt(transaction.InstallmentNumber()), nullableInt(transaction.InstallmentCount()), string(transaction.Recurrence()), nullableInt(transaction.RecurrenceLimit()), nullableTimePtr(transaction.LastPaidAt()), timeValue(transaction.UpdatedAt()), transaction.ID())
}

func (repository *SQLiteTransactionRepository) queryTransactions(ctx context.Context, query string, arguments []interface{}, message string, keysAndValues ...interface{}) ([]*domain.Transaction, error) {
	rows, err := repository.database.QueryContext(ctx, query, arguments...)
	if err != nil {
		logValues := append(keysAndValues, "error", err)
		repository.logger.Error(message, logValues...)
		return nil, ports.ErrTransactionRepositoryUnavailable
	}
	defer rows.Close()

	transactions := make([]*domain.Transaction, 0)
	for rows.Next() {
		transaction, err := repository.scanTransaction(rows)
		if err != nil {
			repository.logger.Error("failed to scan transaction row", "error", err)
			return nil, ports.ErrTransactionRepositoryUnavailable
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		repository.logger.Error("failed while iterating transaction rows", "error", err)
		return nil, ports.ErrTransactionRepositoryUnavailable
	}

	return transactions, nil
}

func (repository *SQLiteTransactionRepository) scanTransaction(row interface {
	Scan(dest ...interface{}) error
}) (*domain.Transaction, error) {
	var storedTransaction sqliteStoredTransaction
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
		scanNullableInt(storedTransaction.msi),
		scanNullableString(storedTransaction.creditCardID),
		domain.TransactionRecurrence(storedTransaction.recurrence),
		scanNullableInt(storedTransaction.recurrenceLimit),
		scanNullableTimePtr(storedTransaction.lastPaidAt),
		storedTransaction.createdAt,
		storedTransaction.updatedAt,
	)
	if err != nil {
		return nil, err
	}
	transaction.SetAccountingDetails(scanNullableString(storedTransaction.paymentAccountID), scanNullableString(storedTransaction.destinationAccountID), scanNullableInt(storedTransaction.installmentNumber), scanNullableInt(storedTransaction.installmentCount))
	return transaction, nil
}

func (repository *SQLiteTransactionRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.Transaction, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrTransactionNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrTransactionRepositoryUnavailable
}

func (repository *SQLiteTransactionRepository) mapWriteError(err error, message string, keysAndValues ...interface{}) error {
	if isSQLiteConstraintViolation(err) {
		repository.logger.Warn("transaction already exists")
		return ports.ErrTransactionAlreadyExists
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return ports.ErrTransactionRepositoryUnavailable
}

func buildSQLiteTransactionsByUserIDQuery(userID string, filter ports.TransactionDateFilter) (string, []interface{}) {
	if filter.From != nil && filter.To != nil {
		return sqliteGetTransactionsByUserIDRangeQuery, []interface{}{userID, timeValue(*filter.From), timeValue(*filter.To)}
	}
	if filter.From != nil {
		return sqliteGetTransactionsByUserIDFromQuery, []interface{}{userID, timeValue(*filter.From)}
	}
	if filter.To != nil {
		return sqliteGetTransactionsByUserIDToQuery, []interface{}{userID, timeValue(*filter.To)}
	}

	return sqliteGetTransactionsByUserIDQuery, []interface{}{userID}
}

func buildSQLiteTransactionsByUserIDAndCreditCardIDQuery(userID, creditCardID string, filter ports.TransactionDateFilter) (string, []interface{}) {
	if filter.From != nil && filter.To != nil {
		return sqliteGetTransactionsByUserIDAndCreditCardIDRangeQuery, []interface{}{userID, creditCardID, timeValue(*filter.From), timeValue(*filter.To)}
	}
	if filter.From != nil {
		return sqliteGetTransactionsByUserIDAndCreditCardIDFromQuery, []interface{}{userID, creditCardID, timeValue(*filter.From)}
	}
	if filter.To != nil {
		return sqliteGetTransactionsByUserIDAndCreditCardIDToQuery, []interface{}{userID, creditCardID, timeValue(*filter.To)}
	}

	return sqliteGetTransactionsByUserIDAndCreditCardIDQuery, []interface{}{userID, creditCardID}
}

func buildSQLiteTransactionsByUserIDAndPaymentAccountIDQuery(userID, paymentAccountID string, filter ports.TransactionDateFilter) (string, []interface{}) {
	if filter.From != nil && filter.To != nil {
		return sqliteGetTransactionsByUserIDAndPaymentAccountIDRangeQuery, []interface{}{userID, paymentAccountID, timeValue(*filter.From), timeValue(*filter.To)}
	}
	if filter.From != nil {
		return sqliteGetTransactionsByUserIDAndPaymentAccountIDFromQuery, []interface{}{userID, paymentAccountID, timeValue(*filter.From)}
	}
	if filter.To != nil {
		return sqliteGetTransactionsByUserIDAndPaymentAccountIDToQuery, []interface{}{userID, paymentAccountID, timeValue(*filter.To)}
	}

	return sqliteGetTransactionsByUserIDAndPaymentAccountIDQuery, []interface{}{userID, paymentAccountID}
}

type sqliteStoredTransaction struct {
	id                   string
	userID               string
	creditCardID         sql.NullString
	paymentAccountID     sql.NullString
	destinationAccountID sql.NullString
	transactionType      string
	concept              string
	category             string
	amountCents          int64
	date                 time.Time
	status               string
	msi                  sql.NullInt64
	installmentNumber    sql.NullInt64
	installmentCount     sql.NullInt64
	recurrence           string
	recurrenceLimit      sql.NullInt64
	lastPaidAt           sql.NullTime
	createdAt            time.Time
	updatedAt            time.Time
}
