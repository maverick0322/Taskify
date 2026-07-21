package services

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

var syncTestDatabaseCounter atomic.Int64

func TestSyncService_SyncOncePushesAndPullsRows(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)

	localUpdatedAt := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	remoteUpdatedAt := time.Date(2026, 6, 11, 10, 5, 0, 0, time.UTC)
	cycleAt := time.Date(2026, 6, 11, 10, 10, 0, 0, time.UTC)
	insertSyncUser(t, local, "local-user", "local@example.com", localUpdatedAt, nil)
	enqueueSyncOutbox(t, local, "users", "local-user")
	insertSyncUser(t, remote, "remote-user", "remote@example.com", remoteUpdatedAt, nil)

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.now = func() time.Time { return cycleAt }

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	assertSyncUserExists(t, local, "remote-user", "remote@example.com")
	assertSyncUserExists(t, remote, "local-user", "local@example.com")
	assertSyncState(t, local, remotePullSyncStateKey, cycleAt)
	assertOutboxEmpty(t, local)
}

func TestSyncService_SyncOnceLastWriteWinsByUpdatedAt(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)

	localUpdatedAt := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	remoteUpdatedAt := time.Date(2026, 6, 11, 10, 5, 0, 0, time.UTC)
	insertSyncUser(t, local, "user-1", "older@example.com", localUpdatedAt, nil)
	enqueueSyncOutbox(t, local, "users", "user-1")
	insertSyncUser(t, remote, "user-1", "newer@example.com", remoteUpdatedAt, nil)

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.now = func() time.Time { return time.Date(2026, 6, 11, 10, 10, 0, 0, time.UTC) }

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	assertSyncUserExists(t, local, "user-1", "newer@example.com")
	assertSyncUserExists(t, remote, "user-1", "newer@example.com")
}

func TestSyncService_SyncOnceReplicatesSoftDelete(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)

	deletedAt := time.Date(2026, 6, 11, 10, 5, 0, 0, time.UTC)
	insertSyncUser(t, local, "user-1", "deleted@example.com", deletedAt, &deletedAt)
	enqueueSyncOutbox(t, local, "users", "user-1")

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.now = func() time.Time { return time.Date(2026, 6, 11, 10, 10, 0, 0, time.UTC) }

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	var remoteDeletedAt sql.NullTime
	if err := remote.QueryRow("SELECT deleted_at FROM users WHERE id = ?", "user-1").Scan(&remoteDeletedAt); err != nil {
		t.Fatalf("failed to read remote deleted_at: %v", err)
	}
	if !remoteDeletedAt.Valid || !remoteDeletedAt.Time.Equal(deletedAt) {
		t.Fatalf("expected remote deleted_at %v, got %+v", deletedAt, remoteDeletedAt)
	}
}

func TestSyncService_SyncOnceRemoteFailureDoesNotAdvanceState(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)
	insertSyncUser(t, local, "local-user", "local@example.com", time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC), nil)
	enqueueSyncOutbox(t, local, "users", "local-user")
	remote.Close()

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.now = func() time.Time { return time.Date(2026, 6, 11, 10, 10, 0, 0, time.UTC) }

	if err := service.SyncOnce(context.Background()); err == nil {
		t.Fatal("expected sync error, got nil")
	}

	var count int
	if err := local.QueryRow("SELECT COUNT(*) FROM sync_state").Scan(&count); err != nil {
		t.Fatalf("failed to count sync state rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected sync_state to remain empty, got %d rows", count)
	}
	assertOutboxCount(t, local, 1)
}

func TestSyncService_SyncOncePullDoesNotEnqueueOutbox(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)

	remoteUpdatedAt := time.Date(2026, 6, 11, 10, 5, 0, 0, time.UTC)
	insertSyncUser(t, remote, "remote-user", "remote@example.com", remoteUpdatedAt, nil)

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.now = func() time.Time { return time.Date(2026, 6, 11, 10, 10, 0, 0, time.UTC) }

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	assertSyncUserExists(t, local, "remote-user", "remote@example.com")
	assertOutboxEmpty(t, local)
}

