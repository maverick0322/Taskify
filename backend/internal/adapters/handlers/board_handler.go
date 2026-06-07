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

// BoardHandler adapts HTTP requests to the Kanban board application port.
type BoardHandler struct {
	boardUseCase ports.BoardUseCase
	logger       ports.Logger
}

func NewBoardHandler(boardUseCase ports.BoardUseCase, logger ports.Logger) *BoardHandler {
	return &BoardHandler{
		boardUseCase: boardUseCase,
		logger:       logger,
	}
}

func (handler *BoardHandler) RegisterRoutes(router chi.Router) {
	router.Post("/boards", handler.CreateBoard)
	router.Get("/boards", handler.GetUserBoards)
	router.Get("/boards/{id}", handler.GetBoard)
	router.Patch("/boards/{id}/name", handler.UpdateBoardName)
	router.Delete("/boards/{id}", handler.DeleteBoard)
	router.Post("/boards/{id}/columns", handler.CreateColumn)
	router.Get("/boards/{id}/columns", handler.GetBoardColumns)
	router.Patch("/columns/{id}/name", handler.UpdateColumnName)
	router.Patch("/columns/{id}/position", handler.MoveColumn)
	router.Delete("/columns/{id}", handler.DeleteColumn)
}

func (handler *BoardHandler) CreateBoard(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var createRequest boardNameRequest
	if err := json.NewDecoder(request.Body).Decode(&createRequest); err != nil {
		handler.logger.Warn("create board request contains invalid json")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	board, err := handler.boardUseCase.CreateBoard(request.Context(), userID, createRequest.Name)
	if err != nil {
		handler.handleBoardError(response, err)
		return
	}

	writeJSON(response, http.StatusCreated, boardResponseFromDomain(board))
}

func (handler *BoardHandler) GetUserBoards(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	boards, err := handler.boardUseCase.GetUserBoards(request.Context(), userID)
	if err != nil {
		handler.handleBoardError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, boardListResponseFromDomain(boards))
}

func (handler *BoardHandler) GetBoard(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	board, err := handler.boardUseCase.GetBoard(request.Context(), userID, chi.URLParam(request, "id"))
	if err != nil {
		handler.handleBoardError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, boardResponseFromDomain(board))
}

func (handler *BoardHandler) UpdateBoardName(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var updateRequest boardNameRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		handler.logger.Warn("update board name request contains invalid json")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	err := handler.boardUseCase.UpdateBoardName(request.Context(), userID, chi.URLParam(request, "id"), updateRequest.Name)
	if err != nil {
		handler.handleBoardError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *BoardHandler) DeleteBoard(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	err := handler.boardUseCase.DeleteBoard(request.Context(), userID, chi.URLParam(request, "id"))
	if err != nil {
		handler.handleBoardError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *BoardHandler) CreateColumn(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var createRequest createColumnRequest
	if err := json.NewDecoder(request.Body).Decode(&createRequest); err != nil {
		handler.logger.Warn("create column request contains invalid json")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	column, err := handler.boardUseCase.CreateColumn(request.Context(), userID, chi.URLParam(request, "id"), createRequest.Name, createRequest.Position)
	if err != nil {
		handler.handleColumnError(response, err)
		return
	}

	writeJSON(response, http.StatusCreated, columnResponseFromDomain(column))
}

func (handler *BoardHandler) GetBoardColumns(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	columns, err := handler.boardUseCase.GetBoardColumns(request.Context(), userID, chi.URLParam(request, "id"))
	if err != nil {
		handler.handleColumnError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, columnListResponseFromDomain(columns))
}

func (handler *BoardHandler) UpdateColumnName(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var updateRequest boardNameRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		handler.logger.Warn("update column name request contains invalid json")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	err := handler.boardUseCase.UpdateColumnName(request.Context(), userID, chi.URLParam(request, "id"), updateRequest.Name)
	if err != nil {
		handler.handleColumnError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *BoardHandler) MoveColumn(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var moveRequest moveColumnRequest
	if err := json.NewDecoder(request.Body).Decode(&moveRequest); err != nil {
		handler.logger.Warn("move column request contains invalid json")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	err := handler.boardUseCase.MoveColumn(request.Context(), userID, chi.URLParam(request, "id"), moveRequest.Position)
	if err != nil {
		handler.handleColumnError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *BoardHandler) DeleteColumn(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	err := handler.boardUseCase.DeleteColumn(request.Context(), userID, chi.URLParam(request, "id"))
	if err != nil {
		handler.handleColumnError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *BoardHandler) userIDFromRequest(response http.ResponseWriter, request *http.Request) (string, bool) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		handler.logger.Warn("authenticated board request is missing user context")
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return "", false
	}

	return userID, true
}

func (handler *BoardHandler) handleBoardError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrBoardNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "board not found"})
	case isBoardDomainValidationError(err):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid board data"})
	default:
		handler.logger.Error("board request failed due to internal processing error", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func (handler *BoardHandler) handleColumnError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrColumnNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "column not found"})
	case errors.Is(err, ports.ErrBoardNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "board not found"})
	case isColumnDomainValidationError(err):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid column data"})
	default:
		handler.logger.Error("column request failed due to internal processing error", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func isBoardDomainValidationError(err error) bool {
	return errors.Is(err, domain.ErrInvalidBoardID) ||
		errors.Is(err, domain.ErrInvalidBoardUserID) ||
		errors.Is(err, domain.ErrInvalidBoardName) ||
		errors.Is(err, domain.ErrInvalidBoardCreatedAt) ||
		errors.Is(err, domain.ErrInvalidBoardUpdatedAt)
}

func isColumnDomainValidationError(err error) bool {
	return errors.Is(err, domain.ErrInvalidColumnID) ||
		errors.Is(err, domain.ErrInvalidColumnBoardID) ||
		errors.Is(err, domain.ErrInvalidColumnName) ||
		errors.Is(err, domain.ErrInvalidColumnPosition) ||
		errors.Is(err, domain.ErrInvalidColumnCreatedAt) ||
		errors.Is(err, domain.ErrInvalidColumnUpdatedAt)
}

func boardResponseFromDomain(board *domain.Board) boardResponse {
	return boardResponse{
		ID:        board.ID(),
		Name:      board.Name(),
		CreatedAt: board.CreatedAt().Format(time.RFC3339),
		UpdatedAt: board.UpdatedAt().Format(time.RFC3339),
	}
}

func boardListResponseFromDomain(boards []*domain.Board) []boardResponse {
	responses := make([]boardResponse, 0, len(boards))
	for _, board := range boards {
		responses = append(responses, boardResponseFromDomain(board))
	}

	return responses
}

func columnResponseFromDomain(column *domain.Column) columnResponse {
	return columnResponse{
		ID:        column.ID(),
		BoardID:   column.BoardID(),
		Name:      column.Name(),
		Position:  column.Position(),
		CreatedAt: column.CreatedAt().Format(time.RFC3339),
		UpdatedAt: column.UpdatedAt().Format(time.RFC3339),
	}
}

func columnListResponseFromDomain(columns []*domain.Column) []columnResponse {
	responses := make([]columnResponse, 0, len(columns))
	for _, column := range columns {
		responses = append(responses, columnResponseFromDomain(column))
	}

	return responses
}

type boardNameRequest struct {
	Name string `json:"name"`
}

type createColumnRequest struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type moveColumnRequest struct {
	Position int `json:"position"`
}

type boardResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type columnResponse struct {
	ID        string `json:"id"`
	BoardID   string `json:"boardId"`
	Name      string `json:"name"`
	Position  int    `json:"position"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}
