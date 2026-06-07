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
	tokenTTL := 15 * time.Minute

	// Act
	generator, err := NewJWTTokenGenerator(secretKey, tokenTTL)

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
	tokenTTL := 15 * time.Minute

	// Act
	generator, err := NewJWTTokenGenerator(emptySecretKey, tokenTTL)

	// Assert
	if generator != nil {
		t.Fatal("expected nil generator")
	}
	if !errors.Is(err, ErrEmptyJWTSecret) {
		t.Errorf("expected error %v, got %v", ErrEmptyJWTSecret, err)
	}
}

func TestNewJWTTokenGenerator_InvalidTTL_ReturnsErrInvalidTokenTTL(t *testing.T) {
	// Arrange
	secretKey := "test-secret"
	invalidTokenTTL := 0 * time.Second

	// Act
	generator, err := NewJWTTokenGenerator(secretKey, invalidTokenTTL)

	// Assert
	if generator != nil {
		t.Fatal("expected nil generator")
	}
	if !errors.Is(err, ErrInvalidTokenTTL) {
		t.Errorf("expected error %v, got %v", ErrInvalidTokenTTL, err)
	}
}

func TestJWTTokenGenerator_GenerateTokenValidSubject_ReturnsSignedToken(t *testing.T) {
	// Arrange
	secretKey := "test-secret"
	tokenTTL := 15 * time.Minute
	userID := "user-123"
	generator, _ := NewJWTTokenGenerator(secretKey, tokenTTL)

	// Act
	signedToken, err := generator.GenerateToken(userID)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if signedToken == "" {
		t.Fatal("expected signed token, got empty string")
	}
}

func TestJWTTokenGenerator_GenerateTokenEmptySubject_ReturnsErrEmptyTokenSubject(t *testing.T) {
	// Arrange
	generator, _ := NewJWTTokenGenerator("test-secret", 15*time.Minute)

	// Act
	signedToken, err := generator.GenerateToken("")

	// Assert
	if signedToken != "" {
		t.Fatal("expected empty signed token")
	}
	if !errors.Is(err, ErrEmptyTokenSubject) {
		t.Errorf("expected error %v, got %v", ErrEmptyTokenSubject, err)
	}
}

func TestJWTTokenGenerator_GenerateTokenValidSubject_ContainsExpectedClaims(t *testing.T) {
	// Arrange
	secretKey := "test-secret"
	tokenTTL := 15 * time.Minute
	userID := "user-123"
	generator, _ := NewJWTTokenGenerator(secretKey, tokenTTL)

	// Act
	signedToken, err := generator.GenerateToken(userID)
	parsedToken, claims := parseSignedToken(t, signedToken, secretKey)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if !parsedToken.Valid {
		t.Fatal("expected token to be valid")
	}
	if claims.Subject != userID {
		t.Errorf("expected subject %s, got %s", userID, claims.Subject)
	}
	if claims.IssuedAt == nil {
		t.Fatal("expected issued at claim")
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected expires at claim")
	}
	if claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) != tokenTTL {
		t.Errorf("expected token ttl %v, got %v", tokenTTL, claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time))
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
