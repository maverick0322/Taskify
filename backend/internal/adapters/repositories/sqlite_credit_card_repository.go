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
	sqliteCreateCreditCardQuery = `
		INSERT INTO credit_cards (id, user_id, name, bank, last4, cutoff_day, payment_day, limit_cents, color, network, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	sqliteCreateCreditCardFinancialAccountQuery = `
		INSERT INTO financial_accounts (id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, network, created_at, updated_at)
		VALUES (?, ?, 'CREDIT_CARD', ?, ?, ?, 0, 0, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name, institution = excluded.institution, last4 = excluded.last4, credit_limit_cents = excluded.credit_limit_cents, cutoff_day = excluded.cutoff_day, payment_day = excluded.payment_day, color = excluded.color, network = excluded.network, updated_at = excluded.updated_at
	`

	sqliteGetCreditCardByIDQuery = `
		SELECT id, user_id, name, bank, last4, cutoff_day, payment_day, limit_cents, color, network, created_at, updated_at
		FROM credit_cards
		WHERE id = ? AND deleted_at IS NULL
	`

	sqliteGetCreditCardsByUserIDQuery = `
		SELECT id, user_id, name, bank, last4, cutoff_day, payment_day, limit_cents, color, network, created_at, updated_at
		FROM credit_cards
		WHERE user_id = ? AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	sqliteUpdateCreditCardQuery = `
		UPDATE credit_cards
		SET name = ?,
			bank = ?,
			last4 = ?,
			cutoff_day = ?,
			payment_day = ?,
			limit_cents = ?,
			color = ?,
			network = ?,
			updated_at = ?
		WHERE id = ?
	`
	sqliteUpdateCreditCardFinancialAccountQuery = `
		UPDATE financial_accounts
		SET name = ?, institution = ?, last4 = ?, credit_limit_cents = ?, cutoff_day = ?, payment_day = ?, color = ?, network = ?, updated_at = ?
		WHERE id = ?
	`

	sqliteDeleteCreditCardQuery = `
		UPDATE credit_cards
		SET deleted_at = ?, updated_at = ?
		WHERE id = ?
	`
	sqliteDeleteCreditCardFinancialAccountQuery = `
		UPDATE financial_accounts
		SET deleted_at = ?, updated_at = ?
		WHERE id = ?
	`

	sqliteNullDeletedCreditCardTransactionsQuery = `
		UPDATE transactions
		SET credit_card_id = NULL, updated_at = ?
		WHERE credit_card_id = ? AND deleted_at IS NULL
	`
)

type SQLiteCreditCardRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteCreditCardRepository(database *sql.DB, logger ports.Logger) ports.CreditCardRepository {
	return &SQLiteCreditCardRepository{database: database, logger: logger}
}

func (repository *SQLiteCreditCardRepository) Create(ctx context.Context, creditCard *domain.CreditCard) error {
	if creditCard == nil {
		repository.logger.Error("cannot create nil credit card")
		return ports.ErrCreditCardRepositoryUnavailable
	}

	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return ports.ErrCreditCardRepositoryUnavailable
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		sqliteCreateCreditCardQuery,
		creditCard.ID(),
		creditCard.UserID(),
		creditCard.Name(),
		creditCard.Bank(),
		creditCard.Last4(),
		creditCard.CutoffDay(),
		creditCard.PaymentDay(),
		creditCard.LimitCents(),
		creditCard.Color(),
		creditCard.Network(),
		timeValue(creditCard.CreatedAt()),
		timeValue(creditCard.UpdatedAt()),
	)
	if err != nil {
		return repository.mapWriteError(err, "failed to create credit card", "userID", creditCard.UserID(), "creditCardID", creditCard.ID())
	}
	if _, err := tx.ExecContext(ctx, sqliteCreateCreditCardFinancialAccountQuery, creditCard.ID(), creditCard.UserID(), creditCard.Name(), creditCard.Bank(), creditCard.Last4(), creditCard.LimitCents(), creditCard.CutoffDay(), creditCard.PaymentDay(), creditCard.Color(), creditCard.Network(), timeValue(creditCard.CreatedAt()), timeValue(creditCard.UpdatedAt())); err != nil {
		return repository.mapWriteError(err, "failed to create credit card financial account", "userID", creditCard.UserID(), "creditCardID", creditCard.ID())
	}
	if err := tx.Commit(); err != nil {
		return ports.ErrCreditCardRepositoryUnavailable
	}
	return nil
}

func (repository *SQLiteCreditCardRepository) GetByID(ctx context.Context, id string) (*domain.CreditCard, error) {
	creditCard, err := repository.scanCreditCard(repository.database.QueryRowContext(ctx, sqliteGetCreditCardByIDQuery, id))
	if err == nil {
		return creditCard, nil
	}

	return repository.mapReadError(err, "failed to retrieve credit card by id", "creditCardID", id)
}

