package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestHTTPRemoteSyncService_ApplyPulledChanges_ReordersTaskHierarchy(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "https://taskify-7n1b.onrender.com", &mockLogger{})

	changes := []RemoteSyncChange{
		{
			Table: "tasks",
			Values: []interface{}{
				"task-1", "user-1", "board-1", "column-1", "Tarea", "Desc", "todo", "high",
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "columns",
			Values: []interface{}{
				"column-1", "board-1", "Por hacer", "blue", int64(0),
				timeValue(time.Date(2026, 6, 27, 8, 30, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 8, 30, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "boards",
			Values: []interface{}{
				"board-1", "user-1", "Trabajo",
				timeValue(time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "users",
			Values: []interface{}{
				"user-1", "user1@example.com", "hashedpassword", "User", "One",
				timeValue(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
				"", "", timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				nil,
			},
		},
	}

	pulledRows, err := service.applyPulledChanges(context.Background(), changes)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pulledRows != 4 {
		t.Fatalf("expected 4 pulled rows, got %d", pulledRows)
	}
	assertHTTPSyncRowCount(t, database, "users", 1)
	assertHTTPSyncRowCount(t, database, "boards", 1)
	assertHTTPSyncRowCount(t, database, "columns", 1)
	assertHTTPSyncRowCount(t, database, "tasks", 1)
}

func TestHTTPRemoteSyncService_ApplyPulledChanges_ReordersFinancialHierarchy(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "https://taskify-7n1b.onrender.com", &mockLogger{})

	changes := []RemoteSyncChange{
		{
			Table: "ledger_entries",
			Values: []interface{}{
				"entry-1", "user-1", "account-1", "transaction-1", int64(5000), "debit",
				timeValue(time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "transactions",
			Values: []interface{}{
				"transaction-1", "user-1", nil, "account-1", nil, "EXPENSE", "Comida", "Gastos", int64(5000),
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)), "PAID", nil, nil, nil, false, "once", nil, nil,
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "financial_accounts",
			Values: []interface{}{
				"account-1", "user-1", "checking", "Cuenta", "Banco", "1234", int64(0), int64(100000), nil, nil, nil, "blue", nil,
				timeValue(time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "users",
			Values: []interface{}{
				"user-1", "user1@example.com", "hashedpassword", "User", "One",
				timeValue(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
				"", "", timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				nil,
			},
		},
	}

	pulledRows, err := service.applyPulledChanges(context.Background(), changes)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pulledRows != 4 {
		t.Fatalf("expected 4 pulled rows, got %d", pulledRows)
	}
	assertHTTPSyncRowCount(t, database, "financial_accounts", 1)
	assertHTTPSyncRowCount(t, database, "transactions", 1)
	assertHTTPSyncRowCount(t, database, "ledger_entries", 1)
}

func TestHTTPRemoteSyncService_ApplyPulledChanges_ReordersCreditCardHierarchy(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "https://taskify-7n1b.onrender.com", &mockLogger{})

	changes := []RemoteSyncChange{
		{
			Table: "transactions",
			Values: []interface{}{
				"transaction-1", "user-1", "card-1", nil, nil, "EXPENSE", "Laptop", "Compras", int64(250000),
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)), "PAID", nil, nil, nil, false, "once", nil, nil,
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "credit_cards",
			Values: []interface{}{
				"card-1", "user-1", "Tarjeta Oro", "Banco Uno", "4242", int64(20), int64(10), int64(500000), "gold", "Visa",
				timeValue(time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "users",
			Values: []interface{}{
				"user-1", "user1@example.com", "hashedpassword", "User", "One",
				timeValue(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
				"", "", timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				nil,
			},
		},
	}

	pulledRows, err := service.applyPulledChanges(context.Background(), changes)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pulledRows != 3 {
		t.Fatalf("expected 3 pulled rows, got %d", pulledRows)
	}
	assertHTTPSyncRowCount(t, database, "credit_cards", 1)
	assertHTTPSyncRowCount(t, database, "transactions", 1)
}

func TestHTTPRemoteSyncService_ApplyPulledChanges_FailsWhenParentIsMissing(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "https://taskify-7n1b.onrender.com", &mockLogger{})

	changes := []RemoteSyncChange{
		{
			Table: "tasks",
			Values: []interface{}{
				"task-1", "user-1", "board-missing", "column-missing", "Tarea", "Desc", "todo", "high",
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "users",
			Values: []interface{}{
				"user-1", "user1@example.com", "hashedpassword", "User", "One",
				timeValue(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
				"", "", timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				nil,
			},
		},
	}

	_, err := service.applyPulledChanges(context.Background(), changes)

	if err == nil {
		t.Fatal("expected foreign key error, got nil")
	}
}

func TestHTTPRemoteSyncService_ApplyPulledChanges_FailsWhenCreditCardParentIsMissing(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "https://taskify-7n1b.onrender.com", &mockLogger{})

	changes := []RemoteSyncChange{
		{
			Table: "transactions",
			Values: []interface{}{
				"transaction-1", "user-1", "card-missing", nil, nil, "EXPENSE", "Laptop", "Compras", int64(250000),
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)), "PAID", nil, nil, nil, false, "once", nil, nil,
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)),
				nil,
			},
		},
		{
			Table: "users",
			Values: []interface{}{
				"user-1", "user1@example.com", "hashedpassword", "User", "One",
				timeValue(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
				"", "", timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				timeValue(time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)),
				nil,
			},
		},
	}

	_, err := service.applyPulledChanges(context.Background(), changes)

	if err == nil {
		t.Fatal("expected foreign key error, got nil")
	}
}

func TestHTTPRemoteSyncService_ApplyPulledChanges_FailsOnUnknownTable(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "https://taskify-7n1b.onrender.com", &mockLogger{})

	_, err := service.applyPulledChanges(context.Background(), []RemoteSyncChange{
		{Table: "mystery_table", Values: []interface{}{"id-1"}},
	})

	if err == nil || !strings.Contains(err.Error(), "unknown table") {
		t.Fatalf("expected unknown table error, got %v", err)
	}
}

func TestHTTPRemoteSyncService_EnsureRemoteSessionRefreshesPersistedTokens(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "", &mockLogger{})

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/users/refresh" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		writeHTTPJSON(t, response, map[string]string{
			"accessToken":  "remote-access-new",
			"refreshToken": "remote-refresh-new",
		})
	}))
	defer server.Close()

	service.remoteAPIURL = server.URL
	service.RestoreRemoteSession("remote-access-old", "remote-refresh-old")

	tokenPair, err := service.EnsureRemoteSession(context.Background())

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tokenPair.AccessToken != "remote-access-new" || tokenPair.RefreshToken != "remote-refresh-new" {
		t.Fatalf("expected refreshed token pair, got %+v", tokenPair)
	}
}

