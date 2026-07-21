package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type SQLiteCreditCardStatementRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteCreditCardStatementRepository(database *sql.DB, logger ports.Logger) ports.CreditCardStatementRepository {
	return &SQLiteCreditCardStatementRepository{database: database, logger: logger}
}

func (repository *SQLiteCreditCardStatementRepository) Reconcile(ctx context.Context, userID, creditCardID string, statements []ports.CreditCardStatement, items []ports.CreditCardStatementItem, allocations []ports.CreditCardPaymentAllocation) error {
	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return statementPersistenceError("begin reconciliation", err)
	}
	defer tx.Rollback()

	now := timeValue(time.Now().UTC())
	if err := softDeleteMissingSQLiteCardRows(ctx, tx, "credit_card_payment_allocations", allocationIDs(allocations), now, userID, creditCardID); err != nil {
		return statementPersistenceError("reset payment allocations", err)
	}
	if err := softDeleteMissingSQLiteCardRows(ctx, tx, "credit_card_statement_items", statementItemIDs(items), now, userID, creditCardID); err != nil {
		return statementPersistenceError("reset statement items", err)
	}
	if err := softDeleteMissingSQLiteStatements(ctx, tx, statementIDs(statements), now, userID, creditCardID); err != nil {
		return statementPersistenceError("reset statements", err)
	}

	for _, statement := range statements {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO credit_card_statements (id, user_id, credit_account_id, cycle_start, cycle_end, payment_due_date, statement_amount_cents, paid_amount_cents, status, created_at, updated_at, deleted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
			ON CONFLICT(id) DO UPDATE SET cycle_start = excluded.cycle_start, cycle_end = excluded.cycle_end, payment_due_date = excluded.payment_due_date, statement_amount_cents = excluded.statement_amount_cents, paid_amount_cents = excluded.paid_amount_cents, status = excluded.status, updated_at = excluded.updated_at, deleted_at = NULL
			WHERE credit_card_statements.cycle_start IS NOT excluded.cycle_start OR credit_card_statements.cycle_end IS NOT excluded.cycle_end OR credit_card_statements.payment_due_date IS NOT excluded.payment_due_date OR credit_card_statements.statement_amount_cents IS NOT excluded.statement_amount_cents OR credit_card_statements.paid_amount_cents IS NOT excluded.paid_amount_cents OR credit_card_statements.status IS NOT excluded.status OR credit_card_statements.deleted_at IS NOT NULL`,
			statement.ID, statement.UserID, statement.CreditAccountID, timeValue(statement.CycleStart), timeValue(statement.CycleEnd), timeValue(statement.PaymentDueDate), statement.StatementAmountCents, statement.PaidAmountCents, statement.Status, timeValue(statement.CreatedAt), timeValue(statement.UpdatedAt))
		if err != nil {
			return statementPersistenceError("upsert statement", err)
		}
	}
	for _, item := range items {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO credit_card_statement_items (id, user_id, statement_id, transaction_id, amount_cents, created_at, updated_at, deleted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NULL)
			ON CONFLICT(id) DO UPDATE SET statement_id = excluded.statement_id, transaction_id = excluded.transaction_id, amount_cents = excluded.amount_cents, updated_at = excluded.updated_at, deleted_at = NULL
			WHERE credit_card_statement_items.statement_id IS NOT excluded.statement_id OR credit_card_statement_items.transaction_id IS NOT excluded.transaction_id OR credit_card_statement_items.amount_cents IS NOT excluded.amount_cents OR credit_card_statement_items.deleted_at IS NOT NULL`,
			item.ID, item.UserID, item.StatementID, item.TransactionID, item.AmountCents, timeValue(item.CreatedAt), timeValue(item.UpdatedAt))
		if err != nil {
			return statementPersistenceError("upsert statement item", err)
		}
	}
	for _, allocation := range allocations {
		if err := execSQLiteStatementAllocation(ctx, tx, allocation); err != nil {
			return statementPersistenceError("upsert payment allocation", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return statementPersistenceError("commit reconciliation", err)
	}
	return nil
}

func (repository *SQLiteCreditCardStatementRepository) GetByCreditCardID(ctx context.Context, userID, creditCardID string) ([]ports.CreditCardStatement, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT id, user_id, credit_account_id, cycle_start, cycle_end, payment_due_date, statement_amount_cents, paid_amount_cents, status, created_at, updated_at FROM credit_card_statements WHERE user_id = ? AND credit_account_id = ? AND deleted_at IS NULL ORDER BY payment_due_date ASC`, userID, creditCardID)
	if err != nil {
		return nil, statementPersistenceError("query statements", err)
	}
	defer rows.Close()
	statements := make([]ports.CreditCardStatement, 0)
	for rows.Next() {
		var statement ports.CreditCardStatement
		if err := rows.Scan(&statement.ID, &statement.UserID, &statement.CreditAccountID, &statement.CycleStart, &statement.CycleEnd, &statement.PaymentDueDate, &statement.StatementAmountCents, &statement.PaidAmountCents, &statement.Status, &statement.CreatedAt, &statement.UpdatedAt); err != nil {
			return nil, statementPersistenceError("scan statement", err)
		}
		statements = append(statements, statement)
	}
	if err := rows.Err(); err != nil {
		return nil, statementPersistenceError("iterate statements", err)
	}
	return statements, nil
}

func (repository *SQLiteCreditCardStatementRepository) ApplyPayment(ctx context.Context, payment *domain.Transaction, accountDeltas []ports.AccountBalanceDelta, ledgerEntries []ports.LedgerEntry, allocations []ports.CreditCardPaymentAllocation, statements []ports.CreditCardStatement) error {
	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return statementPersistenceError("begin card payment", err)
	}
	defer tx.Rollback()
	if _, err := execSQLiteCreateTransaction(ctx, tx, payment); err != nil {
		return statementPersistenceError("insert payment transaction", err)
	}
	for _, delta := range accountDeltas {
		result, err := tx.ExecContext(ctx, `UPDATE financial_accounts SET current_balance_cents = current_balance_cents + ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL AND current_balance_cents + ? >= 0`, delta.DeltaCents, timeValue(time.Now().UTC()), delta.AccountID, delta.DeltaCents)
		if err != nil {
			return statementPersistenceError("update account balance", err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("%w: account %s was not updated", ports.ErrCreditCardPaymentConflict, delta.AccountID)
		}
	}
	for _, entry := range ledgerEntries {
		if _, err := tx.ExecContext(ctx, `INSERT INTO ledger_entries (id, user_id, account_id, transaction_id, amount_cents, entry_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, entry.ID, entry.UserID, entry.AccountID, entry.TransactionID, entry.AmountCents, entry.EntryType, timeValue(entry.CreatedAt), timeValue(entry.UpdatedAt)); err != nil {
			return statementPersistenceError("insert ledger entry", err)
		}
	}
	allocatedByStatement := allocatedAmountsByStatement(allocations)
	for _, statement := range statements {
		expectedPaidAmount := statement.PaidAmountCents - allocatedByStatement[statement.ID]
		result, err := tx.ExecContext(ctx, `UPDATE credit_card_statements SET paid_amount_cents = ?, status = ?, updated_at = ?, deleted_at = NULL WHERE id = ? AND user_id = ? AND paid_amount_cents = ?`, statement.PaidAmountCents, statement.Status, timeValue(statement.UpdatedAt), statement.ID, statement.UserID, expectedPaidAmount)
		if err != nil {
			return statementPersistenceError("update paid statement", err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("%w: statement %s was not updated", ports.ErrCreditCardPaymentConflict, statement.ID)
		}
	}
	for _, allocation := range allocations {
		if err := execSQLiteStatementAllocation(ctx, tx, allocation); err != nil {
			return statementPersistenceError("insert payment allocation", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return statementPersistenceError("commit card payment", err)
	}
	return nil
}

func execSQLiteStatementAllocation(ctx context.Context, tx *sql.Tx, allocation ports.CreditCardPaymentAllocation) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO credit_card_payment_allocations (id, user_id, statement_id, payment_transaction_id, amount_cents, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL)
		ON CONFLICT(id) DO UPDATE SET amount_cents = excluded.amount_cents, updated_at = excluded.updated_at, deleted_at = NULL
		WHERE credit_card_payment_allocations.amount_cents IS NOT excluded.amount_cents OR credit_card_payment_allocations.deleted_at IS NOT NULL`,
		allocation.ID, allocation.UserID, allocation.StatementID, allocation.PaymentTransactionID, allocation.AmountCents, timeValue(allocation.CreatedAt), timeValue(allocation.UpdatedAt))
	return err
}

func statementPersistenceError(stage string, err error) error {
	return fmt.Errorf("%w: %s: %v", ports.ErrCreditCardStatementUnavailable, stage, err)
}

func statementIDs(statements []ports.CreditCardStatement) []string {
	ids := make([]string, 0, len(statements))
	for _, statement := range statements {
		ids = append(ids, statement.ID)
	}
	return ids
}

func allocatedAmountsByStatement(allocations []ports.CreditCardPaymentAllocation) map[string]int64 {
	amounts := make(map[string]int64, len(allocations))
	for _, allocation := range allocations {
		amounts[allocation.StatementID] += allocation.AmountCents
	}
	return amounts
}

func statementItemIDs(items []ports.CreditCardStatementItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func allocationIDs(allocations []ports.CreditCardPaymentAllocation) []string {
	ids := make([]string, 0, len(allocations))
	for _, allocation := range allocations {
		ids = append(ids, allocation.ID)
	}
	return ids
}

func softDeleteMissingSQLiteCardRows(ctx context.Context, tx *sql.Tx, table string, ids []string, now interface{}, userID, creditCardID string) error {
	query := fmt.Sprintf(`UPDATE %s SET deleted_at = ?, updated_at = ? WHERE deleted_at IS NULL AND statement_id IN (SELECT id FROM credit_card_statements WHERE user_id = ? AND credit_account_id = ?)`, table)
	args := []interface{}{now, now, userID, creditCardID}
	if len(ids) > 0 {
		query += " AND id NOT IN (" + strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",") + ")"
		for _, id := range ids {
			args = append(args, id)
		}
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func softDeleteMissingSQLiteStatements(ctx context.Context, tx *sql.Tx, ids []string, now interface{}, userID, creditCardID string) error {
	query := `UPDATE credit_card_statements SET deleted_at = ?, updated_at = ? WHERE user_id = ? AND credit_account_id = ? AND deleted_at IS NULL`
	args := []interface{}{now, now, userID, creditCardID}
	if len(ids) > 0 {
		query += " AND id NOT IN (" + strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",") + ")"
		for _, id := range ids {
			args = append(args, id)
		}
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}
