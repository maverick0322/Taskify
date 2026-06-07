package services

import (
	"context"
	"errors"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

// taskService keeps task application rules separate from transport and persistence details.
type taskService struct {
	taskRepository ports.TaskRepository
	idGenerator    ports.IDGenerator
	logger         ports.Logger
}

func NewTaskService(
	taskRepository ports.TaskRepository,
	idGenerator ports.IDGenerator,
	logger ports.Logger,
) ports.TaskUseCase {
	return &taskService{
		taskRepository: taskRepository,
		idGenerator:    idGenerator,
		logger:         logger,
	}
}

func (service *taskService) CreateTask(
	ctx context.Context,
	userID,
	title,
	description string,
	priority domain.TaskPriority,
	dueDate time.Time,
) (*domain.Task, error) {
	taskID := service.idGenerator.Generate()
	task, err := domain.NewTask(taskID, userID, title, description, domain.TaskStatusTodo, priority, dueDate)
	if err != nil {
		return nil, err
	}

	if err := service.taskRepository.Save(ctx, task); err != nil {
		service.logger.Error("failed to save task", "userID", userID, "taskID", taskID, "error", err)
		return nil, ErrInternalProcessing
	}

	return task, nil
}

func (service *taskService) GetTask(ctx context.Context, userID, taskID string) (*domain.Task, error) {
	return service.getAuthorizedTask(ctx, userID, taskID)
}

func (service *taskService) GetUserTasks(ctx context.Context, userID string) ([]*domain.Task, error) {
	tasks, err := service.taskRepository.GetByUserID(ctx, userID)
	if err != nil {
		service.logger.Error("failed to retrieve user tasks", "userID", userID, "error", err)
		return nil, ErrInternalProcessing
	}

	return tasks, nil
}

func (service *taskService) UpdateTaskDetails(ctx context.Context, userID, taskID, title, description string) error {
	task, err := service.getAuthorizedTask(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if err := task.UpdateDetails(title, description); err != nil {
		return err
	}

	return service.persistTaskUpdate(ctx, task, "failed to update task details")
}

func (service *taskService) UpdateTaskStatus(ctx context.Context, userID, taskID string, status domain.TaskStatus) error {
	task, err := service.getAuthorizedTask(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if err := task.ChangeStatus(status); err != nil {
		return err
	}

	return service.persistTaskUpdate(ctx, task, "failed to update task status")
}

func (service *taskService) UpdateTaskPriority(ctx context.Context, userID, taskID string, priority domain.TaskPriority) error {
	task, err := service.getAuthorizedTask(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if err := task.ChangePriority(priority); err != nil {
		return err
	}

	return service.persistTaskUpdate(ctx, task, "failed to update task priority")
}

func (service *taskService) DeleteTask(ctx context.Context, userID, taskID string) error {
	task, err := service.getAuthorizedTask(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if err := service.taskRepository.Delete(ctx, task.ID()); err != nil {
		service.logger.Error("failed to delete task", "userID", userID, "taskID", task.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func (service *taskService) getAuthorizedTask(ctx context.Context, userID, taskID string) (*domain.Task, error) {
	task, err := service.taskRepository.GetByID(ctx, taskID)
	if errors.Is(err, ports.ErrTaskNotFound) {
		return nil, ports.ErrTaskNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve task", "userID", userID, "taskID", taskID, "error", err)
		return nil, ErrInternalProcessing
	}
	if task == nil {
		return nil, ports.ErrTaskNotFound
	}
	if !isTaskOwnedByUser(task, userID) {
		service.logger.Warn("unauthorized task access attempt", "userID", userID, "taskID", taskID)
		return nil, ports.ErrTaskNotFound
	}

	return task, nil
}

func (service *taskService) persistTaskUpdate(ctx context.Context, task *domain.Task, message string) error {
	if err := service.taskRepository.Update(ctx, task); err != nil {
		service.logger.Error(message, "userID", task.UserID(), "taskID", task.ID(), "error", err)
		return ErrInternalProcessing
	}

	return nil
}

func isTaskOwnedByUser(task *domain.Task, userID string) bool {
	return task.UserID() == userID
}
