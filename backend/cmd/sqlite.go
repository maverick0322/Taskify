package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	sqliteStartupTimeout = 5 * time.Second
	sqliteAppFolderName  = "Taskify"
	sqliteDatabaseName   = "taskify.db"
)

func openLocalSQLiteDatabase(ctx context.Context) (*sql.DB, error) {
	databasePath, err := localSQLiteDatabasePath()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(databasePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create sqlite database directory: %w", err)
	}

	database, err := sql.Open("sqlite", sqliteDSN(databasePath))
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to connect to sqlite database: %w", err)
	}

	if err := initializeSQLiteSchema(ctx, database); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

func localSQLiteDatabasePath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user config directory: %w", err)
	}

	return filepath.Join(userConfigDir, sqliteAppFolderName, sqliteDatabaseName), nil
}

func sqliteDSN(databasePath string) string {
	cleanPath := filepath.ToSlash(databasePath)
	query := url.Values{}
	query.Set("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "busy_timeout(5000)")

	return fmt.Sprintf("file:///%s?%s", cleanPath, query.Encode())
}

func initializeSQLiteSchema(ctx context.Context, database *sql.DB) error {
	if _, err := database.ExecContext(ctx, sqliteSchema); err != nil {
		return fmt.Errorf("failed to initialize sqlite schema: %w", err)
	}

	if err := ensureSQLiteSyncMetadata(ctx, database); err != nil {
		return err
	}
	if err := ensureSQLiteOutbox(ctx, database); err != nil {
		return err
	}
	if err := ensureSQLiteAvatarStorage(ctx, database); err != nil {
		return err
	}

	if err := ensureSQLiteTransactionsCompletedStatus(ctx, database); err != nil {
		return err
	}
	if err := backfillSQLiteCreditCardFinancialAccounts(ctx, database); err != nil {
		return err
	}

	return nil
}

func backfillSQLiteCreditCardFinancialAccounts(ctx context.Context, database *sql.DB) error {
	_, err := database.ExecContext(ctx, `
		INSERT INTO financial_accounts (id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, network, created_at, updated_at, deleted_at)
		SELECT id, user_id, 'CREDIT_CARD', name, bank, last4, 0, 0, limit_cents, cutoff_day, payment_day, color, network, created_at, updated_at, deleted_at
		FROM credit_cards
		WHERE deleted_at IS NULL
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			institution = excluded.institution,
			last4 = excluded.last4,
			credit_limit_cents = excluded.credit_limit_cents,
			cutoff_day = excluded.cutoff_day,
			payment_day = excluded.payment_day,
			color = excluded.color,
			network = excluded.network,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return fmt.Errorf("failed to backfill credit card financial accounts: %w", err)
	}
	return nil
}

func ensureSQLiteTransactionsCompletedStatus(ctx context.Context, database *sql.DB) error {
	var createSQL string
	err := database.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'transactions'").Scan(&createSQL)
	if err != nil {
		return fmt.Errorf("failed to inspect sqlite transactions schema: %w", err)
	}
	if strings.Contains(createSQL, "'COMPLETED'") && strings.Contains(createSQL, "'DEBT_PAYMENT'") {
		return nil
	}

	statements := []string{
		"ALTER TABLE transactions RENAME TO transactions_legacy",
		`CREATE TABLE transactions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			credit_card_id TEXT REFERENCES credit_cards(id) ON DELETE SET NULL,
			payment_account_id TEXT REFERENCES financial_accounts(id) ON DELETE SET NULL,
			destination_account_id TEXT REFERENCES financial_accounts(id) ON DELETE SET NULL,
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
			deleted_at DATETIME NULL,
			CONSTRAINT chk_transactions_type CHECK (type IN ('INCOME', 'EXPENSE', 'DEBT_PAYMENT', 'TRANSFER')),
			CONSTRAINT chk_transactions_concept_not_empty CHECK (length(trim(concept)) > 0),
			CONSTRAINT chk_transactions_category_not_empty CHECK (length(trim(category)) > 0),
			CONSTRAINT chk_transactions_amount_positive CHECK (amount_cents > 0),
			CONSTRAINT chk_transactions_date_not_zero CHECK (date > '0001-01-01 00:00:00+00:00'),
			CONSTRAINT chk_transactions_status CHECK (status IN ('PAID', 'PENDING', 'COMPLETED')),
			CONSTRAINT chk_transactions_msi_positive CHECK (msi IS NULL OR msi >= 1),
			CONSTRAINT chk_transactions_installment_number_positive CHECK (installment_number IS NULL OR installment_number >= 1),
			CONSTRAINT chk_transactions_installment_count_positive CHECK (installment_count IS NULL OR installment_count >= 1),
			CONSTRAINT chk_transactions_recurrence CHECK (recurrence IN ('once', 'monthly', 'quarterly', 'biannual', 'annual')),
			CONSTRAINT chk_transactions_recurrence_limit_non_negative CHECK (recurrence_limit IS NULL OR recurrence_limit >= 0),
			CONSTRAINT chk_transactions_created_at_not_zero CHECK (created_at > '0001-01-01 00:00:00+00:00'),
			CONSTRAINT chk_transactions_updated_at_not_zero CHECK (updated_at > '0001-01-01 00:00:00+00:00')
		)`,
		`INSERT INTO transactions (id, user_id, credit_card_id, payment_account_id, destination_account_id, type, concept, category, amount_cents, date, status, msi, installment_number, installment_count, is_historical, recurrence, recurrence_limit, last_paid_at, created_at, updated_at, deleted_at)
		 SELECT id, user_id, credit_card_id, NULL, NULL, type, concept, category, amount_cents, date, status, msi, NULL, NULL, 0, recurrence, recurrence_limit, last_paid_at, created_at, updated_at, deleted_at
		 FROM transactions_legacy`,
		"DROP TABLE transactions_legacy",
		"CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_user_id_credit_card_id ON transactions(user_id, credit_card_id)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_user_id_date ON transactions(user_id, date DESC)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_user_id_status ON transactions(user_id, status)",
	}

	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start sqlite transactions schema migration: %w", err)
	}
	defer tx.Rollback()

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("failed to migrate sqlite transactions schema: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit sqlite transactions schema migration: %w", err)
	}

	return nil
}

func ensureSQLiteSyncMetadata(ctx context.Context, database *sql.DB) error {
	for _, table := range []string{"users", "boards", "columns", "tasks", "credit_cards", "transactions", "financial_accounts", "ledger_entries", "credit_card_statements", "account_payable_payments", "notifications"} {
		if err := ensureSQLiteColumn(ctx, database, table, "updated_at", "DATETIME DEFAULT CURRENT_TIMESTAMP"); err != nil {
			return err
		}
		if err := ensureSQLiteColumn(ctx, database, table, "deleted_at", "DATETIME NULL"); err != nil {
			return err
		}
		if _, err := database.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL", table)); err != nil {
			return fmt.Errorf("failed to backfill sqlite updated_at for %s: %w", table, err)
		}
	}
	if err := ensureSQLiteColumn(ctx, database, "columns", "color", "TEXT NOT NULL DEFAULT 'slate'"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "credit_cards", "network", "TEXT NOT NULL DEFAULT 'Visa'"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "financial_accounts", "network", "TEXT NOT NULL DEFAULT 'Visa'"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "tasks", "column_id", "TEXT REFERENCES columns(id) ON DELETE SET NULL"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "users", "avatar_local_path", "TEXT NULL"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "users", "avatar_url", "TEXT NULL"); err != nil {
		return err
	}
	if _, err := database.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_tasks_user_id_column_id ON tasks(user_id, column_id)"); err != nil {
		return fmt.Errorf("failed to create sqlite task column index: %w", err)
	}
	if err := ensureSQLiteColumn(ctx, database, "transactions", "recurrence", "TEXT NOT NULL DEFAULT 'once'"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "transactions", "recurrence_limit", "INTEGER NULL"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "transactions", "last_paid_at", "DATETIME NULL"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "transactions", "payment_account_id", "TEXT REFERENCES financial_accounts(id) ON DELETE SET NULL"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "transactions", "destination_account_id", "TEXT REFERENCES financial_accounts(id) ON DELETE SET NULL"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "transactions", "installment_number", "INTEGER NULL"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "transactions", "installment_count", "INTEGER NULL"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(ctx, database, "transactions", "is_historical", "BOOLEAN NOT NULL DEFAULT 0"); err != nil {
		return err
	}

	return nil
}

func ensureSQLiteOutbox(ctx context.Context, database *sql.DB) error {
	_, err := database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sync_runtime_flags (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		INSERT INTO sync_runtime_flags (key, value)
		VALUES ('suppress_outbox', '0')
		ON CONFLICT(key) DO NOTHING;

		CREATE TABLE IF NOT EXISTS sync_outbox (
			id TEXT PRIMARY KEY,
			table_name TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			operation TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT NULL,
			next_attempt_at DATETIME NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT uq_sync_outbox_entity UNIQUE (table_name, entity_id)
		);
		CREATE INDEX IF NOT EXISTS idx_sync_outbox_status_next_attempt ON sync_outbox(status, next_attempt_at, updated_at);
	`)
	if err != nil {
		return fmt.Errorf("failed to initialize sqlite sync outbox: %w", err)
	}

	for _, table := range []string{"users", "boards", "columns", "tasks", "credit_cards", "financial_accounts", "transactions", "ledger_entries", "credit_card_statements", "account_payable_payments", "notifications"} {
		if err := ensureSQLiteOutboxTriggers(ctx, database, table); err != nil {
			return err
		}
	}

	return nil
}

