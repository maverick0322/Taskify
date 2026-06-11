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
		INSERT INTO transactions (id, user_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	getTransactionByIDQuery = `
		SELECT id, user_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE id = $1
	`

	getTransactionsByUserIDQuery = `
		SELECT id, user_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY date DESC
	`

	getTransactionsByUserIDFromQuery = `
		SELECT id, user_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND date >= $2
		ORDER BY date DESC
	`

	getTransactionsByUserIDToQuery = `
		SELECT id, user_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND date < $2
		ORDER BY date DESC
	`

	getTransactionsByUserIDRangeQuery = `
		SELECT id, user_id, type, concept, category, amount_cents, date, status, msi, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND date >= $2 AND date < $3
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
			updated_at = $9
		WHERE id = $1
	`

	deleteTransactionQuery = `
		DELETE FROM transactions
		WHERE id = $1
	`
)

type postgresTransactionDatabase interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
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

	_, err := repository.database.Exec(
		ctx,
		createTransactionQuery,
		transaction.ID(),
		transaction.UserID(),
		string(transaction.Type()),
		transaction.Concept(),
		transaction.Category(),
		transaction.AmountCents(),
		transaction.Date(),
		string(transaction.Status()),
		nullableTransactionMSI(transaction.MSI()),
		transaction.CreatedAt(),
		transaction.UpdatedAt(),
	)
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
	rows, err := repository.database.Query(ctx, query, arguments...)
	if err != nil {
		repository.logger.Error("failed to retrieve transactions by user id", "userID", userID, "error", err)
		return nil, ports.ErrTransactionRepositoryUnavailable
	}
	defer rows.Close()

	transactions := make([]*domain.Transaction, 0)
	for rows.Next() {
		transaction, err := repository.scanTransaction(rows)
		if err != nil {
			repository.logger.Error("failed to scan transaction row", "userID", userID, "error", err)
			return nil, ports.ErrTransactionRepositoryUnavailable
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		repository.logger.Error("failed while iterating transaction rows", "userID", userID, "error", err)
		return nil, ports.ErrTransactionRepositoryUnavailable
	}

	return transactions, nil
}

func (repository *PostgresTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	if transaction == nil {
		repository.logger.Error("cannot update nil transaction")
		return ports.ErrTransactionRepositoryUnavailable
	}

	if _, err := repository.database.Exec(
		ctx,
		updateTransactionQuery,
		transaction.ID(),
		string(transaction.Type()),
		transaction.Concept(),
		transaction.Category(),
		transaction.AmountCents(),
		transaction.Date(),
		string(transaction.Status()),
		nullableTransactionMSI(transaction.MSI()),
		transaction.UpdatedAt(),
	); err != nil {
		repository.logger.Error("failed to update transaction", "userID", transaction.UserID(), "transactionID", transaction.ID(), "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresTransactionRepository) Delete(ctx context.Context, id string) error {
	if _, err := repository.database.Exec(ctx, deleteTransactionQuery, id); err != nil {
		repository.logger.Error("failed to delete transaction", "transactionID", id, "error", err)
		return ports.ErrTransactionRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresTransactionRepository) scanTransaction(row pgx.Row) (*domain.Transaction, error) {
	var storedTransaction storedTransaction
	if err := row.Scan(
		&storedTransaction.id,
		&storedTransaction.userID,
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
		storedTransaction.msi,
		storedTransaction.createdAt,
		storedTransaction.updatedAt,
	)
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
	id              string
	userID          string
	transactionType string
	concept         string
	category        string
	amountCents     int64
	date            time.Time
	status          string
	msi             *int
	createdAt       time.Time
	updatedAt       time.Time
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

func nullableTransactionMSI(msi *int) interface{} {
	if msi == nil {
		return nil
	}

	return *msi
}
