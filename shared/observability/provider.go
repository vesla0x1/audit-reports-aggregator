// Package observability provides a centralized provider for logging and metrics
// components used throughout the audit reports aggregator system.
package observability

import (
	"fmt"
	"io"
	"os"
	"sync"

	"shared/observability/logger"
	"shared/observability/metrics"
	"shared/observability/types"
)

// Logger is a type alias for the Logger interface from the types package.
// It provides structured logging capabilities with context support.
type Logger = types.Logger

// Metrics is a type alias for the Metrics interface from the types package.
// It provides Prometheus-compatible metrics collection.
type Metrics = types.Metrics

// Fields is a type alias for structured logging fields.
// It represents a map of key-value pairs for contextual information.
type Fields = types.Fields

// Config is a type alias for the observability configuration.
// It contains settings for service name, environment, log level, and output.
type Config = types.Config

// Provider is a type alias for the Provider interface from the types package.
// It manages the lifecycle of logging and metrics components.
type Provider = types.Provider

// DefaultProvider implements the Provider interface.
// It manages Logger and Metrics instances for different components,
// ensuring thread-safe access and singleton pattern for each component.
// The provider uses lazy initialization to create loggers and metrics
// only when they are first requested.
type DefaultProvider struct {
	// config holds the observability configuration
	config  *Config
	// loggers stores Logger instances indexed by component name
	loggers map[string]Logger
	// metrics stores Metrics instances indexed by component name
	metrics map[string]Metrics
	// mu provides thread-safe access to the maps
	mu      sync.RWMutex
}

// NewProvider creates a new observability provider with the given configuration.
// If LogOutput is not specified in the config, it defaults to os.Stdout.
// The provider manages Logger and Metrics instances for different components,
// creating them lazily on first access.
//
// Example:
//
//	config := &Config{
//		ServiceName: "audit-aggregator",
//		Environment: "production",
//		LogLevel:    "info",
//	}
//	provider := NewProvider(config)
//	logger := provider.Logger("processor")
func NewProvider(config *Config) Provider {
	if config.LogOutput == nil {
		config.LogOutput = os.Stdout
	}

	return &DefaultProvider{
		config:  config,
		loggers: make(map[string]Logger),
		metrics: make(map[string]Metrics),
	}
}

// Logger returns a Logger instance for the specified component.
// It implements a thread-safe singleton pattern, ensuring that each component
// gets the same Logger instance across multiple calls. The logger is created
// lazily on first access with component-specific fields.
//
// The returned logger includes:
//   - All fields from the provider's config.AdditionalFields
//   - A "component" field set to the provided component name
//   - Service name formatted as "{config.ServiceName}.{component}"
//
// Parameters:
//   - component: The name of the component requesting the logger
//
// Returns:
//   - A Logger instance configured for the specified component
func (p *DefaultProvider) Logger(component string) Logger {
	p.mu.RLock()
	if l, exists := p.loggers[component]; exists {
		p.mu.RUnlock()
		return l
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if l, exists := p.loggers[component]; exists {
		return l
	}

	// Create new logger
	fields := make(Fields)
	for k, v := range p.config.AdditionalFields {
		fields[k] = v
	}
	fields["component"] = component

	serviceName := fmt.Sprintf("%s.%s", p.config.ServiceName, component)

	// Create the logger using the logger package's factory function
	// The logger package should return a Logger interface, not a concrete type
	l := logger.New(
		serviceName,
		p.config.Environment,
		p.config.LogLevel,
		p.config.LogOutput,
		fields,
	)

	var loggerInterface Logger = l
	p.loggers[component] = loggerInterface

	return loggerInterface
}

// Metrics returns a Metrics instance for the specified component.
// It implements a thread-safe singleton pattern, ensuring that each component
// gets the same Metrics instance across multiple calls. The metrics collector
// is created lazily on first access.
//
// The returned metrics collector provides Prometheus-compatible metrics
// with component-specific labels and metric names.
//
// Parameters:
//   - component: The name of the component requesting the metrics collector
//
// Returns:
//   - A Metrics instance configured for the specified component
func (p *DefaultProvider) Metrics(component string) Metrics {
	p.mu.RLock()
	if m, exists := p.metrics[component]; exists {
		p.mu.RUnlock()
		return m
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if m, exists := p.metrics[component]; exists {
		return m
	}

	// Create new metrics
	m := metrics.New(component)

	// Type assertion to convert *metrics.PrometheusMetrics to Metrics interface
	var metricsInterface Metrics = m
	p.metrics[component] = metricsInterface

	return metricsInterface
}

// Close shuts down the provider and releases associated resources.
// It closes the LogOutput if it implements io.Closer, except for
// os.Stdout and os.Stderr which should not be closed.
//
// This method is thread-safe and should be called when the provider
// is no longer needed to ensure proper cleanup.
//
// Returns:
//   - An error if closing the LogOutput fails, nil otherwise
func (p *DefaultProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If LogOutput is a closer, close it
	if closer, ok := p.config.LogOutput.(io.Closer); ok {
		if closer != os.Stdout && closer != os.Stderr {
			return closer.Close()
		}
	}

	return nil
}
