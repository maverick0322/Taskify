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

type mockBoardUseCase struct {
	boardToReturn   *domain.Board
	boardsToReturn  []*domain.Board
	columnToReturn  *domain.Column
	columnsToReturn []*domain.Column
	errToReturn     error
	receivedUserID  string
}

func (useCase *mockBoardUseCase) CreateBoard(ctx context.Context, userID, name string) (*domain.Board, error) {
	useCase.receivedUserID = userID
	return useCase.boardToReturn, useCase.errToReturn
}

func (useCase *mockBoardUseCase) GetBoard(ctx context.Context, userID, boardID string) (*domain.Board, error) {
	useCase.receivedUserID = userID
	return useCase.boardToReturn, useCase.errToReturn
}

func (useCase *mockBoardUseCase) GetUserBoards(ctx context.Context, userID string) ([]*domain.Board, error) {
	useCase.receivedUserID = userID
	return useCase.boardsToReturn, useCase.errToReturn
}

func (useCase *mockBoardUseCase) UpdateBoardName(ctx context.Context, userID, boardID, name string) error {
	useCase.receivedUserID = userID
	return useCase.errToReturn
}

func (useCase *mockBoardUseCase) DeleteBoard(ctx context.Context, userID, boardID string) error {
	useCase.receivedUserID = userID
	return useCase.errToReturn
}

func (useCase *mockBoardUseCase) CreateColumn(ctx context.Context, userID, boardID, name string, position int) (*domain.Column, error) {
	useCase.receivedUserID = userID
	return useCase.columnToReturn, useCase.errToReturn
}

func (useCase *mockBoardUseCase) GetBoardColumns(ctx context.Context, userID, boardID string) ([]*domain.Column, error) {
	useCase.receivedUserID = userID
	return useCase.columnsToReturn, useCase.errToReturn
}

func (useCase *mockBoardUseCase) UpdateColumnName(ctx context.Context, userID, columnID, name string) error {
	useCase.receivedUserID = userID
	return useCase.errToReturn
}

func (useCase *mockBoardUseCase) MoveColumn(ctx context.Context, userID, columnID string, position int) error {
	useCase.receivedUserID = userID
	return useCase.errToReturn
}

func (useCase *mockBoardUseCase) DeleteColumn(ctx context.Context, userID, columnID string) error {
	useCase.receivedUserID = userID
	return useCase.errToReturn
}

func TestBoardHandler_CreateBoardValidRequest_ReturnsCreated(t *testing.T) {
	// Arrange
	useCase := &mockBoardUseCase{boardToReturn: createHandlerBoard(t)}
	router := createBoardTestRouter(useCase)
	request := authenticatedBoardRequest(http.MethodPost, "/boards", validBoardNameJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"id":"board-123"`) {
		t.Errorf("expected response to contain board ID")
	}
	if strings.Contains(response.Body.String(), "user-123") {
		t.Errorf("expected response not to expose user ID")
	}
	if useCase.receivedUserID != "user-123" {
		t.Errorf("expected user ID from context, got %s", useCase.receivedUserID)
	}
}

func TestBoardHandler_CreateBoardMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodPost, "/boards", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_CreateBoardDomainError_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: domain.ErrInvalidBoardName})
	request := authenticatedBoardRequest(http.MethodPost, "/boards", validBoardNameJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_CreateBoardInternalError_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: services.ErrInternalProcessing})
	request := authenticatedBoardRequest(http.MethodPost, "/boards", validBoardNameJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestBoardHandler_GetUserBoards_ReturnsOK(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{boardsToReturn: []*domain.Board{createHandlerBoard(t)}})
	request := authenticatedBoardRequest(http.MethodGet, "/boards", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"id":"board-123"`) {
		t.Errorf("expected response to contain board ID")
	}
}

