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
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTTokenGenerator injects secret and lifetime so security settings stay outside code.
func NewJWTTokenGenerator(secretKey string, accessTokenTTL, refreshTokenTTL time.Duration) (ports.TokenGenerator, error) {
	if secretKey == "" {
		return nil, ErrEmptyJWTSecret
	}
	if accessTokenTTL <= 0 || refreshTokenTTL <= 0 {
		return nil, ErrInvalidTokenTTL
	}

	return &JWTTokenGenerator{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}, nil
}

func (generator *JWTTokenGenerator) GenerateTokenPair(userID string) (ports.TokenPair, error) {
	if userID == "" {
		return ports.TokenPair{}, ErrEmptyTokenSubject
	}

	now := time.Now().UTC()
	accessTokenExpiresAt := now.Add(generator.accessTokenTTL)
	refreshTokenExpiresAt := now.Add(generator.refreshTokenTTL)

	accessToken, err := generator.signToken(userID, now, accessTokenExpiresAt)
	if err != nil {
		return ports.TokenPair{}, err
	}

	refreshToken, err := generator.signToken(userID, now, refreshTokenExpiresAt)
	if err != nil {
		return ports.TokenPair{}, err
	}

	return ports.TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
	}, nil
}

func (generator *JWTTokenGenerator) signToken(userID string, issuedAt, expiresAt time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(generator.secretKey)
	if err != nil {
		return "", ErrTokenSigningFailed
	}

	return signedToken, nil
}
