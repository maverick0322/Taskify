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

type CreditCardHandler struct {
	creditCardUseCase ports.CreditCardUseCase
	logger            ports.Logger
}

func NewCreditCardHandler(creditCardUseCase ports.CreditCardUseCase, logger ports.Logger) *CreditCardHandler {
	return &CreditCardHandler{
		creditCardUseCase: creditCardUseCase,
		logger:            logger,
	}
}

func (handler *CreditCardHandler) RegisterRoutes(router chi.Router) {
	router.Post("/credit-cards", handler.CreateCreditCard)
	router.Get("/credit-cards", handler.GetCreditCards)
	router.Get("/financial/payables", handler.GetFinancialPayables)
	router.Post("/credit-cards/{id}/pay", handler.PayCreditCardDebt)
	router.Patch("/credit-cards/{id}", handler.UpdateCreditCard)
	router.Delete("/credit-cards/{id}", handler.DeleteCreditCard)
}

func (handler *CreditCardHandler) GetFinancialPayables(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	payables, err := handler.creditCardUseCase.GetPayables(request.Context(), userID, request.URL.Query().Get("timezone"))
	if err != nil {
		handler.handleCreditCardError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, payables)
}

func (handler *CreditCardHandler) CreateCreditCard(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var createRequest creditCardRequest
	if err := json.NewDecoder(request.Body).Decode(&createRequest); err != nil {
		handler.logger.Warn("create credit card request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	creditCard, err := handler.creditCardUseCase.CreateCreditCard(
		request.Context(),
		userID,
		createRequest.Name,
		createRequest.Bank,
		createRequest.Last4,
		createRequest.CutoffDay,
		createRequest.PaymentDay,
		createRequest.LimitCents,
		createRequest.Color,
		createRequest.Network,
	)
	if err != nil {
		handler.handleCreditCardError(response, err)
		return
	}

	writeJSON(response, http.StatusCreated, creditCardResponseFromDomain(creditCard, 0))
}

func (handler *CreditCardHandler) GetCreditCards(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	creditCards, err := handler.creditCardUseCase.GetCardsWithSummary(request.Context(), userID)
	if err != nil {
		handler.handleCreditCardError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, creditCardSummaryListResponseFromDomain(creditCards))
}

func (handler *CreditCardHandler) UpdateCreditCard(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var updateRequest creditCardRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		handler.logger.Warn("update credit card request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	err := handler.creditCardUseCase.UpdateCreditCard(
		request.Context(),
		userID,
		chi.URLParam(request, "id"),
		updateRequest.Name,
		updateRequest.Bank,
		updateRequest.Last4,
		updateRequest.CutoffDay,
		updateRequest.PaymentDay,
		updateRequest.LimitCents,
		updateRequest.Color,
		updateRequest.Network,
	)
	if err != nil {
		handler.handleCreditCardError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *CreditCardHandler) PayCreditCardDebt(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	var payRequest payCreditCardDebtRequest
	if err := json.NewDecoder(request.Body).Decode(&payRequest); err != nil {
		handler.logger.Warn("pay credit card request contains invalid json", "userID", userID)
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if payRequest.AmountCents <= 0 || payRequest.SourceAccountID == "" {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid credit card payment data"})
		return
	}

	result, err := handler.creditCardUseCase.PayCreditCardDebt(
		request.Context(),
		userID,
		chi.URLParam(request, "id"),
		payRequest.SourceAccountID,
		payRequest.AmountCents,
		payRequest.Timezone,
	)
	if err != nil {
		handler.logger.Warn(
			"credit card payment failed",
			"userID", userID,
			"creditCardID", chi.URLParam(request, "id"),
			"sourceAccountID", payRequest.SourceAccountID,
			"amountCents", payRequest.AmountCents,
			"error", err,
		)
		handler.handleCreditCardError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, result)
}

func (handler *CreditCardHandler) DeleteCreditCard(response http.ResponseWriter, request *http.Request) {
	userID, ok := handler.userIDFromRequest(response, request)
	if !ok {
		return
	}

	if err := handler.creditCardUseCase.DeleteCreditCard(request.Context(), userID, chi.URLParam(request, "id")); err != nil {
		handler.handleCreditCardError(response, err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler *CreditCardHandler) userIDFromRequest(response http.ResponseWriter, request *http.Request) (string, bool) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		handler.logger.Warn("authenticated credit card request is missing user context")
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return "", false
	}

	return userID, true
}

func (handler *CreditCardHandler) handleCreditCardError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrCreditCardNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "credit card not found"})
	case errors.Is(err, ports.ErrFinancialAccountNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "financial account not found"})
	case errors.Is(err, domain.ErrInsufficientFinancialAccountFunds):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "insufficient funds"})
	case errors.Is(err, domain.ErrInvalidTransactionAmount):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid transaction amount"})
	case errors.Is(err, ports.ErrCreditCardNoPayableDebt):
		writeJSON(response, http.StatusConflict, errorResponse{Error: "credit card has no payable debt"})
	case errors.Is(err, ports.ErrCreditCardPaymentExceedsDebt):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "payment exceeds payable debt"})
	case errors.Is(err, ports.ErrCreditCardStatementUnavailable):
		writeJSON(response, http.StatusConflict, errorResponse{Error: "credit card statement is unavailable"})
	case errors.Is(err, ports.ErrCreditCardPaymentConflict):
		writeJSON(response, http.StatusConflict, errorResponse{Error: "credit card payment changed concurrently; try again"})
	case isCreditCardDomainValidationError(err):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid credit card data"})
	default:
		handler.logger.Error("credit card request failed due to internal processing error", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func isCreditCardDomainValidationError(err error) bool {
	return errors.Is(err, domain.ErrInvalidCreditCardID) ||
		errors.Is(err, domain.ErrInvalidCreditCardUserID) ||
		errors.Is(err, domain.ErrInvalidCreditCardName) ||
		errors.Is(err, domain.ErrInvalidCreditCardBank) ||
		errors.Is(err, domain.ErrInvalidCreditCardLast4) ||
		errors.Is(err, domain.ErrInvalidCreditCardCutoffDay) ||
		errors.Is(err, domain.ErrInvalidCreditCardPaymentDay) ||
		errors.Is(err, domain.ErrInvalidCreditCardLimit) ||
		errors.Is(err, domain.ErrInvalidCreditCardColor) ||
		errors.Is(err, domain.ErrInvalidCreditCardNetwork) ||
		errors.Is(err, domain.ErrInvalidCreditCardCreatedAt) ||
		errors.Is(err, domain.ErrInvalidCreditCardUpdatedAt)
}

func creditCardResponseFromDomain(creditCard *domain.CreditCard, currentDebtCents int64) creditCardResponse {
	return creditCardResponse{
		ID:               creditCard.ID(),
		Name:             creditCard.Name(),
		Bank:             creditCard.Bank(),
		Last4:            creditCard.Last4(),
		CutoffDay:        creditCard.CutoffDay(),
		PaymentDay:       creditCard.PaymentDay(),
		LimitCents:       creditCard.LimitCents(),
		Color:            creditCard.Color(),
		Network:          creditCard.Network(),
		CurrentDebtCents: currentDebtCents,
		CreatedAt:        creditCard.CreatedAt().Format(time.RFC3339),
		UpdatedAt:        creditCard.UpdatedAt().Format(time.RFC3339),
	}
}

func creditCardSummaryListResponseFromDomain(creditCards []ports.CreditCardWithSummary) []creditCardResponse {
	responses := make([]creditCardResponse, 0, len(creditCards))
	for _, creditCard := range creditCards {
		if creditCard.CreditCard == nil {
			continue
		}
		item := creditCardResponseFromDomain(creditCard.CreditCard, creditCard.CurrentDebtCents)
		item.TotalBalanceCents = creditCard.TotalBalanceCents
		item.PaymentStatus = creditCard.Status
		if creditCard.PaymentDueDate != nil {
			item.PaymentDueDate = creditCard.PaymentDueDate.Format("2006-01-02")
		}
		responses = append(responses, item)
	}

	return responses
}

type creditCardRequest struct {
	Name       string `json:"name"`
	Bank       string `json:"bank"`
	Last4      string `json:"last4"`
	CutoffDay  int    `json:"cutoffDay"`
	PaymentDay int    `json:"paymentDay"`
	LimitCents int64  `json:"limitCents"`
	Color      string `json:"color"`
	Network    string `json:"network"`
}

type payCreditCardDebtRequest struct {
	SourceAccountID string `json:"sourceAccountId"`
	AmountCents     int64  `json:"amountCents"`
	Timezone        string `json:"timezone"`
}

type creditCardResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Bank              string `json:"bank"`
	Last4             string `json:"last4"`
	CutoffDay         int    `json:"cutoffDay"`
	PaymentDay        int    `json:"paymentDay"`
	LimitCents        int64  `json:"limitCents"`
	Color             string `json:"color"`
	Network           string `json:"network"`
	CurrentDebtCents  int64  `json:"currentDebtCents"`
	TotalBalanceCents int64  `json:"totalBalanceCents"`
	PaymentDueDate    string `json:"paymentDueDate,omitempty"`
	PaymentStatus     string `json:"paymentStatus,omitempty"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}
