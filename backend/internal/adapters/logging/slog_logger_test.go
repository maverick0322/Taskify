package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSlogLogger_InfoWritesMessage(t *testing.T) {
	// Arrange
	var logBuffer bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	logger := NewSlogLogger(baseLogger)

	// Act
	logger.Info("application started", "port", "8080")

	// Assert
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "application started") {
		t.Errorf("expected log output to contain message")
	}
	if !strings.Contains(logOutput, "8080") {
		t.Errorf("expected log output to contain structured value")
	}
}

func TestSlogLogger_WarnWritesMessage(t *testing.T) {
	// Arrange
	var logBuffer bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	logger := NewSlogLogger(baseLogger)

	// Act
	logger.Warn("client request rejected", "status", "bad_request")

	// Assert
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "client request rejected") {
		t.Errorf("expected log output to contain message")
	}
	if !strings.Contains(logOutput, "bad_request") {
		t.Errorf("expected log output to contain structured value")
	}
}

func TestSlogLogger_ErrorWritesMessage(t *testing.T) {
	// Arrange
	var logBuffer bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	logger := NewSlogLogger(baseLogger)

	// Act
	logger.Error("database operation failed", "component", "repository")

	// Assert
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "database operation failed") {
		t.Errorf("expected log output to contain message")
	}
	if !strings.Contains(logOutput, "repository") {
		t.Errorf("expected log output to contain structured value")
	}
}

func TestNormalizeKeysAndValues_EvenArguments_ReturnsOriginalArguments(t *testing.T) {
	// Arrange
	keysAndValues := []interface{}{"port", "8080"}

	// Act
	normalizedKeysAndValues := normalizeKeysAndValues(keysAndValues)

	// Assert
	if len(normalizedKeysAndValues) != len(keysAndValues) {
		t.Fatalf("expected length %d, got %d", len(keysAndValues), len(normalizedKeysAndValues))
	}
	if normalizedKeysAndValues[1] != "8080" {
		t.Errorf("expected value 8080, got %v", normalizedKeysAndValues[1])
	}
}

func TestNormalizeKeysAndValues_OddArguments_AppendsMissingValue(t *testing.T) {
	// Arrange
	keysAndValues := []interface{}{"port"}

	// Act
	normalizedKeysAndValues := normalizeKeysAndValues(keysAndValues)

	// Assert
	if len(normalizedKeysAndValues) != 2 {
		t.Fatalf("expected two values, got %d", len(normalizedKeysAndValues))
	}
	if normalizedKeysAndValues[1] != "missing_value" {
		t.Errorf("expected missing value marker, got %v", normalizedKeysAndValues[1])
	}
}
