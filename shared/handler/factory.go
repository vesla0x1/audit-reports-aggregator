package handler

import (
	"os"
	"shared/config"
	"shared/observability"
)

// Factory provides methods for creating platform-specific handlers.
// This is the main entry point for creating handlers with proper
// configuration and middleware setup.
type Factory struct {
	worker     Worker
	provider   observability.Provider
	handlerCfg config.HandlerConfig
	retryCfg   config.RetryConfig
}

// NewFactory creates a new handler factory with sensible defaults.
// This is the recommended way to create handlers.
func NewFactory(worker Worker, provider observability.Provider) *Factory {
	return &Factory{
		worker:     worker,
		provider:   provider,
		handlerCfg: config.DefaultHandlerConfig(),
		retryCfg:   config.DefaultRetryConfig(),
	}
}

// WithHandlerConfig sets custom handler configuration.
func (f *Factory) WithHandlerConfig(config config.HandlerConfig) *Factory {
	f.handlerCfg = config
	return f
}

// WithRetryConfig sets custom retry configuration.
func (f *Factory) WithRetryConfig(config config.RetryConfig) *Factory {
	f.retryCfg = config
	return f
}

// Create creates a handler for the detected or configured platform.
// This automatically detects the platform and creates the appropriate handler.
func (f *Factory) Create() *Handler {
	// Detect platform if not set
	if f.handlerCfg.Platform == "" || f.handlerCfg.Platform == "auto" {
		f.handlerCfg.Platform = DetectPlatform()
	}

	// Create handler
	handler := NewHandler(f.worker, f.provider, &f.handlerCfg)

	// Add default middleware stack
	f.applyDefaultMiddleware(handler)

	return handler
}

// CreateHTTP creates a handler specifically for HTTP/Knative.
func (f *Factory) CreateHTTP() *Handler {
	f.handlerCfg.Platform = "http"
	return f.Create()
}

// CreateLambda creates a handler specifically for AWS Lambda.
func (f *Factory) CreateLambda() *Handler {
	f.handlerCfg.Platform = "lambda"
	return f.Create()
}

// applyDefaultMiddleware adds the standard middleware stack.
func (f *Factory) applyDefaultMiddleware(handler *Handler) {
	// Recovery middleware (outermost - catches all panics)
	handler.Use(RecoveryMiddleware(f.provider))

	if f.handlerCfg.Timeout > 0 {
		handler.Use(TimeoutMiddleware(f.handlerCfg.Timeout))
	}

	// Tracing middleware
	if f.handlerCfg.EnableTracing {
		handler.Use(TracingMiddleware())
	}

	// Metrics middleware
	if f.handlerCfg.EnableMetrics {
		handler.Use(MetricsMiddleware(f.provider))
	}

	// Logging middleware
	handler.Use(LoggingMiddleware(f.provider))

	// Validation middleware
	handler.Use(ValidationMiddleware())

	// Retry middleware (if enabled)
	if f.retryCfg.MaxAttempts > 0 {
		handler.Use(RetryMiddleware(&f.retryCfg))
	}
}

// DetectPlatform attempts to detect the runtime platform from environment.
func DetectPlatform() string {

	// Check for Lambda runtime
	if _, exists := os.LookupEnv("AWS_LAMBDA_FUNCTION_NAME"); exists {
		return "lambda"
	}

	// Check for Lambda runtime API (another Lambda indicator)
	if _, exists := os.LookupEnv("AWS_LAMBDA_RUNTIME_API"); exists {
		return "lambda"
	}

	// Default to HTTP
	return "http"
}
