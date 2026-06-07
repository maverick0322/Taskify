package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maverick0322/taskify/backend/internal/core/domain"
	"github.com/maverick0322/taskify/backend/internal/core/services"
)

type mockUserUseCase struct {
	userToReturn         *domain.User
	accessTokenToReturn  string
	refreshTokenToReturn string
	errToReturn          error
}

func (useCase *mockUserUseCase) Register(ctx context.Context, email, plainPassword, firstName, lastName string, birthDate time.Time) (*domain.User, error) {
	return useCase.userToReturn, useCase.errToReturn
}

func (useCase *mockUserUseCase) Authenticate(ctx context.Context, email, plainPassword string) (string, string, error) {
	return useCase.accessTokenToReturn, useCase.refreshTokenToReturn, useCase.errToReturn
}

func (useCase *mockUserUseCase) RefreshSession(ctx context.Context, refreshToken string) (string, string, error) {
	return useCase.accessTokenToReturn, useCase.refreshTokenToReturn, useCase.errToReturn
}

func (useCase *mockUserUseCase) UpdateProfile(ctx context.Context, userID, firstName, lastName string, birthDate time.Time) error {
	return useCase.errToReturn
}

type mockHandlerLogger struct {
	errorMessages []string
	warnMessages  []string
}

func (logger *mockHandlerLogger) Info(msg string, keysAndValues ...interface{}) {}

func (logger *mockHandlerLogger) Warn(msg string, keysAndValues ...interface{}) {
	logger.warnMessages = append(logger.warnMessages, msg)
}

func (logger *mockHandlerLogger) Error(msg string, keysAndValues ...interface{}) {
	logger.errorMessages = append(logger.errorMessages, msg)
}

func TestUserHandler_RegisterValidRequest_ReturnsCreated(t *testing.T) {
	// Arrange
	user := createHandlerTestUser(t)
	router := createUserTestRouter(&mockUserUseCase{userToReturn: user}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/register", strings.NewReader(validRegisterJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"id":"user-123"`) {
		t.Errorf("expected response to contain user ID")
	}
}

func TestUserHandler_RegisterMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/register", strings.NewReader("{"))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestUserHandler_RegisterInvalidBirthDate_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{}, &mockHandlerLogger{})
	requestBody := `{"email":"test@domain.com","password":"securePass123","firstName":"John","lastName":"Doe","birthDate":"01-02-1990"}`
	request := httptest.NewRequest(http.MethodPost, "/users/register", strings.NewReader(requestBody))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestUserHandler_RegisterDuplicateEmail_ReturnsConflict(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: services.ErrUserAlreadyExists}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/register", strings.NewReader(validRegisterJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, response.Code)
	}
}

func TestUserHandler_RegisterDomainError_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: domain.ErrInvalidEmail}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/register", strings.NewReader(validRegisterJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestUserHandler_RegisterInternalError_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: services.ErrInternalProcessing}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/register", strings.NewReader(validRegisterJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestUserHandler_LoginValidCredentials_ReturnsOK(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{accessTokenToReturn: "access-token", refreshTokenToReturn: "refresh-token"}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/login", strings.NewReader(validLoginJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"accessToken":"access-token"`) {
		t.Errorf("expected response to contain access token")
	}
	if !strings.Contains(response.Body.String(), `"refreshToken":"refresh-token"`) {
		t.Errorf("expected response to contain refresh token")
	}
}

func TestUserHandler_LoginInvalidCredentials_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: services.ErrInvalidCredentials}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/login", strings.NewReader(validLoginJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestUserHandler_LoginInternalError_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: errors.New("token generator failure")}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/login", strings.NewReader(validLoginJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func TestUserHandler_RefreshValidToken_ReturnsOK(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{accessTokenToReturn: "new-access-token", refreshTokenToReturn: "new-refresh-token"}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/refresh", strings.NewReader(validRefreshJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"accessToken":"new-access-token"`) {
		t.Errorf("expected response to contain refreshed access token")
	}
}

func TestUserHandler_RefreshMalformedJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/refresh", strings.NewReader("{"))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestUserHandler_RefreshInvalidToken_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: services.ErrInvalidRefreshToken}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/refresh", strings.NewReader(validRefreshJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestUserHandler_RefreshRevokedToken_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: services.ErrSessionRevoked}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/refresh", strings.NewReader(validRefreshJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestUserHandler_RefreshExpiredToken_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: services.ErrRefreshSessionExpired}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/refresh", strings.NewReader(validRefreshJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestUserHandler_RefreshInternalError_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	router := createUserTestRouter(&mockUserUseCase{errToReturn: services.ErrInternalProcessing}, &mockHandlerLogger{})
	request := httptest.NewRequest(http.MethodPost, "/users/refresh", strings.NewReader(validRefreshJSON()))
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func createUserTestRouter(useCase *mockUserUseCase, logger *mockHandlerLogger) chi.Router {
	router := chi.NewRouter()
	handler := NewUserHandler(useCase, logger)
	handler.RegisterRoutes(router)
	return router
}

func createHandlerTestUser(t *testing.T) *domain.User {
	t.Helper()

	profile, err := domain.NewUserProfile("John", "Doe", time.Date(1990, time.January, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected profile to be valid, got: %v", err)
	}

	user, err := domain.NewUser("user-123", "test@domain.com", "hashedPassword", profile)
	if err != nil {
		t.Fatalf("expected user to be valid, got: %v", err)
	}

	return user
}

func validRegisterJSON() string {
	return `{"email":"test@domain.com","password":"securePass123","firstName":"John","lastName":"Doe","birthDate":"1990-01-02"}`
}

func validLoginJSON() string {
	return `{"email":"test@domain.com","password":"securePass123"}`
}

func validRefreshJSON() string {
	return `{"refreshToken":"refresh-token"}`
}
