package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestRemoteSyncService_PullChangesFiltersByUser(t *testing.T) {
	database := openSyncTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)
	insertSyncUser(t, database, "user-2", "user2@example.com", time.Date(2025, 6, 26, 10, 5, 0, 0, time.UTC), nil)
	insertBoardRow(t, database, "board-1", "user-1", "Personal", time.Date(2025, 6, 26, 12, 0, 0, 0, time.UTC))
	insertBoardRow(t, database, "board-2", "user-2", "Team", time.Date(2025, 6, 26, 13, 0, 0, 0, time.UTC))

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	service.now = func() time.Time { return time.Date(2025, 6, 26, 14, 0, 0, 0, time.UTC) }

	result, err := service.PullChanges(context.Background(), "user-1", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("expected pull success, got %v", err)
	}

	if len(result.Changes) != 2 {
		t.Fatalf("expected 2 changes for user-1, got %d", len(result.Changes))
	}
	for _, change := range result.Changes {
		if change.Table == "boards" && syncStringValue(change.Values[1]) != "user-1" {
			t.Fatalf("expected board to belong to user-1, got %v", change.Values[1])
		}
		if change.Table == "users" && syncStringValue(change.Values[0]) != "user-1" {
			t.Fatalf("expected user row for user-1, got %v", change.Values[0])
		}
	}
	if !result.Cursor.Equal(service.now()) {
		t.Fatalf("expected cursor %v, got %v", service.now(), result.Cursor)
	}
}

func TestRemoteSyncService_PushChangesRejectsForeignUserRows(t *testing.T) {
	database := openSyncTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	_, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "boards",
			Values: []interface{}{
				"board-1",
				"user-2",
				"Invalid",
				timeValue(time.Now().UTC()),
				timeValue(time.Now().UTC()),
				nil,
			},
		},
	})
	if err == nil {
		t.Fatal("expected foreign row rejection, got nil")
	}
}

func TestRemoteSyncService_PushChangesUpsertsOwnedRows(t *testing.T) {
	database := openSyncTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	updatedAt := time.Date(2025, 6, 26, 15, 0, 0, 0, time.UTC)

	applied, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "boards",
			Values: []interface{}{
				"board-1",
				"user-1",
				"Owned Board",
				timeValue(updatedAt),
				timeValue(updatedAt),
				nil,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected push success, got %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied change, got %d", applied)
	}

	assertRemoteSyncBoardName(t, database, "board-1", "Owned Board")
}

func TestRemoteSyncService_PushChangesReordersTaskHierarchy(t *testing.T) {
	database := openRemotePushFKTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	updatedAt := time.Date(2025, 6, 26, 15, 0, 0, 0, time.UTC)

	applied, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "tasks",
			Values: []interface{}{
				"task-1", "user-1", "board-1", "column-1", "Tarea", "Desc", "todo", "high", nil,
				timeValue(updatedAt), timeValue(updatedAt), nil,
			},
		},
		{
			Table: "columns",
			Values: []interface{}{
				"column-1", "board-1", "Por hacer", "blue", int64(0),
				timeValue(updatedAt), timeValue(updatedAt), nil,
			},
		},
		{
			Table: "boards",
			Values: []interface{}{
				"board-1", "user-1", "Owned Board",
				timeValue(updatedAt), timeValue(updatedAt), nil,
			},
		},
	})

	if err != nil {
		t.Fatalf("expected push success, got %v", err)
	}
	if applied != 3 {
		t.Fatalf("expected 3 applied changes, got %d", applied)
	}
	assertRemoteSyncTaskExists(t, database, "task-1")
}

func TestRemoteSyncService_PushChangesReordersCreditCardHierarchy(t *testing.T) {
	database := openRemotePushFKTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	updatedAt := time.Date(2025, 6, 26, 15, 0, 0, 0, time.UTC)

	applied, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "transactions",
			Values: []interface{}{
				"tx-1", "user-1", "card-1", nil, nil, "EXPENSE", "Compra", "Gastos", int64(2500),
				timeValue(updatedAt), "PAID", nil, nil, nil, false, "once", nil, nil,
				timeValue(updatedAt), timeValue(updatedAt), nil,
			},
		},
		{
			Table: "credit_cards",
			Values: []interface{}{
				"card-1", "user-1", "Tarjeta Oro", "Banco Uno", "4242", int64(20), int64(10), int64(500000), "gold", "Visa",
				timeValue(updatedAt), timeValue(updatedAt), nil,
			},
		},
	})

	if err != nil {
		t.Fatalf("expected push success, got %v", err)
	}
	if applied != 2 {
		t.Fatalf("expected 2 applied changes, got %d", applied)
	}
	assertRemoteSyncTransactionExists(t, database, "tx-1")
}

