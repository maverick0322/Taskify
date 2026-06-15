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

type FinancialAccountHandler struct {
	useCase ports.FinancialAccountUseCase
	logger  ports.Logger
}

func NewFinancialAccountHandler(useCase ports.FinancialAccountUseCase, logger ports.Logger) *FinancialAccountHandler {
	return &FinancialAccountHandler{useCase: useCase, logger: logger}
}

func (handler *FinancialAccountHandler) RegisterRoutes(router chi.Router) {
	router.Get("/financial-accounts", handler.GetAccounts)
	router.Get("/financial-accounts/{id}/summary", handler.GetAccountSummary)
	router.Get("/financial-accounts/{id}/transactions", handler.GetAccountTransactions)
	router.Post("/financial-accounts", handler.CreateAccount)
	router.Put("/financial-accounts/{id}", handler.UpdateAccount)
	router.Patch("/financial-accounts/{id}", handler.UpdateAccount)
	router.Delete("/financial-accounts/{id}", handler.DeleteAccount)
}

func (handler *FinancialAccountHandler) GetAccounts(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	accounts, err := handler.useCase.GetAccounts(request.Context(), userID)
	if err != nil {
		handler.handleError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, financialAccountResponses(accounts))
}

func (handler *FinancialAccountHandler) GetAccountSummary(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	summary, err := handler.useCase.GetAccountSummary(request.Context(), userID, chi.URLParam(request, "id"))
	if err != nil {
		handler.handleError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, financialAccountSummaryResponseFromDomain(summary))
}

func (handler *FinancialAccountHandler) GetAccountTransactions(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	filter, ok := handler.transactionDateFilterFromRequest(response, request)
	if !ok {
		return
	}
	transactions, err := handler.useCase.GetAccountTransactions(request.Context(), userID, chi.URLParam(request, "id"), filter)
	if err != nil {
		handler.handleError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, transactionListResponseFromDomain(transactions))
}

func (handler *FinancialAccountHandler) CreateAccount(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	var createRequest financialAccountRequest
	if err := json.NewDecoder(request.Body).Decode(&createRequest); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	account, err := handler.useCase.CreateAccount(request.Context(), userID, domain.FinancialAccountType(createRequest.Type), createRequest.Name, createRequest.Institution, createRequest.Last4, createRequest.OpeningBalanceCents, createRequest.CreditLimitCents, createRequest.CutoffDay, createRequest.PaymentDay, createRequest.Color)
	if err != nil {
		handler.handleError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, financialAccountResponseFromDomain(account))
}