func TestBoardHandler_GetBoard_ReturnsOK(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{boardToReturn: createHandlerBoard(t)})
	request := authenticatedBoardRequest(http.MethodGet, "/boards/board-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestBoardHandler_GetBoardNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: ports.ErrBoardNotFound})
	request := authenticatedBoardRequest(http.MethodGet, "/boards/board-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestBoardHandler_UpdateBoardNameValidRequest_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodPatch, "/boards/board-123/name", validBoardNameJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestBoardHandler_UpdateBoardNameMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodPatch, "/boards/board-123/name", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_UpdateBoardNameDomainError_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: domain.ErrInvalidBoardName})
	request := authenticatedBoardRequest(http.MethodPatch, "/boards/board-123/name", validBoardNameJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_DeleteBoard_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodDelete, "/boards/board-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestBoardHandler_CreateColumnValidRequest_ReturnsCreated(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{columnToReturn: createHandlerColumn(t)})
	request := authenticatedBoardRequest(http.MethodPost, "/boards/board-123/columns", validCreateColumnJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"boardId":"board-123"`) {
		t.Errorf("expected response to contain board ID")
	}
}

func TestBoardHandler_CreateColumnMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodPost, "/boards/board-123/columns", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_CreateColumnDomainError_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: domain.ErrInvalidColumnPosition})
	request := authenticatedBoardRequest(http.MethodPost, "/boards/board-123/columns", validCreateColumnJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_CreateColumnBoardNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: ports.ErrBoardNotFound})
	request := authenticatedBoardRequest(http.MethodPost, "/boards/board-123/columns", validCreateColumnJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestBoardHandler_GetBoardColumns_ReturnsOK(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{columnsToReturn: []*domain.Column{createHandlerColumn(t)}})
	request := authenticatedBoardRequest(http.MethodGet, "/boards/board-123/columns", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"id":"column-123"`) {
		t.Errorf("expected response to contain column ID")
	}
}

func TestBoardHandler_UpdateColumnNameValidRequest_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodPatch, "/columns/column-123/name", validBoardNameJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestBoardHandler_UpdateColumnNameMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodPatch, "/columns/column-123/name", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_UpdateColumnNameNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: ports.ErrColumnNotFound})
	request := authenticatedBoardRequest(http.MethodPatch, "/columns/column-123/name", validBoardNameJSON())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestBoardHandler_MoveColumnValidRequest_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodPatch, "/columns/column-123/position", `{"position":2}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestBoardHandler_MoveColumnMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodPatch, "/columns/column-123/position", "{")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_MoveColumnDomainError_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: domain.ErrInvalidColumnPosition})
	request := authenticatedBoardRequest(http.MethodPatch, "/columns/column-123/position", `{"position":-1}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestBoardHandler_MoveColumnInternalError_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: services.ErrInternalProcessing})
	request := authenticatedBoardRequest(http.MethodPatch, "/columns/column-123/position", `{"position":2}`)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestBoardHandler_DeleteColumn_ReturnsNoContent(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := authenticatedBoardRequest(http.MethodDelete, "/columns/column-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestBoardHandler_DeleteColumnNotFound_ReturnsNotFound(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{errToReturn: ports.ErrColumnNotFound})
	request := authenticatedBoardRequest(http.MethodDelete, "/columns/column-123", "")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestBoardHandler_MissingUserContext_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	router := createBoardTestRouter(&mockBoardUseCase{})
	request := httptest.NewRequest(http.MethodGet, "/boards", nil)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func createBoardTestRouter(useCase *mockBoardUseCase) chi.Router {
	router := chi.NewRouter()
	handler := NewBoardHandler(useCase, &mockHandlerLogger{})
	handler.RegisterRoutes(router)
	return router
}

func authenticatedBoardRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	return request.WithContext(middleware.ContextWithUserID(request.Context(), "user-123"))
}

func createHandlerBoard(t *testing.T) *domain.Board {
	t.Helper()

	board, err := domain.RehydrateBoard(
		"board-123",
		"user-123",
		"Product Roadmap",
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("expected board to be valid, got: %v", err)
	}

	return board
}

func createHandlerColumn(t *testing.T) *domain.Column {
	t.Helper()

	column, err := domain.RehydrateColumn(
		"column-123",
		"board-123",
		"Backlog",
		0,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("expected column to be valid, got: %v", err)
	}

	return column
}

func validBoardNameJSON() string {
	return `{"name":"Product Roadmap"}`
}

func validCreateColumnJSON() string {
	return `{"name":"Backlog","position":0}`
}
