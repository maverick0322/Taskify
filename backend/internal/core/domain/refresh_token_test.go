package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewRefreshToken_ValidFields_ReturnsRefreshToken(t *testing.T) {
	// Arrange
	expiresAt := time.Now().Add(time.Hour)

	// Act
	refreshToken, err := NewRefreshToken("session-123", "user-123", "token-hash", expiresAt, false)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if refreshToken.ID() != "session-123" {
		t.Errorf("expected ID session-123, got %s", refreshToken.ID())
	}
	if refreshToken.UserID() != "user-123" {
		t.Errorf("expected user ID user-123, got %s", refreshToken.UserID())
	}
	if refreshToken.TokenHash() != "token-hash" {
		t.Errorf("expected token hash token-hash, got %s", refreshToken.TokenHash())
	}
	if !refreshToken.ExpiresAt().Equal(expiresAt) {
		t.Errorf("expected expiration %v, got %v", expiresAt, refreshToken.ExpiresAt())
	}
	if refreshToken.IsRevoked() {
		t.Error("expected token to not be revoked")
	}
}

func TestNewRefreshToken_EmptyID_ReturnsErrEmptyRefreshTokenID(t *testing.T) {
	// Arrange
	expiresAt := time.Now().Add(time.Hour)

	// Act
	_, err := NewRefreshToken("", "user-123", "token-hash", expiresAt, false)

	// Assert
	if !errors.Is(err, ErrEmptyRefreshTokenID) {
		t.Errorf("expected error %v, got %v", ErrEmptyRefreshTokenID, err)
	}
}

func TestNewRefreshToken_EmptyUserID_ReturnsErrEmptyRefreshTokenUserID(t *testing.T) {
	// Arrange
	expiresAt := time.Now().Add(time.Hour)

	// Act
	_, err := NewRefreshToken("session-123", "", "token-hash", expiresAt, false)

	// Assert
	if !errors.Is(err, ErrEmptyRefreshTokenUserID) {
		t.Errorf("expected error %v, got %v", ErrEmptyRefreshTokenUserID, err)
	}
}

func TestNewRefreshToken_EmptyTokenHash_ReturnsErrEmptyRefreshTokenHash(t *testing.T) {
	// Arrange
	expiresAt := time.Now().Add(time.Hour)

	// Act
	_, err := NewRefreshToken("session-123", "user-123", "", expiresAt, false)

	// Assert
	if !errors.Is(err, ErrEmptyRefreshTokenHash) {
		t.Errorf("expected error %v, got %v", ErrEmptyRefreshTokenHash, err)
	}
}

func TestNewRefreshToken_ZeroExpiration_ReturnsErrInvalidRefreshTokenDate(t *testing.T) {
	// Arrange
	zeroExpiration := time.Time{}

	// Act
	_, err := NewRefreshToken("session-123", "user-123", "token-hash", zeroExpiration, false)

	// Assert
	if !errors.Is(err, ErrInvalidRefreshTokenDate) {
		t.Errorf("expected error %v, got %v", ErrInvalidRefreshTokenDate, err)
	}
}

func TestRefreshToken_IsExpiredPastExpiration_ReturnsTrue(t *testing.T) {
	// Arrange
	now := time.Now()
	refreshToken, _ := NewRefreshToken("session-123", "user-123", "token-hash", now.Add(-time.Minute), false)

	// Act
	isExpired := refreshToken.IsExpired(now)

	// Assert
	if !isExpired {
		t.Error("expected refresh token to be expired")
	}
}

func TestRefreshToken_IsExpiredFutureExpiration_ReturnsFalse(t *testing.T) {
	// Arrange
	now := time.Now()
	refreshToken, _ := NewRefreshToken("session-123", "user-123", "token-hash", now.Add(time.Minute), false)

	// Act
	isExpired := refreshToken.IsExpired(now)

	// Assert
	if isExpired {
		t.Error("expected refresh token to not be expired")
	}
}

func TestRefreshToken_Revoke_MarksTokenAsRevoked(t *testing.T) {
	// Arrange
	refreshToken, _ := NewRefreshToken("session-123", "user-123", "token-hash", time.Now().Add(time.Hour), false)

	// Act
	refreshToken.Revoke()

	// Assert
	if !refreshToken.IsRevoked() {
		t.Error("expected refresh token to be revoked")
	}
}
