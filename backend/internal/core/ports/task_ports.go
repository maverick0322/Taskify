package ports

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
)

// TaskRepository defines the outbound port for task persistence.
type TaskRepository interface {
	Save(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id string) (*domain.Task, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id string) error
}

// TaskUseCase defines user-scoped application operations for task management.
type TaskUseCase interface {
	CreateTask(ctx context.Context, userID, title, description string, priority domain.TaskPriority, dueDate time.Time) (*domain.Task, error)
	GetTask(ctx context.Context, userID, taskID string) (*domain.Task, error)
	GetUserTasks(ctx context.Context, userID string) ([]*domain.Task, error)
	UpdateTaskDetails(ctx context.Context, userID, taskID, title, description string) error
	UpdateTaskStatus(ctx context.Context, userID, taskID string, status domain.TaskStatus) error
	UpdateTaskPriority(ctx context.Context, userID, taskID string, priority domain.TaskPriority) error
	DeleteTask(ctx context.Context, userID, taskID string) error
}

var (
	ErrTaskNotFound              = errors.New("repository: task not found")
	ErrTaskAlreadyExists         = errors.New("repository: task already exists")
	ErrTaskRepositoryUnavailable = errors.New("repository: task persistence layer is unavailable or corrupted")
)
