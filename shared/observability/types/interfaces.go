// Package observability provides structured logging and metrics collection
// utilities for all workers in the audit system.
//
// The package follows clean architecture principles with minimal abstraction,
// focusing on essential features while maintaining testability and extensibility.
//
// Design Patterns:
//   - Provider Pattern: Manages instances and configuration
//   - Dependency Inversion: Core depends on interfaces, not implementations
package types

import (
	"context"
	"io"
)

// Logger defines the contract for structured logging.
// Implementations should provide JSON-formatted output suitable for log aggregation systems like Loki.
// All methods are context-aware to support request tracing and correlation.
type Logger interface {
	// Info logs an informational message.
	// Use for general operational information that doesn't require action.
	//
	// Parameters:
	//   - ctx: Context for request tracing and cancellation
	//   - msg: The log message describing the event
	//   - fields: Additional structured fields for context
	Info(ctx context.Context, msg string, fields Fields)

	// Error logs an error message with the associated error.
	// Use for errors that indicate failures in the application.
	//
	// Parameters:
	//   - ctx: Context for request tracing and cancellation
	//   - msg: The log message describing the error context
	//   - err: The error object to be logged
	//   - fields: Additional structured fields for context
	Error(ctx context.Context, msg string, err error, fields Fields)

	// Warn logs a warning message.
	// Use for potentially harmful situations that don't prevent operation.
	//
	// Parameters:
	//   - ctx: Context for request tracing and cancellation
	//   - msg: The log message describing the warning
	//   - fields: Additional structured fields for context
	Warn(ctx context.Context, msg string, fields Fields)

	// Debug logs a debug message.
	// Use for detailed information useful during development and troubleshooting.
	// These messages are typically filtered out in production.
	//
	// Parameters:
	//   - ctx: Context for request tracing and cancellation
	//   - msg: The log message with debugging information
	//   - fields: Additional structured fields for context
	Debug(ctx context.Context, msg string, fields Fields)

	// WithFields returns a new Logger instance with additional persistent fields.
	// The returned logger will include these fields in all subsequent log entries.
	// This is useful for adding consistent context like user IDs or request IDs.
	//
	// Parameters:
	//   - fields: Fields to be included in all log entries from the returned logger
	//
	// Returns:
	//   - A new Logger instance with the additional fields
	WithFields(fields Fields) Logger
}

// Metrics defines the contract for metrics collection.
// Implementations should provide Prometheus-compatible metrics for monitoring
// and alerting. All metrics should follow Prometheus naming conventions.
type Metrics interface {
	// RecordSuccess increments the success counter for a specific operation type.
	// Use this to track successful completions of operations.
	//
	// Parameters:
	//   - operationType: The type of operation that succeeded (e.g., "parse", "upload")
	RecordSuccess(operationType string)

	// RecordError increments the error counter for a specific operation and error type.
	// Use this to track failures and their categories for monitoring and alerting.
	//
	// Parameters:
	//   - operationType: The type of operation that failed (e.g., "parse", "upload")
	//   - errorType: The category of error (e.g., "timeout", "validation", "network")
	RecordError(operationType string, errorType string)

	// RecordDuration records the duration of an operation in seconds.
	// Use this to track performance metrics and identify bottlenecks.
	//
	// Parameters:
	//   - operation: The name of the operation being measured
	//   - duration: The duration in seconds (use time.Since(start).Seconds())
	RecordDuration(operation string, duration float64)

	// RecordFileSize records the size of processed files in bytes.
	// Use this to track data volume and identify large file processing.
	//
	// Parameters:
	//   - fileType: The type of file being processed (e.g., "json", "pdf", "markdown")
	//   - bytes: The size of the file in bytes
	RecordFileSize(fileType string, bytes int64)

	// StartOperation increments the in-progress gauge for an operation.
	// Use this to track concurrent operations and identify potential bottlenecks.
	// Must be paired with EndOperation to maintain accurate counts.
	//
	// Parameters:
	//   - operation: The name of the operation starting
	StartOperation(operation string)

	// EndOperation decrements the in-progress gauge for an operation.
	// Use this to mark the completion of an operation started with StartOperation.
	// Should be called in a defer statement to ensure it runs even on errors.
	//
	// Parameters:
	//   - operation: The name of the operation ending
	EndOperation(operation string)
}

// Fields represents structured logging fields as key-value pairs.
// Values can be any type that is JSON-serializable.
// Common fields include "user_id", "request_id", "duration", "error_type".
//
// Example:
//
//	fields := Fields{
//		"user_id":    "123",
//		"request_id": "abc-def",
//		"duration":   1.23,
//	}
type Fields map[string]interface{}

// Config holds observability configuration for the provider.
// It defines how logging and metrics should be configured across all components.
type Config struct {
	// ServiceName identifies the service in logs and metrics.
	// This should be a unique identifier for your service (e.g., "audit-aggregator").
	ServiceName string

	// Environment specifies the deployment environment.
	// Common values: "development", "staging", "production".
	// This helps filter and analyze logs/metrics by environment.
	Environment string

	// LogLevel sets the minimum log level to output.
	// Valid values: "debug", "info", "warn", "error".
	// Messages below this level will be filtered out.
	LogLevel string

	// LogOutput specifies where logs should be written.
	// Common values: os.Stdout, os.Stderr, or a file writer.
	// If nil, defaults to os.Stdout.
	LogOutput io.Writer

	// AdditionalFields are fields included in every log entry.
	// Use for global context like version, region, or deployment ID.
	// These fields are automatically added to all logs from this provider.
	AdditionalFields Fields
}

// Provider manages the lifecycle of observability components.
// It acts as a factory for Logger and Metrics instances, ensuring proper
// configuration and resource management. Each component gets its own
// Logger and Metrics instances with appropriate labels and context.
type Provider interface {
	// Logger returns a Logger instance for the specified component.
	// The logger is configured with component-specific fields and the provider's settings.
	// Multiple calls with the same component name return the same logger instance.
	//
	// Parameters:
	//   - component: Name of the component requesting the logger (e.g., "processor", "uploader")
	//
	// Returns:
	//   - A configured Logger instance for the component
	Logger(component string) Logger

	// Metrics returns a Metrics instance for the specified component.
	// The metrics collector is configured with component-specific labels.
	// Multiple calls with the same component name return the same metrics instance.
	//
	// Parameters:
	//   - component: Name of the component requesting metrics (e.g., "processor", "uploader")
	//
	// Returns:
	//   - A configured Metrics instance for the component
	Metrics(component string) Metrics

	// Close shuts down the provider and releases all resources.
	// This should be called when the application terminates to ensure
	// proper cleanup of file handles and other resources.
	//
	// Returns:
	//   - An error if cleanup fails, nil on success
	Close() error
}
