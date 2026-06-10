package domain

import (
	"errors"
	"testing"
	"time"
)

const (
	validTaskID          = "task-123"
	validTaskUserID      = "user-123"
	validTaskBoardID     = "board-123"
	validTaskTitle       = "Write tests"
	validTaskDescription = "Cover domain business rules"
)

func TestNewTask_ValidFields_ReturnsTask(t *testing.T) {
	// Arrange
	dueDate := time.Now().Add(24 * time.Hour)

	// Act
	task, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, dueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if task.ID() != validTaskID {
		t.Errorf("expected task ID %s, got %s", validTaskID, task.ID())
	}
}

func TestNewTask_ValidFields_TrimsTextFields(t *testing.T) {
	// Arrange
	titleWithSpaces := "  Write tests  "
	descriptionWithSpaces := "  Cover domain business rules  "

	// Act
	task, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), titleWithSpaces, descriptionWithSpaces, TaskStatusTodo, TaskPriorityMedium, time.Time{})

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task.Title() != validTaskTitle {
		t.Errorf("expected title %s, got %s", validTaskTitle, task.Title())
	}
	if task.Description() != validTaskDescription {
		t.Errorf("expected description %s, got %s", validTaskDescription, task.Description())
	}
}

func TestNewTask_ZeroDueDate_ReturnsTask(t *testing.T) {
	// Arrange
	emptyDueDate := time.Time{}

	// Act
	task, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, emptyDueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !task.DueDate().IsZero() {
		t.Errorf("expected zero due date, got %v", task.DueDate())
	}
}

func TestNewTask_EmptyID_ReturnsErrEmptyTaskID(t *testing.T) {
	// Arrange
	emptyTaskID := ""

	// Act
	_, err := NewTask(emptyTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, time.Time{})

	// Assert
	if !errors.Is(err, ErrEmptyTaskID) {
		t.Errorf("expected error %v, got %v", ErrEmptyTaskID, err)
	}
}

func TestNewTask_EmptyUserID_ReturnsErrEmptyTaskUserID(t *testing.T) {
	// Arrange
	emptyTaskUserID := ""

	// Act
	_, err := NewTask(validTaskID, emptyTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, time.Time{})

	// Assert
	if !errors.Is(err, ErrEmptyTaskUserID) {
		t.Errorf("expected error %v, got %v", ErrEmptyTaskUserID, err)
	}
}

func TestNewTask_EmptyBoardID_CreatesGlobalTask(t *testing.T) {
	// Arrange
	emptyTaskBoardID := ""

	// Act
	task, err := NewTask(validTaskID, validTaskUserID, &emptyTaskBoardID, validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, time.Time{})

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task.BoardID() != nil {
		t.Errorf("expected global task board ID to be nil")
	}
}

func TestNewTask_ShortTitle_ReturnsErrInvalidTaskTitle(t *testing.T) {
	// Arrange
	shortTitle := "Go"

	// Act
	_, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), shortTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, time.Time{})

	// Assert
	if !errors.Is(err, ErrInvalidTaskTitle) {
		t.Errorf("expected error %v, got %v", ErrInvalidTaskTitle, err)
	}
}

func TestNewTask_InvalidStatus_ReturnsErrInvalidTaskStatus(t *testing.T) {
	// Arrange
	invalidStatus := TaskStatus("blocked")

	// Act
	_, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, invalidStatus, TaskPriorityMedium, time.Time{})

	// Assert
	if !errors.Is(err, ErrInvalidTaskStatus) {
		t.Errorf("expected error %v, got %v", ErrInvalidTaskStatus, err)
	}
}

func TestNewTask_InvalidPriority_ReturnsErrInvalidTaskPriority(t *testing.T) {
	// Arrange
	invalidPriority := TaskPriority("urgent")

	// Act
	_, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, invalidPriority, time.Time{})

	// Assert
	if !errors.Is(err, ErrInvalidTaskPriority) {
		t.Errorf("expected error %v, got %v", ErrInvalidTaskPriority, err)
	}
}

func TestNewTask_PastDueDate_ReturnsTask(t *testing.T) {
	// Arrange
	pastDueDate := time.Now().AddDate(0, 0, -1)

	// Act
	task, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, pastDueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !task.DueDate().Equal(pastDueDate) {
		t.Errorf("expected due date %v, got %v", pastDueDate, task.DueDate())
	}
}

func TestNewTask_TodayDueDate_ReturnsTask(t *testing.T) {
	// Arrange
	now := time.Now()
	todayDueDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Act
	task, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, todayDueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !task.DueDate().Equal(todayDueDate) {
		t.Errorf("expected due date %v, got %v", todayDueDate, task.DueDate())
	}
}

