package services

import (
	"context"
	"database/sql"
	"testing"
	"time"
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
