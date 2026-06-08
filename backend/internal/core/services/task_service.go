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
	taskRepository  ports.TaskRepository
	boardRepository ports.BoardRepository
	idGenerator     ports.IDGenerator
	logger          ports.Logger
}

func NewTaskService(
	taskRepository ports.TaskRepository,
	boardRepository ports.BoardRepository,
	idGenerator ports.IDGenerator,
	logger ports.Logger,
) ports.TaskUseCase {
	return &taskService{
		taskRepository:  taskRepository,
		boardRepository: boardRepository,
		idGenerator:     idGenerator,
		logger:          logger,
	}
}

func (service *taskService) CreateTask(
	ctx context.Context,
	userID,
	boardID,
	title,
	description string,
	priority domain.TaskPriority,
	dueDate time.Time,
) (*domain.Task, error) {
	taskID := service.idGenerator.Generate()
	task, err := domain.NewTask(taskID, userID, boardID, title, description, domain.TaskStatusTodo, priority, dueDate)
	if err != nil {
		return nil, err
	}

	if _, err := service.getAuthorizedBoard(ctx, userID, task.BoardID()); err != nil {
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

func (service *taskService) GetBoardTasks(ctx context.Context, userID, boardID string) ([]*domain.Task, error) {
	board, err := service.getAuthorizedBoard(ctx, userID, boardID)
	if err != nil {
		return nil, err
	}

	tasks, err := service.taskRepository.GetByUserIDAndBoardID(ctx, userID, board.ID())
	if err != nil {
		service.logger.Error("failed to retrieve board tasks", "userID", userID, "boardID", board.ID(), "error", err)
		return nil, ErrInternalProcessing
	}

	return tasks, nil
}

func (service *taskService) UpdateTask(ctx context.Context, userID, taskID, title, description string, priority domain.TaskPriority, dueDate time.Time) error {
	task, err := service.getAuthorizedTask(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if err := task.Update(title, description, priority, dueDate); err != nil {
		return err
	}

	return service.persistTaskUpdate(ctx, task, "failed to update task")
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

func (service *taskService) getAuthorizedBoard(ctx context.Context, userID, boardID string) (*domain.Board, error) {
	board, err := service.boardRepository.GetByID(ctx, boardID)
	if errors.Is(err, ports.ErrBoardNotFound) {
		return nil, ports.ErrBoardNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve task board", "userID", userID, "boardID", boardID, "error", err)
		return nil, ErrInternalProcessing
	}
	if board == nil {
		return nil, ports.ErrBoardNotFound
	}
	if board.UserID() != userID {
		service.logger.Warn("unauthorized task board access attempt", "boardID", boardID)
		return nil, ports.ErrBoardNotFound
	}

	return board, nil
}

func isTaskOwnedByUser(task *domain.Task, userID string) bool {
	return task.UserID() == userID
}