func TestHTTPRemoteSyncService_EnsureRemoteSessionFallsBackToCurrentTokensWhenRefreshFailsTransiently(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "", &mockLogger{})

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Error(response, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	service.remoteAPIURL = server.URL
	service.RestoreRemoteSession("remote-access-current", "remote-refresh-current")

	tokenPair, err := service.EnsureRemoteSession(context.Background())

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tokenPair.AccessToken != "remote-access-current" || tokenPair.RefreshToken != "remote-refresh-current" {
		t.Fatalf("expected current token pair fallback, got %+v", tokenPair)
	}
}

func TestHTTPRemoteSyncService_EnsureRemoteSessionFailsWhenNoSessionExists(t *testing.T) {
	database := openHTTPSyncTestDatabase(t)
	service := NewHTTPRemoteSyncService(database, "https://taskify-7n1b.onrender.com", &mockLogger{})

	_, err := service.EnsureRemoteSession(context.Background())

	if err == nil || !strings.Contains(err.Error(), ErrRemoteSyncSessionUnavailable.Error()) {
		t.Fatalf("expected remote session unavailable error, got %v", err)
	}
}

func openHTTPSyncTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", fmt.Sprintf("file:http-sync-test-%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Exec(httpSyncTestSchema); err != nil {
		t.Fatalf("failed to initialize http sync schema: %v", err)
	}

	return database
}

func assertHTTPSyncRowCount(t *testing.T, database *sql.DB, table string, expected int) {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("failed to count %s rows: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %s row count %d, got %d", table, expected, count)
	}
}

