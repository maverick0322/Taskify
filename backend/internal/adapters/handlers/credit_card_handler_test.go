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

type mockCreditCardUseCase struct {
	creditCardToReturn *domain.CreditCard
	summariesToReturn  []ports.CreditCardWithSummary
	errToReturn        error
	requestedUserID    string
	requestedID        string
	createdLimitCents  int64
	updatedLimitCents  int64
}

func (useCase *mockCreditCardUseCase) CreateCreditCard(ctx context.Context, userID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) (*domain.CreditCard, error) {
	useCase.requestedUserID = userID
	useCase.createdLimitCents = limitCents
	return useCase.creditCardToReturn, useCase.errToReturn
}

func (useCase *mockCreditCardUseCase) GetCardsWithSummary(ctx context.Context, userID string) ([]ports.CreditCardWithSummary, error) {
	useCase.requestedUserID = userID
	return useCase.summariesToReturn, useCase.errToReturn
}

func (useCase *mockCreditCardUseCase) UpdateCreditCard(ctx context.Context, userID, creditCardID, name, bank, last4 string, cutoffDay, paymentDay int, limitCents int64, color string) error {
	useCase.requestedUserID = userID
	useCase.requestedID = creditCardID
	useCase.updatedLimitCents = limitCents
	return useCase.errToReturn
}

func (useCase *mockCreditCardUseCase) DeleteCreditCard(ctx context.Context, userID, creditCardID string) error {
	useCase.requestedUserID = userID
	useCase.requestedID = creditCardID
	return useCase.errToReturn
}

func TestCreditCardHandler_CreateCreditCardValidRequest_ReturnsCreated(t *testing.T) {
	useCase := &mockCreditCardUseCase{creditCardToReturn: createHandlerCreditCard(t)}
	router := createCreditCardTestRouter(useCase)
	request := authenticatedCreditCardRequest(http.MethodPost, "/credit-cards", validCreateCreditCardJSON())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"id":"credit-card-123"`) {
		t.Errorf("expected response to contain credit card ID")
	}
	if useCase.createdLimitCents != 5000000 {
		t.Errorf("expected limit cents 5000000, got %d", useCase.createdLimitCents)
	}
}

func TestCreditCardHandler_GetCreditCards_ReturnsOKWithCurrentDebt(t *testing.T) {
	creditCard := createHandlerCreditCard(t)
	useCase := &mockCreditCardUseCase{
		summariesToReturn: []ports.CreditCardWithSummary{
			{CreditCard: creditCard, CurrentDebtCents: 15334},
		},
	}
	router := createCreditCardTestRouter(useCase)
	request := authenticatedCreditCardRequest(http.MethodGet, "/credit-cards", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"currentDebtCents":15334`) {
		t.Errorf("expected response to contain current debt cents")
	}
	if useCase.requestedUserID != "user-123" {
		t.Errorf("expected requested user ID user-123, got %s", useCase.requestedUserID)
	}
}

func TestCreditCardHandler_UpdateCreditCardValidRequest_ReturnsNoContent(t *testing.T) {
	useCase := &mockCreditCardUseCase{}
	router := createCreditCardTestRouter(useCase)
	request := authenticatedCreditCardRequest(http.MethodPatch, "/credit-cards/credit-card-123", validCreateCreditCardJSON())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if useCase.requestedID != "credit-card-123" {
		t.Errorf("expected requested credit card ID credit-card-123, got %s", useCase.requestedID)
	}
	if useCase.updatedLimitCents != 5000000 {
		t.Errorf("expected updated limit cents 5000000, got %d", useCase.updatedLimitCents)
	}
}

func TestCreditCardHandler_DeleteCreditCard_ReturnsNoContent(t *testing.T) {
	useCase := &mockCreditCardUseCase{}
	router := createCreditCardTestRouter(useCase)
	request := authenticatedCreditCardRequest(http.MethodDelete, "/credit-cards/credit-card-123", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if useCase.requestedID != "credit-card-123" {
		t.Errorf("expected requested credit card ID credit-card-123, got %s", useCase.requestedID)
	}
}

func TestCreditCardHandler_CreateCreditCardMalformedJSON_ReturnsBadRequest(t *testing.T) {
	router := createCreditCardTestRouter(&mockCreditCardUseCase{})
	request := authenticatedCreditCardRequest(http.MethodPost, "/credit-cards", "{")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestCreditCardHandler_CreateCreditCardDomainError_ReturnsBadRequest(t *testing.T) {
	router := createCreditCardTestRouter(&mockCreditCardUseCase{errToReturn: domain.ErrInvalidCreditCardLast4})
	request := authenticatedCreditCardRequest(http.MethodPost, "/credit-cards", validCreateCreditCardJSON())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestCreditCardHandler_UpdateCreditCardNotFound_ReturnsNotFound(t *testing.T) {
	router := createCreditCardTestRouter(&mockCreditCardUseCase{errToReturn: ports.ErrCreditCardNotFound})
	request := authenticatedCreditCardRequest(http.MethodPatch, "/credit-cards/credit-card-123", validCreateCreditCardJSON())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestCreditCardHandler_InternalError_ReturnsInternalServerError(t *testing.T) {
	router := createCreditCardTestRouter(&mockCreditCardUseCase{errToReturn: services.ErrInternalProcessing})
	request := authenticatedCreditCardRequest(http.MethodGet, "/credit-cards", "")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestCreditCardHandler_MissingUserContext_ReturnsUnauthorized(t *testing.T) {
	router := createCreditCardTestRouter(&mockCreditCardUseCase{})
	request := httptest.NewRequest(http.MethodGet, "/credit-cards", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func createCreditCardTestRouter(useCase *mockCreditCardUseCase) chi.Router {
	router := chi.NewRouter()
	handler := NewCreditCardHandler(useCase, &mockHandlerLogger{})
	handler.RegisterRoutes(router)
	return router
}

func authenticatedCreditCardRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	return request.WithContext(middleware.ContextWithUserID(request.Context(), "user-123"))
}

func createHandlerCreditCard(t *testing.T) *domain.CreditCard {
	t.Helper()

	creditCard, err := domain.RehydrateCreditCard(
		"credit-card-123",
		"user-123",
		"Clasica",
		"BBVA",
		"1234",
		15,
		5,
		5000000,
		"from-blue-500 to-sky-400",
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("expected credit card to be valid, got: %v", err)
	}

	return creditCard
}

func validCreateCreditCardJSON() string {
	return `{"name":"Clasica","bank":"BBVA","last4":"1234","cutoffDay":15,"paymentDay":5,"limitCents":5000000,"color":"from-blue-500 to-sky-400"}`
}
