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
	saveTaskQuery = `
		INSERT INTO tasks (id, user_id, title, description, status, priority, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	getTaskByIDQuery = `
		SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	getTasksByUserIDQuery = `
		SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	updateTaskQuery = `
		UPDATE tasks
		SET title = $2,
			description = $3,
			status = $4,
			priority = $5,
			due_date = $6,
			updated_at = $7
		WHERE id = $1
	`

	deleteTaskQuery = `
		DELETE FROM tasks
		WHERE id = $1
	`
)

type postgresTaskDatabase interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

// PostgresTaskRepository implements task persistence using PostgreSQL.
type PostgresTaskRepository struct {
	database postgresTaskDatabase
	logger   ports.Logger
}

// NewPostgresTaskRepository receives the concrete pool at the infrastructure edge.
func NewPostgresTaskRepository(pool *pgxpool.Pool, logger ports.Logger) ports.TaskRepository {
	return &PostgresTaskRepository{
		database: pool,
		logger:   logger,
	}
}

func (repository *PostgresTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	if task == nil {
		repository.logger.Error("cannot save nil task")
		return ports.ErrTaskRepositoryUnavailable
	}

	_, err := repository.database.Exec(
		ctx,
		saveTaskQuery,
		task.ID(),
		task.UserID(),
		task.Title(),
		task.Description(),
		string(task.Status()),
		string(task.Priority()),
		nullableTaskDueDate(task.DueDate()),
		task.CreatedAt(),
		task.UpdatedAt(),
	)
	if err == nil {
		return nil
	}

	return repository.mapWriteError(err, "failed to save task", "userID", task.UserID(), "taskID", task.ID())
}

func (repository *PostgresTaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	task, err := repository.scanTask(repository.database.QueryRow(ctx, getTaskByIDQuery, id))
	if err == nil {
		return task, nil
	}

	return repository.mapReadError(err, "failed to retrieve task by id", "taskID", id)
}

func (repository *PostgresTaskRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Task, error) {
	rows, err := repository.database.Query(ctx, getTasksByUserIDQuery, userID)
	if err != nil {
		repository.logger.Error("failed to retrieve tasks by user id", "userID", userID, "error", err)
		return nil, ports.ErrTaskRepositoryUnavailable
	}
	defer rows.Close()

	tasks := make([]*domain.Task, 0)
	for rows.Next() {
		task, err := repository.scanTask(rows)
		if err != nil {
			repository.logger.Error("failed to scan task row", "userID", userID, "error", err)
			return nil, ports.ErrTaskRepositoryUnavailable
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		repository.logger.Error("failed while iterating task rows", "userID", userID, "error", err)
		return nil, ports.ErrTaskRepositoryUnavailable
	}

	return tasks, nil
}

func (repository *PostgresTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	if task == nil {
		repository.logger.Error("cannot update nil task")
		return ports.ErrTaskRepositoryUnavailable
	}

	_, err := repository.database.Exec(
		ctx,
		updateTaskQuery,
		task.ID(),
		task.Title(),
		task.Description(),
		string(task.Status()),
		string(task.Priority()),
		nullableTaskDueDate(task.DueDate()),
		task.UpdatedAt(),
	)
	if err == nil {
		return nil
	}

	repository.logger.Error("failed to update task", "userID", task.UserID(), "taskID", task.ID(), "error", err)
	return ports.ErrTaskRepositoryUnavailable
}

func (repository *PostgresTaskRepository) Delete(ctx context.Context, id string) error {
	if _, err := repository.database.Exec(ctx, deleteTaskQuery, id); err != nil {
		repository.logger.Error("failed to delete task", "taskID", id, "error", err)
		return ports.ErrTaskRepositoryUnavailable
	}

	return nil
}

func (repository *PostgresTaskRepository) scanTask(row pgx.Row) (*domain.Task, error) {
	var storedTask storedTask
	if err := row.Scan(
		&storedTask.id,
		&storedTask.userID,
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
		storedTask.title,
		storedTask.description,
		domain.TaskStatus(storedTask.status),
		domain.TaskPriority(storedTask.priority),
		storedTask.createdAt,
		storedTask.updatedAt,
		taskDueDateValue(storedTask.dueDate),
	)
}

func (repository *PostgresTaskRepository) mapReadError(err error, message string, keysAndValues ...interface{}) (*domain.Task, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrTaskNotFound
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return nil, ports.ErrTaskRepositoryUnavailable
}

func (repository *PostgresTaskRepository) mapWriteError(err error, message string, keysAndValues ...interface{}) error {
	if isUniqueViolation(err) {
		repository.logger.Warn("task already exists")
		return ports.ErrTaskAlreadyExists
	}

	logValues := append(keysAndValues, "error", err)
	repository.logger.Error(message, logValues...)
	return ports.ErrTaskRepositoryUnavailable
}

type storedTask struct {
	id          string
	userID      string
	title       string
	description string
	status      string
	priority    string
	dueDate     *time.Time
	createdAt   time.Time
	updatedAt   time.Time
}

func nullableTaskDueDate(dueDate time.Time) interface{} {
	if dueDate.IsZero() {
		return nil
	}

	return dueDate
}

func taskDueDateValue(dueDate *time.Time) time.Time {
	if dueDate == nil {
		return time.Time{}
	}

	return *dueDate
}