func writeHTTPJSON(t *testing.T, response http.ResponseWriter, payload interface{}) {
	t.Helper()

	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(payload); err != nil {
		t.Fatalf("failed to encode json response: %v", err)
	}
}

const httpSyncTestSchema = `
CREATE TABLE users (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	birth_date DATETIME NOT NULL,
	avatar_local_path TEXT,
	avatar_url TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE boards (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	name TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE columns (
	id TEXT PRIMARY KEY,
	board_id TEXT NOT NULL REFERENCES boards(id),
	name TEXT NOT NULL,
	color TEXT NOT NULL,
	position INTEGER NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE tasks (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	board_id TEXT REFERENCES boards(id),
	column_id TEXT REFERENCES columns(id),
	title TEXT NOT NULL,
	description TEXT NOT NULL,
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	due_date DATETIME,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE credit_cards (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	name TEXT NOT NULL,
	bank TEXT NOT NULL,
	last4 TEXT NOT NULL,
	cutoff_day INTEGER NOT NULL,
	payment_day INTEGER NOT NULL,
	limit_cents INTEGER NOT NULL,
	color TEXT NOT NULL,
	network TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE financial_accounts (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	type TEXT NOT NULL,
	name TEXT NOT NULL,
	institution TEXT NOT NULL,
	last4 TEXT NOT NULL,
	opening_balance_cents INTEGER NOT NULL,
	current_balance_cents INTEGER NOT NULL,
	credit_limit_cents INTEGER,
	cutoff_day INTEGER,
	payment_day INTEGER,
	color TEXT NOT NULL,
	network TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE transactions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	credit_card_id TEXT REFERENCES credit_cards(id),
	payment_account_id TEXT REFERENCES financial_accounts(id),
	destination_account_id TEXT REFERENCES financial_accounts(id),
	type TEXT NOT NULL,
	concept TEXT NOT NULL,
	category TEXT NOT NULL,
	amount_cents INTEGER NOT NULL,
	date DATETIME NOT NULL,
	status TEXT NOT NULL,
	msi INTEGER,
	installment_number INTEGER,
	installment_count INTEGER,
	is_historical BOOLEAN NOT NULL DEFAULT 0,
	recurrence TEXT NOT NULL,
	recurrence_limit INTEGER,
	last_paid_at DATETIME,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE ledger_entries (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	account_id TEXT NOT NULL REFERENCES financial_accounts(id),
	transaction_id TEXT NOT NULL REFERENCES transactions(id),
	amount_cents INTEGER NOT NULL,
	entry_type TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE credit_card_statements (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	credit_account_id TEXT NOT NULL REFERENCES financial_accounts(id),
	cycle_start DATETIME NOT NULL,
	cycle_end DATETIME NOT NULL,
	payment_due_date DATETIME NOT NULL,
	statement_amount_cents INTEGER NOT NULL,
	paid_amount_cents INTEGER NOT NULL,
	status TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE credit_card_statement_items (
	id TEXT PRIMARY KEY, user_id TEXT, statement_id TEXT, transaction_id TEXT,
	amount_cents INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
);
CREATE TABLE credit_card_payment_allocations (
	id TEXT PRIMARY KEY, user_id TEXT, statement_id TEXT, payment_transaction_id TEXT,
	amount_cents INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
);
CREATE TABLE account_payable_payments (
	id TEXT PRIMARY KEY,
	account_payable_id TEXT NOT NULL REFERENCES transactions(id),
	user_id TEXT NOT NULL REFERENCES users(id),
	due_date DATETIME NOT NULL,
	paid_at DATETIME NOT NULL,
	amount_cents INTEGER NOT NULL,
	concept TEXT NOT NULL,
	category TEXT NOT NULL,
	created_transaction_id TEXT NOT NULL REFERENCES transactions(id),
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE notifications (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	title TEXT NOT NULL,
	message TEXT NOT NULL,
	is_read BOOLEAN NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);
CREATE TABLE sync_runtime_flags (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
INSERT INTO sync_runtime_flags (key, value) VALUES ('suppress_outbox', '0');
CREATE TABLE sync_state (
	key TEXT PRIMARY KEY,
	last_successful_sync_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`
