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

	return nil
}
