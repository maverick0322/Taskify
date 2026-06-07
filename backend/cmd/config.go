package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	dbURLEnvKey      = "DB_URL"
	jwtSecretEnvKey  = "JWT_SECRET"
	jwtTTLEnvKey     = "JWT_TTL"
	portEnvKey       = "PORT"
	bcryptCostEnvKey = "BCRYPT_COST"
)

var (
	ErrMissingEnvironmentVariable = errors.New("config: missing required environment variable")
	ErrInvalidBcryptCost          = errors.New("config: invalid bcrypt cost")
	ErrInvalidJWTTTL              = errors.New("config: invalid jwt ttl")
)

type appConfig struct {
	databaseURL string
	jwtSecret   string
	jwtTTL      time.Duration
	port        string
	bcryptCost  int
}

type getenvFunc func(string) string

func loadAppConfig(getenv getenvFunc) (appConfig, error) {
	databaseURL, err := requiredEnvironmentValue(getenv, dbURLEnvKey)
	if err != nil {
		return appConfig{}, err
	}

	jwtSecret, err := requiredEnvironmentValue(getenv, jwtSecretEnvKey)
	if err != nil {
		return appConfig{}, err
	}

	jwtTTLValue, err := requiredEnvironmentValue(getenv, jwtTTLEnvKey)
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

	jwtTTL, err := parseJWTTTL(jwtTTLValue)
	if err != nil {
		return appConfig{}, err
	}

	bcryptCost, err := parseBcryptCost(bcryptCostValue)
	if err != nil {
		return appConfig{}, err
	}

	return appConfig{
		databaseURL: databaseURL,
		jwtSecret:   jwtSecret,
		jwtTTL:      jwtTTL,
		port:        port,
		bcryptCost:  bcryptCost,
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

func parseJWTTTL(rawValue string) (time.Duration, error) {
	jwtTTL, err := time.ParseDuration(strings.TrimSpace(rawValue))
	if err != nil || jwtTTL <= 0 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidJWTTTL, rawValue)
	}

	return jwtTTL, nil
}