func TestTask_Getters_ReturnExpectedValues(t *testing.T) {
	// Arrange
	dueDate := time.Now().Add(24 * time.Hour)
	task, _ := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusInProgress, TaskPriorityHigh, dueDate)

	// Act
	retrievedUserID := task.UserID()
	retrievedBoardID := task.BoardID()
	retrievedTitle := task.Title()
	retrievedDescription := task.Description()
	retrievedStatus := task.Status()
	retrievedPriority := task.Priority()
	retrievedCreatedAt := task.CreatedAt()
	retrievedUpdatedAt := task.UpdatedAt()
	retrievedDueDate := task.DueDate()

	// Assert
	if retrievedUserID != validTaskUserID {
		t.Errorf("expected user ID %s, got %s", validTaskUserID, retrievedUserID)
	}
	if retrievedBoardID == nil || *retrievedBoardID != validTaskBoardID {
		t.Errorf("expected board ID %s, got %v", validTaskBoardID, retrievedBoardID)
	}
	if retrievedTitle != validTaskTitle {
		t.Errorf("expected title %s, got %s", validTaskTitle, retrievedTitle)
	}
	if retrievedDescription != validTaskDescription {
		t.Errorf("expected description %s, got %s", validTaskDescription, retrievedDescription)
	}
	if retrievedStatus != TaskStatusInProgress {
		t.Errorf("expected status %s, got %s", TaskStatusInProgress, retrievedStatus)
	}
	if retrievedPriority != TaskPriorityHigh {
		t.Errorf("expected priority %s, got %s", TaskPriorityHigh, retrievedPriority)
	}
	if retrievedCreatedAt.IsZero() {
		t.Fatal("expected created at to be set")
	}
	if !retrievedUpdatedAt.Equal(retrievedCreatedAt) {
		t.Errorf("expected updated at %v, got %v", retrievedCreatedAt, retrievedUpdatedAt)
	}
	if !retrievedDueDate.Equal(dueDate) {
		t.Errorf("expected due date %v, got %v", dueDate, retrievedDueDate)
	}
}

func TestTaskStatus_IsValid_ReturnsExpectedValues(t *testing.T) {
	// Arrange
	validStatus := TaskStatusDone
	invalidStatus := TaskStatus("blocked")

	// Act
	validStatusResult := validStatus.IsValid()
	invalidStatusResult := invalidStatus.IsValid()

	// Assert
	if !validStatusResult {
		t.Error("expected valid status to return true")
	}
	if invalidStatusResult {
		t.Error("expected invalid status to return false")
	}
}

func TestTaskPriority_IsValid_ReturnsExpectedValues(t *testing.T) {
	// Arrange
	validPriority := TaskPriorityLow
	invalidPriority := TaskPriority("urgent")

	// Act
	validPriorityResult := validPriority.IsValid()
	invalidPriorityResult := invalidPriority.IsValid()

	// Assert
	if !validPriorityResult {
		t.Error("expected valid priority to return true")
	}
	if invalidPriorityResult {
		t.Error("expected invalid priority to return false")
	}
}

func TestTask_ChangeStatusValidStatus_UpdatesStatusAndUpdatedAt(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	previousUpdatedAt := task.UpdatedAt()
	waitForTimestampChange()

	// Act
	err := task.ChangeStatus(TaskStatusDone)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task.Status() != TaskStatusDone {
		t.Errorf("expected status %s, got %s", TaskStatusDone, task.Status())
	}
	if !task.UpdatedAt().After(previousUpdatedAt) {
		t.Errorf("expected updated at to be after %v, got %v", previousUpdatedAt, task.UpdatedAt())
	}
}

func TestTask_ChangeStatusInvalidStatus_ReturnsErrInvalidTaskStatus(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	invalidStatus := TaskStatus("blocked")

	// Act
	err := task.ChangeStatus(invalidStatus)

	// Assert
	if !errors.Is(err, ErrInvalidTaskStatus) {
		t.Errorf("expected error %v, got %v", ErrInvalidTaskStatus, err)
	}
}

func TestTask_UpdateDetailsValidFields_UpdatesDetailsAndUpdatedAt(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	previousUpdatedAt := task.UpdatedAt()
	newTitle := "Review pull request"
	newDescription := "Check test coverage"
	waitForTimestampChange()

	// Act
	err := task.UpdateDetails(newTitle, newDescription)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task.Title() != newTitle {
		t.Errorf("expected title %s, got %s", newTitle, task.Title())
	}
	if task.Description() != newDescription {
		t.Errorf("expected description %s, got %s", newDescription, task.Description())
	}
	if !task.UpdatedAt().After(previousUpdatedAt) {
		t.Errorf("expected updated at to be after %v, got %v", previousUpdatedAt, task.UpdatedAt())
	}
}

func TestTask_UpdateDetailsShortTitle_ReturnsErrInvalidTaskTitle(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	shortTitle := "No"

	// Act
	err := task.UpdateDetails(shortTitle, validTaskDescription)

	// Assert
	if !errors.Is(err, ErrInvalidTaskTitle) {
		t.Errorf("expected error %v, got %v", ErrInvalidTaskTitle, err)
	}
}

func TestTask_ChangePriorityValidPriority_UpdatesPriorityAndUpdatedAt(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	previousUpdatedAt := task.UpdatedAt()
	waitForTimestampChange()

	// Act
	err := task.ChangePriority(TaskPriorityHigh)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task.Priority() != TaskPriorityHigh {
		t.Errorf("expected priority %s, got %s", TaskPriorityHigh, task.Priority())
	}
	if !task.UpdatedAt().After(previousUpdatedAt) {
		t.Errorf("expected updated at to be after %v, got %v", previousUpdatedAt, task.UpdatedAt())
	}
}

