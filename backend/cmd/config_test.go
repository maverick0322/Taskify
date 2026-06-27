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
		remoteDBURLEnvKey:     "postgres://remote.example/taskify",
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
	if config.remoteDatabaseURL != "postgres://remote.example/taskify" {
		t.Errorf("expected remote database URL to be preserved, got %q", config.remoteDatabaseURL)
	}
	if config.remoteAPIURL != "" {
		t.Errorf("expected empty remote api url by default, got %q", config.remoteAPIURL)
	}
}

func TestLoadAppConfig_LocalEnvironmentWithoutJWTSecret_ReturnsConfig(t *testing.T) {
	getenv := mapGetenv(map[string]string{
		environmentEnvKey:        "development",
		accessTokenTTLEnvKey:     "5m",
		refreshTokenTTLEnvKey:    "24h",
		portEnvKey:               "8080",
		bcryptCostEnvKey:         "10",
		remoteAPIURLEnvKey:       "https://taskify-api.example.com",
		jwtSecretEnvKey:          "",
		supabaseServiceKeyEnvKey: "",
	})

	config, err := loadAppConfig(getenv)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if config.jwtSecret != "" {
		t.Fatalf("expected empty jwt secret in local mode, got %q", config.jwtSecret)
	}
	if config.remoteAPIURL != "https://taskify-api.example.com" {
		t.Fatalf("expected remote api url to be preserved, got %q", config.remoteAPIURL)
	}
}

func TestLoadAppConfig_LocalEnvironmentUsesDefaultsWhenValuesAreMissing(t *testing.T) {
	getenv := mapGetenv(map[string]string{
		environmentEnvKey:  "development",
		remoteAPIURLEnvKey: "https://taskify-api.example.com",
		jwtSecretEnvKey:    "",
	})

	config, err := loadAppConfig(getenv)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if config.accessTokenTTL != 15*time.Minute {
		t.Fatalf("expected default access token ttl 15m, got %v", config.accessTokenTTL)
	}
	if config.refreshTokenTTL != 24*time.Hour {
		t.Fatalf("expected default refresh token ttl 24h, got %v", config.refreshTokenTTL)
	}
	if config.port != "8080" {
		t.Fatalf("expected default port 8080, got %q", config.port)
	}
	if config.bcryptCost != 10 {
		t.Fatalf("expected default bcrypt cost 10, got %d", config.bcryptCost)
	}
}

func TestLoadAppConfig_ProductionWithoutJWTSecret_ReturnsErrMissingEnvironmentVariable(t *testing.T) {
	getenv := mapGetenv(map[string]string{
		environmentEnvKey:     "production",
		accessTokenTTLEnvKey:  "5m",
		refreshTokenTTLEnvKey: "24h",
		portEnvKey:            "8080",
		bcryptCostEnvKey:      "10",
		jwtSecretEnvKey:       "",
	})

	_, err := loadAppConfig(getenv)

	if !errors.Is(err, ErrMissingEnvironmentVariable) {
		t.Fatalf("expected %v, got %v", ErrMissingEnvironmentVariable, err)
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

func TestEnvironmentValueOrDefault_BlankValueReturnsDefault(t *testing.T) {
	getenv := mapGetenv(map[string]string{
		accessTokenTTLEnvKey: "   ",
	})

	value := environmentValueOrDefault(getenv, accessTokenTTLEnvKey, defaultAccessTokenTTL)

	if value != defaultAccessTokenTTL {
		t.Fatalf("expected default value %q, got %q", defaultAccessTokenTTL, value)
	}
}

func mapGetenv(values map[string]string) getenvFunc {
	return func(key string) string {
		return values[key]
	}
}
