package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	jwtSecretEnvKey       = "JWT_SECRET"
	accessTokenTTLEnvKey  = "ACCESS_TOKEN_TTL"
	refreshTokenTTLEnvKey = "REFRESH_TOKEN_TTL"
	portEnvKey            = "PORT"
	bcryptCostEnvKey      = "BCRYPT_COST"
	remoteDBURLEnvKey     = "REMOTE_DB_URL"
)

var (
	ErrMissingEnvironmentVariable = errors.New("config: missing required environment variable")
	ErrInvalidBcryptCost          = errors.New("config: invalid bcrypt cost")
	ErrInvalidAccessTokenTTL      = errors.New("config: invalid access token ttl")
	ErrInvalidRefreshTokenTTL     = errors.New("config: invalid refresh token ttl")
)

type appConfig struct {
	jwtSecret         string
	accessTokenTTL    time.Duration
	refreshTokenTTL   time.Duration
	port              string
	bcryptCost        int
	remoteDatabaseURL string
}

type getenvFunc func(string) string

func loadAppConfig(getenv getenvFunc) (appConfig, error) {
	jwtSecret, err := requiredEnvironmentValue(getenv, jwtSecretEnvKey)
	if err != nil {
		return appConfig{}, err
	}

	accessTokenTTLValue, err := requiredEnvironmentValue(getenv, accessTokenTTLEnvKey)
	if err != nil {
		return appConfig{}, err
	}

	refreshTokenTTLValue, err := requiredEnvironmentValue(getenv, refreshTokenTTLEnvKey)
	if err != nil {
		return appConfig{}, err
	}

	port, err := requiredEnvironmentValue(getenv, portEnvKey)
	if err != nil {
		return appConfig{}, err
	}

	bcryptCostValue, err := requiredEnvironmentValue(getenv, bcryptCostEnvKey)
	if err != nil {
		return appConfig{}, err
	}

	accessTokenTTL, err := parsePositiveDuration(accessTokenTTLValue, ErrInvalidAccessTokenTTL)
	if err != nil {
		return appConfig{}, err
	}

	refreshTokenTTL, err := parsePositiveDuration(refreshTokenTTLValue, ErrInvalidRefreshTokenTTL)
	if err != nil {
		return appConfig{}, err
	}

	bcryptCost, err := parseBcryptCost(bcryptCostValue)
	if err != nil {
		return appConfig{}, err
	}

	return appConfig{
		jwtSecret:         jwtSecret,
		accessTokenTTL:    accessTokenTTL,
		refreshTokenTTL:   refreshTokenTTL,
		port:              port,
		bcryptCost:        bcryptCost,
		remoteDatabaseURL: strings.TrimSpace(getenv(remoteDBURLEnvKey)),
	}, nil
}

func requiredEnvironmentValue(getenv getenvFunc, key string) (string, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingEnvironmentVariable, key)
	}

	return value, nil
}

func parseBcryptCost(rawValue string) (int, error) {
	bcryptCost, err := strconv.Atoi(strings.TrimSpace(rawValue))
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidBcryptCost, rawValue)
	}

	return bcryptCost, nil
}

func parsePositiveDuration(rawValue string, sentinelError error) (time.Duration, error) {
	parsedDuration, err := time.ParseDuration(strings.TrimSpace(rawValue))
	if err != nil || parsedDuration <= 0 {
		return 0, fmt.Errorf("%w: %s", sentinelError, rawValue)
	}

	return parsedDuration, nil
}