func TestSyncService_SyncOncePublishesEventWhenPullAppliesRows(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)

	remoteUpdatedAt := time.Date(2026, 6, 11, 10, 5, 0, 0, time.UTC)
	insertSyncUser(t, remote, "remote-user", "remote@example.com", remoteUpdatedAt, nil)

	eventHub := NewSyncEventHub()
	events, unsubscribe := eventHub.Subscribe()
	defer unsubscribe()

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.SetEventHub(eventHub)
	service.now = func() time.Time { return time.Date(2026, 6, 11, 10, 10, 0, 0, time.UTC) }

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	select {
	case eventName := <-events:
		if eventName != SyncUpdatedEvent {
			t.Fatalf("expected event %s, got %s", SyncUpdatedEvent, eventName)
		}
	case <-time.After(time.Second):
		t.Fatal("expected sync updated event")
	}
}

func TestSyncService_SyncOnceDoesNotPublishEventWhenPullIsEmpty(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)

	eventHub := NewSyncEventHub()
	events, unsubscribe := eventHub.Subscribe()
	defer unsubscribe()

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.SetEventHub(eventHub)
	service.now = func() time.Time { return time.Date(2026, 6, 11, 10, 10, 0, 0, time.UTC) }

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	select {
	case eventName := <-events:
		t.Fatalf("expected no event, got %s", eventName)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestSyncService_SyncOnceSyncsAccountPayablePayments(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)

	updatedAt := time.Date(2026, 6, 11, 10, 5, 0, 0, time.UTC)
	insertAccountPayablePayment(t, remote, "payment-1", updatedAt)

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.now = func() time.Time { return time.Date(2026, 6, 11, 10, 10, 0, 0, time.UTC) }

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	var amountCents int
	if err := local.QueryRow("SELECT amount_cents FROM account_payable_payments WHERE id = ?", "payment-1").Scan(&amountCents); err != nil {
		t.Fatalf("failed to query account payable payment: %v", err)
	}
	if amountCents != 12500 {
		t.Fatalf("expected amount 12500, got %d", amountCents)
	}
}

func openSyncTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:sync-test-%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", syncTestDatabaseCounter.Add(1))
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec(syncTestSchema); err != nil {
		t.Fatalf("failed to initialize sync test schema: %v", err)
	}

	return database
}

