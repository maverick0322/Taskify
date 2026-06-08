package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	validTaskServiceUserID      = "user-123"
	validTaskServiceBoardID     = "board-123"
	validTaskServiceTaskID      = "task-123"
	validTaskServiceTitle       = "Write tests"
	validTaskServiceDescription = "Cover task service rules"
)

type mockTaskRepository struct {
	taskToReturn               *domain.Task
	tasksToReturn              []*domain.Task
	saveError                  error
	getByIDError               error
	getByUserIDError           error
	getByUserIDAndBoardIDError error
	updateError                error
	deleteError                error
	savedTask                  *domain.Task
	updatedTask                *domain.Task
	deletedTaskID              string
	requestedTaskID            string
	requestedUserID            string
	requestedBoardID           string
}

func (repository *mockTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	repository.savedTask = task
	return repository.saveError
}

func (repository *mockTaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	repository.requestedTaskID = id
	return repository.taskToReturn, repository.getByIDError
}

func (repository *mockTaskRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Task, error) {
	repository.requestedUserID = userID
	return repository.tasksToReturn, repository.getByUserIDError
}

func (repository *mockTaskRepository) GetByUserIDAndBoardID(ctx context.Context, userID, boardID string) ([]*domain.Task, error) {
	repository.requestedUserID = userID
	repository.requestedBoardID = boardID
	return repository.tasksToReturn, repository.getByUserIDAndBoardIDError
}

func (repository *mockTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	repository.updatedTask = task
	return repository.updateError
}

func (repository *mockTaskRepository) Delete(ctx context.Context, id string) error {
	repository.deletedTaskID = id
	return repository.deleteError
}

type mockTaskIDGenerator struct {
	id string
}

func (generator *mockTaskIDGenerator) Generate() string {
	return generator.id
}

type mockTaskLogger struct {
	warnMessages  []string
	errorMessages []string
}

func (logger *mockTaskLogger) Info(msg string, keysAndValues ...interface{}) {}

func (logger *mockTaskLogger) Warn(msg string, keysAndValues ...interface{}) {
	logger.warnMessages = append(logger.warnMessages, msg)
}

func (logger *mockTaskLogger) Error(msg string, keysAndValues ...interface{}) {
	logger.errorMessages = append(logger.errorMessages, msg)
}

type mockTaskBoardRepository struct {
	boardToReturn *domain.Board
	getByIDError  error
	requestedID   string
}

func (repository *mockTaskBoardRepository) Save(ctx context.Context, board *domain.Board) error {
	return nil
}

func (repository *mockTaskBoardRepository) GetByID(ctx context.Context, id string) (*domain.Board, error) {
	repository.requestedID = id
	return repository.boardToReturn, repository.getByIDError
}

func (repository *mockTaskBoardRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Board, error) {
	return nil, nil
}

func (repository *mockTaskBoardRepository) Update(ctx context.Context, board *domain.Board) error {
	return nil
}

func (repository *mockTaskBoardRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func newTaskTestService(t *testing.T, repository *mockTaskRepository, idGenerator *mockTaskIDGenerator, logger *mockTaskLogger) ports.TaskUseCase {
	t.Helper()

	return NewTaskService(repository, &mockTaskBoardRepository{boardToReturn: createTaskServiceBoard(t, validTaskServiceUserID)}, idGenerator, logger)
}

func TestCreateTask_ValidData_ReturnsTaskAndSaves(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{id: validTaskServiceTaskID}, &mockTaskLogger{})

	// Act
	task, err := service.CreateTask(context.Background(), validTaskServiceUserID, validTaskServiceBoardID, validTaskServiceTitle, validTaskServiceDescription, domain.TaskPriorityMedium, time.Time{})

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task.ID() != validTaskServiceTaskID {
		t.Errorf("expected task ID %s, got %s", validTaskServiceTaskID, task.ID())
	}
	if task.Status() != domain.TaskStatusTodo {
		t.Errorf("expected status %s, got %s", domain.TaskStatusTodo, task.Status())
	}
	if repository.savedTask == nil {
		t.Fatal("expected task to be saved")
	}
}

func TestCreateTask_InvalidTitle_ReturnsDomainError(t *testing.T) {
	// Arrange
	service := newTaskTestService(t, &mockTaskRepository{}, &mockTaskIDGenerator{id: validTaskServiceTaskID}, &mockTaskLogger{})

	// Act
	_, err := service.CreateTask(context.Background(), validTaskServiceUserID, validTaskServiceBoardID, "No", validTaskServiceDescription, domain.TaskPriorityMedium, time.Time{})

	// Assert
	if !errors.Is(err, domain.ErrInvalidTaskTitle) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTaskTitle, err)
	}
}

