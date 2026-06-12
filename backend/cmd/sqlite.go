package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	databaseURL := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(databasePath),
	}
	query := databaseURL.Query()
	query.Set("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "busy_timeout(5000)")
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String()
}

func initializeSQLiteSchema(ctx context.Context, database *sql.DB) error {
	if _, err := database.ExecContext(ctx, sqliteSchema); err != nil {
		return fmt.Errorf("failed to initialize sqlite schema: %w", err)
	}

	if err := ensureSQLiteSyncMetadata(ctx, database); err != nil {
		return err
	}

	return nil
}

func ensureSQLiteSyncMetadata(ctx context.Context, database *sql.DB) error {
	for _, table := range []string{"users", "boards", "columns", "tasks", "credit_cards", "transactions"} {
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
