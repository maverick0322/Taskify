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
	ErrEmptyToken         = errors.New("auth: token cannot be empty")
	ErrInvalidToken       = errors.New("auth: invalid token")
	ErrExpiredToken       = errors.New("auth: expired token")
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

type taskifyClaims struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	jwt.RegisteredClaims
}

func (generator *JWTTokenGenerator) GenerateTokenPair(subject ports.TokenSubject) (ports.TokenPair, error) {
	if subject.UserID == "" {
		return ports.TokenPair{}, ErrEmptyTokenSubject
	}

	now := time.Now().UTC()
	accessTokenExpiresAt := now.Add(generator.accessTokenTTL)
	refreshTokenExpiresAt := now.Add(generator.refreshTokenTTL)

	accessToken, err := generator.signToken(subject, now, accessTokenExpiresAt)
	if err != nil {
		return ports.TokenPair{}, err
	}

	refreshToken, err := generator.signToken(subject, now, refreshTokenExpiresAt)
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

func (generator *JWTTokenGenerator) ValidateToken(token string) (string, error) {
	if token == "" {
		return "", ErrEmptyToken
	}

	claims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(parsedToken *jwt.Token) (interface{}, error) {
		if parsedToken.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}

		return generator.secretKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}

		return "", ErrInvalidToken
	}
	if parsedToken == nil || !parsedToken.Valid {
		return "", ErrInvalidToken
	}
	if claims.Subject == "" {
		return "", ErrInvalidToken
	}

	return claims.Subject, nil
}

func (generator *JWTTokenGenerator) signToken(subject ports.TokenSubject, issuedAt, expiresAt time.Time) (string, error) {
	claims := taskifyClaims{
		Email:     subject.Email,
		FirstName: subject.FirstName,
		LastName:  subject.LastName,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject.UserID,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(generator.secretKey)
	if err != nil {
		return "", ErrTokenSigningFailed
	}

	return signedToken, nil
}
