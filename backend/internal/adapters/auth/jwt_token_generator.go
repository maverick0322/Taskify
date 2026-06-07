package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

var (
	ErrEmptyJWTSecret     = errors.New("auth: jwt secret cannot be empty")
	ErrInvalidTokenTTL    = errors.New("auth: token ttl must be positive")
	ErrEmptyTokenSubject  = errors.New("auth: token subject cannot be empty")
	ErrTokenSigningFailed = errors.New("auth: token signing failed")
)

// JWTTokenGenerator signs stateless access tokens without leaking JWT details into the core.
type JWTTokenGenerator struct {
	secretKey []byte
	tokenTTL  time.Duration
}

// NewJWTTokenGenerator injects secret and lifetime so security settings stay outside code.
func NewJWTTokenGenerator(secretKey string, tokenTTL time.Duration) (ports.TokenGenerator, error) {
	if secretKey == "" {
		return nil, ErrEmptyJWTSecret
	}
	if tokenTTL <= 0 {
		return nil, ErrInvalidTokenTTL
	}

	return &JWTTokenGenerator{
		secretKey: []byte(secretKey),
		tokenTTL:  tokenTTL,
	}, nil
}

func (generator *JWTTokenGenerator) GenerateToken(userID string) (string, error) {
	if userID == "" {
		return "", ErrEmptyTokenSubject
	}

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(generator.tokenTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(generator.secretKey)
	if err != nil {
		return "", ErrTokenSigningFailed
	}

	return signedToken, nil
}
