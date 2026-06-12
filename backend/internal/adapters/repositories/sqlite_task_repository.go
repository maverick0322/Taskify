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
	sqliteSaveTaskQuery = `
		INSERT INTO tasks (id, user_id, board_id, column_id, title, description, status, priority, due_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	sqliteGetTaskByIDQuery = `
		SELECT id, user_id, board_id, column_id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks
		WHERE id = ? AND deleted_at IS NULL
	`

	sqliteGetTasksByUserIDQuery = `
		SELECT id, user_id, board_id, column_id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks
		WHERE user_id = ? AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	sqliteGetTasksByUserIDAndBoardIDQuery = `
		SELECT id, user_id, board_id, column_id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks
		WHERE user_id = ? AND board_id = ? AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	sqliteUpdateTaskQuery = `
		UPDATE tasks
		SET board_id = ?,
			column_id = ?,
			title = ?,
			description = ?,
			status = ?,
			priority = ?,
			due_date = ?,
			updated_at = ?
		WHERE id = ?
	`

	sqliteDeleteTaskQuery = `
		UPDATE tasks
		SET deleted_at = ?, updated_at = ?
		WHERE id = ?
	`
)

type SQLiteTaskRepository struct {
	database *sql.DB
	logger   ports.Logger
}

func NewSQLiteTaskRepository(database *sql.DB, logger ports.Logger) ports.TaskRepository {
	return &SQLiteTaskRepository{database: database, logger: logger}
}

func (repository *SQLiteTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	if task == nil {
		repository.logger.Error("cannot save nil task")
		return ports.ErrTaskRepositoryUnavailable
	}

	_, err := repository.database.ExecContext(
		ctx,
		sqliteSaveTaskQuery,
		task.ID(),
		task.UserID(),
		nullableString(task.BoardID()),
		nullableString(task.ColumnID()),
		task.Title(),
		task.Description(),
		string(task.Status()),
		string(task.Priority()),
		nullableTime(task.DueDate()),
		timeValue(task.CreatedAt()),
		timeValue(task.UpdatedAt()),
	)
	if err == nil {
		return nil
	}

	return repository.mapWriteError(err, "failed to save task", "userID", task.UserID(), "taskID", task.ID())
}

func (repository *SQLiteTaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	task, err := repository.scanTask(repository.database.QueryRowContext(ctx, sqliteGetTaskByIDQuery, id))
	if err == nil {
		return task, nil
	}

	return repository.mapReadError(err, "failed to retrieve task by id", "taskID", id)
}

func (repository *SQLiteTaskRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Task, error) {
	return repository.queryTasks(ctx, sqliteGetTasksByUserIDQuery, []interface{}{userID}, "failed to retrieve tasks by user id", "userID", userID)
}

func (repository *SQLiteTaskRepository) GetByUserIDAndBoardID(ctx context.Context, userID, boardID string) ([]*domain.Task, error) {
	return repository.queryTasks(ctx, sqliteGetTasksByUserIDAndBoardIDQuery, []interface{}{userID, boardID}, "failed to retrieve tasks by user id and board id", "userID", userID, "boardID", boardID)
}

func (repository *SQLiteTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	if task == nil {
		repository.logger.Error("cannot update nil task")
		return ports.ErrTaskRepositoryUnavailable
	}

	if _, err := repository.database.ExecContext(
		ctx,
		sqliteUpdateTaskQuery,
		nullableString(task.BoardID()),
		nullableString(task.ColumnID()),
		task.Title(),
		task.Description(),
		string(task.Status()),
		string(task.Priority()),
		nullableTime(task.DueDate()),
		timeValue(task.UpdatedAt()),
		task.ID(),
	); err != nil {
		repository.logger.Error("failed to update task", "userID", task.UserID(), "taskID", task.ID(), "error", err)
		return ports.ErrTaskRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteTaskRepository) Delete(ctx context.Context, id string) error {
	deletedAt := timeValue(time.Now())
	if _, err := repository.database.ExecContext(ctx, sqliteDeleteTaskQuery, deletedAt, deletedAt, id); err != nil {
		repository.logger.Error("failed to delete task", "taskID", id, "error", err)
		return ports.ErrTaskRepositoryUnavailable
	}

	return nil
}

func (repository *SQLiteTaskRepository) queryTasks(ctx context.Context, query string, arguments []interface{}, message string, keysAndValues ...interface{}) ([]*domain.Task, error) {
	rows, err := repository.database.QueryContext(ctx, query, arguments...)
	if err != nil {
		logValues := append(keysAndValues, "error", err)
		repository.logger.Error(message, logValues...)
		return nil, ports.ErrTaskRepositoryUnavailable
	}
	defer rows.Close()

	tasks := make([]*domain.Task, 0)
	for rows.Next() {
		task, err := repository.scanTask(rows)
		if err != nil {
			repository.logger.Error("failed to scan task row", "error", err)
			return nil, ports.ErrTaskRepositoryUnavailable
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		repository.logger.Error("failed while iterating task rows", "error", err)
		return nil, ports.ErrTaskRepositoryUnavailable
	}

	return tasks, nil
}

func (repository *SQLiteTaskRepository) scanTask(row interface {
	Scan(dest ...interface{}) error
}) (*domain.Task, error) {
	var storedTask sqliteStoredTask
	if err := row.Scan(
		&storedTask.id,
		&storedTask.userID,
		&storedTask.boardID,
		&storedTask.columnID,
		&storedTask.title,
		&storedTask.description,
		&storedTask.status,
		&storedTask.priority,
		&storedTask.dueDate,
		&storedTask.createdAt,
		&storedTask.updatedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateTask(
		storedTask.id,
		storedTask.userID,
		scanNullableString(storedTask.boardID),
		scanNullableString(storedTask.columnID),
		storedTask.title,
		storedTask.description,
		domain.TaskStatus(storedTask.status),
		domain.TaskPriority(storedTask.priority),
		storedTask.createdAt,
		storedTask.updatedAt,
		scanNullableTime(storedTask.dueDate),
	)
}

func (repository *SQLiteTaskRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.Task, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrTaskNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrTaskRepositoryUnavailable
}

func (repository *SQLiteTaskRepository) mapWriteError(err error, message string, keysAndValues ...interface{}) error {
	if isSQLiteConstraintViolation(err) {
		repository.logger.Warn("task already exists")
		return ports.ErrTaskAlreadyExists
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return ports.ErrTaskRepositoryUnavailable
}

type sqliteStoredTask struct {
	id          string
	userID      string
	boardID     sql.NullString
	columnID    sql.NullString
	title       string
	description string
	status      string
	priority    string
	dueDate     sql.NullTime
	createdAt   time.Time
	updatedAt   time.Time
}