func insertSyncUser(t *testing.T, database *sql.DB, id, email string, updatedAt time.Time, deletedAt *time.Time) {
	t.Helper()

	var deletedAtValue interface{}
	if deletedAt != nil {
		deletedAtValue = *deletedAt
	}

	_, err := database.Exec(
		`INSERT INTO users (id, email, password_hash, first_name, last_name, birth_date, created_at, updated_at, deleted_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		email,
		"hashed-password-value",
		"Erick",
		"Lara",
		time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		updatedAt,
		updatedAt,
		deletedAtValue,
	)
	if err != nil {
		t.Fatalf("failed to insert sync user: %v", err)
	}
}

func assertSyncUserExists(t *testing.T, database *sql.DB, id, expectedEmail string) {
	t.Helper()

	var email string
	if err := database.QueryRow("SELECT email FROM users WHERE id = ?", id).Scan(&email); err != nil {
		t.Fatalf("failed to query user %s: %v", id, err)
	}
	if email != expectedEmail {
		t.Fatalf("expected email %s, got %s", expectedEmail, email)
	}
}

func enqueueSyncOutbox(t *testing.T, database *sql.DB, tableName, entityID string) {
	t.Helper()

	_, err := database.Exec(
		`INSERT INTO sync_outbox (id, table_name, entity_id, operation, status, created_at, updated_at)
		 VALUES (?, ?, ?, 'upsert', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT(table_name, entity_id) DO UPDATE SET status = 'pending', updated_at = CURRENT_TIMESTAMP`,
		tableName+":"+entityID,
		tableName,
		entityID,
	)
	if err != nil {
		t.Fatalf("failed to enqueue sync outbox row: %v", err)
	}
}

func insertAccountPayablePayment(t *testing.T, database *sql.DB, id string, updatedAt time.Time) {
	t.Helper()

	_, err := database.Exec(
		`INSERT INTO account_payable_payments (id, account_payable_id, user_id, due_date, paid_at, amount_cents, concept, category, created_transaction_id, created_at, updated_at, deleted_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		id,
		"payable-1",
		"user-1",
		updatedAt,
		updatedAt,
		12500,
		"Pago CFE",
		"Servicios",
		"transaction-1",
		updatedAt,
		updatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert account payable payment: %v", err)
	}
}

func assertSyncState(t *testing.T, database *sql.DB, key string, expected time.Time) {
	t.Helper()

	var lastSyncAt time.Time
	if err := database.QueryRow("SELECT last_successful_sync_at FROM sync_state WHERE key = ?", key).Scan(&lastSyncAt); err != nil {
		t.Fatalf("failed to query sync state: %v", err)
	}
	if !lastSyncAt.Equal(expected) {
		t.Fatalf("expected sync state %v, got %v", expected, lastSyncAt)
	}
}

func assertOutboxEmpty(t *testing.T, database *sql.DB) {
	t.Helper()
	assertOutboxCount(t, database, 0)
}

func assertOutboxCount(t *testing.T, database *sql.DB, expected int) {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM sync_outbox").Scan(&count); err != nil {
		t.Fatalf("failed to count sync outbox: %v", err)
	}
	if count != expected {
		t.Fatalf("expected sync_outbox count %d, got %d", expected, count)
	}
}

const syncTestSchema = `
CREATE TABLE users (
	id TEXT PRIMARY KEY,
	email TEXT,
	password_hash TEXT,
	first_name TEXT,
	last_name TEXT,
	birth_date DATETIME,
	avatar_local_path TEXT NULL,
	avatar_url TEXT NULL,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE boards (id TEXT PRIMARY KEY, user_id TEXT, name TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE columns (id TEXT PRIMARY KEY, board_id TEXT, name TEXT, color TEXT, position INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE tasks (id TEXT PRIMARY KEY, user_id TEXT, board_id TEXT, column_id TEXT, title TEXT, description TEXT, status TEXT, priority TEXT, due_date DATETIME NULL, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE credit_cards (id TEXT PRIMARY KEY, user_id TEXT, name TEXT, bank TEXT, last4 TEXT, cutoff_day INTEGER, payment_day INTEGER, limit_cents INTEGER, color TEXT, network TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE financial_accounts (id TEXT PRIMARY KEY, user_id TEXT, type TEXT, name TEXT, institution TEXT, last4 TEXT NULL, opening_balance_cents INTEGER, current_balance_cents INTEGER, credit_limit_cents INTEGER NULL, cutoff_day INTEGER NULL, payment_day INTEGER NULL, color TEXT, network TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE transactions (id TEXT PRIMARY KEY, user_id TEXT, credit_card_id TEXT, payment_account_id TEXT NULL, destination_account_id TEXT NULL, type TEXT, concept TEXT, category TEXT, amount_cents INTEGER, date DATETIME, status TEXT, msi INTEGER NULL, installment_number INTEGER NULL, installment_count INTEGER NULL, is_historical BOOLEAN NOT NULL DEFAULT 0, recurrence TEXT, recurrence_limit INTEGER NULL, last_paid_at DATETIME NULL, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE ledger_entries (id TEXT PRIMARY KEY, user_id TEXT, account_id TEXT, transaction_id TEXT, amount_cents INTEGER, entry_type TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE credit_card_statements (id TEXT PRIMARY KEY, user_id TEXT, credit_account_id TEXT, cycle_start DATETIME, cycle_end DATETIME, payment_due_date DATETIME, statement_amount_cents INTEGER, paid_amount_cents INTEGER, status TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE credit_card_statement_items (id TEXT PRIMARY KEY, user_id TEXT, statement_id TEXT, transaction_id TEXT, amount_cents INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE credit_card_payment_allocations (id TEXT PRIMARY KEY, user_id TEXT, statement_id TEXT, payment_transaction_id TEXT, amount_cents INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE account_payable_payments (id TEXT PRIMARY KEY, account_payable_id TEXT, user_id TEXT, due_date DATETIME, paid_at DATETIME, amount_cents INTEGER, concept TEXT, category TEXT, created_transaction_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE notifications (id TEXT PRIMARY KEY, user_id TEXT, title TEXT, message TEXT, is_read BOOLEAN, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE sync_state (key TEXT PRIMARY KEY, last_successful_sync_at DATETIME NOT NULL, updated_at DATETIME NOT NULL);
CREATE TABLE sync_runtime_flags (key TEXT PRIMARY KEY, value TEXT NOT NULL);
INSERT INTO sync_runtime_flags (key, value) VALUES ('suppress_outbox', '0');
CREATE TABLE sync_outbox (
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
	UNIQUE(table_name, entity_id)
);
`
