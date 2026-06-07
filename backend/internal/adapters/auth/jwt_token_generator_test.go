package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewJWTTokenGenerator_ValidSettings_ReturnsGenerator(t *testing.T) {
	// Arrange
	secretKey := "test-secret"
	accessTokenTTL := 5 * time.Minute
	refreshTokenTTL := 24 * time.Hour

	// Act
	generator, err := NewJWTTokenGenerator(secretKey, accessTokenTTL, refreshTokenTTL)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if generator == nil {
		t.Fatal("expected generator, got nil")
	}
}

func TestNewJWTTokenGenerator_EmptySecret_ReturnsErrEmptyJWTSecret(t *testing.T) {
	// Arrange
	emptySecretKey := ""

	// Act
	generator, err := NewJWTTokenGenerator(emptySecretKey, 5*time.Minute, 24*time.Hour)

	// Assert
	if generator != nil {
		t.Fatal("expected nil generator")
	}
	if !errors.Is(err, ErrEmptyJWTSecret) {
		t.Errorf("expected error %v, got %v", ErrEmptyJWTSecret, err)
	}
}

func TestNewJWTTokenGenerator_InvalidAccessTTL_ReturnsErrInvalidTokenTTL(t *testing.T) {
	// Arrange
	secretKey := "test-secret"

	// Act
	generator, err := NewJWTTokenGenerator(secretKey, 0, 24*time.Hour)

	// Assert
	if generator != nil {
		t.Fatal("expected nil generator")
	}
	if !errors.Is(err, ErrInvalidTokenTTL) {
		t.Errorf("expected error %v, got %v", ErrInvalidTokenTTL, err)
	}
}

func TestNewJWTTokenGenerator_InvalidRefreshTTL_ReturnsErrInvalidTokenTTL(t *testing.T) {
	// Arrange
	secretKey := "test-secret"

	// Act
	generator, err := NewJWTTokenGenerator(secretKey, 5*time.Minute, 0)

	// Assert
	if generator != nil {
		t.Fatal("expected nil generator")
	}
	if !errors.Is(err, ErrInvalidTokenTTL) {
		t.Errorf("expected error %v, got %v", ErrInvalidTokenTTL, err)
	}
}

func TestJWTTokenGenerator_GenerateTokenPairValidSubject_ReturnsSignedTokens(t *testing.T) {
	// Arrange
	generator, _ := NewJWTTokenGenerator("test-secret", 5*time.Minute, 24*time.Hour)

	// Act
	tokenPair, err := generator.GenerateTokenPair("user-123")

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if tokenPair.AccessToken == "" {
		t.Fatal("expected access token, got empty string")
	}
	if tokenPair.RefreshToken == "" {
		t.Fatal("expected refresh token, got empty string")
	}
	if tokenPair.AccessToken == tokenPair.RefreshToken {
		t.Fatal("expected access and refresh tokens to differ")
	}
}

func TestJWTTokenGenerator_GenerateTokenPairEmptySubject_ReturnsErrEmptyTokenSubject(t *testing.T) {
	// Arrange
	generator, _ := NewJWTTokenGenerator("test-secret", 5*time.Minute, 24*time.Hour)

	// Act
	tokenPair, err := generator.GenerateTokenPair("")

	// Assert
	if tokenPair.AccessToken != "" {
		t.Fatal("expected empty access token")
	}
	if !errors.Is(err, ErrEmptyTokenSubject) {
		t.Errorf("expected error %v, got %v", ErrEmptyTokenSubject, err)
	}
}

func TestJWTTokenGenerator_GenerateTokenPairValidSubject_ContainsExpectedClaims(t *testing.T) {
	// Arrange
	secretKey := "test-secret"
	accessTokenTTL := 5 * time.Minute
	refreshTokenTTL := 24 * time.Hour
	userID := "user-123"
	generator, _ := NewJWTTokenGenerator(secretKey, accessTokenTTL, refreshTokenTTL)

	// Act
	tokenPair, err := generator.GenerateTokenPair(userID)
	accessToken, accessClaims := parseSignedToken(t, tokenPair.AccessToken, secretKey)
	refreshToken, refreshClaims := parseSignedToken(t, tokenPair.RefreshToken, secretKey)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !accessToken.Valid {
		t.Fatal("expected access token to be valid")
	}
	if !refreshToken.Valid {
		t.Fatal("expected refresh token to be valid")
	}
	if accessClaims.Subject != userID {
		t.Errorf("expected access subject %s, got %s", userID, accessClaims.Subject)
	}
	if refreshClaims.Subject != userID {
		t.Errorf("expected refresh subject %s, got %s", userID, refreshClaims.Subject)
	}
	if accessClaims.ExpiresAt.Time.Sub(accessClaims.IssuedAt.Time) != accessTokenTTL {
		t.Errorf("expected access ttl %v, got %v", accessTokenTTL, accessClaims.ExpiresAt.Time.Sub(accessClaims.IssuedAt.Time))
	}
	if refreshClaims.ExpiresAt.Time.Sub(refreshClaims.IssuedAt.Time) != refreshTokenTTL {
		t.Errorf("expected refresh ttl %v, got %v", refreshTokenTTL, refreshClaims.ExpiresAt.Time.Sub(refreshClaims.IssuedAt.Time))
	}
}

func parseSignedToken(t *testing.T, signedToken, secretKey string) (*jwt.Token, *jwt.RegisteredClaims) {
	t.Helper()

	claims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(signedToken, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(secretKey), nil
	})
	if err != nil {
		t.Fatalf("expected token to parse, got: %v", err)
	}

	return parsedToken, claims
}