func (handler *FinancialAccountHandler) UpdateAccount(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	var updateRequest financialAccountRequest
	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	err := handler.useCase.UpdateAccount(request.Context(), userID, chi.URLParam(request, "id"), domain.FinancialAccountType(updateRequest.Type), updateRequest.Name, updateRequest.Institution, updateRequest.Last4, updateRequest.OpeningBalanceCents, updateRequest.CreditLimitCents, updateRequest.CutoffDay, updateRequest.PaymentDay, updateRequest.Color)
	if err != nil {
		handler.handleError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (handler *FinancialAccountHandler) DeleteAccount(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	if err := handler.useCase.DeleteAccount(request.Context(), userID, chi.URLParam(request, "id")); err != nil {
		handler.handleError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (handler *FinancialAccountHandler) handleError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrFinancialAccountNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "financial account not found"})
	case isFinancialAccountDomainValidationError(err):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid financial account data"})
	default:
		handler.logger.Error("financial account request failed", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func isFinancialAccountDomainValidationError(err error) bool {
	return errors.Is(err, domain.ErrInvalidFinancialAccountID) ||
		errors.Is(err, domain.ErrInvalidFinancialAccountUserID) ||
		errors.Is(err, domain.ErrInvalidFinancialAccountType) ||
		errors.Is(err, domain.ErrInvalidFinancialAccountName) ||
		errors.Is(err, domain.ErrInvalidFinancialAccountBalance) ||
		errors.Is(err, domain.ErrInvalidFinancialAccountCreditLimit) ||
		errors.Is(err, domain.ErrInvalidFinancialAccountCutoffDay) ||
		errors.Is(err, domain.ErrInvalidFinancialAccountPaymentDay)
}

func (handler *FinancialAccountHandler) transactionDateFilterFromRequest(response http.ResponseWriter, request *http.Request) (ports.TransactionDateFilter, bool) {
	query := request.URL.Query()
	startDate, hasStartDate, ok := handler.optionalDateQueryParam(response, query.Get("start_date"), "invalid start_date")
	if !ok {
		return ports.TransactionDateFilter{}, false
	}
	endDate, hasEndDate, ok := handler.optionalDateQueryParam(response, query.Get("end_date"), "invalid end_date")
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

func (handler *FinancialAccountHandler) optionalDateQueryParam(response http.ResponseWriter, rawDate, errorMessage string) (time.Time, bool, bool) {
	if rawDate == "" {
		return time.Time{}, false, true
	}
	parsedDate, err := time.Parse(transactionDateLayout, rawDate)
	if err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: errorMessage})
		return time.Time{}, false, false
	}
	return parsedDate, true, true
}

type financialAccountRequest struct {
	Type                string  `json:"type"`
	Name                string  `json:"name"`
	Institution         string  `json:"institution"`
	Last4               *string `json:"last4"`
	OpeningBalanceCents int64   `json:"openingBalanceCents"`
	CreditLimitCents    *int64  `json:"creditLimitCents"`
	CutoffDay           *int    `json:"cutoffDay"`
	PaymentDay          *int    `json:"paymentDay"`
	Color               string  `json:"color"`
}

type financialAccountResponse struct {
	ID                  string  `json:"id"`
	Type                string  `json:"type"`
	Name                string  `json:"name"`
	Institution         string  `json:"institution"`
	Last4               *string `json:"last4"`
	OpeningBalanceCents int64   `json:"openingBalanceCents"`
	CurrentBalanceCents int64   `json:"currentBalanceCents"`
	CreditLimitCents    *int64  `json:"creditLimitCents"`
	CutoffDay           *int    `json:"cutoffDay"`
	PaymentDay          *int    `json:"paymentDay"`
	Color               string  `json:"color"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
}

type financialAccountSummaryResponse struct {
	AccountID            string `json:"accountId"`
	CurrentBalanceCents  int64  `json:"currentBalanceCents"`
	OpeningBalanceCents  int64  `json:"openingBalanceCents"`
	CreditLimitCents     *int64 `json:"creditLimitCents"`
	CurrentDebtCents     int64  `json:"currentDebtCents"`
	AvailableCreditCents int64  `json:"availableCreditCents"`
	TotalIncomeCents     int64  `json:"totalIncomeCents"`
	TotalExpenseCents    int64  `json:"totalExpenseCents"`
	CalculatedAt         string `json:"calculatedAt"`
}

func financialAccountResponses(accounts []*domain.FinancialAccount) []financialAccountResponse {
	responses := make([]financialAccountResponse, 0, len(accounts))
	for _, account := range accounts {
		if account != nil {
			responses = append(responses, financialAccountResponseFromDomain(account))
		}
	}
	return responses
}

func financialAccountResponseFromDomain(account *domain.FinancialAccount) financialAccountResponse {
	return financialAccountResponse{
		ID:                  account.ID(),
		Type:                string(account.Type()),
		Name:                account.Name(),
		Institution:         account.Institution(),
		Last4:               account.Last4(),
		OpeningBalanceCents: account.OpeningBalanceCents(),
		CurrentBalanceCents: account.CurrentBalanceCents(),
		CreditLimitCents:    account.CreditLimitCents(),
		CutoffDay:           account.CutoffDay(),
		PaymentDay:          account.PaymentDay(),
		Color:               account.Color(),
		CreatedAt:           account.CreatedAt().Format(time.RFC3339),
		UpdatedAt:           account.UpdatedAt().Format(time.RFC3339),
	}
}

func financialAccountSummaryResponseFromDomain(summary ports.FinancialAccountSummary) financialAccountSummaryResponse {
	return financialAccountSummaryResponse{
		AccountID:            summary.AccountID,
		CurrentBalanceCents:  summary.CurrentBalanceCents,
		OpeningBalanceCents:  summary.OpeningBalanceCents,
		CreditLimitCents:     summary.CreditLimitCents,
		CurrentDebtCents:     summary.CurrentDebtCents,
		AvailableCreditCents: summary.AvailableCreditCents,
		TotalIncomeCents:     summary.TotalIncomeCents,
		TotalExpenseCents:    summary.TotalExpenseCents,
		CalculatedAt:         summary.CalculatedAt.Format(time.RFC3339),
	}
}
