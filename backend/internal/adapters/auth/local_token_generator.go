package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

var errInvalidLocalTokenClaims = errors.New("auth: invalid local token claims")

type LocalTokenGenerator struct {
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	now             func() time.Time
}

type localTokenClaims struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func NewLocalTokenGenerator(accessTokenTTL, refreshTokenTTL time.Duration) (ports.TokenGenerator, error) {
	if accessTokenTTL <= 0 || refreshTokenTTL <= 0 {
		return nil, ErrInvalidTokenTTL
	}

	return &LocalTokenGenerator{
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		now:             time.Now,
	}, nil
}

func (generator *LocalTokenGenerator) GenerateTokenPair(subject ports.TokenSubject) (ports.TokenPair, error) {
	if strings.TrimSpace(subject.UserID) == "" {
		return ports.TokenPair{}, ErrEmptyTokenSubject
	}

	now := generator.now().UTC()
	accessTokenExpiresAt := now.Add(generator.accessTokenTTL)
	refreshTokenExpiresAt := now.Add(generator.refreshTokenTTL)

	accessToken, err := generator.buildAccessToken(subject, now, accessTokenExpiresAt)
	if err != nil {
		return ports.TokenPair{}, err
	}

	refreshToken, err := randomOpaqueToken(32)
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

func (generator *LocalTokenGenerator) ValidateToken(token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", ErrEmptyToken
	}

	claims, err := parseLocalTokenClaims(token)
	if err != nil {
		return "", ErrInvalidToken
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return "", ErrInvalidToken
	}
	if claims.ExpiresAt <= generator.now().UTC().Unix() {
		return "", ErrExpiredToken
	}

	return claims.Subject, nil
}

func (generator *LocalTokenGenerator) buildAccessToken(subject ports.TokenSubject, issuedAt, expiresAt time.Time) (string, error) {
	headerSegment, err := encodeLocalTokenSegment(map[string]string{
		"alg": "none",
		"typ": "JWT",
	})
	if err != nil {
		return "", ErrTokenSigningFailed
	}

	payloadSegment, err := encodeLocalTokenSegment(localTokenClaims{
		Subject:   subject.UserID,
		Email:     subject.Email,
		FirstName: subject.FirstName,
		LastName:  subject.LastName,
		Role:      "authenticated",
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		return "", ErrTokenSigningFailed
	}

	return headerSegment + "." + payloadSegment + ".", nil
}

func encodeLocalTokenSegment(value interface{}) (string, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(jsonBytes), nil
}

func parseLocalTokenClaims(token string) (localTokenClaims, error) {
	segments := strings.Split(token, ".")
	if len(segments) < 2 {
		return localTokenClaims{}, errInvalidLocalTokenClaims
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return localTokenClaims{}, errInvalidLocalTokenClaims
	}

	var claims localTokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return localTokenClaims{}, errInvalidLocalTokenClaims
	}

	return claims, nil
}

func randomOpaqueToken(size int) (string, error) {
	tokenBytes := make([]byte, size)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}