func (repository *SQLiteCreditCardRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.CreditCard, error) {
	rows, err := repository.database.QueryContext(ctx, sqliteGetCreditCardsByUserIDQuery, userID)
	if err != nil {
		repository.logger.Error("failed to retrieve credit cards by user id", "userID", userID, "error", err)
		return nil, ports.ErrCreditCardRepositoryUnavailable
	}
	defer rows.Close()

	creditCards := make([]*domain.CreditCard, 0)
	for rows.Next() {
		creditCard, err := repository.scanCreditCard(rows)
		if err != nil {
			repository.logger.Error("failed to scan credit card row", "userID", userID, "error", err)
			return nil, ports.ErrCreditCardRepositoryUnavailable
		}
		creditCards = append(creditCards, creditCard)
	}
	if err := rows.Err(); err != nil {
		repository.logger.Error("failed while iterating credit card rows", "userID", userID, "error", err)
		return nil, ports.ErrCreditCardRepositoryUnavailable
	}

	return creditCards, nil
}

func (repository *SQLiteCreditCardRepository) Update(ctx context.Context, creditCard *domain.CreditCard) error {
	if creditCard == nil {
		repository.logger.Error("cannot update nil credit card")
		return ports.ErrCreditCardRepositoryUnavailable
	}

	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return ports.ErrCreditCardRepositoryUnavailable
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(
		ctx,
		sqliteUpdateCreditCardQuery,
		creditCard.Name(),
		creditCard.Bank(),
		creditCard.Last4(),
		creditCard.CutoffDay(),
		creditCard.PaymentDay(),
		creditCard.LimitCents(),
		creditCard.Color(),
		creditCard.Network(),
		timeValue(creditCard.UpdatedAt()),
		creditCard.ID(),
	); err != nil {
		repository.logger.Error("failed to update credit card", "userID", creditCard.UserID(), "creditCardID", creditCard.ID(), "error", err)
		return ports.ErrCreditCardRepositoryUnavailable
	}
	if _, err := tx.ExecContext(ctx, sqliteUpdateCreditCardFinancialAccountQuery, creditCard.Name(), creditCard.Bank(), creditCard.Last4(), creditCard.LimitCents(), creditCard.CutoffDay(), creditCard.PaymentDay(), creditCard.Color(), creditCard.Network(), timeValue(creditCard.UpdatedAt()), creditCard.ID()); err != nil {
		return ports.ErrCreditCardRepositoryUnavailable
	}
	if err := tx.Commit(); err != nil {
		return ports.ErrCreditCardRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteCreditCardRepository) Delete(ctx context.Context, id string) error {
	deletedAt := timeValue(time.Now())
	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		repository.logger.Error("failed to begin credit card soft delete", "creditCardID", id, "error", err)
		return ports.ErrCreditCardRepositoryUnavailable
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, sqliteDeleteCreditCardQuery, deletedAt, deletedAt, id); err != nil {
		repository.logger.Error("failed to delete credit card", "creditCardID", id, "error", err)
		return ports.ErrCreditCardRepositoryUnavailable
	}
	if _, err := tx.ExecContext(ctx, sqliteDeleteCreditCardFinancialAccountQuery, deletedAt, deletedAt, id); err != nil {
		return ports.ErrCreditCardRepositoryUnavailable
	}
	if _, err := tx.ExecContext(ctx, sqliteNullDeletedCreditCardTransactionsQuery, deletedAt, id); err != nil {
		repository.logger.Error("failed to detach deleted credit card transactions", "creditCardID", id, "error", err)
		return ports.ErrCreditCardRepositoryUnavailable
	}
	if err := tx.Commit(); err != nil {
		repository.logger.Error("failed to commit credit card soft delete", "creditCardID", id, "error", err)
		return ports.ErrCreditCardRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteCreditCardRepository) scanCreditCard(row interface {
	Scan(dest ...interface{}) error
}) (*domain.CreditCard, error) {
	var storedCreditCard storedCreditCard
	if err := row.Scan(
		&storedCreditCard.id,
		&storedCreditCard.userID,
		&storedCreditCard.name,
		&storedCreditCard.bank,
		&storedCreditCard.last4,
		&storedCreditCard.cutoffDay,
		&storedCreditCard.paymentDay,
		&storedCreditCard.limitCents,
		&storedCreditCard.color,
		&storedCreditCard.network,
		&storedCreditCard.createdAt,
		&storedCreditCard.updatedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateCreditCard(
		storedCreditCard.id,
		storedCreditCard.userID,
		storedCreditCard.name,
		storedCreditCard.bank,
		storedCreditCard.last4,
		storedCreditCard.cutoffDay,
		storedCreditCard.paymentDay,
		storedCreditCard.limitCents,
		storedCreditCard.color,
		storedCreditCard.network,
		storedCreditCard.createdAt,
		storedCreditCard.updatedAt,
	)
}

func (repository *SQLiteCreditCardRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.CreditCard, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrCreditCardNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrCreditCardRepositoryUnavailable
}

func (repository *SQLiteCreditCardRepository) mapWriteError(err error, message string, keysAndValues ...interface{}) error {
	if isSQLiteConstraintViolation(err) {
		repository.logger.Warn("credit card already exists")
		return ports.ErrCreditCardAlreadyExists
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return ports.ErrCreditCardRepositoryUnavailable
}
