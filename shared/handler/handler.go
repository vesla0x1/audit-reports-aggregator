package handler

import (
	"context"
	"os"
	"time"

	"shared/config"
	"shared/observability"
)

// Handler is the main handler that wraps a Worker with platform-specific adapters.
// It provides common functionality like middleware, error handling, and observability.
type Handler struct {
	worker      Worker
	obs         observability.Provider
	middlewares []Middleware
	config      *config.HandlerConfig
}

// Middleware defines the interface for handler middleware.
// Middlewares wrap the handler function to add cross-cutting concerns.
type Middleware func(next HandlerFunc) HandlerFunc

// HandlerFunc is the function signature for handling requests.
// This is the core processing function that middlewares wrap.
type HandlerFunc func(ctx context.Context, req Request) (Response, error)

// NewHandler creates a new handler with the given worker and configuration.
// This is the low-level constructor - most users should use the Factory instead.
func NewHandler(worker Worker, provider observability.Provider, config *config.HandlerConfig) *Handler {
	return &Handler{
		worker:      worker,
		obs:         provider,
		config:      config,
		middlewares: []Middleware{},
	}
}

// Use adds middleware to the handler chain.
// Middleware is executed in the order it's added.
func (h *Handler) Use(middleware Middleware) {
	h.middlewares = append(h.middlewares, middleware)
}

// Handle processes a request through the middleware chain and worker.
// This is the main entry point for request processing.
func (h *Handler) Handle(ctx context.Context, req Request) (Response, error) {
	// Build the handler chain
	handler := h.buildHandlerChain()

	// Create context with timeout
	if h.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.config.Timeout)
		defer cancel()
	}

	// Add context values
	ctx = context.WithValue(ctx, "request_id", req.ID)
	ctx = context.WithValue(ctx, "worker", h.worker.Name())
	ctx = context.WithValue(ctx, "platform", h.config.Platform)

	return handler(ctx, req)
}

// gracefulShutdown handles the shutdown process with proper metrics
func GracefulShutdown(logger observability.Logger, metrics observability.Metrics, startTime time.Time) {
	ctx := context.Background()

	// Record shutdown initiation
	metrics.RecordSuccess("shutdown_initiated")

	logger.Info(ctx, "Shutting down gracefully", observability.Fields{
		"uptime_seconds": time.Since(startTime).Seconds(),
	})

	// Create a timeout context for shutdown operations
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Perform cleanup operations here
	// For example: close database connections, flush logs, etc.

	// Record final metrics
	metrics.RecordDuration("service_uptime", time.Since(startTime).Seconds())
	metrics.RecordSuccess("shutdown_complete")

	logger.Info(shutdownCtx, "Shutdown complete", nil)

	// Give time for final metrics to be sent
	time.Sleep(2 * time.Second)

	os.Exit(0)
}

// buildHandlerChain builds the middleware chain with the worker at the end.
// Middleware is applied in reverse order so that the first middleware
// added is the outermost layer.
func (h *Handler) buildHandlerChain() HandlerFunc {
	// Start with the worker handler
	handler := h.workerHandler

	// Apply middleware in reverse order
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		handler = h.middlewares[i](handler)
	}

	return handler
}

// workerHandler is the final handler that calls the worker.
// This is the innermost layer of the middleware chain.
func (h *Handler) workerHandler(ctx context.Context, req Request) (Response, error) {
	return h.worker.Process(ctx, req)
}

// Health checks the health of the worker and handler.
func (h *Handler) Health(ctx context.Context) error {
	return h.worker.Health(ctx)
}

// Config returns the handler configuration.
func (h *Handler) Config() *config.HandlerConfig {
	return h.config
}

// Worker returns the underlying worker.
func (h *Handler) Worker() Worker {
	return h.worker
}