func TestRemoteSyncService_PushChangesFailsWhenParentIsMissingWithDetailedContext(t *testing.T) {
	database := openRemotePushFKTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	updatedAt := time.Date(2025, 6, 26, 15, 0, 0, 0, time.UTC)

	_, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "transactions",
			Values: []interface{}{
				"tx-1", "user-1", "card-missing", nil, nil, "EXPENSE", "Compra", "Gastos", int64(2500),
				timeValue(updatedAt), "PAID", nil, nil, nil, false, "once", nil, nil,
				timeValue(updatedAt), timeValue(updatedAt), nil,
			},
		},
	})

	if err == nil {
		t.Fatal("expected push failure, got nil")
	}
	if !strings.Contains(err.Error(), "change[0] transactions(tx-1)") {
		t.Fatalf("expected detailed change context, got %v", err)
	}
}

func TestRemoteSyncService_PushChangesRejectsForeignUserRowsWithDetailedContext(t *testing.T) {
	database := openSyncTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	_, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "boards",
			Values: []interface{}{
				"board-1",
				"user-2",
				"Invalid",
				timeValue(time.Now().UTC()),
				timeValue(time.Now().UTC()),
				nil,
			},
		},
	})
	if err == nil {
		t.Fatal("expected foreign row rejection, got nil")
	}
	if !strings.Contains(err.Error(), "change[0] boards(board-1)") {
		t.Fatalf("expected detailed change context, got %v", err)
	}
}

func TestRemoteSyncService_PushChangesRejectsInvalidValueCountWithDetailedContext(t *testing.T) {
	database := openSyncTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	_, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "boards",
			Values: []interface{}{
				"board-1",
				"user-1",
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid value count rejection, got nil")
	}
	if !strings.Contains(err.Error(), "change[0] boards(board-1)") {
		t.Fatalf("expected detailed change context, got %v", err)
	}
}

func TestRemoteSyncService_PushChangesPublishesRealtimeUpdateAfterCommit(t *testing.T) {
	database := openSyncTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	hub := NewUserRealtimeHub()
	events, unsubscribe := hub.Subscribe("user-1")
	defer unsubscribe()

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	service.SetRealtimeHub(hub)
	updatedAt := time.Date(2025, 6, 26, 15, 0, 0, 0, time.UTC)

	applied, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "boards",
			Values: []interface{}{
				"board-1",
				"user-1",
				"Owned Board",
				timeValue(updatedAt),
				timeValue(updatedAt),
				nil,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected push success, got %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied change, got %d", applied)
	}

	select {
	case event := <-events:
		if event.Type != RealtimeSyncUpdateEvent || event.Source != RealtimeSourceSyncPush {
			t.Fatalf("expected sync_push realtime event, got %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected realtime event after successful push")
	}
}

func TestRemoteSyncService_PushChangesDoesNotPublishRealtimeUpdateOnFailure(t *testing.T) {
	database := openSyncTestDatabase(t)
	insertSyncUser(t, database, "user-1", "user1@example.com", time.Date(2025, 6, 26, 10, 0, 0, 0, time.UTC), nil)

	hub := NewUserRealtimeHub()
	events, unsubscribe := hub.Subscribe("user-1")
	defer unsubscribe()

	service := NewRemoteSyncService(database, SyncDialectSQLite, &mockLogger{})
	service.SetRealtimeHub(hub)

	_, err := service.PushChanges(context.Background(), "user-1", []RemoteSyncChange{
		{
			Table: "boards",
			Values: []interface{}{
				"board-1",
				"user-2",
				"Invalid",
				timeValue(time.Now().UTC()),
				timeValue(time.Now().UTC()),
				nil,
			},
		},
	})
	if err == nil {
		t.Fatal("expected push failure, got nil")
	}

	select {
	case event := <-events:
		t.Fatalf("expected no realtime event on failed push, got %+v", event)
	case <-time.After(25 * time.Millisecond):
	}
}

func openRemotePushFKTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:remote-push-fk-test-%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", time.Now().UnixNano())
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open remote push sqlite database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec(remotePushFKTestSchema); err != nil {
		t.Fatalf("failed to initialize remote push schema: %v", err)
	}

	return database
}

func assertRemoteSyncTaskExists(t *testing.T, database *sql.DB, id string) {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", id).Scan(&count); err != nil {
		t.Fatalf("failed to count task %s: %v", id, err)
	}
	if count != 1 {
		t.Fatalf("expected task %s to exist, got count %d", id, count)
	}
}

