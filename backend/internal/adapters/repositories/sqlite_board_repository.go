package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	sqliteSaveBoardQuery = `
		INSERT INTO boards (id, user_id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	sqliteGetBoardByIDQuery = `
		SELECT id, user_id, name, created_at, updated_at
		FROM boards
		WHERE id = ? AND deleted_at IS NULL
	`

	sqliteGetBoardsByUserIDQuery = `
		SELECT id, user_id, name, created_at, updated_at
		FROM boards
		WHERE user_id = ? AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	sqliteUpdateBoardQuery = `
		UPDATE boards
		SET name = ?,
			updated_at = ?
		WHERE id = ?
	`

	sqliteDeleteBoardQuery = `
		UPDATE boards
		SET deleted_at = ?, updated_at = ?
		WHERE id = ?
	`

	sqliteSoftDeleteBoardColumnsQuery = `
		UPDATE columns
		SET deleted_at = ?, updated_at = ?
		WHERE board_id = ? AND deleted_at IS NULL
	`

	sqliteSoftDeleteBoardTasksQuery = `
		UPDATE tasks
		SET deleted_at = ?, updated_at = ?
		WHERE board_id = ? AND deleted_at IS NULL
	`
)

type SQLiteBoardRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteBoardRepository(database *sql.DB, logger ports.Logger) ports.BoardRepository {
	return &SQLiteBoardRepository{database: database, logger: logger}
}

func (repository *SQLiteBoardRepository) Save(ctx context.Context, board *domain.Board) error {
	if board == nil {
		repository.logger.Error("cannot save nil board")
		return ports.ErrBoardRepositoryUnavailable
	}

	_, err := repository.database.ExecContext(ctx, sqliteSaveBoardQuery, board.ID(), board.UserID(), board.Name(), timeValue(board.CreatedAt()), timeValue(board.UpdatedAt()))
	if err != nil {
		repository.logger.Error("failed to save board", "userID", board.UserID(), "boardID", board.ID(), "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteBoardRepository) GetByID(ctx context.Context, id string) (*domain.Board, error) {
	board, err := repository.scanBoard(repository.database.QueryRowContext(ctx, sqliteGetBoardByIDQuery, id))
	if err == nil {
		return board, nil
	}

	return repository.mapReadError(err, "failed to retrieve board by id", "boardID", id)
}

func (repository *SQLiteBoardRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Board, error) {
	rows, err := repository.database.QueryContext(ctx, sqliteGetBoardsByUserIDQuery, userID)
	if err != nil {
		repository.logger.Error("failed to retrieve boards by user id", "userID", userID, "error", err)
		return nil, ports.ErrBoardRepositoryUnavailable
	}
	defer rows.Close()

	boards := make([]*domain.Board, 0)
	for rows.Next() {
		board, err := repository.scanBoard(rows)
		if err != nil {
			repository.logger.Error("failed to scan board row", "userID", userID, "error", err)
			return nil, ports.ErrBoardRepositoryUnavailable
		}
		boards = append(boards, board)
	}
	if err := rows.Err(); err != nil {
		repository.logger.Error("failed while iterating board rows", "userID", userID, "error", err)
		return nil, ports.ErrBoardRepositoryUnavailable
	}

	return boards, nil
}

func (repository *SQLiteBoardRepository) Update(ctx context.Context, board *domain.Board) error {
	if board == nil {
		repository.logger.Error("cannot update nil board")
		return ports.ErrBoardRepositoryUnavailable
	}

	if _, err := repository.database.ExecContext(ctx, sqliteUpdateBoardQuery, board.Name(), timeValue(board.UpdatedAt()), board.ID()); err != nil {
		repository.logger.Error("failed to update board", "userID", board.UserID(), "boardID", board.ID(), "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteBoardRepository) Delete(ctx context.Context, id string) error {
	deletedAt := timeValue(time.Now())
	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		repository.logger.Error("failed to begin board soft delete", "boardID", id, "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, sqliteDeleteBoardQuery, deletedAt, deletedAt, id); err != nil {
		repository.logger.Error("failed to delete board", "boardID", id, "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}
	if _, err := tx.ExecContext(ctx, sqliteSoftDeleteBoardColumnsQuery, deletedAt, deletedAt, id); err != nil {
		repository.logger.Error("failed to delete board columns", "boardID", id, "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}
	if _, err := tx.ExecContext(ctx, sqliteSoftDeleteBoardTasksQuery, deletedAt, deletedAt, id); err != nil {
		repository.logger.Error("failed to delete board tasks", "boardID", id, "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}
	if err := tx.Commit(); err != nil {
		repository.logger.Error("failed to commit board soft delete", "boardID", id, "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteBoardRepository) scanBoard(row interface {
	Scan(dest ...interface{}) error
}) (*domain.Board, error) {
	var storedBoard storedBoard
	if err := row.Scan(&storedBoard.id, &storedBoard.userID, &storedBoard.name, &storedBoard.createdAt, &storedBoard.updatedAt); err != nil {
		return nil, err
	}

	return domain.RehydrateBoard(storedBoard.id, storedBoard.userID, storedBoard.name, storedBoard.createdAt, storedBoard.updatedAt)
}

func (repository *SQLiteBoardRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.Board, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrBoardNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrBoardRepositoryUnavailable
}
