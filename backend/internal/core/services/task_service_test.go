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
	validTaskServiceTaskID      = "task-123"
	validTaskServiceTitle       = "Write tests"
	validTaskServiceDescription = "Cover task service rules"
)

type mockTaskRepository struct {
	taskToReturn     *domain.Task
	tasksToReturn    []*domain.Task
	saveError        error
	getByIDError     error
	getByUserIDError error
	updateError      error
	deleteError      error
	savedTask        *domain.Task
	updatedTask      *domain.Task
	deletedTaskID    string
	requestedTaskID  string
	requestedUserID  string
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

func TestCreateTask_ValidData_ReturnsTaskAndSaves(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{}
	service := NewTaskService(repository, &mockTaskIDGenerator{id: validTaskServiceTaskID}, &mockTaskLogger{})

	// Act
	task, err := service.CreateTask(context.Background(), validTaskServiceUserID, validTaskServiceTitle, validTaskServiceDescription, domain.TaskPriorityMedium, time.Time{})

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
	service := NewTaskService(&mockTaskRepository{}, &mockTaskIDGenerator{id: validTaskServiceTaskID}, &mockTaskLogger{})

	// Act
	_, err := service.CreateTask(context.Background(), validTaskServiceUserID, "No", validTaskServiceDescription, domain.TaskPriorityMedium, time.Time{})

	// Assert
	if !errors.Is(err, domain.ErrInvalidTaskTitle) {
		t.Errorf("expected error %v, got %v", domain.ErrInvalidTaskTitle, err)
	}
}

func TestCreateTask_SaveFailure_ReturnsErrInternalProcessing(t *testing.T) {
	// Arrange
	repository := &mockTaskRepository{saveError: ports.ErrTaskRepositoryUnavailable}
	service := NewTaskService(repository, &mockTaskIDGenerator{id: validTaskServiceTaskID}, &mockTaskLogger{})

	// Act
	_, err := service.CreateTask(context.Background(), validTaskServiceUserID, validTaskServiceTitle, validTaskServiceDescription, domain.TaskPriorityMedium, time.Time{})

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestGetTask_OwnedTask_ReturnsTask(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, logger)

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

	// Act
	_, err := service.GetUserTasks(context.Background(), validTaskServiceUserID)

	// Assert
	if !errors.Is(err, ErrInternalProcessing) {
		t.Errorf("expected error %v, got %v", ErrInternalProcessing, err)
	}
}

func TestUpdateTaskDetails_OwnedTask_UpdatesAndPersists(t *testing.T) {
	// Arrange
	task := createTaskServiceTask(t, validTaskServiceUserID)
	repository := &mockTaskRepository{taskToReturn: task}
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
	service := NewTaskService(repository, &mockTaskIDGenerator{}, &mockTaskLogger{})

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
