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
	insertSyncUser(t, remote, "remote-user", "remote@example.com", remoteUpdatedAt, nil)

	service := NewSyncService(local, remote, SyncDialectSQLite, &mockLogger{})
	service.now = func() time.Time { return cycleAt }

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	assertSyncUserExists(t, local, "remote-user", "remote@example.com")
	assertSyncUserExists(t, remote, "local-user", "local@example.com")
	assertSyncState(t, local, cycleAt)
}

func TestSyncService_SyncOnceLastWriteWinsByUpdatedAt(t *testing.T) {
	local := openSyncTestDatabase(t)
	remote := openSyncTestDatabase(t)

	localUpdatedAt := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	remoteUpdatedAt := time.Date(2026, 6, 11, 10, 5, 0, 0, time.UTC)
	insertSyncUser(t, local, "user-1", "older@example.com", localUpdatedAt, nil)
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

func assertSyncState(t *testing.T, database *sql.DB, expected time.Time) {
	t.Helper()

	var lastSyncAt time.Time
	if err := database.QueryRow("SELECT last_successful_sync_at FROM sync_state WHERE key = ?", syncStateKey).Scan(&lastSyncAt); err != nil {
		t.Fatalf("failed to query sync state: %v", err)
	}
	if !lastSyncAt.Equal(expected) {
		t.Fatalf("expected sync state %v, got %v", expected, lastSyncAt)
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
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME NULL
);
CREATE TABLE boards (id TEXT PRIMARY KEY, user_id TEXT, name TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE columns (id TEXT PRIMARY KEY, board_id TEXT, name TEXT, color TEXT, position INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE tasks (id TEXT PRIMARY KEY, user_id TEXT, board_id TEXT, column_id TEXT, title TEXT, description TEXT, status TEXT, priority TEXT, due_date DATETIME NULL, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE credit_cards (id TEXT PRIMARY KEY, user_id TEXT, name TEXT, bank TEXT, last4 TEXT, cutoff_day INTEGER, payment_day INTEGER, limit_cents INTEGER, color TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE transactions (id TEXT PRIMARY KEY, user_id TEXT, credit_card_id TEXT, type TEXT, concept TEXT, category TEXT, amount_cents INTEGER, date DATETIME, status TEXT, msi INTEGER NULL, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME NULL);
CREATE TABLE sync_state (key TEXT PRIMARY KEY, last_successful_sync_at DATETIME NOT NULL, updated_at DATETIME NOT NULL);
`
