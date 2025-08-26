package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", DebugLevel},
		{"info", InfoLevel},
		{"warn", WarnLevel},
		{"error", ErrorLevel},
		{"unknown", InfoLevel},
		{"", InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseLevel(tt.input))
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{LogLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	logger := New("test-service", "test", "info", &buf, map[string]interface{}{
		"version": "1.0.0",
	})

	assert.NotNil(t, logger)
	assert.Equal(t, "test-service", logger.serviceName)
	assert.Equal(t, "test", logger.environment)
	assert.Equal(t, InfoLevel, logger.minLevel)
}

func TestLokiLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logMethod func(*LokiLogger, context.Context)
		shouldLog bool
	}{
		{
			name:     "debug level logs debug",
			logLevel: "debug",
			logMethod: func(l *LokiLogger, ctx context.Context) {
				l.Debug(ctx, "test", nil)
			},
			shouldLog: true,
		},
		{
			name:     "info level skips debug",
			logLevel: "info",
			logMethod: func(l *LokiLogger, ctx context.Context) {
				l.Debug(ctx, "test", nil)
			},
			shouldLog: false,
		},
		{
			name:     "warn level logs warn",
			logLevel: "warn",
			logMethod: func(l *LokiLogger, ctx context.Context) {
				l.Warn(ctx, "test", nil)
			},
			shouldLog: true,
		},
		{
			name:     "error level logs error",
			logLevel: "error",
			logMethod: func(l *LokiLogger, ctx context.Context) {
				l.Error(ctx, "test", errors.New("error"), nil)
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New("test", "test", tt.logLevel, &buf, nil)

			tt.logMethod(logger, context.Background())

			if tt.shouldLog {
				assert.NotEmpty(t, buf.String())
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestLokiLogger_StructuredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New("test", "prod", "info", &buf, map[string]interface{}{
		"version": "1.0.0",
	})

	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	ctx = context.WithValue(ctx, "trace_id", "trace-456")
	ctx = context.WithValue(ctx, "report_id", "report-789")

	logger.Info(ctx, "Test message", map[string]interface{}{
		"user_id": "user-001",
		"action":  "download",
	})

	// Parse output
	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	// Verify standard fields
	assert.Equal(t, "info", entry["level"])
	assert.Equal(t, "test", entry["service"])
	assert.Equal(t, "prod", entry["env"])
	assert.Equal(t, "Test message", entry["message"])

	// Verify context fields
	assert.Equal(t, "req-123", entry["request_id"])
	assert.Equal(t, "trace-456", entry["trace_id"])
	assert.Equal(t, "report-789", entry["report_id"])

	// Verify additional fields
	assert.Equal(t, "1.0.0", entry["version"])
	assert.Equal(t, "user-001", entry["user_id"])
	assert.Equal(t, "download", entry["action"])

	// Verify metadata fields exist
	assert.NotEmpty(t, entry["timestamp"])
	assert.NotEmpty(t, entry["hostname"])
}

func TestLokiLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := New("test", "test", "error", &buf, nil)

	testErr := errors.New("something went wrong")
	logger.Error(context.Background(), "Operation failed", testErr, map[string]interface{}{
		"operation": "download",
	})

	// Parse output
	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "error", entry["level"])
	assert.Equal(t, "Operation failed", entry["message"])
	assert.Equal(t, "something went wrong", entry["error"])
	assert.Equal(t, "*errors.errorString", entry["error_type"])
	assert.Equal(t, "download", entry["operation"])
}

func TestLokiLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New("test", "test", "info", &buf, nil)

	// Create logger with persistent fields
	withFields := logger.WithFields(map[string]interface{}{
		"request_id": "req-123",
		"user_id":    "user-456",
	})

	// Type assertion to get the concrete type
	newLogger, ok := withFields.(*LokiLogger)
	require.True(t, ok)

	// Log with the new logger
	newLogger.Info(context.Background(), "Test", map[string]interface{}{
		"extra": "field",
	})

	// Parse output
	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "req-123", entry["request_id"])
	assert.Equal(t, "user-456", entry["user_id"])
	assert.Equal(t, "field", entry["extra"])
}
