package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRun_MissingConfiguration_ReturnsErrMissingEnvironmentVariable(t *testing.T) {
	// Arrange
	t.Setenv(dbURLEnvKey, "")
	t.Setenv(jwtSecretEnvKey, "")
	t.Setenv(accessTokenTTLEnvKey, "")
	t.Setenv(refreshTokenTTLEnvKey, "")
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
	t.Setenv(accessTokenTTLEnvKey, "5m")
	t.Setenv(refreshTokenTTLEnvKey, "24h")
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

func TestWithCORS_OptionsRequest_ReturnsOKAndHeaders(t *testing.T) {
	// Arrange
	nextWasCalled := false
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextWasCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodOptions, "/boards", nil)
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if nextWasCalled {
		t.Fatal("expected preflight request to stop before the next handler")
	}
	assertCORSHeaders(t, response)
}

func TestWithCORS_RegularRequest_AddsHeadersAndCallsNext(t *testing.T) {
	// Arrange
	nextWasCalled := false
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextWasCalled = true
		w.WriteHeader(http.StatusAccepted)
	}))
	request := httptest.NewRequest(http.MethodGet, "/boards", nil)
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, response.Code)
	}
	if !nextWasCalled {
		t.Fatal("expected regular request to reach the next handler")
	}
	assertCORSHeaders(t, response)
}

func assertCORSHeaders(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  corsAllowedOrigin,
		"Access-Control-Allow-Methods": corsAllowedMethods,
		"Access-Control-Allow-Headers": corsAllowedHeaders,
	}

	for header, expectedValue := range expectedHeaders {
		if got := response.Header().Get(header); got != expectedValue {
			t.Errorf("expected %s header %q, got %q", header, expectedValue, got)
		}
	}
}
