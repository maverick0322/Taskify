package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/adapters/logging"
	"github.com/maverick0322/taskify/backend/internal/adapters/repositories"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"log/slog"
	_ "modernc.org/sqlite"
)

func TestInitializeSQLiteSchema_LegacyTransactionsMigrationRecreatesOutboxTriggers(t *testing.T) {
	ctx := context.Background()
	database := openCmdTestSQLiteDatabase(t)

	if _, err := database.ExecContext(ctx, `
		CREATE TABLE transactions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			credit_card_id TEXT,
			payment_account_id TEXT,
			destination_account_id TEXT,
			type TEXT NOT NULL,
			concept TEXT NOT NULL,
			category TEXT NOT NULL,
			amount_cents INTEGER NOT NULL,
			date DATETIME NOT NULL,
			status TEXT NOT NULL,
			msi INTEGER NULL,
			installment_number INTEGER NULL,
			installment_count INTEGER NULL,
			is_historical BOOLEAN NOT NULL DEFAULT 0,
			recurrence TEXT NOT NULL DEFAULT 'once',
			recurrence_limit INTEGER NULL,
			last_paid_at DATETIME NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME NULL
		)
	`); err != nil {
		t.Fatalf("failed to create legacy transactions table: %v", err)
	}

	if err := initializeSQLiteSchema(ctx, database); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertSQLiteTriggerExists(t, database, "trg_transactions_sync_outbox_insert")
	assertSQLiteTriggerExists(t, database, "trg_transactions_sync_outbox_update")
}

func TestInitializeSQLiteSchema_TransactionTriggersWriteToOutbox(t *testing.T) {
	ctx := context.Background()
	database := openCmdTestSQLiteDatabase(t)

	if err := initializeSQLiteSchema(ctx, database); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	insertUserForSQLiteSyncTests(t, ctx, database, "user-1")

	createdAt := timeValue(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	if _, err := database.ExecContext(ctx, `
		INSERT INTO transactions (
			id, user_id, credit_card_id, payment_account_id, destination_account_id,
			type, concept, category, amount_cents, date, status, msi, installment_number,
			installment_count, is_historical, recurrence, recurrence_limit, last_paid_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"tx-1", "user-1", nil, nil, nil,
		"EXPENSE", "Internet", "Servicios", int64(89900), createdAt, "PAID", nil, nil,
		nil, false, "once", nil, nil, createdAt, createdAt,
	); err != nil {
		t.Fatalf("failed to insert transaction: %v", err)
	}

	assertOutboxRow(t, database, "transactions", "tx-1")

	updatedAt := timeValue(time.Date(2026, 6, 30, 13, 0, 0, 0, time.UTC))
	if _, err := database.ExecContext(ctx, `
		UPDATE transactions
		SET concept = ?, updated_at = ?
		WHERE id = ?
	`, "Internet Fibra", updatedAt, "tx-1"); err != nil {
		t.Fatalf("failed to update transaction: %v", err)
	}

	assertOutboxRow(t, database, "transactions", "tx-1")
}

func TestInitializeSQLiteSchema_FinancialCreateManyWithLedgerWritesCompleteOutbox(t *testing.T) {
	ctx := context.Background()
	database := openCmdTestSQLiteDatabase(t)

	if err := initializeSQLiteSchema(ctx, database); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	logger := logging.NewSlogLogger(slog.New(slog.NewTextHandler(testWriter{t}, nil)))
	userRepository := repositories.NewSQLiteUserRepository(database, logger)
	accountRepository := repositories.NewSQLiteFinancialAccountRepository(database, logger)
	transactionRepository := repositories.NewSQLiteTransactionRepository(database, logger)

	profile, err := domain.NewUserProfile("Erick", "Vazquez", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to create user profile: %v", err)
	}
	user, err := domain.NewUser("user-1", "erick@example.com", "secureHash1234!", profile)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := userRepository.Save(ctx, user); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	account, err := domain.NewFinancialAccount(
		"account-1",
		user.ID(),
		domain.FinancialAccountTypeDebitCard,
		"Cuenta principal",
		"BBVA",
		stringPtr("1234"),
		0,
		150000,
		nil,
		nil,
		nil,
		"from-zinc-700 to-zinc-950",
		"Visa",
	)
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}
	if err := accountRepository.Create(ctx, account); err != nil {
		t.Fatalf("failed to save account: %v", err)
	}

	transaction, err := domain.NewTransaction(
		"tx-1",
		user.ID(),
		domain.TransactionTypeExpense,
		"Supermercado",
		"Despensa",
		25000,
		time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC),
		domain.TransactionStatusPaid,
		nil,
		nil,
		domain.TransactionRecurrenceOnce,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}
	transaction.SetAccountingDetails(stringPtr(account.ID()), nil, nil, nil)

	entryTime := time.Date(2026, 6, 30, 9, 0, 1, 0, time.UTC)
	if err := transactionRepository.CreateManyWithLedger(
		ctx,
		[]*domain.Transaction{transaction},
		[]ports.AccountBalanceDelta{{AccountID: account.ID(), DeltaCents: -25000}},
		[]ports.LedgerEntry{{
			ID:            "entry-1",
			UserID:        user.ID(),
			AccountID:     account.ID(),
			TransactionID: transaction.ID(),
			AmountCents:   -25000,
			EntryType:     string(domain.TransactionTypeExpense),
			CreatedAt:     entryTime,
			UpdatedAt:     entryTime,
		}},
	); err != nil {
		t.Fatalf("failed to create transaction with ledger: %v", err)
	}

	assertOutboxRow(t, database, "transactions", "tx-1")
	assertOutboxRow(t, database, "financial_accounts", "account-1")
	assertOutboxRow(t, database, "ledger_entries", "entry-1")
}

func openCmdTestSQLiteDatabase(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", fmt.Sprintf("file:cmd-sqlite-test-%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func insertUserForSQLiteSyncTests(t *testing.T, ctx context.Context, database *sql.DB, userID string) {
	t.Helper()

	timestamp := timeValue(time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC))
	if _, err := database.ExecContext(ctx, `
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, birth_date,
			avatar_local_path, avatar_url, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, userID+"@example.com", "hash", "User", "One", timestamp, "", "", timestamp, timestamp); err != nil {
		t.Fatalf("failed to insert user %s: %v", userID, err)
	}
}

func assertSQLiteTriggerExists(t *testing.T, database *sql.DB, triggerName string) {
	t.Helper()

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name = ?`, triggerName).Scan(&count); err != nil {
		t.Fatalf("failed to query trigger %s: %v", triggerName, err)
	}
	if count != 1 {
		t.Fatalf("expected trigger %s to exist, got count %d", triggerName, count)
	}
}

func assertOutboxRow(t *testing.T, database *sql.DB, tableName, entityID string) {
	t.Helper()

	var operation string
	err := database.QueryRow(`
		SELECT operation
		FROM sync_outbox
		WHERE table_name = ? AND entity_id = ?
	`, tableName, entityID).Scan(&operation)
	if err != nil {
		t.Fatalf("expected outbox row for %s.%s, got error: %v", tableName, entityID, err)
	}
	if strings.TrimSpace(operation) != "upsert" {
		t.Fatalf("expected outbox operation upsert for %s.%s, got %q", tableName, entityID, operation)
	}
}

func stringPtr(value string) *string {
	return &value
}

func timeValue(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

type testWriter struct {
	t *testing.T
}

func (writer testWriter) Write(payload []byte) (int, error) {
	writer.t.Log(strings.TrimSpace(string(payload)))
	return len(payload), nil
}
