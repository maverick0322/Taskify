package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

const (
	birthDateLayout = "2006-01-02"

	contentTypeHeader = "Content-Type"
	jsonContentType   = "application/json"
)

// UserHandler adapts HTTP requests to the user application port.
type UserHandler struct {
	userUseCase ports.UserUseCase
	logger      ports.Logger
}

// NewUserHandler injects the use case instead of coupling HTTP to application internals.
func NewUserHandler(userUseCase ports.UserUseCase, logger ports.Logger) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
		logger:      logger,
	}
}

func (handler *UserHandler) RegisterRoutes(router chi.Router) {
	router.Post("/users/register", handler.Register)
	router.Post("/users/login", handler.Login)
}

func (handler *UserHandler) Register(response http.ResponseWriter, request *http.Request) {
	var registerRequest registerUserRequest
	if err := json.NewDecoder(request.Body).Decode(&registerRequest); err != nil {
		handler.logger.Warn("registration request contains invalid json")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	birthDate, err := time.Parse(birthDateLayout, registerRequest.BirthDate)
	if err != nil {
		handler.logger.Warn("registration request contains invalid birth date")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid birth date"})
		return
	}

	user, err := handler.userUseCase.Register(
		request.Context(),
		registerRequest.Email,
		registerRequest.Password,
		registerRequest.FirstName,
		registerRequest.LastName,
		birthDate,
	)
	if err != nil {
		handler.handleRegisterError(response, err)
		return
	}

	writeJSON(response, http.StatusCreated, registerUserResponse{
		ID:    user.ID(),
		Email: user.Email(),
	})
}

func (handler *UserHandler) Login(response http.ResponseWriter, request *http.Request) {
	var loginRequest loginUserRequest
	if err := json.NewDecoder(request.Body).Decode(&loginRequest); err != nil {
		handler.logger.Warn("login request contains invalid json")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	token, err := handler.userUseCase.Authenticate(request.Context(), loginRequest.Email, loginRequest.Password)
	if err != nil {
		handler.handleLoginError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, loginUserResponse{Token: token})
}

func (handler *UserHandler) handleRegisterError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrUserAlreadyExists):
		handler.logger.Warn("registration rejected because user already exists")
		writeJSON(response, http.StatusConflict, errorResponse{Error: "user already exists"})
	case isDomainValidationError(err):
		handler.logger.Warn("registration rejected because user data is invalid")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid user data"})
	default:
		handler.logger.Error("registration failed due to internal processing error", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func (handler *UserHandler) handleLoginError(response http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrInvalidCredentials) {
		handler.logger.Warn("login rejected because credentials are invalid")
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "invalid credentials"})
		return
	}

	handler.logger.Error("login failed due to internal processing error", "error", err)
	writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}

func isDomainValidationError(err error) bool {
	return errors.Is(err, domain.ErrInvalidName) ||
		errors.Is(err, domain.ErrInvalidEmail) ||
		errors.Is(err, domain.ErrInvalidPassword) ||
		errors.Is(err, domain.ErrUnderageUser) ||
		errors.Is(err, domain.ErrEmptyID)
}

func writeJSON(response http.ResponseWriter, statusCode int, payload interface{}) {
	response.Header().Set(contentTypeHeader, jsonContentType)
	response.WriteHeader(statusCode)
	if err := json.NewEncoder(response).Encode(payload); err != nil {
		return
	}
}

type registerUserRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	BirthDate string `json:"birthDate"`
}

type registerUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type loginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginUserResponse struct {
	Token string `json:"token"`
}

type errorResponse struct {
	Error string `json:"error"`
}
