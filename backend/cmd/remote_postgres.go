package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func openRemotePostgresDatabase(ctx context.Context, remoteDatabaseURL string) (*sql.DB, error) {
	remoteDatabase, err := sql.Open("pgx", remoteDatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote postgres database: %w", err)
	}

	if err := remoteDatabase.PingContext(ctx); err != nil {
		remoteDatabase.Close()
		return nil, fmt.Errorf("failed to connect to remote postgres database: %w", err)
	}

	if _, err := remoteDatabase.ExecContext(ctx, postgresSyncSchema); err != nil {
		remoteDatabase.Close()
		return nil, fmt.Errorf("failed to initialize remote postgres schema: %w", err)
	}

	return remoteDatabase, nil
}
