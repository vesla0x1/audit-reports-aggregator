package observability

import (
	"shared/config"
)

// Logger defines the interface for structured logging in the application.
// It provides context-aware logging with support for structured fields.
type Logger interface {
	// Info logs informational messages for normal operations.
	// Use for tracking successful operations, state changes, and general flow.
	Info(msg string, fields ...interface{})

	// Error logs error conditions with the associated error object.
	// Always pass the actual error; the implementation will extract details.
	Error(msg string, fields ...interface{})

	// WithFields returns a new Logger with the given fields added to all subsequent logs.
	// Useful for adding consistent context like request_id or component name.
	WithFields(fields map[string]interface{}) Logger
}

// Metrics defines the interface for recording application metrics.
type Metrics interface {
	// IncrementCounter increments a counter metric by 1.
	// Use for counting discrete events: requests, errors, completions.
	IncrementCounter(name string, tags map[string]string)

	// RecordHistogram records a value in a histogram distribution.
	// Use for latencies, sizes, or any value where distribution matters.
	RecordHistogram(name string, value float64, tags map[string]string)

	// RecordGauge records a point-in-time measurement.
	// Use sparingly in serverless; containers are ephemeral.
	RecordGauge(name string, value float64, tags map[string]string)
}

// Factory interface - defined in domain
type ObservabilityFactory interface {
	CreateObservability(cfg *config.Config) (Logger, Metrics, error)
}

// Provider interface for those who need it
type ObservabilityProvider interface {
	GetLogger() (Logger, error)
	GetMetrics() (Metrics, error)
	GetObservability() (Logger, Metrics, error)
	//Close() error NOT IMPLEMENTED
	MustGetLogger() Logger
	MustGetMetrics() Metrics
	IsInitialized() bool
}