func assertRemoteSyncTransactionExists(t *testing.T, database *sql.DB, id string) {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM transactions WHERE id = ?", id).Scan(&count); err != nil {
		t.Fatalf("failed to count transaction %s: %v", id, err)
	}
	if count != 1 {
		t.Fatalf("expected transaction %s to exist, got count %d", id, count)
	}
}

const remotePushFKTestSchema = `
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
CREATE TABLE boards (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	name TEXT,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE columns (
	id TEXT PRIMARY KEY,
	board_id TEXT NOT NULL REFERENCES boards(id),
	name TEXT,
	color TEXT,
	position INTEGER,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE tasks (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	board_id TEXT REFERENCES boards(id),
	column_id TEXT REFERENCES columns(id),
	title TEXT,
	description TEXT,
	status TEXT,
	priority TEXT,
	due_date DATETIME NULL,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE credit_cards (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	name TEXT,
	bank TEXT,
	last4 TEXT,
	cutoff_day INTEGER,
	payment_day INTEGER,
	limit_cents INTEGER,
	color TEXT,
	network TEXT,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE financial_accounts (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	type TEXT,
	name TEXT,
	institution TEXT,
	last4 TEXT NULL,
	opening_balance_cents INTEGER,
	current_balance_cents INTEGER,
	credit_limit_cents INTEGER NULL,
	cutoff_day INTEGER NULL,
	payment_day INTEGER NULL,
	color TEXT,
	network TEXT,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE transactions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	credit_card_id TEXT REFERENCES credit_cards(id),
	payment_account_id TEXT NULL REFERENCES financial_accounts(id),
	destination_account_id TEXT NULL REFERENCES financial_accounts(id),
	type TEXT,
	concept TEXT,
	category TEXT,
	amount_cents INTEGER,
	date DATETIME,
	status TEXT,
	msi INTEGER NULL,
	installment_number INTEGER NULL,
	installment_count INTEGER NULL,
	is_historical BOOLEAN NOT NULL DEFAULT 0,
	recurrence TEXT,
	recurrence_limit INTEGER NULL,
	last_paid_at DATETIME NULL,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE ledger_entries (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	account_id TEXT NOT NULL REFERENCES financial_accounts(id),
	transaction_id TEXT NOT NULL REFERENCES transactions(id),
	amount_cents INTEGER,
	entry_type TEXT,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE credit_card_statements (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	credit_account_id TEXT NOT NULL REFERENCES financial_accounts(id),
	cycle_start DATETIME,
	cycle_end DATETIME,
	payment_due_date DATETIME,
	statement_amount_cents INTEGER,
	paid_amount_cents INTEGER,
	status TEXT,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE credit_card_statement_items (
	id TEXT PRIMARY KEY, user_id TEXT, statement_id TEXT, transaction_id TEXT,
	amount_cents INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
);
CREATE TABLE credit_card_payment_allocations (
	id TEXT PRIMARY KEY, user_id TEXT, statement_id TEXT, payment_transaction_id TEXT,
	amount_cents INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL
);
CREATE TABLE account_payable_payments (
	id TEXT PRIMARY KEY,
	account_payable_id TEXT NOT NULL REFERENCES transactions(id),
	user_id TEXT NOT NULL REFERENCES users(id),
	due_date DATETIME,
	paid_at DATETIME,
	amount_cents INTEGER,
	concept TEXT,
	category TEXT,
	created_transaction_id TEXT NOT NULL REFERENCES transactions(id),
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE notifications (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	title TEXT,
	message TEXT,
	is_read BOOLEAN,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
`

func insertBoardRow(t *testing.T, database *sql.DB, id, userID, name string, updatedAt time.Time) {
	t.Helper()

	if _, err := database.Exec(
		`INSERT INTO boards (id, user_id, name, created_at, updated_at, deleted_at)
		 VALUES (?, ?, ?, ?, ?, NULL)`,
		id,
		userID,
		name,
		updatedAt,
		updatedAt,
	); err != nil {
		t.Fatalf("failed to insert board row: %v", err)
	}
}

func assertRemoteSyncBoardName(t *testing.T, database *sql.DB, id, expectedName string) {
	t.Helper()

	var name string
	if err := database.QueryRow("SELECT name FROM boards WHERE id = ?", id).Scan(&name); err != nil {
		t.Fatalf("failed to read board %s: %v", id, err)
	}
	if name != expectedName {
		t.Fatalf("expected board name %s, got %s", expectedName, name)
	}
}
