package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	jwtSecretEnvKey          = "JWT_SECRET"
	accessTokenTTLEnvKey     = "ACCESS_TOKEN_TTL"
	refreshTokenTTLEnvKey    = "REFRESH_TOKEN_TTL"
	portEnvKey               = "PORT"
	bcryptCostEnvKey         = "BCRYPT_COST"
	environmentEnvKey        = "ENV"
	remoteDBURLEnvKey        = "REMOTE_DB_URL"
	remoteAPIURLEnvKey       = "REMOTE_API_URL"
	supabaseURLEnvKey        = "SUPABASE_URL"
	supabaseServiceKeyEnvKey = "SUPABASE_SERVICE_ROLE_KEY"
	corsAllowedOriginsEnvKey = "CORS_ALLOWED_ORIGINS"
	defaultAccessTokenTTL    = "15m"
	defaultRefreshTokenTTL   = "24h"
	defaultPort              = "8080"
	defaultBcryptCost        = "10"
)

var (
	ErrMissingEnvironmentVariable = errors.New("config: missing required environment variable")
	ErrInvalidBcryptCost          = errors.New("config: invalid bcrypt cost")
	ErrInvalidAccessTokenTTL      = errors.New("config: invalid access token ttl")
	ErrInvalidRefreshTokenTTL     = errors.New("config: invalid refresh token ttl")
)

type appConfig struct {
	jwtSecret          string
	accessTokenTTL     time.Duration
	refreshTokenTTL    time.Duration
	port               string
	bcryptCost         int
	environment        string
	remoteDatabaseURL  string
	remoteAPIURL       string
	supabaseURL        string
	supabaseServiceKey string
	corsAllowedOrigins []string
}

type getenvFunc func(string) string

func loadAppConfig(getenv getenvFunc) (appConfig, error) {
	environment := strings.ToLower(strings.TrimSpace(getenv(environmentEnvKey)))
	jwtSecret := strings.TrimSpace(getenv(jwtSecretEnvKey))
	if environment == "production" && jwtSecret == "" {
		return appConfig{}, fmt.Errorf("%w: %s", ErrMissingEnvironmentVariable, jwtSecretEnvKey)
	}

	accessTokenTTLValue := environmentValueOrDefault(getenv, accessTokenTTLEnvKey, defaultAccessTokenTTL)
	refreshTokenTTLValue := environmentValueOrDefault(getenv, refreshTokenTTLEnvKey, defaultRefreshTokenTTL)
	port := environmentValueOrDefault(getenv, portEnvKey, defaultPort)
	bcryptCostValue := environmentValueOrDefault(getenv, bcryptCostEnvKey, defaultBcryptCost)

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
		jwtSecret:          jwtSecret,
		accessTokenTTL:     accessTokenTTL,
		refreshTokenTTL:    refreshTokenTTL,
		port:               port,
		bcryptCost:         bcryptCost,
		environment:        environment,
		remoteDatabaseURL:  strings.TrimSpace(getenv(remoteDBURLEnvKey)),
		remoteAPIURL:       strings.TrimRight(strings.TrimSpace(getenv(remoteAPIURLEnvKey)), "/"),
		supabaseURL:        strings.TrimRight(strings.TrimSpace(getenv(supabaseURLEnvKey)), "/"),
		supabaseServiceKey: strings.TrimSpace(getenv(supabaseServiceKeyEnvKey)),
		corsAllowedOrigins: parseCSV(getenv(corsAllowedOriginsEnvKey)),
	}, nil
}

func (config appConfig) isProduction() bool {
	return config.environment == "production"
}

func requiredEnvironmentValue(getenv getenvFunc, key string) (string, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingEnvironmentVariable, key)
	}

	return value, nil
}

func environmentValueOrDefault(getenv getenvFunc, key, defaultValue string) string {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return defaultValue
	}

	return value
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

func parseCSV(rawValue string) []string {
	parts := strings.Split(rawValue, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
