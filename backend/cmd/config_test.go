package main

import (
	"errors"
	"testing"
	"time"
)

func TestLoadAppConfig_ValidEnvironment_ReturnsConfig(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{
		jwtSecretEnvKey:       "local-secret",
		accessTokenTTLEnvKey:  "5m",
		refreshTokenTTLEnvKey: "24h",
		portEnvKey:            "8080",
		bcryptCostEnvKey:      "10",
	})

	// Act
	config, err := loadAppConfig(getenv)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if config.accessTokenTTL != 5*time.Minute {
		t.Errorf("expected access token ttl %v, got %v", 5*time.Minute, config.accessTokenTTL)
	}
	if config.refreshTokenTTL != 24*time.Hour {
		t.Errorf("expected refresh token ttl %v, got %v", 24*time.Hour, config.refreshTokenTTL)
	}
	if config.bcryptCost != 10 {
		t.Errorf("expected bcrypt cost 10, got %d", config.bcryptCost)
	}
}

func TestLoadAppConfig_InvalidBcryptCost_ReturnsErrInvalidBcryptCost(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{
		jwtSecretEnvKey:       "local-secret",
		accessTokenTTLEnvKey:  "5m",
		refreshTokenTTLEnvKey: "24h",
		portEnvKey:            "8080",
		bcryptCostEnvKey:      "invalid",
	})

	// Act
	_, err := loadAppConfig(getenv)

	// Assert
	if !errors.Is(err, ErrInvalidBcryptCost) {
		t.Errorf("expected error %v, got %v", ErrInvalidBcryptCost, err)
	}
}

func TestLoadAppConfig_InvalidAccessTokenTTL_ReturnsErrInvalidAccessTokenTTL(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{
		jwtSecretEnvKey:       "local-secret",
		accessTokenTTLEnvKey:  "0s",
		refreshTokenTTLEnvKey: "24h",
		portEnvKey:            "8080",
		bcryptCostEnvKey:      "10",
	})

	// Act
	_, err := loadAppConfig(getenv)

	// Assert
	if !errors.Is(err, ErrInvalidAccessTokenTTL) {
		t.Errorf("expected error %v, got %v", ErrInvalidAccessTokenTTL, err)
	}
}

func TestLoadAppConfig_InvalidRefreshTokenTTL_ReturnsErrInvalidRefreshTokenTTL(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{
		jwtSecretEnvKey:       "local-secret",
		accessTokenTTLEnvKey:  "5m",
		refreshTokenTTLEnvKey: "0s",
		portEnvKey:            "8080",
		bcryptCostEnvKey:      "10",
	})

	// Act
	_, err := loadAppConfig(getenv)

	// Assert
	if !errors.Is(err, ErrInvalidRefreshTokenTTL) {
		t.Errorf("expected error %v, got %v", ErrInvalidRefreshTokenTTL, err)
	}
}

func TestRequiredEnvironmentValue_BlankValue_ReturnsErrMissingEnvironmentVariable(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{jwtSecretEnvKey: "   "})

	// Act
	_, err := requiredEnvironmentValue(getenv, jwtSecretEnvKey)

	// Assert
	if !errors.Is(err, ErrMissingEnvironmentVariable) {
		t.Errorf("expected error %v, got %v", ErrMissingEnvironmentVariable, err)
	}
}

func mapGetenv(values map[string]string) getenvFunc {
	return func(key string) string {
		return values[key]
	}
}