func TestCreateTask_SaveFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{saveError: ports.ErrTaskRepositoryUnavailable}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{id: validTaskServiceTaskID}, &mockTaskLogger{})

	// Act
	_, err := service.CreateTask(context.Background(), validTaskServiceUserID, validTaskServiceBoardID, validTaskServiceTitle, validTaskServiceDescription, domain.TaskPriorityMedium, time.Time{})

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestGetTask_OwnedTask_ReturnsTask(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	retrievedTask, err := service.GetTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if retrievedTask.ID() != validTaskServiceTaskID {
		t.Errorf("expected task ID %s, got %s", validTaskServiceTaskID, retrievedTask.ID())
	}
}

func TestGetTask_MissingTask_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{getByIDError: ports.ErrTaskNotFound}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	_, err := service.GetTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID)

	// Assert
	if !errors.Is(err, ports.ErrTaskNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskNotFound, err)
	}
}

func TestGetTask_NilTask_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	_, err := service.GetTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID)

	// Assert
	if !errors.Is(err, ports.ErrTaskNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskNotFound, err)
	}
}

func TestGetTask_RepositoryUnavailable_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{getByIDError: ports.ErrTaskRepositoryUnavailable}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	_, err := service.GetTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestGetTask_TaskOwnedByAnotherUser_ReturnsErrTaskNotFoundAndWarns(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, "other-user-123")
	repository := &mockTaskRepository{taskToReturn: task}
	logger := &mockTaskLogger{}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, logger)

	// Act
	_, err := service.GetTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID)

	// Assert
	if !errors.Is(err, ports.ErrTaskNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskNotFound, err)
	}
	if len(logger.warnMessages) != 1 {
		t.Fatalf("expected one warning log, got %d", len(logger.warnMessages))
	}
}

func TestGetUserTasks_RepositorySuccess_ReturnsTasks(t *testing.T) {
	// Arrange
	tasks := []*domain.Task{createTaskServiceTask(t, validTaskServiceUserID)}
	repository := &mockTaskRepository{tasksToReturn: tasks}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	retrievedTasks, err := service.GetUserTasks(context.Background(), validTaskServiceUserID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(retrievedTasks) != 1 {
		t.Fatalf("expected one task, got %d", len(retrievedTasks))
	}
	if repository.requestedUserID != validTaskServiceUserID {
		t.Errorf("expected requested user ID %s, got %s", validTaskServiceUserID, repository.requestedUserID)
	}
}

func TestGetUserTasks_RepositoryFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{getByUserIDError: ports.ErrTaskRepositoryUnavailable}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	_, err := service.GetUserTasks(context.Background(), validTaskServiceUserID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestGetBoardTasks_AuthorizedBoard_ReturnsTasks(t *testing.T) {
	// Arrange
	tasks := []*domain.Task{createTaskServiceTask(t, validTaskServiceUserID)}
	repository := &mockTaskRepository{tasksToReturn: tasks}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	retrievedTasks, err := service.GetBoardTasks(context.Background(), validTaskServiceUserID, validTaskServiceBoardID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(retrievedTasks) != 1 {
		t.Fatalf("expected one task, got %d", len(retrievedTasks))
	}
	if repository.requestedBoardID != validTaskServiceBoardID {
		t.Errorf("expected requested board ID %s, got %s", validTaskServiceBoardID, repository.requestedBoardID)
	}
}

func TestGetBoardTasks_UnauthorizedBoard_ReturnsErrBoardNotFound(t *testing.T) {
	// Arrange
	boardRepository := &mockTaskBoardRepository{boardToReturn: createTaskServiceBoard(t, "other-user-123")}
	service := NewTaskService(&mockTaskRepository{}, boardRepository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	_, err := service.GetBoardTasks(context.Background(), validTaskServiceUserID, validTaskServiceBoardID)

	// Assert
	if !errors.Is(err, ports.ErrBoardNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrBoardNotFound, err)
	}
}

func TestGetBoardTasks_RepositoryFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{getByUserIDAndBoardIDError: ports.ErrTaskRepositoryUnavailable}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	_, err := service.GetBoardTasks(context.Background(), validTaskServiceUserID, validTaskServiceBoardID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestUpdateTask_OwnedTask_UpdatesAndPersists(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})
	dueDate := time.Now().Add(48 * time.Hour)

	// Act
	err := service.UpdateTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, "Review code", "Check edge cases", domain.TaskPriorityHigh, dueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.updatedTask == nil {
		t.Fatal("expected task to be updated")
	}
	if repository.updatedTask.Title() != "Review code" {
		t.Errorf("expected title Review code, got %s", repository.updatedTask.Title())
	}
	if repository.updatedTask.Priority() != domain.TaskPriorityHigh {
		t.Errorf("expected priority %s, got %s", domain.TaskPriorityHigh, repository.updatedTask.Priority())
	}
	if !repository.updatedTask.DueDate().Equal(dueDate) {
		t.Errorf("expected due date %v, got %v", dueDate, repository.updatedTask.DueDate())
	}
}

func TestUpdateTask_InvalidPriority_ReturnsDomainError(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, "Review code", "Check edge cases", domain.TaskPriority("urgent"), time.Time{})

	// Assert
	if !errors.Is(err, domain.ErrInvalidTaskPriority) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTaskPriority, err)
	}
}

