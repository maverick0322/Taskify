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
		INSERT INTO transactions (id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	sqliteGetTransactionByIDQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE id = ? AND deleted_at IS NULL
	`

	sqliteGetTransactionsByUserIDQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDFromQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND date >= ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDToQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDRangeQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND date >= ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndCreditCardIDQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND credit_card_id = ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndCreditCardIDFromQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND credit_card_id = ? AND date >= ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndCreditCardIDToQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND credit_card_id = ? AND date < ? AND deleted_at IS NULL
		ORDER BY date DESC
	`

	sqliteGetTransactionsByUserIDAndCreditCardIDRangeQuery = `
		SELECT id, user_id, credit_card_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = ? AND credit_card_id = ? AND date >= ? AND date < ? AND deleted_at IS NULL
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
			updated_at = ?
		WHERE id = ?
	`

	sqliteDeleteTransactionQuery = `
		UPDATE transactions
		SET deleted_at = ?, updated_at = ?
		WHERE id = ?
	`
)

type SQLiteTransactionRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteTransactionRepository(database *sql.DB, logger ports.Logger) ports.TransactionRepository {
	return &SQLiteTransactionRepository{database: database, logger: logger}
}

func (repository *SQLiteTransactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	if transaction == nil {
		repository.logger.Error("cannot create nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	_, err := repository.database.ExecContext(
		ctx,
		sqliteCreateTransactionQuery,
		transaction.ID(),
		transaction.UserID(),
		nullableString(transaction.CreditCardID()),
		string(transaction.Type()),
		transaction.Concept(),
		transaction.Category(),
		transaction.AmountCents(),
		timeValue(transaction.Date()),
		string(transaction.Status()),
		nullableInt(transaction.MSI()),
		timeValue(transaction.CreatedAt()),
		timeValue(transaction.UpdatedAt()),
	)
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

func (repository *SQLiteTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	if transaction == nil {
		repository.logger.Error("cannot update nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := repository.database.ExecContext(
		ctx,
		sqliteUpdateTransactionQuery,
		string(transaction.Type()),
		transaction.Concept(),
		transaction.Category(),
		transaction.AmountCents(),
		timeValue(transaction.Date()),
		string(transaction.Status()),
		nullableInt(transaction.MSI()),
		nullableString(transaction.CreditCardID()),
		timeValue(transaction.UpdatedAt()),
		transaction.ID(),
	); err != nil {
		repository.logger.Error("failed to update transaction", "userID", transaction.UserID(), "transactionID", transaction.ID(), "error", err)
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
		&storedTransaction.transactionType,
		&storedTransaction.concept,
		&storedTransaction.category,
		&storedTransaction.amountCents,
		&storedTransaction.date,
		&storedTransaction.status,
		&storedTransaction.msi,
		&storedTransaction.createdAt,
		&storedTransaction.updatedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateTransaction(
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
		storedTransaction.createdAt,
		storedTransaction.updatedAt,
	)
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

type sqliteStoredTransaction struct {
	id              string
	userID          string
	creditCardID    sql.NullString
	transactionType string
	concept         string
	category        string
	amountCents     int64
	date            time.Time
	status          string
	msi             sql.NullInt64
	createdAt       time.Time
	updatedAt       time.Time
}
