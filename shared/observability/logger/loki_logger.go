// Package logger provides structured logging implementation optimized for Loki.
// It outputs JSON-formatted logs with consistent field structure for efficient
// querying and aggregation in log management systems.
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"shared/observability/types"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log message.
// Higher values indicate more severe messages.
type LogLevel int

// Log level constants ordered by severity (lowest to highest).
const (
	// DebugLevel is for detailed debugging information
	DebugLevel LogLevel = iota
	// InfoLevel is for general informational messages
	InfoLevel
	// WarnLevel is for warning messages that don't prevent operation
	WarnLevel
	// ErrorLevel is for error messages indicating failures
	ErrorLevel
)

// ParseLevel converts a string representation to a LogLevel.
// Unrecognized levels default to InfoLevel for safety.
//
// Valid levels:
//   - "debug": DebugLevel
//   - "info": InfoLevel
//   - "warn": WarnLevel
//   - "error": ErrorLevel
//
// Parameters:
//   - level: String representation of the log level
//
// Returns:
//   - The corresponding LogLevel, defaults to InfoLevel if unrecognized
func ParseLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// String returns the string representation of a LogLevel.
// This is used when serializing log entries to JSON.
//
// Returns:
//   - String representation of the log level ("debug", "info", "warn", "error", or "unknown")
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}

// LokiLogger implements the Logger interface with JSON output optimized for Loki.
// It provides structured logging with consistent field formatting, automatic
// context extraction, and thread-safe operations. Each log entry includes
// standard fields like timestamp, level, service name, and hostname.
type LokiLogger struct {
	// mu provides thread-safe access to logger fields
	mu               sync.RWMutex
	// output is where log entries are written (typically os.Stdout)
	output           io.Writer
	// serviceName identifies the service in log entries
	serviceName      string
	// environment specifies the deployment environment
	environment      string
	// hostname is automatically detected system hostname
	hostname         string
	// minLevel filters out messages below this severity
	minLevel         LogLevel
	// persistentFields are included in every log entry from this logger
	persistentFields types.Fields
}

// New creates a new LokiLogger instance with the specified configuration.
// The logger automatically detects the system hostname and includes it in all log entries.
// If output is nil, it defaults to os.Stdout.
//
// Parameters:
//   - serviceName: Name of the service for identification in logs
//   - environment: Deployment environment (e.g., "production", "staging")
//   - logLevel: Minimum log level to output ("debug", "info", "warn", "error")
//   - output: Where to write log entries (defaults to os.Stdout if nil)
//   - additionalFields: Fields to include in every log entry
//
// Returns:
//   - A configured LokiLogger instance
//
// Example:
//
//	logger := New(
//		"audit-processor",
//		"production",
//		"info",
//		os.Stdout,
//		types.Fields{"version": "1.0.0"},
//	)
func New(serviceName, environment, logLevel string, output io.Writer, additionalFields types.Fields) *LokiLogger {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	if output == nil {
		output = os.Stdout
	}

	return &LokiLogger{
		output:           output,
		serviceName:      serviceName,
		environment:      environment,
		hostname:         hostname,
		minLevel:         ParseLevel(logLevel),
		persistentFields: additionalFields,
	}
}

// Info logs an informational message at INFO level.
// The message is only logged if the logger's minimum level is INFO or lower.
// Context values like trace_id, request_id, and report_id are automatically extracted.
//
// Parameters:
//   - ctx: Context for extracting trace information
//   - msg: The log message
//   - fields: Additional structured fields for this log entry
func (l *LokiLogger) Info(ctx context.Context, msg string, fields types.Fields) {
	if l.minLevel > InfoLevel {
		return
	}
	l.log(ctx, InfoLevel, msg, nil, fields)
}

// Error logs an error message at ERROR level.
// The error object is included in the log entry with both the error message
// and error type for better debugging. Context values are automatically extracted.
//
// Parameters:
//   - ctx: Context for extracting trace information
//   - msg: Description of the error context
//   - err: The error object to log
//   - fields: Additional structured fields for this log entry
func (l *LokiLogger) Error(ctx context.Context, msg string, err error, fields types.Fields) {
	if l.minLevel > ErrorLevel {
		return
	}
	l.log(ctx, ErrorLevel, msg, err, fields)
}