func TestUpdateTaskDetails_OwnedTask_UpdatesAndPersists(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskDetails(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, "Review code", "Check edge cases")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.updatedTask == nil {
		t.Fatal("expected task to be updated")
	}
	if repository.updatedTask.Title() != "Review code" {
		t.Errorf("expected title Review code, got %s", repository.updatedTask.Title())
	}
}

func TestUpdateTaskDetails_InvalidTitle_ReturnsDomainError(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskDetails(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, "No", "Check edge cases")

	// Assert
	if !errors.Is(err, domain.ErrInvalidTaskTitle) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTaskTitle, err)
	}
}

func TestUpdateTaskDetails_UnauthorizedTask_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, "other-user-123")
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskDetails(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, "Review code", "Check edge cases")

	// Assert
	if !errors.Is(err, ports.ErrTaskNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskNotFound, err)
	}
}

func TestUpdateTaskDetails_UpdateFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task, updateError: ports.ErrTaskRepositoryUnavailable}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskDetails(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, "Review code", "Check edge cases")

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestUpdateTaskStatus_OwnedTask_UpdatesAndPersists(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskStatus(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, domain.TaskStatusDone)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.updatedTask.Status() != domain.TaskStatusDone {
		t.Errorf("expected status %s, got %s", domain.TaskStatusDone, repository.updatedTask.Status())
	}
}

func TestUpdateTaskStatus_InvalidStatus_ReturnsDomainError(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskStatus(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, domain.TaskStatus("blocked"))

	// Assert
	if !errors.Is(err, domain.ErrInvalidTaskStatus) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTaskStatus, err)
	}
}

func TestUpdateTaskPriority_OwnedTask_UpdatesAndPersists(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskPriority(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, domain.TaskPriorityHigh)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.updatedTask.Priority() != domain.TaskPriorityHigh {
		t.Errorf("expected priority %s, got %s", domain.TaskPriorityHigh, repository.updatedTask.Priority())
	}
}

func TestUpdateTaskPriority_InvalidPriority_ReturnsDomainError(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskPriority(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, domain.TaskPriority("urgent"))

	// Assert
	if !errors.Is(err, domain.ErrInvalidTaskPriority) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTaskPriority, err)
	}
}

func TestUpdateTaskPriority_UnauthorizedTask_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, "other-user-123")
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.UpdateTaskPriority(context.Background(), validTaskServiceUserID, validTaskServiceTaskID, domain.TaskPriorityHigh)

	// Assert
	if !errors.Is(err, ports.ErrTaskNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskNotFound, err)
	}
}

func TestDeleteTask_OwnedTask_DeletesTask(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.DeleteTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if repository.deletedTaskID != validTaskServiceTaskID {
		t.Errorf("expected deleted task ID %s, got %s", validTaskServiceTaskID, repository.deletedTaskID)
	}
}

func TestDeleteTask_UnauthorizedTask_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, "other-user-123")
	repository := &mockTaskRepository{taskToReturn: task}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.DeleteTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID)

	// Assert
	if !errors.Is(err, ports.ErrTaskNotFound) {
		t.Errorf("expected error %v, got %v", ports.ErrTaskNotFound, err)
	}
}

func TestDeleteTask_DeleteFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task, deleteError: ports.ErrTaskRepositoryUnavailable}
	service := newTaskTestService(t, repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	err := service.DeleteTask(context.Background(), validTaskServiceUserID, validTaskServiceTaskID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func createTaskServiceTask(t *testing.T, userID string) *domain.Task {
	t.Helper()

	task, err := domain.NewTask(
		validTaskServiceTaskID,
		userID,
		validTaskServiceBoardID,
		validTaskServiceTitle,
		validTaskServiceDescription,
		domain.TaskStatusTodo,
		domain.TaskPriorityMedium,
		time.Now().Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("expected task to be valid, got: %v", err)
	}

	return task
}

func createTaskServiceBoard(t *testing.T, userID string) *domain.Board {
	t.Helper()

	board, err := domain.NewBoard(validTaskServiceBoardID, userID, "Project Board")
	if err != nil {
		t.Fatalf("expected board to be valid, got: %v", err)
	}

	return board
}
