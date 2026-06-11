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

const (
	transactionDateLayout = "2006-01-02"
)

type TransactionHandler struct {
	transactionUseCase ports.TransactionUseCase
	logger             ports.Logger
}

func NewTransactionHandler(transactionUseCase ports.TransactionUseCase, logger ports.Logger) *TransactionHandler {
	return &TransactionHandler{
		transactionUseCase: transactionUseCase,
		logger:             logger,
	}
}

func (handler *TransactionHandler) RegisterRoutes(router chi.Router) {
	router.Post("/transactions", handler.CreateTransaction)
	router.Get("/transactions", handler.GetTransactions)
	router.Get("/transactions/summary", handler.GetFinancialSummary)
	router.Patch("/transactions/{id}", handler.UpdateTransaction)
	router.Delete("/transactions/{id}", handler.DeleteTransaction)
}

func (handler *TransactionHandler) CreateTransaction(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var createRequest transactionRequest
	if err := json.NewDecoder(request.Body).Decode(&createRequest); err != nil {
		handler.logger.Warn("create transaction request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	transactionDate, ok := handler.transactionDateFromRequest(response, createRequest.Date, "create transaction request contains invalid date", userID)
	if !ok {
		return
	}

	transaction, err := handler.transactionUseCase.CreateTransaction(
		request.Context(),
		userID,
		domain.TransactionType(createRequest.Type),
		createRequest.Concept,
		createRequest.Category,
		createRequest.AmountCents,
		transactionDate,
		domain.TransactionStatus(createRequest.Status),
		createRequest.MSI,
		createRequest.CreditCardID,
	)
	if err != nil {
		handler.handleTransactionError(response, err)
		return
	}

	writeJSON(response, http.StatusCreated, transactionResponseFromDomain(transaction))
}

func (handler *TransactionHandler) GetTransactions(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	filter, ok := handler.transactionDateFilterFromRequest(response, request, userID)
	if !ok {
		return
	}

	transactions, err := handler.transactionUseCase.GetUserTransactions(request.Context(), userID, filter)
	if err != nil {
		handler.handleTransactionError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, transactionListResponseFromDomain(transactions))
}

func (handler *TransactionHandler) UpdateTransaction(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var updateRequest transactionRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		handler.logger.Warn("update transaction request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	transactionDate, ok := handler.transactionDateFromRequest(response, updateRequest.Date, "update transaction request contains invalid date", userID)
	if !ok {
		return
	}

	err := handler.transactionUseCase.UpdateTransaction(
		request.Context(),
		userID,
		chi.URLParam(request, "id"),
		domain.TransactionType(updateRequest.Type),
		updateRequest.Concept,
		updateRequest.Category,
		updateRequest.AmountCents,
		transactionDate,
		domain.TransactionStatus(updateRequest.Status),
		updateRequest.MSI,
		updateRequest.CreditCardID,
	)
	if err != nil {
		handler.handleTransactionError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *TransactionHandler) DeleteTransaction(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	if err := handler.transactionUseCase.DeleteTransaction(request.Context(), userID, chi.URLParam(request, "id")); err != nil {
		handler.handleTransactionError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *TransactionHandler) GetFinancialSummary(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	startDate, endDate, ok := handler.summaryDateRangeFromRequest(response, request, userID)
	if !ok {
		return
	}

	summary, err := handler.transactionUseCase.GetFinancialSummary(request.Context(), userID, startDate, endDate)
	if err != nil {
		handler.handleTransactionError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, financialSummaryResponseFromDomain(summary))
}

func (handler *TransactionHandler) userIDFromRequest(response http.ResponseWriter, request *http.Request) (string, bool) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		handler.logger.Warn("authenticated transaction request is missing user context")
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return "", false
	}

	return userID, true
}

func (handler *TransactionHandler) transactionDateFromRequest(response http.ResponseWriter, rawDate, message, userID string) (time.Time, bool) {
	transactionDate, err := parseTransactionDate(rawDate)
	if err != nil {
		handler.logger.Warn(message, "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid date"})
		return time.Time{}, false
	}

	return transactionDate, true
}

func (handler *TransactionHandler) transactionDateFilterFromRequest(response http.ResponseWriter, request *http.Request, userID string) (ports.TransactionDateFilter, bool) {
	query := request.URL.Query()
	startDate, hasStartDate, ok := handler.optionalDateQueryParam(response, query.Get("start_date"), "transaction list request contains invalid start date", userID)
	if !ok {
		return ports.TransactionDateFilter{}, false
	}
	endDate, hasEndDate, ok := handler.optionalDateQueryParam(response, query.Get("end_date"), "transaction list request contains invalid end date", userID)
	if !ok {
		return ports.TransactionDateFilter{}, false
	}

	filter := ports.TransactionDateFilter{}
	if hasStartDate {
		filter.From = &startDate
	}
	if hasEndDate {
		exclusiveEndDate := endDate.AddDate(0, 0, 1)
		filter.To = &exclusiveEndDate
	}

	return filter, true
}

func (handler *TransactionHandler) optionalDateQueryParam(response http.ResponseWriter, rawDate, message, userID string) (time.Time, bool, bool) {
	if rawDate == "" {
		return time.Time{}, false, true
	}

	parsedDate, err := parseTransactionDate(rawDate)
	if err != nil {
		handler.logger.Warn(message, "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid date"})
		return time.Time{}, false, false
	}

	return parsedDate, true, true
}

func (handler *TransactionHandler) summaryDateRangeFromRequest(response http.ResponseWriter, request *http.Request, userID string) (time.Time, time.Time, bool) {
	query := request.URL.Query()
	rawStartDate := query.Get("start_date")
	rawEndDate := query.Get("end_date")
	if rawStartDate == "" || rawEndDate == "" {
		handler.logger.Warn("financial summary request is missing date range", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "start_date and end_date are required"})
		return time.Time{}, time.Time{}, false
	}

	startDate, err := parseTransactionDate(rawStartDate)
	if err != nil {
		handler.logger.Warn("financial summary request contains invalid start date", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid start_date"})
		return time.Time{}, time.Time{}, false
	}

	endDate, err := parseTransactionDate(rawEndDate)
	if err != nil {
		handler.logger.Warn("financial summary request contains invalid end date", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid end_date"})
		return time.Time{}, time.Time{}, false
	}

	return startDate, endDate.AddDate(0, 0, 1), true
}

func (handler *TransactionHandler) handleTransactionError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrTransactionNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "transaction not found"})
	case isTransactionDomainValidationError(err):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid transaction data"})
	default:
		handler.logger.Error("transaction request failed due to internal processing error", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func isTransactionDomainValidationError(err error) bool {
	return errors.Is(err, domain.ErrEmptyTransactionID) ||
		errors.Is(err, domain.ErrEmptyTransactionUserID) ||
		errors.Is(err, domain.ErrInvalidTransactionType) ||
		errors.Is(err, domain.ErrEmptyTransactionConcept) ||
		errors.Is(err, domain.ErrEmptyTransactionCategory) ||
		errors.Is(err, domain.ErrInvalidTransactionAmount) ||
		errors.Is(err, domain.ErrInvalidTransactionDate) ||
		errors.Is(err, domain.ErrInvalidTransactionStatus) ||
		errors.Is(err, domain.ErrInvalidTransactionMSI) ||
		errors.Is(err, domain.ErrInvalidTransactionCreatedAt) ||
		errors.Is(err, domain.ErrInvalidTransactionUpdatedAt)
}

func parseTransactionDate(rawDate string) (time.Time, error) {
	return time.Parse(transactionDateLayout, rawDate)
}

func transactionResponseFromDomain(transaction *domain.Transaction) transactionResponse {
	return transactionResponse{
		ID:           transaction.ID(),
		Type:         string(transaction.Type()),
		Concept:      transaction.Concept(),
		Category:     transaction.Category(),
		AmountCents:  transaction.AmountCents(),
		Date:         transaction.Date().Format(transactionDateLayout),
		Status:       string(transaction.Status()),
		MSI:          transaction.MSI(),
		CreditCardID: transaction.CreditCardID(),
		CreatedAt:    transaction.CreatedAt().Format(time.RFC3339),
		UpdatedAt:    transaction.UpdatedAt().Format(time.RFC3339),
	}
}

func transactionListResponseFromDomain(transactions []*domain.Transaction) []transactionResponse {
	responses := make([]transactionResponse, 0, len(transactions))
	for _, transaction := range transactions {
		responses = append(responses, transactionResponseFromDomain(transaction))
	}

	return responses
}

func financialSummaryResponseFromDomain(summary ports.FinancialSummary) financialSummaryResponse {
	return financialSummaryResponse{
		TotalIncomeCents:  summary.TotalIncomeCents,
		TotalExpenseCents: summary.TotalExpenseCents,
		ProfitMarginCents: summary.ProfitMarginCents,
	}
}

type transactionRequest struct {
	Type         string  `json:"type"`
	Concept      string  `json:"concept"`
	Category     string  `json:"category"`
	AmountCents  int64   `json:"amountCents"`
	Date         string  `json:"date"`
	Status       string  `json:"status"`
	MSI          *int    `json:"msi"`
	CreditCardID *string `json:"creditCardId"`
}

type transactionResponse struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Concept      string  `json:"concept"`
	Category     string  `json:"category"`
	AmountCents  int64   `json:"amountCents"`
	Date         string  `json:"date"`
	Status       string  `json:"status"`
	MSI          *int    `json:"msi"`
	CreditCardID *string `json:"creditCardId"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type financialSummaryResponse struct {
	TotalIncomeCents  int64 `json:"totalIncomeCents"`
	TotalExpenseCents int64 `json:"totalExpenseCents"`
	ProfitMarginCents int64 `json:"profitMarginCents"`
}
