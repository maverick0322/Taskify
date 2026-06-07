package logging

import (
	"log/slog"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

// SlogLogger keeps the core logging port independent from the standard logger implementation.
type SlogLogger struct {
	logger *slog.Logger
}

func NewSlogLogger(logger *slog.Logger) ports.Logger {
	return &SlogLogger{logger: logger}
}

func (logger *SlogLogger) Info(msg string, keysAndValues ...interface{}) {
	logger.logger.Info(msg, normalizeKeysAndValues(keysAndValues)...)
}

func (logger *SlogLogger) Warn(msg string, keysAndValues ...interface{}) {
	logger.logger.Warn(msg, normalizeKeysAndValues(keysAndValues)...)
}

func (logger *SlogLogger) Error(msg string, keysAndValues ...interface{}) {
	logger.logger.Error(msg, normalizeKeysAndValues(keysAndValues)...)
}

func normalizeKeysAndValues(keysAndValues []interface{}) []interface{} {
	if len(keysAndValues)%2 == 0 {
		return keysAndValues
	}

	// slog expects paired attributes; preserving the orphan value makes malformed calls observable.
	return append(keysAndValues, "missing_value")
}