func TestTask_ChangePriorityInvalidPriority_ReturnsErrInvalidTaskPriority(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	invalidPriority := TaskPriority("urgent")

	// Act
	err := task.ChangePriority(invalidPriority)

	// Assert
	if !errors.Is(err, ErrInvalidTaskPriority) {
		t.Errorf("expected error %v, got %v", ErrInvalidTaskPriority, err)
	}
}

func TestTask_UpdateValidFields_UpdatesTaskAndUpdatedAt(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	previousUpdatedAt := task.UpdatedAt()
	newDueDate := time.Now().Add(48 * time.Hour)
	waitForTimestampChange()

	// Act
	err := task.Update("Review pull request", "Check test coverage", TaskPriorityHigh, newDueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if task.Title() != "Review pull request" {
		t.Errorf("expected updated title, got %s", task.Title())
	}
	if task.Priority() != TaskPriorityHigh {
		t.Errorf("expected priority %s, got %s", TaskPriorityHigh, task.Priority())
	}
	if !task.DueDate().Equal(newDueDate) {
		t.Errorf("expected due date %v, got %v", newDueDate, task.DueDate())
	}
	if !task.UpdatedAt().After(previousUpdatedAt) {
		t.Errorf("expected updated at to be after %v, got %v", previousUpdatedAt, task.UpdatedAt())
	}
}

func TestTask_UpdatePastDueDate_UpdatesTask(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	pastDueDate := time.Now().AddDate(0, 0, -1)

	// Act
	err := task.Update("Review pull request", "Check test coverage", TaskPriorityHigh, pastDueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !task.DueDate().Equal(pastDueDate) {
		t.Errorf("expected due date %v, got %v", pastDueDate, task.DueDate())
	}
}

func TestTask_UpdateTodayDueDate_UpdatesTask(t *testing.T) {
	// Arrange
	task := createValidTask(t)
	now := time.Now()
	todayDueDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Act
	err := task.Update("Review pull request", "Check test coverage", TaskPriorityHigh, todayDueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !task.DueDate().Equal(todayDueDate) {
		t.Errorf("expected due date %v, got %v", todayDueDate, task.DueDate())
	}
}

func TestRehydrateTask_ValidFields_ReturnsTaskWithPersistedDates(t *testing.T) {
	// Arrange
	createdAt := time.Now().Add(-2 * time.Hour)
	updatedAt := time.Now().Add(-time.Hour)
	dueDate := time.Now().Add(24 * time.Hour)

	// Act
	task, err := RehydrateTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusDone, TaskPriorityHigh, createdAt, updatedAt, dueDate)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !task.CreatedAt().Equal(createdAt) {
		t.Errorf("expected created at %v, got %v", createdAt, task.CreatedAt())
	}
	if !task.UpdatedAt().Equal(updatedAt) {
		t.Errorf("expected updated at %v, got %v", updatedAt, task.UpdatedAt())
	}
	if task.Status() != TaskStatusDone {
		t.Errorf("expected status %s, got %s", TaskStatusDone, task.Status())
	}
	if task.Priority() != TaskPriorityHigh {
		t.Errorf("expected priority %s, got %s", TaskPriorityHigh, task.Priority())
	}
}

func TestRehydrateTask_ZeroCreatedAt_ReturnsErrInvalidTaskCreatedAt(t *testing.T) {
	// Arrange
	zeroCreatedAt := time.Time{}
	updatedAt := time.Now().Add(-time.Hour)

	// Act
	_, err := RehydrateTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, zeroCreatedAt, updatedAt, time.Time{})

	// Assert
	if !errors.Is(err, ErrInvalidTaskCreatedAt) {
		t.Errorf("expected error %v, got %v", ErrInvalidTaskCreatedAt, err)
	}
}

func TestRehydrateTask_ZeroUpdatedAt_ReturnsErrInvalidTaskUpdatedAt(t *testing.T) {
	// Arrange
	createdAt := time.Now().Add(-2 * time.Hour)
	zeroUpdatedAt := time.Time{}

	// Act
	_, err := RehydrateTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, createdAt, zeroUpdatedAt, time.Time{})

	// Assert
	if !errors.Is(err, ErrInvalidTaskUpdatedAt) {
		t.Errorf("expected error %v, got %v", ErrInvalidTaskUpdatedAt, err)
	}
}

func createValidTask(t *testing.T) *Task {
	t.Helper()

	task, err := NewTask(validTaskID, validTaskUserID, taskBoardIDPtr(validTaskBoardID), validTaskTitle, validTaskDescription, TaskStatusTodo, TaskPriorityMedium, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("expected task to be valid, got: %v", err)
	}

	return task
}

func taskBoardIDPtr(boardID string) *string {
	return &boardID
}

func waitForTimestampChange() {
	time.Sleep(time.Millisecond)
}
