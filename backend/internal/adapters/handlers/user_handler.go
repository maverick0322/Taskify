package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/adapters/handlers/middleware"
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
	router.Post("/users/refresh", handler.RefreshSession)
}

func (handler *UserHandler) RegisterProtectedRoutes(router chi.Router) {
	router.Get("/users/me", handler.GetMe)
	router.Patch("/users/me", handler.UpdateMe)
	router.Patch("/users/{id}/avatar", handler.UpdateAvatar)
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

	accessToken, refreshToken, err := handler.userUseCase.Authenticate(request.Context(), loginRequest.Email, loginRequest.Password)
	if err != nil {
		handler.handleLoginError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, tokenPairResponse{AccessToken: accessToken, RefreshToken: refreshToken})
}

func (handler *UserHandler) RefreshSession(response http.ResponseWriter, request *http.Request) {
	var refreshRequest refreshSessionRequest
	if err := json.NewDecoder(request.Body).Decode(&refreshRequest); err != nil {
		handler.logger.Warn("refresh request contains invalid json")
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	accessToken, refreshToken, err := handler.userUseCase.RefreshSession(request.Context(), refreshRequest.RefreshToken)
	if err != nil {
		handler.handleRefreshError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, tokenPairResponse{AccessToken: accessToken, RefreshToken: refreshToken})
}

func (handler *UserHandler) GetMe(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	user, err := handler.userUseCase.GetProfile(request.Context(), userID)
	if err != nil {
		handler.handleProfileError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, userProfileResponseFromDomain(user))
}

func (handler *UserHandler) UpdateMe(response http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	var profileRequest updateProfileRequest
	if err := json.NewDecoder(request.Body).Decode(&profileRequest); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if strings.TrimSpace(profileRequest.Name) == "" {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}

	user, err := handler.userUseCase.UpdateProfileName(request.Context(), userID, profileRequest.Name)
	if err != nil {
		handler.handleProfileError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, userProfileResponseFromDomain(user))
}

func (handler *UserHandler) UpdateAvatar(response http.ResponseWriter, request *http.Request) {
	authenticatedUserID, ok := middleware.UserIDFromContext(request.Context())
	if !ok {
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	requestedUserID := chi.URLParam(request, "id")
	if requestedUserID != authenticatedUserID {
		writeJSON(response, http.StatusForbidden, errorResponse{Error: "forbidden"})
		return
	}

	var avatarRequest updateAvatarRequest
	if err := json.NewDecoder(request.Body).Decode(&avatarRequest); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	avatarLocalPath := strings.TrimSpace(avatarRequest.AvatarLocalPath)
	if avatarLocalPath == "" {
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "avatar path is required"})
		return
	}

	user, err := handler.userUseCase.UpdateAvatarLocalPath(request.Context(), authenticatedUserID, avatarLocalPath)
	if err != nil {
		handler.handleProfileError(response, err)
		return
	}

	writeJSON(response, http.StatusOK, userProfileResponseFromDomain(user))
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

func (handler *UserHandler) handleRefreshError(response http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrInvalidRefreshToken) ||
		errors.Is(err, services.ErrSessionRevoked) ||
		errors.Is(err, services.ErrRefreshSessionExpired) {
		handler.logger.Warn("refresh rejected because session token is invalid")
		writeJSON(response, http.StatusUnauthorized, errorResponse{Error: "invalid refresh token"})
		return
	}

	handler.logger.Error("refresh failed due to internal processing error", "error", err)
	writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}

func (handler *UserHandler) handleProfileError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "user not found"})
	case errors.Is(err, services.ErrInvalidAvatarPath):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "avatar path is required"})
	case isDomainValidationError(err):
		writeJSON(response, http.StatusBadRequest, errorResponse{Error: "invalid user data"})
	default:
		handler.logger.Error("profile request failed due to internal processing error", "error", err)
		writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
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

type refreshSessionRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type updateAvatarRequest struct {
	AvatarLocalPath string `json:"avatarLocalPath"`
}

type updateProfileRequest struct {
	Name string `json:"name"`
}

type userProfileResponse struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	AvatarLocalPath string `json:"avatarLocalPath,omitempty"`
	AvatarURL       string `json:"avatarUrl,omitempty"`
}

func userProfileResponseFromDomain(user *domain.User) userProfileResponse {
	profile := user.Profile()
	return userProfileResponse{
		ID:              user.ID(),
		Email:           user.Email(),
		FirstName:       profile.FirstName(),
		LastName:        profile.LastName(),
		AvatarLocalPath: user.AvatarLocalPath(),
		AvatarURL:       user.AvatarURL(),
	}
}

type tokenPairResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type errorResponse struct {
	Error string `json:"error"`
}
