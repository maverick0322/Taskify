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
	createFinancialAccountQuery = `
		INSERT INTO financial_accounts (id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	getFinancialAccountByIDQuery = `
		SELECT id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, created_at, updated_at
		FROM financial_accounts WHERE id = $1 AND deleted_at IS NULL
	`
	getFinancialAccountsByUserIDQuery = `
		SELECT id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, created_at, updated_at
		FROM financial_accounts WHERE user_id = $1 AND deleted_at IS NULL ORDER BY type ASC, created_at ASC
	`
	updateFinancialAccountQuery = `
		UPDATE financial_accounts
		SET name = $2, institution = $3, last4 = $4, opening_balance_cents = $5, current_balance_cents = $6, credit_limit_cents = $7, cutoff_day = $8, payment_day = $9, color = $10, updated_at = $11
		WHERE id = $1
	`
	deleteFinancialAccountQuery = `
		UPDATE financial_accounts SET deleted_at = $2, updated_at = $2 WHERE id = $1
	`
)

type postgresFinancialAccountDatabase interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

type PostgresFinancialAccountRepository struct {
	database postgresFinancialAccountDatabase
	logger   ports.Logger
}

func NewPostgresFinancialAccountRepository(pool *pgxpool.Pool, logger ports.Logger) ports.FinancialAccountRepository {
	return &PostgresFinancialAccountRepository{database: pool, logger: logger}
}

func (repository *PostgresFinancialAccountRepository) Create(ctx context.Context, account *domain.FinancialAccount) error {
	if account == nil {
		return ports.ErrFinancialAccountRepositoryUnavailable
	}
	_, err := repository.database.Exec(ctx, createFinancialAccountQuery, account.ID(), account.UserID(), string(account.Type()), account.Name(), account.Institution(), nullableTransactionCreditCardID(account.Last4()), account.OpeningBalanceCents(), account.CurrentBalanceCents(), nullableAccountInt64(account.CreditLimitCents()), nullableAccountInt(account.CutoffDay()), nullableAccountInt(account.PaymentDay()), account.Color(), account.CreatedAt(), account.UpdatedAt())
	if err == nil {
		return nil
	}
	if isUniqueViolation(err) {
		return ports.ErrFinancialAccountAlreadyExists
	}
	repository.logger.Error("failed to create financial account", "error", err)
	return ports.ErrFinancialAccountRepositoryUnavailable
}

func (repository *PostgresFinancialAccountRepository) GetByID(ctx context.Context, id string) (*domain.FinancialAccount, error) {
	account, err := repository.scanAccount(repository.database.QueryRow(ctx, getFinancialAccountByIDQuery, id))
	if err == nil {
		return account, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrFinancialAccountNotFound
	}
	return nil, ports.ErrFinancialAccountRepositoryUnavailable
}

func (repository *PostgresFinancialAccountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.FinancialAccount, error) {
	rows, err := repository.database.Query(ctx, getFinancialAccountsByUserIDQuery, userID)
	if err != nil {
		return nil, ports.ErrFinancialAccountRepositoryUnavailable
	}
	defer rows.Close()
	accounts := make([]*domain.FinancialAccount, 0)
	for rows.Next() {
		account, err := repository.scanAccount(rows)
		if err != nil {
			return nil, ports.ErrFinancialAccountRepositoryUnavailable
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, ports.ErrFinancialAccountRepositoryUnavailable
	}
	return accounts, nil
}

func (repository *PostgresFinancialAccountRepository) Update(ctx context.Context, account *domain.FinancialAccount) error {
	if account == nil {
		return ports.ErrFinancialAccountRepositoryUnavailable
	}
	_, err := repository.database.Exec(ctx, updateFinancialAccountQuery, account.ID(), account.Name(), account.Institution(), nullableTransactionCreditCardID(account.Last4()), account.OpeningBalanceCents(), account.CurrentBalanceCents(), nullableAccountInt64(account.CreditLimitCents()), nullableAccountInt(account.CutoffDay()), nullableAccountInt(account.PaymentDay()), account.Color(), account.UpdatedAt())
	if err != nil {
		return ports.ErrFinancialAccountRepositoryUnavailable
	}
	return nil
}

func (repository *PostgresFinancialAccountRepository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := repository.database.Exec(ctx, deleteFinancialAccountQuery, id, now)
	if err != nil {
		return ports.ErrFinancialAccountRepositoryUnavailable
	}
	return nil
}

func (repository *PostgresFinancialAccountRepository) scanAccount(row pgx.Row) (*domain.FinancialAccount, error) {
	var stored struct {
		id, userID, accountType, name, institution, color string
		last4                                             *string
		openingBalanceCents, currentBalanceCents          int64
		creditLimitCents                                  *int64
		cutoffDay, paymentDay                             *int
		createdAt, updatedAt                              time.Time
	}
	if err := row.Scan(&stored.id, &stored.userID, &stored.accountType, &stored.name, &stored.institution, &stored.last4, &stored.openingBalanceCents, &stored.currentBalanceCents, &stored.creditLimitCents, &stored.cutoffDay, &stored.paymentDay, &stored.color, &stored.createdAt, &stored.updatedAt); err != nil {
		return nil, err
	}
	return domain.RehydrateFinancialAccount(stored.id, stored.userID, domain.FinancialAccountType(stored.accountType), stored.name, stored.institution, stored.last4, stored.openingBalanceCents, stored.currentBalanceCents, stored.creditLimitCents, stored.cutoffDay, stored.paymentDay, stored.color, stored.createdAt, stored.updatedAt)
}

func nullableAccountInt(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableAccountInt64(value *int64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}
