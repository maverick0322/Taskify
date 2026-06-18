package main

import (
	"errors"
	"os"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRun_MissingConfiguration_ReturnsErrMissingEnvironmentVariable(t *testing.T) {
	// Arrange
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

func TestLocalSQLiteDatabasePath_UsesUserConfigDirTaskifyDatabase(t *testing.T) {
	// Arrange
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Act
	databasePath, err := localSQLiteDatabasePath()

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	expectedSuffix := sqliteAppFolderName + string(filepath.Separator) + sqliteDatabaseName
	if !strings.HasSuffix(databasePath, expectedSuffix) {
		t.Fatalf("expected path to end with %q, got %q", expectedSuffix, databasePath)
	}
}

func TestSQLiteDSN_FormatsWindowsAbsolutePathAsFileURI(t *testing.T) {
	// Arrange
	databasePath := `C:\Users\meler\AppData\Roaming\Taskify\taskify.db`

	// Act
	dsn := sqliteDSN(databasePath)

	// Assert
	if !strings.HasPrefix(dsn, "file:///C:/Users/meler/AppData/Roaming/Taskify/taskify.db?") {
		t.Fatalf("expected Windows absolute path to use a file URI, got %q", dsn)
	}
	for _, expectedPragma := range []string{
		"_pragma=foreign_keys%281%29",
		"_pragma=journal_mode%28WAL%29",
		"_pragma=busy_timeout%285000%29",
	} {
		if !strings.Contains(dsn, expectedPragma) {
			t.Errorf("expected DSN to contain %q, got %q", expectedPragma, dsn)
		}
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

func TestLoadLocalEnvironment_MissingFilesReturnsNil(t *testing.T) {
	// Arrange
	changeWorkingDirectory(t, t.TempDir())

	// Act
	err := loadLocalEnvironment()

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestLoadLocalEnvironment_LoadsBackendRelativeEnv(t *testing.T) {
	// Arrange
	root := t.TempDir()
	workingDirectory := filepath.Join(root, "frontend", "src-tauri")
	backendDirectory := filepath.Join(root, "backend")
	if err := os.MkdirAll(workingDirectory, 0755); err != nil {
		t.Fatalf("failed to create working directory: %v", err)
	}
	if err := os.MkdirAll(backendDirectory, 0755); err != nil {
		t.Fatalf("failed to create backend directory: %v", err)
	}
	envPath := filepath.Join(backendDirectory, ".env")
	if err := os.WriteFile(envPath, []byte("TASKIFY_TEST_SIDE_CAR_ENV=loaded\n"), 0644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}
	t.Setenv("TASKIFY_TEST_SIDE_CAR_ENV", "")
	changeWorkingDirectory(t, workingDirectory)

	// Act
	err := loadLocalEnvironment()

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := os.Getenv("TASKIFY_TEST_SIDE_CAR_ENV"); got != "loaded" {
		t.Fatalf("expected env to be loaded, got %q", got)
	}
}

func TestWithCORS_OptionsRequest_ReturnsOKAndHeaders(t *testing.T) {
	// Arrange
	nextWasCalled := false
	handler := withCORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextWasCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodOptions, "/boards", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if nextWasCalled {
		t.Fatal("expected preflight request to stop before the next handler")
	}
	assertCORSHeaders(t, response, "http://localhost:5173")
}

func TestWithCORS_RegularRequest_AddsHeadersAndCallsNext(t *testing.T) {
	// Arrange
	nextWasCalled := false
	handler := withCORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextWasCalled = true
		w.WriteHeader(http.StatusAccepted)
	}))
	request := httptest.NewRequest(http.MethodGet, "/boards", nil)
	request.Header.Set("Origin", "https://taskify-preview.vercel.app")
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
	assertCORSHeaders(t, response, "https://taskify-preview.vercel.app")
}

func TestWithCORS_DisallowedPreflight_ReturnsForbidden(t *testing.T) {
	// Arrange
	handler := withCORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	request := httptest.NewRequest(http.MethodOptions, "/boards", nil)
	request.Header.Set("Origin", "https://example.com")
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow origin header, got %q", got)
	}
}

func assertCORSHeaders(t *testing.T, response *httptest.ResponseRecorder, expectedOrigin string) {
	t.Helper()

	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  expectedOrigin,
		"Access-Control-Allow-Methods": corsAllowedMethods,
		"Access-Control-Allow-Headers": corsAllowedHeaders,
	}

	for header, expectedValue := range expectedHeaders {
		if got := response.Header().Get(header); got != expectedValue {
			t.Errorf("expected %s header %q, got %q", header, expectedValue, got)
		}
	}
}

func changeWorkingDirectory(t *testing.T, directory string) {
	t.Helper()

	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to resolve current directory: %v", err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDirectory); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}
