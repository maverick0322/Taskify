package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	sqliteSaveColumnQuery = `
		INSERT INTO columns (id, board_id, name, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	sqliteGetColumnByIDQuery = `
		SELECT id, board_id, name, position, created_at, updated_at
		FROM columns
		WHERE id = ?
	`

	sqliteGetColumnsByBoardIDQuery = `
		SELECT id, board_id, name, position, created_at, updated_at
		FROM columns
		WHERE board_id = ?
		ORDER BY position ASC
	`

	sqliteUpdateColumnQuery = `
		UPDATE columns
		SET name = ?,
			position = ?,
			updated_at = ?
		WHERE id = ?
	`

	sqliteUpdateColumnPositionQuery = `
		UPDATE columns
		SET position = ?,
			updated_at = ?
		WHERE id = ?
	`

	sqliteDeleteColumnQuery = `
		DELETE FROM columns
		WHERE id = ?
	`
)

type SQLiteColumnRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteColumnRepository(database *sql.DB, logger ports.Logger) ports.ColumnRepository {
	return &SQLiteColumnRepository{database: database, logger: logger}
}

func (repository *SQLiteColumnRepository) Save(ctx context.Context, column *domain.Column) error {
	if column == nil {
		repository.logger.Error("cannot save nil column")
		return ports.ErrColumnRepositoryUnavailable
	}

	_, err := repository.database.ExecContext(ctx, sqliteSaveColumnQuery, column.ID(), column.BoardID(), column.Name(), column.Position(), timeValue(column.CreatedAt()), timeValue(column.UpdatedAt()))
	if err != nil {
		repository.logger.Error("failed to save column", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteColumnRepository) GetByID(ctx context.Context, id string) (*domain.Column, error) {
	column, err := repository.scanColumn(repository.database.QueryRowContext(ctx, sqliteGetColumnByIDQuery, id))
	if err == nil {
		return column, nil
	}

	return repository.mapReadError(err, "failed to retrieve column by id", "columnID", id)
}

func (repository *SQLiteColumnRepository) GetByBoardID(ctx context.Context, boardID string) ([]*domain.Column, error) {
	rows, err := repository.database.QueryContext(ctx, sqliteGetColumnsByBoardIDQuery, boardID)
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

func (repository *SQLiteColumnRepository) Update(ctx context.Context, column *domain.Column) error {
	if column == nil {
		repository.logger.Error("cannot update nil column")
		return ports.ErrColumnRepositoryUnavailable
	}

	if _, err := repository.database.ExecContext(ctx, sqliteUpdateColumnQuery, column.Name(), column.Position(), timeValue(column.UpdatedAt()), column.ID()); err != nil {
		repository.logger.Error("failed to update column", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteColumnRepository) UpdatePositions(ctx context.Context, columns []*domain.Column) error {
	tx, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		repository.logger.Error("failed to begin column position update", "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}
	defer tx.Rollback()

	for _, column := range columns {
		if column == nil {
			repository.logger.Error("cannot update nil column position")
			return ports.ErrColumnRepositoryUnavailable
		}
		if _, err := tx.ExecContext(ctx, sqliteUpdateColumnPositionQuery, column.Position(), timeValue(column.UpdatedAt()), column.ID()); err != nil {
			repository.logger.Error("failed to update column position", "boardID", column.BoardID(), "columnID", column.ID(), "error", err)
			return ports.ErrColumnRepositoryUnavailable
		}
	}

	if err := tx.Commit(); err != nil {
		repository.logger.Error("failed to commit column position update", "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteColumnRepository) Delete(ctx context.Context, id string) error {
	if _, err := repository.database.ExecContext(ctx, sqliteDeleteColumnQuery, id); err != nil {
		repository.logger.Error("failed to delete column", "columnID", id, "error", err)
		return ports.ErrColumnRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteColumnRepository) scanColumn(row interface {
	Scan(dest ...interface{}) error
}) (*domain.Column, error) {
	var storedColumn storedColumn
	if err := row.Scan(&storedColumn.id, &storedColumn.boardID, &storedColumn.name, &storedColumn.position, &storedColumn.createdAt, &storedColumn.updatedAt); err != nil {
		return nil, err
	}

	return domain.RehydrateColumn(storedColumn.id, storedColumn.boardID, storedColumn.name, storedColumn.position, storedColumn.createdAt, storedColumn.updatedAt)
}

func (repository *SQLiteColumnRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.Column, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrColumnNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrColumnRepositoryUnavailable
}
