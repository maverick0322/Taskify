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
	createCreditCardQuery = `
		INSERT INTO credit_cards (id, user_id, name, bank, last4, cutoff_day, payment_day, limit_cents, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	getCreditCardByIDQuery = `
		SELECT id, user_id, name, bank, last4, cutoff_day, payment_day, limit_cents, color, created_at, updated_at
		FROM credit_cards
		WHERE id = $1
	`

	getCreditCardsByUserIDQuery = `
		SELECT id, user_id, name, bank, last4, cutoff_day, payment_day, limit_cents, color, created_at, updated_at
		FROM credit_cards
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	updateCreditCardQuery = `
		UPDATE credit_cards
		SET name = $2,
			bank = $3,
			last4 = $4,
			cutoff_day = $5,
			payment_day = $6,
			limit_cents = $7,
			color = $8,
			updated_at = $9
		WHERE id = $1
	`

	deleteCreditCardQuery = `
		DELETE FROM credit_cards
		WHERE id = $1
	`
)

type postgresCreditCardDatabase interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

type PostgresCreditCardRepository struct {
	database postgresCreditCardDatabase
	logger   ports.Logger
}

func NewPostgresCreditCardRepository(pool *pgxpool.Pool, logger ports.Logger) ports.CreditCardRepository {
	return &PostgresCreditCardRepository{
		database: pool,
		logger:   logger,
	}
}

func (repository *PostgresCreditCardRepository) Create(ctx context.Context, creditCard *domain.CreditCard) error {
	if creditCard == nil {
		repository.logger.Error("cannot create nil credit card")
		return ports.ErrCreditCardRepositoryUnavailable
	}

	_, err := repository.database.Exec(
		ctx,
		createCreditCardQuery,
		creditCard.ID(),
		creditCard.UserID(),
		creditCard.Name(),
		creditCard.Bank(),
		creditCard.Last4(),
		creditCard.CutoffDay(),
		creditCard.PaymentDay(),
		creditCard.LimitCents(),
		creditCard.Color(),
		creditCard.CreatedAt(),
		creditCard.UpdatedAt(),
	)
	if err == nil {
		return nil
	}

	return repository.mapWriteError(err, "failed to create credit card", "userID", creditCard.UserID(), "creditCardID", creditCard.ID())
}

func (repository *PostgresCreditCardRepository) GetByID(ctx context.Context, id string) (*domain.CreditCard, error) {
	creditCard, err := repository.scanCreditCard(repository.database.QueryRow(ctx, getCreditCardByIDQuery, id))
	if err == nil {
		return creditCard, nil
	}

	return repository.mapReadError(err, "failed to retrieve credit card by id", "creditCardID", id)
}

func (repository *PostgresCreditCardRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.CreditCard, error) {
	rows, err := repository.database.Query(ctx, getCreditCardsByUserIDQuery, userID)
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

func (repository *PostgresCreditCardRepository) Update(ctx context.Context, creditCard *domain.CreditCard) error {
	if creditCard == nil {
		repository.logger.Error("cannot update nil credit card")
		return ports.ErrCreditCardRepositoryUnavailable
	}

	if _, err := repository.database.Exec(
		ctx,
		updateCreditCardQuery,
		creditCard.ID(),
		creditCard.Name(),
		creditCard.Bank(),
		creditCard.Last4(),
		creditCard.CutoffDay(),
		creditCard.PaymentDay(),
		creditCard.LimitCents(),
		creditCard.Color(),
		creditCard.UpdatedAt(),
	); err != nil {
		repository.logger.Error("failed to update credit card", "userID", creditCard.UserID(), "creditCardID", creditCard.ID(), "error", err)
		return ports.ErrCreditCardRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresCreditCardRepository) Delete(ctx context.Context, id string) error {
	if _, err := repository.database.Exec(ctx, deleteCreditCardQuery, id); err != nil {
		repository.logger.Error("failed to delete credit card", "creditCardID", id, "error", err)
		return ports.ErrCreditCardRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresCreditCardRepository) scanCreditCard(row pgx.Row) (*domain.CreditCard, error) {
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
		storedCreditCard.createdAt,
		storedCreditCard.updatedAt,
	)
}

func (repository *PostgresCreditCardRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.CreditCard, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrCreditCardNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrCreditCardRepositoryUnavailable
}

func (repository *PostgresCreditCardRepository) mapWriteError(err error, message string, keysAndValues ...interface{}) error {
	if isUniqueViolation(err) {
		repository.logger.Warn("credit card already exists")
		return ports.ErrCreditCardAlreadyExists
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return ports.ErrCreditCardRepositoryUnavailable
}

type storedCreditCard struct {
	id         string
	userID     string
	name       string
	bank       string
	last4      string
	cutoffDay  int
	paymentDay int
	limitCents int64
	color      string
	createdAt  time.Time
	updatedAt  time.Time
}
