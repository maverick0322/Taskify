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
	sqliteCreateFinancialAccountQuery = `
		INSERT INTO financial_accounts (id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, network, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	sqliteGetFinancialAccountByIDQuery = `
		SELECT id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, network, created_at, updated_at
		FROM financial_accounts
		WHERE id = ? AND deleted_at IS NULL
	`
	sqliteGetFinancialAccountsByUserIDQuery = `
		SELECT id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, network, created_at, updated_at
		FROM financial_accounts
		WHERE user_id = ? AND deleted_at IS NULL
		ORDER BY type ASC, created_at ASC
	`
	sqliteUpdateFinancialAccountQuery = `
		UPDATE financial_accounts
		SET name = ?, institution = ?, last4 = ?, opening_balance_cents = ?, current_balance_cents = ?, credit_limit_cents = ?, cutoff_day = ?, payment_day = ?, color = ?, network = ?, updated_at = ?
		WHERE id = ?
	`
	sqliteDeleteFinancialAccountQuery = `
		UPDATE financial_accounts SET deleted_at = ?, updated_at = ? WHERE id = ?
	`
)

type SQLiteFinancialAccountRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteFinancialAccountRepository(database *sql.DB, logger ports.Logger) ports.FinancialAccountRepository {
	return &SQLiteFinancialAccountRepository{database: database, logger: logger}
}

func (repository *SQLiteFinancialAccountRepository) Create(ctx context.Context, account *domain.FinancialAccount) error {
	if account == nil {
		return ports.ErrFinancialAccountRepositoryUnavailable
	}
	_, err := repository.database.ExecContext(ctx, sqliteCreateFinancialAccountQuery, account.ID(), account.UserID(), string(account.Type()), account.Name(), account.Institution(), nullableString(account.Last4()), account.OpeningBalanceCents(), account.CurrentBalanceCents(), nullableInt64(account.CreditLimitCents()), nullableInt(account.CutoffDay()), nullableInt(account.PaymentDay()), account.Color(), account.Network(), timeValue(account.CreatedAt()), timeValue(account.UpdatedAt()))
	if err == nil {
		return nil
	}
	if isSQLiteConstraintViolation(err) {
		return ports.ErrFinancialAccountAlreadyExists
	}
	repository.logger.Error("failed to create financial account", "error", err)
	return ports.ErrFinancialAccountRepositoryUnavailable
}

func (repository *SQLiteFinancialAccountRepository) GetByID(ctx context.Context, id string) (*domain.FinancialAccount, error) {
	account, err := repository.scanAccount(repository.database.QueryRowContext(ctx, sqliteGetFinancialAccountByIDQuery, id))
	if err == nil {
		return account, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrFinancialAccountNotFound
	}
	return nil, ports.ErrFinancialAccountRepositoryUnavailable
}

func (repository *SQLiteFinancialAccountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.FinancialAccount, error) {
	rows, err := repository.database.QueryContext(ctx, sqliteGetFinancialAccountsByUserIDQuery, userID)
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

func (repository *SQLiteFinancialAccountRepository) Update(ctx context.Context, account *domain.FinancialAccount) error {
	if account == nil {
		return ports.ErrFinancialAccountRepositoryUnavailable
	}
	_, err := repository.database.ExecContext(ctx, sqliteUpdateFinancialAccountQuery, account.Name(), account.Institution(), nullableString(account.Last4()), account.OpeningBalanceCents(), account.CurrentBalanceCents(), nullableInt64(account.CreditLimitCents()), nullableInt(account.CutoffDay()), nullableInt(account.PaymentDay()), account.Color(), account.Network(), timeValue(account.UpdatedAt()), account.ID())
	if err != nil {
		return ports.ErrFinancialAccountRepositoryUnavailable
	}
	return nil
}

func (repository *SQLiteFinancialAccountRepository) Delete(ctx context.Context, id string) error {
	deletedAt := timeValue(time.Now())
	_, err := repository.database.ExecContext(ctx, sqliteDeleteFinancialAccountQuery, deletedAt, deletedAt, id)
	if err != nil {
		return ports.ErrFinancialAccountRepositoryUnavailable
	}
	return nil
}

func (repository *SQLiteFinancialAccountRepository) scanAccount(row interface {
	Scan(dest ...interface{}) error
}) (*domain.FinancialAccount, error) {
	var stored struct {
		id, userID, accountType, name, institution, color, network string
		last4                                             sql.NullString
		openingBalanceCents, currentBalanceCents          int64
		creditLimitCents                                  sql.NullInt64
		cutoffDay, paymentDay                             sql.NullInt64
		createdAt, updatedAt                              time.Time
	}
	if err := row.Scan(&stored.id, &stored.userID, &stored.accountType, &stored.name, &stored.institution, &stored.last4, &stored.openingBalanceCents, &stored.currentBalanceCents, &stored.creditLimitCents, &stored.cutoffDay, &stored.paymentDay, &stored.color, &stored.network, &stored.createdAt, &stored.updatedAt); err != nil {
		return nil, err
	}
	return domain.RehydrateFinancialAccount(stored.id, stored.userID, domain.FinancialAccountType(stored.accountType), stored.name, stored.institution, scanNullableString(stored.last4), stored.openingBalanceCents, stored.currentBalanceCents, scanNullableInt64(stored.creditLimitCents), scanNullableInt(stored.cutoffDay), scanNullableInt(stored.paymentDay), stored.color, stored.network, stored.createdAt, stored.updatedAt)
}
