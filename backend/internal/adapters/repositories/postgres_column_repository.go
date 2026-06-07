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
	saveColumnQuery = `
		INSERT INTO columns (id, board_id, name, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	getColumnByIDQuery = `
		SELECT id, board_id, name, position, created_at, updated_at
		FROM columns
		WHERE id = $1
	`

	getColumnsByBoardIDQuery = `
		SELECT id, board_id, name, position, created_at, updated_at
		FROM columns
		WHERE board_id = $1
		ORDER BY position ASC
	`

	updateColumnQuery = `
		UPDATE columns
		SET name = $2,
			position = $3,
			updated_at = $4
		WHERE id = $1
	`

	updateColumnPositionQuery = `
		UPDATE columns
		SET position = $2,
			updated_at = $3
		WHERE id = $1
	`

	deleteColumnQuery = `
		DELETE FROM columns
		WHERE id = $1
	`
)

type postgresColumnDatabase interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

type postgresColumnTransaction interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type beginPostgresColumnTransactionFunc func(ctx context.Context) (postgresColumnTransaction, error)

// PostgresColumnRepository implements Kanban column persistence using PostgreSQL.
type PostgresColumnRepository struct {
	database         postgresColumnDatabase
	beginTransaction beginPostgresColumnTransactionFunc
	logger           ports.Logger
}

// NewPostgresColumnRepository receives the concrete pool at the infrastructure edge.
func NewPostgresColumnRepository(pool *pgxpool.Pool, logger ports.Logger) ports.ColumnRepository {
	return &PostgresColumnRepository{
		database: pool,
		beginTransaction: func(ctx context.Context) (postgresColumnTransaction, error) {
			return pool.Begin(ctx)
		},
		logger: logger,
	}
}

func (repository *PostgresColumnRepository) Save(ctx context.Context, column *domain.Column) error {
	if column == nil {
		repository.logger.Error("cannot save nil column")
		return ports.ErrColumnRepositoryUnavailable
	}

	_, err := repository.database.Exec(
		ctx,
		saveColumnQuery,
		column.ID(),
		column.BoardID(),
		column.Name(),
		column.Position(),
		column.CreatedAt(),
		column.UpdatedAt(),
	)
	if err == nil {
		return nil
	}

	repository.logger.Error("failed to save column", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
	return ports.ErrColumnRepositoryUnavailable
}

func (repository *PostgresColumnRepository) GetByID(ctx context.Context, id string) (*domain.Column, error) {
	column, err := repository.scanColumn(repository.database.QueryRow(ctx, getColumnByIDQuery, id))
	if err == nil {
		return column, nil
	}

	return repository.mapReadError(err, "failed to retrieve column by id", "columnID", id)
}

func (repository *PostgresColumnRepository) GetByBoardID(ctx context.Context, boardID string) ([]*domain.Column, error) {
	rows, err := repository.database.Query(ctx, getColumnsByBoardIDQuery, boardID)
	if err != nil {
		repository.logger.Error("failed to retrieve columns by board id", "boardID", boardID, "error", err)
		return nil, ports.ErrColumnRepositoryUnavailable
	}
	defer rows.Close()

	columns := make([]*domain.Column, 0)
	for rows.Next() {
		column, err := repository.scanColumn(rows)
		if err != nil {
			repository.logger.Error("failed to scan column row", "boardID", boardID, "error", err)
			return nil, ports.ErrColumnRepositoryUnavailable
		}
		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		repository.logger.Error("failed while iterating column rows", "boardID", boardID, "error", err)
		return nil, ports.ErrColumnRepositoryUnavailable
	}

	return columns, nil
}

func (repository *PostgresColumnRepository) Update(ctx context.Context, column *domain.Column) error {
	if column == nil {
		repository.logger.Error("cannot update nil column")
		return ports.ErrColumnRepositoryUnavailable
	}

	if _, err := repository.database.Exec(ctx, updateColumnQuery, column.ID(), column.Name(), column.Position(), column.UpdatedAt()); err != nil {
		repository.logger.Error("failed to update column", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresColumnRepository) UpdatePositions(ctx context.Context, columns []*domain.Column) error {
	tx, err := repository.beginTransaction(ctx)
	if err != nil {
		repository.logger.Error("failed to begin column position update", "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}
	defer tx.Rollback(ctx)

	for _, column := range columns {
		if column == nil {
			repository.logger.Error("cannot update nil column position")
			return ports.ErrColumnRepositoryUnavailable
		}

		if _, err := tx.Exec(ctx, updateColumnPositionQuery, column.ID(), column.Position(), column.UpdatedAt()); err != nil {
			repository.logger.Error("failed to update column position", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
			return ports.ErrColumnRepositoryUnavailable
		}
	}

	if err := tx.Commit(ctx); err != nil {
		repository.logger.Error("failed to commit column position update", "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresColumnRepository) Delete(ctx context.Context, id string) error {
	if _, err := repository.database.Exec(ctx, deleteColumnQuery, id); err != nil {
		repository.logger.Error("failed to delete column", "columnID", id, "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresColumnRepository) scanColumn(row pgx.Row) (*domain.Column, error) {
	var storedColumn storedColumn
	if err := row.Scan(
		&storedColumn.id,
		&storedColumn.boardID,
		&storedColumn.name,
		&storedColumn.position,
		&storedColumn.createdAt,
		&storedColumn.updatedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateColumn(
		storedColumn.id,
		storedColumn.boardID,
		storedColumn.name,
		storedColumn.position,
		storedColumn.createdAt,
		storedColumn.updatedAt,
	)
}

func (repository *PostgresColumnRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.Column, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrColumnNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrColumnRepositoryUnavailable
}

type storedColumn struct {
	id        string
	boardID   string
	name      string
	position  int
	createdAt time.Time
	updatedAt time.Time
}