// Warn logs a warning message at WARN level.
// The message is only logged if the logger's minimum level is WARN or lower.
// Context values are automatically extracted.
//
// Parameters:
//   - ctx: Context for extracting trace information
//   - msg: The warning message
//   - fields: Additional structured fields for this log entry
func (l *LokiLogger) Warn(ctx context.Context, msg string, fields types.Fields) {
	if l.minLevel > WarnLevel {
		return
	}
	l.log(ctx, WarnLevel, msg, nil, fields)
}

// Debug logs a debug message at DEBUG level.
// The message is only logged if the logger's minimum level is DEBUG.
// This level is typically disabled in production environments.
// Context values are automatically extracted.
//
// Parameters:
//   - ctx: Context for extracting trace information
//   - msg: The debug message
//   - fields: Additional structured fields for this log entry
func (l *LokiLogger) Debug(ctx context.Context, msg string, fields types.Fields) {
	if l.minLevel > DebugLevel {
		return
	}
	l.log(ctx, DebugLevel, msg, nil, fields)
}

// WithFields returns a new LokiLogger instance with additional persistent fields.
// The new logger inherits all configuration from the parent logger and adds
// the specified fields to every log entry. This is useful for adding consistent
// context like user IDs or request IDs to multiple related log entries.
//
// Parameters:
//   - fields: Fields to add to all log entries from the new logger
//
// Returns:
//   - A new Logger instance with the additional persistent fields
//
// Example:
//
//	requestLogger := logger.WithFields(types.Fields{
//		"request_id": "abc-123",
//		"user_id":    "user-456",
//	})
//	requestLogger.Info(ctx, "Processing request", nil)
func (l *LokiLogger) WithFields(fields types.Fields) types.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make(types.Fields)
	for k, v := range l.persistentFields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &LokiLogger{
		output:           l.output,
		serviceName:      l.serviceName,
		environment:      l.environment,
		hostname:         l.hostname,
		minLevel:         l.minLevel,
		persistentFields: newFields,
	}
}

// log is the internal logging method that formats and writes log entries.
// It combines standard fields, context values, persistent fields, and call-specific
// fields into a single JSON object. The method is thread-safe and handles
// context extraction for distributed tracing.
//
// Standard fields included:
//   - timestamp: RFC3339 nano format in UTC
//   - level: Log level as string
//   - service: Service name from configuration
//   - env: Environment from configuration
//   - hostname: System hostname
//   - message: The log message
//
// Context fields (if present):
//   - trace_id: Distributed trace identifier
//   - request_id: Request correlation identifier
//   - report_id: Audit report identifier
//
// Parameters:
//   - ctx: Context for extracting trace information
//   - level: Severity level of the message
//   - msg: The log message
//   - err: Optional error object (can be nil)
//   - fields: Additional fields for this log entry
func (l *LokiLogger) log(ctx context.Context, level LogLevel, msg string, err error, fields types.Fields) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entry := make(types.Fields)

	// Standard fields
	entry["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry["level"] = level.String()
	entry["service"] = l.serviceName
	entry["env"] = l.environment
	entry["hostname"] = l.hostname
	entry["message"] = msg

	// Extract context values
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		entry["trace_id"] = traceID
	}
	if requestID, ok := ctx.Value("request_id").(string); ok {
		entry["request_id"] = requestID
	}
	if reportID, ok := ctx.Value("report_id").(string); ok {
		entry["report_id"] = reportID
	}

	// Add error
	if err != nil {
		entry["error"] = err.Error()
		entry["error_type"] = fmt.Sprintf("%T", err)
	}

	// Add persistent fields
	for k, v := range l.persistentFields {
		entry[k] = v
	}

	// Add call-specific fields
	for k, v := range fields {
		entry[k] = v
	}

	// Output as JSON
	if jsonBytes, err := json.Marshal(entry); err == nil {
		l.output.Write(jsonBytes)
		l.output.Write([]byte("\n"))
	}
}
