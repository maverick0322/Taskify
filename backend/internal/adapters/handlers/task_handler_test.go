package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/adapters/handlers/middleware"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

type mockTaskUseCase struct {
	taskToReturn     *domain.Task
	tasksToReturn    []*domain.Task
	errToReturn      error
	requestedBoardID string
}

func (useCase *mockTaskUseCase) CreateTask(ctx context.Context, userID, boardID, title, description string, priority domain.TaskPriority, dueDate time.Time) (*domain.Task, error) {
	useCase.requestedBoardID = boardID
	return useCase.taskToReturn, useCase.errToReturn
}

func (useCase *mockTaskUseCase) GetTask(ctx context.Context, userID, taskID string) (*domain.Task, error) {
	return useCase.taskToReturn, useCase.errToReturn
}

func (useCase *mockTaskUseCase) GetUserTasks(ctx context.Context, userID string) ([]*domain.Task, error) {
	return useCase.tasksToReturn, useCase.errToReturn
}

func (useCase *mockTaskUseCase) GetBoardTasks(ctx context.Context, userID, boardID string) ([]*domain.Task, error) {
	useCase.requestedBoardID = boardID
	return useCase.tasksToReturn, useCase.errToReturn
}

func (useCase *mockTaskUseCase) UpdateTask(ctx context.Context, userID, taskID, title, description string, priority domain.TaskPriority, dueDate time.Time) error {
	return useCase.errToReturn
}

func (useCase *mockTaskUseCase) UpdateTaskDetails(ctx context.Context, userID, taskID, title, description string) error {
	return useCase.errToReturn
}

func (useCase *mockTaskUseCase) UpdateTaskStatus(ctx context.Context, userID, taskID string, status domain.TaskStatus) error {
	return useCase.errToReturn
}

func (useCase *mockTaskUseCase) UpdateTaskPriority(ctx context.Context, userID, taskID string, priority domain.TaskPriority) error {
	return useCase.errToReturn
}

func (useCase *mockTaskUseCase) DeleteTask(ctx context.Context, userID, taskID string) error {
	return useCase.errToReturn
}

func TestTaskHandler_CreateTaskValidRequest_ReturnsCreated(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{taskToReturn: createHandlerTask(t)})
	request := authenticatedTaskRequest(http.MethodPost, "/tasks", validCreateTaskJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"id":"task-123"`) {
		t.Errorf("expected response to contain task ID")
	}
}

func TestTaskHandler_CreateTaskMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPost, "/tasks", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_CreateTaskInvalidDueDate_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	requestBody := `{"title":"Write tests","description":"Cover handler","priority":"medium","dueDate":"01-01-2027"}`
	request := authenticatedTaskRequest(http.MethodPost, "/tasks", requestBody)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_CreateTaskDomainError_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: domain.ErrInvalidTaskTitle})
	request := authenticatedTaskRequest(http.MethodPost, "/tasks", validCreateTaskJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_CreateTaskInternalError_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: services.ErrInternalProcessing})
	request := authenticatedTaskRequest(http.MethodPost, "/tasks", validCreateTaskJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestTaskHandler_GetUserTasks_ReturnsOK(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{tasksToReturn: []*domain.Task{createHandlerTask(t)}})
	request := authenticatedTaskRequest(http.MethodGet, "/tasks", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"id":"task-123"`) {
		t.Errorf("expected response to contain task ID")
	}
}

func TestTaskHandler_GetBoardTasksWithBoardID_ReturnsOK(t *testing.T) {
	// Arrange
	useCase := &mockTaskUseCase{tasksToReturn: []*domain.Task{createHandlerTask(t)}}
	router := createTaskTestRouter(useCase)
	request := authenticatedTaskRequest(http.MethodGet, "/tasks?board_id=board-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if useCase.requestedBoardID != "board-123" {
		t.Errorf("expected requested board ID board-123, got %s", useCase.requestedBoardID)
	}
	if !strings.Contains(response.Body.String(), `"boardId":"board-123"`) {
		t.Errorf("expected response to contain board ID")
	}
}

func TestTaskHandler_GetTask_ReturnsOK(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{taskToReturn: createHandlerTask(t)})
	request := authenticatedTaskRequest(http.MethodGet, "/tasks/task-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestTaskHandler_GetTaskNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: ports.ErrTaskNotFound})
	request := authenticatedTaskRequest(http.MethodGet, "/tasks/task-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestTaskHandler_UpdateTaskValidRequest_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123", `{"title":"Review code","description":"Check handler","priority":"high","dueDate":"2027-01-01"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestTaskHandler_UpdateTaskInvalidDueDate_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123", `{"title":"Review code","description":"Check handler","priority":"high","dueDate":"01-01-2027"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_UpdateDetailsValidRequest_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/details", `{"title":"Review code","description":"Check handler"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestTaskHandler_UpdateDetailsMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/details", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_UpdateDetailsDomainError_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: domain.ErrInvalidTaskTitle})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/details", `{"title":"No","description":"Check handler"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_UpdateStatusValidRequest_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/status", `{"status":"done"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestTaskHandler_UpdateStatusMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/status", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_UpdateStatusInvalidStatus_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: domain.ErrInvalidTaskStatus})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/status", `{"status":"blocked"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_UpdateStatusInternalError_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: services.ErrInternalProcessing})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/status", `{"status":"done"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestTaskHandler_UpdatePriorityValidRequest_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/priority", `{"priority":"high"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestTaskHandler_UpdatePriorityMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/priority", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_UpdatePriorityInvalidPriority_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: domain.ErrInvalidTaskPriority})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/priority", `{"priority":"urgent"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTaskHandler_UpdatePriorityNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: ports.ErrTaskNotFound})
	request := authenticatedTaskRequest(http.MethodPatch, "/tasks/task-123/priority", `{"priority":"high"}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestTaskHandler_DeleteTask_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := authenticatedTaskRequest(http.MethodDelete, "/tasks/task-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestTaskHandler_DeleteTaskNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: ports.ErrTaskNotFound})
	request := authenticatedTaskRequest(http.MethodDelete, "/tasks/task-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestTaskHandler_InternalError_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{errToReturn: services.ErrInternalProcessing})
	request := authenticatedTaskRequest(http.MethodGet, "/tasks", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestTaskHandler_MissingUserContext_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	router := createTaskTestRouter(&mockTaskUseCase{})
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func createTaskTestRouter(useCase *mockTaskUseCase) chi.Router {
	router := chi.NewRouter()
	handler := NewTaskHandler(useCase, &mockHandlerLogger{})
	handler.RegisterRoutes(router)
	return router
}

func authenticatedTaskRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	return request.WithContext(middleware.ContextWithUserID(request.Context(), "user-123"))
}

func createHandlerTask(t *testing.T) *domain.Task {
	t.Helper()

	task, err := domain.RehydrateTask(
		"task-123",
		"user-123",
		"board-123",
		"Write tests",
		"Cover handler",
		domain.TaskStatusTodo,
		domain.TaskPriorityMedium,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-time.Hour),
		time.Now().Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("expected task to be valid, got: %v", err)
	}

	return task
}

func validCreateTaskJSON() string {
	return `{"boardId":"board-123","title":"Write tests","description":"Cover handler","priority":"medium","dueDate":"2027-01-01"}`
}
