package handler

import (
	"os"
	"strconv"
	"time"

	"shared/observability"
)

// Factory provides methods for creating platform-specific handlers.
// This is the main entry point for creating handlers with proper
// configuration and middleware setup.
type Factory struct {
	worker   Worker
	provider observability.Provider
	config   *Config
}

// NewFactory creates a new handler factory.
// This is the recommended way to create handlers.
func NewFactory(worker Worker, provider observability.Provider) *Factory {
	return &Factory{
		worker:   worker,
		provider: provider,
		config:   ConfigFromEnv(),
	}
}

// WithConfig sets custom configuration.
func (f *Factory) WithConfig(config *Config) *Factory {
	f.config = config
	return f
}

// Create creates a handler for the detected or configured platform.
// This automatically detects the platform and creates the appropriate handler.
func (f *Factory) Create() *Handler {
	// Detect platform if not set
	if f.config.Platform == "" || f.config.Platform == "auto" {
		f.config.Platform = DetectPlatform()
	}

	// Validate configuration
	f.config.Validate()

	// Create handler
	handler := NewHandler(f.worker, f.provider, f.config)

	// Add default middleware stack
	f.applyDefaultMiddleware(handler)

	return handler
}

// CreateHTTP creates a handler specifically for HTTP/Knative.
func (f *Factory) CreateHTTP() *Handler {
	f.config.Platform = "http"
	return f.Create()
}

// CreateOpenFaaS creates a handler specifically for OpenFaaS.
func (f *Factory) CreateOpenFaaS() *Handler {
	f.config.Platform = "openfaas"
	if timeout := getOpenFaaSTimeout(); timeout > 0 {
		f.config.Timeout = timeout
	}
	return f.Create()
}

// applyDefaultMiddleware adds the standard middleware stack.
func (f *Factory) applyDefaultMiddleware(handler *Handler) {
	// Recovery middleware (outermost - catches all panics)
	handler.Use(RecoveryMiddleware(f.provider))

	if f.config.Timeout > 0 {
		handler.Use(TimeoutMiddleware(f.config.Timeout))
	}

	// Tracing middleware
	if f.config.EnableTracing {
		handler.Use(TracingMiddleware())
	}

	// Metrics middleware
	if f.config.EnableMetrics {
		handler.Use(MetricsMiddleware(f.provider))
	}

	// Logging middleware
	handler.Use(LoggingMiddleware(f.provider))

	// Validation middleware
	handler.Use(ValidationMiddleware())

	// Retry middleware (if configured)
	if f.config.RetryConfig != nil && f.config.RetryConfig.MaxRetries > 0 {
		handler.Use(RetryMiddleware(f.config.RetryConfig))
	}
}

// DetectPlatform attempts to detect the runtime platform from environment.
func DetectPlatform() string {
	// OpenFaaS
	if os.Getenv("OPENFAAS_FUNCTION_NAME") != "" {
		return "openfaas"
	}

	// Knative
	if os.Getenv("K_SERVICE") != "" {
		return "knative"
	}

	// Default to HTTP
	return "http"
}

// getOpenFaaSTimeout extracts timeout from OpenFaaS environment.
func getOpenFaaSTimeout() time.Duration {
	// Check multiple possible timeout sources
	timeoutSources := []string{
		"OPENFAAS_TIMEOUT",
		"timeout",
		"exec_timeout",
		"function_timeout",
	}

	for _, source := range timeoutSources {
		if timeoutStr := os.Getenv(source); timeoutStr != "" {
			// Try parsing as duration (e.g., "30s")
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				return timeout
			}
			// Try parsing as seconds
			if seconds, err := strconv.Atoi(timeoutStr); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	// Default OpenFaaS timeout
	return 30 * time.Second
}
