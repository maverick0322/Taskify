package main

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestRun_MissingConfiguration_ReturnsErrMissingEnvironmentVariable(t *testing.T) {
	// Arrange
	t.Setenv(dbURLEnvKey, "")
	t.Setenv(jwtSecretEnvKey, "")
	t.Setenv(jwtTTLEnvKey, "")
	t.Setenv(portEnvKey, "")
	t.Setenv(bcryptCostEnvKey, "")

	// Act
	err := run()

	// Assert
	if !errors.Is(err, ErrMissingEnvironmentVariable) {
		t.Errorf("expected error %v, got %v", ErrMissingEnvironmentVariable, err)
	}
}

func TestRun_InvalidDatabaseURL_ReturnsError(t *testing.T) {
	// Arrange
	t.Setenv(dbURLEnvKey, "://invalid")
	t.Setenv(jwtSecretEnvKey, "local-secret")
	t.Setenv(jwtTTLEnvKey, "24h")
	t.Setenv(portEnvKey, "8080")
	t.Setenv(bcryptCostEnvKey, "10")

	// Act
	err := run()

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStartHTTPServer_InvalidAddress_SendsServerError(t *testing.T) {
	// Arrange
	server := &http.Server{Addr: "invalid-address"}
	serverErrors := make(chan error, 1)

	// Act
	go startHTTPServer(server, serverErrors)

	// Assert
	select {
	case err := <-serverErrors:
		if err == nil {
			t.Fatal("expected server error, got nil")
		}
	case <-time.After(time.Second):
		t.Fatal("expected server error before timeout")
	}
}
