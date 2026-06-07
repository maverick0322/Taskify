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
	saveBoardQuery = `
		INSERT INTO boards (id, user_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	getBoardByIDQuery = `
		SELECT id, user_id, name, created_at, updated_at
		FROM boards
		WHERE id = $1
	`

	getBoardsByUserIDQuery = `
		SELECT id, user_id, name, created_at, updated_at
		FROM boards
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	updateBoardQuery = `
		UPDATE boards
		SET name = $2,
			updated_at = $3
		WHERE id = $1
	`

	deleteBoardQuery = `
		DELETE FROM boards
		WHERE id = $1
	`
)

type postgresBoardDatabase interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

// PostgresBoardRepository implements Kanban board persistence using PostgreSQL.
type PostgresBoardRepository struct {
	database postgresBoardDatabase
	logger   ports.Logger
}

// NewPostgresBoardRepository receives the concrete pool at the infrastructure edge.
func NewPostgresBoardRepository(pool *pgxpool.Pool, logger ports.Logger) ports.BoardRepository {
	return &PostgresBoardRepository{
		database: pool,
		logger:   logger,
	}
}

func (repository *PostgresBoardRepository) Save(ctx context.Context, board *domain.Board) error {
	if board == nil {
		repository.logger.Error("cannot save nil board")
		return ports.ErrBoardRepositoryUnavailable
	}

	_, err := repository.database.Exec(
		ctx,
		saveBoardQuery,
		board.ID(),
		board.UserID(),
		board.Name(),
		board.CreatedAt(),
		board.UpdatedAt(),
	)
	if err == nil {
		return nil
	}

	repository.logger.Error("failed to save board", "userID", board.UserID(), "boardID", board.ID(), "error", err)
	return ports.ErrBoardRepositoryUnavailable
}

func (repository *PostgresBoardRepository) GetByID(ctx context.Context, id string) (*domain.Board, error) {
	board, err := repository.scanBoard(repository.database.QueryRow(ctx, getBoardByIDQuery, id))
	if err == nil {
		return board, nil
	}

	return repository.mapReadError(err, "failed to retrieve board by id", "boardID", id)
}

func (repository *PostgresBoardRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Board, error) {
	rows, err := repository.database.Query(ctx, getBoardsByUserIDQuery, userID)
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

func (repository *PostgresBoardRepository) Update(ctx context.Context, board *domain.Board) error {
	if board == nil {
		repository.logger.Error("cannot update nil board")
		return ports.ErrBoardRepositoryUnavailable
	}

	if _, err := repository.database.Exec(ctx, updateBoardQuery, board.ID(), board.Name(), board.UpdatedAt()); err != nil {
		repository.logger.Error("failed to update board", "userID", board.UserID(), "boardID", board.ID(), "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresBoardRepository) Delete(ctx context.Context, id string) error {
	if _, err := repository.database.Exec(ctx, deleteBoardQuery, id); err != nil {
		repository.logger.Error("failed to delete board", "boardID", id, "error", err)
		return ports.ErrBoardRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresBoardRepository) scanBoard(row pgx.Row) (*domain.Board, error) {
	var storedBoard storedBoard
	if err := row.Scan(
		&storedBoard.id,
		&storedBoard.userID,
		&storedBoard.name,
		&storedBoard.createdAt,
		&storedBoard.updatedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateBoard(
		storedBoard.id,
		storedBoard.userID,
		storedBoard.name,
		storedBoard.createdAt,
		storedBoard.updatedAt,
	)
}

func (repository *PostgresBoardRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.Board, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrBoardNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrBoardRepositoryUnavailable
}

type storedBoard struct {
	id        string
	userID    string
	name      string
	createdAt time.Time
	updatedAt time.Time
}
