package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
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

func openRemotePostgresPool(ctx context.Context, remoteDatabaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, remoteDatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to remote postgres pool: %w", err)
	}

	return pool, nil
}
