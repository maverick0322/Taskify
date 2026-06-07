package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockTokenValidator struct {
	userIDToReturn string
	errToReturn    error
}

func (validator *mockTokenValidator) ValidateToken(token string) (string, error) {
	return validator.userIDToReturn, validator.errToReturn
}

type mockMiddlewareLogger struct {
	warnMessages []string
}

func (logger *mockMiddlewareLogger) Info(msg string, keysAndValues ...interface{})  {}
func (logger *mockMiddlewareLogger) Error(msg string, keysAndValues ...interface{}) {}

func (logger *mockMiddlewareLogger) Warn(msg string, keysAndValues ...interface{}) {
	logger.warnMessages = append(logger.warnMessages, msg)
}

func TestAuthMiddleware_MissingAuthorizationHeader_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	middleware := NewAuthMiddleware(&mockTokenValidator{}, &mockMiddlewareLogger{})
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	response := httptest.NewRecorder()

	// Act
	middleware.RequireAuthentication(successHandler()).ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestAuthMiddleware_MalformedAuthorizationHeader_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	middleware := NewAuthMiddleware(&mockTokenValidator{}, &mockMiddlewareLogger{})
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	request.Header.Set(authorizationHeader, "Basic token")
	response := httptest.NewRecorder()

	// Act
	middleware.RequireAuthentication(successHandler()).ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestAuthMiddleware_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	// Arrange
	middleware := NewAuthMiddleware(&mockTokenValidator{errToReturn: errors.New("invalid token")}, &mockMiddlewareLogger{})
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	request.Header.Set(authorizationHeader, "Bearer invalid-token")
	response := httptest.NewRecorder()

	// Act
	middleware.RequireAuthentication(successHandler()).ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestAuthMiddleware_ValidToken_CallsNextWithUserID(t *testing.T) {
	// Arrange
	middleware := NewAuthMiddleware(&mockTokenValidator{userIDToReturn: "user-123"}, &mockMiddlewareLogger{})
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	request.Header.Set(authorizationHeader, "Bearer valid-token")
	response := httptest.NewRecorder()
	var retrievedUserID string

	nextHandler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		retrievedUserID, _ = UserIDFromContext(request.Context())
		response.WriteHeader(http.StatusNoContent)
	})

	// Act
	middleware.RequireAuthentication(nextHandler).ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if retrievedUserID != "user-123" {
		t.Errorf("expected user ID user-123, got %s", retrievedUserID)
	}
}

func TestUserIDFromContext_MissingUserID_ReturnsFalse(t *testing.T) {
	// Arrange
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)

	// Act
	userID, ok := UserIDFromContext(request.Context())

	// Assert
	if ok {
		t.Fatal("expected false, got true")
	}
	if userID != "" {
		t.Errorf("expected empty user ID, got %s", userID)
	}
}

func successHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
}
