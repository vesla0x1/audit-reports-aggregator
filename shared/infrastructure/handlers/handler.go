package infrahandler

import (
	"context"

	"shared/config"
	"shared/domain/handler"
	"shared/domain/observability"
)

type Handler struct {
	useCase handler.UseCase
	chain   handler.HandlerFunc
	logger  observability.Logger
	metrics observability.Metrics
}

func NewHandler(
	useCase handler.UseCase,
	cfg *config.Config,
	logger observability.Logger,
	metrics observability.Metrics,
) *Handler {
	h := &Handler{
		useCase: useCase,
		logger:  logger,
		metrics: metrics,
	}

	// Build middleware chain
	chain := func(ctx context.Context, req handler.Request) (handler.Response, error) {
		return useCase.Run(ctx, req)
	}

	// Apply middleware in reverse order
	middlewares := buildMiddlewares(cfg, logger, metrics)
	for i := len(middlewares) - 1; i >= 0; i-- {
		chain = middlewares[i](chain)
	}

	h.chain = chain
	return h
}

func buildMiddlewares(
	cfg *config.Config,
	logger observability.Logger,
	metrics observability.Metrics,
) []handler.Middleware {
	var middlewares []handler.Middleware

	/*middlewares = append(middlewares, middleware.RecoveryMiddleware(logger))

	if cfg.Handler.Timeout > 0 {
		middlewares = append(middlewares, middleware.TimeoutMiddleware(cfg.Handler.Timeout))
	}

	if cfg.Handler.EnableMetrics {
		middlewares = append(middlewares, middleware.MetricsMiddleware(metrics))
	}

	middlewares = append(middlewares, middleware.LoggingMiddleware(logger))
	middlewares = append(middlewares, middleware.ValidationMiddleware())

	if cfg.Retry.MaxAttempts > 0 {
		middlewares = append(middlewares, middleware.RetryMiddleware(&cfg.Retry))
	}*/

	return middlewares
}

func (h *Handler) Logger() observability.Logger {
	return h.logger
}

func (h *Handler) Metrics() observability.Metrics {
	return h.metrics
}

func (h *Handler) Handle(ctx context.Context, req handler.Request) (handler.Response, error) {
	return h.chain(ctx, req)
}
