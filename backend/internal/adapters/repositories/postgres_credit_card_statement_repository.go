package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type PostgresCreditCardStatementRepository struct {
	database *pgxpool.Pool
	logger   ports.Logger
}

func NewPostgresCreditCardStatementRepository(database *pgxpool.Pool, logger ports.Logger) ports.CreditCardStatementRepository {
	return &PostgresCreditCardStatementRepository{database: database, logger: logger}
}

func (repository *PostgresCreditCardStatementRepository) Reconcile(ctx context.Context, userID, creditCardID string, statements []ports.CreditCardStatement, items []ports.CreditCardStatementItem, allocations []ports.CreditCardPaymentAllocation) error {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return statementPersistenceError("begin reconciliation", err)
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE credit_card_payment_allocations SET deleted_at = $1, updated_at = $1 WHERE deleted_at IS NULL AND statement_id IN (SELECT id FROM credit_card_statements WHERE user_id = $2 AND credit_account_id = $3) AND NOT (id = ANY($4::text[]))`, now, userID, creditCardID, allocationIDs(allocations)); err != nil {
		return statementPersistenceError("reset payment allocations", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE credit_card_statement_items SET deleted_at = $1, updated_at = $1 WHERE deleted_at IS NULL AND statement_id IN (SELECT id FROM credit_card_statements WHERE user_id = $2 AND credit_account_id = $3) AND NOT (id = ANY($4::text[]))`, now, userID, creditCardID, statementItemIDs(items)); err != nil {
		return statementPersistenceError("reset statement items", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE credit_card_statements SET deleted_at = $1, updated_at = $1 WHERE user_id = $2 AND credit_account_id = $3 AND deleted_at IS NULL AND NOT (id = ANY($4::text[]))`, now, userID, creditCardID, statementIDs(statements)); err != nil {
		return statementPersistenceError("reset statements", err)
	}
	for _, statement := range statements {
		_, err := tx.Exec(ctx, `INSERT INTO credit_card_statements (id, user_id, credit_account_id, cycle_start, cycle_end, payment_due_date, statement_amount_cents, paid_amount_cents, status, created_at, updated_at, deleted_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULL) ON CONFLICT(id) DO UPDATE SET cycle_start=EXCLUDED.cycle_start, cycle_end=EXCLUDED.cycle_end, payment_due_date=EXCLUDED.payment_due_date, statement_amount_cents=EXCLUDED.statement_amount_cents, paid_amount_cents=EXCLUDED.paid_amount_cents, status=EXCLUDED.status, updated_at=EXCLUDED.updated_at, deleted_at=NULL WHERE (credit_card_statements.cycle_start,credit_card_statements.cycle_end,credit_card_statements.payment_due_date,credit_card_statements.statement_amount_cents,credit_card_statements.paid_amount_cents,credit_card_statements.status,credit_card_statements.deleted_at) IS DISTINCT FROM (EXCLUDED.cycle_start,EXCLUDED.cycle_end,EXCLUDED.payment_due_date,EXCLUDED.statement_amount_cents,EXCLUDED.paid_amount_cents,EXCLUDED.status,NULL)`, statement.ID, statement.UserID, statement.CreditAccountID, statement.CycleStart, statement.CycleEnd, statement.PaymentDueDate, statement.StatementAmountCents, statement.PaidAmountCents, statement.Status, statement.CreatedAt, statement.UpdatedAt)
		if err != nil {
			return statementPersistenceError("upsert statement", err)
		}
	}
	for _, item := range items {
		_, err := tx.Exec(ctx, `INSERT INTO credit_card_statement_items (id,user_id,statement_id,transaction_id,amount_cents,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,$5,$6,$7,NULL) ON CONFLICT(id) DO UPDATE SET statement_id=EXCLUDED.statement_id, transaction_id=EXCLUDED.transaction_id, amount_cents=EXCLUDED.amount_cents, updated_at=EXCLUDED.updated_at, deleted_at=NULL WHERE (credit_card_statement_items.statement_id,credit_card_statement_items.transaction_id,credit_card_statement_items.amount_cents,credit_card_statement_items.deleted_at) IS DISTINCT FROM (EXCLUDED.statement_id,EXCLUDED.transaction_id,EXCLUDED.amount_cents,NULL)`, item.ID, item.UserID, item.StatementID, item.TransactionID, item.AmountCents, item.CreatedAt, item.UpdatedAt)
		if err != nil {
			return statementPersistenceError("upsert statement item", err)
		}
	}
	for _, allocation := range allocations {
		if err := execPostgresStatementAllocation(ctx, tx, allocation); err != nil {
			return statementPersistenceError("upsert payment allocation", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return statementPersistenceError("commit reconciliation", err)
	}
	return nil
}

func (repository *PostgresCreditCardStatementRepository) GetByCreditCardID(ctx context.Context, userID, creditCardID string) ([]ports.CreditCardStatement, error) {
	rows, err := repository.database.Query(ctx, `SELECT id,user_id,credit_account_id,cycle_start,cycle_end,payment_due_date,statement_amount_cents,paid_amount_cents,status,created_at,updated_at FROM credit_card_statements WHERE user_id=$1 AND credit_account_id=$2 AND deleted_at IS NULL ORDER BY payment_due_date ASC`, userID, creditCardID)
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

func (repository *PostgresCreditCardStatementRepository) ApplyPayment(ctx context.Context, payment *domain.Transaction, accountDeltas []ports.AccountBalanceDelta, ledgerEntries []ports.LedgerEntry, allocations []ports.CreditCardPaymentAllocation, statements []ports.CreditCardStatement) error {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return statementPersistenceError("begin card payment", err)
	}
	defer tx.Rollback(ctx)
	if _, err := execPostgresCreateTransaction(ctx, tx, payment); err != nil {
		return statementPersistenceError("insert payment transaction", err)
	}
	for _, delta := range accountDeltas {
		tag, err := tx.Exec(ctx, `UPDATE financial_accounts SET current_balance_cents=current_balance_cents+$1, updated_at=$2 WHERE id=$3 AND deleted_at IS NULL AND current_balance_cents+$1 >= 0`, delta.DeltaCents, time.Now().UTC(), delta.AccountID)
		if err != nil {
			return statementPersistenceError("update account balance", err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("%w: account %s was not updated", ports.ErrCreditCardPaymentConflict, delta.AccountID)
		}
	}
	for _, entry := range ledgerEntries {
		if _, err := tx.Exec(ctx, `INSERT INTO ledger_entries (id,user_id,account_id,transaction_id,amount_cents,entry_type,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, entry.ID, entry.UserID, entry.AccountID, entry.TransactionID, entry.AmountCents, entry.EntryType, entry.CreatedAt, entry.UpdatedAt); err != nil {
			return statementPersistenceError("insert ledger entry", err)
		}
	}
	allocatedByStatement := allocatedAmountsByStatement(allocations)
	for _, statement := range statements {
		expectedPaidAmount := statement.PaidAmountCents - allocatedByStatement[statement.ID]
		tag, err := tx.Exec(ctx, `UPDATE credit_card_statements SET paid_amount_cents=$1,status=$2,updated_at=$3,deleted_at=NULL WHERE id=$4 AND user_id=$5 AND paid_amount_cents=$6`, statement.PaidAmountCents, statement.Status, statement.UpdatedAt, statement.ID, statement.UserID, expectedPaidAmount)
		if err != nil {
			return statementPersistenceError("update paid statement", err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("%w: statement %s was not updated", ports.ErrCreditCardPaymentConflict, statement.ID)
		}
	}
	for _, allocation := range allocations {
		if err := execPostgresStatementAllocation(ctx, tx, allocation); err != nil {
			return statementPersistenceError("insert payment allocation", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return statementPersistenceError("commit card payment", err)
	}
	return nil
}

func execPostgresStatementAllocation(ctx context.Context, tx pgx.Tx, allocation ports.CreditCardPaymentAllocation) error {
	_, err := tx.Exec(ctx, `INSERT INTO credit_card_payment_allocations (id,user_id,statement_id,payment_transaction_id,amount_cents,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,$5,$6,$7,NULL) ON CONFLICT(id) DO UPDATE SET amount_cents=EXCLUDED.amount_cents,updated_at=EXCLUDED.updated_at,deleted_at=NULL WHERE (credit_card_payment_allocations.amount_cents,credit_card_payment_allocations.deleted_at) IS DISTINCT FROM (EXCLUDED.amount_cents,NULL)`, allocation.ID, allocation.UserID, allocation.StatementID, allocation.PaymentTransactionID, allocation.AmountCents, allocation.CreatedAt, allocation.UpdatedAt)
	return err
}
