package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	sqliteSaveRefreshTokenQuery = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, is_revoked)
		VALUES (?, ?, ?, ?, ?)
	`

	sqliteGetRefreshTokenByHashQuery = `
		SELECT id, user_id, token_hash, expires_at, is_revoked
		FROM refresh_tokens
		WHERE token_hash = ?
	`

	sqliteRevokeRefreshTokenQuery = `
		UPDATE refresh_tokens
		SET is_revoked = 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
)

type SQLiteSessionRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteSessionRepository(database *sql.DB, logger ports.Logger) ports.SessionRepository {
	return &SQLiteSessionRepository{database: database, logger: logger}
}

func (repository *SQLiteSessionRepository) Save(ctx context.Context, refreshToken *domain.RefreshToken) error {
	if refreshToken == nil {
		repository.logger.Error("cannot save nil refresh session")
		return ports.ErrSessionRepositoryUnavailable
	}

	_, err := repository.database.ExecContext(
		ctx,
		sqliteSaveRefreshTokenQuery,
		refreshToken.ID(),
		refreshToken.UserID(),
		refreshToken.TokenHash(),
		timeValue(refreshToken.ExpiresAt()),
		refreshToken.IsRevoked(),
	)
	if err == nil {
		return nil
	}

	return repository.mapWriteError(err, "failed to save refresh session", "sessionID", refreshToken.ID(), "userID", refreshToken.UserID())
}

func (repository *SQLiteSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	refreshToken, err := repository.scanRefreshToken(repository.database.QueryRowContext(ctx, sqliteGetRefreshTokenByHashQuery, tokenHash))
	if err == nil {
		return refreshToken, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrSessionNotFound
	}

	repository.logger.Error("failed to retrieve refresh session", "error", err)
	return nil, ports.ErrSessionRepositoryUnavailable
}

func (repository *SQLiteSessionRepository) Revoke(ctx context.Context, id string) error {
	if _, err := repository.database.ExecContext(ctx, sqliteRevokeRefreshTokenQuery, id); err != nil {
		repository.logger.Error("failed to revoke refresh session", "sessionID", id, "error", err)
		return ports.ErrSessionRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteSessionRepository) Rotate(ctx context.Context, revokedTokenID string, newRefreshToken *domain.RefreshToken) error {
	if newRefreshToken == nil {
		repository.logger.Error("cannot rotate to nil refresh session", "sessionID", revokedTokenID)
		return ports.ErrSessionRepositoryUnavailable
	}

	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		repository.logger.Error("failed to begin refresh session rotation", "sessionID", revokedTokenID, "error", err)
		return ports.ErrSessionRepositoryUnavailable
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, sqliteRevokeRefreshTokenQuery, revokedTokenID); err != nil {
		repository.logger.Error("failed to revoke refresh session during rotation", "sessionID", revokedTokenID, "error", err)
		return ports.ErrSessionRepositoryUnavailable
	}

	if _, err := tx.ExecContext(
		ctx,
		sqliteSaveRefreshTokenQuery,
		newRefreshToken.ID(),
		newRefreshToken.UserID(),
		newRefreshToken.TokenHash(),
		timeValue(newRefreshToken.ExpiresAt()),
		newRefreshToken.IsRevoked(),
	); err != nil {
		return repository.mapWriteError(err, "failed to save rotated refresh session", "sessionID", newRefreshToken.ID(), "userID", newRefreshToken.UserID())
	}

	if err := tx.Commit(); err != nil {
		repository.logger.Error("failed to commit refresh session rotation", "sessionID", revokedTokenID, "error", err)
		return ports.ErrSessionRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteSessionRepository) scanRefreshToken(row interface {
	Scan(dest ...interface{}) error
}) (*domain.RefreshToken, error) {
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

func (repository *SQLiteSessionRepository) mapWriteError(err error, message string, keysAndValues ...interface{}) error {
	if isSQLiteConstraintViolation(err) {
		repository.logger.Warn("refresh session already exists")
		return ports.ErrSessionAlreadyExists
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return ports.ErrSessionRepositoryUnavailable
}
