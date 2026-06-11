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

type mockTransactionUseCase struct {
	transactionToReturn  *domain.Transaction
	transactionsToReturn []*domain.Transaction
	summaryToReturn      ports.FinancialSummary
	errToReturn          error
	createdAmountCents   int64
	updatedAmountCents   int64
	requestedUserID      string
	requestedID          string
	receivedFilter       ports.TransactionDateFilter
	receivedStartDate    time.Time
	receivedEndDate      time.Time
}

func (useCase *mockTransactionUseCase) CreateTransaction(ctx context.Context, userID string, transactionType domain.TransactionType, concept, category string, amountCents int64, date time.Time, status domain.TransactionStatus, msi *int, creditCardID *string) (*domain.Transaction, error) {
	useCase.requestedUserID = userID
	useCase.createdAmountCents = amountCents
	return useCase.transactionToReturn, useCase.errToReturn
}

func (useCase *mockTransactionUseCase) GetTransaction(ctx context.Context, userID, transactionID string) (*domain.Transaction, error) {
	useCase.requestedUserID = userID
	useCase.requestedID = transactionID
	return useCase.transactionToReturn, useCase.errToReturn
}

func (useCase *mockTransactionUseCase) GetUserTransactions(ctx context.Context, userID string, filter ports.TransactionDateFilter) ([]*domain.Transaction, error) {
	useCase.requestedUserID = userID
	useCase.receivedFilter = filter
	return useCase.transactionsToReturn, useCase.errToReturn
}

func (useCase *mockTransactionUseCase) UpdateTransaction(ctx context.Context, userID, transactionID string, transactionType domain.TransactionType, concept, category string, amountCents int64, date time.Time, status domain.TransactionStatus, msi *int, creditCardID *string) error {
	useCase.requestedUserID = userID
	useCase.requestedID = transactionID
	useCase.updatedAmountCents = amountCents
	return useCase.errToReturn
}

func (useCase *mockTransactionUseCase) DeleteTransaction(ctx context.Context, userID, transactionID string) error {
	useCase.requestedUserID = userID
	useCase.requestedID = transactionID
	return useCase.errToReturn
}

func (useCase *mockTransactionUseCase) GetFinancialSummary(ctx context.Context, userID string, startDate, endDate time.Time) (ports.FinancialSummary, error) {
	useCase.requestedUserID = userID
	useCase.receivedStartDate = startDate
	useCase.receivedEndDate = endDate
	return useCase.summaryToReturn, useCase.errToReturn
}

func TestTransactionHandler_CreateTransactionValidRequest_ReturnsCreated(t *testing.T) {
	useCase := &mockTransactionUseCase{transactionToReturn: createHandlerTransaction(t)}
	router := createTransactionTestRouter(useCase)
	request := authenticatedTransactionRequest(http.MethodPost, "/transactions", validCreateTransactionJSON())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"id":"transaction-123"`) {
		t.Errorf("expected response to contain transaction ID")
	}
	if useCase.createdAmountCents != 12500 {
		t.Errorf("expected amount cents 12500, got %d", useCase.createdAmountCents)
	}
}

func TestTransactionHandler_CreateTransactionMalformedJSON_ReturnsBadRequest(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{})
	request := authenticatedTransactionRequest(http.MethodPost, "/transactions", "{")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTransactionHandler_CreateTransactionDecimalAmount_ReturnsBadRequest(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{})
	request := authenticatedTransactionRequest(http.MethodPost, "/transactions", `{"type":"EXPENSE","concept":"CFE - Luz","category":"Servicios","amountCents":125.50,"date":"2026-06-10","status":"PAID","msi":null}`)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTransactionHandler_CreateTransactionInvalidDate_ReturnsBadRequest(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{})
	request := authenticatedTransactionRequest(http.MethodPost, "/transactions", `{"type":"EXPENSE","concept":"CFE - Luz","category":"Servicios","amountCents":12500,"date":"10-06-2026","status":"PAID","msi":null}`)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTransactionHandler_CreateTransactionDomainError_ReturnsBadRequest(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{errToReturn: domain.ErrInvalidTransactionAmount})
	request := authenticatedTransactionRequest(http.MethodPost, "/transactions", validCreateTransactionJSON())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTransactionHandler_GetTransactions_ReturnsOK(t *testing.T) {
	useCase := &mockTransactionUseCase{transactionsToReturn: []*domain.Transaction{createHandlerTransaction(t)}}
	router := createTransactionTestRouter(useCase)
	request := authenticatedTransactionRequest(http.MethodGet, "/transactions", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"amountCents":12500`) {
		t.Errorf("expected response to contain amount cents")
	}
	if useCase.requestedUserID != "user-123" {
		t.Errorf("expected requested user ID user-123, got %s", useCase.requestedUserID)
	}
}