func ensureSQLiteOutboxTriggers(ctx context.Context, database *sql.DB, table string) error {
	_, err := database.ExecContext(ctx, fmt.Sprintf(`
		CREATE TRIGGER IF NOT EXISTS trg_%[1]s_sync_outbox_insert
		AFTER INSERT ON %[1]s
		WHEN (SELECT value FROM sync_runtime_flags WHERE key = 'suppress_outbox') != '1'
		BEGIN
			INSERT INTO sync_outbox (id, table_name, entity_id, operation, status, attempts, last_error, next_attempt_at, created_at, updated_at)
			VALUES ('%[1]s:' || NEW.id, '%[1]s', NEW.id, 'upsert', 'pending', 0, NULL, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT(table_name, entity_id) DO UPDATE SET
				operation = 'upsert',
				status = 'pending',
				last_error = NULL,
				next_attempt_at = NULL,
				updated_at = CURRENT_TIMESTAMP;
		END;

		CREATE TRIGGER IF NOT EXISTS trg_%[1]s_sync_outbox_update
		AFTER UPDATE ON %[1]s
		WHEN (SELECT value FROM sync_runtime_flags WHERE key = 'suppress_outbox') != '1'
		BEGIN
			INSERT INTO sync_outbox (id, table_name, entity_id, operation, status, attempts, last_error, next_attempt_at, created_at, updated_at)
			VALUES ('%[1]s:' || NEW.id, '%[1]s', NEW.id, 'upsert', 'pending', 0, NULL, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT(table_name, entity_id) DO UPDATE SET
				operation = 'upsert',
				status = 'pending',
				last_error = NULL,
				next_attempt_at = NULL,
				updated_at = CURRENT_TIMESTAMP;
		END;
	`, table))
	if err != nil {
		return fmt.Errorf("failed to initialize sqlite sync outbox triggers for %s: %w", table, err)
	}
	return nil
}

func ensureSQLiteAvatarStorage(ctx context.Context, database *sql.DB) error {
	_, err := database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS storage_sync_jobs (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			local_path TEXT NOT NULL,
			bucket TEXT NOT NULL,
			object_key TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT uq_storage_sync_jobs_entity UNIQUE (entity_type, entity_id)
		);
		CREATE INDEX IF NOT EXISTS idx_storage_sync_jobs_status ON storage_sync_jobs(status, updated_at);
	`)
	if err != nil {
		return fmt.Errorf("failed to initialize sqlite avatar storage metadata: %w", err)
	}
	return nil
}

func ensureSQLiteColumn(ctx context.Context, database *sql.DB, table, column, definition string) error {
	exists, err := sqliteColumnExists(ctx, database, table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if _, err := database.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
		return fmt.Errorf("failed to add sqlite column %s.%s: %w", table, column, err)
	}

	return nil
}

func sqliteColumnExists(ctx context.Context, database *sql.DB, table, column string) (bool, error) {
	rows, err := database.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("failed to inspect sqlite table %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("failed to scan sqlite table info for %s: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("failed to iterate sqlite table info for %s: %w", table, err)
	}

	return false, nil
}
