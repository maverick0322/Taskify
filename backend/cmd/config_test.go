package main

import (
	"errors"
	"testing"
	"time"
)

func TestLoadAppConfig_ValidEnvironment_ReturnsConfig(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{
		dbURLEnvKey:      "postgres://taskify:taskify@localhost:5432/taskify?sslmode=disable",
		jwtSecretEnvKey:  "local-secret",
		jwtTTLEnvKey:     "24h",
		portEnvKey:       "8080",
		bcryptCostEnvKey: "10",
	})

	// Act
	config, err := loadAppConfig(getenv)

	// Assert
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if config.databaseURL == "" {
		t.Fatal("expected database URL")
	}
	if config.jwtTTL != 24*time.Hour {
		t.Errorf("expected jwt ttl %v, got %v", 24*time.Hour, config.jwtTTL)
	}
	if config.bcryptCost != 10 {
		t.Errorf("expected bcrypt cost 10, got %d", config.bcryptCost)
	}
}

func TestLoadAppConfig_MissingDatabaseURL_ReturnsErrMissingEnvironmentVariable(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{
		jwtSecretEnvKey:  "local-secret",
		jwtTTLEnvKey:     "24h",
		portEnvKey:       "8080",
		bcryptCostEnvKey: "10",
	})

	// Act
	_, err := loadAppConfig(getenv)

	// Assert
	if !errors.Is(err, ErrMissingEnvironmentVariable) {
		t.Errorf("expected error %v, got %v", ErrMissingEnvironmentVariable, err)
	}
}

func TestLoadAppConfig_InvalidBcryptCost_ReturnsErrInvalidBcryptCost(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{
		dbURLEnvKey:      "postgres://taskify:taskify@localhost:5432/taskify?sslmode=disable",
		jwtSecretEnvKey:  "local-secret",
		jwtTTLEnvKey:     "24h",
		portEnvKey:       "8080",
		bcryptCostEnvKey: "invalid",
	})

	// Act
	_, err := loadAppConfig(getenv)

	// Assert
	if !errors.Is(err, ErrInvalidBcryptCost) {
		t.Errorf("expected error %v, got %v", ErrInvalidBcryptCost, err)
	}
}

func TestLoadAppConfig_InvalidJWTTTL_ReturnsErrInvalidJWTTTL(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{
		dbURLEnvKey:      "postgres://taskify:taskify@localhost:5432/taskify?sslmode=disable",
		jwtSecretEnvKey:  "local-secret",
		jwtTTLEnvKey:     "0s",
		portEnvKey:       "8080",
		bcryptCostEnvKey: "10",
	})

	// Act
	_, err := loadAppConfig(getenv)

	// Assert
	if !errors.Is(err, ErrInvalidJWTTTL) {
		t.Errorf("expected error %v, got %v", ErrInvalidJWTTTL, err)
	}
}

func TestRequiredEnvironmentValue_BlankValue_ReturnsErrMissingEnvironmentVariable(t *testing.T) {
	// Arrange
	getenv := mapGetenv(map[string]string{dbURLEnvKey: "   "})

	// Act
	_, err := requiredEnvironmentValue(getenv, dbURLEnvKey)

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
