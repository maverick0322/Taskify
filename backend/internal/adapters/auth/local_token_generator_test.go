package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

func TestNewLocalTokenGenerator_InvalidTTL_ReturnsErrInvalidTokenTTL(t *testing.T) {
	_, err := NewLocalTokenGenerator(0, 24*time.Hour)
	if err != ErrInvalidTokenTTL {
		t.Fatalf("expected %v, got %v", ErrInvalidTokenTTL, err)
	}
}

func TestLocalTokenGenerator_GenerateTokenPair_ReturnsJWTLikeClaims(t *testing.T) {
	generator, err := NewLocalTokenGenerator(5*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	localGenerator := generator.(*LocalTokenGenerator)
	localGenerator.now = func() time.Time {
		return time.Date(2025, time.June, 26, 12, 0, 0, 0, time.UTC)
	}

	tokenPair, err := localGenerator.GenerateTokenPair(ports.TokenSubject{
		UserID:    "user-123",
		Email:     "user@example.com",
		FirstName: "Jane",
		LastName:  "Doe",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tokenPair.RefreshToken == "" {
		t.Fatal("expected refresh token")
	}
	if strings.Count(tokenPair.AccessToken, ".") != 2 {
		t.Fatalf("expected jwt-like token with three segments, got %q", tokenPair.AccessToken)
	}

	claims := decodeLocalClaims(t, tokenPair.AccessToken)
	if claims.Subject != "user-123" {
		t.Fatalf("expected subject user-123, got %s", claims.Subject)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", claims.Email)
	}
	if claims.FirstName != "Jane" || claims.LastName != "Doe" {
		t.Fatalf("expected Jane Doe, got %s %s", claims.FirstName, claims.LastName)
	}
}

func TestLocalTokenGenerator_ValidateTokenExpiredToken_ReturnsErrExpiredToken(t *testing.T) {
	generator, err := NewLocalTokenGenerator(5*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	localGenerator := generator.(*LocalTokenGenerator)
	localGenerator.now = func() time.Time {
		return time.Date(2025, time.June, 26, 12, 10, 0, 0, time.UTC)
	}

	expiredToken := localTokenFromClaims(t, localTokenClaims{
		Subject:   "user-123",
		Email:     "user@example.com",
		FirstName: "Jane",
		LastName:  "Doe",
		Role:      "authenticated",
		IssuedAt:  time.Date(2025, time.June, 26, 12, 0, 0, 0, time.UTC).Unix(),
		ExpiresAt: time.Date(2025, time.June, 26, 12, 5, 0, 0, time.UTC).Unix(),
	})

	_, err = localGenerator.ValidateToken(expiredToken)
	if err != ErrExpiredToken {
		t.Fatalf("expected %v, got %v", ErrExpiredToken, err)
	}
}

func TestLocalTokenGenerator_ValidateTokenMissingSubject_ReturnsErrInvalidToken(t *testing.T) {
	generator, err := NewLocalTokenGenerator(5*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	localGenerator := generator.(*LocalTokenGenerator)
	localGenerator.now = func() time.Time {
		return time.Date(2025, time.June, 26, 12, 0, 0, 0, time.UTC)
	}

	token := localTokenFromClaims(t, localTokenClaims{
		Email:     "user@example.com",
		FirstName: "Jane",
		LastName:  "Doe",
		Role:      "authenticated",
		IssuedAt:  time.Date(2025, time.June, 26, 12, 0, 0, 0, time.UTC).Unix(),
		ExpiresAt: time.Date(2025, time.June, 26, 12, 5, 0, 0, time.UTC).Unix(),
	})

	_, err = localGenerator.ValidateToken(token)
	if err != ErrInvalidToken {
		t.Fatalf("expected %v, got %v", ErrInvalidToken, err)
	}
}

func decodeLocalClaims(t *testing.T, token string) localTokenClaims {
	t.Helper()

	segments := strings.Split(token, ".")
	if len(segments) < 2 {
		t.Fatalf("expected jwt-like token, got %q", token)
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		t.Fatalf("expected payload to decode, got %v", err)
	}

	var claims localTokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		t.Fatalf("expected payload json, got %v", err)
	}

	return claims
}

func localTokenFromClaims(t *testing.T, claims localTokenClaims) string {
	t.Helper()

	headerSegment, err := encodeLocalTokenSegment(map[string]string{
		"alg": "none",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("expected header encode to succeed, got %v", err)
	}
	payloadSegment, err := encodeLocalTokenSegment(claims)
	if err != nil {
		t.Fatalf("expected payload encode to succeed, got %v", err)
	}

	return headerSegment + "." + payloadSegment + "."
}
