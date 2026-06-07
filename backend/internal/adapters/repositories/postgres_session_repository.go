package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	saveRefreshTokenQuery = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, is_revoked)
		VALUES ($1, $2, $3, $4, $5)
	`

	getRefreshTokenByHashQuery = `
		SELECT id, user_id, token_hash, expires_at, is_revoked
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	revokeRefreshTokenQuery = `
		UPDATE refresh_tokens
		SET is_revoked = TRUE, updated_at = NOW()
		WHERE id = $1
	`
)

type postgresSessionExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

type postgresSessionTransaction interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type beginPostgresSessionTransactionFunc func(ctx context.Context) (postgresSessionTransaction, error)

// PostgresSessionRepository persists hashed refresh-token sessions in PostgreSQL.
type PostgresSessionRepository struct {
	database         postgresSessionExecutor
	beginTransaction beginPostgresSessionTransactionFunc
	logger           ports.Logger
}

// NewPostgresSessionRepository wraps pgxpool transaction creation to keep tests free of database dependencies.
func NewPostgresSessionRepository(pool *pgxpool.Pool, logger ports.Logger) ports.SessionRepository {
	return &PostgresSessionRepository{
		database: pool,
		beginTransaction: func(ctx context.Context) (postgresSessionTransaction, error) {
			return pool.Begin(ctx)
		},
		logger: logger,
	}
}

func (repository *PostgresSessionRepository) Save(ctx context.Context, refreshToken *domain.RefreshToken) error {
	if refreshToken == nil {
		repository.logger.Error("cannot save nil refresh session")
		return ports.ErrSessionRepositoryUnavailable
	}

	_, err := repository.database.Exec(
		ctx,
		saveRefreshTokenQuery,
		refreshToken.ID(),
		refreshToken.UserID(),
		refreshToken.TokenHash(),
		refreshToken.ExpiresAt(),
		refreshToken.IsRevoked(),
	)
	if err == nil {
		return nil
	}

	return repository.mapWriteError(err, "failed to save refresh session", "sessionID", refreshToken.ID(), "userID", refreshToken.UserID())
}

func (repository *PostgresSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	refreshToken, err := repository.scanRefreshToken(repository.database.QueryRow(ctx, getRefreshTokenByHashQuery, tokenHash))
	if err == nil {
		return refreshToken, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrSessionNotFound
	}

	repository.logger.Error("failed to retrieve refresh session", "error", err)
	return nil, ports.ErrSessionRepositoryUnavailable
}

func (repository *PostgresSessionRepository) Revoke(ctx context.Context, id string) error {
	_, err := repository.database.Exec(ctx, revokeRefreshTokenQuery, id)
	if err == nil {
		return nil
	}

	repository.logger.Error("failed to revoke refresh session", "sessionID", id, "error", err)
	return ports.ErrSessionRepositoryUnavailable
}

func (repository *PostgresSessionRepository) Rotate(ctx context.Context, revokedTokenID string, newRefreshToken *domain.RefreshToken) error {
	if newRefreshToken == nil {
		repository.logger.Error("cannot rotate to nil refresh session", "sessionID", revokedTokenID)
		return ports.ErrSessionRepositoryUnavailable
	}

	tx, err := repository.beginTransaction(ctx)
	if err != nil {
		repository.logger.Error("failed to begin refresh session rotation", "sessionID", revokedTokenID, "error", err)
		return ports.ErrSessionRepositoryUnavailable
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, revokeRefreshTokenQuery, revokedTokenID); err != nil {
		repository.logger.Error("failed to revoke refresh session during rotation", "sessionID", revokedTokenID, "error", err)
		return ports.ErrSessionRepositoryUnavailable
	}

	if _, err := tx.Exec(
		ctx,
		saveRefreshTokenQuery,
		newRefreshToken.ID(),
		newRefreshToken.UserID(),
		newRefreshToken.TokenHash(),
		newRefreshToken.ExpiresAt(),
		newRefreshToken.IsRevoked(),
	); err != nil {
		return repository.mapWriteError(err, "failed to save rotated refresh session", "sessionID", newRefreshToken.ID(), "userID", newRefreshToken.UserID())
	}

	if err := tx.Commit(ctx); err != nil {
		repository.logger.Error("failed to commit refresh session rotation", "sessionID", revokedTokenID, "error", err)
		return ports.ErrSessionRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresSessionRepository) scanRefreshToken(row pgx.Row) (*domain.RefreshToken, error) {
	var storedRefreshToken storedRefreshToken
	if err := row.Scan(
		&storedRefreshToken.id,
		&storedRefreshToken.userID,
		&storedRefreshToken.tokenHash,
		&storedRefreshToken.expiresAt,
		&storedRefreshToken.isRevoked,
	); err != nil {
		return nil, err
	}

	return domain.NewRefreshToken(
		storedRefreshToken.id,
		storedRefreshToken.userID,
		storedRefreshToken.tokenHash,
		storedRefreshToken.expiresAt,
		storedRefreshToken.isRevoked,
	)
}

func (repository *PostgresSessionRepository) mapWriteError(err error, message string, keysAndValues ...interface{}) error {
	if isUniqueViolation(err) {
		repository.logger.Warn("refresh session already exists")
		return ports.ErrSessionAlreadyExists
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return ports.ErrSessionRepositoryUnavailable
}

type storedRefreshToken struct {
	id        string
	userID    string
	tokenHash string
	expiresAt time.Time
	isRevoked bool
}
