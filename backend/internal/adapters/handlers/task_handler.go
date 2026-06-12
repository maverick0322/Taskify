package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/adapters/handlers/middleware"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

type TaskHandler struct {
	taskUseCase ports.TaskUseCase
	logger      ports.Logger
}

func NewTaskHandler(taskUseCase ports.TaskUseCase, logger ports.Logger) *TaskHandler {
	return &TaskHandler{
		taskUseCase: taskUseCase,
		logger:      logger,
	}
}

func (handler *TaskHandler) RegisterRoutes(router chi.Router) {
	router.Post("/tasks", handler.CreateTask)
	router.Get("/tasks", handler.GetUserTasks)
	router.Get("/tasks/{id}", handler.GetTask)
	router.Patch("/tasks/{id}", handler.UpdateTask)
	router.Patch("/tasks/{id}/details", handler.UpdateTaskDetails)
	router.Patch("/tasks/{id}/status", handler.UpdateTaskStatus)
	router.Patch("/tasks/{id}/priority", handler.UpdateTaskPriority)
	router.Patch("/tasks/{id}/column", handler.MoveTaskToColumn)
	router.Delete("/tasks/{id}", handler.DeleteTask)
}

// CreateTask creates a task for the authenticated user.
// @Summary Create task
// @Tags Tasks
// @Accept json
// @Produce json
// @Param request body createTaskRequest true "Task creation payload"
// @Success 201 {object} taskResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /tasks [post]
func (handler *TaskHandler) CreateTask(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var createRequest createTaskRequest
	if err := json.NewDecoder(request.Body).Decode(&createRequest); err != nil {
		handler.logger.Warn("create task request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	dueDate, err := parseTaskDueDate(createRequest.DueDate)
	if err != nil {
		handler.logger.Warn("create task request contains invalid due date", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid due date"})
		return
	}

	task, err := handler.taskUseCase.CreateTask(
		request.Context(),
		userID,
		createRequest.BoardID,
		createRequest.ColumnID,
		createRequest.Title,
		createRequest.Description,
		domain.TaskPriority(createRequest.Priority),
		dueDate,
	)
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	writeJSON(response, http.StatusCreated, taskResponseFromDomain(task))
}

// GetUserTasks lists tasks for the authenticated user.
// @Summary List user tasks
// @Tags Tasks
// @Produce json
// @Param board_id query string false "Optional board ID filter"
// @Success 200 {array} taskResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /tasks [get]
func (handler *TaskHandler) GetUserTasks(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	boardID := request.URL.Query().Get("board_id")
	var tasks []*domain.Task
	var err error
	if boardID == "" {
		tasks, err = handler.taskUseCase.GetUserTasks(request.Context(), userID)
	} else {
		tasks, err = handler.taskUseCase.GetBoardTasks(request.Context(), userID, boardID)
	}
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, taskListResponseFromDomain(tasks))
}

// GetTask retrieves one task owned by the authenticated user.
// @Summary Get task
// @Tags Tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} taskResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /tasks/{id} [get]
func (handler *TaskHandler) GetTask(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	task, err := handler.taskUseCase.GetTask(request.Context(), userID, chi.URLParam(request, "id"))
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, taskResponseFromDomain(task))
}

// UpdateTask updates task fields in one request.
// @Summary Update task
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Param request body updateTaskRequest true "Task update payload"
// @Success 204 "No Content"
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /tasks/{id} [patch]
func (handler *TaskHandler) UpdateTask(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var updateRequest updateTaskRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		handler.logger.Warn("update task request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	dueDate, err := parseTaskDueDate(updateRequest.DueDate)
	if err != nil {
		handler.logger.Warn("update task request contains invalid due date", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid due date"})
		return
	}

	err = handler.taskUseCase.UpdateTask(
		request.Context(),
		userID,
		chi.URLParam(request, "id"),
		updateRequest.Title,
		updateRequest.Description,
		updateRequest.ColumnID,
		domain.TaskPriority(updateRequest.Priority),
		dueDate,
	)
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *TaskHandler) MoveTaskToColumn(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var moveRequest moveTaskToColumnRequest
	if err := json.NewDecoder(request.Body).Decode(&moveRequest); err != nil {
		handler.logger.Warn("move task to column request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	err := handler.taskUseCase.MoveTaskToColumn(request.Context(), userID, chi.URLParam(request, "id"), moveRequest.ColumnID)
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

// UpdateTaskDetails updates task title and description.
// @Summary Update task details
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Param request body updateTaskDetailsRequest true "Task details update payload"
// @Success 204 "No Content"
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /tasks/{id}/details [patch]
func (handler *TaskHandler) UpdateTaskDetails(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var updateRequest updateTaskDetailsRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		handler.logger.Warn("update task details request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	err := handler.taskUseCase.UpdateTaskDetails(request.Context(), userID, chi.URLParam(request, "id"), updateRequest.Title, updateRequest.Description)
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

// UpdateTaskStatus updates task status.
// @Summary Update task status
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Param request body updateTaskStatusRequest true "Task status update payload"
// @Success 204 "No Content"
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /tasks/{id}/status [patch]
func (handler *TaskHandler) UpdateTaskStatus(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var updateRequest updateTaskStatusRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		handler.logger.Warn("update task status request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	err := handler.taskUseCase.UpdateTaskStatus(request.Context(), userID, chi.URLParam(request, "id"), domain.TaskStatus(updateRequest.Status))
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

// UpdateTaskPriority updates task priority.
// @Summary Update task priority
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Param request body updateTaskPriorityRequest true "Task priority update payload"
// @Success 204 "No Content"
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /tasks/{id}/priority [patch]
func (handler *TaskHandler) UpdateTaskPriority(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var updateRequest updateTaskPriorityRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		handler.logger.Warn("update task priority request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	err := handler.taskUseCase.UpdateTaskPriority(request.Context(), userID, chi.URLParam(request, "id"), domain.TaskPriority(updateRequest.Priority))
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

// DeleteTask deletes a task owned by the authenticated user.
// @Summary Delete task
// @Tags Tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 204 "No Content"
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /tasks/{id} [delete]
func (handler *TaskHandler) DeleteTask(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	err := handler.taskUseCase.DeleteTask(request.Context(), userID, chi.URLParam(request, "id"))
	if err != nil {
		handler.handleTaskError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *TaskHandler) userIDFromRequest(response http.ResponseWriter, request *http.Request) (string, bool) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		handler.logger.Warn("authenticated task request is missing user context")
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return "", false
	}

	return userID, true
}

func (handler *TaskHandler) handleTaskError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrTaskNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "task not found"})
	case errors.Is(err, ports.ErrBoardNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "board not found"})
	case errors.Is(err, ports.ErrColumnNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "column not found"})
	case isTaskDomainValidationError(err):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid task data"})
	default:
		handler.logger.Error("task request failed due to internal processing error", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func isTaskDomainValidationError(err error) bool {
	return errors.Is(err, domain.ErrEmptyTaskID) ||
		errors.Is(err, domain.ErrEmptyTaskUserID) ||
		errors.Is(err, domain.ErrInvalidTaskTitle) ||
		errors.Is(err, domain.ErrInvalidTaskStatus) ||
		errors.Is(err, domain.ErrInvalidTaskPriority) ||
		errors.Is(err, domain.ErrInvalidTaskCreatedAt) ||
		errors.Is(err, domain.ErrInvalidTaskUpdatedAt)
}

func parseTaskDueDate(rawDueDate string) (time.Time, error) {
	if rawDueDate == "" {
		return time.Time{}, nil
	}

	dueDate, err := time.Parse(time.RFC3339Nano, rawDueDate)
	if err == nil {
		return dueDate, nil
	}

	return time.Parse("2006-01-02", rawDueDate)
}

func taskResponseFromDomain(task *domain.Task) taskResponse {
	return taskResponse{
		ID:          task.ID(),
		BoardID:     task.BoardID(),
		ColumnID:    task.ColumnID(),
		Title:       task.Title(),
		Description: task.Description(),
		Status:      string(task.Status()),
		Priority:    string(task.Priority()),
		DueDate:     formatTaskDueDate(task.DueDate()),
		CreatedAt:   task.CreatedAt().Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt().Format(time.RFC3339),
	}
}

func taskListResponseFromDomain(tasks []*domain.Task) []taskResponse {
	responses := make([]taskResponse, 0, len(tasks))
	for _, task := range tasks {
		responses = append(responses, taskResponseFromDomain(task))
	}

	return responses
}

func formatTaskDueDate(dueDate time.Time) string {
	if dueDate.IsZero() {
		return ""
	}

	return dueDate.Format(time.RFC3339)
}

type createTaskRequest struct {
	BoardID     *string `json:"boardId"`
	ColumnID    *string `json:"columnId"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	DueDate     string  `json:"dueDate"`
}

type updateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ColumnID    *string `json:"columnId"`
	Priority    string  `json:"priority"`
	DueDate     string  `json:"dueDate"`
}

type updateTaskDetailsRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updateTaskStatusRequest struct {
	Status string `json:"status"`
}

type updateTaskPriorityRequest struct {
	Priority string `json:"priority"`
}

type moveTaskToColumnRequest struct {
	ColumnID *string `json:"columnId"`
}

type taskResponse struct {
	ID          string  `json:"id"`
	BoardID     *string `json:"boardId"`
	ColumnID    *string `json:"columnId"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	DueDate     string  `json:"dueDate"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}
