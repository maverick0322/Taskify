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
	taskRepository   ports.TaskRepository
	boardRepository  ports.BoardRepository
	columnRepository ports.ColumnRepository
	idGenerator      ports.IDGenerator
	logger           ports.Logger
}

func NewTaskService(
	taskRepository ports.TaskRepository,
	boardRepository ports.BoardRepository,
	serviceOptions ...interface{},
) ports.TaskUseCase {
	var columnRepository ports.ColumnRepository
	var idGenerator ports.IDGenerator
	var logger ports.Logger
	if len(serviceOptions) == 3 {
		columnRepository, _ = serviceOptions[0].(ports.ColumnRepository)
		idGenerator, _ = serviceOptions[1].(ports.IDGenerator)
		logger, _ = serviceOptions[2].(ports.Logger)
	} else if len(serviceOptions) == 2 {
		idGenerator, _ = serviceOptions[0].(ports.IDGenerator)
		logger, _ = serviceOptions[1].(ports.Logger)
	}

	return &taskService{
		taskRepository:   taskRepository,
		boardRepository:  boardRepository,
		columnRepository: columnRepository,
		idGenerator:      idGenerator,
		logger:           logger,
	}
}

func (service *taskService) CreateTask(
	ctx context.Context,
	userID string,
	boardID *string,
	options ...interface{},
) (*domain.Task, error) {
	columnID, title, description, priority, dueDate, err := parseTaskCreationOptions(options...)
	if err != nil {
		return nil, err
	}
	taskID := service.idGenerator.Generate()
	task, err := domain.NewTask(taskID, userID, boardID, columnID, title, description, domain.TaskStatusTodo, priority, dueDate)
	if err != nil {
		return nil, err
	}

	if task.BoardID() != nil {
		if _, err := service.getAuthorizedBoard(ctx, userID, *task.BoardID()); err != nil {
			return nil, err
		}
	}
	if err := service.validateTaskColumn(ctx, userID, task.BoardID(), task.ColumnID()); err != nil {
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

func (service *taskService) UpdateTask(ctx context.Context, userID, taskID, title, description string, options ...interface{}) error {
	columnID, priority, dueDate, err := parseTaskUpdateOptions(options...)
	if err != nil {
		return err
	}
	task, err := service.getAuthorizedTask(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if err := task.Update(title, description, priority, dueDate); err != nil {
		return err
	}
	if err := service.validateTaskColumn(ctx, userID, task.BoardID(), columnID); err != nil {
		return err
	}
	task.MoveToColumn(columnID)

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

func (service *taskService) MoveTaskToColumn(ctx context.Context, userID, taskID string, columnID *string) error {
	task, err := service.getAuthorizedTask(ctx, userID, taskID)
	if err != nil {
		return err
	}

	if err := service.validateTaskColumn(ctx, userID, task.BoardID(), columnID); err != nil {
		return err
	}
	task.MoveToColumn(columnID)

	return service.persistTaskUpdate(ctx, task, "failed to move task to column")
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

func (service *taskService) validateTaskColumn(ctx context.Context, userID string, boardID, columnID *string) error {
	if columnID == nil {
		return nil
	}
	if boardID == nil {
		return ports.ErrBoardNotFound
	}

	column, err := service.columnRepository.GetByID(ctx, *columnID)
	if errors.Is(err, ports.ErrColumnNotFound) {
		return ports.ErrColumnNotFound
	}
	if err != nil {
		service.logger.Error("failed to retrieve task column", "userID", userID, "columnID", *columnID, "error", err)
		return ErrInternalProcessing
	}
	if column == nil || column.BoardID() != *boardID {
		return ports.ErrColumnNotFound
	}
	if _, err := service.getAuthorizedBoard(ctx, userID, column.BoardID()); err != nil {
		return err
	}

	return nil
}

func isTaskOwnedByUser(task *domain.Task, userID string) bool {
	return task.UserID() == userID
}

func parseTaskCreationOptions(options ...interface{}) (*string, string, string, domain.TaskPriority, time.Time, error) {
	switch len(options) {
	case 4:
		title, titleOK := options[0].(string)
		description, descriptionOK := options[1].(string)
		priority, priorityOK := options[2].(domain.TaskPriority)
		dueDate, dueDateOK := options[3].(time.Time)
		if !titleOK || !descriptionOK || !priorityOK || !dueDateOK {
			return nil, "", "", "", time.Time{}, domain.ErrInvalidTaskPriority
		}
		return nil, title, description, priority, dueDate, nil
	case 5:
		columnID, columnOK := options[0].(*string)
		title, titleOK := options[1].(string)
		description, descriptionOK := options[2].(string)
		priority, priorityOK := options[3].(domain.TaskPriority)
		dueDate, dueDateOK := options[4].(time.Time)
		if !columnOK || !titleOK || !descriptionOK || !priorityOK || !dueDateOK {
			return nil, "", "", "", time.Time{}, domain.ErrInvalidTaskPriority
		}
		return columnID, title, description, priority, dueDate, nil
	default:
		return nil, "", "", "", time.Time{}, domain.ErrInvalidTaskPriority
	}
}

func parseTaskUpdateOptions(options ...interface{}) (*string, domain.TaskPriority, time.Time, error) {
	switch len(options) {
	case 2:
		priority, priorityOK := options[0].(domain.TaskPriority)
		dueDate, dueDateOK := options[1].(time.Time)
		if !priorityOK || !dueDateOK {
			return nil, "", time.Time{}, domain.ErrInvalidTaskPriority
		}
		return nil, priority, dueDate, nil
	case 3:
		columnID, columnOK := options[0].(*string)
		priority, priorityOK := options[1].(domain.TaskPriority)
		dueDate, dueDateOK := options[2].(time.Time)
		if !columnOK || !priorityOK || !dueDateOK {
			return nil, "", time.Time{}, domain.ErrInvalidTaskPriority
		}
		return columnID, priority, dueDate, nil
	default:
		return nil, "", time.Time{}, domain.ErrInvalidTaskPriority
	}
}
