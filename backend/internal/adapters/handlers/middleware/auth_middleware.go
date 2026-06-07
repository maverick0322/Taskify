package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const (
	authorizationHeader = "Authorization"
	bearerTokenPrefix   = "Bearer "
)

type authenticatedUserIDKey struct{}

type AuthMiddleware struct {
	tokenValidator ports.TokenValidator
	logger         ports.Logger
}

func NewAuthMiddleware(tokenValidator ports.TokenValidator, logger ports.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator: tokenValidator,
		logger:         logger,
	}
}

func (middleware *AuthMiddleware) RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		token, ok := bearerTokenFromRequest(request)
		if !ok {
			middleware.logger.Warn("request rejected because authorization header is invalid")
			writeUnauthorized(response)
			return
		}

		userID, err := middleware.tokenValidator.ValidateToken(token)
		if err != nil {
			middleware.logger.Warn("request rejected because access token is invalid")
			writeUnauthorized(response)
			return
		}

		next.ServeHTTP(response, request.WithContext(ContextWithUserID(request.Context(), userID)))
	})
}

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, authenticatedUserIDKey{}, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(authenticatedUserIDKey{}).(string)
	if !ok || userID == "" {
		return "", false
	}

	return userID, true
}

func bearerTokenFromRequest(request *http.Request) (string, bool) {
	authorizationValue := request.Header.Get(authorizationHeader)
	if !strings.HasPrefix(authorizationValue, bearerTokenPrefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authorizationValue, bearerTokenPrefix))
	if token == "" {
		return "", false
	}

	return token, true
}

func writeUnauthorized(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusUnauthorized)
	if err := json.NewEncoder(response).Encode(struct {
		Error string `json:"error"`
	}{Error: "unauthorized"}); err != nil {
		return
	}
}
