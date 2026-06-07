package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyRefreshTokenID     = errors.New("domain: refresh token ID cannot be empty")
	ErrEmptyRefreshTokenUserID = errors.New("domain: refresh token user ID cannot be empty")
	ErrEmptyRefreshTokenHash   = errors.New("domain: refresh token hash cannot be empty")
	ErrInvalidRefreshTokenDate = errors.New("domain: refresh token expiration cannot be zero")
)

// RefreshToken represents a persisted session credential without storing the raw token.
type RefreshToken struct {
	id        string
	userID    string
	tokenHash string
	expiresAt time.Time
	isRevoked bool
}

// NewRefreshToken rebuilds persisted sessions and creates new ones through the same invariants.
func NewRefreshToken(id, userID, tokenHash string, expiresAt time.Time, isRevoked bool) (*RefreshToken, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrEmptyRefreshTokenID
	}
	if strings.TrimSpace(userID) == "" {
		return nil, ErrEmptyRefreshTokenUserID
	}
	if strings.TrimSpace(tokenHash) == "" {
		return nil, ErrEmptyRefreshTokenHash
	}
	if expiresAt.IsZero() {
		return nil, ErrInvalidRefreshTokenDate
	}

	return &RefreshToken{
		id:        strings.TrimSpace(id),
		userID:    strings.TrimSpace(userID),
		tokenHash: strings.TrimSpace(tokenHash),
		expiresAt: expiresAt,
		isRevoked: isRevoked,
	}, nil
}

func (refreshToken *RefreshToken) ID() string {
	return refreshToken.id
}

func (refreshToken *RefreshToken) UserID() string {
	return refreshToken.userID
}

func (refreshToken *RefreshToken) TokenHash() string {
	return refreshToken.tokenHash
}

func (refreshToken *RefreshToken) ExpiresAt() time.Time {
	return refreshToken.expiresAt
}

func (refreshToken *RefreshToken) IsRevoked() bool {
	return refreshToken.isRevoked
}

func (refreshToken *RefreshToken) IsExpired(now time.Time) bool {
	return !now.Before(refreshToken.expiresAt)
}

func (refreshToken *RefreshToken) Revoke() {
	refreshToken.isRevoked = true
}