func TestTransactionHandler_GetTransactionsWithDateFilter_PassesExclusiveEndDate(t *testing.T) {
	useCase := &mockTransactionUseCase{}
	router := createTransactionTestRouter(useCase)
	request := authenticatedTransactionRequest(http.MethodGet, "/transactions?start_date=2026-06-01&end_date=2026-06-30", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if useCase.receivedFilter.From == nil || !useCase.receivedFilter.From.Equal(time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected start date filter to be set, got %v", useCase.receivedFilter.From)
	}
	if useCase.receivedFilter.To == nil || !useCase.receivedFilter.To.Equal(time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected exclusive end date filter 2026-07-01, got %v", useCase.receivedFilter.To)
	}
}

func TestTransactionHandler_GetTransactionsInvalidFilterDate_ReturnsBadRequest(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{})
	request := authenticatedTransactionRequest(http.MethodGet, "/transactions?start_date=01-06-2026", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTransactionHandler_UpdateTransactionValidRequest_ReturnsNoContent(t *testing.T) {
	useCase := &mockTransactionUseCase{}
	router := createTransactionTestRouter(useCase)
	request := authenticatedTransactionRequest(http.MethodPatch, "/transactions/transaction-123", validCreateTransactionJSON())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if useCase.requestedID != "transaction-123" {
		t.Errorf("expected requested transaction ID transaction-123, got %s", useCase.requestedID)
	}
	if useCase.updatedAmountCents != 12500 {
		t.Errorf("expected updated amount cents 12500, got %d", useCase.updatedAmountCents)
	}
}

func TestTransactionHandler_UpdateTransactionNotFound_ReturnsNotFound(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{errToReturn: ports.ErrTransactionNotFound})
	request := authenticatedTransactionRequest(http.MethodPatch, "/transactions/transaction-123", validCreateTransactionJSON())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestTransactionHandler_DeleteTransaction_ReturnsNoContent(t *testing.T) {
	useCase := &mockTransactionUseCase{}
	router := createTransactionTestRouter(useCase)
	request := authenticatedTransactionRequest(http.MethodDelete, "/transactions/transaction-123", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if useCase.requestedID != "transaction-123" {
		t.Errorf("expected requested transaction ID transaction-123, got %s", useCase.requestedID)
	}
}

func TestTransactionHandler_GetFinancialSummary_ReturnsOK(t *testing.T) {
	useCase := &mockTransactionUseCase{
		summaryToReturn: ports.FinancialSummary{
			TotalIncomeCents:  10500,
			TotalExpenseCents: 2500,
			ProfitMarginCents: 8000,
		},
	}
	router := createTransactionTestRouter(useCase)
	request := authenticatedTransactionRequest(http.MethodGet, "/transactions/summary?start_date=2026-06-01&end_date=2026-06-30", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"profitMarginCents":8000`) {
		t.Errorf("expected response to contain profit margin cents")
	}
	if !useCase.receivedStartDate.Equal(time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected start date 2026-06-01, got %v", useCase.receivedStartDate)
	}
	if !useCase.receivedEndDate.Equal(time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected exclusive end date 2026-07-01, got %v", useCase.receivedEndDate)
	}
}

func TestTransactionHandler_GetFinancialSummaryMissingDates_ReturnsBadRequest(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{})
	request := authenticatedTransactionRequest(http.MethodGet, "/transactions/summary?start_date=2026-06-01", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTransactionHandler_GetFinancialSummaryInvalidDate_ReturnsBadRequest(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{})
	request := authenticatedTransactionRequest(http.MethodGet, "/transactions/summary?start_date=2026-06-01&end_date=30-06-2026", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestTransactionHandler_InternalError_ReturnsInternalServerError(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{errToReturn: services.ErrInternalProcessing})
	request := authenticatedTransactionRequest(http.MethodGet, "/transactions", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestTransactionHandler_MissingUserContext_ReturnsUnauthorized(t *testing.T) {
	router := createTransactionTestRouter(&mockTransactionUseCase{})
	request := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func createTransactionTestRouter(useCase *mockTransactionUseCase) chi.Router {
	router := chi.NewRouter()
	handler := NewTransactionHandler(useCase, &mockHandlerLogger{})
	handler.RegisterRoutes(router)
	return router
}

func authenticatedTransactionRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	return request.WithContext(middleware.ContextWithUserID(request.Context(), "user-123"))
}

func createHandlerTransaction(t *testing.T) *domain.Transaction {
	t.Helper()

	transaction, err := domain.RehydrateTransaction(
		"transaction-123",
		"user-123",
		domain.TransactionTypeExpense,
		"CFE - Luz",
		"Servicios",
		12500,
		time.Date(2026, time.June, 10, 0, 0, 0, 0, time.UTC),
		domain.TransactionStatusPaid,
		nil,
		nil,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("expected transaction to be valid, got: %v", err)
	}

	return transaction
}

func validCreateTransactionJSON() string {
	return `{"type":"EXPENSE","concept":"CFE - Luz","category":"Servicios","amountCents":12500,"date":"2026-06-10","status":"PAID","msi":null}`
}
